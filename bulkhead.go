package goservice

import "errors"

// BulkheadConfig limits concurrent execution of a single action handler to prevent
// resource exhaustion. Each action with a non-nil Bulkhead config gets its own
// independent concurrency pool.
type BulkheadConfig struct {
	// MaxConcurrency is the maximum number of simultaneously executing calls (>0 required).
	MaxConcurrency int
	// MaxQueueSize is the number of additional callers that may wait for a free slot.
	// Requests exceeding this limit are rejected immediately with errBulkheadFull.
	MaxQueueSize int
}

// errBulkheadFull is returned when both the concurrency pool and wait queue are saturated.
var errBulkheadFull = errors.New("Bulkhead queue is full")

// bulkheadState manages concurrency for one action.
type bulkheadState struct {
	sem   chan struct{} // capacity = MaxConcurrency
	queue chan struct{} // capacity = MaxQueueSize
}

func newBulkheadState(cfg BulkheadConfig) *bulkheadState {
	maxC := cfg.MaxConcurrency
	if maxC <= 0 {
		maxC = 1
	}
	maxQ := cfg.MaxQueueSize
	if maxQ < 0 {
		maxQ = 0
	}
	return &bulkheadState{
		sem:   make(chan struct{}, maxC),
		queue: make(chan struct{}, maxQ),
	}
}

// acquire tries to obtain one concurrency slot, blocking if the wait queue has room.
// Returns errBulkheadFull immediately when both the concurrency pool and queue are full.
func (bs *bulkheadState) acquire() error {
	// Fast path: concurrency slot available.
	select {
	case bs.sem <- struct{}{}:
		return nil
	default:
	}
	// Slow path: try to enter the wait queue.
	select {
	case bs.queue <- struct{}{}:
	default:
		return errBulkheadFull
	}
	// Block until a concurrency slot opens.
	bs.sem <- struct{}{}
	<-bs.queue
	return nil
}

// release frees one concurrency slot.
func (bs *bulkheadState) release() {
	<-bs.sem
}
