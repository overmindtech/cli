package logging

import (
	log "github.com/sirupsen/logrus"
)

// ConfigureLogrusJSON sets the logger to emit JSON logs with a GCP severity field.
func ConfigureLogrusJSON(logger *log.Logger) {
	if logger == nil {
		return
	}

	logger.SetFormatter(&log.JSONFormatter{})
	logger.AddHook(OtelSeverityHook{})
}

// OtelSeverityHook adds a GCP-compatible severity field to log entries.
type OtelSeverityHook struct{}

func (OtelSeverityHook) Levels() []log.Level {
	return log.AllLevels
}

func (OtelSeverityHook) Fire(entry *log.Entry) error {
	if entry == nil {
		return nil
	}
	if _, ok := entry.Data["severity"]; ok {
		return nil
	}

	entry.Data["severity"] = severityForLevel(entry.Level)
	return nil
}

func severityForLevel(level log.Level) string {
	switch level {
	case log.PanicLevel:
		return "emergency"
	case log.FatalLevel:
		return "critical"
	case log.ErrorLevel:
		return "error"
	case log.WarnLevel:
		return "warning"
	case log.InfoLevel:
		return "info"
	case log.DebugLevel, log.TraceLevel:
		return "debug"
	default:
		return "default"
	}
}
