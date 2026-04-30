# Go Service Framework

[![Go Version](https://img.shields.io/badge/Go-1.24-blue.svg)](https://golang.org/doc/go1.24)

Go Service is a microservice framework written in Go, inspired by [Moleculer](https://moleculer.services/). It provides service discovery, inter-service communication, load balancing, distributed tracing, and an HTTP API gateway.

## Features

- **Service broker** – central hub that registers services and routes calls between them
- **Actions** – request/response RPC calls between services
- **Events** – fire-and-forget publish/subscribe messaging between services
- **Service discovery** – automatic node registration and deregistration via Redis
- **Transporter** – Redis-backed message bus for cross-node communication
- **Load balancing** – least-connections round-robin across service instances
- **API gateway** – auto-generated HTTP endpoints from service action definitions
- **Metrics** – optional Prometheus text exporter on the API gateway
- **Distributed tracing** – built-in span tracking with a console exporter
- **Structured logging** – configurable log levels with a console exporter

## Requirements

- Go 1.24+
- Redis (for multi-node discovery and transport)

## Installation

```bash
go get github.com/hinora/goservice
```

## Quick Start

### Single-node example

```go
package main

import "github.com/hinora/goservice"

func main() {
    b := goservice.Init(goservice.BrokerConfig{
        NodeId:         "node-1",
        RequestTimeOut: 5000, // ms
        LoggerConfig: goservice.Logconfig{
            Enable:   true,
            Type:     goservice.LogConsole,
            LogLevel: goservice.LogTypeInfo,
        },
    })

    b.LoadService(&goservice.Service{
        Name: "math",
        Actions: []goservice.Action{
            {
                Name: "add",
                Handle: func(ctx *goservice.Context) (interface{}, error) {
                    return map[string]interface{}{"result": 42}, nil
                },
            },
        },
    })

    b.Hold() // block forever
}
```

### Multi-node example (with Redis)

```go
b := goservice.Init(goservice.BrokerConfig{
    NodeId: "node-1",
    DiscoveryConfig: goservice.DiscoveryConfig{
        Enable:                   true,
        DiscoveryType:            goservice.DiscoveryTypeRedis,
        Config: goservice.DiscoveryRedisConfig{
            Host: "127.0.0.1",
            Port: 6379,
        },
        HeartbeatInterval:        3000, // ms
        HeartbeatTimeout:         7000, // ms
        CleanOfflineNodesTimeout: 9000, // ms
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

## Core Concepts

### Broker

`Broker` is the central component that manages service registration, message routing, load balancing, tracing, and logging.

```go
broker := goservice.Init(goservice.BrokerConfig{...})
broker.LoadService(&goservice.Service{...})
broker.Hold() // block until process exits
```

**`BrokerConfig` fields:**

| Field | Type | Description |
|---|---|---|
| `NodeId` | `string` | Unique identifier for this node |
| `TransporterConfig` | `TransporterConfig` | Message bus configuration |
| `DiscoveryConfig` | `DiscoveryConfig` | Service discovery configuration |
| `LoggerConfig` | `Logconfig` | Logging configuration |
| `Metrics` | `string` | Optional metrics exporter (`goservice.MetricsPrometheus`) |
| `TraceConfig` | `TraceConfig` | Distributed tracing configuration |
| `RequestTimeOut` | `int` | Request timeout in milliseconds |
| `Serializer` | `SerializerType` | Wire serializer for Redis transporter and discovery messages (`SerializerJSON` by default, or `SerializerMsgPack`) |

### Service

A `Service` groups related actions and events under a single name.

```go
&goservice.Service{
    Name: "greeter",
    Actions: []goservice.Action{ ... },
    Events:  []goservice.Event{ ... },
    Started: func(ctx *goservice.Context) { /* called after service loads */ },
    Stoped:  func(ctx *goservice.Context) { /* called on shutdown */ },
}
```

### Actions

Actions implement request/response RPC. They can be called from other services or exposed via the API gateway.

```go
goservice.Action{
    Name: "hello",
    Params: map[string]interface{}{}, // optional schema hint
    Rest: goservice.Rest{             // expose via HTTP gateway
        Method: goservice.GET,
        Path:   "/hello",
    },
    Handle: func(ctx *goservice.Context) (interface{}, error) {
        return map[string]interface{}{"msg": "Hello!"}, nil
    },
}
```

**Calling an action from within a handler:**

```go
result, err := ctx.Call("math.add", map[string]interface{}{"a": 1, "b": 2})
```

**Calling an action from outside a service (e.g. in tests):**

```go
result, err := broker.Call("caller-service", "trace-name", "math.add", params, goservice.CallOpts{})
```

### Events

Events implement fire-and-forget publish/subscribe. Multiple services can subscribe to the same event name.

```go
goservice.Event{
    Name: "order.created",
    Handle: func(ctx *goservice.Context) {
        // handle the event
    },
}
```

**Emitting an event from an action handler:**

```go
ctx.Call("order.created", payload)
```

### Context

`Context` is passed to every action and event handler.

| Field | Type | Description |
|---|---|---|
| `RequestId` | `string` | Unique ID for this request |
| `Params` | `interface{}` | Input parameters |
| `Meta` | `interface{}` | Metadata passed between calls |
| `FromService` | `string` | Name of the calling service |
| `FromNode` | `string` | Node ID of the caller |
| `CallingLevel` | `int` | Depth of the call chain |
| `Call` | `func` | Make a nested action/event call |
| `Service` | `*Service` | Pointer to the owning service |

**Logging from a handler:**

```go
ctx.LogInfo("action started")
ctx.LogWarning("something unexpected")
ctx.LogError("something failed")
```

## API Gateway

`InitGateway` creates a service that automatically generates HTTP routes based on registered service actions. It is backed by [Gin](https://github.com/gin-gonic/gin).

```go
gateway := goservice.InitGateway(goservice.GatewayConfig{
    Name: "api",
    Host: "0.0.0.0",
    Port: 8080,
    Routes: []goservice.GatewayConfigRoute{
        {
            Path: "/api",
            WhileList: []string{".*"}, // regex patterns of "service.action" to expose
            // optional: serve static files
            StaticPath:       "/",
            StaticFolderRoot: "./public",
        },
    },
})
broker.LoadService(gateway)
```

For an action with `Rest: goservice.Rest{Method: goservice.POST, Path: "/plus"}` on service `math`, the gateway generates `POST /api/mathplus`.

**Supported HTTP methods:** `GET`, `POST`, `PUT`, `DELETE`, `PATCH`, `HEAD`, `OPTIONS`

Parameters are merged from query string, URL path params, and the JSON request body.

## Discovery

When `DiscoveryConfig.Enable` is `true`, nodes automatically discover each other over Redis pub/sub.

```go
DiscoveryConfig: goservice.DiscoveryConfig{
    Enable:        true,
    DiscoveryType: goservice.DiscoveryTypeRedis,
    Config: goservice.DiscoveryRedisConfig{
        Host:     "127.0.0.1",
        Port:     6379,
        Password: "",
        Db:       0,
    },
    HeartbeatInterval:        3000, // how often to send heartbeat (ms)
    HeartbeatTimeout:         7000, // mark node offline if no heartbeat for this long (ms)
    CleanOfflineNodesTimeout: 9000, // remove offline node from registry after this long (ms)
},
```

For a detailed sequence diagram of the discovery protocol, see [docs/discovery.md](docs/discovery.md).

## Transporter

The transporter routes action calls and event emissions to remote nodes.

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

## Wire Serializer

JSON is the default wire format for Redis transporter and discovery messages. Set `BrokerConfig.Serializer` to `goservice.SerializerMsgPack` to use MessagePack across the cluster.

```go
b := goservice.Init(goservice.BrokerConfig{
    NodeId:     "node-1",
    Serializer: goservice.SerializerMsgPack,
    // DiscoveryConfig and TransporterConfig ...
})
```

All nodes that communicate through Redis must use the same serializer.

## Distributed Tracing

Tracing records the call tree for each request. The console exporter prints trace spans to stdout.

```go
TraceConfig: goservice.TraceConfig{
    Enabled:      true,
    TraceExpoter: goservice.TraceExporterConsole,
},
```

## Metrics

Set `BrokerConfig.Metrics` to `goservice.MetricsPrometheus` to enable the built-in Prometheus text exporter. When an API gateway service is loaded, it exposes action call counters at `GET /metrics`.

```go
b := goservice.Init(goservice.BrokerConfig{
    NodeId:  "node-1",
    Metrics: goservice.MetricsPrometheus,
})
```

## Logging

```go
LoggerConfig: goservice.Logconfig{
    Enable:   true,
    Type:     goservice.LogConsole,
    LogLevel: goservice.LogTypeInfo, // LogTypeInfo | LogTypeWarning | LogTypeError
},
```

## Load Balancing

When multiple nodes register the same service action, the broker uses least-connections round-robin to distribute calls across instances automatically.

## Examples

See the [`example/`](example/) directory for three-node examples that demonstrate cross-node action calls, events, and the gateway.

```
example/
├── node-1/   # math service + API gateway
├── node-2/   # additional services
└── node-3/   # additional services
```

To run the examples, start a local Redis instance and then run each node:

```bash
cd example/node-1 && go run main.go
cd example/node-2 && go run main.go
cd example/node-3 && go run main.go
```

## License

See [LICENSE](LICENSE).
