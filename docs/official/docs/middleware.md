# Middleware

Middleware wraps local action execution. Register middlewares on `BrokerConfig.Middlewares`.

```go
broker := goservice.Init(goservice.BrokerConfig{
    NodeId: "node-1",
    Middlewares: []goservice.Middleware{
        {
            LocalAction: func(ctx *goservice.Context, action goservice.Action, next func(*goservice.Context) (interface{}, error)) (interface{}, error) {
                ctx.LogInfo("before " + action.Name)
                result, err := next(ctx)
                ctx.LogInfo("after " + action.Name)
                return result, err
            },
        },
    },
})
```

## Execution order

- Parameter validation runs before middleware.
- The first registered middleware is the outermost wrapper.
- Middleware wraps the core handler, which runs before hooks, the action handler, and after/error hooks.
- Returning an error aborts the action and propagates that error to the caller.

Only `LocalAction` middleware is currently supported.
