# Metrics

Set broker metrics to `goservice.MetricsPrometheus` to enable Prometheus text export.

```go
broker := goservice.Init(goservice.BrokerConfig{
    NodeId:  "node-1",
    Metrics: goservice.MetricsPrometheus,
})
```

When an API gateway is loaded, metrics are available at:

```text
GET /metrics
```

## Exported metrics

| Metric | Description |
| --- | --- |
| `goservice_action_calls_total` | Total routed action calls. Labels: `node`, `service`, `action`. |

The exporter emits Prometheus text format with content type `text/plain; version=0.0.4; charset=utf-8`.
