package discovery

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/overmindtech/cli/sdp-go"
	"google.golang.org/protobuf/types/known/structpb"
)

type SpeedTestAdapter struct {
	QueryDelay   time.Duration
	ReturnType   string
	ReturnScopes []string
}

func (s *SpeedTestAdapter) Type() string {
	if s.ReturnType != "" {
		return s.ReturnType
	}

	return "person"
}

func (s *SpeedTestAdapter) Name() string {
	return "SpeedTestAdapter"
}

func (s *SpeedTestAdapter) Scopes() []string {
	if len(s.ReturnScopes) > 0 {
		return s.ReturnScopes
	}

	return []string{"test"}
}

func (s *SpeedTestAdapter) Metadata() *sdp.AdapterMetadata {
	return &sdp.AdapterMetadata{}
}

func (s *SpeedTestAdapter) Get(ctx context.Context, scope string, query string, ignoreCache bool) (*sdp.Item, error) {
	select {
	case <-time.After(s.QueryDelay):
		return &sdp.Item{
			Type:            s.Type(),
			UniqueAttribute: "name",
			Attributes: &sdp.ItemAttributes{
				AttrStruct: &structpb.Struct{
					Fields: map[string]*structpb.Value{
						"name": {
							Kind: &structpb.Value_StringValue{
								StringValue: query,
							},
						},
					},
				},
			},
			// TODO(LIQs): convert to returning edges
			LinkedItemQueries: []*sdp.LinkedItemQuery{
				{
					Query: &sdp.Query{
						Type:   "person",
						Method: sdp.QueryMethod_GET,
						Query:  query + time.Now().String(),
						Scope:  scope,
					},
				},
			},
			Scope: scope,
		}, nil
	case <-ctx.Done():
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_TIMEOUT,
			ErrorString: ctx.Err().Error(),
			Scope:       scope,
		}
	}
}

func (s *SpeedTestAdapter) List(ctx context.Context, scope string, ignoreCache bool) ([]*sdp.Item, error) {
	item, err := s.Get(ctx, scope, "dylan", ignoreCache)

	return []*sdp.Item{item}, err
}

func (s *SpeedTestAdapter) Weight() int {
	return 10
}

func TestExecute(t *testing.T) {
	adapter := TestAdapter{
		ReturnType: "person",
		ReturnScopes: []string{
			"test",
		},
	}

	e := newStartedEngine(t, "TestExecute", nil, nil, &adapter)

	t.Run("Without linking", func(t *testing.T) {
		t.Parallel()

		qt := QueryTracker{
			Engine: e,
			Query: &sdp.Query{
				Type:   "person",
				Method: sdp.QueryMethod_GET,
				Query:  "Dylan",
				RecursionBehaviour: &sdp.Query_RecursionBehaviour{
					LinkDepth: 0,
				},
				Scope: "test",
			},
		}

		items, edges, errs, err := qt.Execute(context.Background())
		if err != nil {
			t.Error(err)
		}

		for _, e := range errs {
			t.Error(e)
		}

		if l := len(items); l != 1 {
			t.Errorf("expected 1 items, got %v: %v", l, items)
		}

		if l := len(edges); l != 0 {
			t.Errorf("expected 0 items, got %v: %v", l, edges)
		}
	})

	t.Run("With no engine", func(t *testing.T) {
		t.Parallel()

		qt := QueryTracker{
			Engine: nil,
			Query: &sdp.Query{
				Type:   "person",
				Method: sdp.QueryMethod_GET,
				Query:  "Dylan",
				RecursionBehaviour: &sdp.Query_RecursionBehaviour{
					LinkDepth: 10,
				},
				Scope: "test",
			},
		}

		_, _, _, err := qt.Execute(context.Background())

		if err == nil {
			t.Error("expected error but got nil")
		}
	})

	t.Run("With no queries", func(t *testing.T) {
		t.Parallel()

		qt := QueryTracker{
			Engine: e,
		}

		_, _, _, err := qt.Execute(context.Background())
		if err != nil {
			t.Error(err)
		}
	})
}

func TestTimeout(t *testing.T) {
	adapter := SpeedTestAdapter{
		QueryDelay: 100 * time.Millisecond,
	}
	e := newStartedEngine(t, "TestTimeout", nil, nil, &adapter)

	t.Run("With a timeout, but not exceeding it", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)

		qt := QueryTracker{
			Engine:  e,
			Context: ctx,
			Cancel:  cancel,
			Query: &sdp.Query{
				Type:   "person",
				Method: sdp.QueryMethod_GET,
				Query:  "Dylan",
				RecursionBehaviour: &sdp.Query_RecursionBehaviour{
					LinkDepth: 0,
				},
				Scope: "test",
			},
		}

		items, edges, errs, err := qt.Execute(context.Background())
		if err != nil {
			t.Error(err)
		}

		for _, e := range errs {
			t.Error(e)
		}

		if l := len(items); l != 1 {
			t.Errorf("expected 1 items, got %v: %v", l, items)
		}

		if l := len(edges); l != 0 {
			t.Errorf("expected 0 edges, got %v: %v", l, edges)
		}
	})

	t.Run("With a timeout that is exceeded", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)

		qt := QueryTracker{
			Engine:  e,
			Context: ctx,
			Cancel:  cancel,
			Query: &sdp.Query{
				Type:   "person",
				Method: sdp.QueryMethod_GET,
				Query:  "somethingElse",
				RecursionBehaviour: &sdp.Query_RecursionBehaviour{
					LinkDepth: 0,
				},
				Scope: "test",
			},
		}

		_, _, _, err := qt.Execute(ctx)

		if err == nil {
			t.Error("Expected timeout but got no error")
		}
	})
}

func TestCancel(t *testing.T) {
	e := newStartedEngine(t, "TestCancel", nil, nil)

	u := uuid.New()
	ctx, cancel := context.WithCancel(context.Background())

	qt := QueryTracker{
		Engine:  e,
		Context: ctx,
		Cancel:  cancel,
		Query: &sdp.Query{
			Type:   "person",
			Method: sdp.QueryMethod_GET,
			Query:  "somethingElse1",
			RecursionBehaviour: &sdp.Query_RecursionBehaviour{
				LinkDepth: 10,
			},
			Scope: "test",
			UUID:  u[:],
		},
	}

	items := make([]*sdp.Item, 0)
	edges := make([]*sdp.Edge, 0)
	var wg sync.WaitGroup

	var err error
	wg.Add(1)
	go func() {
		items, edges, _, err = qt.Execute(context.Background())
		wg.Done()
	}()

	// Give it some time to populate the cancelFunc
	time.Sleep(100 * time.Millisecond)

	qt.Cancel()

	wg.Wait()

	if err == nil {
		t.Error("expected error but got none")
	}

	if len(items) != 0 {
		t.Errorf("Expected no items but got %v", items)
	}

	if len(edges) != 0 {
		t.Errorf("Expected no edges but got %v", edges)
	}
}
