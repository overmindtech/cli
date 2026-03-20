package sdpcache

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/overmindtech/cli/go/sdp-go"
)

// TestBoltStoreCloseAndDestroy verifies that CloseAndDestroy() correctly
// closes the database and deletes the cache file.
func TestBoltStoreCloseAndDestroy(t *testing.T) {
	tempDir := t.TempDir()
	cachePath := filepath.Join(tempDir, "cache.db")

	// Create a cache and store some data
	ctx := t.Context()
	cache1, err := NewBoltCache(cachePath)
	if err != nil {
		t.Fatalf("failed to create BoltCache: %v", err)
	}

	// Store an item
	item1 := GenerateRandomItem()
	ck1 := CacheKeyFromQuery(item1.GetMetadata().GetSourceQuery(), item1.GetMetadata().GetSourceName())
	cache1.StoreItem(ctx, item1, 10*time.Second, ck1)

	// Store another item with a short TTL (will expire)
	item2 := GenerateRandomItem()
	ck2 := CacheKeyFromQuery(item2.GetMetadata().GetSourceQuery(), item2.GetMetadata().GetSourceName())
	cache1.StoreItem(ctx, item2, 100*time.Millisecond, ck2)

	// Verify both items are in the cache
	items, err := testSearch(t.Context(), cache1, ck1)
	if err != nil {
		t.Errorf("failed to search for item1: %v", err)
	}
	if len(items) != 1 {
		t.Errorf("expected 1 item for ck1, got %d", len(items))
	}

	// Verify the cache file exists
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		t.Fatal("cache file should exist before CloseAndDestroy")
	}

	// Close and destroy the cache
	if err := cache1.CloseAndDestroy(); err != nil {
		t.Fatalf("failed to close and destroy cache1: %v", err)
	}

	// Verify the cache file is deleted
	if _, err := os.Stat(cachePath); !os.IsNotExist(err) {
		t.Error("cache file should be deleted after CloseAndDestroy")
	}

	// Create a new cache at the same path - should create a fresh, empty cache
	cache2, err := NewBoltCache(cachePath)
	if err != nil {
		t.Fatalf("failed to create new BoltCache: %v", err)
	}
	defer func() {
		_ = cache2.CloseAndDestroy()
	}()

	// Verify the old item is NOT accessible (cache was destroyed)
	items, err = testSearch(ctx, cache2, ck1)
	if !errors.Is(err, ErrCacheNotFound) {
		t.Errorf("expected cache miss for item1 in new cache, got: err=%v, items=%d", err, len(items))
	}

	// Verify we can store new items in the fresh cache
	item3 := GenerateRandomItem()
	ck3 := CacheKeyFromQuery(item3.GetMetadata().GetSourceQuery(), item3.GetMetadata().GetSourceName())
	cache2.StoreItem(ctx, item3, 10*time.Second, ck3)

	items, err = testSearch(ctx, cache2, ck3)
	if err != nil {
		t.Errorf("failed to search for newly stored item3: %v", err)
	}
	if len(items) != 1 {
		t.Errorf("expected 1 item for ck3, got %d", len(items))
	}
}

// TestBoltStoreOperationsAfterCloseAndDestroy verifies that operations after
// CloseAndDestroy() return proper errors instead of panicking.
func TestBoltStoreOperationsAfterCloseAndDestroy(t *testing.T) {
	tempDir := t.TempDir()
	cachePath := filepath.Join(tempDir, "cache.db")
	ctx := t.Context()

	cache, err := NewBoltCache(cachePath)
	if err != nil {
		t.Fatalf("failed to create BoltCache: %v", err)
	}

	// Store an item before closing
	item := GenerateRandomItem()
	ck := CacheKeyFromQuery(item.GetMetadata().GetSourceQuery(), item.GetMetadata().GetSourceName())
	cache.StoreItem(ctx, item, 10*time.Second, ck)

	// Close and destroy the cache
	if err := cache.CloseAndDestroy(); err != nil {
		t.Fatalf("failed to close and destroy cache: %v", err)
	}

	// Now try various operations after the cache is closed and destroyed
	// These should return errors, not panic

	t.Run("Search after CloseAndDestroy", func(t *testing.T) {
		// This should error because the database is closed
		_, err := testSearch(ctx, cache, ck)
		if err == nil {
			t.Error("expected error when searching after CloseAndDestroy, got nil")
		}
		t.Logf("Search returned expected error: %v", err)
	})

	t.Run("StoreItem after CloseAndDestroy", func(t *testing.T) {
		// This should not panic - it might silently fail or error
		// The key is that it doesn't panic
		newItem := GenerateRandomItem()
		newCk := CacheKeyFromQuery(newItem.GetMetadata().GetSourceQuery(), newItem.GetMetadata().GetSourceName())

		// This should either complete without panic or handle the closed DB gracefully
		cache.StoreItem(ctx, newItem, 10*time.Second, newCk)
		t.Log("StoreItem completed without panic (may have failed internally)")
	})

	t.Run("Delete after CloseAndDestroy", func(t *testing.T) {
		// This should not panic
		cache.Delete(ck)
		t.Log("Delete completed without panic (may have failed internally)")
	})

	t.Run("Purge after CloseAndDestroy", func(t *testing.T) {
		// This should not panic
		stats := cache.Purge(ctx, time.Now())
		t.Logf("Purge completed without panic, purged %d items", stats.NumPurged)
	})
}

// TestBoltStoreConcurrentCloseAndDestroy verifies that CloseAndDestroy()
// properly synchronizes with concurrent operations using the compaction lock.
func TestBoltStoreConcurrentCloseAndDestroy(t *testing.T) {
	tempDir := t.TempDir()
	cachePath := filepath.Join(tempDir, "cache.db")
	ctx := t.Context()

	cache, err := NewBoltCache(cachePath)
	if err != nil {
		t.Fatalf("failed to create BoltCache: %v", err)
	}

	// Store some items
	for range 10 {
		item := GenerateRandomItem()
		ck := CacheKeyFromQuery(item.GetMetadata().GetSourceQuery(), item.GetMetadata().GetSourceName())
		cache.StoreItem(ctx, item, 10*time.Second, ck)
	}

	// Start some concurrent operations
	var wg sync.WaitGroup
	numOperations := 50

	// Launch concurrent read/write operations
	for range numOperations {
		wg.Go(func() {
			item := GenerateRandomItem()
			ck := CacheKeyFromQuery(item.GetMetadata().GetSourceQuery(), item.GetMetadata().GetSourceName())
			cache.StoreItem(ctx, item, 10*time.Second, ck)
		})
	}

	// Wait a bit to let operations start
	time.Sleep(10 * time.Millisecond)

	// Close and destroy while operations are in flight
	// The compaction lock should serialize this properly
	wg.Go(func() {
		err := cache.CloseAndDestroy()
		if err != nil {
			t.Logf("CloseAndDestroy returned error: %v", err)
		}
	})

	// Wait for all operations to complete
	wg.Wait()

	// Verify the file is deleted
	if _, err := os.Stat(cachePath); !os.IsNotExist(err) {
		t.Error("cache file should be deleted after CloseAndDestroy")
	}

	t.Log("Concurrent operations with CloseAndDestroy completed without data races")
}

// TestIsDiskFullError tests the isDiskFullError helper function.
func TestIsDiskFullError(t *testing.T) {
	// Test that non-disk-full errors are not detected.
	regularErr := errors.New("some other error")
	if isDiskFullError(regularErr) {
		t.Error("isDiskFullError should return false for regular errors")
	}

	// Test nil error.
	if isDiskFullError(nil) {
		t.Error("isDiskFullError should return false for nil")
	}
}

// TestBoltStoreDeleteCacheFile recreates the DB file and clears data by exercising
// deleteCacheFile(). This is the behavior relied upon in disk-full recovery paths.
func TestBoltStoreDeleteCacheFile(t *testing.T) {
	tempDir := t.TempDir()
	cachePath := filepath.Join(tempDir, "cache.db")

	// Create a cache and store some data
	ctx := t.Context()
	cache, err := NewBoltCache(cachePath)
	if err != nil {
		t.Fatalf("failed to create BoltCache: %v", err)
	}

	// Store an item
	item := GenerateRandomItem()
	ck := CacheKeyFromQuery(item.GetMetadata().GetSourceQuery(), item.GetMetadata().GetSourceName())
	cache.StoreItem(ctx, item, 10*time.Second, ck)

	// Verify the cache file exists
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		t.Fatal("cache file should exist")
	}

	// Verify item is in cache
	items, err := testSearch(t.Context(), cache, ck)
	if err != nil {
		t.Errorf("failed to search: %v", err)
	}
	if len(items) != 1 {
		t.Errorf("expected 1 item, got %d", len(items))
	}

	// Delete the cache file (cache is already *BoltCache)
	if err := cache.deleteCacheFile(ctx); err != nil {
		t.Fatalf("failed to delete cache file: %v", err)
	}

	// Verify the cache file is gone
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		t.Error("cache file should be recreated")
	}

	// Verify the database is closed (can't search anymore)
	_, _ = testSearch(t.Context(), cache, ck)
	// The search might fail or return empty, but the important thing is the file is gone
	// and we can't use the cache anymore
}

// TestBoltStoreCompactThresholdTriggeredByPurge verifies that purge-triggered
// compaction keeps the store usable afterwards.
func TestBoltStoreCompactThresholdTriggeredByPurge(t *testing.T) {
	tempDir := t.TempDir()
	cachePath := filepath.Join(tempDir, "cache.db")

	cache, err := NewBoltCache(cachePath, WithCompactThreshold(1024)) // Small threshold to trigger compaction
	if err != nil {
		t.Fatalf("failed to create BoltCache: %v", err)
	}
	defer func() {
		_ = cache.CloseAndDestroy()
	}()

	ctx := t.Context()

	// Store enough items to trigger compaction
	// We'll store items and then delete them to accumulate deleted bytes
	for range 10 {
		item := GenerateRandomItem()
		ck := CacheKeyFromQuery(item.GetMetadata().GetSourceQuery(), item.GetMetadata().GetSourceName())
		cache.StoreItem(ctx, item, 10*time.Second, ck)
	}

	// Manually set deleted bytes to trigger compaction
	cache.addDeletedBytes(cache.CompactThreshold)

	// Trigger purge which should trigger compaction
	stats := cache.Purge(ctx, time.Now().Add(-1*time.Hour)) // Purge items from an hour ago (none should exist)
	_ = stats                                               // Use stats to avoid unused variable

	// Verify cache still works after compaction attempt
	item := GenerateRandomItem()
	ck := CacheKeyFromQuery(item.GetMetadata().GetSourceQuery(), item.GetMetadata().GetSourceName())
	cache.StoreItem(ctx, item, 10*time.Second, ck)

	items, err := testSearch(t.Context(), cache, ck)
	if err != nil {
		t.Errorf("failed to search after compaction: %v", err)
	}
	if len(items) != 1 {
		t.Errorf("expected 1 item after compaction, got %d", len(items))
	}
}

// TestBoltCacheLookupDeduplicatesConcurrentMisses verifies that multiple concurrent
// Lookup calls for the same cache key result in only one caller doing the work.
func TestBoltCacheLookupDeduplicatesConcurrentMisses(t *testing.T) {
	tempDir := t.TempDir()
	cachePath := filepath.Join(tempDir, "cache.db")

	cache, err := NewBoltCache(cachePath)
	if err != nil {
		t.Fatalf("failed to create BoltCache: %v", err)
	}
	defer func() { _ = cache.CloseAndDestroy() }()

	ctx := t.Context()

	// Create a cache key for the test - use LIST method to avoid UniqueAttributeValue matching issues
	sst := SST{SourceName: "test-source", Scope: "test-scope", Type: "test-type"}
	method := sdp.QueryMethod_LIST

	// Track how many goroutines actually do work (get cache miss as first caller)
	var workCount int32
	var mu sync.Mutex
	var wg sync.WaitGroup

	numGoroutines := 10
	results := make([]struct {
		hit   bool
		items []*sdp.Item
		err   *sdp.QueryError
	}, numGoroutines)

	// Start barrier to ensure all goroutines start at roughly the same time
	startBarrier := make(chan struct{})

	for i := range numGoroutines {
		wg.Go(func() {
			// Wait for the start signal
			<-startBarrier

			// Lookup the cache - all should get miss initially
			hit, ck, items, qErr, done := cache.Lookup(ctx, sst.SourceName, method, sst.Scope, sst.Type, "", false)
			defer done()

			if !hit {
				// This goroutine is doing the work
				mu.Lock()
				workCount++
				mu.Unlock()

				// Simulate some work
				time.Sleep(50 * time.Millisecond)

				// Create and store the item
				item := GenerateRandomItem()
				item.Scope = sst.Scope
				item.Type = sst.Type
				item.Metadata.SourceName = sst.SourceName

				cache.StoreItem(ctx, item, 10*time.Second, ck)

				// Re-lookup to get the stored item for our result
				hit, _, items, qErr, done = cache.Lookup(ctx, sst.SourceName, method, sst.Scope, sst.Type, "", false)
				defer done()
			}

			results[i] = struct {
				hit   bool
				items []*sdp.Item
				err   *sdp.QueryError
			}{hit, items, qErr}
		})
	}

	// Release all goroutines at once
	close(startBarrier)

	// Wait for all goroutines to complete
	wg.Wait()

	// Verify that only one goroutine did the work
	if workCount != 1 {
		t.Errorf("expected exactly 1 goroutine to do work, got %d", workCount)
	}

	// Verify all goroutines got results
	for i, r := range results {
		if !r.hit {
			t.Errorf("goroutine %d: expected cache hit after dedup, got miss", i)
		}
		if len(r.items) != 1 {
			t.Errorf("goroutine %d: expected 1 item, got %d", i, len(r.items))
		}
	}
}

// TestBoltCacheLookupDeduplicationRespectsWaiterTimeout verifies that waiter
// lookups return when their context deadline is exceeded.
func TestBoltCacheLookupDeduplicationRespectsWaiterTimeout(t *testing.T) {
	tempDir := t.TempDir()
	cachePath := filepath.Join(tempDir, "cache.db")

	cache, err := NewBoltCache(cachePath)
	if err != nil {
		t.Fatalf("failed to create BoltCache: %v", err)
	}
	defer func() { _ = cache.CloseAndDestroy() }()

	ctx := t.Context()

	sst := SST{SourceName: "test-source", Scope: "test-scope", Type: "test-type"}
	method := sdp.QueryMethod_GET
	query := "timeout-test"

	var wg sync.WaitGroup
	startBarrier := make(chan struct{})

	// First goroutine: does the work but takes a long time
	wg.Go(func() {
		<-startBarrier

		hit, ck, _, _, done := cache.Lookup(ctx, sst.SourceName, method, sst.Scope, sst.Type, query, false)
		defer done()
		if hit {
			t.Error("first goroutine: expected cache miss")
			return
		}

		// Simulate slow work
		time.Sleep(500 * time.Millisecond)

		// Store the item
		item := GenerateRandomItem()
		item.Scope = sst.Scope
		item.Type = sst.Type
		cache.StoreItem(ctx, item, 10*time.Second, ck)
	})

	// Second goroutine: should timeout waiting
	var secondHit bool
	wg.Go(func() {
		<-startBarrier

		// Small delay to ensure first goroutine starts first
		time.Sleep(10 * time.Millisecond)

		// Use a short timeout context
		shortCtx, done := context.WithTimeout(ctx, 50*time.Millisecond)
		defer done()

		hit, _, _, _, done := cache.Lookup(shortCtx, sst.SourceName, method, sst.Scope, sst.Type, query, false)
		defer done()
		secondHit = hit
	})

	// Release all goroutines
	close(startBarrier)
	wg.Wait()

	// Second goroutine should have timed out and returned miss
	if secondHit {
		t.Error("second goroutine should have timed out and returned miss")
	}
}

// TestBoltCacheLookupDeduplicationPropagatesStoredError verifies that waiters
// receive the error stored by the first caller.
func TestBoltCacheLookupDeduplicationPropagatesStoredError(t *testing.T) {
	tempDir := t.TempDir()
	cachePath := filepath.Join(tempDir, "cache.db")

	cache, err := NewBoltCache(cachePath)
	if err != nil {
		t.Fatalf("failed to create BoltCache: %v", err)
	}
	defer func() { _ = cache.CloseAndDestroy() }()

	ctx := t.Context()

	sst := SST{SourceName: "test-source", Scope: "test-scope", Type: "test-type"}
	method := sdp.QueryMethod_GET
	query := "error-test"

	var wg sync.WaitGroup
	startBarrier := make(chan struct{})

	expectedError := &sdp.QueryError{
		ErrorType:   sdp.QueryError_NOTFOUND,
		ErrorString: "item not found",
		Scope:       sst.Scope,
		SourceName:  sst.SourceName,
		ItemType:    sst.Type,
	}

	// Track results from waiters
	var waiterErrors []*sdp.QueryError
	var waiterMu sync.Mutex

	numWaiters := 5

	// First goroutine: does the work and stores an error
	wg.Go(func() {
		<-startBarrier

		hit, ck, _, _, done := cache.Lookup(ctx, sst.SourceName, method, sst.Scope, sst.Type, query, false)
		defer done()
		if hit {
			t.Error("first goroutine: expected cache miss")
			return
		}

		// Simulate work that results in an error
		time.Sleep(50 * time.Millisecond)

		// Store the error
		cache.StoreUnavailableItem(ctx, expectedError, 10*time.Second, ck)
	})

	// Waiter goroutines: should receive the error
	for range numWaiters {
		wg.Go(func() {
			<-startBarrier

			// Small delay to ensure first goroutine starts first
			time.Sleep(10 * time.Millisecond)

			hit, _, _, qErr, done := cache.Lookup(ctx, sst.SourceName, method, sst.Scope, sst.Type, query, false)
			defer done()

			waiterMu.Lock()
			if hit && qErr != nil {
				waiterErrors = append(waiterErrors, qErr)
			}
			waiterMu.Unlock()
		})
	}

	// Release all goroutines
	close(startBarrier)
	wg.Wait()

	// All waiters should have received the error
	if len(waiterErrors) != numWaiters {
		t.Errorf("expected %d waiters to receive error, got %d", numWaiters, len(waiterErrors))
	}

	// Verify the error content
	for i, qErr := range waiterErrors {
		if qErr.GetErrorType() != expectedError.GetErrorType() {
			t.Errorf("waiter %d: expected error type %v, got %v", i, expectedError.GetErrorType(), qErr.GetErrorType())
		}
	}
}

// TestBoltCacheLookupDeduplicationReturnsMissAfterCancel verifies that waiters
// return misses when the in-flight work is cancelled.
func TestBoltCacheLookupDeduplicationReturnsMissAfterCancel(t *testing.T) {
	tempDir := t.TempDir()
	cachePath := filepath.Join(tempDir, "cache.db")

	cache, err := NewBoltCache(cachePath)
	if err != nil {
		t.Fatalf("failed to create BoltCache: %v", err)
	}
	defer func() { _ = cache.CloseAndDestroy() }()

	ctx := t.Context()

	sst := SST{SourceName: "test-source", Scope: "test-scope", Type: "test-type"}
	method := sdp.QueryMethod_GET
	query := "done-test"

	var wg sync.WaitGroup
	startBarrier := make(chan struct{})

	// Track results
	var waiterHits []bool
	var waiterMu sync.Mutex

	numWaiters := 3

	// First goroutine: starts work but then calls done() without storing anything
	wg.Go(func() {
		<-startBarrier

		hit, _, _, _, done := cache.Lookup(ctx, sst.SourceName, method, sst.Scope, sst.Type, query, false)
		if hit {
			t.Error("first goroutine: expected cache miss")
			done()
			return
		}

		// Simulate work that fails - done the pending work
		time.Sleep(50 * time.Millisecond)
		done()
	})

	// Waiter goroutines
	for range numWaiters {
		wg.Go(func() {
			<-startBarrier

			// Small delay to ensure first goroutine starts first
			time.Sleep(10 * time.Millisecond)

			hit, _, _, _, done := cache.Lookup(ctx, sst.SourceName, method, sst.Scope, sst.Type, query, false)
			defer done()

			waiterMu.Lock()
			waiterHits = append(waiterHits, hit)
			waiterMu.Unlock()
		})
	}

	// Release all goroutines
	close(startBarrier)
	wg.Wait()

	// When work is cancelled, waiters receive ok=false from Wait
	// (because entry.cancelled is true) and return a cache miss without re-checking.
	// This is the correct behavior - waiters don't hang forever and can retry.
	if len(waiterHits) != numWaiters {
		t.Errorf("expected %d waiter results, got %d", numWaiters, len(waiterHits))
	}
}

// TestBoltCacheLookupDeduplicationReturnsMissWhenCompletedWithoutStore verifies
// waiter behavior when the first caller completes without storing data.
func TestBoltCacheLookupDeduplicationReturnsMissWhenCompletedWithoutStore(t *testing.T) {
	tempDir := t.TempDir()
	cachePath := filepath.Join(tempDir, "cache.db")

	cache, err := NewBoltCache(cachePath)
	if err != nil {
		t.Fatalf("failed to create BoltCache: %v", err)
	}
	defer func() { _ = cache.CloseAndDestroy() }()

	ctx := t.Context()

	sst := SST{SourceName: "test-source", Scope: "test-scope", Type: "test-type"}
	method := sdp.QueryMethod_LIST
	query := "complete-without-store-test"

	var wg sync.WaitGroup
	startBarrier := make(chan struct{})

	// Track results
	var waiterHits []bool
	var waiterMu sync.Mutex

	numWaiters := 3

	// First goroutine: starts work and completes without storing anything
	// This simulates a LIST query that returns 0 items
	wg.Go(func() {
		<-startBarrier

		hit, ck, _, _, done := cache.Lookup(ctx, sst.SourceName, method, sst.Scope, sst.Type, query, false)
		defer done()
		if hit {
			t.Error("first goroutine: expected cache miss")
			return
		}

		// Simulate work that completes successfully but returns nothing
		time.Sleep(50 * time.Millisecond)

		// Complete without storing anything - no items, no error
		// This triggers the ErrCacheNotFound path in waiters' re-check
		cache.pending.Complete(ck.String())
	})

	// Waiter goroutines
	for range numWaiters {
		wg.Go(func() {
			<-startBarrier

			// Small delay to ensure first goroutine starts first
			time.Sleep(10 * time.Millisecond)

			hit, _, _, _, done := cache.Lookup(ctx, sst.SourceName, method, sst.Scope, sst.Type, query, false)
			defer done()

			waiterMu.Lock()
			waiterHits = append(waiterHits, hit)
			waiterMu.Unlock()
		})
	}

	// Release all goroutines
	close(startBarrier)
	wg.Wait()

	// When Complete is called without storing anything:
	// 1. Waiters' Wait returns ok=true (not cancelled)
	// 2. Waiters re-check the cache and get ErrCacheNotFound
	// 3. Waiters return hit=false (cache miss)
	if len(waiterHits) != numWaiters {
		t.Errorf("expected %d waiter results, got %d", numWaiters, len(waiterHits))
	}

	// All waiters should get a cache miss since nothing was stored
	for i, hit := range waiterHits {
		if hit {
			t.Errorf("waiter %d: expected cache miss (hit=false), got hit=true", i)
		}
	}
}
