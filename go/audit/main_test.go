package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/overmindtech/cli/go/auth"
	log "github.com/sirupsen/logrus"
)

func TestAuditMiddleware(t *testing.T) {
	called := false
	var buf bytes.Buffer
	testLogger := log.New()
	testLogger.SetOutput(&buf)
	testLogger.SetFormatter(&log.JSONFormatter{})

	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	mw := NewAuditMiddleware(testLogger)
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/testpath", nil)
	req = req.WithContext(context.WithValue(t.Context(), auth.CurrentSubjectContextKey{}, "testvalue"))
	req = req.WithContext(context.WithValue(req.Context(), auth.CustomClaimsContextKey{}, &auth.CustomClaims{
		Scope: "testScope",
	}))
	mw(h).ServeHTTP(rec, req)

	if !called {
		t.Error("handler was not called")
	}
	var logEntry map[string]any
	err := json.Unmarshal(buf.Bytes(), &logEntry)
	if err != nil {
		t.Fatalf("failed to unmarshal log entry: %v", err)
	}
	if logEntry["method"] != "GET" {
		t.Errorf("expected method to be 'GET', got '%s'", logEntry["method"])
	}
	if logEntry["url"] != "/testpath" {
		t.Errorf("expected url to be '/testpath', got '%s'", logEntry["url"])
	}
	if logEntry["sub"] != "testvalue" {
		t.Errorf("expected subject to be 'testvalue', got '%s'", logEntry["sub"])
	}
	if logEntry["account"] != "not set in context" {
		t.Errorf("expected account to be 'not set in context', got '%s'", logEntry["account"])
	}
	if logEntry["ovm.audit"] != true {
		t.Errorf("expected ovm.audit to be true, got '%v'", logEntry["ovm.audit"])
	}
	if logEntry["level"] != "info" {
		t.Errorf("expected log level to be 'info', got '%s'", logEntry["level"])
	}
	if logEntry["scopes"] != "testScope" {
		t.Errorf("expected scopes to be nil, got '%v'", logEntry["scopes"])
	}
}
