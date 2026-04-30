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

`LogConsole` is implemented. `LogFile` exists as a constant but does not currently have an exporter implementation.
