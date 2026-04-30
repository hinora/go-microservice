# Transporters

The transporter routes action calls, responses, and events between nodes. Redis is the currently supported transporter.

## Redis transporter

```go
TransporterConfig: goservice.TransporterConfig{
    Enable:          true,
    TransporterType: goservice.TransporterTypeRedis,
    Config: goservice.TransporterRedisConfig{
        Host:     "127.0.0.1",
        Port:     6379,
        Password: "",
        Db:       0,
    },
},
```

## Local-only mode

If `TransporterConfig.Enable` is false, the broker still supports local services in the same process. Remote calls require the transporter and discovery to be enabled.

## Serialization

Transporter payloads use the broker serializer. JSON is the default; MessagePack is available with `SerializerMsgPack`. Every node in the same Redis cluster must use the same serializer.
