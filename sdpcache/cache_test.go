package sdpcache

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/overmindtech/cli/sdp-go"
)

func TestStoreItem(t *testing.T) {
	cache := NewCache()

	t.Run("one match", func(t *testing.T) {
		item := GenerateRandomItem()
		ck := CacheKeyFromQuery(item.GetMetadata().GetSourceQuery(), item.GetMetadata().GetSourceName())
		cache.StoreItem(item, 10*time.Second, ck)

		results, err := cache.Search(ck)
		if err != nil {
			t.Error(err)
		}

		if len(results) != 1 {
			t.Errorf("expected 1 result, got %v", len(results))
		}
	})

	t.Run("another match", func(t *testing.T) {
		item := GenerateRandomItem()
		ck := CacheKeyFromQuery(item.GetMetadata().GetSourceQuery(), item.GetMetadata().GetSourceName())

		cache.StoreItem(item, 10*time.Second, ck)

		results, err := cache.Search(ck)
		if err != nil {
			t.Error(err)
		}

		if len(results) != 1 {
			t.Errorf("expected 1 result, got %v", len(results))
		}
	})

	t.Run("different scope", func(t *testing.T) {
		item := GenerateRandomItem()
		ck := CacheKeyFromQuery(item.GetMetadata().GetSourceQuery(), item.GetMetadata().GetSourceName())

		cache.StoreItem(item, 10*time.Second, ck)

		ck.SST.Scope = fmt.Sprintf("new scope %v", ck.SST.Scope)

		results, err := cache.Search(ck)
		if err != nil {
			if !errors.Is(err, ErrCacheNotFound) {
				t.Error(err)
			} else {
				t.Log("expected cache miss")
			}
		}

		if len(results) != 0 {
			t.Errorf("expected 0 result, got %v", results)
		}
	})
}

func TestStoreError(t *testing.T) {
	cache := NewCache()

	t.Run("with just an error", func(t *testing.T) {
		sst := SST{
			SourceName: "foo",
			Scope:      "foo",
			Type:       "foo",
		}

		uav := "foo"

		cache.StoreError(errors.New("arse"), 10*time.Second, CacheKey{
			SST:    sst,
			Method: sdp.QueryMethod_GET.Enum(),
			Query:  &uav,
		})

		items, err := cache.Search(CacheKey{
			SST:    sst,
			Method: sdp.QueryMethod_GET.Enum(),
			Query:  &uav,
		})

		if len(items) > 0 {
			t.Errorf("expected 0 items, got %v", len(items))
		}

		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("with items and an error for the same query", func(t *testing.T) {
		// Add an item with the same details as above
		item := GenerateRandomItem()
		item.Metadata.SourceQuery.Method = sdp.QueryMethod_GET
		item.Metadata.SourceQuery.Query = "foo"
		item.Metadata.SourceName = "foo"
		item.Scope = "foo"
		item.Type = "foo"

		ck := CacheKeyFromQuery(item.GetMetadata().GetSourceQuery(), item.GetMetadata().GetSourceName())

		items, err := cache.Search(ck)

		if len(items) > 0 {
			t.Errorf("expected 0 items, got %v", len(items))
		}

		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("with multiple errors", func(t *testing.T) {
		sst := SST{
			SourceName: "foo",
			Scope:      "foo",
			Type:       "foo",
		}

		uav := "foo"

		cache.StoreError(errors.New("nope"), 10*time.Second, CacheKey{
			SST:    sst,
			Method: sdp.QueryMethod_GET.Enum(),
			Query:  &uav,
		})

		items, err := cache.Search(CacheKey{
			SST:    sst,
			Method: sdp.QueryMethod_GET.Enum(),
			Query:  &uav,
		})

		if len(items) > 0 {
			t.Errorf("expected 0 items, got %v", len(items))
		}

		if err == nil {
			t.Error("expected error, got nil")
		}
	})
}

func TestPurge(t *testing.T) {
	cache := NewCache()

	cachedItems := []struct {
		Item   *sdp.Item
		Expiry time.Time
	}{
		{
			Item:   GenerateRandomItem(),
			Expiry: time.Now().Add(0 * time.Second),
		},
		{
			Item:   GenerateRandomItem(),
			Expiry: time.Now().Add(1 * time.Second),
		},
		{
			Item:   GenerateRandomItem(),
			Expiry: time.Now().Add(2 * time.Second),
		},
		{
			Item:   GenerateRandomItem(),
			Expiry: time.Now().Add(3 * time.Second),
		},
		{
			Item:   GenerateRandomItem(),
			Expiry: time.Now().Add(4 * time.Second),
		},
		{
			Item:   GenerateRandomItem(),
			Expiry: time.Now().Add(5 * time.Second),
		},
	}

	for _, i := range cachedItems {
		ck := CacheKeyFromQuery(i.Item.GetMetadata().GetSourceQuery(), i.Item.GetMetadata().GetSourceName())
		cache.StoreItem(i.Item, time.Until(i.Expiry), ck)
	}

	// Make sure all the items are in the cache
	for _, i := range cachedItems {
		ck := CacheKeyFromQuery(i.Item.GetMetadata().GetSourceQuery(), i.Item.GetMetadata().GetSourceName())
		items, err := cache.Search(ck)
		if err != nil {
			t.Error(err)
		}

		if len(items) != 1 {
			t.Errorf("expected 1 item, got %v", len(items))
		}
	}

	// Purge just the first one
	stats := cache.Purge(cachedItems[0].Expiry.Add(500 * time.Millisecond))

	if stats.NumPurged != 1 {
		t.Errorf("expected 1 item purged, got %v", stats.NumPurged)
	}

	// The times won't be exactly equal because we're checking it against
	// time.Now more than once. So I need to check that they are *almost* the
	// same, but not exactly
	nextExpiryString := stats.NextExpiry.Format(time.RFC3339)
	expectedNextExpiryString := cachedItems[1].Expiry.Format(time.RFC3339)

	if nextExpiryString != expectedNextExpiryString {
		t.Errorf("expected next expiry to be %v, got %v", expectedNextExpiryString, nextExpiryString)
	}

	// Purge all but the last one
	stats = cache.Purge(cachedItems[4].Expiry.Add(500 * time.Millisecond))

	if stats.NumPurged != 4 {
		t.Errorf("expected 4 item purged, got %v", stats.NumPurged)
	}

	// Purge the last one
	stats = cache.Purge(cachedItems[5].Expiry.Add(500 * time.Millisecond))

	if stats.NumPurged != 1 {
		t.Errorf("expected 1 item purged, got %v", stats.NumPurged)
	}

	if stats.NextExpiry != nil {
		t.Errorf("expected expiry to be nil, got %v", stats.NextExpiry)
	}
}

func TestStartPurge(t *testing.T) {
	cache := NewCache()
	cache.MinWaitTime = 100 * time.Millisecond

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
		cache.StoreItem(i.Item, time.Until(i.Expiry), ck)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := cache.StartPurger(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Wait for everything to be purged
	time.Sleep(200 * time.Millisecond)

	// At this point everything should be been cleaned, and the purger should be
	// sleeping forever
	items, err := cache.Search(CacheKeyFromQuery(
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
		cache.StoreItem(i.Item, 100*time.Millisecond, ck)
	}

	time.Sleep(200 * time.Millisecond)

	// It should be empty again
	items, err = cache.Search(CacheKeyFromQuery(
		cachedItems[1].Item.GetMetadata().GetSourceQuery(),
		cachedItems[1].Item.GetMetadata().GetSourceName(),
	))

	if !errors.Is(err, ErrCacheNotFound) {
		t.Errorf("unexpected error: %v", err)
		t.Errorf("unexpected items: %v: %v", len(items), items)
	}
}

func TestStopPurge(t *testing.T) {
	cache := NewCache()
	cache.MinWaitTime = 1 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())

	err := cache.StartPurger(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Stop the purger
	cancel()

	// Insert an item
	item := GenerateRandomItem()
	ck := CacheKeyFromQuery(item.GetMetadata().GetSourceQuery(), item.GetMetadata().GetSourceName())

	cache.StoreItem(item, time.Millisecond, ck)
	sst := SST{
		SourceName: item.GetMetadata().GetSourceName(),
		Scope:      item.GetScope(),
		Type:       item.GetType(),
	}

	// Make sure it's not purged
	time.Sleep(100 * time.Millisecond)
	items, err := cache.Search(CacheKey{
		SST: sst,
	})

	if err != nil {
		t.Error(err)
	}

	if len(items) != 1 {
		t.Errorf("Expected 1 item, got %v", len(items))
	}
}

func TestDelete(t *testing.T) {
	cache := NewCache()

	// Insert an item
	item := GenerateRandomItem()
	ck := CacheKeyFromQuery(item.GetMetadata().GetSourceQuery(), item.GetMetadata().GetSourceName())

	cache.StoreItem(item, time.Millisecond, ck)
	sst := SST{
		SourceName: item.GetMetadata().GetSourceName(),
		Scope:      item.GetScope(),
		Type:       item.GetType(),
	}

	// It should be there
	items, err := cache.Search(CacheKey{
		SST: sst,
	})

	if err != nil {
		t.Error(err)
	}

	if len(items) != 1 {
		t.Errorf("expected 1 item, got %v", len(items))
	}

	// Delete it
	cache.Delete(CacheKey{
		SST: sst,
	})

	// It should be gone
	items, err = cache.Search(CacheKey{
		SST: sst,
	})

	if !errors.Is(err, ErrCacheNotFound) {
		t.Errorf("expected ErrCacheNotFound, got %v", err)
	}

	if len(items) != 0 {
		t.Errorf("expected 0 item, got %v", len(items))
	}
}

// This test is designed to be run with -race to ensure that there aren't any
// data races
func TestConcurrent(t *testing.T) {
	cache := NewCache()
	// Run the purger super fast to generate a worst-case scenario
	cache.MinWaitTime = 1 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	err := cache.StartPurger(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var wg sync.WaitGroup

	numParallel := 1_000

	for range numParallel {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Store the item
			item := GenerateRandomItem()
			ck := CacheKeyFromQuery(item.GetMetadata().GetSourceQuery(), item.GetMetadata().GetSourceName())

			cache.StoreItem(item, 100*time.Millisecond, ck)

			wg.Add(1)
			// Create a goroutine to also delete in parallel
			go func() {
				defer wg.Done()
				cache.Delete(ck)
			}()
		}()
	}

	wg.Wait()
}

func TestPointers(t *testing.T) {
	cache := NewCache()

	item := GenerateRandomItem()
	ck := CacheKeyFromQuery(item.GetMetadata().GetSourceQuery(), item.GetMetadata().GetSourceName())

	cache.StoreItem(item, time.Minute, ck)

	item.Type = "bad"

	items, err := cache.Search(ck)

	if err != nil {
		t.Error(err)
	}

	if len(items) != 1 {
		t.Errorf("expected 1 item, got %v", len(items))
	}

	if items[0].GetType() == "bad" {
		t.Error("item was changed in cache")
	}
}

func TestCacheClear(t *testing.T) {
	cache := NewCache()

	cache.Clear()

	// Populate the cache
	item := GenerateRandomItem()
	ck := CacheKeyFromQuery(item.GetMetadata().GetSourceQuery(), item.GetMetadata().GetSourceName())

	cache.StoreItem(item, 500*time.Millisecond, ck)

	// Start purging just to make sure it doesn't break
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	err := cache.StartPurger(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Make sure the cache is populated
	_, err = cache.Search(ck)

	if err != nil {
		t.Error(err)
	}

	// Clear the cache
	cache.Clear()

	// Make sure the cache is empty
	_, err = cache.Search(ck)

	if err == nil {
		t.Error("expected error, cache not cleared")
	}

	// Make sure we can populate it again
	cache.StoreItem(item, 500*time.Millisecond, ck)
	_, err = cache.Search(ck)

	if err != nil {
		t.Error(err)
	}
}

func TestToIndexValues(t *testing.T) {
	ck := CacheKey{
		SST: SST{
			SourceName: "foo",
			Scope:      "foo",
			Type:       "foo",
		},
	}

	t.Run("with just SST", func(t *testing.T) {
		iv := ck.ToIndexValues()

		if iv.SSTHash != ck.SST.Hash() {
			t.Error("hash mismatch")
		}
	})

	t.Run("with SST & Method", func(t *testing.T) {
		ck.Method = sdp.QueryMethod_GET.Enum()
		iv := ck.ToIndexValues()

		if iv.Method != sdp.QueryMethod_GET {
			t.Errorf("expected %v, got %v", sdp.QueryMethod_GET, iv.Method)
		}
	})

	t.Run("with SST & Query", func(t *testing.T) {
		q := "query"
		ck.Query = &q
		iv := ck.ToIndexValues()

		if iv.Query != "query" {
			t.Errorf("expected %v, got %v", "query", iv.Query)
		}
	})

	t.Run("with SST & UniqueAttributeValue", func(t *testing.T) {
		q := "foo"
		ck.UniqueAttributeValue = &q
		iv := ck.ToIndexValues()

		if iv.UniqueAttributeValue != "foo" {
			t.Errorf("expected %v, got %v", "foo", iv.UniqueAttributeValue)
		}
	})
}

func TestLookup(t *testing.T) {
	ctx := context.Background()
	cache := NewCache()

	item := GenerateRandomItem()
	ck := CacheKeyFromQuery(item.GetMetadata().GetSourceQuery(), item.GetMetadata().GetSourceName())

	cache.StoreItem(item, 10*time.Second, ck)

	// ignore the cache
	cacheHit, _, cachedItems, err := cache.Lookup(ctx, item.GetMetadata().GetSourceName(), sdp.QueryMethod_GET, item.GetScope(), item.GetType(), item.UniqueAttributeValue(), true)
	if err != nil {
		t.Fatal(err)
	}
	if cacheHit {
		t.Error("expected cache miss, got hit")
	}
	if cachedItems != nil {
		t.Errorf("expected nil items, got %v", cachedItems)
	}

	// Lookup the item
	cacheHit, _, cachedItems, err = cache.Lookup(ctx, item.GetMetadata().GetSourceName(), sdp.QueryMethod_GET, item.GetScope(), item.GetType(), item.UniqueAttributeValue(), false)

	if err != nil {
		t.Fatal(err)
	}
	if !cacheHit {
		t.Fatal("expected cache hit, got miss")
	}
	if len(cachedItems) != 1 {
		t.Fatalf("expected 1 item, got %v", len(cachedItems))
	}

	if cachedItems[0].GetType() != item.GetType() {
		t.Errorf("expected type %v, got %v", item.GetType(), cachedItems[0].GetType())
	}

	if cachedItems[0].Health == nil {
		t.Error("expected health to be set")
	}

	if len(cachedItems[0].GetTags()) != len(item.GetTags()) {
		t.Error("expected tags to be set")
	}

	stats := cache.Purge(time.Now().Add(1 * time.Hour))
	if stats.NumPurged != 1 {
		t.Errorf("expected 1 item purged, got %v", stats.NumPurged)
	}

	// Lookup the item
	cacheHit, _, cachedItems, err = cache.Lookup(ctx, item.GetMetadata().GetSourceName(), sdp.QueryMethod_GET, item.GetScope(), item.GetType(), item.UniqueAttributeValue(), false)

	if err != nil {
		t.Fatal(err)
	}
	if cacheHit {
		t.Fatal("expected cache miss, got hit")
	}
	if len(cachedItems) != 0 {
		t.Fatalf("expected 0 item, got %v", len(cachedItems))
	}
}

func TestStoreSearch(t *testing.T) {
	ctx := context.Background()
	cache := NewCache()

	item := GenerateRandomItem()
	item.Metadata.SourceQuery.Method = sdp.QueryMethod_SEARCH
	ck := CacheKeyFromQuery(item.GetMetadata().GetSourceQuery(), item.GetMetadata().GetSourceName())

	cache.StoreItem(item, 10*time.Second, ck)

	// Lookup the item as GET request
	cacheHit, _, cachedItems, err := cache.Lookup(ctx, item.GetMetadata().GetSourceName(), sdp.QueryMethod_GET, item.GetScope(), item.GetType(), item.UniqueAttributeValue(), false)
	if err != nil {
		t.Fatal(err)
	}

	if !cacheHit {
		t.Fatal("expected cache hit, got miss")
	}

	if len(cachedItems) != 1 {
		t.Fatalf("expected 1 item, got %v", len(cachedItems))
	}

	if cachedItems[0].GetType() != item.GetType() {
		t.Errorf("expected type %v, got %v", item.GetType(), cachedItems[0].GetType())
	}
}
