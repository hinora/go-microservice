package goservice

import (
	"fmt"
	"time"
)

// RetryPolicy configures automatic retry behaviour for action calls.
type RetryPolicy struct {
	// MaxRetries is the maximum number of additional attempts after the first failure (0 = no retries).
	MaxRetries int
	// RetryDelay is the base delay in milliseconds between retry attempts.
	RetryDelay int
	// Factor is the exponential-backoff multiplier applied to RetryDelay after each attempt.
	// 0 or 1.0 produces a fixed delay; 2.0 doubles the delay on every retry.
	Factor float64
}

// delayForAttempt returns the sleep duration before the nth retry (0-indexed).
func (r RetryPolicy) delayForAttempt(n int) time.Duration {
	if r.RetryDelay <= 0 {
		return 0
	}
	factor := r.Factor
	if factor <= 0 {
		factor = 1.0
	}
	delay := float64(r.RetryDelay)
	for i := 0; i < n; i++ {
		delay *= factor
	}
	return time.Duration(delay) * time.Millisecond
}

// callWithRetry wraps callActionOrEvent with the retry policy from callOpts or
// the broker-level default.  Only transient errors (timeout, circuit open) trigger
// a retry; application-level errors returned by the handler are propagated immediately.
func (b *Broker) callWithRetry(ctx Context, actionName string, params interface{}, callOpts CallOpts, callerService string, callerAction string, callerEvent string) (ResponseTranferData, error) {
	policy := b.Config.Retry
	if callOpts.Retry != nil {
		policy = *callOpts.Retry
	}

	var result ResponseTranferData
	var lastErr error

	for attempt := 0; attempt <= policy.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := policy.delayForAttempt(attempt - 1)
			if delay > 0 {
				time.Sleep(delay)
			}
			b.LogInfo(fmt.Sprintf("Retrying `%s` (attempt %d/%d)", actionName, attempt, policy.MaxRetries))
		}

		result, lastErr = b.callActionOrEvent(ctx, actionName, params, callOpts, callerService, callerAction, callerEvent)
		if lastErr == nil {
			// Either a clean success or an application-level handler error
			// (result.Error == true with lastErr == nil); do not retry either case.
			return result, nil
		}
		// lastErr != nil means a transient error (timeout, circuit open) – retry.
	}

	return result, lastErr
}
