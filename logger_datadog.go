package goservice

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"sync"
	"time"
)

// LoggerDatadog ships log entries to Datadog's HTTPS log intake.
//
// Logs are batched and flushed periodically (or immediately when the batch
// reaches FlushBatchSize). The exporter never blocks the broker call site:
// WriteLog simply appends to an in-memory buffer guarded by a mutex.
//
// Reference: https://moleculer.services/docs/0.14/logging
//
// Datadog HTTP intake reference:
// https://docs.datadoghq.com/api/latest/logs/#send-logs
type LoggerDatadog struct {
	// APIKey is the Datadog API key. Required. When empty, falls back to
	// the DD_API_KEY environment variable.
	APIKey string
	// Site is the Datadog site (e.g. "datadoghq.com", "datadoghq.eu",
	// "us3.datadoghq.com"). Defaults to "datadoghq.com".
	Site string
	// Service tags every log line with the originating service name.
	Service string
	// Source tags the log lines (e.g. "go", "goservice"). Defaults to
	// "goservice".
	Source string
	// Hostname is reported to Datadog. Defaults to os.Hostname().
	Hostname string
	// Tags are extra ddtags appended to every log line (comma separated).
	Tags string
	// Endpoint overrides the full intake URL. Mostly useful for tests.
	// When empty it is derived from Site.
	Endpoint string
	// HTTPClient is the HTTP client used for shipping. Defaults to a
	// client with a 5s timeout.
	HTTPClient *http.Client
	// FlushInterval is how often queued entries are shipped. Defaults to
	// 1 second.
	FlushInterval time.Duration
	// FlushBatchSize is the size that triggers an immediate flush.
	// Defaults to 100.
	FlushBatchSize int

	mu   sync.Mutex
	data []LogData
}

// WriteLog enqueues a log entry; the background loop flushes asynchronously.
func (l *LoggerDatadog) WriteLog(log LogData) {
	l.mu.Lock()
	l.data = append(l.data, log)
	l.mu.Unlock()
}

// Start runs the export loop and blocks; callers should run it in a goroutine.
func (l *LoggerDatadog) Start() {
	l.applyDefaults()
	for {
		l.flush()
		time.Sleep(l.FlushInterval)
	}
}

func (l *LoggerDatadog) applyDefaults() {
	if l.APIKey == "" {
		l.APIKey = os.Getenv("DD_API_KEY")
	}
	if l.Site == "" {
		if env := os.Getenv("DD_SITE"); env != "" {
			l.Site = env
		} else {
			l.Site = "datadoghq.com"
		}
	}
	if l.Source == "" {
		l.Source = "goservice"
	}
	if l.Hostname == "" {
		if h, err := os.Hostname(); err == nil {
			l.Hostname = h
		}
	}
	if l.Endpoint == "" {
		l.Endpoint = "https://http-intake.logs." + l.Site + "/api/v2/logs"
	}
	if l.HTTPClient == nil {
		l.HTTPClient = &http.Client{Timeout: 5 * time.Second}
	}
	if l.FlushInterval <= 0 {
		l.FlushInterval = time.Second
	}
	if l.FlushBatchSize <= 0 {
		l.FlushBatchSize = 100
	}
}

// flush ships any pending entries and returns the number of records shipped.
func (l *LoggerDatadog) flush() int {
	l.mu.Lock()
	if len(l.data) == 0 {
		l.mu.Unlock()
		return 0
	}
	batch := l.data
	if len(batch) > l.FlushBatchSize {
		batch = l.data[:l.FlushBatchSize]
		l.data = l.data[l.FlushBatchSize:]
	} else {
		l.data = nil
	}
	l.mu.Unlock()

	if l.APIKey == "" {
		// Without credentials we cannot ship; drop silently rather than
		// queueing forever.
		return len(batch)
	}

	payload := make([]map[string]interface{}, 0, len(batch))
	for _, entry := range batch {
		payload = append(payload, l.formatEntry(entry))
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return len(batch)
	}

	req, err := http.NewRequest(http.MethodPost, l.Endpoint, bytes.NewReader(body))
	if err != nil {
		return len(batch)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("DD-API-KEY", l.APIKey)

	resp, err := l.HTTPClient.Do(req)
	if err != nil {
		return len(batch)
	}
	resp.Body.Close()
	return len(batch)
}

func (l *LoggerDatadog) formatEntry(log LogData) map[string]interface{} {
	level := "info"
	status := "info"
	switch log.Type {
	case LogTypeWarning:
		level = "warning"
		status = "warning"
	case LogTypeError:
		level = "error"
		status = "error"
	}
	entry := map[string]interface{}{
		"message":   log.Message,
		"ddsource":  l.Source,
		"service":   l.Service,
		"hostname":  l.Hostname,
		"level":     level,
		"status":    status,
		"timestamp": time.Unix(int64(log.Time), 0).Format(time.RFC3339),
	}
	if l.Tags != "" {
		entry["ddtags"] = l.Tags
	}
	if log.Payload != nil {
		entry["payload"] = log.Payload
	}
	return entry
}
