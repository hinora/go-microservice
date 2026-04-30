# Tracing

Tracing records spans for broker calls, actions, events, service startup, broadcasts, and nested calls.

```go
TraceConfig: goservice.TraceConfig{
    Enabled:      true,
    TraceExpoter: goservice.TraceExporterConsole,
},
```

## TraceConfig

| Field | Description |
| --- | --- |
| `Enabled` | Enables tracing. |
| `WindowTime` | Reserved configuration field. |
| `TraceExpoter` | Exporter type. Use `TraceExporterConsole`. |
| `TraceExporterConfig` | Optional exporter-specific config. |

## Span data

Spans include IDs, parent IDs, service/action names, caller node, request ID, params, duration, errors, and cache-hit information.

## Exporters

### Console

`TraceExporterConsole` renders an ASCII tree of spans to stdout, useful for
local development.

### Datadog APM

`TraceExporterDDDog` ships spans to the local
[Datadog APM agent](https://docs.datadoghq.com/tracing/) using the JSON
`v0.4/traces` endpoint. Spans are converted to Datadog's wire format: each
broker request becomes a single trace and child calls inherit the parent
`trace_id`. Errors surface as `error=1` with `error.message` meta.

```go
TraceConfig: goservice.TraceConfig{
    Enabled:      true,
    TraceExpoter: goservice.TraceExporterDDDog,
    TraceExporterConfig: goservice.TraceDatadogConfig{
        AgentURL: "http://localhost:8126/v0.4/traces", // or DD_TRACE_AGENT_URL
        Service:  "my-service",
        Env:      "prod",
        Version:  "1.2.3",
    },
},
```

| Field | Description |
| --- | --- |
| `AgentURL` | Trace endpoint URL. Falls back to `DD_TRACE_AGENT_URL` then `http://localhost:8126/v0.4/traces`. |
| `Service` | Default Datadog service name. Defaults to `BrokerConfig.NodeId`. |
| `Env` | Datadog `env` meta tag. |
| `Version` | Datadog `version` meta tag. |
| `HTTPClient` | Override the HTTP client used for shipping (mainly for tests). |
