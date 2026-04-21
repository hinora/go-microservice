package goservice

import (
	"expvar"
	"strings"

	"github.com/zserge/metric"
)

// matchEventPattern reports whether the emitted event name matches a subscriber's pattern.
// Supported wildcards (Moleculer-style):
//   - "*"  matches exactly one segment between dots (e.g. "order.*" matches "order.created")
//   - "**" matches zero or more segments (e.g. "order.**" matches "order.created.hook")
func matchEventPattern(pattern, name string) bool {
	if pattern == name || pattern == "**" {
		return true
	}
	patternParts := strings.Split(pattern, ".")
	nameParts := strings.Split(name, ".")
	return matchEventParts(patternParts, nameParts)
}

// matchEventParts recursively matches split pattern segments against split name segments.
func matchEventParts(pattern, name []string) bool {
	if len(pattern) == 0 {
		return len(name) == 0
	}
	if pattern[0] == "**" {
		// Match zero or more segments.
		for i := 0; i <= len(name); i++ {
			if matchEventParts(pattern[1:], name[i:]) {
				return true
			}
		}
		return false
	}
	if len(name) == 0 {
		return false
	}
	if pattern[0] == "*" || pattern[0] == name[0] {
		return matchEventParts(pattern[1:], name[1:])
	}
	return false
}

func (b *Broker) balancingRoundRobin(name string) (RegistryService, RegistryAction, []RegistryService) {
	var rs RegistryService
	var ra RegistryAction
	var re []RegistryService
	var minCall float64 = 0
	var actions []RegistryAction
	var services []RegistryService
	for _, s := range b.registryServices {
		for _, a := range s.Actions {
			if name == s.Name+"."+a.Name {
				actions = append(actions, a)
				services = append(services, s)
			}
		}
		for _, e := range s.Events {
			if matchEventPattern(e.Name, name) {
				re = append(re, s)
			}
		}
	}
	if len(actions) != 0 {
		// When the circuit breaker is enabled, prefer endpoints whose circuit is not open.
		// Build a filtered list; fall back to all endpoints only if every circuit is open.
		type ep struct {
			svc    RegistryService
			action RegistryAction
		}
		var available []ep
		for i, a := range actions {
			if !b.Config.CircuitBreaker.Enabled {
				available = append(available, ep{services[i], a})
				continue
			}
			key := b.circuitBreakerKey(services[i].Node.NodeId, services[i].Name, a.Name)
			cb := b.getOrCreateCircuitBreaker(key)
			if cb.isAllowed(b.Config.CircuitBreaker) {
				available = append(available, ep{services[i], a})
			}
		}
		if len(available) == 0 {
			// All circuits are open; let callActionOrEvent reject the call via the circuit-breaker check.
			for i, a := range actions {
				available = append(available, ep{services[i], a})
			}
		}

		ra = available[0].action
		rs = available[0].svc
		for _, e := range available {
			nameCheck := MCountCall + "." + e.svc.Node.NodeId + "." + e.svc.Name + "." + e.action.Name
			countCheck := MetricsGetValueCounter(expvar.Get(nameCheck).(metric.Metric))
			if countCheck <= minCall {
				minCall = countCheck
				ra = e.action
				rs = e.svc
			}
		}
		if rs.Name != "" && ra.Name != "" {
			nameCheck := MCountCall + "." + rs.Node.NodeId + "." + rs.Name + "." + ra.Name
			expvar.Get(nameCheck).(metric.Metric).Add(1)
		}
	} else if len(re) != 0 {
		for i := 0; i < len(re); i++ {
			re[i].Actions = []RegistryAction{}
			var events []RegistryEvent
			for _, e := range re[i].Events {
				if matchEventPattern(e.Name, name) {
					events = append(events, e)
				}
			}
			re[i].Events = events
		}
	}
	return rs, ra, re
}
