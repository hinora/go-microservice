# Broker

`Broker` is the runtime container for a node.

```go
broker := goservice.Init(goservice.BrokerConfig{
    NodeId:         "node-1",
    RequestTimeOut: 5000,
})

broker.LoadService(&goservice.Service{Name: "math"})
broker.Hold()
```

## Loading services

`LoadService` registers the service, applies mixins, adds actions and events to the local registry, starts action/event listeners, emits service info, and runs the `Started` lifecycle hook when dependencies are ready.

```go
broker.LoadService(&goservice.Service{
    Name: "greeter",
    Actions: []goservice.Action{ /* ... */ },
    Events:  []goservice.Event{ /* ... */ },
})
```

## Calling actions externally

Use `broker.Call` from tests, startup logic, or integration code outside a handler:

```go
result, err := broker.Call(
    "caller-service",
    "manual math.add call",
    "math.add",
    map[string]interface{}{"a": 1, "b": 2},
    goservice.CallOpts{},
)
```

## Broadcasting events externally

`Broadcast` sends an event to every matching subscriber on every known node:

```go
broker.Broadcast("order.created", map[string]interface{}{"id": "A-100"})
```

## Holding the process

`Hold` blocks forever and is useful for examples or long-running worker processes:

```go
broker.Hold()
```
