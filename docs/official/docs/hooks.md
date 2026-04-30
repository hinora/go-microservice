# Hooks

Hooks run around action handlers. Hooks can be configured on a service or on an individual action.

```go
goservice.Action{
    Name: "create",
    Hooks: goservice.ActionHooks{
        Before: []goservice.BeforeHookFunc{
            func(ctx *goservice.Context) error { return nil },
        },
        After: []goservice.AfterHookFunc{
            func(ctx *goservice.Context, result interface{}) (interface{}, error) {
                return result, nil
            },
        },
        Error: []goservice.ErrorHookFunc{
            func(ctx *goservice.Context, err error) error { return err },
        },
    },
    Handle: create,
}
```

## Hook types

| Hook | Description |
| --- | --- |
| `Before` | Runs before the handler. A non-nil error aborts execution. |
| `After` | Runs after a successful handler and may transform the result. |
| `Error` | Runs when the handler returns an error and may recover by returning nil. |

## Ordering

Service-level hooks run before action-level hooks. Mixin hooks are merged before service hooks.
