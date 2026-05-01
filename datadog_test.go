package goservice

import (
	"encoding/json"
	"errors"
	"expvar"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/zserge/metric"
)

// --- Logger Datadog --------------------------------------------------------

func TestLoggerDatadogFormatEntryLevels(t *testing.T) {
	l := &LoggerDatadog{Service: "svc", Source: "src", Hostname: "h", Tags: "env:test"}
	l.applyDefaults()

	cases := []struct {
		in     LogType
		level  string
		status string
	}{
		{LogTypeInfo, "info", "info"},
		{LogTypeWarning, "warning", "warning"},
		{LogTypeError, "error", "error"},
	}
	for _, c := range cases {
		entry := l.formatEntry(LogData{Type: c.in, Message: "m", Time: int(time.Now().Unix())})
		if entry["level"] != c.level || entry["status"] != c.status {
			t.Fatalf("level=%v status=%v want %s/%s", entry["level"], entry["status"], c.level, c.status)
		}
		if entry["service"] != "svc" || entry["ddsource"] != "src" || entry["hostname"] != "h" || entry["ddtags"] != "env:test" {
			t.Fatalf("unexpected entry: %v", entry)
		}
	}
}

func TestLoggerDatadogFlushPostsBatch(t *testing.T) {
	var got atomic.Value
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("DD-API-KEY") != "secret" {
			t.Errorf("missing api key, got headers: %v", r.Header)
		}
		body, _ := io.ReadAll(r.Body)
		got.Store(body)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	l := &LoggerDatadog{
		APIKey:         "secret",
		Endpoint:       srv.URL,
		Service:        "svc",
		FlushBatchSize: 10,
		FlushInterval:  10 * time.Millisecond,
	}
	l.applyDefaults()
	l.WriteLog(LogData{Type: LogTypeInfo, Message: "hello", Time: int(time.Now().Unix())})
	l.WriteLog(LogData{Type: LogTypeError, Message: "boom", Time: int(time.Now().Unix())})
	if n := l.flush(); n != 2 {
		t.Fatalf("expected 2 flushed, got %d", n)
	}

	raw, _ := got.Load().([]byte)
	var entries []map[string]interface{}
	if err := json.Unmarshal(raw, &entries); err != nil {
		t.Fatalf("invalid json payload: %v: %s", err, raw)
	}
	if len(entries) != 2 || entries[0]["message"] != "hello" || entries[1]["status"] != "error" {
		t.Fatalf("unexpected payload: %v", entries)
	}
}

func TestLoggerDatadogSkipsWhenNoAPIKey(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	defer srv.Close()

	l := &LoggerDatadog{Endpoint: srv.URL}
	l.applyDefaults()
	l.APIKey = ""
	l.WriteLog(LogData{Type: LogTypeInfo, Message: "x", Time: int(time.Now().Unix())})
	if n := l.flush(); n != 1 {
		t.Fatalf("expected entry to be drained, got %d", n)
	}
	if called {
		t.Fatal("HTTP server should not have been called without API key")
	}
}

func TestInitLogDatadog(t *testing.T) {
	dd := &LoggerDatadog{APIKey: "k", Endpoint: "http://127.0.0.1:0"}
	b := Broker{Config: BrokerConfig{LoggerConfig: Logconfig{
		Enable:   true,
		Type:     LogDatadog,
		LogLevel: LogTypeInfo,
		Datadog:  dd,
	}}}
	b.initLog()
	if _, ok := b.logs.Extenal.(*LoggerDatadog); !ok {
		t.Fatalf("expected LoggerDatadog exporter, got %T", b.logs.Extenal)
	}
}

// --- Trace Datadog ---------------------------------------------------------

func TestTraceDatadogStringToIDStable(t *testing.T) {
	if stringToID("abc") != stringToID("abc") {
		t.Fatal("hashing not deterministic")
	}
	if stringToID("") != 0 {
		t.Fatal("empty string should hash to 0")
	}
	// Top bit cleared.
	if stringToID("abc")&(1<<63) != 0 {
		t.Fatal("expected 63-bit positive ID")
	}
}

func TestTraceDatadogExportSpanPostsToAgent(t *testing.T) {
	type capture struct {
		method string
		body   []byte
	}
	ch := make(chan capture, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		ch <- capture{method: r.Method, body: body}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	b := &Broker{
		Config: BrokerConfig{
			NodeId: "node-x",
			TraceConfig: TraceConfig{
				Enabled:      true,
				TraceExpoter: TraceExporterDDDog,
				TraceExporterConfig: TraceDatadogConfig{
					AgentURL: srv.URL,
					Service:  "svc-x",
					Env:      "test",
					Version:  "1.0",
				},
			},
		},
		traceSpans: map[string]*traceSpan{},
	}
	b.initTrace()
	exporter, ok := b.trace.Exporter.(*traceDatadog)
	if !ok {
		t.Fatalf("expected *traceDatadog exporter, got %T", b.trace.Exporter)
	}

	parent := &traceSpan{
		Name: "parent", Type: "action", TraceId: "T1", ParentId: "",
		Service: "math", StartTime: 1000, FinishTime: 1100, Duration: 100,
		Tags: tags{Action: "math.add", NodeId: "node-x", RequestId: "T1", CallerNodeId: "node-x"},
	}
	child := &traceSpan{
		Name: "child", Type: "action", TraceId: "T2", ParentId: "T1",
		Service: "math", StartTime: 1010, FinishTime: 1080, Duration: 70,
		Error: errors.New("boom"),
		Tags:  tags{Action: "math.mul", NodeId: "node-x", RequestId: "T1", RemoteCall: true},
	}
	exporter.ExportSpan([]*traceSpan{parent, child})

	select {
	case got := <-ch:
		if got.method != http.MethodPut {
			t.Fatalf("expected PUT, got %s", got.method)
		}
		var traces [][]ddSpan
		if err := json.Unmarshal(got.body, &traces); err != nil {
			t.Fatalf("invalid payload: %v: %s", err, got.body)
		}
		if len(traces) != 1 || len(traces[0]) != 2 {
			t.Fatalf("expected 1 trace with 2 spans: %s", got.body)
		}
		if traces[0][0].TraceID != traces[0][1].TraceID {
			t.Fatal("spans within the same request should share trace_id")
		}
		if traces[0][0].Service != "svc-x" || traces[0][0].Resource != "math.add" {
			t.Fatalf("unexpected first span: %+v", traces[0][0])
		}
		if traces[0][1].Error != 1 || traces[0][1].Meta["error.message"] != "boom" {
			t.Fatalf("expected error metadata on second span: %+v", traces[0][1])
		}
		if traces[0][0].Meta["env"] != "test" || traces[0][0].Meta["version"] != "1.0" {
			t.Fatalf("env/version meta missing: %+v", traces[0][0].Meta)
		}
	case <-time.After(time.Second):
		t.Fatal("agent did not receive trace payload")
	}
}

func TestInitTraceDatadogDisabled(t *testing.T) {
	b := &Broker{Config: BrokerConfig{TraceConfig: TraceConfig{Enabled: false, TraceExpoter: TraceExporterDDDog}}}
	b.initTrace()
	if b.trace.Exporter != nil {
		t.Fatal("expected no exporter when tracing disabled")
	}
}

// --- Metrics Datadog -------------------------------------------------------

func TestInitMetricsDatadog(t *testing.T) {
	b := Broker{Config: BrokerConfig{
		Metrics: MetricsDatadog,
		MetricsDatadogConfig: MetricsDatadogConfig{
			Endpoint:      "http://127.0.0.1:0",
			FlushInterval: time.Hour, // never flush during the test
		},
	}}
	b.initMetrics()
	if b.metricsExporter == nil {
		t.Fatal("expected datadog metrics exporter")
	}
	if ct := b.metricsExporter.ContentType(); ct != "application/json" {
		t.Fatalf("unexpected content type: %s", ct)
	}
	if ex, ok := b.metricsExporter.(*datadogMetricsExporter); ok {
		close(ex.stop)
	}
}

func TestDatadogMetricsExporterCollectAndFlush(t *testing.T) {
	node := "test-node-datadog"
	metricName := MCountCall + "." + node + ".v2.math.add"
	counter := metric.NewCounter(MCountCallInterval)
	counter.Add(5)
	expvar.Publish(metricName, counter)

	type captured struct {
		apiKey string
		body   []byte
	}
	ch := make(chan captured, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		ch <- captured{apiKey: r.Header.Get("DD-API-KEY"), body: body}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	exp := newDatadogMetricsExporter(MetricsDatadogConfig{
		APIKey:    "secret",
		Endpoint:  srv.URL,
		Hostname:  "host-1",
		Tags:      []string{"env:test"},
		Namespace: "demo",
	})
	exp.flush()

	select {
	case got := <-ch:
		if got.apiKey != "secret" {
			t.Fatalf("missing api key: %q", got.apiKey)
		}
		var payload ddSeriesPayload
		if err := json.Unmarshal(got.body, &payload); err != nil {
			t.Fatalf("invalid payload: %v: %s", err, got.body)
		}
		found := false
		for _, s := range payload.Series {
			if s.Metric != "demo.goservice.action.calls.total" {
				continue
			}
			if !containsTag(s.Tags, "node:"+node) {
				continue
			}
			if s.Type != "count" || s.Host != "host-1" {
				t.Fatalf("unexpected series: %+v", s)
			}
			if !containsTag(s.Tags, "service:v2.math") || !containsTag(s.Tags, "action:add") || !containsTag(s.Tags, "env:test") {
				t.Fatalf("missing required tags: %v", s.Tags)
			}
			if len(s.Points) != 1 || s.Points[0][1] != 5 {
				t.Fatalf("expected count=5 point, got %v", s.Points)
			}
			found = true
		}
		if !found {
			t.Fatalf("expected metric series for %s, got: %s", node, got.body)
		}
	case <-time.After(time.Second):
		t.Fatal("datadog metrics endpoint never received payload")
	}

	// Export() should reflect the most recent flushed payload.
	snap := exp.Export()
	if !strings.Contains(string(snap), "demo.goservice.action.calls.total") {
		t.Fatalf("Export() snapshot missing metric: %s", snap)
	}
}

func TestDatadogMetricsExporterSkipsWithoutAPIKey(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	defer srv.Close()
	exp := newDatadogMetricsExporter(MetricsDatadogConfig{Endpoint: srv.URL})
	exp.config.APIKey = ""
	exp.flush()
	if called {
		t.Fatal("HTTP intake should not be called without API key")
	}
}

func containsTag(tags []string, want string) bool {
	for _, t := range tags {
		if t == want {
			return true
		}
	}
	return false
}
