# Configuration

`goservice.Init` creates a broker from `goservice.BrokerConfig`.

```go
broker := goservice.Init(goservice.BrokerConfig{
    NodeId:         "node-1",
    RequestTimeOut: 5000,
})
```

## BrokerConfig

| Field | Type | Description |
| --- | --- | --- |
| `NodeId` | `string` | Unique node identifier. |
| `TransporterConfig` | `TransporterConfig` | Configures cross-node message transport. |
| `LoggerConfig` | `Logconfig` | Configures logging. |
| `Metrics` | `string` | Set to `goservice.MetricsPrometheus` to enable Prometheus metrics. |
| `TraceConfig` | `TraceConfig` | Configures distributed tracing. |
| `DiscoveryConfig` | `DiscoveryConfig` | Configures node discovery and registry sharing. |
| `RequestTimeOut` | `int` | Default action call timeout in milliseconds. |
| `Serializer` | `SerializerType` | Wire serializer. Defaults to JSON; use `SerializerMsgPack` for MessagePack. |
| `Retry` | `RetryPolicy` | Broker-level default retry policy. |
| `CircuitBreaker` | `CircuitBreakerConfig` | Broker-level circuit breaker settings. |
| `Middlewares` | `[]Middleware` | Ordered local action middleware chain. |

## Multi-node configuration

Use matching Redis discovery and transporter settings on every node:

```go
broker := goservice.Init(goservice.BrokerConfig{
    NodeId: "node-1",
    DiscoveryConfig: goservice.DiscoveryConfig{
        Enable:        true,
        DiscoveryType: goservice.DiscoveryTypeRedis,
        Config: goservice.DiscoveryRedisConfig{
            Host: "127.0.0.1",
            Port: 6379,
        },
        HeartbeatInterval:        3000,
        HeartbeatTimeout:         7000,
        CleanOfflineNodesTimeout: 9000,
    },
    TransporterConfig: goservice.TransporterConfig{
        Enable:          true,
        TransporterType: goservice.TransporterTypeRedis,
        Config: goservice.TransporterRedisConfig{
            Host: "127.0.0.1",
            Port: 6379,
        },
    },
    RequestTimeOut: 5000,
})
```

All nodes that exchange Redis payloads must use the same serializer.
