package sources

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
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
// CancelPendingWork() is called so that concurrent waiters are woken up
// immediately rather than hanging until their context timeout.
//
// This test will FAIL when the bug is present because:
// - Second goroutine will take ~200ms waiting for timeout
// - Test expects second goroutine to complete quickly (<100ms)
//
// This test will PASS after the bug is fixed because:
// - First goroutine calls CancelPendingWork() on error
// - Second goroutine is woken immediately and completes quickly
func TestListErrorCausesCacheHang(t *testing.T) {
	ctx := context.Background()
	cache := sdpcache.NewCache(ctx)
	if boltCache, ok := cache.(*sdpcache.BoltCache); ok {
		defer boltCache.Close()
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
	wg.Add(1)
	go func() {
		defer wg.Done()
		start := time.Now()
		_, firstErr = adapter.(interface {
			List(context.Context, string, bool) ([]*sdp.Item, error)
		}).List(ctx, scope, false)
		firstDuration = time.Since(start)
	}()

	// Give first goroutine time to start and hit the error
	time.Sleep(50 * time.Millisecond)

	// Second goroutine: calls List() after first has hit error
	// Should be woken immediately by CancelPendingWork() and retry quickly
	wg.Add(1)
	go func() {
		defer wg.Done()
		// Use a timeout to prevent infinite hang if bug exists
		ctx2, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
		defer cancel()

		start := time.Now()
		_, secondErr = adapter.(interface {
			List(context.Context, string, bool) ([]*sdp.Item, error)
		}).List(ctx2, scope, false)
		secondDuration = time.Since(start)
	}()

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
	// With the fix: takes <100ms because CancelPendingWork() wakes it immediately
	if secondDuration > 100*time.Millisecond {
		t.Errorf("Second goroutine took too long (%v), indicating pending work was not cancelled. "+
			"Expected <100ms after CancelPendingWork() wakes waiting goroutines.", secondDuration)
		t.Logf("BUG PRESENT: First goroutine returned error without calling CancelPendingWork()")
		t.Logf("  First: completed in %v", firstDuration)
		t.Logf("  Second: hung for %v waiting on pending work timeout", secondDuration)
		t.Logf("  List() called %d times", mockWrapper.callCount.Load())
	}

	// List() is called twice - once by first, once by second after being woken
	callCount := mockWrapper.callCount.Load()
	if callCount != 2 {
		t.Errorf("Expected List to be called twice, was called %d times", callCount)
	}

	t.Logf("Test results:")
	t.Logf("  First goroutine: %v", firstDuration)
	t.Logf("  Second goroutine: %v", secondDuration)
	t.Logf("  List() calls: %d", callCount)
}
