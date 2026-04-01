package audit

import (
	"bufio"
	"bytes"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	log "github.com/sirupsen/logrus"
)

func TestAuditMiddleware_AuthenticatedRequest(t *testing.T) {
	var buf bytes.Buffer
	testLogger := log.New()
	testLogger.SetOutput(&buf)
	testLogger.SetFormatter(&log.JSONFormatter{})

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ad := AuditDataFromContext(r.Context()); ad != nil {
			ad.Subject = "auth0|user123"
			ad.AccountName = "acme-corp"
			ad.Scopes = "read:items write:items"
		}
		w.WriteHeader(http.StatusOK)
	})

	mw := NewAuditMiddleware(testLogger)
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/items", nil)
	mw(inner).ServeHTTP(rec, req)

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("failed to unmarshal log entry: %v", err)
	}
	if entry["method"] != "GET" {
		t.Errorf("expected method GET, got %q", entry["method"])
	}
	if entry["url"] != "/api/items" {
		t.Errorf("expected url /api/items, got %q", entry["url"])
	}
	if entry["sub"] != "auth0|user123" {
		t.Errorf("expected sub auth0|user123, got %q", entry["sub"])
	}
	if entry["account"] != "acme-corp" {
		t.Errorf("expected account acme-corp, got %q", entry["account"])
	}
	if entry["scopes"] != "read:items write:items" {
		t.Errorf("expected scopes 'read:items write:items', got %q", entry["scopes"])
	}
	if entry["ovm.audit"] != true {
		t.Errorf("expected ovm.audit true, got %v", entry["ovm.audit"])
	}
	if entry["status"] != float64(http.StatusOK) {
		t.Errorf("expected status 200, got %v", entry["status"])
	}
}

func TestAuditMiddleware_UnauthenticatedRequest(t *testing.T) {
	var buf bytes.Buffer
	testLogger := log.New()
	testLogger.SetOutput(&buf)
	testLogger.SetFormatter(&log.JSONFormatter{})

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})

	mw := NewAuditMiddleware(testLogger)
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/secret", nil)
	mw(inner).ServeHTTP(rec, req)

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("failed to unmarshal log entry: %v", err)
	}
	if entry["sub"] != "" {
		t.Errorf("expected empty sub for unauthenticated request, got %q", entry["sub"])
	}
	if entry["account"] != "" {
		t.Errorf("expected empty account for unauthenticated request, got %q", entry["account"])
	}
	if entry["status"] != float64(http.StatusUnauthorized) {
		t.Errorf("expected status 401, got %v", entry["status"])
	}
}

func TestAuditMiddleware_ExcludedPath(t *testing.T) {
	var buf bytes.Buffer
	testLogger := log.New()
	testLogger.SetOutput(&buf)
	testLogger.SetFormatter(&log.JSONFormatter{})

	called := false
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	mw := NewAuditMiddleware(testLogger, WithExcludePaths("/healthz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/healthz", nil)
	mw(inner).ServeHTTP(rec, req)

	if !called {
		t.Error("inner handler was not called for excluded path")
	}
	if buf.Len() > 0 {
		t.Errorf("expected no audit log for excluded path, got: %s", buf.String())
	}
}

func TestAuditMiddleware_NonExcludedPathStillLogged(t *testing.T) {
	var buf bytes.Buffer
	testLogger := log.New()
	testLogger.SetOutput(&buf)
	testLogger.SetFormatter(&log.JSONFormatter{})

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mw := NewAuditMiddleware(testLogger, WithExcludePaths("/healthz"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/changes", nil)
	mw(inner).ServeHTTP(rec, req)

	if buf.Len() == 0 {
		t.Error("expected audit log for non-excluded path")
	}
}

func TestAuditMiddleware_CapturesStatusCode(t *testing.T) {
	var buf bytes.Buffer
	testLogger := log.New()
	testLogger.SetOutput(&buf)
	testLogger.SetFormatter(&log.JSONFormatter{})

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	})

	mw := NewAuditMiddleware(testLogger)
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodDelete, "/api/admin/user", nil)
	mw(inner).ServeHTTP(rec, req)

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("failed to unmarshal log entry: %v", err)
	}
	if entry["status"] != float64(http.StatusForbidden) {
		t.Errorf("expected status 403, got %v", entry["status"])
	}
	if entry["method"] != "DELETE" {
		t.Errorf("expected method DELETE, got %q", entry["method"])
	}
}

func TestAuditMiddleware_DefaultStatusIs200(t *testing.T) {
	var buf bytes.Buffer
	testLogger := log.New()
	testLogger.SetOutput(&buf)
	testLogger.SetFormatter(&log.JSONFormatter{})

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})

	mw := NewAuditMiddleware(testLogger)
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/items", nil)
	mw(inner).ServeHTTP(rec, req)

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("failed to unmarshal log entry: %v", err)
	}
	if entry["status"] != float64(http.StatusOK) {
		t.Errorf("expected status 200 when handler writes body without explicit WriteHeader, got %v", entry["status"])
	}
}

func TestAuditDataFromContext_NilOutsideMiddleware(t *testing.T) {
	if ad := AuditDataFromContext(t.Context()); ad != nil {
		t.Error("expected nil AuditData outside audit middleware chain")
	}
}

func TestStatusRecorder_Hijack(t *testing.T) {
	hijacked := false
	mock := &mockHijackWriter{
		ResponseWriter: httptest.NewRecorder(),
		hijackFunc: func() (net.Conn, *bufio.ReadWriter, error) {
			hijacked = true
			return nil, nil, nil
		},
	}

	var w http.ResponseWriter = &statusRecorder{ResponseWriter: mock, status: http.StatusOK}

	h, ok := w.(http.Hijacker)
	if !ok {
		t.Fatal("statusRecorder should implement http.Hijacker")
	}

	_, _, err := h.Hijack()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hijacked {
		t.Error("expected Hijack to be delegated to underlying writer")
	}
}

func TestStatusRecorder_HijackNotSupported(t *testing.T) {
	var w http.ResponseWriter = &statusRecorder{ResponseWriter: httptest.NewRecorder(), status: http.StatusOK}

	_, _, err := w.(http.Hijacker).Hijack()
	if err == nil {
		t.Error("expected error when underlying writer doesn't support Hijack")
	}
}

func TestStatusRecorder_Flush(t *testing.T) {
	flushed := false
	mock := &mockFlushWriter{
		ResponseWriter: httptest.NewRecorder(),
		flushFunc:      func() { flushed = true },
	}

	var w http.ResponseWriter = &statusRecorder{ResponseWriter: mock, status: http.StatusOK}

	f, ok := w.(http.Flusher)
	if !ok {
		t.Fatal("statusRecorder should implement http.Flusher")
	}

	f.Flush()
	if !flushed {
		t.Error("expected Flush to be delegated to underlying writer")
	}
}

type mockHijackWriter struct {
	http.ResponseWriter
	hijackFunc func() (net.Conn, *bufio.ReadWriter, error)
}

func (m *mockHijackWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return m.hijackFunc()
}

type mockFlushWriter struct {
	http.ResponseWriter
	flushFunc func()
}

func (m *mockFlushWriter) Flush() {
	m.flushFunc()
}
