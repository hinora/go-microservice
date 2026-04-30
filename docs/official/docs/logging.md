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
backend (zap, logrus, zerolog, an HTTP shipper, Datadog, ...). Implement the
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
