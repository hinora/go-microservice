# Logging

Enable console logging through `BrokerConfig.LoggerConfig`.

```go
LoggerConfig: goservice.Logconfig{
    Enable:   true,
    Type:     goservice.LogConsole,
    LogLevel: goservice.LogTypeInfo,
},
```

## Levels

| Constant | Description |
| --- | --- |
| `LogTypeInfo` | Informational messages. |
| `LogTypeWarning` | Warnings and errors. |
| `LogTypeError` | Errors only. |

A configured `LogLevel` suppresses messages below that level.

## Handler logging

Use context helpers inside actions and events:

```go
ctx.LogInfo("processing started")
ctx.LogWarning("using fallback")
ctx.LogError("processing failed")
```

## Exporters

The framework follows a pluggable model similar to
[Moleculer's logging](https://moleculer.services/docs/0.14/logging): you pick a
built-in exporter or supply your own.

### Console (built-in)

Colorized output to stdout. This is the default `LogConsole` exporter shown
above.

### File (built-in)

Appends each entry as a JSON line to `FilePath`. When `FilePath` is empty the
file exporter is disabled (no-op).

```go
LoggerConfig: goservice.Logconfig{
    Enable:   true,
    Type:     goservice.LogFile,
    LogLevel: goservice.LogTypeInfo,
    FilePath: "/var/log/goservice.log",
},
```

### Custom exporter

Use `LogCustom` together with `Logconfig.Custom` to plug in any third-party
backend (zap, logrus, zerolog, an HTTP shipper, ...). Implement the
`LoggerExternal` interface:

```go
type LoggerExternal interface {
    Start()              // runs in its own goroutine and blocks
    WriteLog(LogData)    // called by the broker for each log line
}
```

Then pass your implementation in:

```go
LoggerConfig: goservice.Logconfig{
    Enable:   true,
    Type:     goservice.LogCustom,
    LogLevel: goservice.LogTypeInfo,
    Custom:   myZapAdapter,
},
```

### Datadog (built-in)

Use `LogDatadog` to ship log entries to the
[Datadog HTTP log intake](https://docs.datadoghq.com/api/latest/logs/#send-logs).
Logs are batched and flushed in the background — `WriteLog` never blocks the
broker call site.

```go
LoggerConfig: goservice.Logconfig{
    Enable:   true,
    Type:     goservice.LogDatadog,
    LogLevel: goservice.LogTypeInfo,
    Datadog: &goservice.LoggerDatadog{
        APIKey:  os.Getenv("DD_API_KEY"), // or rely on DD_API_KEY env
        Site:    "datadoghq.com",         // or DD_SITE
        Service: "my-service",
        Source:  "goservice",
        Tags:    "env:prod,team:platform",
    },
},
```

When `APIKey` is empty the exporter falls back to the `DD_API_KEY` environment
variable; when `Site` is empty it falls back to `DD_SITE` and finally to
`datadoghq.com`. With no API key configured the exporter quietly drops entries
instead of buffering forever — useful for local development.

| Field | Description |
| --- | --- |
| `APIKey` | Datadog API key. Falls back to `DD_API_KEY`. |
| `Site` | Datadog site. Falls back to `DD_SITE` then `datadoghq.com`. |
| `Service` | Service name attached to every log entry. |
| `Source` | Datadog `ddsource` tag. Defaults to `goservice`. |
| `Hostname` | Reported host. Defaults to `os.Hostname()`. |
| `Tags` | Comma-separated `ddtags` value. |
| `Endpoint` | Override the full intake URL (mainly for tests). |
| `FlushInterval` | Background flush cadence. Defaults to `1s`. |
| `FlushBatchSize` | Maximum entries per HTTP batch. Defaults to `100`. |
