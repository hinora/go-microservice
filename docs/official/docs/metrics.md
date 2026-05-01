# Metrics

Set broker metrics to a built-in exporter to enable metric collection.

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
| `goservice_action_calls_total` (Prometheus) | Total routed action calls. Labels: `node`, `service`, `action`. |
| `goservice.action.calls.total` (Datadog) | Total routed action calls. Tags: `node:<id>`, `service:<name>`, `action:<name>`. |

## Exporters

### Prometheus (`MetricsPrometheus`)

Pull-based. The exporter emits Prometheus text format with content type
`text/plain; version=0.0.4; charset=utf-8` and is scraped via `GET /metrics`.

### Datadog (`MetricsDatadog`)

Push-based. The exporter periodically pushes counters to the
[Datadog v1/series API](https://docs.datadoghq.com/api/latest/metrics/#submit-metrics).
The `/metrics` gateway endpoint still exposes the most recent serialized
payload as JSON for inspection.

```go
broker := goservice.Init(goservice.BrokerConfig{
    NodeId:  "node-1",
    Metrics: goservice.MetricsDatadog,
    MetricsDatadogConfig: goservice.MetricsDatadogConfig{
        APIKey:        os.Getenv("DD_API_KEY"), // or rely on DD_API_KEY env
        Site:          "datadoghq.com",         // or DD_SITE
        Tags:          []string{"env:prod"},
        Namespace:     "myapp",
        FlushInterval: 10 * time.Second,
    },
})
```

| Field | Description |
| --- | --- |
| `APIKey` | Datadog API key. Falls back to `DD_API_KEY`. |
| `Site` | Datadog site. Falls back to `DD_SITE` then `datadoghq.com`. |
| `Endpoint` | Override the full series URL (mainly for tests). |
| `Hostname` | `host` tag for every series. Defaults to `os.Hostname()`. |
| `Tags` | Extra tags appended to every series. |
| `Namespace` | Optional metric-name prefix (e.g. `myapp.goservice.action.calls.total`). |
| `FlushInterval` | Push cadence. Defaults to `10s`. |

When `APIKey` is empty the exporter still updates its in-memory snapshot but
does not contact Datadog — the same flag pattern as the logger exporter.
