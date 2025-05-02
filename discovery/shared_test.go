package discovery

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/goombaio/namegenerator"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
	"google.golang.org/protobuf/types/known/structpb"
)

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func randString(length int) string {
	var seededRand = rand.New(rand.NewSource(time.Now().UnixNano()))

	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

func RandomName() string {
	seed := time.Now().UTC().UnixNano()
	nameGenerator := namegenerator.NewNameGenerator(seed)
	name := nameGenerator.Generate()
	randGarbage := randString(10)
	return fmt.Sprintf("%v-%v", name, randGarbage)
}

var generation atomic.Int32

func (s *TestAdapter) NewTestItem(scope string, query string) *sdp.Item {
	gen := generation.Add(1)
	return &sdp.Item{
		Type:            s.Type(),
		Scope:           scope,
		UniqueAttribute: "name",
		Attributes: &sdp.ItemAttributes{
			AttrStruct: &structpb.Struct{
				Fields: map[string]*structpb.Value{
					"name":       structpb.NewStringValue(query),
					"age":        structpb.NewNumberValue(28),
					"generation": structpb.NewNumberValue(float64(gen)),
				},
			},
		},
		// TODO(LIQs): convert to returning edges
		LinkedItemQueries: []*sdp.LinkedItemQuery{
			{
				Query: &sdp.Query{
					Type:   "person",
					Method: sdp.QueryMethod_GET,
					Query:  RandomName(),
					Scope:  scope,
				},
			},
		},
	}
}

type TestAdapter struct {
	ReturnScopes []string
	ReturnType   string
	GetCalls     [][]string
	ListCalls    [][]string
	SearchCalls  [][]string
	IsHidden     bool
	ReturnWeight int    // Weight to be returned
	ReturnName   string // The name of the Adapter
	mutex        sync.Mutex

	CacheDuration time.Duration   // How long to cache items for
	cache         *sdpcache.Cache // The sdpcache of this Adapter
	cacheInitMu   sync.Mutex      // Mutex to ensure cache is only initialised once
}

// assert interface implementation
var _ CachingAdapter = (*TestAdapter)(nil)

// ClearCalls Clears the call counters between tests
func (s *TestAdapter) ClearCalls() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.ListCalls = make([][]string, 0)
	s.SearchCalls = make([][]string, 0)
	s.GetCalls = make([][]string, 0)
	s.cache.Clear()
}

func (s *TestAdapter) Type() string {
	if s.ReturnType != "" {
		return s.ReturnType
	}

	return "person"
}

func (s *TestAdapter) Name() string {
	return fmt.Sprintf("testAdapter-%v", s.ReturnName)
}

func (s *TestAdapter) DefaultCacheDuration() time.Duration {
	return 100 * time.Millisecond
}

func (s *TestAdapter) Metadata() *sdp.AdapterMetadata {
	return &sdp.AdapterMetadata{
		Type:            s.Type(),
		DescriptiveName: "Person",
	}
}

func (s *TestAdapter) ensureCache() {
	s.cacheInitMu.Lock()
	defer s.cacheInitMu.Unlock()

	if s.cache == nil {
		s.cache = sdpcache.NewCache()
		s.cache.MinWaitTime = 100 * time.Millisecond
	}
}

func (s *TestAdapter) Cache() *sdpcache.Cache {
	s.ensureCache()
	return s.cache
}

func (s *TestAdapter) Scopes() []string {
	if len(s.ReturnScopes) > 0 {
		return s.ReturnScopes
	}

	return []string{"test"}
}

func (s *TestAdapter) Hidden() bool {
	return s.IsHidden
}

func (s *TestAdapter) Get(ctx context.Context, scope string, query string, ignoreCache bool) (*sdp.Item, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.ensureCache()
	cacheHit, ck, cachedItems, qErr := s.cache.Lookup(ctx, s.Name(), sdp.QueryMethod_GET, scope, s.Type(), query, ignoreCache)
	if qErr != nil {
		return nil, qErr
	}
	if cacheHit {
		if len(cachedItems) > 0 {
			return cachedItems[0], nil
		} else {
			return nil, nil
		}
	}

	s.GetCalls = append(s.GetCalls, []string{scope, query})

	switch scope {
	case "empty":
		err := &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOTFOUND,
			ErrorString: "no items found",
			Scope:       scope,
		}
		s.cache.StoreError(err, s.DefaultCacheDuration(), ck)
		return nil, err
	case "error":
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "Error for testing",
			Scope:       scope,
		}
	default:
		item := s.NewTestItem(scope, query)
		s.cache.StoreItem(item, s.DefaultCacheDuration(), ck)
		return item, nil
	}
}

func (s *TestAdapter) List(ctx context.Context, scope string, ignoreCache bool) ([]*sdp.Item, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.ensureCache()
	cacheHit, ck, cachedItems, qErr := s.cache.Lookup(ctx, s.Name(), sdp.QueryMethod_LIST, scope, s.Type(), "", ignoreCache)
	if qErr != nil {
		return nil, qErr
	}
	if cacheHit {
		return cachedItems, nil
	}

	s.ListCalls = append(s.ListCalls, []string{scope})

	switch scope {
	case "empty":
		err := &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOTFOUND,
			ErrorString: "no items found",
			Scope:       scope,
		}
		s.cache.StoreError(err, s.DefaultCacheDuration(), ck)
		return nil, err
	case "error":
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "Error for testing",
			Scope:       scope,
		}
	default:
		item := s.NewTestItem(scope, "Dylan")
		s.cache.StoreItem(item, s.DefaultCacheDuration(), ck)
		return []*sdp.Item{item}, nil
	}
}

func (s *TestAdapter) Search(ctx context.Context, scope string, query string, ignoreCache bool) ([]*sdp.Item, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.ensureCache()
	cacheHit, ck, cachedItems, qErr := s.cache.Lookup(ctx, s.Name(), sdp.QueryMethod_SEARCH, scope, s.Type(), query, ignoreCache)
	if qErr != nil {
		return nil, qErr
	}
	if cacheHit {
		return cachedItems, nil
	}

	s.SearchCalls = append(s.SearchCalls, []string{scope, query})

	switch scope {
	case "empty":
		err := &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOTFOUND,
			ErrorString: "no items found",
			Scope:       scope,
		}
		s.cache.StoreError(err, s.DefaultCacheDuration(), ck)
		return nil, err
	case "error":
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "Error for testing",
			Scope:       scope,
		}
	default:
		item := s.NewTestItem(scope, "Dylan")
		s.cache.StoreItem(item, s.DefaultCacheDuration(), ck)
		return []*sdp.Item{item}, nil
	}
}

func (s *TestAdapter) Weight() int {
	return s.ReturnWeight
}
