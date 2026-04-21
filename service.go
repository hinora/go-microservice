package goservice

type CallOpts struct {
	// Retry overrides the broker-level retry policy for this specific call.
	Retry *RetryPolicy
	// Timeout overrides the broker-level request timeout for this specific call (milliseconds; 0 = use default).
	Timeout int
}
type Call func(action string, params interface{}, opts ...CallOpts) (interface{}, error)
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
	Service           *Service
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
		break
	case POST:
		return "POST"
		break
	case PUT:
		return "PUT"
		break
	case DELETE:
		return "DELETE"
		break
	case PATCH:
		return "PATCH"
		break
	case HEAD:
		return "HEAD"
		break
	case OPTIONS:
		return "OPTIONS"
		break
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
	Actions []Action
	Events  []Event
	Started func(*Context)
	Stoped  func(*Context)
	Broker  *Broker
	// Hooks contains service-wide Before/After/Error hooks applied to every action in this service.
	Hooks ActionHooks
}
