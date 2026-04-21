package goservice

import (
	"errors"
	"sync"
	"time"

	"github.com/bep/debounce"
	"github.com/google/uuid"
)

// BROKER
type BrokerConfig struct {
	NodeId            string
	TransporterConfig TransporterConfig
	LoggerConfig      Logconfig
	Metrics           string
	TraceConfig       TraceConfig
	DiscoveryConfig   DiscoveryConfig
	RequestTimeOut    int
	// Retry is the broker-level default retry policy applied to all action calls.
	Retry RetryPolicy
	// CircuitBreaker is the broker-level circuit-breaker configuration.
	CircuitBreaker CircuitBreakerConfig
	// Middlewares is the ordered list of middleware applied to every local action call.
	// The first middleware registered is the outermost wrapper in the pipeline.
	Middlewares []Middleware
}

type Broker struct {
	Config             BrokerConfig
	Services           []*Service
	Started            func(*Context)
	Stoped             func(*Context)
	transporter        Transporter
	bus                EventBus
	traceSpans         map[string]*traceSpan
	channelPrivateInfo string
	registryServices   []RegistryService
	registryNodes      []RegistryNode
	registryNode       RegistryNode
	debouncedEmitInfo  func(f func())
	trace              Trace
	logs               Log
	circuitBreakers    map[string]*endpointCircuitBreaker
	cbMu               sync.RWMutex
	cache              *memoryCache
	bulkheads          map[string]*bulkheadState
	bulkMu             sync.RWMutex
}

func Init(config BrokerConfig) *Broker {
	initMetrics()
	broker := Broker{
		Config:          config,
		circuitBreakers: make(map[string]*endpointCircuitBreaker),
		bulkheads:       make(map[string]*bulkheadState),
		cache:           newMemoryCache(),
	}
	broker.initDiscovery()
	broker.initTransporter()
	broker.debouncedEmitInfo = debounce.New(1000 * time.Millisecond)
	broker.debouncedEmitInfo(broker.startDiscovery)
	broker.initTrace()
	broker.initLog()
	broker.LogInfo("Broker started")
	return &broker
}

// applyMixins returns a new Service with mixin actions, events, and hooks merged in.
// Mixin definitions come first; the service's own definitions override same-named entries.
func (b *Broker) applyMixins(service *Service) *Service {
	if len(service.Mixins) == 0 {
		return service
	}
	merged := &Service{
		Name:         service.Name,
		Version:      service.Version,
		Dependencies: service.Dependencies,
		Started:      service.Started,
		Stoped:       service.Stoped,
	}

	// Collect mixin actions (later mixins override earlier ones; service overrides all).
	actionMap := make(map[string]Action)
	for _, mixin := range service.Mixins {
		for _, a := range mixin.Actions {
			actionMap[a.Name] = a
		}
		merged.Events = append(merged.Events, mixin.Events...)
		// Mixin Before/Error hooks run before service hooks.
		merged.Hooks.Before = append(merged.Hooks.Before, mixin.Hooks.Before...)
		merged.Hooks.After = append(merged.Hooks.After, mixin.Hooks.After...)
		merged.Hooks.Error = append(merged.Hooks.Error, mixin.Hooks.Error...)
	}
	// Service's own actions override mixins.
	for _, a := range service.Actions {
		actionMap[a.Name] = a
	}
	for _, a := range actionMap {
		merged.Actions = append(merged.Actions, a)
	}
	merged.Events = append(merged.Events, service.Events...)
	// Service hooks appended after mixin hooks.
	merged.Hooks.Before = append(merged.Hooks.Before, service.Hooks.Before...)
	merged.Hooks.After = append(merged.Hooks.After, service.Hooks.After...)
	merged.Hooks.Error = append(merged.Hooks.Error, service.Hooks.Error...)

	return merged
}

// getOrCreateBulkhead returns the existing bulkhead state for key or creates a new one.
func (b *Broker) getOrCreateBulkhead(key string, cfg BulkheadConfig) *bulkheadState {
	b.bulkMu.RLock()
	bs, ok := b.bulkheads[key]
	b.bulkMu.RUnlock()
	if ok {
		return bs
	}
	b.bulkMu.Lock()
	defer b.bulkMu.Unlock()
	if bs, ok = b.bulkheads[key]; ok {
		return bs
	}
	bs = newBulkheadState(cfg)
	b.bulkheads[key] = bs
	return bs
}

func (b *Broker) LoadService(service *Service) {
	// Apply mixins before anything else.
	service = b.applyMixins(service)

	b.LogInfo("Load service `" + service.Name + "`")
	b.Services = append(b.Services, service)

	// Determine the addressable name (versioned services use "v{version}.{name}").
	qualName := service.qualifiedName()

	// add service to registry
	var registryActions []RegistryAction
	for _, a := range service.Actions {
		registryActions = append(registryActions, RegistryAction{
			Name:    a.Name,
			Params:  a.Params,
			Rest:    a.Rest,
			Timeout: a.Timeout,
		})
		// Pre-create bulkheads for actions that define one.
		if a.Bulkhead != nil {
			key := qualName + "." + a.Name
			b.getOrCreateBulkhead(key, *a.Bulkhead)
		}
	}
	var registryEvents []RegistryEvent
	for _, e := range service.Events {
		registryEvents = append(registryEvents, RegistryEvent{
			Name:   e.Name,
			Params: e.Params,
		})
	}
	b.registryServices = append(b.registryServices, RegistryService{
		Node:    b.registryNode,
		Name:    qualName,
		Version: service.Version,
		Actions: registryActions,
		Events:  registryEvents,
	})

	// emit info service
	b.debouncedEmitInfo(b.startDiscovery)
	// handle logic service

	// mapping broker to service
	service.Broker = b
	// service lifecycle
	if service.Started != nil {
		go func() {
			// Wait for declared service dependencies to appear in the registry.
			if len(service.Dependencies) > 0 {
				b.waitForDependencies(service.Dependencies)
			}

			//trace
			spanId := b.startTraceSpan("Service `"+service.Name+"` started", "action", "", "", map[string]interface{}{}, "", "", 1)
			// started

			context := Context{
				RequestId:         uuid.New().String(),
				Params:            map[string]interface{}{},
				Meta:              map[string]interface{}{},
				FromService:       "",
				FromNode:          b.Config.NodeId,
				CallingLevel:      1,
				TraceParentId:     spanId,
				TraceParentRootId: spanId,
			}
			context.Service = service
			context.Call = func(action string, params interface{}, opts ...CallOpts) (interface{}, error) {
				optsTemp := CallOpts{}
				if len(opts) != 0 {
					optsTemp = opts[0]
				}
				ctxCall := Context{
					RequestId:         uuid.New().String(),
					ResponseId:        uuid.New().String(),
					Params:            params,
					Meta:              context.Meta,
					FromNode:          b.Config.NodeId,
					FromService:       service.Name,
					FromAction:        "",
					CallingLevel:      1,
					TraceParentId:     spanId,
					TraceParentRootId: spanId,
				}
				callResult, err := b.callWithRetry(ctxCall, action, params, optsTemp, service.Name, "", "")
				b.addTraceSpans(callResult.TraceSpans)
				if err != nil {
					return nil, err
				}
				if callResult.Error {
					return nil, errors.New(callResult.ErrorMessage)
				}
				return callResult.Data, nil
			}
			context.Broadcast = func(event string, params interface{}) {
				b.Broadcast(event, params)
			}
			service.Started(&context)
			b.endTraceSpan(spanId, nil)
		}()
	}

	// actions handle — use the qualified name as the service identifier
	for _, a := range service.Actions {
		go b.listenActionCall(qualName, a, service)
	}

	// events handle — use the original service name for event channels (events are not versioned)
	for _, e := range service.Events {
		go b.listenEventCall(qualName, e)
	}
}

// waitForDependencies blocks until all listed service names appear in the registry,
// checking every 500 ms. It gives up and logs a warning after 60 seconds.
func (b *Broker) waitForDependencies(deps []string) {
	deadline := time.Now().Add(60 * time.Second)
	for {
		allFound := true
		for _, dep := range deps {
			found := false
			for _, rs := range b.registryServices {
				if rs.Name == dep {
					found = true
					break
				}
			}
			if !found {
				allFound = false
				break
			}
		}
		if allFound {
			return
		}
		if time.Now().After(deadline) {
			b.LogWarning("Dependency wait timed out for service `" + b.Config.NodeId + "`")
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func (b *Broker) Call(callerService string, traceName string, action string, params interface{}, opts CallOpts) (interface{}, error) {
	//trace
	if traceName == "" {
		traceName = "Call from service `" + callerService + "`"
	}
	spanId := b.startTraceSpan(traceName, "action", "", "", map[string]interface{}{}, "", "", 1)

	ctxCall := Context{
		RequestId:         uuid.New().String(),
		ResponseId:        uuid.New().String(),
		Params:            params,
		Meta:              nil,
		FromNode:          b.Config.NodeId,
		FromService:       callerService,
		FromAction:        "",
		CallingLevel:      1,
		TraceParentId:     spanId,
		TraceParentRootId: spanId,
	}
	callResult, err := b.callWithRetry(ctxCall, action, params, opts, "", "", "")
	b.addTraceSpans(callResult.TraceSpans)
	b.endTraceSpan(spanId, err)
	if err != nil {
		return nil, err
	}
	if callResult.Error {
		return nil, errors.New(callResult.ErrorMessage)
	}
	return callResult.Data, nil
}

// Broadcast sends an event to ALL subscriber instances on every node, bypassing
// the least-connections group balancing used by Emit / ctx.Call.
func (b *Broker) Broadcast(eventName string, params interface{}) {
	b.broadcastEvent(eventName, params, "", "", "")
}

// broadcastEvent is the internal implementation used by Broadcast and ctx.Broadcast.
// It sends to every registered subscriber regardless of load-balancing.
func (b *Broker) broadcastEvent(eventName string, params interface{}, callerService, callerAction, callerEvent string) {
	spanId := b.startTraceSpan("Broadcast `"+eventName+"`", "event", "", "", params, "", "", 1)
	defer b.endTraceSpan(spanId, nil)

	for _, s := range b.registryServices {
		for _, e := range s.Events {
			if !matchEventPattern(e.Name, eventName) {
				continue
			}
			responseId := uuid.New().String()
			dataSend := RequestTranferData{
				Params:            params,
				Meta:              nil,
				RequestId:         uuid.New().String(),
				ResponseId:        responseId,
				CallerNodeId:      b.Config.NodeId,
				CallerService:     callerService,
				CallerAction:      callerAction,
				CallerEvent:       callerEvent,
				CallingLevel:      1,
				CalledTime:        time.Now().UnixNano(),
				CallToService:     s.Name,
				CallToEvent:       e.Name,
				TraceParentId:     spanId,
				TraceRootParentId: spanId,
			}
			if s.Node.NodeId == b.Config.NodeId {
				channelCall := GO_SERVICE_PREFIX + "." + b.Config.NodeId + "." + s.Name + "." + e.Name
				b.emitWithTimeout(channelCall, "", dataSend)
			} else if b.Config.TransporterConfig.Enable {
				channelTransporter := GO_SERVICE_PREFIX + "." + s.Node.NodeId + ".request"
				b.transporter.Emit(channelTransporter, dataSend)
			}
		}
	}
}

func (b *Broker) Hold() {
	select {}
}
