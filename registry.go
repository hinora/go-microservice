package goservice

type RegistryNode struct {
	NodeId     string   `json:"node_id" mapstructure:"node_id"`
	IP         []string `json:"ip" mapstructure:"ip"`
	LastActive int      `json:"last_active" mapstructure:"last_active"`
}
type RegistryAction struct {
	Name string      `json:"name" mapstructure:"name"`
	Rest Rest        `json:"rest" mapstructure:"rest"`
	// Params is the raw parameter hint (informational only; actual validation uses Action.Schema).
	Params  interface{} `json:"params" mapstructure:"params"`
	// Timeout is the action-level timeout in milliseconds (0 = use broker default).
	// A per-call CallOpts.Timeout takes precedence over this value.
	Timeout int         `json:"timeout" mapstructure:"timeout"`
}
type RegistryEvent struct {
	Name   string      `json:"name" mapstructure:"name"`
	Params interface{} `json:"params" mapstructure:"params"`
}
type RegistryService struct {
	Node    RegistryNode     `json:"node" mapstructure:"node"`
	Name    string           `json:"name" mapstructure:"name"`
	// Version is the service version string (e.g. "2"). Empty means unversioned.
	// Versioned services are addressed as "v{version}.{name}.{action}" (e.g. "v2.math.add").
	Version string           `json:"version" mapstructure:"version"`
	Actions []RegistryAction `json:"actions" mapstructure:"actions"`
	Events  []RegistryEvent  `json:"events" mapstructure:"events"`
}
