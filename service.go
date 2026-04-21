package goservice

type CallOpts struct {
	// Retry overrides the broker-level retry policy for this specific call.
	Retry *RetryPolicy
	// Timeout overrides the broker-level request timeout for this specific call (milliseconds; 0 = use default).
	Timeout int
}
type Call func(action string, params interface{}, opts ...CallOpts) (interface{}, error)

// BroadcastFunc sends an event to ALL subscribers on every node, bypassing group-based balancing.
type BroadcastFunc func(event string, params interface{})

type Context struct {
	RequestId         string
	TraceParentId     string
	TraceParentRootId string
	ResponseId        string
	Params            interface{}
	Meta              interface{}
	FromService       string
	FromAction        string
	FromEvent         string
	FromNode          string
	CallingLevel      int
	Call              Call
	// Broadcast sends an event to ALL instances on every node (no group balancing).
	Broadcast BroadcastFunc
	Service   *Service
}

func (c *Context) LogInfo(message string) {
	c.Service.Broker.LogInfo(message)
}
func (c *Context) LogWarning(message string) {
	c.Service.Broker.LogWarning(message)
}
func (c *Context) LogError(message string) {
	c.Service.Broker.LogError(message)
}

type Method int

const (
	GET Method = iota + 1
	POST
	PUT
	DELETE
	PATCH
	HEAD
	OPTIONS
)

func (m Method) String() string {
	switch m {
	case GET:
		return "GET"
	case POST:
		return "POST"
	case PUT:
		return "PUT"
	case DELETE:
		return "DELETE"
	case PATCH:
		return "PATCH"
	case HEAD:
		return "HEAD"
	case OPTIONS:
		return "OPTIONS"
	}
	return ""
}

type Rest struct {
	Method Method `json:"method" mapstructure:"method"`
	Path   string `json:"path" mapstructure:"path"`
}
type Action struct {
	Name   string
	Params interface{}
	// Schema is an optional validation schema applied to incoming params before the handler runs.
	Schema map[string]ParamRule
	Rest   Rest
	// Timeout overrides the broker-level request timeout for this action (milliseconds; 0 = use broker default).
	// Per-call timeout set in CallOpts.Timeout takes precedence over this value.
	Timeout int
	// Cache enables result caching for this action. nil means caching is disabled.
	Cache *CacheConfig
	// Bulkhead limits concurrent in-flight calls to this action handler. nil means unlimited.
	Bulkhead *BulkheadConfig
	// Hooks contains Before/After/Error lifecycle hooks for this specific action.
	Hooks  ActionHooks
	Handle func(*Context) (interface{}, error)
}
type Event struct {
	Name   string
	Params interface{}
	Handle func(*Context)
}
type Service struct {
	Name    string
	// Version is an optional version string (e.g. "2"). Versioned services are
	// addressable as "v{version}.{name}.{action}" (e.g. "v2.math.add") in addition
	// to the plain "math.add" which resolves to the latest registered version.
	Version string
	// Mixins lists service fragments whose actions, events, and hooks are merged into
	// this service at load time. A service's own definitions override same-named mixin entries.
	Mixins  []Service
	// Dependencies lists service names that must be available in the registry before this
	// service's Started lifecycle hook is invoked.
	Dependencies []string
	Actions []Action
	Events  []Event
	Started func(*Context)
	Stoped  func(*Context)
	Broker  *Broker
	// Hooks contains service-wide Before/After/Error hooks applied to every action in this service.
	Hooks ActionHooks
}

// qualifiedName returns the registry-addressable service name.
// When Version is set it returns "v{version}.{name}" (e.g. "v2.math");
// otherwise it returns the plain Name.
func (s *Service) qualifiedName() string {
	if s.Version == "" {
		return s.Name
	}
	return "v" + s.Version + "." + s.Name
}
