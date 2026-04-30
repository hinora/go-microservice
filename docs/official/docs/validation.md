# Validation

Set `Action.Schema` to validate incoming params before middleware, hooks, or the handler run.

```go
min := 1.0
max := 100.0

goservice.Action{
    Name: "create",
    Schema: map[string]goservice.ParamRule{
        "name": {Type: "string", Required: true, Min: &min},
        "age":  {Type: "number", Max: &max},
    },
    Handle: create,
}
```

## ParamRule

| Field | Description |
| --- | --- |
| `Type` | Expected type: `string`, `number`, `bool`, `array`, `object`, or `any`. Empty type accepts all values. |
| `Required` | Requires the field to exist and be non-nil. |
| `Min` | Minimum number value or minimum string length. |
| `Max` | Maximum number value or maximum string length. |

Validation errors are returned as `Validation error: field: message` and include all failing fields.
