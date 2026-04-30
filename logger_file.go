package goservice

import (
	"encoding/json"
	"os"
	"sync"
	"time"
)

// LoggerFile is a log exporter that appends entries to a file as JSON lines.
// It mirrors the design of LoggerConsole: WriteLog enqueues a record, Start
// drains the queue in a background goroutine.
type LoggerFile struct {
	mu       sync.Mutex
	data     []LogData
	filePath string
}

// WriteLog enqueues a log entry for the background exporter to flush to disk.
func (l *LoggerFile) WriteLog(log LogData) {
	l.mu.Lock()
	l.data = append(l.data, log)
	l.mu.Unlock()
}

// Start runs the file export loop and blocks; callers should run it in a goroutine.
func (l *LoggerFile) Start() {
	for {
		l.exportLog()
		time.Sleep(time.Millisecond)
	}
}

func (l *LoggerFile) exportLog() {
	l.mu.Lock()
	if len(l.data) == 0 {
		l.mu.Unlock()
		time.Sleep(time.Millisecond * 1)
		return
	}
	log := l.data[0]
	l.data = l.data[1:]
	l.mu.Unlock()

	if l.filePath == "" {
		return
	}
	f, err := os.OpenFile(l.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	level := "info"
	switch log.Type {
	case LogTypeWarning:
		level = "warning"
	case LogTypeError:
		level = "error"
	}
	entry := map[string]interface{}{
		"time":    time.Unix(int64(log.Time), 0).Format(time.RFC3339),
		"level":   level,
		"message": log.Message,
	}
	if log.Payload != nil {
		entry["payload"] = log.Payload
	}
	if buf, err := json.Marshal(entry); err == nil {
		buf = append(buf, '\n')
		_, _ = f.Write(buf)
	}
}
