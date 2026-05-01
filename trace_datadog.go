package goservice

import (
	"bytes"
	"encoding/json"
	"hash/fnv"
	"net/http"
	"os"
	"time"
)

// TraceDatadogConfig configures the Datadog APM trace exporter. It can be
// supplied via TraceConfig.TraceExporterConfig when TraceConfig.TraceExpoter
// is TraceExporterDDDog.
//
// Reference: https://moleculer.services/docs/0.14/tracing
//
// Datadog APM trace ingestion is performed by the local agent at
// http://localhost:8126/v0.4/traces (JSON payload).
type TraceDatadogConfig struct {
	// AgentURL is the trace endpoint URL. Defaults to
	// "http://localhost:8126/v0.4/traces". Can be overridden by the
	// DD_TRACE_AGENT_URL environment variable.
	AgentURL string
	// Service is the default Datadog service name. When empty the broker
	// node id is used.
	Service string
	// Env is the Datadog environment (e.g. "prod", "staging").
	Env string
	// Version is the Datadog service version tag.
	Version string
	// HTTPClient is the HTTP client used to ship traces. Defaults to a
	// client with a 5s timeout.
	HTTPClient *http.Client
}

// traceDatadog implements an exporter that converts framework spans into
// Datadog APM JSON spans and POSTs them to the agent.
type traceDatadog struct {
	Broker *Broker
	Config TraceDatadogConfig
}

func initTraceDatadog(broker *Broker) *traceDatadog {
	cfg := TraceDatadogConfig{}
	if v, ok := broker.Config.TraceConfig.TraceExporterConfig.(TraceDatadogConfig); ok {
		cfg = v
	} else if v, ok := broker.Config.TraceConfig.TraceExporterConfig.(*TraceDatadogConfig); ok && v != nil {
		cfg = *v
	}
	if cfg.AgentURL == "" {
		if env := os.Getenv("DD_TRACE_AGENT_URL"); env != "" {
			cfg.AgentURL = env
		} else {
			cfg.AgentURL = "http://localhost:8126/v0.4/traces"
		}
	}
	if cfg.Service == "" {
		cfg.Service = broker.Config.NodeId
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{Timeout: 5 * time.Second}
	}
	return &traceDatadog{Broker: broker, Config: cfg}
}

// ddSpan is the JSON representation of a Datadog APM span for the v0.4 API.
type ddSpan struct {
	TraceID  uint64            `json:"trace_id"`
	SpanID   uint64            `json:"span_id"`
	ParentID uint64            `json:"parent_id,omitempty"`
	Name     string            `json:"name"`
	Resource string            `json:"resource"`
	Service  string            `json:"service"`
	Type     string            `json:"type,omitempty"`
	Start    int64             `json:"start"`
	Duration int64             `json:"duration"`
	Error    int32             `json:"error"`
	Meta     map[string]string `json:"meta,omitempty"`
}

// ExportSpan converts the supplied spans into a Datadog trace payload and
// ships it to the configured agent.
func (e *traceDatadog) ExportSpan(spans []*traceSpan) {
	if len(spans) == 0 {
		return
	}
	ddSpans := make([]ddSpan, 0, len(spans))
	traceIDByReq := map[string]uint64{}
	for _, s := range spans {
		tid, ok := traceIDByReq[s.Tags.RequestId]
		if !ok {
			tid = stringToID(s.Tags.RequestId)
			if tid == 0 {
				tid = stringToID(s.TraceId)
			}
			traceIDByReq[s.Tags.RequestId] = tid
		}
		converted := ddSpan{
			TraceID:  tid,
			SpanID:   stringToID(s.TraceId),
			ParentID: stringToID(s.ParentId),
			Name:     s.Name,
			Resource: s.Tags.Action,
			Service:  e.Config.Service,
			Type:     s.Type,
			Start:    s.StartTime,
			Duration: s.Duration,
		}
		meta := map[string]string{
			"service.name":  s.Service,
			"action.name":   s.Tags.Action,
			"node.id":       s.Tags.NodeId,
			"caller.node":   s.Tags.CallerNodeId,
			"request.id":    s.Tags.RequestId,
			"remote.call":   boolString(s.Tags.RemoteCall),
			"from.cache":    boolString(s.Tags.FromCache),
		}
		if e.Config.Env != "" {
			meta["env"] = e.Config.Env
		}
		if e.Config.Version != "" {
			meta["version"] = e.Config.Version
		}
		if s.Error != nil {
			converted.Error = 1
			if err, ok := s.Error.(error); ok {
				meta["error.message"] = err.Error()
			} else {
				meta["error.message"] = "error"
			}
		}
		converted.Meta = meta
		ddSpans = append(ddSpans, converted)
	}

	// Datadog expects a list of traces (each trace is a list of spans).
	payload := [][]ddSpan{ddSpans}
	body, err := json.Marshal(payload)
	if err == nil {
		req, err := http.NewRequest(http.MethodPut, e.Config.AgentURL, bytes.NewReader(body))
		if err == nil {
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Datadog-Trace-Count", "1")
			if resp, err := e.Config.HTTPClient.Do(req); err == nil {
				resp.Body.Close()
			}
		}
	}

	// Mirror the console exporter: drop spans shortly after export so the
	// in-memory map does not grow unbounded.
	go func() {
		time.Sleep(5 * time.Second)
		ids := make([]string, 0, len(spans))
		for _, s := range spans {
			ids = append(ids, s.TraceId)
		}
		for _, id := range ids {
			e.Broker.removeSpan(id)
		}
	}()
}

func stringToID(s string) uint64 {
	if s == "" {
		return 0
	}
	h := fnv.New64a()
	h.Write([]byte(s))
	// Datadog span/trace IDs are 64-bit unsigned integers; mask the top bit
	// to stay within the documented "non-negative" range.
	return h.Sum64() & 0x7FFFFFFFFFFFFFFF
}

func boolString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
