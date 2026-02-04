package adapters

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
)

// TestGetListAdapterV2_GetNotFoundCaching tests that GetListAdapterV2 caches not-found error results
func TestGetListAdapterV2_GetNotFoundCaching(t *testing.T) {
	ctx := context.Background()
	cache := sdpcache.NewMemoryCache()
	getCalls := 0

	// Mock AWS item type
	type MockAWSItem struct {
		Name string
	}

	adapter := &GetListAdapterV2[*MockInput, *MockOutput, *MockAWSItem, *MockClient, *MockOptions]{
		ItemType:  "test-item",
		cache:     cache,
		AccountID: "123456789012",
		Region:    "us-east-1",
		GetFunc: func(ctx context.Context, client *MockClient, scope string, query string) (*MockAWSItem, error) {
			getCalls++
			// Return NOTFOUND error (typical AWS behavior)
			return nil, &sdp.QueryError{
				ErrorType:   sdp.QueryError_NOTFOUND,
				ErrorString: "resource not found",
				Scope:       scope,
			}
		},
		ItemMapper: func(query *string, scope string, awsItem *MockAWSItem) (*sdp.Item, error) {
			return &sdp.Item{
				Type:            "test-item",
				UniqueAttribute: "name",
				Attributes:      &sdp.ItemAttributes{},
				Scope:           scope,
			}, nil
		},
		AdapterMetadata: &sdp.AdapterMetadata{
			Type:            "test-item",
			DescriptiveName: "Test Item",
			SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
				Get:             true,
				List:            true,
				GetDescription:  "Get a test item",
				ListDescription: "List all test items",
			},
		},
	}

	// First call should invoke GetFunc and get error
	item, err := adapter.Get(ctx, "123456789012.us-east-1", "test-query", false)
	if item != nil {
		t.Errorf("Expected nil item, got %v", item)
	}
	// First call returns the error (but it's cached)
	if err == nil {
		t.Error("Expected NOTFOUND error, got nil")
	}
	if getCalls != 1 {
		t.Errorf("Expected 1 GetFunc call, got %d", getCalls)
	}

	// Second call should hit cache and return the cached NOTFOUND error
	item, err = adapter.Get(ctx, "123456789012.us-east-1", "test-query", false)
	if item != nil {
		t.Errorf("Expected nil item on cache hit, got %v", item)
	}
	var qErr *sdp.QueryError
	if err == nil {
		t.Error("Expected NOTFOUND error on cache hit, got nil")
	} else if !errors.As(err, &qErr) || qErr.GetErrorType() != sdp.QueryError_NOTFOUND {
		t.Errorf("Expected NOTFOUND error on cache hit, got %v", err)
	}
	if getCalls != 1 {
		t.Errorf("Expected still 1 GetFunc call (cache hit), got %d", getCalls)
	}
}

// TestGetListAdapterV2_ListNotFoundCaching tests that GetListAdapterV2 caches not-found results when LIST returns 0 items
func TestGetListAdapterV2_ListNotFoundCaching(t *testing.T) {
	ctx := context.Background()
	cache := sdpcache.NewMemoryCache()
	listCalls := 0

	type MockAWSItem struct {
		Name string
	}

	adapter := &GetListAdapterV2[*MockInput, *MockOutput, *MockAWSItem, *MockClient, *MockOptions]{
		ItemType:  "test-item",
		cache:     cache,
		AccountID: "123456789012",
		Region:    "us-east-1",
		GetFunc: func(ctx context.Context, client *MockClient, scope string, query string) (*MockAWSItem, error) {
			return nil, errors.New("should not be called in LIST test")
		},
		InputMapperList: func(scope string) (*MockInput, error) {
			return &MockInput{}, nil
		},
		ListFunc: func(ctx context.Context, client *MockClient, input *MockInput) (*MockOutput, error) {
			listCalls++
			return &MockOutput{}, nil
		},
		ListExtractor: func(ctx context.Context, output *MockOutput, client *MockClient) ([]*MockAWSItem, error) {
			// Return empty slice to simulate no items found
			return []*MockAWSItem{}, nil
		},
		ItemMapper: func(query *string, scope string, awsItem *MockAWSItem) (*sdp.Item, error) {
			return &sdp.Item{
				Type:            "test-item",
				UniqueAttribute: "name",
				Attributes:      &sdp.ItemAttributes{},
				Scope:           scope,
			}, nil
		},
		AdapterMetadata: &sdp.AdapterMetadata{
			Type:            "test-item",
			DescriptiveName: "Test Item",
			SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
				Get:             true,
				List:            true,
				GetDescription:  "Get a test item",
				ListDescription: "List all test items",
			},
		},
	}

	// Use test stream to collect results
	stream := &testQueryResultStream{}

	// First call should invoke ListFunc
	adapter.ListStream(ctx, "123456789012.us-east-1", false, stream)
	if len(stream.items) != 0 {
		t.Errorf("Expected 0 items, got %d", len(stream.items))
	}
	if listCalls != 1 {
		t.Errorf("Expected 1 ListFunc call, got %d", listCalls)
	}

	// Second call should hit cache
	stream2 := &testQueryResultStream{}
	adapter.ListStream(ctx, "123456789012.us-east-1", false, stream2)
	if len(stream2.items) != 0 {
		t.Errorf("Expected 0 items on cache hit, got %d", len(stream2.items))
	}
	// For backward compatibility, cached NOTFOUND is treated as empty result (no error)
	// This matches the behavior of the first call which returns empty stream with no errors
	if len(stream2.errors) != 0 {
		t.Errorf("Expected 0 errors from cache (backward compatibility), got %d", len(stream2.errors))
	}
	if listCalls != 1 {
		t.Errorf("Expected still 1 ListFunc call (cache hit), got %d", listCalls)
	}
}

// TestGetListAdapter_GetNotFoundCaching tests GetListAdapter's GET not-found caching
func TestGetListAdapter_GetNotFoundCaching(t *testing.T) {
	ctx := context.Background()
	cache := sdpcache.NewMemoryCache()
	getCalls := 0

	type MockAWSItem struct {
		Name string
	}

	adapter := &GetListAdapter[*MockAWSItem, *MockClient, *MockOptions]{
		ItemType:  "test-item",
		cache:     cache,
		AccountID: "123456789012",
		Region:    "us-east-1",
		GetFunc: func(ctx context.Context, client *MockClient, scope string, query string) (*MockAWSItem, error) {
			getCalls++
			// Return NOTFOUND error (typical AWS behavior)
			return nil, &sdp.QueryError{
				ErrorType:   sdp.QueryError_NOTFOUND,
				ErrorString: "resource not found",
				Scope:       scope,
			}
		},
		ItemMapper: func(query, scope string, awsItem *MockAWSItem) (*sdp.Item, error) {
			return &sdp.Item{
				Type:            "test-item",
				UniqueAttribute: "name",
				Attributes:      &sdp.ItemAttributes{},
				Scope:           scope,
			}, nil
		},
		AdapterMetadata: &sdp.AdapterMetadata{
			Type:            "test-item",
			DescriptiveName: "Test Item",
			SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
				Get:             true,
				List:            true,
				GetDescription:  "Get a test item",
				ListDescription: "List all test items",
			},
		},
	}

	// First call returns error (which gets cached)
	item, err := adapter.Get(ctx, "123456789012.us-east-1", "test-query", false)
	if item != nil {
		t.Errorf("Expected nil item, got %v", item)
	}
	if err == nil {
		t.Error("Expected NOTFOUND error, got nil")
	}
	if getCalls != 1 {
		t.Errorf("Expected 1 GetFunc call, got %d", getCalls)
	}

	// Second call should hit cache and return the cached NOTFOUND error
	item, err = adapter.Get(ctx, "123456789012.us-east-1", "test-query", false)
	if item != nil {
		t.Errorf("Expected nil item on cache hit, got %v", item)
	}
	var qErr *sdp.QueryError
	if err == nil {
		t.Error("Expected NOTFOUND error on cache hit, got nil")
	} else if !errors.As(err, &qErr) || qErr.GetErrorType() != sdp.QueryError_NOTFOUND {
		t.Errorf("Expected NOTFOUND error on cache hit, got %v", err)
	}
	if getCalls != 1 {
		t.Errorf("Expected still 1 GetFunc call (cache hit), got %d", getCalls)
	}
}

// TestGetListAdapter_ListNotFoundCaching tests GetListAdapter's LIST not-found caching
func TestGetListAdapter_ListNotFoundCaching(t *testing.T) {
	ctx := context.Background()
	cache := sdpcache.NewMemoryCache()
	listCalls := 0

	type MockAWSItem struct {
		Name string
	}

	adapter := &GetListAdapter[*MockAWSItem, *MockClient, *MockOptions]{
		ItemType:  "test-item",
		cache:     cache,
		AccountID: "123456789012",
		Region:    "us-east-1",
		GetFunc: func(ctx context.Context, client *MockClient, scope string, query string) (*MockAWSItem, error) {
			return nil, errors.New("should not be called")
		},
		ListFunc: func(ctx context.Context, client *MockClient, scope string) ([]*MockAWSItem, error) {
			listCalls++
			return []*MockAWSItem{}, nil // Empty list
		},
		ItemMapper: func(query, scope string, awsItem *MockAWSItem) (*sdp.Item, error) {
			return &sdp.Item{
				Type:            "test-item",
				UniqueAttribute: "name",
				Attributes:      &sdp.ItemAttributes{},
				Scope:           scope,
			}, nil
		},
		AdapterMetadata: &sdp.AdapterMetadata{
			Type:            "test-item",
			DescriptiveName: "Test Item",
			SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
				Get:             true,
				List:            true,
				GetDescription:  "Get a test item",
				ListDescription: "List all test items",
			},
		},
	}

	// First call
	items, err := adapter.List(ctx, "123456789012.us-east-1", false)
	if len(items) != 0 {
		t.Errorf("Expected 0 items, got %d", len(items))
	}
	if err != nil {
		t.Errorf("Expected nil error, got %v", err)
	}
	if listCalls != 1 {
		t.Errorf("Expected 1 ListFunc call, got %d", listCalls)
	}

	// Second call should hit cache and return empty result with nil error (backward compatibility)
	items2, err := adapter.List(ctx, "123456789012.us-east-1", false)
	// Should get empty result with nil error for backward compatibility
	if len(items2) != 0 {
		t.Errorf("Expected 0 items from cache, got %d", len(items2))
	}
	if err != nil {
		t.Errorf("Expected nil error from cache (backward compat), got %v", err)
	}
	if listCalls != 1 {
		t.Errorf("Expected still 1 ListFunc call (cache hit), got %d", listCalls)
	}
}

// TestAlwaysGetAdapter_GetNotFoundCaching tests AlwaysGetAdapter's GET not-found caching
func TestAlwaysGetAdapter_GetNotFoundCaching(t *testing.T) {
	ctx := context.Background()
	cache := sdpcache.NewMemoryCache()
	getFuncCalls := 0

	adapter := &AlwaysGetAdapter[*MockInput, *MockOutput, *MockGetInput, *MockGetOutput, *MockClient, *MockOptions]{
		ItemType:  "test-item",
		cache:     cache,
		AccountID: "123456789012",
		Region:    "us-east-1",
		GetInputMapper: func(scope, query string) *MockGetInput {
			return &MockGetInput{}
		},
		GetFunc: func(ctx context.Context, client *MockClient, scope string, input *MockGetInput) (*sdp.Item, error) {
			getFuncCalls++
			// Return NOTFOUND error (typical AWS behavior)
			return nil, &sdp.QueryError{
				ErrorType:   sdp.QueryError_NOTFOUND,
				ErrorString: "resource not found",
				Scope:       scope,
			}
		},
		// Add ListFuncPaginatorBuilder to avoid validation error
		ListFuncPaginatorBuilder: func(client *MockClient, input *MockInput) Paginator[*MockOutput, *MockOptions] {
			return nil // Not used in GET test
		},
		ListFuncOutputMapper: func(output *MockOutput, input *MockInput) ([]*MockGetInput, error) {
			return nil, nil // Not used in GET test
		},
		AdapterMetadata: &sdp.AdapterMetadata{
			Type:            "test-item",
			DescriptiveName: "Test Item",
			SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
				Get:             true,
				List:            true,
				GetDescription:  "Get a test item",
				ListDescription: "List all test items",
			},
		},
	}

	// First call returns error (which gets cached)
	item, err := adapter.Get(ctx, "123456789012.us-east-1", "test-query", false)
	if item != nil {
		t.Errorf("Expected nil item, got %v", item)
	}
	if err == nil {
		t.Error("Expected NOTFOUND error, got nil")
	}
	if getFuncCalls != 1 {
		t.Errorf("Expected 1 GetFunc call, got %d", getFuncCalls)
	}

	// Second call should hit cache and return the cached NOTFOUND error
	item, err = adapter.Get(ctx, "123456789012.us-east-1", "test-query", false)
	if item != nil {
		t.Errorf("Expected nil item on cache hit, got %v", item)
	}
	var qErr *sdp.QueryError
	if err == nil {
		t.Error("Expected NOTFOUND error on cache hit, got nil")
	} else if !errors.As(err, &qErr) || qErr.GetErrorType() != sdp.QueryError_NOTFOUND {
		t.Errorf("Expected NOTFOUND error on cache hit, got %v", err)
	}
	if getFuncCalls != 1 {
		t.Errorf("Expected still 1 GetFunc call (cache hit), got %d", getFuncCalls)
	}
}

// TestDescribeOnlyAdapter_ListNotFoundCaching tests DescribeOnlyAdapter's LIST not-found caching
func TestDescribeOnlyAdapter_ListNotFoundCaching(t *testing.T) {
	ctx := context.Background()
	cache := sdpcache.NewMemoryCache()
	describeCalls := 0

	adapter := &DescribeOnlyAdapter[*MockInput, *MockOutput, *MockClient, *MockOptions]{
		ItemType:          "test-item",
		cache:             cache,
		AccountID:         "123456789012",
		Region:            "us-east-1",
		MaxResultsPerPage: 100, // Set to avoid validation using default
		DescribeFunc: func(ctx context.Context, client *MockClient, input *MockInput) (*MockOutput, error) {
			describeCalls++
			return &MockOutput{}, nil
		},
		InputMapperGet: func(scope, query string) (*MockInput, error) {
			return &MockInput{}, nil
		},
		InputMapperList: func(scope string) (*MockInput, error) {
			return &MockInput{}, nil
		},
		OutputMapper: func(ctx context.Context, client *MockClient, scope string, input *MockInput, output *MockOutput) ([]*sdp.Item, error) {
			// Return empty slice to simulate no items found
			return []*sdp.Item{}, nil
		},
		AdapterMetadata: &sdp.AdapterMetadata{
			Type:            "test-item",
			DescriptiveName: "Test Item",
			SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
				Get:             true,
				List:            true,
				GetDescription:  "Get a test item",
				ListDescription: "List all test items",
			},
		},
	}

	stream := &testQueryResultStream{}

	// First call
	adapter.ListStream(ctx, "123456789012.us-east-1", false, stream)
	if len(stream.items) != 0 {
		t.Errorf("Expected 0 items, got %d", len(stream.items))
	}
	if describeCalls != 1 {
		t.Errorf("Expected 1 DescribeFunc call, got %d", describeCalls)
	}

	// Second call should hit cache
	stream2 := &testQueryResultStream{}
	adapter.ListStream(ctx, "123456789012.us-east-1", false, stream2)
	if len(stream2.items) != 0 {
		t.Errorf("Expected 0 items on cache hit, got %d", len(stream2.items))
	}
	// For backward compatibility, cached NOTFOUND is treated as empty result (no error)
	// This matches the behavior of the first call which returns empty stream with no errors
	if len(stream2.errors) != 0 {
		t.Errorf("Expected 0 errors from cache (backward compatibility), got %d", len(stream2.errors))
	}
	if describeCalls != 1 {
		t.Errorf("Expected still 1 DescribeFunc call (cache hit), got %d", describeCalls)
	}
}

// Mock types for testing
type MockClient struct{}
type MockInput struct{}
type MockOutput struct{}
type MockGetInput struct{}
type MockGetOutput struct{}
type MockOptions struct{}

// testQueryResultStream is a simple implementation of QueryResultStream for testing
type testQueryResultStream struct {
	items  []*sdp.Item
	errors []*sdp.QueryError
}

func (s *testQueryResultStream) SendItem(item *sdp.Item) {
	s.items = append(s.items, item)
}

func (s *testQueryResultStream) SendError(err error) {
	var qErr *sdp.QueryError
	if errors.As(err, &qErr) {
		s.errors = append(s.errors, qErr)
	} else {
		s.errors = append(s.errors, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: err.Error(),
		})
	}
}

// TestNotFoundCacheExpiry tests that not-found cache entries expire correctly
func TestNotFoundCacheExpiry(t *testing.T) {
	ctx := context.Background()
	cache := sdpcache.NewMemoryCache()
	getFuncCalls := 0

	type MockAWSItem struct {
		Name string
	}

	adapter := &GetListAdapterV2[*MockInput, *MockOutput, *MockAWSItem, *MockClient, *MockOptions]{
		ItemType:      "test-item",
		cache:         cache,
		CacheDuration: 100 * time.Millisecond, // Short duration for testing
		AccountID:     "123456789012",
		Region:        "us-east-1",
		GetFunc: func(ctx context.Context, client *MockClient, scope string, query string) (*MockAWSItem, error) {
			getFuncCalls++
			return nil, &sdp.QueryError{
				ErrorType:   sdp.QueryError_NOTFOUND,
				ErrorString: "not found",
			}
		},
		ItemMapper: func(query *string, scope string, awsItem *MockAWSItem) (*sdp.Item, error) {
			return &sdp.Item{
				Type:            "test-item",
				UniqueAttribute: "name",
				Attributes:      &sdp.ItemAttributes{},
				Scope:           scope,
			}, nil
		},
		AdapterMetadata: &sdp.AdapterMetadata{
			Type:            "test-item",
			DescriptiveName: "Test Item",
			SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
				Get:             true,
				List:            true,
				GetDescription:  "Get a test item",
				ListDescription: "List all test items",
			},
		},
	}

	// First call - should cache not-found
	_, _ = adapter.Get(ctx, "123456789012.us-east-1", "test-query", false)
	if getFuncCalls != 1 {
		t.Errorf("Expected 1 GetFunc call, got %d", getFuncCalls)
	}

	// Immediate second call - should hit cache
	_, _ = adapter.Get(ctx, "123456789012.us-east-1", "test-query", false)
	if getFuncCalls != 1 {
		t.Errorf("Expected still 1 GetFunc call (cache hit), got %d", getFuncCalls)
	}

	// Wait for cache to expire
	time.Sleep(150 * time.Millisecond)

	// Third call after expiry - should invoke GetFunc again
	_, _ = adapter.Get(ctx, "123456789012.us-east-1", "test-query", false)
	if getFuncCalls != 2 {
		t.Errorf("Expected 2 GetFunc calls (cache expired), got %d", getFuncCalls)
	}
}

// TestNotFoundCacheIgnoreCache tests that ignoreCache parameter bypasses not-found cache
func TestNotFoundCacheIgnoreCache(t *testing.T) {
	ctx := context.Background()
	cache := sdpcache.NewMemoryCache()
	getFuncCalls := 0

	type MockAWSItem struct {
		Name string
	}

	adapter := &GetListAdapterV2[*MockInput, *MockOutput, *MockAWSItem, *MockClient, *MockOptions]{
		ItemType:  "test-item",
		cache:     cache,
		AccountID: "123456789012",
		Region:    "us-east-1",
		GetFunc: func(ctx context.Context, client *MockClient, scope string, query string) (*MockAWSItem, error) {
			getFuncCalls++
			return nil, &sdp.QueryError{
				ErrorType:   sdp.QueryError_NOTFOUND,
				ErrorString: "not found",
			}
		},
		ItemMapper: func(query *string, scope string, awsItem *MockAWSItem) (*sdp.Item, error) {
			return &sdp.Item{
				Type:            "test-item",
				UniqueAttribute: "name",
				Attributes:      &sdp.ItemAttributes{},
				Scope:           scope,
			}, nil
		},
		AdapterMetadata: &sdp.AdapterMetadata{
			Type:            "test-item",
			DescriptiveName: "Test Item",
			SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
				Get:             true,
				List:            true,
				GetDescription:  "Get a test item",
				ListDescription: "List all test items",
			},
		},
	}

	// First call with ignoreCache=false
	_, _ = adapter.Get(ctx, "123456789012.us-east-1", "test-query", false)
	if getFuncCalls != 1 {
		t.Errorf("Expected 1 GetFunc call, got %d", getFuncCalls)
	}

	// Second call with ignoreCache=true - should bypass cache
	_, _ = adapter.Get(ctx, "123456789012.us-east-1", "test-query", true)
	if getFuncCalls != 2 {
		t.Errorf("Expected 2 GetFunc calls (ignore cache), got %d", getFuncCalls)
	}

	// Third call with ignoreCache=false - should still hit cache from first call
	_, _ = adapter.Get(ctx, "123456789012.us-east-1", "test-query", false)
	if getFuncCalls != 2 {
		t.Errorf("Expected still 2 GetFunc calls (cache hit), got %d", getFuncCalls)
	}
}

// TestNotFoundCacheDifferentQueries tests that different queries get separate cache entries
func TestNotFoundCacheDifferentQueries(t *testing.T) {
	ctx := context.Background()
	cache := sdpcache.NewMemoryCache()
	getFuncCalls := 0
	queriesReceived := make(map[string]int)

	type MockAWSItem struct {
		Name string
	}

	adapter := &GetListAdapterV2[*MockInput, *MockOutput, *MockAWSItem, *MockClient, *MockOptions]{
		ItemType:  "test-item",
		cache:     cache,
		AccountID: "123456789012",
		Region:    "us-east-1",
		GetFunc: func(ctx context.Context, client *MockClient, scope string, query string) (*MockAWSItem, error) {
			getFuncCalls++
			queriesReceived[query]++
			return nil, &sdp.QueryError{
				ErrorType:   sdp.QueryError_NOTFOUND,
				ErrorString: "not found",
			}
		},
		ItemMapper: func(query *string, scope string, awsItem *MockAWSItem) (*sdp.Item, error) {
			return &sdp.Item{
				Type:            "test-item",
				UniqueAttribute: "name",
				Attributes:      &sdp.ItemAttributes{},
				Scope:           scope,
			}, nil
		},
		AdapterMetadata: &sdp.AdapterMetadata{
			Type:            "test-item",
			DescriptiveName: "Test Item",
			SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
				Get:             true,
				List:            true,
				GetDescription:  "Get a test item",
				ListDescription: "List all test items",
			},
		},
	}

	// Query for item1
	_, _ = adapter.Get(ctx, "123456789012.us-east-1", "item1", false)
	_, _ = adapter.Get(ctx, "123456789012.us-east-1", "item1", false) // Cache hit

	// Query for item2
	_, _ = adapter.Get(ctx, "123456789012.us-east-1", "item2", false)
	_, _ = adapter.Get(ctx, "123456789012.us-east-1", "item2", false) // Cache hit

	// Should have called GetFunc once per unique query
	if getFuncCalls != 2 {
		t.Errorf("Expected 2 GetFunc calls (1 per unique query), got %d", getFuncCalls)
	}

	if queriesReceived["item1"] != 1 {
		t.Errorf("Expected 1 call for item1, got %d", queriesReceived["item1"])
	}

	if queriesReceived["item2"] != 1 {
		t.Errorf("Expected 1 call for item2, got %d", queriesReceived["item2"])
	}
}

// TestGetListAdapter_ListItemMapperErrorNoNotFoundCache tests that when ListFunc returns items
// but ItemMapper fails for all of them, we don't incorrectly cache NOTFOUND. Items actually exist
// but couldn't be mapped, so NOTFOUND should not be cached.
func TestGetListAdapter_ListItemMapperErrorNoNotFoundCache(t *testing.T) {
	ctx := context.Background()
	cache := sdpcache.NewMemoryCache()
	listCalls := 0

	type MockAWSItem struct {
		Name string
	}

	adapter := &GetListAdapter[*MockAWSItem, *MockClient, *MockOptions]{
		ItemType:  "test-item",
		cache:     cache,
		AccountID: "123456789012",
		Region:    "us-east-1",
		GetFunc: func(ctx context.Context, client *MockClient, scope string, query string) (*MockAWSItem, error) {
			return nil, errors.New("should not be called in LIST test")
		},
		ListFunc: func(ctx context.Context, client *MockClient, scope string) ([]*MockAWSItem, error) {
			listCalls++
			// Return items that exist
			return []*MockAWSItem{
				{Name: "item1"},
				{Name: "item2"},
			}, nil
		},
		ItemMapper: func(query, scope string, awsItem *MockAWSItem) (*sdp.Item, error) {
			// Simulate mapping failure for all items - this should NOT result in NOTFOUND caching
			return nil, errors.New("mapping failed")
		},
		AdapterMetadata: &sdp.AdapterMetadata{
			Type:            "test-item",
			DescriptiveName: "Test Item",
			SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
				Get:             true,
				List:            true,
				GetDescription:  "Get a test item",
				ListDescription: "List all test items",
			},
		},
	}

	// First call - ItemMapper fails for all items, should NOT cache NOTFOUND
	items1, err1 := adapter.List(ctx, "123456789012.us-east-1", false)

	if len(items1) != 0 {
		t.Errorf("Expected 0 items (all mapping failed), got %d", len(items1))
	}
	if err1 != nil {
		t.Errorf("Expected nil error (errors are silently ignored via continue), got %v", err1)
	}
	if listCalls != 1 {
		t.Errorf("Expected 1 ListFunc call, got %d", listCalls)
	}

	// Second call - should NOT hit cache (NOTFOUND was not cached), should try again
	items2, err2 := adapter.List(ctx, "123456789012.us-east-1", false)

	if listCalls != 2 {
		t.Errorf("Expected 2 ListFunc calls (no cache hit because NOTFOUND was not cached), got %d", listCalls)
	}
	if len(items2) != 0 {
		t.Errorf("Expected 0 items, got %d", len(items2))
	}
	if err2 != nil {
		t.Errorf("Expected nil error, got %v", err2)
	}
}

// TestGetListAdapter_SearchCustomItemMapperErrorNoNotFoundCache tests that when SearchFunc returns items
// but ItemMapper fails for all of them, we don't incorrectly cache NOTFOUND. Items actually exist
// but couldn't be mapped, so NOTFOUND should not be cached.
func TestGetListAdapter_SearchCustomItemMapperErrorNoNotFoundCache(t *testing.T) {
	ctx := context.Background()
	cache := sdpcache.NewMemoryCache()
	searchCalls := 0

	type MockAWSItem struct {
		Name string
	}

	adapter := &GetListAdapter[*MockAWSItem, *MockClient, *MockOptions]{
		ItemType:  "test-item",
		cache:     cache,
		AccountID: "123456789012",
		Region:    "us-east-1",
		GetFunc: func(ctx context.Context, client *MockClient, scope string, query string) (*MockAWSItem, error) {
			return nil, errors.New("should not be called in SEARCH test")
		},
		ListFunc: func(ctx context.Context, client *MockClient, scope string) ([]*MockAWSItem, error) {
			return nil, errors.New("should not be called in SEARCH test")
		},
		SearchFunc: func(ctx context.Context, client *MockClient, scope string, query string) ([]*MockAWSItem, error) {
			searchCalls++
			// Return items that exist
			return []*MockAWSItem{
				{Name: "item1"},
				{Name: "item2"},
			}, nil
		},
		ItemMapper: func(query, scope string, awsItem *MockAWSItem) (*sdp.Item, error) {
			// Simulate mapping failure for all items - this should NOT result in NOTFOUND caching
			return nil, errors.New("mapping failed")
		},
		AdapterMetadata: &sdp.AdapterMetadata{
			Type:            "test-item",
			DescriptiveName: "Test Item",
			SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
				Get:             true,
				List:            true,
				GetDescription:  "Get a test item",
				ListDescription: "List all test items",
			},
		},
	}

	// First call - ItemMapper fails for all items, should NOT cache NOTFOUND
	items1, err1 := adapter.SearchCustom(ctx, "123456789012.us-east-1", "test-query", false)

	if len(items1) != 0 {
		t.Errorf("Expected 0 items (all mapping failed), got %d", len(items1))
	}
	if err1 != nil {
		t.Errorf("Expected nil error (errors are silently ignored via continue), got %v", err1)
	}
	if searchCalls != 1 {
		t.Errorf("Expected 1 SearchFunc call, got %d", searchCalls)
	}

	// Second call - should NOT hit cache (NOTFOUND was not cached), should try again
	items2, err2 := adapter.SearchCustom(ctx, "123456789012.us-east-1", "test-query", false)

	if searchCalls != 2 {
		t.Errorf("Expected 2 SearchFunc calls (no cache hit because NOTFOUND was not cached), got %d", searchCalls)
	}
	if len(items2) != 0 {
		t.Errorf("Expected 0 items, got %d", len(items2))
	}
	if err2 != nil {
		t.Errorf("Expected nil error, got %v", err2)
	}
}
