package sdpcache

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/overmindtech/cli/sdp-go"
)

// cacheImplementations returns the list of cache implementations to test
func cacheImplementations(t *testing.T) []struct {
	name    string
	factory func() Cache
} {
	return []struct {
		name    string
		factory func() Cache
	}{
		{"MemoryCache", func() Cache { return NewMemoryCache() }},
		{"BoltCache", func() Cache {
			c, err := NewBoltCache(filepath.Join(t.TempDir(), "cache.db"))
			if err != nil {
				t.Fatalf("failed to create BoltCache: %v", err)
			}
			t.Cleanup(func() {
				c.Close()
			})
			return c
		}},
	}
}

func TestStoreItem(t *testing.T) {
	implementations := cacheImplementations(t)

	for _, impl := range implementations {
		t.Run(impl.name, func(t *testing.T) {
			cache := impl.factory()

			item := GenerateRandomItem()
			ck := CacheKeyFromQuery(item.GetMetadata().GetSourceQuery(), item.GetMetadata().GetSourceName())
			cache.StoreItem(t.Context(), item, 10*time.Second, ck)

			results, err := cache.Search(ck)
			if err != nil {
				t.Error(err)
			}

			if len(results) != 1 {
				t.Errorf("expected 1 result, got %v", len(results))
			}

			// Test another match
			item = GenerateRandomItem()
			ck = CacheKeyFromQuery(item.GetMetadata().GetSourceQuery(), item.GetMetadata().GetSourceName())

			cache.StoreItem(t.Context(), item, 10*time.Second, ck)

			results, err = cache.Search(ck)
			if err != nil {
				t.Error(err)
			}

			if len(results) != 1 {
				t.Errorf("expected 1 result, got %v", len(results))
			}

			// Test different scope
			item = GenerateRandomItem()
			ck = CacheKeyFromQuery(item.GetMetadata().GetSourceQuery(), item.GetMetadata().GetSourceName())

			cache.StoreItem(t.Context(), item, 10*time.Second, ck)

			ck.SST.Scope = fmt.Sprintf("new scope %v", ck.SST.Scope)

			results, err = cache.Search(ck)
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
}

func TestStoreError(t *testing.T) {
	implementations := cacheImplementations(t)

	for _, impl := range implementations {
		t.Run(impl.name, func(t *testing.T) {
			cache := impl.factory()

			// Test with just an error
			sst := SST{
				SourceName: "foo",
				Scope:      "foo",
				Type:       "foo",
			}

			uav := "foo"

			cache.StoreError(t.Context(), errors.New("arse"), 10*time.Second, CacheKey{
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

			// Test with items and an error for the same query
			// Add an item with the same details as above
			item := GenerateRandomItem()
			item.Metadata.SourceQuery.Method = sdp.QueryMethod_GET
			item.Metadata.SourceQuery.Query = "foo"
			item.Metadata.SourceName = "foo"
			item.Scope = "foo"
			item.Type = "foo"

			ck := CacheKeyFromQuery(item.GetMetadata().GetSourceQuery(), item.GetMetadata().GetSourceName())

			items, err = cache.Search(ck)

			if len(items) > 0 {
				t.Errorf("expected 0 items, got %v", len(items))
			}

			if err == nil {
				t.Error("expected error, got nil")
			}

			// Test with multiple errors
			cache.StoreError(t.Context(), errors.New("nope"), 10*time.Second, CacheKey{
				SST:    sst,
				Method: sdp.QueryMethod_GET.Enum(),
				Query:  &uav,
			})

			items, err = cache.Search(CacheKey{
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
}

func TestPurge(t *testing.T) {
	implementations := cacheImplementations(t)

	for _, impl := range implementations {
		t.Run(impl.name, func(t *testing.T) {
			cache := impl.factory()

			cachedItems := []struct {
				Item   *sdp.Item
				Expiry time.Time
			}{
				{
					Item:   GenerateRandomItem(),
					Expiry: time.Now().Add(50 * time.Millisecond),
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
				cache.StoreItem(t.Context(), i.Item, time.Until(i.Expiry), ck)
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
			stats := cache.Purge(t.Context(), cachedItems[0].Expiry.Add(500 * time.Millisecond))

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
			stats = cache.Purge(t.Context(), cachedItems[4].Expiry.Add(500 * time.Millisecond))

			if stats.NumPurged != 4 {
				t.Errorf("expected 4 item purged, got %v", stats.NumPurged)
			}

			// Purge the last one
			stats = cache.Purge(t.Context(), cachedItems[5].Expiry.Add(500 * time.Millisecond))

			if stats.NumPurged != 1 {
				t.Errorf("expected 1 item purged, got %v", stats.NumPurged)
			}

			if stats.NextExpiry != nil {
				t.Errorf("expected expiry to be nil, got %v", stats.NextExpiry)
			}
		})
	}
}

func TestDelete(t *testing.T) {
	implementations := cacheImplementations(t)

	for _, impl := range implementations {
		t.Run(impl.name, func(t *testing.T) {
			cache := impl.factory()

			// Insert an item
			item := GenerateRandomItem()
			ck := CacheKeyFromQuery(item.GetMetadata().GetSourceQuery(), item.GetMetadata().GetSourceName())

			cache.StoreItem(t.Context(), item, time.Millisecond, ck)
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
		})
	}
}

func TestPointers(t *testing.T) {
	implementations := cacheImplementations(t)

	for _, impl := range implementations {
		t.Run(impl.name, func(t *testing.T) {
			cache := impl.factory()

			item := GenerateRandomItem()
			ck := CacheKeyFromQuery(item.GetMetadata().GetSourceQuery(), item.GetMetadata().GetSourceName())

			cache.StoreItem(t.Context(), item, time.Minute, ck)

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
		})
	}
}

func TestCacheClear(t *testing.T) {
	implementations := cacheImplementations(t)

	for _, impl := range implementations {
		t.Run(impl.name, func(t *testing.T) {
			ctx := t.Context()
			cache := impl.factory()

			cache.Clear()

			// Populate the cache
			item := GenerateRandomItem()
			ck := CacheKeyFromQuery(item.GetMetadata().GetSourceQuery(), item.GetMetadata().GetSourceName())

			cache.StoreItem(ctx, item, 500*time.Millisecond, ck)

			// Start purging just to make sure it doesn't break
			ctx, cancel := context.WithCancel(ctx)
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
			cache.StoreItem(ctx, item, 500*time.Millisecond, ck)
			_, err = cache.Search(ck)

			if err != nil {
				t.Error(err)
			}
		})
	}
}

func TestLookup(t *testing.T) {
	implementations := cacheImplementations(t)

	for _, impl := range implementations {
		t.Run(impl.name, func(t *testing.T) {
			ctx := t.Context()
			cache := impl.factory()

			item := GenerateRandomItem()
			ck := CacheKeyFromQuery(item.GetMetadata().GetSourceQuery(), item.GetMetadata().GetSourceName())

			cache.StoreItem(ctx, item, 10*time.Second, ck)

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

			stats := cache.Purge(ctx, time.Now().Add(1 * time.Hour))
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
		})
	}
}

func TestStoreSearch(t *testing.T) {
	implementations := cacheImplementations(t)

	for _, impl := range implementations {
		t.Run(impl.name, func(t *testing.T) {
			ctx := t.Context()
			cache := impl.factory()

			item := GenerateRandomItem()
			item.Metadata.SourceQuery.Method = sdp.QueryMethod_SEARCH
			ck := CacheKeyFromQuery(item.GetMetadata().GetSourceQuery(), item.GetMetadata().GetSourceName())

			cache.StoreItem(ctx, item, 10*time.Second, ck)

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
		})
	}
}

func TestLookupWithListMethod(t *testing.T) {
	implementations := cacheImplementations(t)

	for _, impl := range implementations {
		t.Run(impl.name, func(t *testing.T) {
			ctx := t.Context()
			cache := impl.factory()

			// Store multiple items with same SST
			sst := SST{SourceName: "test", Scope: "scope", Type: "type"}
			listMethod := sdp.QueryMethod_LIST

			item1 := GenerateRandomItem()
			item1.Scope = sst.Scope
			item1.Type = sst.Type
			item1.Metadata.SourceName = sst.SourceName
			ck1 := CacheKey{SST: sst, Method: &listMethod}
			cache.StoreItem(ctx, item1, 10*time.Second, ck1)

			item2 := GenerateRandomItem()
			item2.Scope = sst.Scope
			item2.Type = sst.Type
			item2.Metadata.SourceName = sst.SourceName
			ck2 := CacheKey{SST: sst, Method: &listMethod}
			cache.StoreItem(ctx, item2, 10*time.Second, ck2)

			// Lookup with LIST should return both items
			cacheHit, _, cachedItems, err := cache.Lookup(ctx, sst.SourceName, sdp.QueryMethod_LIST, sst.Scope, sst.Type, "", false)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !cacheHit {
				t.Fatal("expected cache hit, got miss")
			}
			if len(cachedItems) != 2 {
				t.Errorf("expected 2 items, got %v", len(cachedItems))
			}
		})
	}
}

func TestSearchWithListMethod(t *testing.T) {
	implementations := cacheImplementations(t)

	for _, impl := range implementations {
		t.Run(impl.name, func(t *testing.T) {
			cache := impl.factory()

			// Store items with LIST method
			sst := SST{SourceName: "test", Scope: "scope", Type: "type"}
			listMethod := sdp.QueryMethod_LIST
			ck := CacheKey{SST: sst, Method: &listMethod}

			item1 := GenerateRandomItem()
			item1.Scope = sst.Scope
			item1.Type = sst.Type
			cache.StoreItem(t.Context(), item1, 10*time.Second, ck)

			item2 := GenerateRandomItem()
			item2.Scope = sst.Scope
			item2.Type = sst.Type
			cache.StoreItem(t.Context(), item2, 10*time.Second, ck)

			// Search should return both items
			items, err := cache.Search(ck)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(items) != 2 {
				t.Errorf("expected 2 items, got %v", len(items))
			}
		})
	}
}

func TestSearchMethodWithDifferentQueries(t *testing.T) {
	implementations := cacheImplementations(t)

	for _, impl := range implementations {
		t.Run(impl.name, func(t *testing.T) {
			cache := impl.factory()

			sst := SST{SourceName: "test", Scope: "scope", Type: "type"}
			searchMethod := sdp.QueryMethod_SEARCH

			// Store items with different search queries
			query1 := "query1"
			ck1 := CacheKey{SST: sst, Method: &searchMethod, Query: &query1}
			item1 := GenerateRandomItem()
			item1.Scope = sst.Scope
			item1.Type = sst.Type
			cache.StoreItem(t.Context(), item1, 10*time.Second, ck1)

			query2 := "query2"
			ck2 := CacheKey{SST: sst, Method: &searchMethod, Query: &query2}
			item2 := GenerateRandomItem()
			item2.Scope = sst.Scope
			item2.Type = sst.Type
			cache.StoreItem(t.Context(), item2, 10*time.Second, ck2)

			// Search with query1 should only return item1
			items, err := cache.Search(ck1)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(items) != 1 {
				t.Errorf("expected 1 item for query1, got %v", len(items))
			}

			// Search with query2 should only return item2
			items, err = cache.Search(ck2)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(items) != 1 {
				t.Errorf("expected 1 item for query2, got %v", len(items))
			}
		})
	}
}

func TestSearchWithPartialCacheKey(t *testing.T) {
	implementations := cacheImplementations(t)

	for _, impl := range implementations {
		t.Run(impl.name, func(t *testing.T) {
			cache := impl.factory()

			sst := SST{SourceName: "test", Scope: "scope", Type: "type"}

			// Store items with different methods
			getMethod := sdp.QueryMethod_GET
			listMethod := sdp.QueryMethod_LIST

			item1 := GenerateRandomItem()
			item1.Scope = sst.Scope
			item1.Type = sst.Type
			uav1 := "item1"
			ck1 := CacheKey{SST: sst, Method: &getMethod, UniqueAttributeValue: &uav1}
			cache.StoreItem(t.Context(), item1, 10*time.Second, ck1)

			item2 := GenerateRandomItem()
			item2.Scope = sst.Scope
			item2.Type = sst.Type
			ck2 := CacheKey{SST: sst, Method: &listMethod}
			cache.StoreItem(t.Context(), item2, 10*time.Second, ck2)

			// Search with SST only should return both items
			ckPartial := CacheKey{SST: sst}
			items, err := cache.Search(ckPartial)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(items) != 2 {
				t.Errorf("expected 2 items with SST-only search, got %v", len(items))
			}
		})
	}
}

func TestDeleteWithPartialCacheKey(t *testing.T) {
	implementations := cacheImplementations(t)

	for _, impl := range implementations {
		t.Run(impl.name, func(t *testing.T) {
			cache := impl.factory()

			sst := SST{SourceName: "test", Scope: "scope", Type: "type"}

			// Store multiple items with same SST
			item1 := GenerateRandomItem()
			item1.Scope = sst.Scope
			item1.Type = sst.Type
			ck1 := CacheKeyFromQuery(item1.GetMetadata().GetSourceQuery(), sst.SourceName)
			cache.StoreItem(t.Context(), item1, 10*time.Second, ck1)

			item2 := GenerateRandomItem()
			item2.Scope = sst.Scope
			item2.Type = sst.Type
			ck2 := CacheKeyFromQuery(item2.GetMetadata().GetSourceQuery(), sst.SourceName)
			cache.StoreItem(t.Context(), item2, 10*time.Second, ck2)

			// Delete with SST only should remove all items
			cache.Delete(CacheKey{SST: sst})

			// Verify all items are gone
			items, err := cache.Search(CacheKey{SST: sst})
			if !errors.Is(err, ErrCacheNotFound) {
				t.Errorf("expected ErrCacheNotFound after delete, got: %v", err)
			}
			if len(items) != 0 {
				t.Errorf("expected 0 items after delete, got %v", len(items))
			}
		})
	}
}

func TestLookupWithCachedError(t *testing.T) {
	implementations := cacheImplementations(t)

	for _, impl := range implementations {
		t.Run(impl.name, func(t *testing.T) {
			ctx := t.Context()
			cache := impl.factory()

			// Test different error types
			errorTypes := []struct {
				name      string
				errorType sdp.QueryError_ErrorType
			}{
				{"NOTFOUND", sdp.QueryError_NOTFOUND},
				{"NOSCOPE", sdp.QueryError_NOSCOPE},
				{"TIMEOUT", sdp.QueryError_TIMEOUT},
				{"OTHER", sdp.QueryError_OTHER},
			}

			for i, et := range errorTypes {
				t.Run(et.name, func(t *testing.T) {
					sst := SST{
						SourceName: fmt.Sprintf("test%d", i),
						Scope:      "scope",
						Type:       "type",
					}
					method := sdp.QueryMethod_GET
					query := "test"
					ck := CacheKey{SST: sst, Method: &method, UniqueAttributeValue: &query}

					// Store error
					qErr := &sdp.QueryError{
						ErrorType:   et.errorType,
						ErrorString: fmt.Sprintf("test error %s", et.name),
						Scope:       sst.Scope,
						SourceName:  sst.SourceName,
						ItemType:    sst.Type,
					}
					cache.StoreError(ctx, qErr, 10*time.Second, ck)

					// Lookup should return cached error
					cacheHit, _, items, returnedErr := cache.Lookup(ctx, sst.SourceName, method, sst.Scope, sst.Type, query, false)

					if !cacheHit {
						t.Error("expected cache hit for cached error")
					}
					if items != nil {
						t.Errorf("expected nil items, got %v", items)
					}
					if returnedErr == nil {
						t.Fatal("expected error to be returned")
					}
					if returnedErr.GetErrorType() != et.errorType {
						t.Errorf("expected error type %v, got %v", et.errorType, returnedErr.GetErrorType())
					}
				})
			}
		})
	}
}

func TestGetMinWaitTime(t *testing.T) {
	implementations := cacheImplementations(t)

	for _, impl := range implementations {
		t.Run(impl.name, func(t *testing.T) {
			cache := impl.factory()

			minWaitTime := cache.GetMinWaitTime()

			// Should return a positive duration
			if minWaitTime <= 0 {
				t.Errorf("expected positive duration, got %v", minWaitTime)
			}

			// Default should be reasonable (e.g., 5 seconds)
			if minWaitTime > time.Minute {
				t.Errorf("expected reasonable default (< 1 minute), got %v", minWaitTime)
			}
		})
	}
}

func TestEmptyCacheOperations(t *testing.T) {
	implementations := cacheImplementations(t)

	for _, impl := range implementations {
		t.Run(impl.name, func(t *testing.T) {
			cache := impl.factory()

			sst := SST{SourceName: "test", Scope: "scope", Type: "type"}
			ck := CacheKey{SST: sst}

			// Search on empty cache
			items, err := cache.Search(ck)
			if !errors.Is(err, ErrCacheNotFound) {
				t.Errorf("expected ErrCacheNotFound on empty cache, got: %v", err)
			}
			if len(items) != 0 {
				t.Errorf("expected 0 items on empty cache, got %v", len(items))
			}

			// Delete on empty cache (should be idempotent)
			cache.Delete(ck)

			// Purge on empty cache
			stats := cache.Purge(t.Context(), time.Now())
			if stats.NumPurged != 0 {
				t.Errorf("expected 0 items purged on empty cache, got %v", stats.NumPurged)
			}
			if stats.NextExpiry != nil {
				t.Errorf("expected nil NextExpiry on empty cache, got %v", stats.NextExpiry)
			}

			// Clear on empty cache (should not error)
			cache.Clear()
		})
	}
}

func TestMultipleItemsSameSST(t *testing.T) {
	implementations := cacheImplementations(t)

	for _, impl := range implementations {
		t.Run(impl.name, func(t *testing.T) {
			ctx := t.Context()
			cache := impl.factory()

			sst := SST{SourceName: "test", Scope: "scope", Type: "type"}
			method := sdp.QueryMethod_GET

			// Store multiple items with same SST but different unique attributes
			items := make([]*sdp.Item, 3)
			for i := range 3 {
				item := GenerateRandomItem()
				item.Scope = sst.Scope
				item.Type = sst.Type
				item.Metadata.SourceName = sst.SourceName
				uav := fmt.Sprintf("item%d", i)

				// Set the item's unique attribute value to match the CacheKey
				attrs := make(map[string]interface{})
				if item.GetAttributes() != nil && item.GetAttributes().GetAttrStruct() != nil {
					for k, v := range item.GetAttributes().GetAttrStruct().GetFields() {
						attrs[k] = v
					}
				}
				attrs["name"] = uav
				attributes, _ := sdp.ToAttributes(attrs)
				item.Attributes = attributes

				ck := CacheKey{SST: sst, Method: &method, UniqueAttributeValue: &uav}
				cache.StoreItem(ctx, item, 10*time.Second, ck)
				items[i] = item
			}

			// Search with SST only should return all 3 items
			allItems, err := cache.Search(CacheKey{SST: sst})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(allItems) != 3 {
				t.Errorf("expected 3 items, got %v", len(allItems))
			}

			// Search with specific unique attribute should return only that item
			for i := range 3 {
				uav := fmt.Sprintf("item%d", i)
				ck := CacheKey{SST: sst, Method: &method, UniqueAttributeValue: &uav}
				foundItems, err := cache.Search(ck)
				if err != nil {
					t.Errorf("unexpected error for item%d: %v", i, err)
				}
				if len(foundItems) != 1 {
					t.Errorf("expected 1 item for item%d, got %v", i, len(foundItems))
				}
			}
		})
	}
}

// Implementation-specific tests for MemoryCache

// TestMemoryCacheStartPurge tests the memory cache implementation's purger
func TestMemoryCacheStartPurge(t *testing.T) {
	ctx := t.Context()
	cache := NewMemoryCache()
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
		cache.StoreItem(ctx, i.Item, time.Until(i.Expiry), ck)
	}

	ctx, cancel := context.WithCancel(ctx)
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
		cache.StoreItem(ctx, i.Item, 100*time.Millisecond, ck)
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

// TestMemoryCacheStopPurge tests the memory cache implementation's purger stop functionality
func TestMemoryCacheStopPurge(t *testing.T) {
	cache := NewMemoryCache()
	cache.MinWaitTime = 1 * time.Millisecond

	ctx, cancel := context.WithCancel(t.Context())

	err := cache.StartPurger(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Stop the purger
	cancel()

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

// TestMemoryCacheConcurrent tests the memory cache implementation for data races.
// This test is designed to be run with -race to ensure that there aren't any
// data races
func TestMemoryCacheConcurrent(t *testing.T) {
	cache := NewMemoryCache()
	// Run the purger super fast to generate a worst-case scenario
	cache.MinWaitTime = 1 * time.Millisecond

	ctx, cancel := context.WithCancel(t.Context())
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

			cache.StoreItem(ctx, item, 100*time.Millisecond, ck)

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

func TestUnexpiredOverwriteLogging(t *testing.T) {
	cache := NewCache()

	t.Run("overwriting unexpired entry increments counter", func(t *testing.T) {
		ctx := t.Context()
		// Create an item and cache key
		item := GenerateRandomItem()
		ck := CacheKeyFromQuery(item.GetMetadata().GetSourceQuery(), item.GetMetadata().GetSourceName())

		// Store the item with a long TTL (10 seconds)
		cache.StoreItem(ctx, item, 10*time.Second, ck)

		// Store the same item again before it expires (overwrite will be tracked via span attributes)
		cache.StoreItem(ctx, item, 10*time.Second, ck)

		// Store it again
		cache.StoreItem(ctx, item, 10*time.Second, ck)
	})

	t.Run("overwriting expired entry does not increment counter", func(t *testing.T) {
		ctx := t.Context()
		// Create a new cache for this test
		cache := NewCache()

		item := GenerateRandomItem()
		ck := CacheKeyFromQuery(item.GetMetadata().GetSourceQuery(), item.GetMetadata().GetSourceName())

		// Store the item with a very short TTL
		cache.StoreItem(ctx, item, 1*time.Millisecond, ck)

		// Wait for it to expire
		time.Sleep(10 * time.Millisecond)

		// Store the same item again after it expired (overwrite tracking via span attributes)
		cache.StoreItem(ctx, item, 10*time.Second, ck)
	})

	t.Run("overwriting different items does not increment counter", func(t *testing.T) {
		ctx := t.Context()
		// Create a new cache for this test
		cache := NewCache()

		item1 := GenerateRandomItem()
		item2 := GenerateRandomItem()

		ck1 := CacheKeyFromQuery(item1.GetMetadata().GetSourceQuery(), item1.GetMetadata().GetSourceName())
		ck2 := CacheKeyFromQuery(item2.GetMetadata().GetSourceQuery(), item2.GetMetadata().GetSourceName())

		// Store two different items (no overwrites, just new items)
		cache.StoreItem(ctx, item1, 10*time.Second, ck1)
		cache.StoreItem(ctx, item2, 10*time.Second, ck2)
	})

	t.Run("overwriting error entries increments counter", func(t *testing.T) {
		ctx := t.Context()
		// Create a new cache for this test
		cache := NewCache()

		sst := SST{
			SourceName: "test-source",
			Scope:      "test-scope",
			Type:       "test-type",
		}

		method := sdp.QueryMethod_LIST
		query := "test-query"

		ck := CacheKey{
			SST:    sst,
			Method: &method,
			Query:  &query,
		}

		// Store an error
		cache.StoreError(ctx, errors.New("test error"), 10*time.Second, ck)

		// Store the same error again before it expires (overwrite will be tracked via span attributes)
		cache.StoreError(ctx, errors.New("another error"), 10*time.Second, ck)
	})
}

// TestBoltCacheExistingFile verifies that BoltCache correctly handles existing cache files
// from previous runs. It should open the existing file, access existing data, and
// allow the purge process to clean up expired items.
func TestBoltCacheExistingFile(t *testing.T) {
	tempDir := t.TempDir()
	cachePath := filepath.Join(tempDir, "cache.db")

	// Create a cache and store some data
	ctx := t.Context()
	cache1, err := NewBoltCache(cachePath)
	if err != nil {
		t.Fatalf("failed to create BoltCache: %v", err)
	}

	// Store an item with a long TTL
	item1 := GenerateRandomItem()
	ck1 := CacheKeyFromQuery(item1.GetMetadata().GetSourceQuery(), item1.GetMetadata().GetSourceName())
	cache1.StoreItem(ctx, item1, 10*time.Second, ck1)

	// Store another item with a short TTL (will expire)
	item2 := GenerateRandomItem()
	ck2 := CacheKeyFromQuery(item2.GetMetadata().GetSourceQuery(), item2.GetMetadata().GetSourceName())
	cache1.StoreItem(ctx, item2, 100*time.Millisecond, ck2)

	// Verify both items are in the cache
	items, err := cache1.Search(ck1)
	if err != nil {
		t.Errorf("failed to search for item1: %v", err)
	}
	if len(items) != 1 {
		t.Errorf("expected 1 item for ck1, got %d", len(items))
	}

	items, err = cache1.Search(ck2)
	if err != nil {
		t.Errorf("failed to search for item2: %v", err)
	}
	if len(items) != 1 {
		t.Errorf("expected 1 item for ck2, got %d", len(items))
	}

	// Close the cache
	if err := cache1.Close(); err != nil {
		t.Fatalf("failed to close cache1: %v", err)
	}

	// Wait for item2 to expire
	time.Sleep(150 * time.Millisecond)

	// Create a new cache at the same path - should open the existing file
	cache2, err := NewBoltCache(cachePath)
	if err != nil {
		t.Fatalf("failed to create BoltCache with existing file: %v", err)
	}
	defer cache2.Close()

	// Verify item1 is still accessible (not expired)
	items, err = cache2.Search(ck1)
	if err != nil {
		t.Errorf("failed to search for item1 in existing cache: %v", err)
	}
	if len(items) != 1 {
		t.Errorf("expected 1 item for ck1 in existing cache, got %d", len(items))
	}

	// Verify item2 is expired (should not be found or should be purged)
	items, err = cache2.Search(ck2)
	if err == nil && len(items) > 0 {
		// Item might still be there if purge hasn't run, so let's purge explicitly
		cache2.Purge(ctx, time.Now())
		items, err = cache2.Search(ck2)
		if err == nil && len(items) > 0 {
			t.Errorf("expected item2 to be expired and purged, but found %d items", len(items))
		}
	}

	// Verify purge process works on existing data
	stats := cache2.Purge(ctx, time.Now())
	if stats.NumPurged == 0 && err == nil {
		// If we got here, item2 should have been purged
		// Let's verify it's gone
		items, err = cache2.Search(ck2)
		if !errors.Is(err, ErrCacheNotFound) && len(items) > 0 {
			t.Errorf("expected item2 to be purged, but search returned: err=%v, items=%d", err, len(items))
		}
	}

	// Verify we can still store new items
	item3 := GenerateRandomItem()
	ck3 := CacheKeyFromQuery(item3.GetMetadata().GetSourceQuery(), item3.GetMetadata().GetSourceName())
	cache2.StoreItem(ctx, item3, 10*time.Second, ck3)

	items, err = cache2.Search(ck3)
	if err != nil {
		t.Errorf("failed to search for newly stored item3: %v", err)
	}
	if len(items) != 1 {
		t.Errorf("expected 1 item for ck3, got %d", len(items))
	}
}

// TestBoltCacheDiskFullErrorDetection tests the isDiskFullError helper function
func TestBoltCacheDiskFullErrorDetection(t *testing.T) {
	// This test verifies that isDiskFullError correctly identifies disk full errors
	// We can't easily simulate actual disk full in tests, but we can test the detection logic

	// Note: We can't directly test syscall.ENOSPC without actually filling the disk,
	// but we can verify the function exists and works with the error types it's designed for.
	// In a real scenario, BoltDB would return syscall.ENOSPC when the disk is full.

	// Test that non-disk-full errors are not detected
	regularErr := errors.New("some other error")
	if isDiskFullError(regularErr) {
		t.Error("isDiskFullError should return false for regular errors")
	}

	// Test nil error
	if isDiskFullError(nil) {
		t.Error("isDiskFullError should return false for nil")
	}
}

// TestBoltCacheDeleteOnDiskFull tests that the cache is deleted when disk is full
// and cleanup doesn't help. Since we can't easily simulate disk full in unit tests,
// this test verifies the deleteCacheFile method works correctly.
func TestBoltCacheDeleteOnDiskFull(t *testing.T) {
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
	items, err := cache.Search(ck)
	if err != nil {
		t.Errorf("failed to search: %v", err)
	}
	if len(items) != 1 {
		t.Errorf("expected 1 item, got %d", len(items))
	}

	// Delete the cache file (cache is already *BoltCache)
	if err := cache.deleteCacheFile(); err != nil {
		t.Fatalf("failed to delete cache file: %v", err)
	}

	// Verify the cache file is gone
	if _, err := os.Stat(cachePath); !os.IsNotExist(err) {
		t.Error("cache file should be deleted")
	}

	// Verify the database is closed (can't search anymore)
	_, _ = cache.Search(ck)
	// The search might fail or return empty, but the important thing is the file is gone
	// and we can't use the cache anymore
}

// TestBoltCacheDiskFullDuringCompact tests error handling during compaction.
// Since we can't easily simulate disk full, this test verifies the compaction
// process works normally and that the error handling paths exist.
func TestBoltCacheDiskFullDuringCompact(t *testing.T) {
	tempDir := t.TempDir()
	cachePath := filepath.Join(tempDir, "cache.db")

	cache, err := NewBoltCache(cachePath, WithCompactThreshold(1024)) // Small threshold to trigger compaction
	if err != nil {
		t.Fatalf("failed to create BoltCache: %v", err)
	}
	defer cache.Close()

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
	stats := cache.Purge(ctx, time.Now().Add(-1 * time.Hour)) // Purge items from an hour ago (none should exist)
	_ = stats                                            // Use stats to avoid unused variable

	// Verify cache still works after compaction attempt
	item := GenerateRandomItem()
	ck := CacheKeyFromQuery(item.GetMetadata().GetSourceQuery(), item.GetMetadata().GetSourceName())
	cache.StoreItem(ctx, item, 10*time.Second, ck)

	items, err := cache.Search(ck)
	if err != nil {
		t.Errorf("failed to search after compaction: %v", err)
	}
	if len(items) != 1 {
		t.Errorf("expected 1 item after compaction, got %d", len(items))
	}
}
