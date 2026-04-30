package goservice

import (
	"time"
)

// LOGGER

type LogType int

const (
	LogTypeInfo LogType = iota + 1
	LogTypeWarning
	LogTypeError
)

type LogData struct {
	Type    LogType
	Message string
	Payload map[string]interface{}
	Time    int
}

type LogExternal int

const (
	LogConsole LogExternal = iota + 1
	LogFile
	// LogCustom selects a user-provided exporter supplied via Logconfig.Custom.
	// This is the Go analog of Moleculer's pluggable loggers (Pino, Bunyan,
	// Winston, Log4js, Datadog, ...): bring your own backend (zap, logrus,
	// zerolog, an HTTP shipper, ...) by implementing LoggerExternal.
	LogCustom
	// LogDatadog enables the built-in LoggerDatadog exporter that ships
	// log entries to Datadog's HTTPS log intake.
	LogDatadog
)

// LoggerExternal is the contract every log exporter must satisfy.
// Start is run in its own goroutine by the broker and is expected to block,
// draining queued entries; WriteLog enqueues a single entry from the broker.
type LoggerExternal interface {
	Start()
	WriteLog(log LogData)
}

type Logconfig struct {
	Enable   bool
	Type     LogExternal
	LogLevel LogType
	// FilePath is the destination file used by the LogFile exporter. When empty
	// the file exporter is disabled (no-op) so misconfiguration does not panic.
	FilePath string
	// Custom is the user-supplied exporter used when Type == LogCustom.
	Custom LoggerExternal
	// Datadog configures the built-in LoggerDatadog exporter used when
	// Type == LogDatadog.
	Datadog *LoggerDatadog
}

type Log struct {
	Config  Logconfig
	Extenal interface{}
}

func (b *Broker) initLog() {
	switch b.Config.LoggerConfig.Type {
	case LogConsole:
		logExternal := LoggerConsole{
			data: []LogData{},
		}
		go logExternal.Start()
		b.logs = Log{
			Config:  b.Config.LoggerConfig,
			Extenal: &logExternal,
		}
	case LogFile:
		if b.Config.LoggerConfig.FilePath == "" {
			// Preserve historical no-op behavior when no file path is set.
			return
		}
		logExternal := &LoggerFile{
			data:     []LogData{},
			filePath: b.Config.LoggerConfig.FilePath,
		}
		go logExternal.Start()
		b.logs = Log{
			Config:  b.Config.LoggerConfig,
			Extenal: logExternal,
		}
	case LogCustom:
		if b.Config.LoggerConfig.Custom == nil {
			return
		}
		go b.Config.LoggerConfig.Custom.Start()
		b.logs = Log{
			Config:  b.Config.LoggerConfig,
			Extenal: b.Config.LoggerConfig.Custom,
		}
	case LogDatadog:
		dd := b.Config.LoggerConfig.Datadog
		if dd == nil {
			return
		}
		go dd.Start()
		b.logs = Log{
			Config:  b.Config.LoggerConfig,
			Extenal: dd,
		}
	}
}

// extenal console
func (b *Broker) LogInfo(message string) {
	var log LogData
	log.Time = int(time.Now().Unix())
	log.Message = message
	log.Type = LogTypeInfo
	b.logs.exportLog(log)
}
func (b *Broker) LogWarning(message string) {
	var log LogData
	log.Time = int(time.Now().Unix())
	log.Message = message
	log.Type = LogTypeWarning
	b.logs.exportLog(log)
}
func (b *Broker) LogError(message string) {
	var log LogData
	log.Time = int(time.Now().Unix())
	log.Message = message
	log.Type = LogTypeError
	b.logs.exportLog(log)
}
func (l *Log) exportLog(log LogData) {
	if !l.Config.Enable {
		return
	}
	if l.Config.LogLevel > log.Type {
		return
	}
	if ext, ok := l.Extenal.(LoggerExternal); ok && ext != nil {
		ext.WriteLog(log)
	}
}
