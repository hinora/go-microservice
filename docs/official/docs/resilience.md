# Resilience

Go Microservice includes retries, timeouts, circuit breakers, bulkheads, and action result caching.

## Timeouts

Timeout precedence is:

1. `CallOpts.Timeout`
2. `Action.Timeout`
3. `BrokerConfig.RequestTimeOut`

```go
goservice.Action{
    Name:    "slow",
    Timeout: 1000,
    Handle:  slow,
}
```

## Retries

Configure broker defaults with `BrokerConfig.Retry`, or override a single call with `CallOpts.Retry`.

```go
Retry: goservice.RetryPolicy{
    MaxRetries: 2,
    RetryDelay: 100,
    Factor:     2,
},
```

Retries are attempted for transient errors such as timeouts and open circuits. Handler-level application errors are returned without retrying.

## Circuit breaker

```go
CircuitBreaker: goservice.CircuitBreakerConfig{
    Enabled:          true,
    Threshold:        5,
    HalfOpenTimeout:  10 * time.Second,
    SuccessThreshold: 1,
},
```

Circuit breakers are tracked per endpoint (`node.service.action`). Consecutive failures open the circuit, a later probe moves it to half-open, and enough successful probes close it.

## Bulkheads

Bulkheads limit concurrent executions for a specific action.

```go
goservice.Action{
    Name: "process",
    Bulkhead: &goservice.BulkheadConfig{
        MaxConcurrency: 10,
        MaxQueueSize:   20,
    },
    Handle: process,
}
```

Calls beyond the concurrency and queue limit fail with `Bulkhead queue is full`.

## Caching

Action caching stores successful results in a local in-memory cache.

```go
goservice.Action{
    Name: "find",
    Cache: &goservice.CacheConfig{
        TTL:  30 * time.Second,
        Keys: []string{"id"},
    },
    Handle: find,
}
```

`Keys` limits which parameter keys are included in the cache key. Empty `Keys` uses all params. A zero TTL means cached entries do not expire automatically.
