package goservice

import (
	"errors"
	"sync"
	"time"
)

// CircuitBreakerState represents the current state of a circuit breaker.
type CircuitBreakerState int

const (
	// CircuitClosed is the normal operating state; all calls are allowed through.
	CircuitClosed CircuitBreakerState = iota
	// CircuitOpen means the endpoint has failed too many times; calls are rejected immediately.
	CircuitOpen
	// CircuitHalfOpen allows a single probe call to check whether the endpoint has recovered.
	CircuitHalfOpen
)

// CircuitBreakerConfig holds the global circuit-breaker configuration for the broker.
type CircuitBreakerConfig struct {
	// Enabled activates circuit breaking for all action endpoints (default: false).
	Enabled bool
	// Threshold is the number of consecutive failures that trip the circuit to Open (default: 5).
	Threshold int
	// HalfOpenTimeout is how long to wait after opening the circuit before allowing a probe call (default: 10s).
	HalfOpenTimeout time.Duration
	// SuccessThreshold is the number of consecutive successes in Half-Open state needed to close the circuit (default: 1).
	SuccessThreshold int
}

// endpointCircuitBreaker tracks circuit state for one endpoint (nodeId + service + action).
type endpointCircuitBreaker struct {
	state           CircuitBreakerState
	failures        int
	halfOpenSuccess int
	lastFailureTime time.Time
	mu              sync.Mutex
}

// isAllowed reports whether a new call may proceed through this endpoint.
// When an Open circuit's HalfOpenTimeout has elapsed it is automatically
// transitioned to HalfOpen and true is returned for the first probe call.
func (cb *endpointCircuitBreaker) isAllowed(cfg CircuitBreakerConfig) bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case CircuitClosed:
		return true
	case CircuitOpen:
		halfOpenTimeout := cfg.HalfOpenTimeout
		if halfOpenTimeout <= 0 {
			halfOpenTimeout = 10 * time.Second
		}
		if time.Since(cb.lastFailureTime) >= halfOpenTimeout {
			cb.state = CircuitHalfOpen
			cb.halfOpenSuccess = 0
			return true
		}
		return false
	case CircuitHalfOpen:
		// Block all further calls until the single probe call has completed.
		return false
	}
	return true
}

// recordSuccess updates the circuit state after a successful call.
func (cb *endpointCircuitBreaker) recordSuccess(cfg CircuitBreakerConfig) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case CircuitHalfOpen:
		cb.halfOpenSuccess++
		threshold := cfg.SuccessThreshold
		if threshold <= 0 {
			threshold = 1
		}
		if cb.halfOpenSuccess >= threshold {
			cb.state = CircuitClosed
			cb.failures = 0
			cb.halfOpenSuccess = 0
		}
	case CircuitClosed:
		cb.failures = 0
	}
}

// recordFailure updates the circuit state after a failed call.
func (cb *endpointCircuitBreaker) recordFailure(cfg CircuitBreakerConfig) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures++
	cb.lastFailureTime = time.Now()

	threshold := cfg.Threshold
	if threshold <= 0 {
		threshold = 5
	}
	if cb.state == CircuitHalfOpen || cb.failures >= threshold {
		cb.state = CircuitOpen
		cb.halfOpenSuccess = 0
	}
}

// errCircuitOpen is returned when a call is blocked by an open circuit breaker.
var errCircuitOpen = errors.New("Circuit breaker is open")

// circuitBreakerKey builds the map key for a specific endpoint.
func (b *Broker) circuitBreakerKey(nodeId, serviceName, actionName string) string {
	return nodeId + "." + serviceName + "." + actionName
}

// getOrCreateCircuitBreaker returns the existing circuit breaker for key or creates a new one.
func (b *Broker) getOrCreateCircuitBreaker(key string) *endpointCircuitBreaker {
	b.cbMu.RLock()
	cb, ok := b.circuitBreakers[key]
	b.cbMu.RUnlock()
	if ok {
		return cb
	}

	b.cbMu.Lock()
	defer b.cbMu.Unlock()
	if cb, ok = b.circuitBreakers[key]; ok {
		return cb
	}
	cb = &endpointCircuitBreaker{}
	b.circuitBreakers[key] = cb
	return cb
}
