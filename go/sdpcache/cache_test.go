package sdpcache

// ──────────────────────────────────────────────────────────────────────
// Implementation-detail tests for stateful cache backends.
//
// These tests exercise the internal Search method and storage internals
// that are NOT part of the public Cache contract. Contract-level tests
// (using only the public Cache interface) live in cache_contract_test.go.
// NoOpCache-specific tests live in noop_cache_test.go.
// ──────────────────────────────────────────────────────────────────────

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/overmindtech/cli/go/sdp-go"
)

type searchableCache interface {
	Search(context.Context, CacheKey) ([]*sdp.Item, error)
}

// testSearch is a helper function that calls the lower-level Search method on
// cache implementations for testing purposes.
func testSearch(ctx context.Context, cache Cache, ck CacheKey) ([]*sdp.Item, error) {
	if c, ok := cache.(searchableCache); ok {
		return c.Search(ctx, ck)
	}

	return nil, fmt.Errorf("unsupported cache type for search: %T", cache)
}

// cacheImplementations returns stateful cache implementations used by shared
// behavior tests. NoOpCache is intentionally excluded and tested separately.
// Accepts testing.TB so it can be used by both tests and benchmarks.
func cacheImplementations(tb testing.TB) []struct {
	name    string
	factory func() Cache
} {
	return []struct {
		name    string
		factory func() Cache
	}{
		{"MemoryCache", func() Cache { return NewMemoryCache() }},
		{"BoltCache", func() Cache {
			c, err := NewBoltCache(filepath.Join(tb.TempDir(), "cache.db"))
			if err != nil {
				tb.Fatalf("failed to create BoltCache: %v", err)
			}
			tb.Cleanup(func() {
				_ = c.CloseAndDestroy()
			})
			return c
		}},
		{"ShardedCache", func() Cache {
			c, err := NewShardedCache(
				filepath.Join(tb.TempDir(), "shards"),
				DefaultShardCount,
			)
			if err != nil {
				tb.Fatalf("failed to create ShardedCache: %v", err)
			}
			tb.Cleanup(func() {
				_ = c.CloseAndDestroy()
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

			results, err := testSearch(t.Context(), cache, ck)
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

			results, err = testSearch(t.Context(), cache, ck)
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

			results, err = testSearch(t.Context(), cache, ck)
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

func TestStoreUnavailableItem(t *testing.T) {
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

			cache.StoreUnavailableItem(t.Context(), errors.New("arse"), 10*time.Second, CacheKey{
				SST:    sst,
				Method: sdp.QueryMethod_GET.Enum(),
				Query:  &uav,
			})

			items, err := testSearch(t.Context(), cache, CacheKey{
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

			items, err = testSearch(t.Context(), cache, ck)

			if len(items) > 0 {
				t.Errorf("expected 0 items, got %v", len(items))
			}

			if err == nil {
				t.Error("expected error, got nil")
			}

			// Test with multiple errors
			cache.StoreUnavailableItem(t.Context(), errors.New("nope"), 10*time.Second, CacheKey{
				SST:    sst,
				Method: sdp.QueryMethod_GET.Enum(),
				Query:  &uav,
			})

			items, err = testSearch(t.Context(), cache, CacheKey{
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
				items, err := testSearch(t.Context(), cache, ck)
				if err != nil {
					t.Error(err)
				}

				if len(items) != 1 {
					t.Errorf("expected 1 item, got %v", len(items))
				}
			}

			// Purge just the first one
			stats := cache.Purge(t.Context(), cachedItems[0].Expiry.Add(500*time.Millisecond))

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
			stats = cache.Purge(t.Context(), cachedItems[4].Expiry.Add(500*time.Millisecond))

			if stats.NumPurged != 4 {
				t.Errorf("expected 4 item purged, got %v", stats.NumPurged)
			}

			// Purge the last one
			stats = cache.Purge(t.Context(), cachedItems[5].Expiry.Add(500*time.Millisecond))

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
			items, err := testSearch(t.Context(), cache, CacheKey{
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
			items, err = testSearch(t.Context(), cache, CacheKey{
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
			items, err := testSearch(t.Context(), cache, ck)
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
			items, err := testSearch(t.Context(), cache, ck1)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(items) != 1 {
				t.Errorf("expected 1 item for query1, got %v", len(items))
			}

			// Search with query2 should only return item2
			items, err = testSearch(t.Context(), cache, ck2)
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
			items, err := testSearch(t.Context(), cache, ckPartial)
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
			items, err := testSearch(t.Context(), cache, CacheKey{SST: sst})
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
					cache.StoreUnavailableItem(ctx, qErr, 10*time.Second, ck)

					// Lookup should return cached error
					cacheHit, _, items, returnedErr, done := cache.Lookup(ctx, sst.SourceName, method, sst.Scope, sst.Type, query, false)
					defer done()

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
			items, err := testSearch(t.Context(), cache, ck)
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
				attrs := make(map[string]any)
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
			allItems, err := testSearch(t.Context(), cache, CacheKey{SST: sst})
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
				foundItems, err := testSearch(t.Context(), cache, ck)
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
	cache := NewCache(t.Context())

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
		cache := NewCache(ctx)

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
		cache := NewCache(ctx)

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
		cache := NewCache(ctx)

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
		cache.StoreUnavailableItem(ctx, errors.New("test error"), 10*time.Second, ck)

		// Store the same error again before it expires (overwrite will be tracked via span attributes)
		cache.StoreUnavailableItem(ctx, errors.New("another error"), 10*time.Second, ck)
	})
}

// TestPendingWorkUnit tests the pendingWork component in isolation.
func TestPendingWorkUnit(t *testing.T) {
	t.Run("StartWork first caller", func(t *testing.T) {
		pw := newPendingWork()
		shouldWork, entry := pw.StartWork("key1")

		if !shouldWork {
			t.Error("first caller should do work")
		}
		if entry == nil {
			t.Error("entry should not be nil")
		}
	})

	t.Run("StartWork second caller", func(t *testing.T) {
		pw := newPendingWork()

		// First caller
		shouldWork1, entry1 := pw.StartWork("key1")
		if !shouldWork1 {
			t.Error("first caller should do work")
		}

		// Second caller for same key
		shouldWork2, entry2 := pw.StartWork("key1")
		if shouldWork2 {
			t.Error("second caller should not do work")
		}
		if entry2 != entry1 {
			t.Error("second caller should get same entry")
		}
	})

	t.Run("Complete wakes waiters", func(t *testing.T) {
		pw := newPendingWork()
		ctx := context.Background()

		// First caller
		_, entry := pw.StartWork("key1")

		// Second caller waits
		var wg sync.WaitGroup
		var waitOk bool

		wg.Go(func() {
			waitOk = pw.Wait(ctx, entry)
		})

		// Give waiter time to start waiting
		time.Sleep(10 * time.Millisecond)

		// Complete the work
		pw.Complete("key1")

		wg.Wait()

		if !waitOk {
			t.Error("wait should succeed")
		}
	})

	t.Run("Wait respects context donelation", func(t *testing.T) {
		pw := newPendingWork()
		ctx, done := context.WithCancel(context.Background())

		// First caller
		_, entry := pw.StartWork("key1")

		// Second caller waits with donelable context
		var wg sync.WaitGroup
		var waitOk bool

		wg.Go(func() {
			waitOk = pw.Wait(ctx, entry)
		})

		// Give waiter time to start waiting
		time.Sleep(10 * time.Millisecond)

		// Cancel the context
		done()

		wg.Wait()

		if waitOk {
			t.Error("wait should fail due to context donelation")
		}
	})
}
