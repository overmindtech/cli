package discovery

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats-server/v2/test"
	"github.com/overmindtech/cli/go/auth"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// startEmbeddedNATSServer runs an in-process NATS for tests that need a live Engine Start.
func startEmbeddedNATSServer(t *testing.T) string {
	t.Helper()
	opts := test.DefaultTestOptions
	opts.Port = 4739
	s := test.RunServer(&opts)
	if !s.ReadyForConnections(10 * time.Second) {
		s.Shutdown()
		t.Fatal("could not start embedded NATS server")
	}
	t.Cleanup(func() {
		s.Shutdown()
	})
	return s.ClientURL()
}

func setupTestTracer(t *testing.T) *tracetest.InMemoryExporter {
	t.Helper()
	exp := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exp),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)
	prev := otel.GetTracerProvider()
	otel.SetTracerProvider(tp)
	t.Cleanup(func() {
		_ = tp.Shutdown(context.Background())
		otel.SetTracerProvider(prev)
	})
	return exp
}

func countExceptionEvents(spans []tracetest.SpanStub) int {
	n := 0
	for _, s := range spans {
		if s.Name != "Execute" {
			continue
		}
		for _, ev := range s.Events {
			if ev.Name == semconv.ExceptionEventName {
				n++
			}
		}
	}
	return n
}

// streamTwoSDPQueryErrorsAdapter implements ListStreamableAdapter and emits two *sdp.QueryError
// values on LIST (for multi-error Execute telemetry tests).
type streamTwoSDPQueryErrorsAdapter struct {
	*TestAdapter
}

func (a *streamTwoSDPQueryErrorsAdapter) ListStream(ctx context.Context, scope string, ignoreCache bool, stream QueryResultStream) {
	_ = ctx
	_ = scope
	_ = ignoreCache
	stream.SendError(&sdp.QueryError{
		ErrorType:   sdp.QueryError_OTHER,
		ErrorString: "first sdp query error",
	})
	stream.SendError(&sdp.QueryError{
		ErrorType:   sdp.QueryError_OTHER,
		ErrorString: "second sdp query error",
	})
}

// plainErrOnGetAdapter returns a non-QueryError from Get for every call.
type plainErrOnGetAdapter struct {
	*TestAdapter
}

func (a *plainErrOnGetAdapter) Get(ctx context.Context, scope string, query string, ignoreCache bool) (*sdp.Item, error) {
	_ = ctx
	_ = scope
	_ = query
	_ = ignoreCache
	return nil, fmt.Errorf("plain non-sdp error")
}

func TestExecute_FirstSDPQueryErrorDoesNotRecordExceptionEvent(t *testing.T) {
	exp := setupTestTracer(t)
	natsURL := startEmbeddedNATSServer(t)

	adapter := TestAdapter{
		ReturnType:   "person",
		ReturnScopes: []string{"test", "error"},
		cache:        sdpcache.NewNoOpCache(),
	}

	e := newStartedEngine(t, "TestExecuteTraceSDPQueryError", &auth.NATSOptions{
		Servers:           []string{natsURL},
		ConnectionName:    "test-connection",
		ConnectionTimeout: time.Second,
		MaxReconnects:     5,
	}, nil, &adapter)

	u := uuid.New()
	q := &sdp.Query{
		UUID:     u[:],
		Type:     "person",
		Method:   sdp.QueryMethod_GET,
		Query:    "foo",
		Scope:    "error",
		Deadline: timestamppb.New(time.Now().Add(time.Minute)),
		RecursionBehaviour: &sdp.Query_RecursionBehaviour{
			LinkDepth: 3,
		},
	}

	ch := make(chan *sdp.QueryResponse, 10)
	err := e.ExecuteQuery(context.Background(), q, ch)
	if err != nil {
		t.Fatal(err)
	}

	if n := countExceptionEvents(exp.GetSpans()); n != 0 {
		t.Fatalf("expected 0 exception events on Execute for first *sdp.QueryError, got %d", n)
	}
}

func TestExecute_SecondSDPQueryErrorRecordsExceptionEvent(t *testing.T) {
	exp := setupTestTracer(t)
	natsURL := startEmbeddedNATSServer(t)

	base := &TestAdapter{
		ReturnType:   "person",
		ReturnScopes: []string{"test"},
		cache:        sdpcache.NewNoOpCache(),
	}
	adapter := &streamTwoSDPQueryErrorsAdapter{TestAdapter: base}

	e := newStartedEngine(t, "TestExecuteTraceMultiSDPQueryError", &auth.NATSOptions{
		Servers:           []string{natsURL},
		ConnectionName:    "test-connection",
		ConnectionTimeout: time.Second,
		MaxReconnects:     5,
	}, nil, adapter)

	u := uuid.New()
	q := &sdp.Query{
		UUID:     u[:],
		Type:     "person",
		Method:   sdp.QueryMethod_LIST,
		Scope:    "test",
		Deadline: timestamppb.New(time.Now().Add(time.Minute)),
		RecursionBehaviour: &sdp.Query_RecursionBehaviour{
			LinkDepth: 3,
		},
	}

	ch := make(chan *sdp.QueryResponse, 10)
	err := e.ExecuteQuery(context.Background(), q, ch)
	if err != nil {
		t.Fatal(err)
	}

	if n := countExceptionEvents(exp.GetSpans()); n != 1 {
		t.Fatalf("expected 1 exception event on Execute (2nd *sdp.QueryError only), got %d", n)
	}
}

func TestExecute_PlainErrorStillRecordsExceptionEvent(t *testing.T) {
	exp := setupTestTracer(t)
	natsURL := startEmbeddedNATSServer(t)

	base := &TestAdapter{
		ReturnType:   "person",
		ReturnScopes: []string{"test"},
		cache:        sdpcache.NewNoOpCache(),
	}
	adapter := &plainErrOnGetAdapter{TestAdapter: base}

	e := newStartedEngine(t, "TestExecuteTracePlainErr", &auth.NATSOptions{
		Servers:           []string{natsURL},
		ConnectionName:    "test-connection",
		ConnectionTimeout: time.Second,
		MaxReconnects:     5,
	}, nil, adapter)

	u := uuid.New()
	q := &sdp.Query{
		UUID:     u[:],
		Type:     "person",
		Method:   sdp.QueryMethod_GET,
		Query:    "foo",
		Scope:    "test",
		Deadline: timestamppb.New(time.Now().Add(time.Minute)),
		RecursionBehaviour: &sdp.Query_RecursionBehaviour{
			LinkDepth: 3,
		},
	}

	ch := make(chan *sdp.QueryResponse, 10)
	err := e.ExecuteQuery(context.Background(), q, ch)
	if err != nil {
		t.Fatal(err)
	}

	if n := countExceptionEvents(exp.GetSpans()); n != 1 {
		t.Fatalf("expected 1 exception event for plain error, got %d", n)
	}
}
