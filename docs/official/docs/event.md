# Events

Events are fire-and-forget handlers. Multiple services can subscribe to the same event name.

```go
goservice.Event{
    Name: "order.created",
    Handle: func(ctx *goservice.Context) {
        ctx.LogInfo("order created")
    },
}
```

## Emitting an event

Calling an event name through `ctx.Call` sends the event using the normal group-balancing behavior:

```go
_, err := ctx.Call("order.created", map[string]interface{}{"id": "A-100"})
```

For events, the returned result is normally nil because event handlers do not reply.

## Broadcasting to all subscribers

Use `ctx.Broadcast` inside handlers or `broker.Broadcast` outside handlers to send to every matching subscriber on every known node:

```go
ctx.Broadcast("order.created", map[string]interface{}{"id": "A-100"})
```

## Pattern matching

Broadcast uses the event matching implemented by the broker registry. Prefer stable event names such as `domain.entity.action`, for example `order.created` or `user.deleted`.
