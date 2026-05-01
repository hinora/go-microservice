package goservice

import (
	"bytes"
	"encoding/json"
	"expvar"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// MetricsDatadog enables the built-in Datadog metrics exporter that
// periodically pushes broker counters to the Datadog v1/series API.
//
// Reference: https://moleculer.services/docs/0.14/metrics
//
// Datadog HTTP API reference:
// https://docs.datadoghq.com/api/latest/metrics/#submit-metrics
const MetricsDatadog string = "datadog"

// MetricsDatadogConfig configures the Datadog metrics exporter. When left
// empty, sensible defaults / DD_* environment variables are used.
type MetricsDatadogConfig struct {
	// APIKey is the Datadog API key. Falls back to DD_API_KEY.
	APIKey string
	// Site is the Datadog site (e.g. "datadoghq.com", "datadoghq.eu").
	// Falls back to DD_SITE then "datadoghq.com".
	Site string
	// Endpoint overrides the full series URL. Mostly useful for tests.
	Endpoint string
	// Hostname is the host tag reported with each series. Defaults to
	// os.Hostname().
	Hostname string
	// Tags are attached to every series.
	Tags []string
	// Namespace is prepended to metric names with a dot separator. When
	// empty no prefix is added.
	Namespace string
	// FlushInterval controls how often metrics are pushed. Defaults to
	// 10 seconds.
	FlushInterval time.Duration
	// HTTPClient is the HTTP client used for shipping. Defaults to a
	// client with a 5s timeout.
	HTTPClient *http.Client
}

// datadogMetricsExporter implements MetricsExporter and additionally runs a
// background flusher that pushes counters to Datadog. Export() returns the
// most recent serialized payload so the /metrics gateway endpoint remains
// usable for inspection/debugging.
type datadogMetricsExporter struct {
	config MetricsDatadogConfig

	mu       sync.RWMutex
	snapshot []byte
	stop     chan struct{}
}

func newDatadogMetricsExporter(cfg MetricsDatadogConfig) *datadogMetricsExporter {
	if cfg.APIKey == "" {
		cfg.APIKey = os.Getenv("DD_API_KEY")
	}
	if cfg.Site == "" {
		if env := os.Getenv("DD_SITE"); env != "" {
			cfg.Site = env
		} else {
			cfg.Site = "datadoghq.com"
		}
	}
	if cfg.Endpoint == "" {
		cfg.Endpoint = "https://api." + cfg.Site + "/api/v1/series"
	}
	if cfg.Hostname == "" {
		if h, err := os.Hostname(); err == nil {
			cfg.Hostname = h
		}
	}
	if cfg.FlushInterval <= 0 {
		cfg.FlushInterval = 10 * time.Second
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{Timeout: 5 * time.Second}
	}
	return &datadogMetricsExporter{config: cfg, stop: make(chan struct{})}
}

func (e *datadogMetricsExporter) ContentType() string {
	return "application/json"
}

// Export returns the most recent serialized Datadog payload. Returns an empty
// JSON object when no flush has occurred yet.
func (e *datadogMetricsExporter) Export() []byte {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if len(e.snapshot) == 0 {
		return []byte("{}")
	}
	out := make([]byte, len(e.snapshot))
	copy(out, e.snapshot)
	return out
}

// start runs the periodic flush loop and blocks; callers should run it in a
// goroutine.
func (e *datadogMetricsExporter) start() {
	ticker := time.NewTicker(e.config.FlushInterval)
	defer ticker.Stop()
	for {
		select {
		case <-e.stop:
			return
		case <-ticker.C:
			e.flush()
		}
	}
}

// ddSeries is the JSON shape accepted by the Datadog v1/series endpoint.
type ddSeries struct {
	Metric string      `json:"metric"`
	Points [][]float64 `json:"points"`
	Type   string      `json:"type"`
	Host   string      `json:"host,omitempty"`
	Tags   []string    `json:"tags,omitempty"`
}

type ddSeriesPayload struct {
	Series []ddSeries `json:"series"`
}

func (e *datadogMetricsExporter) collect() ddSeriesPayload {
	now := float64(time.Now().Unix())
	metricName := "goservice.action.calls.total"
	if e.config.Namespace != "" {
		metricName = e.config.Namespace + "." + metricName
	}
	payload := ddSeriesPayload{}
	expvar.Do(func(kv expvar.KeyValue) {
		if !strings.HasPrefix(kv.Key, MCountCall+".") {
			return
		}
		node, service, action, ok := splitActionMetricName(kv.Key)
		if !ok {
			return
		}
		count := metricsCounterValue(kv.Value.String())
		tags := append([]string(nil), e.config.Tags...)
		tags = append(tags,
			"node:"+node,
			"service:"+service,
			"action:"+action,
		)
		payload.Series = append(payload.Series, ddSeries{
			Metric: metricName,
			Points: [][]float64{{now, count}},
			Type:   "count",
			Host:   e.config.Hostname,
			Tags:   tags,
		})
	})
	return payload
}

func (e *datadogMetricsExporter) flush() {
	payload := e.collect()
	body, err := json.Marshal(payload)
	if err != nil {
		return
	}
	e.mu.Lock()
	e.snapshot = body
	e.mu.Unlock()

	if e.config.APIKey == "" || len(payload.Series) == 0 {
		return
	}
	req, err := http.NewRequest(http.MethodPost, e.config.Endpoint, bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("DD-API-KEY", e.config.APIKey)
	if resp, err := e.config.HTTPClient.Do(req); err == nil {
		resp.Body.Close()
	}
}
