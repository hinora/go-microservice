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
