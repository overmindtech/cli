package sdpcache

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/overmindtech/cli/go/sdp-go"
)

// TestMemoryCacheStartPurge tests the memory cache implementation's purger.
func TestMemoryCacheStartPurge(t *testing.T) {
	ctx := t.Context()
	cache := NewMemoryCache()
	cache.minWaitTime = 100 * time.Millisecond

	cachedItems := []struct {
		Item   *sdp.Item
		Expiry time.Time
	}{
		{
			Item:   GenerateRandomItem(),
			Expiry: time.Now().Add(0),
		},
		{
			Item:   GenerateRandomItem(),
			Expiry: time.Now().Add(100 * time.Millisecond),
		},
	}

	for _, i := range cachedItems {
		ck := CacheKeyFromQuery(i.Item.GetMetadata().GetSourceQuery(), i.Item.GetMetadata().GetSourceName())
		cache.StoreItem(ctx, i.Item, time.Until(i.Expiry), ck)
	}

	ctx, done := context.WithCancel(ctx)
	defer done()

	cache.StartPurger(ctx)

	// Wait for everything to be purged
	time.Sleep(200 * time.Millisecond)

	// At this point everything should be been cleaned, and the purger should be
	// sleeping forever
	items, err := testSearch(t.Context(), cache, CacheKeyFromQuery(
		cachedItems[1].Item.GetMetadata().GetSourceQuery(),
		cachedItems[1].Item.GetMetadata().GetSourceName(),
	))

	if !errors.Is(err, ErrCacheNotFound) {
		t.Errorf("unexpected error: %v", err)
		t.Errorf("unexpected items: %v", len(items))
	}

	cache.purgeMutex.Lock()
	if cache.nextPurge.Before(time.Now().Add(time.Hour)) {
		// If the next purge is within the next hour that's an error, it should
		// be really, really for in the future
		t.Errorf("Expected next purge to be in 1000 years, got %v", cache.nextPurge.String())
	}
	cache.purgeMutex.Unlock()

	// Adding a new item should kick off the purging again
	for _, i := range cachedItems {
		ck := CacheKeyFromQuery(i.Item.GetMetadata().GetSourceQuery(), i.Item.GetMetadata().GetSourceName())
		cache.StoreItem(ctx, i.Item, 100*time.Millisecond, ck)
	}

	time.Sleep(200 * time.Millisecond)

	// It should be empty again
	items, err = testSearch(t.Context(), cache, CacheKeyFromQuery(
		cachedItems[1].Item.GetMetadata().GetSourceQuery(),
		cachedItems[1].Item.GetMetadata().GetSourceName(),
	))

	if !errors.Is(err, ErrCacheNotFound) {
		t.Errorf("unexpected error: %v", err)
		t.Errorf("unexpected items: %v: %v", len(items), items)
	}
}

// TestMemoryCacheStopPurge tests the memory cache implementation's purger stop functionality.
func TestMemoryCacheStopPurge(t *testing.T) {
	cache := NewMemoryCache()
	cache.minWaitTime = 1 * time.Millisecond

	ctx, done := context.WithCancel(t.Context())

	cache.StartPurger(ctx)

	// Stop the purger
	done()

	// Insert an item
	item := GenerateRandomItem()
	ck := CacheKeyFromQuery(item.GetMetadata().GetSourceQuery(), item.GetMetadata().GetSourceName())

	cache.StoreItem(ctx, item, 1*time.Second, ck)
	sst := SST{
		SourceName: item.GetMetadata().GetSourceName(),
		Scope:      item.GetScope(),
		Type:       item.GetType(),
	}

	// Make sure it's not purged
	time.Sleep(100 * time.Millisecond)
	items, err := testSearch(t.Context(), cache, CacheKey{
		SST: sst,
	})
	if err != nil {
		t.Error(err)
	}

	if len(items) != 1 {
		t.Errorf("Expected 1 item, got %v", len(items))
	}
}

// TestMemoryCacheConcurrent tests the memory cache implementation for data races.
// This test is designed to be run with -race to ensure that there aren't any
// data races.
func TestMemoryCacheConcurrent(t *testing.T) {
	cache := NewMemoryCache()
	// Run the purger super fast to generate a worst-case scenario
	cache.minWaitTime = 1 * time.Millisecond

	ctx, done := context.WithCancel(t.Context())
	defer done()
	cache.StartPurger(ctx)
	var wg sync.WaitGroup

	numParallel := 1_000

	for range numParallel {
		wg.Go(func() {
			// Store the item
			item := GenerateRandomItem()
			ck := CacheKeyFromQuery(item.GetMetadata().GetSourceQuery(), item.GetMetadata().GetSourceName())

			cache.StoreItem(ctx, item, 100*time.Millisecond, ck)

			// Create a goroutine to also delete in parallel
			wg.Go(func() {
				cache.Delete(ck)
			})
		})
	}

	wg.Wait()
}

// TestMemoryCacheLookupDeduplication tests that multiple concurrent Lookup calls
// for the same cache key in MemoryCache result in only one caller doing work.
func TestMemoryCacheLookupDeduplication(t *testing.T) {
	cache := NewMemoryCache()
	ctx := t.Context()

	// Create a cache key for the test - use LIST method to avoid UniqueAttributeValue matching issues
	sst := SST{SourceName: "test-source", Scope: "test-scope", Type: "test-type"}
	method := sdp.QueryMethod_LIST

	// Track how many goroutines actually do work
	var workCount int32
	var mu sync.Mutex
	var wg sync.WaitGroup

	numGoroutines := 10
	results := make([]struct {
		hit   bool
		items []*sdp.Item
	}, numGoroutines)

	startBarrier := make(chan struct{})

	for idx := range numGoroutines {
		wg.Go(func() {
			<-startBarrier

			hit, ck, items, _, done := cache.Lookup(ctx, sst.SourceName, method, sst.Scope, sst.Type, "", false)
			defer done()

			if !hit {
				mu.Lock()
				workCount++
				mu.Unlock()

				time.Sleep(50 * time.Millisecond)

				item := GenerateRandomItem()
				item.Scope = sst.Scope
				item.Type = sst.Type
				item.Metadata.SourceName = sst.SourceName

				cache.StoreItem(ctx, item, 10*time.Second, ck)
				hit, _, items, _, done = cache.Lookup(ctx, sst.SourceName, method, sst.Scope, sst.Type, "", false)
				defer done()
			}

			results[idx] = struct {
				hit   bool
				items []*sdp.Item
			}{hit, items}
		})
	}

	close(startBarrier)
	wg.Wait()

	if workCount != 1 {
		t.Errorf("expected exactly 1 goroutine to do work, got %d", workCount)
	}

	for i, r := range results {
		if !r.hit {
			t.Errorf("goroutine %d: expected cache hit after dedup, got miss", i)
		}
		if len(r.items) != 1 {
			t.Errorf("goroutine %d: expected 1 item, got %d", i, len(r.items))
		}
	}
}

// TestMemoryCacheLookupDeduplicationCompleteWithoutStore tests the scenario where
// Complete is called but nothing was stored in the cache. This tests the explicit
// ErrCacheNotFound check in the re-check logic.
func TestMemoryCacheLookupDeduplicationCompleteWithoutStore(t *testing.T) {
	cache := NewMemoryCache()
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

		// Complete without storing anything - triggers ErrCacheNotFound on re-check
		cache.pending.Complete(ck.String())
	})

	// Waiter goroutines
	for range numWaiters {
		wg.Go(func() {
			<-startBarrier

			time.Sleep(10 * time.Millisecond)

			hit, _, _, _, done := cache.Lookup(ctx, sst.SourceName, method, sst.Scope, sst.Type, query, false)
			defer done()

			waiterMu.Lock()
			waiterHits = append(waiterHits, hit)
			waiterMu.Unlock()
		})
	}

	close(startBarrier)
	wg.Wait()

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
