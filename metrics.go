package goservice

import (
	"bytes"
	"encoding/json"
	"expvar"
	"sort"
	"strconv"
	"strings"

	"github.com/zserge/metric"
)

type CountSample struct {
	Type  string  `json:"type"`
	Count float64 `json:"count"`
}
type Count struct {
	Interval int           `json:"interval"`
	Samples  []CountSample `json:"samples"`
}

func MetricsGetValueCounter(m metric.Metric) float64 {
	var data Count
	json.Unmarshal([]byte(m.String()), &data)
	total := 0
	for i := 0; i < len(data.Samples); i++ {
		total += int(data.Samples[i].Count)
	}
	return float64(total)
}

func (b *Broker) initMestricCountCallAction() {
	for _, s := range b.registryServices {
		for _, a := range s.Actions {
			nameCheck := MCountCall + "." + s.Node.NodeId + "." + s.Name + "." + a.Name
			if expvar.Get(nameCheck) == nil {
				expvar.Publish(nameCheck, metric.NewCounter(MCountCallTime))
			}
		}
	}
}

const (
	MCountCall string = "count_call"
)
const (
	MCountCallTime string = "1h1h"
)

const (
	// MetricsPrometheus enables the built-in Prometheus text exporter.
	MetricsPrometheus string = "prometheus"
)

// MetricsExporter exposes broker metrics in a concrete wire format.
type MetricsExporter interface {
	ContentType() string
	Export() []byte
}

type prometheusMetricsExporter struct{}

func (prometheusMetricsExporter) ContentType() string {
	return "text/plain; version=0.0.4; charset=utf-8"
}

func (prometheusMetricsExporter) Export() []byte {
	var lines []string
	expvar.Do(func(kv expvar.KeyValue) {
		if !strings.HasPrefix(kv.Key, MCountCall+".") {
			return
		}
		node, service, action, ok := splitActionMetricName(kv.Key)
		if !ok {
			return
		}
		count := metricsCounterValue(kv.Value.String())
		lines = append(lines, prometheusMetricLine("goservice_action_calls_total", map[string]string{
			"node":    node,
			"service": service,
			"action":  action,
		}, count))
	})
	sort.Strings(lines)

	var out bytes.Buffer
	out.WriteString("# HELP goservice_action_calls_total Total routed action calls.\n")
	out.WriteString("# TYPE goservice_action_calls_total counter\n")
	for _, line := range lines {
		out.WriteString(line)
		out.WriteByte('\n')
	}
	return out.Bytes()
}

func (b *Broker) initMetrics() {
	switch strings.ToLower(b.Config.Metrics) {
	case MetricsPrometheus:
		b.metricsExporter = prometheusMetricsExporter{}
	default:
		b.metricsExporter = nil
	}
}

func splitActionMetricName(name string) (node string, service string, action string, ok bool) {
	parts := strings.Split(strings.TrimPrefix(name, MCountCall+"."), ".")
	if len(parts) < 3 {
		return "", "", "", false
	}
	return parts[0], strings.Join(parts[1:len(parts)-1], "."), parts[len(parts)-1], true
}

func metricsCounterValue(raw string) float64 {
	var data Count
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		return 0
	}
	total := 0.0
	for _, sample := range data.Samples {
		total += sample.Count
	}
	return total
}

func prometheusMetricLine(name string, labels map[string]string, value float64) string {
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	labelParts := make([]string, 0, len(labels))
	for _, k := range keys {
		labelParts = append(labelParts, k+`=`+strconv.Quote(labels[k]))
	}
	return name + "{" + strings.Join(labelParts, ",") + "} " + strconv.FormatFloat(value, 'f', -1, 64)
}
