# Introduction

Go Microservice is a microservice framework written in Go. It is designed around small services loaded into a broker, with actions for RPC-style calls and events for pub/sub-style communication.

## Features

- Service broker for registering services and routing calls
- Actions for request/response RPC
- Events for fire-and-forget messaging
- Redis-backed service discovery and inter-node transport
- Least-connections round-robin load balancing across service instances
- HTTP API gateway backed by Gin
- Prometheus text metrics exposed by the gateway
- Console distributed tracing
- Structured console logging
- JSON and MessagePack wire serialization
- Action validation, hooks, middleware, caching, retries, circuit breakers, bulkheads, and timeouts
- Service versions, mixins, and dependency-aware startup hooks

## Requirements

- Go 1.24 or newer
- Redis for multi-node discovery and transport

## Installation

```bash
go get github.com/hinora/go-microservice
```

Import the module with an alias if you want to use the shorter `goservice` package name:

```go
import goservice "github.com/hinora/go-microservice"
```

## Running examples

The repository includes three example nodes under `example/`. Start Redis locally, then run each node in a separate terminal:

```bash
cd example/node-1 && go run main.go
cd example/node-2 && go run main.go
cd example/node-3 && go run main.go
```
