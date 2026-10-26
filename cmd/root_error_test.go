package cmd

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/getsentry/sentry-go"
	log "github.com/sirupsen/logrus"
)

func TestReportCommandErrorSkipsSentryForFlagError(t *testing.T) {
	readStderr := captureStderr(t)
	var captured []error
	capture := func(err error) *sentry.EventID {
		captured = append(captured, err)
		return nil
	}

	err := flagError{usage: "invalid --tags value 'bad', expected format 'key=value'"}
	reportCommandError(context.Background(), err, capture)

	if len(captured) != 0 {
		t.Fatalf("flagError should not be sent to Sentry, got %d events: %v", len(captured), captured)
	}
	got := readStderr()
	if !strings.Contains(got, "invalid --tags value 'bad', expected format 'key=value'") {
		t.Fatalf("flagError should still be printed to stderr, got %q", got)
	}
}

func TestReportCommandErrorSendsUnexpectedErrorsToSentry(t *testing.T) {
	var captured []error
	capture := func(err error) *sentry.EventID {
		captured = append(captured, err)
		return nil
	}

	unexpected := errors.New("backend unavailable")
	reportCommandError(context.Background(), unexpected, capture)
	if len(captured) != 1 || captured[0] != unexpected {
		t.Fatalf("unexpected errors should be sent to Sentry, got %v", captured)
	}

	logged := loggedError{
		err:     errors.New("dial tcp: lookup api.overmind.tech"),
		fields:  log.Fields{"app": "https://app.overmind.tech"},
		message: "Failed searching for existing changes",
	}
	reportCommandError(context.Background(), logged, capture)
	if len(captured) != 2 {
		t.Fatalf("loggedError should still be sent to Sentry, got %d events", len(captured))
	}
	if _, ok := captured[1].(loggedError); !ok {
		t.Fatalf("expected loggedError to be captured, got %T", captured[1])
	}
}

func TestReportCommandErrorNilCaptureIsSafe(t *testing.T) {
	// Should not panic when Sentry is not configured.
	reportCommandError(context.Background(), errors.New("boom"), nil)
	reportCommandError(context.Background(), flagError{usage: "invalid --timeout value 'nope'"}, nil)
}

func captureStderr(t *testing.T) func() string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	old := os.Stderr
	os.Stderr = w
	buf := &bytes.Buffer{}
	done := make(chan struct{})
	go func() {
		_, _ = io.Copy(buf, r)
		close(done)
	}()
	restore := func() string {
		os.Stderr = old
		_ = w.Close()
		<-done
		_ = r.Close()
		return buf.String()
	}
	t.Cleanup(func() {
		if os.Stderr == w {
			_ = restore()
		}
	})
	return restore
}
