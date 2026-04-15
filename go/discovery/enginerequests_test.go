package discovery

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/overmindtech/cli/go/auth"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
	"github.com/overmindtech/cli/go/tracing"
	"github.com/sourcegraph/conc/pool"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// executeQuerySync Executes a Query, waiting for all results, then returns them
// along with the error, rather than using channels. The singular error sill only
// be returned if the query could not be executed, otherwise all errors will be
// in the slice
func (e *Engine) executeQuerySync(ctx context.Context, q *sdp.Query) ([]*sdp.Item, []*sdp.Edge, []*sdp.QueryError, error) {
	responseChan := make(chan *sdp.QueryResponse, 100_000)
	items := make([]*sdp.Item, 0)
	edges := make([]*sdp.Edge, 0)
	errs := make([]*sdp.QueryError, 0)

	err := e.ExecuteQuery(ctx, q, responseChan)

	for r := range responseChan {
		switch r := r.GetResponseType().(type) {
		case *sdp.QueryResponse_NewItem:
			items = append(items, r.NewItem)
		case *sdp.QueryResponse_Edge:
			edges = append(edges, r.Edge)
		case *sdp.QueryResponse_Error:
			errs = append(errs, r.Error)
		}
	}

	return items, edges, errs, err
}

// cancelBlockingGetAdapter blocks in Get until the query context is cancelled.
// Used to exercise ExecuteQuery returning after the stuck-timeout path while
// a worker may still send on responses (must not close the channel until
// wg.Done).
type cancelBlockingGetAdapter struct {
	ready sync.Once
	// started is closed the first time Get begins waiting on ctx.Done().
	started chan struct{}
}

func newCancelBlockingGetAdapter() *cancelBlockingGetAdapter {
	return &cancelBlockingGetAdapter{
		started: make(chan struct{}),
	}
}

func (a *cancelBlockingGetAdapter) Type() string {
	return "blockingcancel"
}

func (a *cancelBlockingGetAdapter) Name() string {
	return "cancelBlockingGetAdapter"
}

func (a *cancelBlockingGetAdapter) Scopes() []string {
	return []string{"test"}
}

func (a *cancelBlockingGetAdapter) Metadata() *sdp.AdapterMetadata {
	return &sdp.AdapterMetadata{
		Type:            a.Type(),
		DescriptiveName: "Blocking cancel test",
	}
}

func (a *cancelBlockingGetAdapter) Get(ctx context.Context, scope, query string, _ bool) (*sdp.Item, error) {
	a.ready.Do(func() { close(a.started) })
	<-ctx.Done()
	return nil, ctx.Err()
}

func TestExecuteQuery_CancelledContextDoesNotPanicOnChannelClose(t *testing.T) {
	natsURL := startEmbeddedNATSServer(t)

	prev := executeQueryLongRunningAdaptersTimeout
	executeQueryLongRunningAdaptersTimeout = 50 * time.Millisecond
	t.Cleanup(func() { executeQueryLongRunningAdaptersTimeout = prev })

	adapter := newCancelBlockingGetAdapter()
	e := newStartedEngine(t, "TestExecuteQueryCancelClose",
		&auth.NATSOptions{
			Servers:           []string{natsURL},
			ConnectionName:    "test-connection",
			ConnectionTimeout: time.Second,
			MaxReconnects:     5,
		},
		nil,
		adapter,
	)

	ctx, cancel := context.WithCancel(context.Background())
	u := uuid.New()
	q := &sdp.Query{
		UUID:     u[:],
		Type:     adapter.Type(),
		Method:   sdp.QueryMethod_GET,
		Query:    "q",
		Scope:    "test",
		Deadline: timestamppb.New(time.Now().Add(10 * time.Minute)),
		RecursionBehaviour: &sdp.Query_RecursionBehaviour{
			LinkDepth: 0,
		},
	}

	responses := make(chan *sdp.QueryResponse, 10)
	errCh := make(chan error, 1)
	go func() {
		errCh <- e.ExecuteQuery(ctx, q, responses)
	}()

	<-adapter.started
	cancel()

	err := <-errCh
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("ExecuteQuery() err = %v, want %v", err, context.Canceled)
	}

	for range responses {
	}
}

// foreverBlockingGetAdapter ignores context cancellation and blocks in Get
// until an external signal. Used to exercise the safety timeout path.
type foreverBlockingGetAdapter struct {
	ready sync.Once
	// started is closed when Get begins blocking.
	started chan struct{}
	// release is closed by the test to let Get return.
	release chan struct{}
}

func newForeverBlockingGetAdapter() *foreverBlockingGetAdapter {
	return &foreverBlockingGetAdapter{
		started: make(chan struct{}),
		release: make(chan struct{}),
	}
}

func (a *foreverBlockingGetAdapter) Type() string            { return "foreverblocking" }
func (a *foreverBlockingGetAdapter) Name() string            { return "foreverBlockingGetAdapter" }
func (a *foreverBlockingGetAdapter) Scopes() []string        { return []string{"test"} }
func (a *foreverBlockingGetAdapter) Metadata() *sdp.AdapterMetadata {
	return &sdp.AdapterMetadata{
		Type:            a.Type(),
		DescriptiveName: "Forever blocking test",
	}
}

func (a *foreverBlockingGetAdapter) Get(_ context.Context, _, _ string, _ bool) (*sdp.Item, error) {
	a.ready.Do(func() { close(a.started) })
	<-a.release
	return nil, errors.New("released")
}

func TestExecuteQuery_SafetyTimeoutClosesResponsesWithoutPanic(t *testing.T) {
	natsURL := startEmbeddedNATSServer(t)

	prevLong := executeQueryLongRunningAdaptersTimeout
	executeQueryLongRunningAdaptersTimeout = 10 * time.Millisecond
	prevSafety := executeQuerySafetyTimeout
	executeQuerySafetyTimeout = 100 * time.Millisecond
	t.Cleanup(func() {
		executeQueryLongRunningAdaptersTimeout = prevLong
		executeQuerySafetyTimeout = prevSafety
	})

	adapter := newForeverBlockingGetAdapter()
	t.Cleanup(func() { close(adapter.release) })

	e := newStartedEngine(t, "TestExecuteQuerySafetyTimeout",
		&auth.NATSOptions{
			Servers:           []string{natsURL},
			ConnectionName:    "test-connection",
			ConnectionTimeout: time.Second,
			MaxReconnects:     5,
		},
		nil,
		adapter,
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	u := uuid.New()
	q := &sdp.Query{
		UUID:     u[:],
		Type:     adapter.Type(),
		Method:   sdp.QueryMethod_GET,
		Query:    "q",
		Scope:    "test",
		Deadline: timestamppb.New(time.Now().Add(10 * time.Minute)),
		RecursionBehaviour: &sdp.Query_RecursionBehaviour{
			LinkDepth: 0,
		},
	}

	responses := make(chan *sdp.QueryResponse, 10)
	errCh := make(chan error, 1)
	go func() {
		errCh <- e.ExecuteQuery(ctx, q, responses)
	}()

	<-adapter.started
	cancel()

	// Drain responses — the safety timeout should close the channel without
	// panicking, even though the worker is still blocked in Get.
	for range responses {
	}

	// ExecuteQuery should have returned after the stuck-timeout path.
	select {
	case err := <-errCh:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("ExecuteQuery() err = %v, want %v", err, context.Canceled)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for ExecuteQuery to return")
	}
}

func TestExecuteQuery(t *testing.T) {
	adapter := TestAdapter{
		ReturnType:   "person",
		ReturnScopes: []string{"test"},
		cache:        sdpcache.NewNoOpCache(),
	}

	e := newStartedEngine(t, "TestExecuteQuery",
		&auth.NATSOptions{
			Servers:           NatsTestURLs,
			ConnectionName:    "test-connection",
			ConnectionTimeout: time.Second,
			MaxReconnects:     5,
		},
		nil,
		&adapter,
	)

	t.Run("Basic happy-path Get query", func(t *testing.T) {
		u := uuid.New()
		q := &sdp.Query{
			UUID:   u[:],
			Type:   "person",
			Method: sdp.QueryMethod_GET,
			Query:  "foo",
			Scope:  "test",
			RecursionBehaviour: &sdp.Query_RecursionBehaviour{
				LinkDepth: 3,
			},
		}

		items, _, errs, err := e.executeQuerySync(context.Background(), q)
		if err != nil {
			t.Error(err)
		}

		for _, e := range errs {
			t.Error(e)
		}

		if x := len(adapter.GetCalls); x != 1 {
			t.Errorf("expected adapter's Get() to have been called 1 time, got %v", x)
		}

		if len(items) == 0 {
			t.Fatal("expected 1 item, got none")
		}

		if len(items) > 1 {
			t.Errorf("expected 1 item, got %v", items)
		}

		item := items[0]

		if !reflect.DeepEqual(item.GetMetadata().GetSourceQuery(), q) {
			t.Logf("adapter query: %+v", item.GetMetadata().GetSourceQuery())
			t.Logf("expected query: %+v", q)
			t.Error("adapter query mismatch")
		}
	})

	t.Run("Wrong scope Get query", func(t *testing.T) {
		q := &sdp.Query{
			Type:   "person",
			Method: sdp.QueryMethod_GET,
			Query:  "foo",
			Scope:  "wrong",
			RecursionBehaviour: &sdp.Query_RecursionBehaviour{
				LinkDepth: 0,
			},
		}

		_, _, errs, err := e.executeQuerySync(context.Background(), q)

		if err == nil {
			t.Error("expected error but got nil")
		}

		if len(errs) == 1 {
			if errs[0].GetErrorType() != sdp.QueryError_NOSCOPE {
				t.Errorf("expected error type to be NOSCOPE, got %v", errs[0].GetErrorType())
			}
		} else {
			t.Errorf("expected 1 error, got %v", len(errs))
		}
	})

	t.Run("Wrong type Get query", func(t *testing.T) {
		q := &sdp.Query{
			Type:   "house",
			Method: sdp.QueryMethod_GET,
			Query:  "foo",
			Scope:  "test",
			RecursionBehaviour: &sdp.Query_RecursionBehaviour{
				LinkDepth: 0,
			},
		}

		_, _, errs, err := e.executeQuerySync(context.Background(), q)

		if err == nil {
			t.Error("expected error but got nil")
		}

		if len(errs) == 1 {
			if errs[0].GetErrorType() != sdp.QueryError_NOSCOPE {
				t.Errorf("expected error type to be NOSCOPE, got %v", errs[0].GetErrorType())
			}
		} else {
			t.Errorf("expected 1 error, got %v", len(errs))
		}
	})

	t.Run("Basic List query", func(t *testing.T) {
		q := &sdp.Query{
			Type:   "person",
			Method: sdp.QueryMethod_LIST,
			Scope:  "test",
			RecursionBehaviour: &sdp.Query_RecursionBehaviour{
				LinkDepth: 5,
			},
		}

		items, _, errs, err := e.executeQuerySync(context.Background(), q)
		if err != nil {
			t.Error(err)
		}

		for _, e := range errs {
			t.Error(e)
		}

		if len(items) < 1 {
			t.Error("expected at least one item")
		}
	})

	t.Run("Basic Search query", func(t *testing.T) {
		q := &sdp.Query{
			Type:   "person",
			Method: sdp.QueryMethod_SEARCH,
			Query:  "TEST",
			Scope:  "test",
			RecursionBehaviour: &sdp.Query_RecursionBehaviour{
				LinkDepth: 5,
			},
		}

		items, _, errs, err := e.executeQuerySync(context.Background(), q)
		if err != nil {
			t.Error(err)
		}

		for _, e := range errs {
			t.Error(e)
		}

		if len(items) < 1 {
			t.Error("expected at least one item")
		}
	})
}

func TestHandleQuery(t *testing.T) {
	personAdapter := TestAdapter{
		ReturnType: "person",
		ReturnScopes: []string{
			"test1",
			"test2",
		},
		cache: sdpcache.NewNoOpCache(),
	}

	dogAdapter := TestAdapter{
		ReturnType: "dog",
		ReturnScopes: []string{
			"test1",
			"testA",
			"testB",
		},
		cache: sdpcache.NewNoOpCache(),
	}

	e := newStartedEngine(t, "TestHandleQuery", nil, nil, &personAdapter, &dogAdapter)

	t.Run("Wildcard type should be expanded", func(t *testing.T) {
		t.Cleanup(func() {
			personAdapter.ClearCalls()
			dogAdapter.ClearCalls()
		})

		req := sdp.Query{
			Type:   sdp.WILDCARD,
			Method: sdp.QueryMethod_GET,
			Query:  "Dylan",
			Scope:  "test1",
			RecursionBehaviour: &sdp.Query_RecursionBehaviour{
				LinkDepth: 0,
			},
		}

		// Run the handler
		e.HandleQuery(context.Background(), &req)

		// I'm expecting both adapter to get a query since the type was *
		if l := len(personAdapter.GetCalls); l != 1 {
			t.Errorf("expected person backend to have 1 Get call, got %v", l)
		}

		if l := len(dogAdapter.GetCalls); l != 1 {
			t.Errorf("expected dog backend to have 1 Get call, got %v", l)
		}
	})

	t.Run("Wildcard scope should be expanded", func(t *testing.T) {
		t.Cleanup(func() {
			personAdapter.ClearCalls()
			dogAdapter.ClearCalls()
		})

		req := sdp.Query{
			Type:   "person",
			Method: sdp.QueryMethod_GET,
			Query:  "Dylan1",
			Scope:  sdp.WILDCARD,
			RecursionBehaviour: &sdp.Query_RecursionBehaviour{
				LinkDepth: 0,
			},
		}

		// Run the handler
		e.HandleQuery(context.Background(), &req)

		if l := len(personAdapter.GetCalls); l != 2 {
			t.Errorf("expected person backend to have 2 Get calls, got %v", l)
		}

		if l := len(dogAdapter.GetCalls); l != 0 {
			t.Errorf("expected dog backend to have 0 Get calls, got %v", l)
		}
	})
}

func TestWildcardAdapterExpansion(t *testing.T) {
	personAdapter := TestAdapter{
		ReturnType: "person",
		ReturnScopes: []string{
			sdp.WILDCARD,
		},
		cache: sdpcache.NewNoOpCache(),
	}

	e := newStartedEngine(t, "TestWildcardAdapterExpansion", nil, nil, &personAdapter)

	t.Run("query scope should be preserved", func(t *testing.T) {
		req := sdp.Query{
			Type:   "person",
			Method: sdp.QueryMethod_GET,
			Query:  "Dylan1",
			Scope:  "something.specific",
			RecursionBehaviour: &sdp.Query_RecursionBehaviour{
				LinkDepth: 0,
			},
		}

		// Run the handler
		e.HandleQuery(context.Background(), &req)

		if len(personAdapter.GetCalls) != 1 {
			t.Errorf("expected 1 get call got %v", len(personAdapter.GetCalls))
		}

		if len(personAdapter.GetCalls) == 0 {
			t.Fatal("Can't continue without calls")
		}

		call := personAdapter.GetCalls[0]

		if expected := "something.specific"; call[0] != expected {
			t.Errorf("expected scope to be %v, got %v", expected, call[0])
		}

		if expected := "Dylan1"; call[1] != expected {
			t.Errorf("expected query to be %v, got %v", expected, call[1])
		}
	})
}

func TestSendQuerySync(t *testing.T) {
	SkipWithoutNats(t)

	ctx := context.Background()

	ctx, span := tracing.Tracer().Start(ctx, "TestSendQuerySync")
	defer span.End()

	adapter := TestAdapter{
		ReturnType: "person",
		ReturnScopes: []string{
			"test",
		},
		cache: sdpcache.NewNoOpCache(),
	}

	e := newStartedEngine(t, "TestSendQuerySync", nil, nil, &adapter)

	p := pool.New()
	for range 250 {
		p.Go(func() {
			u := uuid.New()
			t.Log("starting query: ", u)

			var items []*sdp.Item

			query := &sdp.Query{
				Type:   "person",
				Method: sdp.QueryMethod_GET,
				Query:  "Dylan",
				Scope:  "test",
				RecursionBehaviour: &sdp.Query_RecursionBehaviour{
					LinkDepth: 0,
				},
				IgnoreCache: false,
				UUID:        u[:],
				Deadline:    timestamppb.New(time.Now().Add(10 * time.Minute)),
			}

			items, _, errs, err := sdp.RunSourceQuerySync(ctx, query, 1*time.Second, e.natsConnection)
			if err != nil {
				t.Error(err)
			}

			if len(errs) != 0 {
				for _, err := range errs {
					t.Error(err)
				}
			}

			if len(items) != 1 {
				t.Fatalf("expected 1 item, got %v: %v", len(items), items)
			}
		})
	}

	p.Wait()
}

func TestExpandQuery(t *testing.T) {
	t.Run("with a single adapter with a single scope", func(t *testing.T) {
		simple := TestAdapter{
			ReturnScopes: []string{
				"test1",
			},
			cache: sdpcache.NewNoOpCache(),
		}
		e := newStartedEngine(t, "TestExpandQuery", nil, nil, &simple)

		e.HandleQuery(context.Background(), &sdp.Query{
			Type:   "person",
			Method: sdp.QueryMethod_GET,
			Query:  "Debby",
			Scope:  "*",
		})

		if expected := 1; len(simple.GetCalls) != expected {
			t.Errorf("Expected %v calls, got %v", expected, len(simple.GetCalls))
		}
	})

	t.Run("with a single adapter with many scopes", func(t *testing.T) {
		many := TestAdapter{
			ReturnName: "many",
			ReturnScopes: []string{
				"test1",
				"test2",
				"test3",
			},
			cache: sdpcache.NewNoOpCache(),
		}
		e := newStartedEngine(t, "TestExpandQuery", nil, nil, &many)

		e.HandleQuery(context.Background(), &sdp.Query{
			Type:   "person",
			Method: sdp.QueryMethod_GET,
			Query:  "Debby",
			Scope:  "*",
		})

		if expected := 3; len(many.GetCalls) != expected {
			t.Errorf("Expected %v calls, got %v", expected, many.GetCalls)
		}
	})

	t.Run("with a single wildcard adapter", func(t *testing.T) {
		sx := TestAdapter{
			ReturnType: "person",
			ReturnName: "sx",
			ReturnScopes: []string{
				sdp.WILDCARD,
			},
			cache: sdpcache.NewNoOpCache(),
		}

		e := newStartedEngine(t, "TestExpandQuery", nil, nil, &sx)

		e.HandleQuery(context.Background(), &sdp.Query{
			Type:   "person",
			Method: sdp.QueryMethod_LIST,
			Query:  "Rachel",
			Scope:  "*",
		})

		if expected := 1; len(sx.ListCalls) != expected {
			t.Errorf("Expected %v calls, got %v", expected, sx.ListCalls)
		}
	})
}
