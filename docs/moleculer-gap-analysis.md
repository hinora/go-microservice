# Moleculer Gap Analysis

> What Moleculer 0.14 has that **go-microservice** doesn't (yet).  
> Reference: https://moleculer.services/docs/0.14/

---

## What We Already Have

| Feature | Status |
|---|---|
| Service broker | ✅ |
| Actions (request/response RPC) | ✅ |
| Events (pub/sub fire-and-forget) | ✅ |
| Service discovery (Redis) | ✅ |
| Transporter (Redis) | ✅ |
| Load balancing (least-connections) | ✅ |
| API gateway (HTTP) | ✅ |
| Distributed tracing (console exporter) | ✅ |
| Structured logging | ✅ |
| Service lifecycle hooks (`Started`, `Stopped`) | ✅ |
| Context object | ✅ |
| Action Hooks (Before / After / Error) | ✅ |
| Parameter Validation | ✅ |
| Retry Policy | ✅ |
| Circuit Breaker | ✅ |
| Per-Action / Per-Call Timeout | ✅ |
| Middleware System | ✅ |
| Built-in In-memory Caching | ✅ |
| Bulkhead (Concurrency Limiter) | ✅ |
| Event Broadcast (`broker.Broadcast`) | ✅ |
| Wildcard Event Subscriptions (`order.*`, `**`) | ✅ |
| Service Mixins | ✅ |
| Versioned Services (`v2.math.add`) | ✅ |
| Service Dependencies | ✅ |
| Node Ping | ✅ |
| MessagePack Serializer | ✅ |

---

## Gaps — What Moleculer Has but We Don't

---

### 1. Middleware System ✅ Implemented

Pluggable `Middleware` structs registered in `BrokerConfig.Middlewares`.  
The `LocalAction` hook wraps every local action handler in an onion/chain-of-responsibility pattern.  
Registered in order; the first middleware is the outermost wrapper.

**Example:**
```go
goservice.Middleware{
    LocalAction: func(ctx *goservice.Context, action goservice.Action, next func(*goservice.Context) (interface{}, error)) (interface{}, error) {
        // before
        res, err := next(ctx)
        // after
        return res, err
    },
}
```

---

### 2. Action Hooks (Before / After / Error) ✅ Implemented

Declared on `Action.Hooks` (action-specific) or `Service.Hooks` (service-wide).  
Service-level hooks run before action-level hooks.

---

### 3. Built-in Caching ✅ Implemented

Set `Action.Cache = &goservice.CacheConfig{TTL: 30 * time.Second}` to enable automatic
result caching for that action.

- Default adapter: in-memory (`memoryCache`) with periodic GC.
- Cache key: `actionName + ":" + JSON(params)`. Restrict key fields via `CacheConfig.Keys`.
- `TraceSpan.Tags.FromCache` is set to `true` when a cached result is returned.
- Manual invalidation: `broker.cache.del("serviceName.actionName:*")` (prefix wildcard supported).

---

### 4. Circuit Breaker ✅ Implemented

States: Closed → Open → Half-Open.  
Config via `BrokerConfig.CircuitBreaker` (`Enabled`, `Threshold`, `HalfOpenTimeout`, `SuccessThreshold`).

---

### 5. Retry Policy ✅ Implemented

`BrokerConfig.Retry` (broker-level default) or `CallOpts.Retry` (per-call override).  
Supports fixed and exponential back-off (`Factor`).

---

### 6. Bulkhead (Concurrency Limiter) ✅ Implemented

Set `Action.Bulkhead = &goservice.BulkheadConfig{MaxConcurrency: 10, MaxQueueSize: 100}`.

- Channel-based semaphore; one independent pool per action.
- Excess requests queue up to `MaxQueueSize`; further requests are rejected immediately with `errBulkheadFull`.

---

### 7. Per-Action / Per-Call Timeout ✅ Implemented

Priority: `CallOpts.Timeout` > `Action.Timeout` > `BrokerConfig.RequestTimeOut`.  
`Action.Timeout` is stored in `RegistryAction` and propagated to callers via the registry.

---

### 8. Parameter Validation ✅ Implemented

Set `Action.Schema` to a `map[string]goservice.ParamRule`.  
Supported rules: `Type` (`string`, `number`, `bool`, `array`, `object`, `any`), `Required`, `Min`, `Max`.  
Returns a structured `*ValidationError` with per-field details before the handler runs.

---

### 9. Service Mixins ✅ Implemented

Add reusable service fragments to `Service.Mixins []Service`.  
At `LoadService` time the broker merges mixin actions, events, and hooks into the service.  
The service's own definitions override same-named mixin entries.

```go
var authMixin = goservice.Service{
    Hooks: goservice.ActionHooks{
        Before: []goservice.BeforeHookFunc{ /* auth check */ },
    },
}
b.LoadService(&goservice.Service{
    Name:   "orders",
    Mixins: []goservice.Service{authMixin},
    ...
})
```

---

### 10. Versioned Services ✅ Implemented

Set `Service.Version = "2"` to register the service as `v2.math`.  
Call it with `ctx.Call("v2.math.add", params)`.  
An unversioned call `math.add` continues to resolve to the service registered as `math`.

---

### 11. Event Groups & Broadcast ✅ Implemented (Broadcast)

| Mode | Method | Behaviour |
|---|---|---|
| `emit` (current) | `ctx.Call("event.name", params)` | Delivers to all registered subscribers (least-connections) |
| `broadcast` | `broker.Broadcast("event.name", params)` or `ctx.Broadcast(...)` | Explicitly delivers to **all** instances on every node |

---

### 12. Wildcard Event Subscriptions ✅ Implemented

Register an event handler with a wildcard pattern:

| Pattern | Matches |
|---|---|
| `order.*` | `order.created`, `order.updated` |
| `order.**` | `order.created`, `order.created.v2` |
| `**` | every event |

The broker evaluates patterns at dispatch time in both `emit` and `broadcast`.

---

### 13. Service Dependencies ✅ Implemented

List service names that must be available before `Started` is called:

```go
goservice.Service{
    Name:         "payments",
    Dependencies: []string{"users", "v2.orders"},
    Started: func(ctx *goservice.Context) { /* safe to call users and orders here */ },
}
```

The broker polls the registry every 500 ms and gives up after 60 s (logging a warning).

---

### 14. Node Ping / Health Check ✅ Implemented (Ping)

```go
latencyMs, err := broker.Ping("Node-2")
```

Returns round-trip latency in milliseconds.  
Requires Redis discovery to be enabled. Self-ping always returns 0.

---

### 15. Multiple Transporter Support

We only support Redis. Moleculer supports NATS, MQTT, AMQP, TCP, Kafka, and more.

**High-level idea:**
- Transporter abstraction already exists. New transporters just implement the `Transporter` interface.
- Priority candidates: **NATS** (low-latency), **AMQP/RabbitMQ** (durability), **TCP** (no broker dependency).

---

### 16. Multiple Discovery Backends

We only support Redis. Moleculer supports etcd, Consul, Zookeeper, and a local-only mode (no external dependency).

**High-level idea:**
- Discovery abstraction already exists. New backends implement the `Discovery` interface.
- Priority candidate: **local/in-process** mode (no Redis needed for single-node dev).

---

### 17. Pluggable Metrics Exporters

We have basic `expvar` counters. Moleculer integrates with Prometheus, DataDog, StatsD, etc.

**High-level idea:**
- Metrics exporter interface with adapters: Prometheus (most common), DataDog, StatsD.
- Expose standard metrics: request count, error rate, latency histograms, active calls.
- `GET /metrics` endpoint on the API gateway for Prometheus scraping.

---

### 18. Pluggable Trace Exporters

We have console + a DataDog stub. Moleculer integrates with Jaeger, Zipkin, DataDog, and OpenTelemetry.

**High-level idea:**
- Implement the `TraceExporter` interface for Jaeger (via OTLP or Thrift) and Zipkin.
- Or integrate with the **OpenTelemetry SDK** so any OTLP-compatible backend works out of the box.

---

### 19. Multiple Serializers ✅ Partially Implemented

JSON serializer was already present.  
**MessagePack** (`SerializerMsgPackEncode` / `DeserializerMsgPackDecode`) has been added via
`github.com/vmihailenco/msgpack/v5`.

Remaining work: plumb the serializer choice through `BrokerConfig` so all wire messages use
the same codec automatically.

---

### 20. REPL / CLI

Moleculer ships a developer REPL for interactive service introspection at runtime.

**High-level idea:**
- `broker.Repl()` starts an interactive CLI.
- Commands: `services`, `actions`, `nodes`, `call <action> <params>`, `emit <event> <payload>`, `bench <action>`.
- Useful during development and debugging without writing extra code.

---

### 21. Hot Reload (Dev Mode)

Moleculer can reload changed services without restarting the process.

**High-level idea:**
- Watch service source files; on change, gracefully unload and reload the service.
- Practical in Go via plugin system or by restarting individual goroutines with new config.
- Scope: mainly a developer ergonomics feature.

---

## Priority Recommendation

| Priority | Feature | Value | Complexity | Status |
|---|---|---|---|---|
| 🔴 High | Parameter Validation | Prevents bad data from reaching handlers | Low | ✅ Done |
| 🔴 High | Action Hooks (Before/After/Error) | Foundation for auth, logging, transform | Medium | ✅ Done |
| 🔴 High | Retry Policy | Core resilience | Low | ✅ Done |
| 🔴 High | Circuit Breaker | Core resilience | Medium | ✅ Done |
| 🟡 Medium | Middleware System | Cross-cutting concerns | Medium | ✅ Done |
| 🟡 Medium | Built-in Caching | Performance | Medium | ✅ Done |
| 🟡 Medium | Bulkhead | Overload protection | Low | ✅ Done |
| 🟡 Medium | Event Broadcast + Wildcard | Messaging flexibility | Medium | ✅ Done |
| 🟡 Medium | Service Mixins | Code reuse | Low | ✅ Done |
| 🟡 Medium | Versioned Services | API evolution | Medium | ✅ Done |
| 🟡 Medium | Per-Action Timeout | Granular control | Low | ✅ Done |
| 🟢 Low | Node Ping / Health Check | Ops visibility | Low | ✅ Done |
| 🟢 Low | Service Dependencies | Boot ordering | Low | ✅ Done |
| 🟢 Low | MessagePack Serializer | Performance | Low | ✅ Done (codec; wire plumbing pending) |
| 🟢 Low | Prometheus Metrics | Observability | Medium | ❌ Not yet |
| 🟢 Low | OpenTelemetry Tracing | Observability | Medium | ❌ Not yet |
| 🟢 Low | Additional Transporters (NATS) | Flexibility | Medium | ❌ Not yet |
| 🟢 Low | REPL / CLI | Dev ergonomics | Medium | ❌ Not yet |
| 🟢 Low | Hot Reload | Dev ergonomics | High | ❌ Not yet |


---

## Gaps — What Moleculer Has but We Don't

---

### 1. Middleware System

Moleculer supports pluggable middleware that can wrap the broker and action pipeline.  
Middleware can intercept calls before/after they reach a handler — useful for auth, logging, rate limiting, etc.

**High-level idea:**
- A list of middleware functions registered on the broker at startup.
- Each middleware wraps the action handler call (like an onion/chain-of-responsibility pattern).
- Middleware hooks: `localAction`, `remoteAction`, `emit`, `publish`, `call`, etc.

---

### 2. Action Hooks (Before / After / Error)

Per-action (or per-service) hooks that run before the handler, after it, or on error.  
Different from middleware — these are declared inside the service definition itself.

**High-level idea:**
- `Before []Hook` — runs before handler, can modify params or abort.
- `After []Hook` — runs after handler, can transform the result.
- `Error []Hook` — runs if handler returns an error, can recover or rethrow.

---

### 3. Built-in Caching

Moleculer supports automatic action result caching (Memory or Redis adapters).  
When a cached action is called with the same params, the broker returns the cached result immediately without executing the handler.

**High-level idea:**
- `Cache bool` flag (or struct with TTL/keys config) on `Action`.
- Cache adapters: in-memory (default), Redis.
- Cache key generated from action name + params.
- Manual cache invalidation via patterns.
- Trace span should report `fromCache: true` (we already have the field, just not the logic).

---

### 4. Circuit Breaker

Prevents cascading failures by stopping calls to a failing remote service for a period.

**High-level idea:**
- States: **Closed** (normal) → **Open** (short-circuit, reject calls) → **Half-Open** (trial call).
- Config: failure-rate threshold, minimum request count, window time, half-open probe time.
- Applied per endpoint (per node+service+action combination).
- When open, the broker tries the next available node instead of failing immediately.

---

### 5. Retry Policy

Automatically retry a failed action call a configurable number of times before returning the error.

**High-level idea:**
- Config on `CallOpts` or globally on `BrokerConfig`: `MaxRetries`, `RetryDelay`.
- Retries should back off (fixed or exponential).
- Only retry on transient/network errors, not application-level errors.

---

### 6. Bulkhead (Concurrency Limiter)

Limits the number of concurrent in-flight calls to a single action to prevent resource exhaustion.

**High-level idea:**
- `Bulkhead` config on `Action`: `MaxConcurrency int`, `MaxQueueSize int`.
- Excess requests wait in a queue up to `MaxQueueSize`; further requests are rejected immediately.
- Isolates actions from each other — a slow action can't starve the whole service.

---

### 7. Per-Action / Per-Call Timeout

We currently have a single global `RequestTimeOut`. Moleculer supports timeout at multiple granularities:

**High-level idea:**
- Timeout on `Action` definition (overrides global).
- Timeout on `CallOpts` per individual call (overrides action-level).
- Priority: call-level > action-level > broker-level.

---

### 8. Parameter Validation

Moleculer validates action/event params against a schema before calling the handler.  
Invalid params return a structured validation error immediately.

**High-level idea:**
- `Params` on `Action` / `Event` becomes a real validation schema (not just a hint).
- Pluggable validator interface — default implementation using a schema similar to [Fastest Validator](https://github.com/icebob/fastest-validator) (type, required, min/max, etc.).
- Return a `ValidationError` with field-level details before the handler runs.

---

### 9. Service Mixins

Reusable fragments of service definitions (actions, events, hooks, lifecycle) that can be composed into multiple services.

**High-level idea:**
- `Mixins []Service` field on `Service`.
- The broker merges mixin actions/events/hooks into the service at load time.
- Later declarations override mixin declarations (same-name action wins).
- Useful for shared authorization hooks, common CRUD actions, etc.

---

### 10. Versioned Services

Multiple versions of the same service can run simultaneously.

**High-level idea:**
- `Version string` (or `int`) field on `Service`.
- Action calls use `v2.math.add` syntax to target a specific version.
- Unversioned call `math.add` resolves to the latest (or a configurable default) version.
- Allows zero-downtime API evolution.

---

### 11. Event Groups & Broadcast

Moleculer distinguishes two event-delivery modes:

| Mode | Moleculer | Current state |
|---|---|---|
| `emit` | Deliver to **one** instance per service group | ✅ (approx — round-robin) |
| `broadcast` | Deliver to **all** instances across all nodes | ❌ missing |

**High-level idea:**
- `Event.Group string` — events with the same group name are balanced among group members (current behavior).
- `broker.Broadcast(event, payload)` — sends to every subscriber regardless of group.
- `ctx.Broadcast(...)` inside handlers.

---

### 12. Wildcard Event Subscriptions

Moleculer supports glob-style wildcard patterns in event names.

**High-level idea:**
- `order.*` matches `order.created`, `order.updated`, `order.deleted`.
- `**` matches any depth.
- The broker resolves subscriptions against the pattern at publish time.

---

### 13. Service Dependencies

A service can declare that it requires other services to be available before its `Started` hook runs.

**High-level idea:**
- `Dependencies []string` field on `Service` (list of service names).
- Broker waits (with timeout) until all listed services appear in the registry before calling `Started`.

---

### 14. Node Ping / Health Check

Moleculer supports pinging nodes to measure latency and checking node health.

**High-level idea:**
- `broker.Ping(nodeId)` returns round-trip latency.
- `broker.GetHealthInfo()` returns CPU, memory, event loop stats.
- Useful for dashboards and alerting.

---

### 15. Multiple Transporter Support

We only support Redis. Moleculer supports NATS, MQTT, AMQP, TCP, Kafka, and more.

**High-level idea:**
- Transporter abstraction already exists. New transporters just implement the `Transporter` interface.
- Priority candidates: **NATS** (low-latency), **AMQP/RabbitMQ** (durability), **TCP** (no broker dependency).

---

### 16. Multiple Discovery Backends

We only support Redis. Moleculer supports etcd, Consul, Zookeeper, and a local-only mode (no external dependency).

**High-level idea:**
- Discovery abstraction already exists. New backends implement the `Discovery` interface.
- Priority candidate: **local/in-process** mode (no Redis needed for single-node dev).

---

### 17. Pluggable Metrics Exporters

We have basic `expvar` counters. Moleculer integrates with Prometheus, DataDog, StatsD, etc.

**High-level idea:**
- Metrics exporter interface with adapters: Prometheus (most common), DataDog, StatsD.
- Expose standard metrics: request count, error rate, latency histograms, active calls.
- `GET /metrics` endpoint on the API gateway for Prometheus scraping.

---

### 18. Pluggable Trace Exporters

We have console + a DataDog stub. Moleculer integrates with Jaeger, Zipkin, DataDog, and OpenTelemetry.

**High-level idea:**
- Implement the `TraceExporter` interface for Jaeger (via OTLP or Thrift) and Zipkin.
- Or integrate with the **OpenTelemetry SDK** so any OTLP-compatible backend works out of the box.

---

### 19. Multiple Serializers

We only support JSON. Moleculer supports MessagePack, Avro, Protobuf, etc.

**High-level idea:**
- Serializer interface: `Serialize(interface{}) ([]byte, error)` / `Deserialize([]byte) (interface{}, error)`.
- Adapters: **MessagePack** (compact binary, drop-in), **Protobuf** (strongly typed, schema-based).
- Configured per-broker; all nodes in a cluster must use the same serializer.

---

### 20. REPL / CLI

Moleculer ships a developer REPL for interactive service introspection at runtime.

**High-level idea:**
- `broker.Repl()` starts an interactive CLI.
- Commands: `services`, `actions`, `nodes`, `call <action> <params>`, `emit <event> <payload>`, `bench <action>`.
- Useful during development and debugging without writing extra code.

---

### 21. Hot Reload (Dev Mode)

Moleculer can reload changed services without restarting the process.

**High-level idea:**
- Watch service source files; on change, gracefully unload and reload the service.
- Practical in Go via plugin system or by restarting individual goroutines with new config.
- Scope: mainly a developer ergonomics feature.

---

## Priority Recommendation

| Priority | Feature | Value | Complexity |
|---|---|---|---|
| 🔴 High | Parameter Validation | Prevents bad data from reaching handlers | Low |
| 🔴 High | Action Hooks (Before/After/Error) | Foundation for auth, logging, transform | Medium |
| 🔴 High | Retry Policy | Core resilience | Low |
| 🔴 High | Circuit Breaker | Core resilience | Medium |
| 🟡 Medium | Middleware System | Cross-cutting concerns | Medium |
| 🟡 Medium | Built-in Caching | Performance | Medium |
| 🟡 Medium | Bulkhead | Overload protection | Low |
| 🟡 Medium | Event Broadcast + Wildcard | Messaging flexibility | Medium |
| 🟡 Medium | Service Mixins | Code reuse | Low |
| 🟡 Medium | Versioned Services | API evolution | Medium |
| 🟡 Medium | Per-Action Timeout | Granular control | Low |
| 🟢 Low | Node Ping / Health Check | Ops visibility | Low |
| 🟢 Low | Service Dependencies | Boot ordering | Low |
| 🟢 Low | Prometheus Metrics | Observability | Medium |
| 🟢 Low | OpenTelemetry Tracing | Observability | Medium |
| 🟢 Low | Additional Transporters (NATS) | Flexibility | Medium |
| 🟢 Low | MessagePack Serializer | Performance | Low |
| 🟢 Low | REPL / CLI | Dev ergonomics | Medium |
| 🟢 Low | Hot Reload | Dev ergonomics | High |
