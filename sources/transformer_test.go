package sources

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
	aws "github.com/overmindtech/cli/sources/aws/shared"
	gcp "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

func TestItemTypeReadableFormat(t *testing.T) {
	tests := []struct {
		name     string
		input    shared.ItemType
		expected string
	}{
		{
			name:     "Three parts input",
			input:    shared.NewItemType(gcp.GCP, gcp.Compute, gcp.Instance),
			expected: "GCP Compute Instance",
		},
		{
			name:     "Three parts input",
			input:    shared.NewItemType(aws.AWS, aws.APIGateway, aws.RESTAPI),
			expected: "AWS Api Gateway Rest Api",
			// Note that this is only testing the fallback rendering,
			// adapter implementors will have to supply a custom descriptive name,
			// like "Amazon API Gateway REST API" in the `AdapterMetadata`.
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := tt.input.Readable()
			if actual != tt.expected {
				t.Errorf("readableFormat(%q) = %q; expected %q", tt.input, actual, tt.expected)
			}
		})
	}
}

// errorReturningListableWrapper is a test wrapper that returns an error from List()
// to simulate the scenario where the underlying API call fails.
type errorReturningListableWrapper struct {
	callCount atomic.Int32
	itemType  shared.ItemType
	scope     string
}

func (w *errorReturningListableWrapper) Scopes() []string {
	return []string{w.scope}
}

func (w *errorReturningListableWrapper) GetLookups() ItemTypeLookups {
	return ItemTypeLookups{
		shared.NewItemTypeLookup("id", w.itemType),
	}
}

func (w *errorReturningListableWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	return nil, &sdp.QueryError{
		ErrorType:   sdp.QueryError_NOTFOUND,
		ErrorString: "not implemented",
	}
}

func (w *errorReturningListableWrapper) Type() string {
	return w.itemType.String()
}

func (w *errorReturningListableWrapper) Name() string {
	return "error-returning-adapter"
}

func (w *errorReturningListableWrapper) ItemType() shared.ItemType {
	return w.itemType
}

func (w *errorReturningListableWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return nil
}

func (w *errorReturningListableWrapper) Category() sdp.AdapterCategory {
	return sdp.AdapterCategory_ADAPTER_CATEGORY_COMPUTE_APPLICATION
}

func (w *errorReturningListableWrapper) PotentialLinks() map[shared.ItemType]bool {
	return nil
}

func (w *errorReturningListableWrapper) AdapterMetadata() *sdp.AdapterMetadata {
	return nil
}

func (w *errorReturningListableWrapper) IAMPermissions() []string {
	return nil
}

// List returns an error to trigger the bug where pending work is not canceled
func (w *errorReturningListableWrapper) List(ctx context.Context, scope string) ([]*sdp.Item, *sdp.QueryError) {
	w.callCount.Add(1)
	return nil, &sdp.QueryError{
		ErrorType:   sdp.QueryError_OTHER,
		ErrorString: "simulated list error",
		Scope:       scope,
		SourceName:  w.Name(),
		ItemType:    w.Type(),
	}
}

// TestListErrorCausesCacheHang tests that when List() returns an error,
// done() is called so that concurrent waiters are woken up
// immediately rather than hanging until their context timeout.
//
// This test will FAIL when the bug is present because:
// - Second goroutine will take ~200ms waiting for timeout
// - Test expects second goroutine to complete quickly (<100ms)
//
// This test will PASS after the bug is fixed because:
// - First goroutine calls done() on error
// - Second goroutine is woken immediately and completes quickly
func TestListErrorCausesCacheHang(t *testing.T) {
	ctx := context.Background()
	cache := sdpcache.NewCache(ctx)
	if boltCache, ok := cache.(*sdpcache.BoltCache); ok {
		defer func() { _ = boltCache.CloseAndDestroy() }()
	}

	scope := "test-scope"
	itemType := shared.NewItemType("test", "test", "test")

	mockWrapper := &errorReturningListableWrapper{
		itemType: itemType,
		scope:    scope,
	}

	adapter := WrapperToAdapter(mockWrapper, cache)

	var wg sync.WaitGroup
	var firstErr error
	var secondErr error
	var firstDuration time.Duration
	var secondDuration time.Duration

	// First goroutine: calls List(), gets cache miss, underlying returns error
	wg.Go(func() {
		start := time.Now()
		_, firstErr = adapter.(interface {
			List(context.Context, string, bool) ([]*sdp.Item, error)
		}).List(ctx, scope, false)
		firstDuration = time.Since(start)
	})

	// Give first goroutine time to start and hit the error
	time.Sleep(50 * time.Millisecond)

	// Second goroutine: calls List() after first has hit error
	// Should be woken immediately by done() and retry quickly
	wg.Go(func() {
		// Use a timeout to prevent infinite hang if bug exists
		ctx2, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
		defer cancel()

		start := time.Now()
		_, secondErr = adapter.(interface {
			List(context.Context, string, bool) ([]*sdp.Item, error)
		}).List(ctx2, scope, false)
		secondDuration = time.Since(start)
	})

	wg.Wait()

	// Both goroutines should get errors
	if firstErr == nil {
		t.Fatal("Expected first goroutine to get an error, got nil")
	}

	if secondErr == nil {
		t.Fatal("Expected second goroutine to get an error, got nil")
	}

	// First goroutine should complete quickly (the List error is immediate)
	if firstDuration > 100*time.Millisecond {
		t.Errorf("First goroutine took too long: %v", firstDuration)
	}

	// CRITICAL ASSERTION: Second goroutine should complete quickly
	// With the bug: takes ~200ms+ waiting for timeout
	// With the fix: takes <100ms because done() wakes it immediately
	if secondDuration > 100*time.Millisecond {
		t.Errorf("Second goroutine took too long (%v), indicating pending work was not cancelled. "+
			"Expected <100ms after done() wakes waiting goroutines.", secondDuration)
		t.Logf("BUG PRESENT: First goroutine returned error without calling done()")
		t.Logf("  First: completed in %v", firstDuration)
		t.Logf("  Second: hung for %v waiting on pending work timeout", secondDuration)
		t.Logf("  List() called %d times", mockWrapper.callCount.Load())
	}

	// We only cache NOTFOUND; this wrapper returns QueryError_OTHER so the error is not cached.
	// Both goroutines call List() (callCount == 2). The important assertion is timing above:
	// second goroutine completes quickly because done() wakes it, then it retries and gets the same error.
	callCount := mockWrapper.callCount.Load()
	if callCount != 2 {
		t.Errorf("Expected List to be called twice (error is not cached), was called %d times", callCount)
	}

	t.Logf("Test results:")
	t.Logf("  First goroutine: %v", firstDuration)
	t.Logf("  Second goroutine: %v", secondDuration)
	t.Logf("  List() calls: %d", callCount)
}

// notFoundCachingWrapper returns nil/empty from Get/List/Search to test NOTFOUND caching.
type notFoundCachingWrapper struct {
	getCallCount    atomic.Int32
	listCallCount   atomic.Int32
	searchCallCount atomic.Int32
	itemType        shared.ItemType
	scope           string
}

func (w *notFoundCachingWrapper) Scopes() []string {
	return []string{w.scope}
}

func (w *notFoundCachingWrapper) GetLookups() ItemTypeLookups {
	return ItemTypeLookups{shared.NewItemTypeLookup("id", w.itemType)}
}

func (w *notFoundCachingWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	w.getCallCount.Add(1)
	return nil, nil
}

func (w *notFoundCachingWrapper) Type() string {
	return w.itemType.String()
}

func (w *notFoundCachingWrapper) Name() string {
	return "notfound-caching-adapter"
}

func (w *notFoundCachingWrapper) ItemType() shared.ItemType {
	return w.itemType
}

func (w *notFoundCachingWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return nil
}

func (w *notFoundCachingWrapper) Category() sdp.AdapterCategory {
	return sdp.AdapterCategory_ADAPTER_CATEGORY_COMPUTE_APPLICATION
}

func (w *notFoundCachingWrapper) PotentialLinks() map[shared.ItemType]bool {
	return nil
}

func (w *notFoundCachingWrapper) AdapterMetadata() *sdp.AdapterMetadata {
	return nil
}

func (w *notFoundCachingWrapper) IAMPermissions() []string {
	return nil
}

func (w *notFoundCachingWrapper) List(ctx context.Context, scope string) ([]*sdp.Item, *sdp.QueryError) {
	w.listCallCount.Add(1)
	return []*sdp.Item{}, nil
}

func (w *notFoundCachingWrapper) SearchLookups() []ItemTypeLookups {
	return []ItemTypeLookups{{shared.NewItemTypeLookup("id", w.itemType)}}
}

func (w *notFoundCachingWrapper) Search(ctx context.Context, scope string, queryParts ...string) ([]*sdp.Item, *sdp.QueryError) {
	w.searchCallCount.Add(1)
	return []*sdp.Item{}, nil
}

// TestGetNilCachesNotFound tests that when wrapper Get returns (nil, nil), the adapter
// caches NOTFOUND and a second Get returns the cached error without calling the wrapper again.
func TestGetNilCachesNotFound(t *testing.T) {
	ctx := context.Background()
	cache := sdpcache.NewMemoryCache()
	scope := "test-scope"
	// Use AWS item type so adapter validation does not require GCP predefined role.
	itemType := shared.NewItemType(aws.AWS, aws.APIGateway, aws.RESTAPI)

	wrapper := &notFoundCachingWrapper{itemType: itemType, scope: scope}
	adapter := WrapperToAdapter(wrapper, cache)

	// First Get: miss, wrapper returns (nil, nil), adapter caches NOTFOUND
	item, err := adapter.Get(ctx, scope, "query1", false)
	if item != nil {
		t.Errorf("first Get: expected nil item, got %v", item)
	}
	if err == nil {
		t.Fatal("first Get: expected NOTFOUND error, got nil")
	}
	var qErr *sdp.QueryError
	if !errors.As(err, &qErr) || qErr.GetErrorType() != sdp.QueryError_NOTFOUND {
		t.Errorf("first Get: expected NOTFOUND, got %v", err)
	}
	if wrapper.getCallCount.Load() != 1 {
		t.Errorf("first Get: expected 1 Get call, got %d", wrapper.getCallCount.Load())
	}

	// Second Get: should hit cache, wrapper not called again
	item, err = adapter.Get(ctx, scope, "query1", false)
	if item != nil {
		t.Errorf("second Get: expected nil item, got %v", item)
	}
	if err == nil {
		t.Fatal("second Get: expected NOTFOUND error, got nil")
	}
	if !errors.As(err, &qErr) || qErr.GetErrorType() != sdp.QueryError_NOTFOUND {
		t.Errorf("second Get: expected NOTFOUND, got %v", err)
	}
	if wrapper.getCallCount.Load() != 1 {
		t.Errorf("second Get: expected still 1 Get call (cache hit), got %d", wrapper.getCallCount.Load())
	}
}

// TestListEmptyCachesNotFound tests that when wrapper List returns ([], nil), the adapter
// caches NOTFOUND and a second List returns empty from cache without calling the wrapper again.
func TestListEmptyCachesNotFound(t *testing.T) {
	ctx := context.Background()
	cache := sdpcache.NewMemoryCache()
	scope := "test-scope"
	// Use AWS item type so adapter validation does not require GCP predefined role.
	itemType := shared.NewItemType(aws.AWS, aws.APIGateway, aws.RESTAPI)

	wrapper := &notFoundCachingWrapper{itemType: itemType, scope: scope}
	adapter := WrapperToAdapter(wrapper, cache).(interface {
		List(context.Context, string, bool) ([]*sdp.Item, error)
	})

	// First List: miss, wrapper returns ([], nil), adapter caches NOTFOUND
	items, err := adapter.List(ctx, scope, false)
	if err != nil {
		t.Fatalf("first List: unexpected error %v", err)
	}
	if items == nil {
		t.Error("first List: expected non-nil empty slice, got nil")
	}
	if len(items) != 0 {
		t.Errorf("first List: expected 0 items, got %d", len(items))
	}
	if wrapper.listCallCount.Load() != 1 {
		t.Errorf("first List: expected 1 List call, got %d", wrapper.listCallCount.Load())
	}

	// Second List: should hit cache, wrapper not called again
	items, err = adapter.List(ctx, scope, false)
	if err != nil {
		t.Fatalf("second List: unexpected error %v", err)
	}
	if items == nil {
		t.Error("second List: expected non-nil empty slice, got nil")
	}
	if len(items) != 0 {
		t.Errorf("second List: expected 0 items, got %d", len(items))
	}
	if wrapper.listCallCount.Load() != 1 {
		t.Errorf("second List: expected still 1 List call (cache hit), got %d", wrapper.listCallCount.Load())
	}
}

// TestSearchEmptyCachesNotFound tests that when wrapper Search returns ([], nil), the adapter
// caches NOTFOUND and a second Search returns empty from cache without calling the wrapper again.
func TestSearchEmptyCachesNotFound(t *testing.T) {
	ctx := context.Background()
	cache := sdpcache.NewMemoryCache()
	scope := "test-scope"
	// Use AWS item type so adapter validation does not require GCP predefined role.
	itemType := shared.NewItemType(aws.AWS, aws.APIGateway, aws.RESTAPI)

	wrapper := &notFoundCachingWrapper{itemType: itemType, scope: scope}
	adapter := WrapperToAdapter(wrapper, cache).(interface {
		Search(context.Context, string, string, bool) ([]*sdp.Item, error)
	})

	query := "id1"

	// First Search: miss, wrapper returns ([], nil), adapter caches NOTFOUND
	items, err := adapter.Search(ctx, scope, query, false)
	if err != nil {
		t.Fatalf("first Search: unexpected error %v", err)
	}
	if items == nil {
		t.Error("first Search: expected non-nil empty slice, got nil")
	}
	if len(items) != 0 {
		t.Errorf("first Search: expected 0 items, got %d", len(items))
	}
	if wrapper.searchCallCount.Load() != 1 {
		t.Errorf("first Search: expected 1 Search call, got %d", wrapper.searchCallCount.Load())
	}

	// Second Search: should hit cache, wrapper not called again
	items, err = adapter.Search(ctx, scope, query, false)
	if err != nil {
		t.Fatalf("second Search: unexpected error %v", err)
	}
	if items == nil {
		t.Error("second Search: expected non-nil empty slice, got nil")
	}
	if len(items) != 0 {
		t.Errorf("second Search: expected 0 items, got %d", len(items))
	}
	if wrapper.searchCallCount.Load() != 1 {
		t.Errorf("second Search: expected still 1 Search call (cache hit), got %d", wrapper.searchCallCount.Load())
	}
}

// TestGetNOTFOUNDCacheHitMatchesLiveNOTFOUND asserts response parity: a NOTFOUND cache hit returns
// the same (item, error) as a fresh NOTFOUND — nil item and identical error type and error message.
func TestGetNOTFOUNDCacheHitMatchesLiveNOTFOUND(t *testing.T) {
	ctx := context.Background()
	cache := sdpcache.NewMemoryCache()
	scope := "test-scope"
	itemType := shared.NewItemType(aws.AWS, aws.APIGateway, aws.RESTAPI)
	wrapper := &notFoundCachingWrapper{itemType: itemType, scope: scope}
	adapter := WrapperToAdapter(wrapper, cache)

	query := "query1"
	// Live NOTFOUND
	liveItem, liveErr := adapter.Get(ctx, scope, query, false)
	// Cache NOTFOUND (second call hits cache)
	cacheItem, cacheErr := adapter.Get(ctx, scope, query, false)

	// Same item: both nil
	if liveItem != nil || cacheItem != nil {
		t.Errorf("both responses must have nil item: live=%v cache=%v", liveItem, cacheItem)
	}
	// Same error semantics: both NOTFOUND with same message
	var liveQE, cacheQE *sdp.QueryError
	if !errors.As(liveErr, &liveQE) || !errors.As(cacheErr, &cacheQE) {
		t.Fatalf("both errors must be QueryError: live=%v cache=%v", liveErr, cacheErr)
	}
	if liveQE.GetErrorType() != sdp.QueryError_NOTFOUND || cacheQE.GetErrorType() != sdp.QueryError_NOTFOUND {
		t.Errorf("both must be NOTFOUND: live=%v cache=%v", liveQE.GetErrorType(), cacheQE.GetErrorType())
	}
	if liveQE.GetErrorString() != cacheQE.GetErrorString() {
		t.Errorf("error string must match: live=%q cache=%q", liveQE.GetErrorString(), cacheQE.GetErrorString())
	}
}

// TestListNOTFOUNDCacheHitMatchesLiveNOTFOUND asserts response parity: a NOTFOUND cache hit for List
// returns the same (items, error) as a fresh not-found — empty slice and nil error.
func TestListNOTFOUNDCacheHitMatchesLiveNOTFOUND(t *testing.T) {
	ctx := context.Background()
	cache := sdpcache.NewMemoryCache()
	scope := "test-scope"
	itemType := shared.NewItemType(aws.AWS, aws.APIGateway, aws.RESTAPI)
	wrapper := &notFoundCachingWrapper{itemType: itemType, scope: scope}
	adapter := WrapperToAdapter(wrapper, cache).(interface {
		List(context.Context, string, bool) ([]*sdp.Item, error)
	})

	liveItems, liveErr := adapter.List(ctx, scope, false)
	cacheItems, cacheErr := adapter.List(ctx, scope, false)

	if liveErr != nil || cacheErr != nil {
		t.Errorf("both must return nil error: live=%v cache=%v", liveErr, cacheErr)
	}
	if liveItems == nil || cacheItems == nil {
		t.Errorf("both must return non-nil slice: live=%v cache=%v", liveItems, cacheItems)
	}
	if len(liveItems) != 0 || len(cacheItems) != 0 {
		t.Errorf("both must return empty slice: live len=%d cache len=%d", len(liveItems), len(cacheItems))
	}
}

// TestSearchNOTFOUNDCacheHitMatchesLiveNOTFOUND asserts response parity: a NOTFOUND cache hit for Search
// returns the same (items, error) as a fresh not-found — empty slice and nil error.
func TestSearchNOTFOUNDCacheHitMatchesLiveNOTFOUND(t *testing.T) {
	ctx := context.Background()
	cache := sdpcache.NewMemoryCache()
	scope := "test-scope"
	itemType := shared.NewItemType(aws.AWS, aws.APIGateway, aws.RESTAPI)
	wrapper := &notFoundCachingWrapper{itemType: itemType, scope: scope}
	adapter := WrapperToAdapter(wrapper, cache).(interface {
		Search(context.Context, string, string, bool) ([]*sdp.Item, error)
	})

	query := "id1"
	liveItems, liveErr := adapter.Search(ctx, scope, query, false)
	cacheItems, cacheErr := adapter.Search(ctx, scope, query, false)

	if liveErr != nil || cacheErr != nil {
		t.Errorf("both must return nil error: live=%v cache=%v", liveErr, cacheErr)
	}
	if liveItems == nil || cacheItems == nil {
		t.Errorf("both must return non-nil slice: live=%v cache=%v", liveItems, cacheItems)
	}
	if len(liveItems) != 0 || len(cacheItems) != 0 {
		t.Errorf("both must return empty slice: live len=%d cache len=%d", len(liveItems), len(cacheItems))
	}
}
