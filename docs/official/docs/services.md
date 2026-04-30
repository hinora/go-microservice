# Services

A service groups related actions and events under one name.

```go
broker.LoadService(&goservice.Service{
    Name: "orders",
    Actions: []goservice.Action{ /* ... */ },
    Events:  []goservice.Event{ /* ... */ },
})
```

## Service fields

| Field | Description |
| --- | --- |
| `Name` | Service name used in addresses such as `orders.create`. |
| `Version` | Optional version string. Version `2` is addressable as `v2.orders.create`. |
| `Mixins` | Service fragments merged before registration. Own actions override same-named mixin actions. |
| `Dependencies` | Service names that must be available before `Started` runs. |
| `Actions` | Request/response handlers. |
| `Events` | Fire-and-forget handlers. |
| `Started` | Lifecycle hook called after the service is loaded and dependencies are available. |
| `Stoped` | Lifecycle hook field reserved for shutdown use. |
| `Hooks` | Service-wide action hooks. |

## Versioned services

```go
broker.LoadService(&goservice.Service{
    Name:    "catalog",
    Version: "2",
    Actions: []goservice.Action{{Name: "find", Handle: findV2}},
})

result, err := broker.Call("api", "find catalog", "v2.catalog.find", params, goservice.CallOpts{})
```

When multiple versions exist, the plain `catalog.find` address resolves to the latest registered version.

## Mixins

Mixins let you share actions, events, and hooks between services.

```go
auditable := goservice.Service{
    Actions: []goservice.Action{{Name: "health", Handle: health}},
}

broker.LoadService(&goservice.Service{
    Name:   "orders",
    Mixins: []goservice.Service{auditable},
})
```

## Dependencies

`Started` waits until all dependency names appear in the registry, or logs a warning after the wait timeout.

```go
broker.LoadService(&goservice.Service{
    Name:         "payments",
    Dependencies: []string{"orders"},
    Started: func(ctx *goservice.Context) {
        ctx.LogInfo("payments started")
    },
})
```
