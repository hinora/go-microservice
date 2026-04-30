# API Gateway

`InitGateway` creates a service that maps registered action REST metadata to HTTP routes. The gateway uses Gin internally.

```go
gateway := goservice.InitGateway(goservice.GatewayConfig{
    Name: "api",
    Host: "127.0.0.1",
    Port: 8000,
    Routes: []goservice.GatewayConfigRoute{
        {
            Path:      "/api",
            WhileList: []string{".*"},
        },
    },
})

broker.LoadService(gateway)
```

## GatewayConfig

| Field | Description |
| --- | --- |
| `Name` | Service name for the gateway service. |
| `Host` | Listen host. |
| `Port` | Listen port. |
| `Routes` | Route groups to generate. |

## GatewayConfigRoute

| Field | Description |
| --- | --- |
| `Path` | Base path for the route group, such as `/api`. |
| `WhileList` | Regular expression patterns matching `service.action` names to expose. |
| `StaticPath` | Optional static URL prefix. |
| `StaticFolderRoot` | Optional local folder for static files. |

## Action mapping

An action with REST metadata is exposed when it matches a route whitelist:

```go
goservice.Action{
    Name: "plus",
    Rest: goservice.Rest{Method: goservice.POST, Path: "/plus"},
    Handle: plus,
}
```

For service `math` and route group `/api`, this generates:

```text
POST /api/math/plus
```

Supported methods are `GET`, `POST`, `PUT`, `DELETE`, `PATCH`, `HEAD`, and `OPTIONS`.

## Parameters

Gateway parameters are merged from:

1. query string values
2. path params
3. JSON request body

Later sources override earlier sources when keys collide.

## Static files

```go
GatewayConfigRoute{
    Path:             "/api",
    WhileList:        []string{".*"},
    StaticPath:       "/",
    StaticFolderRoot: "./public",
}
```

## Metrics endpoint

When broker metrics are set to `goservice.MetricsPrometheus`, the gateway exposes `GET /metrics`.
