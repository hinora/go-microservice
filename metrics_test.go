package goservice

import (
	"expvar"
	"strings"
	"testing"
	"time"

	"github.com/zserge/metric"
)

func TestInitMetricsPrometheus(t *testing.T) {
	b := Broker{Config: BrokerConfig{Metrics: MetricsPrometheus}}
	b.initMetrics()
	if b.metricsExporter == nil {
		t.Fatal("expected Prometheus metrics exporter")
	}
	if ct := b.metricsExporter.ContentType(); !strings.Contains(ct, "text/plain") {
		t.Fatalf("unexpected content type: %s", ct)
	}
}

func TestInitMetricsDisabled(t *testing.T) {
	b := Broker{}
	b.initMetrics()
	if b.metricsExporter != nil {
		t.Fatal("expected metrics exporter to be disabled")
	}
}

func TestPrometheusMetricsExporterActionCalls(t *testing.T) {
	node := "node-" + strings.ReplaceAll(time.Now().Format("150405.000000000"), ".", "")
	name := MCountCall + "." + node + ".v2.math.add"
	counter := metric.NewCounter(MCountCallTime)
	counter.Add(3)
	expvar.Publish(name, counter)

	output := string((prometheusMetricsExporter{}).Export())
	if !strings.Contains(output, "# TYPE goservice_action_calls_total counter") {
		t.Fatalf("missing metric type: %s", output)
	}
	if !strings.Contains(output, `action="add"`) ||
		!strings.Contains(output, `node="`+node+`"`) ||
		!strings.Contains(output, `service="v2.math"`) ||
		!strings.Contains(output, "} 3") {
		t.Fatalf("missing action call metric labels/value: %s", output)
	}
}
