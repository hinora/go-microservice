# Actions

Actions implement request/response behavior and are addressed as `service.action`.

```go
goservice.Action{
    Name: "add",
    Handle: func(ctx *goservice.Context) (interface{}, error) {
        params := ctx.Params.(map[string]interface{})
        return map[string]interface{}{"result": params["a"]}, nil
    },
}
```

## Action fields

| Field | Description |
| --- | --- |
| `Name` | Action name under the service. |
| `Params` | Informational parameter hint stored in the registry. |
| `Schema` | Optional validation rules enforced before the handler runs. |
| `Rest` | Optional HTTP gateway mapping. |
| `Timeout` | Action-level timeout in milliseconds. Overrides broker timeout. |
| `Cache` | Optional in-memory result cache. |
| `Bulkhead` | Optional concurrency and queue limits for this action. |
| `Hooks` | Action-specific before/after/error hooks. |
| `Handle` | Function invoked with `*goservice.Context`. |

## Calling another action

Inside an action or event handler, use `ctx.Call`:

```go
result, err := ctx.Call("math.add", map[string]interface{}{"a": 1, "b": 2})
```

Pass call options to override retry or timeout for a single call:

```go
result, err := ctx.Call("math.add", params, goservice.CallOpts{
    Timeout: 1000,
    Retry: &goservice.RetryPolicy{
        MaxRetries: 2,
        RetryDelay: 100,
        Factor:     2,
    },
})
```

## Exposing actions over HTTP

Set `Rest` and load an API gateway service:

```go
goservice.Action{
    Name: "plus",
    Rest: goservice.Rest{Method: goservice.POST, Path: "/plus"},
    Handle: plus,
}
```

See [API Gateway](api-gateway.md) for generated route behavior.
