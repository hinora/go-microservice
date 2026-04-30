# Serialization

The broker serializer controls Redis discovery and transporter payloads.

## JSON

JSON is the default serializer when `BrokerConfig.Serializer` is left unset.

```go
broker := goservice.Init(goservice.BrokerConfig{
    NodeId: "node-1",
})
```

## MessagePack

Use MessagePack for compact binary payloads:

```go
broker := goservice.Init(goservice.BrokerConfig{
    NodeId:     "node-1",
    Serializer: goservice.SerializerMsgPack,
})
```

All nodes sharing Redis discovery or transport channels must use the same serializer.
