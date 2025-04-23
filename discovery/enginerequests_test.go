package discovery

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/overmindtech/cli/auth"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/tracing"
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

func TestExecuteQuery(t *testing.T) {
	adapter := TestAdapter{
		ReturnType:   "person",
		ReturnScopes: []string{"test"},
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
		q := &sdp.Query{
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
	}

	dogAdapter := TestAdapter{
		ReturnType: "dog",
		ReturnScopes: []string{
			"test1",
			"testA",
			"testB",
		},
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
