# Go Microservice Documentation

Go Microservice is a Go framework for building service-oriented systems inspired by Moleculer. It provides a broker, services, request/response actions, fire-and-forget events, Redis-backed discovery and transport, an HTTP API gateway, tracing, metrics, logging, and resilience features.

## Quick links

- [Introduction](introduction.md) for requirements and installation
- [Core Concepts](core-concepts.md) for the mental model
- [Configuration](configuration.md) for all broker options
- [Services](services.md), [Actions](actions.md), and [Events](event.md) for application APIs
- [Discovery Registry](discovery-registry.md) and [Transporters](transporters.md) for multi-node deployments
- [API Gateway](api-gateway.md) for HTTP routing
- [Resilience](resilience.md) for retries, circuit breakers, bulkheads, timeouts, and caching

## Minimal service

```go
package main

import goservice "github.com/hinora/go-microservice"

func main() {
    broker := goservice.Init(goservice.BrokerConfig{
        NodeId:         "node-1",
        RequestTimeOut: 5000,
    })

    broker.LoadService(&goservice.Service{
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

    broker.Hold()
}
```
