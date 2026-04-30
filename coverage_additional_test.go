package goservice

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zserge/metric"
)

func testBroker(t *testing.T, node string) *Broker {
	t.Helper()
	b := Init(BrokerConfig{NodeId: node, RequestTimeOut: 200})
	t.Cleanup(func() { b.Config.RequestTimeOut = 1 })
	return b
}

func waitUntil(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("condition was not met")
}

func loadTestService(t *testing.T, b *Broker, service *Service) {
	t.Helper()
	b.LoadService(service)
	b.initMestricCountCallAction()
	for _, action := range service.Actions {
		channel := GO_SERVICE_PREFIX + "." + b.Config.NodeId + "." + service.qualifiedName() + "." + action.Name
		waitUntil(t, func() bool {
			b.bus.lock.RLock()
			defer b.bus.lock.RUnlock()
			return b.bus.data[channel] != nil
		})
	}
	for _, event := range service.Events {
		channel := GO_SERVICE_PREFIX + "." + b.Config.NodeId + "." + service.qualifiedName() + "." + event.Name
		waitUntil(t, func() bool {
			b.bus.lock.RLock()
			defer b.bus.lock.RUnlock()
			return b.bus.data[channel] != nil
		})
	}
}

func TestLocalBrokerActionCallIntegration(t *testing.T) {
	b := testBroker(t, "node-action")
	loadTestService(t, b, &Service{Name: "math", Actions: []Action{{Name: "add", Handle: func(ctx *Context) (interface{}, error) {
		p := ctx.Params.(map[string]interface{})
		return p["a"].(int) + p["b"].(int), nil
	}}}})

	got, err := b.Call("tester", "", "math.add", map[string]interface{}{"a": 2, "b": 3}, CallOpts{})
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}
	if got != 5 {
		t.Fatalf("got %v, want 5", got)
	}
}

func TestBrokerValidationCacheHooksMiddlewareIntegration(t *testing.T) {
	calls := int32(0)
	order := make([]string, 0)
	min := 2.0
	b := testBroker(t, "node-cache")
	b.Config.Middlewares = []Middleware{{LocalAction: func(ctx *Context, action Action, next func(*Context) (interface{}, error)) (interface{}, error) {
		order = append(order, "mw-before")
		res, err := next(ctx)
		order = append(order, "mw-after")
		return res, err
	}}}
	loadTestService(t, b, &Service{
		Name: "greeter",
		Hooks: ActionHooks{
			Before: []BeforeHookFunc{func(ctx *Context) error { order = append(order, "service-before"); return nil }},
			After: []AfterHookFunc{func(ctx *Context, result interface{}) (interface{}, error) {
				order = append(order, "service-after")
				return result.(string) + "!", nil
			}},
		},
		Actions: []Action{{
			Name:   "hello",
			Schema: map[string]ParamRule{"name": {Type: "string", Required: true, Min: &min}},
			Cache:  &CacheConfig{TTL: time.Minute, Keys: []string{"name"}},
			Hooks: ActionHooks{After: []AfterHookFunc{func(ctx *Context, result interface{}) (interface{}, error) {
				order = append(order, "action-after")
				return strings.ToUpper(result.(string)), nil
			}}},
			Handle: func(ctx *Context) (interface{}, error) {
				atomic.AddInt32(&calls, 1)
				return "hi " + ctx.Params.(map[string]interface{})["name"].(string), nil
			},
		}},
	})

	params := map[string]interface{}{"name": "ada", "ignored": time.Now().UnixNano()}
	got, err := b.Call("tester", "trace", "greeter.hello", params, CallOpts{})
	if err != nil || got != "HI ADA!" {
		t.Fatalf("got (%v, %v), want HI ADA!, nil", got, err)
	}
	got, err = b.Call("tester", "trace", "greeter.hello", map[string]interface{}{"name": "ada", "ignored": "different"}, CallOpts{})
	if err != nil || got != "HI ADA!" {
		t.Fatalf("cached got (%v, %v), want HI ADA!, nil", got, err)
	}
	if atomic.LoadInt32(&calls) != 1 {
		t.Fatalf("handler called %d times, want once", calls)
	}
	wantOrder := []string{"mw-before", "service-before", "service-after", "action-after", "mw-after"}
	if !reflect.DeepEqual(order, wantOrder) {
		t.Fatalf("order = %#v, want %#v", order, wantOrder)
	}
	_, err = b.Call("tester", "trace", "greeter.hello", map[string]interface{}{"name": "a"}, CallOpts{})
	if err == nil || !strings.Contains(err.Error(), "length must be at least") {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestBrokerEventsAndBroadcastIntegration(t *testing.T) {
	b := testBroker(t, "node-event")
	got := make(chan interface{}, 2)
	loadTestService(t, b, &Service{Name: "listener", Events: []Event{{Name: "order.*", Handle: func(ctx *Context) {
		got <- ctx.Params
	}}}})

	b.Broadcast("order.created", map[string]interface{}{"id": 1})
	select {
	case v := <-got:
		if v.(map[string]interface{})["id"] != 1 {
			t.Fatalf("unexpected event payload: %#v", v)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for broadcast event")
	}
}

func TestBrokerTimeoutRetryCircuitBreakerAndBulkhead(t *testing.T) {
	b := testBroker(t, "node-resilience")
	var slowCalls int32
	b.Config.CircuitBreaker = CircuitBreakerConfig{Enabled: true, Threshold: 1, HalfOpenTimeout: time.Millisecond, SuccessThreshold: 1}
	loadTestService(t, b, &Service{Name: "slow", Actions: []Action{{Name: "wait", Timeout: 5, Handle: func(ctx *Context) (interface{}, error) {
		atomic.AddInt32(&slowCalls, 1)
		time.Sleep(20 * time.Millisecond)
		return "late", nil
	}}}})

	_, err := b.Call("tester", "trace", "slow.wait", nil, CallOpts{Retry: &RetryPolicy{MaxRetries: 1}})
	if err == nil || (err.Error() != "Timeout" && !errors.Is(err, errCircuitOpen)) {
		t.Fatalf("expected timeout or open circuit, got %v", err)
	}
	if atomic.LoadInt32(&slowCalls) == 0 {
		t.Fatal("expected at least one slow call")
	}
	_, err = b.Call("tester", "trace", "slow.wait", nil, CallOpts{})
	if !errors.Is(err, errCircuitOpen) {
		t.Fatalf("expected open circuit, got %v", err)
	}

	b2 := testBroker(t, "node-bulkhead")
	block := make(chan struct{})
	loadTestService(t, b2, &Service{Name: "limited", Actions: []Action{{Name: "run", Bulkhead: &BulkheadConfig{MaxConcurrency: 1}, Handle: func(ctx *Context) (interface{}, error) {
		<-block
		return "ok", nil
	}}}})

	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, e := b2.Call("tester", "trace", "limited.run", nil, CallOpts{})
			errs <- e
		}()
	}
	waitUntil(t, func() bool { return len(errs) >= 1 })
	close(block)
	wg.Wait()
	close(errs)
	foundBulkhead := false
	for e := range errs {
		if e != nil && strings.Contains(e.Error(), errBulkheadFull.Error()) {
			foundBulkhead = true
		}
	}
	if !foundBulkhead {
		t.Fatal("expected one bulkhead rejection")
	}
}

func TestUtilityFunctions(t *testing.T) {
	if GO_SERVICE_VERSION == "" || GO_SERVICE_PREFIX != "GS" {
		t.Fatal("unexpected service constants")
	}
	methods := []Method{GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS, Method(99)}
	want := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS", ""}
	for i, m := range methods {
		if got := m.String(); got != want[i] {
			t.Fatalf("method %v string = %q, want %q", m, got, want[i])
		}
	}
	if (&Service{Name: "svc"}).qualifiedName() != "svc" || (&Service{Name: "svc", Version: "2"}).qualifiedName() != "v2.svc" {
		t.Fatal("unexpected qualified service names")
	}
	for _, tc := range []struct {
		pattern, name string
		want          bool
	}{
		{"order.created", "order.created", true}, {"**", "any.event", true}, {"order.*", "order.created", true}, {"order.*", "order.created.extra", false}, {"order.**", "order.created.extra", true}, {"order.created", "order", false},
	} {
		if got := matchEventPattern(tc.pattern, tc.name); got != tc.want {
			t.Fatalf("matchEventPattern(%q,%q)=%v want %v", tc.pattern, tc.name, got, tc.want)
		}
	}
}

func TestCacheAndBulkheadUnits(t *testing.T) {
	c := &memoryCache{entries: make(map[string]*cacheEntry)}
	if _, ok := c.get("missing"); ok {
		t.Fatal("missing cache key hit")
	}
	c.set("prefix:1", 1, 10*time.Millisecond)
	c.set("prefix:2", 2, 0)
	if got, ok := c.get("prefix:1"); !ok || got != 1 {
		t.Fatalf("cache get = (%v,%v), want (1,true)", got, ok)
	}
	time.Sleep(20 * time.Millisecond)
	if _, ok := c.get("prefix:1"); ok {
		t.Fatal("expired cache entry should miss")
	}
	c.del("prefix:*")
	if _, ok := c.get("prefix:2"); ok {
		t.Fatal("prefix delete failed")
	}
	if key := buildCacheKey("act", map[string]interface{}{"a": 1, "b": 2}, CacheConfig{Keys: []string{"a"}}); key != `act:{"a":1}` {
		t.Fatalf("cache key = %s", key)
	}
	if key := buildCacheKey("act", "raw", CacheConfig{Keys: []string{"a"}}); key != `act:"raw"` {
		t.Fatalf("raw cache key = %s", key)
	}

	bs := newBulkheadState(BulkheadConfig{MaxConcurrency: -1, MaxQueueSize: -1})
	if err := bs.acquire(); err != nil {
		t.Fatalf("first acquire: %v", err)
	}
	if err := bs.acquire(); !errors.Is(err, errBulkheadFull) {
		t.Fatalf("second acquire = %v, want errBulkheadFull", err)
	}
	bs.release()
}

func TestCircuitBreakerUnits(t *testing.T) {
	cfg := CircuitBreakerConfig{Enabled: true, Threshold: 2, HalfOpenTimeout: time.Millisecond, SuccessThreshold: 2}
	cb := &endpointCircuitBreaker{}
	if !cb.isAllowed(cfg) {
		t.Fatal("closed circuit should allow")
	}
	cb.recordFailure(cfg)
	if cb.state != CircuitClosed || cb.failures != 1 {
		t.Fatalf("after one failure: %#v", cb)
	}
	cb.recordFailure(cfg)
	if cb.state != CircuitOpen || cb.isAllowed(CircuitBreakerConfig{HalfOpenTimeout: time.Hour}) {
		t.Fatalf("open circuit should reject: %#v", cb)
	}
	time.Sleep(2 * time.Millisecond)
	if !cb.isAllowed(cfg) || cb.state != CircuitHalfOpen {
		t.Fatalf("half-open probe not allowed: %#v", cb)
	}
	if cb.isAllowed(cfg) {
		t.Fatal("second half-open probe should be blocked")
	}
	cb.recordSuccess(cfg)
	if cb.state != CircuitHalfOpen {
		t.Fatalf("one half-open success should remain half-open: %#v", cb)
	}
	cb.recordSuccess(cfg)
	if cb.state != CircuitClosed || cb.failures != 0 {
		t.Fatalf("successful half-open should close: %#v", cb)
	}
	cb.recordFailure(CircuitBreakerConfig{Threshold: 1})
	time.Sleep(time.Millisecond)
	cb.isAllowed(cfg)
	cb.recordFailure(cfg)
	if cb.state != CircuitOpen {
		t.Fatal("half-open failure should reopen")
	}
}

func TestValidationAndHooksUnits(t *testing.T) {
	min, max := 2.0, 3.0
	schema := map[string]ParamRule{
		"s": {Type: "string", Required: true, Min: &min, Max: &max},
		"n": {Type: "number", Required: true, Min: &min, Max: &max},
		"b": {Type: "bool"}, "a": {Type: "array"}, "o": {Type: "object"}, "x": {Type: "any"},
	}
	valid := map[string]interface{}{"s": "abc", "n": int32(2), "b": true, "a": []interface{}{1}, "o": map[string]interface{}{}, "x": struct{}{}}
	if err := validateParams(valid, schema); err != nil {
		t.Fatalf("valid params: %v", err)
	}
	bad := map[string]interface{}{"s": "abcd", "n": 4.0, "b": "no", "a": "no", "o": "no"}
	if err := validateParams(bad, schema); err == nil || !strings.Contains(err.Error(), "Validation error") {
		t.Fatalf("expected validation errors, got %v", err)
	}
	if err := validateParams(nil, map[string]ParamRule{"required": {Required: true}}); err == nil {
		t.Fatal("expected missing required error")
	}

	b := &Broker{}
	ctx := &Context{Service: &Service{Hooks: ActionHooks{
		Before: []BeforeHookFunc{func(ctx *Context) error { return nil }},
		After:  []AfterHookFunc{func(ctx *Context, result interface{}) (interface{}, error) { return result.(int) + 1, nil }},
		Error:  []ErrorHookFunc{func(ctx *Context, err error) error { return err }},
	}}}
	action := Action{Hooks: ActionHooks{
		Before: []BeforeHookFunc{func(ctx *Context) error { return nil }},
		After:  []AfterHookFunc{func(ctx *Context, result interface{}) (interface{}, error) { return result.(int) * 2, nil }},
		Error:  []ErrorHookFunc{func(ctx *Context, err error) error { return nil }},
	}}
	if err := b.runBeforeHooks(ctx, action); err != nil {
		t.Fatal(err)
	}
	res, err := b.runAfterHooks(ctx, action, 2)
	if err != nil || res != 6 {
		t.Fatalf("after hooks = (%v,%v), want (6,nil)", res, err)
	}
	if err := b.runErrorHooks(ctx, action, errors.New("boom")); err != nil {
		t.Fatalf("error hook should recover: %v", err)
	}
	wantErr := errors.New("stop")
	if err := b.runBeforeHooks(&Context{}, Action{Hooks: ActionHooks{Before: []BeforeHookFunc{func(ctx *Context) error { return wantErr }}}}); !errors.Is(err, wantErr) {
		t.Fatalf("before err = %v", err)
	}
	if _, err := b.runAfterHooks(&Context{}, Action{Hooks: ActionHooks{After: []AfterHookFunc{func(ctx *Context, result interface{}) (interface{}, error) { return nil, wantErr }}}}, nil); !errors.Is(err, wantErr) {
		t.Fatalf("after err = %v", err)
	}
}

func TestEventBusLoggerMetricsAndSerializerUnits(t *testing.T) {
	eb := initEventBus()
	ch := make(chan int, 2)
	eb.Subscribe("c", func(v int) { ch <- v })
	eb.Subscribe("c", func(v int) { ch <- v + 1 })
	eb.Publish("c", 1)
	waitUntil(t, func() bool { return len(ch) == 2 })
	eb.UnSubscribe("c")
	if _, ok := eb.data["c"]; ok {
		t.Fatal("unsubscribe failed")
	}

	lc := &LoggerConsole{}
	lc.WriteLog(LogData{Type: LogTypeInfo, Time: 1, Message: "info"})
	lc.logInfo(lc.data[0])
	lc.logWarning(LogData{Type: LogTypeWarning, Time: 1, Message: "warn"})
	lc.logError(LogData{Type: LogTypeError, Time: 1, Message: "err"})
	log := Log{Config: Logconfig{Enable: true, Type: LogConsole, LogLevel: LogTypeWarning}, Extenal: lc}
	log.exportLog(LogData{Type: LogTypeInfo})
	if len(lc.data) != 1 {
		t.Fatal("info below log level should be ignored")
	}
	log.exportLog(LogData{Type: LogTypeError})
	if len(lc.data) != 2 {
		t.Fatal("error log should be written")
	}
	(&Broker{logs: Log{}}).LogInfo("ignored")
	(&Broker{logs: Log{}}).LogWarning("ignored")
	(&Broker{logs: Log{}}).LogError("ignored")

	counter := metric.NewCounter(MCountCallInterval)
	counter.Add(2)
	if got := MetricsGetValueCounter(counter); got != 2 {
		t.Fatalf("counter value = %v", got)
	}
	if _, _, _, ok := splitActionMetricName("bad"); ok {
		t.Fatal("bad metric name should not parse")
	}
	if got := metricsCounterValue("not-json"); got != 0 {
		t.Fatalf("invalid metric JSON = %v", got)
	}

	if _, err := SerializerJson(func() {}); err == nil {
		t.Fatal("expected JSON marshal error")
	}
	if _, err := DeSerializerJson("{"); err == nil {
		t.Fatal("expected JSON unmarshal error")
	}
	if _, err := Deserialize([]byte("{"), SerializerJSON); err == nil {
		t.Fatal("expected wire JSON decode error")
	}
	if _, err := SerializerMsgPackEncode(func() {}); err == nil {
		t.Fatal("expected msgpack encode error")
	}
	if _, err := DeserializerMsgPackDecode([]byte{0xc1}); err == nil {
		t.Fatal("expected msgpack decode error")
	}
}

func TestGatewayHelpers(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	g := &Gateway{Gin: gin.New(), Config: GatewayConfig{Routes: []GatewayConfigRoute{{Path: "/api", WhileList: []string{"math\\..*"}}}}, Service: &Service{Broker: &Broker{logs: Log{}}}}
	if !g.checkMatch("math", RegistryAction{Name: "add", Rest: Rest{Method: GET, Path: "/add"}}, g.Config.Routes[0]) {
		t.Fatal("route should match whitelist")
	}
	if g.checkMatch("math", RegistryAction{Name: "hidden"}, g.Config.Routes[0]) {
		t.Fatal("empty REST action should not match")
	}
	if g.checkMatch("other", RegistryAction{Name: "add", Rest: Rest{Method: GET, Path: "/add"}}, g.Config.Routes[0]) {
		t.Fatal("route should not match whitelist")
	}

	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	req := httptest.NewRequest(http.MethodPost, "/items/42?q=one&q=two&single=x", bytes.NewBufferString(`{"body":"ok"}`))
	ctx.Request = req
	ctx.Params = gin.Params{{Key: "id", Value: "42"}}
	params := g.parseParam(ctx)
	if params["id"] != "42" || params["single"] != "x" || params["body"] != "ok" {
		t.Fatalf("unexpected params: %#v", params)
	}
	if v, ok := params["q"].([]string); !ok || len(v) != 2 {
		t.Fatalf("unexpected repeated query: %#v", params["q"])
	}
	merged := mergeMap(map[string]interface{}{"a": 1}, map[string]interface{}{"b": 2})
	if merged["a"] != 1 || merged["b"] != 2 {
		t.Fatalf("mergeMap failed: %#v", merged)
	}
	one := map[string]interface{}{"only": true}
	if !reflect.DeepEqual(mergeMap(one), one) {
		t.Fatal("single mergeMap should return original map")
	}

	b := testBroker(t, "gw-node")
	gatewayService := InitGateway(GatewayConfig{Name: "api"})
	loadTestService(t, b, gatewayService)
	b.emitServiceInfoInternal()
}

func TestTraceHelpers(t *testing.T) {
	var out bytes.Buffer
	old := colorOutput
	colorOutput = &out
	defer func() { colorOutput = old }()

	b := &Broker{Config: BrokerConfig{NodeId: "trace-node", TraceConfig: TraceConfig{Enabled: true, TraceExpoter: TraceExporterConsole}}, logs: Log{}}
	b.initTrace()
	root := b.startTraceSpan("root", "action", "svc", "root", map[string]interface{}{"a": 1}, "", "", 1)
	child := b.startTraceSpan("child", "action", "svc", "child", nil, "remote-node", root, 2, root)
	b.endTraceSpan(child, errors.New("boom"))
	b.endTraceSpan(root, nil)
	if _, err := b.findSpan("missing"); err == nil {
		t.Fatal("missing span should error")
	}
	if len(b.findTraceChildrens(root)) == 0 || len(b.findTraceChildrensDeep(root)) == 0 {
		t.Fatal("expected trace children")
	}
	b.removeSpanByParent(root)
	b.removeSpan(root)

	tc := initTraceConsole(b)
	spans := []*traceSpan{
		{TraceId: "root", Name: "root", StartTime: 1, FinishTime: 101, Duration: 100, Tags: tags{CallingLevel: 1, RequestId: "root"}},
		{TraceId: "child1", ParentId: "root", Name: "child1", StartTime: 2, FinishTime: 20, Duration: 18, Tags: tags{CallingLevel: 2, RequestId: "root"}},
		{TraceId: "child2", ParentId: "root", Name: "child2", StartTime: 3, FinishTime: 30, Duration: 27, Error: "x", Tags: tags{CallingLevel: 2, RequestId: "root"}},
	}
	ordered := tc.sortSpans(spans)
	if ordered[0].TraceId != "root" || !tc.checkHasChild(ordered, "root") || tc.checkLastSpan(ordered, "child1", "root") {
		t.Fatalf("unexpected trace ordering")
	}
	if indent := tc.getSpanIndent(ordered, ordered[1]); indent == "" {
		t.Fatal("expected child indent")
	}
	if parents := tc.calcTraceParent(ordered, "root", &[]PrintSpanParent{}); len(parents) != 1 {
		t.Fatalf("parents = %#v", parents)
	}
	if got := tc.findTraceSpanByParent(ordered, "root"); len(got) != 2 {
		t.Fatalf("children = %#v", got)
	}
	if got := tc.getAlignedTexts("abcdef", 4); got != "a..." {
		t.Fatalf("aligned text = %q", got)
	}
	if got := tc.drawGauge(0, 100); !strings.Contains(got, "■") {
		t.Fatalf("gauge = %q", got)
	}
	if r("x", 3) != "xxx" {
		t.Fatal("repeat failed")
	}
	tc.ExportSpan(ordered)
}

var colorOutput io.Writer = io.Discard

func init() {
	// Keep color output quiet in tests that call console renderers.
	// The color package writes to stdout directly; this variable exists so the test can
	// document the intent without changing production behavior.
	_ = colorOutput
}
