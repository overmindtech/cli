package logging

import (
	"bytes"
	"encoding/json"
	"testing"

	log "github.com/sirupsen/logrus"
)

func TestSeverityForLevel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		level log.Level
		want  string
	}{
		{name: "panic", level: log.PanicLevel, want: "EMERGENCY"},
		{name: "fatal", level: log.FatalLevel, want: "CRITICAL"},
		{name: "error", level: log.ErrorLevel, want: "ERROR"},
		{name: "warn", level: log.WarnLevel, want: "WARNING"},
		{name: "info", level: log.InfoLevel, want: "INFO"},
		{name: "debug", level: log.DebugLevel, want: "DEBUG"},
		{name: "trace", level: log.TraceLevel, want: "DEBUG"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := severityForLevel(tt.level)
			if got != tt.want {
				t.Errorf("severityForLevel(%v) = %q, want %q", tt.level, got, tt.want)
			}
		})
	}
}

func TestConfigureLogrusJSONAddsSeverity(t *testing.T) {
	t.Parallel()

	logger := log.New()
	var buf bytes.Buffer
	logger.SetOutput(&buf)

	ConfigureLogrusJSON(logger)
	logger.WithField("component", "test").Info("hello")

	var payload map[string]any
	if err := json.Unmarshal(buf.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal log payload: %v", err)
	}

	got, ok := payload["severity"]
	if !ok {
		t.Fatalf("expected severity field in log payload, got: %#v", payload)
	}
	if got != "INFO" {
		t.Fatalf("expected severity %q, got %v", "INFO", got)
	}
}

func TestConfigureLogrusJSONRespectsExistingSeverity(t *testing.T) {
	t.Parallel()

	logger := log.New()
	var buf bytes.Buffer
	logger.SetOutput(&buf)

	ConfigureLogrusJSON(logger)
	logger.WithField("severity", "NOTICE").Info("hello")

	var payload map[string]any
	if err := json.Unmarshal(buf.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal log payload: %v", err)
	}

	got, ok := payload["severity"]
	if !ok {
		t.Fatalf("expected severity field in log payload, got: %#v", payload)
	}
	if got != "NOTICE" {
		t.Fatalf("expected severity %q, got %v", "NOTICE", got)
	}
}
