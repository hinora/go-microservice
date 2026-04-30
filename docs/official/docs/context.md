# Context

`Context` is passed to every action and event handler.

| Field | Description |
| --- | --- |
| `RequestId` | Request identifier for the current call. |
| `TraceParentId` | Current trace span ID. |
| `TraceParentRootId` | Root trace span ID for the call tree. |
| `ResponseId` | Response channel correlation ID. |
| `Params` | Input params. |
| `Meta` | Metadata propagated to nested calls. |
| `FromService` | Calling service name. |
| `FromAction` | Calling action name, when applicable. |
| `FromEvent` | Calling event name, when applicable. |
| `FromNode` | Calling node ID. |
| `CallingLevel` | Depth of the call chain. |
| `Call` | Helper for nested action/event calls. |
| `Broadcast` | Helper for sending an event to all subscribers. |
| `Service` | Owning service. |

## Nested calls

```go
result, err := ctx.Call("inventory.reserve", map[string]interface{}{"sku": "S-1"})
```

`ctx.Meta` is propagated to nested calls. Update it before calling another service if you need to pass metadata.

## Logging helpers

```go
ctx.LogInfo("action started")
ctx.LogWarning("unexpected input")
ctx.LogError("action failed")
```
