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

`TraceExporterConsole` is implemented. `TraceExporterDDDog` exists as a constant but does not currently have an exporter implementation.
