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
	logger.AddHook(GCPSeverityHook{})
}

// GCPSeverityHook adds a GCP-compatible severity field to log entries.
type GCPSeverityHook struct{}

func (GCPSeverityHook) Levels() []log.Level {
	return log.AllLevels
}

func (GCPSeverityHook) Fire(entry *log.Entry) error {
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
		return "EMERGENCY"
	case log.FatalLevel:
		return "CRITICAL"
	case log.ErrorLevel:
		return "ERROR"
	case log.WarnLevel:
		return "WARNING"
	case log.InfoLevel:
		return "INFO"
	case log.DebugLevel, log.TraceLevel:
		return "DEBUG"
	default:
		return "DEFAULT"
	}
}
