package sdpcache

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/overmindtech/cli/go/sdp-go"
)

// ──────────────────────────────────────────────────────────────────────
// Contract tests for the Cache interface.
//
// Every test in this file exercises only the public Cache methods and
// asserts guarantees documented on the Cache interface in cache.go.
// Implementation internals (Search, pending, shardFor, …) are tested
// in the backend-specific test files.
//
// NoOpCache is intentionally excluded; its dedicated no-op semantics
// are validated in noop_cache_test.go.
// ──────────────────────────────────────────────────────────────────────

// --- Lookup: miss / item-hit / error-hit ------------------------------------

func TestCacheContract_LookupMiss(t *testing.T) {
	for _, impl := range cacheImplementations(t) {
		t.Run(impl.name, func(t *testing.T) {
			cache := impl.factory()

			hit, ck, items, qErr, done := cache.Lookup(
				t.Context(), "src", sdp.QueryMethod_GET, "scope", "type", "query", false,
			)
			defer done()

			if hit {
				t.Fatal("expected miss on empty cache")
			}
			if len(items) != 0 {
				t.Fatalf("expected no items, got %d", len(items))
			}
			if qErr != nil {
				t.Fatalf("expected nil error, got %v", qErr)
			}
			if ck.SST.SourceName != "src" || ck.SST.Scope != "scope" || ck.SST.Type != "type" {
				t.Fatalf("returned CacheKey SST mismatch: %v", ck)
			}
		})
	}
}

func TestCacheContract_LookupItemHit(t *testing.T) {
	for _, impl := range cacheImplementations(t) {
		t.Run(impl.name, func(t *testing.T) {
			ctx := t.Context()
			cache := impl.factory()

			item := GenerateRandomItem()
			ck := CacheKeyFromQuery(item.GetMetadata().GetSourceQuery(), item.GetMetadata().GetSourceName())
			cache.StoreItem(ctx, item, 10*time.Second, ck)

			hit, _, items, qErr, done := cache.Lookup(ctx,
				item.GetMetadata().GetSourceName(),
				sdp.QueryMethod_GET,
				item.GetScope(),
				item.GetType(),
				item.UniqueAttributeValue(),
				false,
			)
			defer done()

			if !hit {
				t.Fatal("expected item hit")
			}
			if qErr != nil {
				t.Fatalf("expected nil error, got %v", qErr)
			}
			if len(items) != 1 {
				t.Fatalf("expected 1 item, got %d", len(items))
			}
			if items[0].GetType() != item.GetType() {
				t.Errorf("type mismatch: got %q, want %q", items[0].GetType(), item.GetType())
			}
		})
	}
}

func TestCacheContract_LookupErrorHit(t *testing.T) {
	for _, impl := range cacheImplementations(t) {
		t.Run(impl.name, func(t *testing.T) {
			ctx := t.Context()
			cache := impl.factory()

			sst := SST{SourceName: "src", Scope: "scope", Type: "type"}
			ck := CacheKey{SST: sst, Method: new(sdp.QueryMethod_GET), UniqueAttributeValue: new("q")}

			qErr := &sdp.QueryError{
				ErrorType:   sdp.QueryError_NOTFOUND,
				ErrorString: "not found",
				Scope:       sst.Scope,
				SourceName:  sst.SourceName,
				ItemType:    sst.Type,
			}
			cache.StoreUnavailableItem(ctx, qErr, 10*time.Second, ck)

			hit, _, items, retErr, done := cache.Lookup(ctx, sst.SourceName, sdp.QueryMethod_GET, sst.Scope, sst.Type, "q", false)
			defer done()

			if !hit {
				t.Fatal("expected error hit")
			}
			if items != nil {
				t.Fatalf("expected nil items on error hit, got %d", len(items))
			}
			if retErr == nil {
				t.Fatal("expected non-nil QueryError")
			}
			if retErr.GetErrorType() != sdp.QueryError_NOTFOUND {
				t.Errorf("error type: got %v, want NOTFOUND", retErr.GetErrorType())
			}
		})
	}
}

// --- ignoreCache -----------------------------------------------------------

func TestCacheContract_IgnoreCache(t *testing.T) {
	for _, impl := range cacheImplementations(t) {
		t.Run(impl.name, func(t *testing.T) {
			ctx := t.Context()
			cache := impl.factory()

			item := GenerateRandomItem()
			ck := CacheKeyFromQuery(item.GetMetadata().GetSourceQuery(), item.GetMetadata().GetSourceName())
			cache.StoreItem(ctx, item, 10*time.Second, ck)

			hit, _, items, qErr, done := cache.Lookup(ctx,
				item.GetMetadata().GetSourceName(),
				sdp.QueryMethod_GET,
				item.GetScope(),
				item.GetType(),
				item.UniqueAttributeValue(),
				true, // ignoreCache
			)
			defer done()

			if hit {
				t.Fatal("expected miss with ignoreCache=true")
			}
			if len(items) != 0 {
				t.Errorf("expected no items, got %d", len(items))
			}
			if qErr != nil {
				t.Errorf("expected nil error, got %v", qErr)
			}
		})
	}
}

// --- done() idempotency ----------------------------------------------------

func TestCacheContract_DoneIdempotent(t *testing.T) {
	for _, impl := range cacheImplementations(t) {
		t.Run(impl.name, func(t *testing.T) {
			cache := impl.factory()

			_, _, _, _, done := cache.Lookup(
				t.Context(), "src", sdp.QueryMethod_GET, "scope", "type", "q", false,
			)
			done()
			done() // must not panic
		})
	}
}

func TestCacheContract_DoneIdempotentOnHit(t *testing.T) {
	for _, impl := range cacheImplementations(t) {
		t.Run(impl.name, func(t *testing.T) {
			ctx := t.Context()
			cache := impl.factory()

			item := GenerateRandomItem()
			ck := CacheKeyFromQuery(item.GetMetadata().GetSourceQuery(), item.GetMetadata().GetSourceName())
			cache.StoreItem(ctx, item, 10*time.Second, ck)

			_, _, _, _, done := cache.Lookup(ctx,
				item.GetMetadata().GetSourceName(),
				sdp.QueryMethod_GET,
				item.GetScope(),
				item.GetType(),
				item.UniqueAttributeValue(),
				false,
			)
			done()
			done() // must not panic
		})
	}
}

// --- GET cardinality -------------------------------------------------------

func TestCacheContract_GETMultipleItemsPurgesAndMisses(t *testing.T) {
	for _, impl := range cacheImplementations(t) {
		t.Run(impl.name, func(t *testing.T) {
			ctx := t.Context()
			cache := impl.factory()

			sst := SST{SourceName: "src", Scope: "scope", Type: "type"}
			listMethod := sdp.QueryMethod_LIST
			ck := CacheKey{SST: sst, Method: &listMethod}

			// Store two distinct entries (different GUN via scope) that both share
			// the same unique attribute value used by GET lookup.
			for i := range 2 {
				item := GenerateRandomItem()
				item.Scope = fmt.Sprintf("%s-%d", sst.Scope, i)
				item.Type = sst.Type
				item.Metadata.SourceName = sst.SourceName
				item.GetAttributes().Set("name", "shared-uav")
				cache.StoreItem(ctx, item, 10*time.Second, ck)
			}

			// Precondition: both entries are retrievable.
			hit, _, items, qErr, done := cache.Lookup(ctx, sst.SourceName, sdp.QueryMethod_LIST, sst.Scope, sst.Type, "", false)
			defer done()
			if !hit {
				t.Fatal("expected LIST hit before GET cardinality purge")
			}
			if qErr != nil {
				t.Fatalf("expected nil error for LIST precondition, got %v", qErr)
			}
			if len(items) != 2 {
				t.Fatalf("expected 2 LIST items before purge, got %d", len(items))
			}

			hit, _, _, _, done2 := cache.Lookup(ctx, sst.SourceName, sdp.QueryMethod_GET, sst.Scope, sst.Type, "shared-uav", false)
			defer done2()
			if hit {
				t.Fatal("expected miss when GET finds >1 item (cardinality purge)")
			}

			// The purge should have removed all entries that matched the GET key.
			hit, _, _, _, done3 := cache.Lookup(ctx, sst.SourceName, sdp.QueryMethod_LIST, sst.Scope, sst.Type, "", false)
			defer done3()
			if hit {
				t.Fatal("expected LIST miss after GET cardinality purge")
			}
		})
	}
}

// --- Copy semantics --------------------------------------------------------

func TestCacheContract_StoreItemCopies(t *testing.T) {
	for _, impl := range cacheImplementations(t) {
		t.Run(impl.name, func(t *testing.T) {
			ctx := t.Context()
			cache := impl.factory()

			item := GenerateRandomItem()
			ck := CacheKeyFromQuery(item.GetMetadata().GetSourceQuery(), item.GetMetadata().GetSourceName())
			cache.StoreItem(ctx, item, 10*time.Second, ck)

			original := item.GetType()
			item.Type = "mutated-after-store"

			hit, _, items, _, done := cache.Lookup(ctx,
				item.GetMetadata().GetSourceName(),
				sdp.QueryMethod_GET,
				item.GetScope(),
				original,
				item.UniqueAttributeValue(),
				false,
			)
			defer done()

			if !hit || len(items) == 0 {
				t.Fatal("expected hit after StoreItem")
			}
			if items[0].GetType() == "mutated-after-store" {
				t.Error("cached item was mutated through original pointer")
			}
		})
	}
}

// --- StoreItem + Lookup round-trip for LIST & SEARCH -----------------------

func TestCacheContract_LISTReturnsMultipleItems(t *testing.T) {
	for _, impl := range cacheImplementations(t) {
		t.Run(impl.name, func(t *testing.T) {
			ctx := t.Context()
			cache := impl.factory()

			sst := SST{SourceName: "src", Scope: "scope", Type: "type"}
			ck := CacheKey{SST: sst, Method: new(sdp.QueryMethod_LIST)}

			for i := range 3 {
				item := GenerateRandomItem()
				item.Scope = sst.Scope
				item.Type = sst.Type
				item.Metadata.SourceName = sst.SourceName
				item.GetAttributes().Set("name", fmt.Sprintf("item-%d", i))
				cache.StoreItem(ctx, item, 10*time.Second, ck)
			}

			hit, _, items, qErr, done := cache.Lookup(ctx, sst.SourceName, sdp.QueryMethod_LIST, sst.Scope, sst.Type, "", false)
			defer done()

			if qErr != nil {
				t.Fatalf("unexpected error: %v", qErr)
			}
			if !hit {
				t.Fatal("expected hit")
			}
			if len(items) != 3 {
				t.Errorf("expected 3 items, got %d", len(items))
			}
		})
	}
}

func TestCacheContract_SEARCHIsolatesByQuery(t *testing.T) {
	for _, impl := range cacheImplementations(t) {
		t.Run(impl.name, func(t *testing.T) {
			ctx := t.Context()
			cache := impl.factory()

			sst := SST{SourceName: "src", Scope: "scope", Type: "type"}

			ck1 := CacheKey{SST: sst, Method: new(sdp.QueryMethod_SEARCH), Query: new("alpha")}
			item1 := GenerateRandomItem()
			item1.Scope = sst.Scope
			item1.Type = sst.Type
			item1.Metadata.SourceName = sst.SourceName
			cache.StoreItem(ctx, item1, 10*time.Second, ck1)

			ck2 := CacheKey{SST: sst, Method: new(sdp.QueryMethod_SEARCH), Query: new("beta")}
			item2 := GenerateRandomItem()
			item2.Scope = sst.Scope
			item2.Type = sst.Type
			item2.Metadata.SourceName = sst.SourceName
			cache.StoreItem(ctx, item2, 10*time.Second, ck2)

			// Lookup alpha
			hit, _, items, _, done := cache.Lookup(ctx, sst.SourceName, sdp.QueryMethod_SEARCH, sst.Scope, sst.Type, "alpha", false)
			defer done()
			if !hit || len(items) != 1 {
				t.Errorf("alpha: hit=%v, items=%d", hit, len(items))
			}

			// Lookup beta
			hit, _, items, _, done2 := cache.Lookup(ctx, sst.SourceName, sdp.QueryMethod_SEARCH, sst.Scope, sst.Type, "beta", false)
			defer done2()
			if !hit || len(items) != 1 {
				t.Errorf("beta: hit=%v, items=%d", hit, len(items))
			}
		})
	}
}

// --- SEARCH items retrievable via GET (cross-method hit) -------------------

func TestCacheContract_SEARCHItemRetrievableViaGET(t *testing.T) {
	for _, impl := range cacheImplementations(t) {
		t.Run(impl.name, func(t *testing.T) {
			ctx := t.Context()
			cache := impl.factory()

			item := GenerateRandomItem()
			item.Metadata.SourceQuery.Method = sdp.QueryMethod_SEARCH
			ck := CacheKeyFromQuery(item.GetMetadata().GetSourceQuery(), item.GetMetadata().GetSourceName())
			cache.StoreItem(ctx, item, 10*time.Second, ck)

			hit, _, items, qErr, done := cache.Lookup(ctx,
				item.GetMetadata().GetSourceName(),
				sdp.QueryMethod_GET,
				item.GetScope(),
				item.GetType(),
				item.UniqueAttributeValue(),
				false,
			)
			defer done()

			if qErr != nil {
				t.Fatalf("unexpected error: %v", qErr)
			}
			if !hit {
				t.Fatal("expected GET hit for SEARCH-stored item")
			}
			if len(items) != 1 {
				t.Fatalf("expected 1 item, got %d", len(items))
			}
		})
	}
}

// --- Delete ----------------------------------------------------------------

func TestCacheContract_DeleteRemovesEntry(t *testing.T) {
	for _, impl := range cacheImplementations(t) {
		t.Run(impl.name, func(t *testing.T) {
			ctx := t.Context()
			cache := impl.factory()

			item := GenerateRandomItem()
			ck := CacheKeyFromQuery(item.GetMetadata().GetSourceQuery(), item.GetMetadata().GetSourceName())
			cache.StoreItem(ctx, item, 10*time.Second, ck)

			cache.Delete(ck)

			hit, _, _, _, done := cache.Lookup(ctx,
				item.GetMetadata().GetSourceName(),
				sdp.QueryMethod_GET,
				item.GetScope(),
				item.GetType(),
				item.UniqueAttributeValue(),
				false,
			)
			defer done()

			if hit {
				t.Fatal("expected miss after Delete")
			}
		})
	}
}

func TestCacheContract_DeleteWildcard(t *testing.T) {
	for _, impl := range cacheImplementations(t) {
		t.Run(impl.name, func(t *testing.T) {
			ctx := t.Context()
			cache := impl.factory()

			sst := SST{SourceName: "src", Scope: "scope", Type: "type"}

			for i := range 3 {
				item := GenerateRandomItem()
				item.Scope = sst.Scope
				item.Type = sst.Type
				item.Metadata.SourceName = sst.SourceName
				item.GetAttributes().Set("name", fmt.Sprintf("wc-%d", i))
				ck := CacheKey{SST: sst, Method: new(sdp.QueryMethod_LIST)}
				cache.StoreItem(ctx, item, 10*time.Second, ck)
			}

			// Delete with SST-only (wildcard on method/uav)
			cache.Delete(CacheKey{SST: sst})

			hit, _, _, _, done := cache.Lookup(ctx, sst.SourceName, sdp.QueryMethod_LIST, sst.Scope, sst.Type, "", false)
			defer done()

			if hit {
				t.Fatal("expected miss after wildcard Delete")
			}
		})
	}
}

func TestCacheContract_DeleteOnEmptyCacheIsIdempotent(t *testing.T) {
	for _, impl := range cacheImplementations(t) {
		t.Run(impl.name, func(t *testing.T) {
			cache := impl.factory()
			cache.Delete(CacheKey{SST: SST{SourceName: "x", Scope: "y", Type: "z"}})
		})
	}
}

// --- Clear -----------------------------------------------------------------

func TestCacheContract_Clear(t *testing.T) {
	for _, impl := range cacheImplementations(t) {
		t.Run(impl.name, func(t *testing.T) {
			ctx := t.Context()
			cache := impl.factory()

			item := GenerateRandomItem()
			ck := CacheKeyFromQuery(item.GetMetadata().GetSourceQuery(), item.GetMetadata().GetSourceName())
			cache.StoreItem(ctx, item, 10*time.Second, ck)

			cache.Clear()

			hit, _, _, _, done := cache.Lookup(ctx,
				item.GetMetadata().GetSourceName(),
				sdp.QueryMethod_GET,
				item.GetScope(),
				item.GetType(),
				item.UniqueAttributeValue(),
				false,
			)
			defer done()

			if hit {
				t.Fatal("expected miss after Clear")
			}
		})
	}
}

func TestCacheContract_ClearThenStoreWorks(t *testing.T) {
	for _, impl := range cacheImplementations(t) {
		t.Run(impl.name, func(t *testing.T) {
			ctx := t.Context()
			cache := impl.factory()

			cache.Clear()

			item := GenerateRandomItem()
			ck := CacheKeyFromQuery(item.GetMetadata().GetSourceQuery(), item.GetMetadata().GetSourceName())
			cache.StoreItem(ctx, item, 10*time.Second, ck)

			hit, _, items, _, done := cache.Lookup(ctx,
				item.GetMetadata().GetSourceName(),
				sdp.QueryMethod_GET,
				item.GetScope(),
				item.GetType(),
				item.UniqueAttributeValue(),
				false,
			)
			defer done()

			if !hit || len(items) != 1 {
				t.Fatalf("expected hit with 1 item after Clear+Store, got hit=%v items=%d", hit, len(items))
			}
		})
	}
}

func TestCacheContract_ClearOnEmptyCacheIsIdempotent(t *testing.T) {
	for _, impl := range cacheImplementations(t) {
		t.Run(impl.name, func(t *testing.T) {
			cache := impl.factory()
			cache.Clear()
			cache.Clear()
		})
	}
}

// --- Purge -----------------------------------------------------------------

func TestCacheContract_PurgeRemovesExpired(t *testing.T) {
	for _, impl := range cacheImplementations(t) {
		t.Run(impl.name, func(t *testing.T) {
			ctx := t.Context()
			cache := impl.factory()

			item := GenerateRandomItem()
			ck := CacheKeyFromQuery(item.GetMetadata().GetSourceQuery(), item.GetMetadata().GetSourceName())
			cache.StoreItem(ctx, item, 50*time.Millisecond, ck)

			stats := cache.Purge(ctx, time.Now().Add(100*time.Millisecond))

			if stats.NumPurged != 1 {
				t.Errorf("expected 1 purged, got %d", stats.NumPurged)
			}

			hit, _, _, _, done := cache.Lookup(ctx,
				item.GetMetadata().GetSourceName(),
				sdp.QueryMethod_GET,
				item.GetScope(),
				item.GetType(),
				item.UniqueAttributeValue(),
				false,
			)
			defer done()

			if hit {
				t.Fatal("expected miss after purge")
			}
		})
	}
}

func TestCacheContract_PurgeStatsNextExpiry(t *testing.T) {
	for _, impl := range cacheImplementations(t) {
		t.Run(impl.name, func(t *testing.T) {
			ctx := t.Context()
			cache := impl.factory()

			item1 := GenerateRandomItem()
			ck1 := CacheKeyFromQuery(item1.GetMetadata().GetSourceQuery(), item1.GetMetadata().GetSourceName())
			cache.StoreItem(ctx, item1, 50*time.Millisecond, ck1)

			item2 := GenerateRandomItem()
			ck2 := CacheKeyFromQuery(item2.GetMetadata().GetSourceQuery(), item2.GetMetadata().GetSourceName())
			cache.StoreItem(ctx, item2, 5*time.Second, ck2)

			stats := cache.Purge(ctx, time.Now().Add(100*time.Millisecond))

			if stats.NumPurged != 1 {
				t.Errorf("expected 1 purged, got %d", stats.NumPurged)
			}
			if stats.NextExpiry == nil {
				t.Fatal("expected non-nil NextExpiry (second item still cached)")
			}
		})
	}
}

func TestCacheContract_PurgeEmptyCache(t *testing.T) {
	for _, impl := range cacheImplementations(t) {
		t.Run(impl.name, func(t *testing.T) {
			cache := impl.factory()
			stats := cache.Purge(t.Context(), time.Now())

			if stats.NumPurged != 0 {
				t.Errorf("expected 0 purged on empty cache, got %d", stats.NumPurged)
			}
			if stats.NextExpiry != nil {
				t.Errorf("expected nil NextExpiry on empty cache, got %v", stats.NextExpiry)
			}
		})
	}
}

// --- GetMinWaitTime --------------------------------------------------------

func TestCacheContract_GetMinWaitTimePositive(t *testing.T) {
	for _, impl := range cacheImplementations(t) {
		t.Run(impl.name, func(t *testing.T) {
			cache := impl.factory()
			if d := cache.GetMinWaitTime(); d <= 0 {
				t.Errorf("stateful cache should return positive min wait time, got %v", d)
			}
		})
	}
}

// --- StartPurger -----------------------------------------------------------

func TestCacheContract_StartPurgerPurgesExpired(t *testing.T) {
	for _, impl := range cacheImplementations(t) {
		t.Run(impl.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			cache := impl.factory()

			item := GenerateRandomItem()
			ck := CacheKeyFromQuery(item.GetMetadata().GetSourceQuery(), item.GetMetadata().GetSourceName())
			cache.StoreItem(ctx, item, 50*time.Millisecond, ck)

			cache.StartPurger(ctx)

			// Wait long enough for at least one purge cycle.
			time.Sleep(cache.GetMinWaitTime() + 200*time.Millisecond)

			hit, _, _, _, done := cache.Lookup(ctx,
				item.GetMetadata().GetSourceName(),
				sdp.QueryMethod_GET,
				item.GetScope(),
				item.GetType(),
				item.UniqueAttributeValue(),
				false,
			)
			defer done()

			if hit {
				t.Error("expected miss after purger ran (item expired)")
			}
		})
	}
}

// --- Thundering herd / deduplication (documented contract) ----------------

func TestCacheContract_LookupDeduplication(t *testing.T) {
	for _, impl := range cacheImplementations(t) {
		t.Run(impl.name, func(t *testing.T) {
			cache := impl.factory()
			ctx := t.Context()

			sst := SST{SourceName: "src", Scope: "scope", Type: "type"}

			var workCount int
			var mu sync.Mutex
			var wg sync.WaitGroup

			numGoroutines := 10
			results := make([]bool, numGoroutines)
			startBarrier := make(chan struct{})

			for idx := range numGoroutines {
				wg.Go(func() {
					<-startBarrier

					hit, ck, _, _, done := cache.Lookup(ctx, sst.SourceName, sdp.QueryMethod_LIST, sst.Scope, sst.Type, "", false)
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

						hit, _, _, _, done2 := cache.Lookup(ctx, sst.SourceName, sdp.QueryMethod_LIST, sst.Scope, sst.Type, "", false)
						defer done2()
						results[idx] = hit
					} else {
						results[idx] = true
					}
				})
			}

			close(startBarrier)
			wg.Wait()

			if workCount != 1 {
				t.Fatalf("expected 1 worker, got %d", workCount)
			}
			for i, hit := range results {
				if !hit {
					t.Errorf("goroutine %d: expected hit after dedup, got miss", i)
				}
			}
		})
	}
}

func TestCacheContract_WaitersGetMissWhenWorkerStoresNothing(t *testing.T) {
	for _, impl := range cacheImplementations(t) {
		t.Run(impl.name, func(t *testing.T) {
			cache := impl.factory()
			ctx := t.Context()

			sst := SST{SourceName: "src", Scope: "scope", Type: "type"}

			var wg sync.WaitGroup
			startBarrier := make(chan struct{})

			numWaiters := 3
			waiterHits := make([]bool, 0, numWaiters)
			var waiterMu sync.Mutex

			// Worker: gets miss, completes without storing.
			wg.Go(func() {
				<-startBarrier

				hit, _, _, _, done := cache.Lookup(ctx, sst.SourceName, sdp.QueryMethod_LIST, sst.Scope, sst.Type, "no-store", false)
				if hit {
					t.Error("worker: expected miss")
				}
				time.Sleep(50 * time.Millisecond)
				done()
			})

			for range numWaiters {
				wg.Go(func() {
					<-startBarrier
					time.Sleep(10 * time.Millisecond)

					hit, _, _, _, done := cache.Lookup(ctx, sst.SourceName, sdp.QueryMethod_LIST, sst.Scope, sst.Type, "no-store", false)
					defer done()

					waiterMu.Lock()
					waiterHits = append(waiterHits, hit)
					waiterMu.Unlock()
				})
			}

			close(startBarrier)
			wg.Wait()

			for i, hit := range waiterHits {
				if hit {
					t.Errorf("waiter %d: expected miss when worker stored nothing", i)
				}
			}
		})
	}
}

// --- Error precedence over items -------------------------------------------

func TestCacheContract_ErrorTakesPrecedenceOverItems(t *testing.T) {
	for _, impl := range cacheImplementations(t) {
		t.Run(impl.name, func(t *testing.T) {
			ctx := t.Context()
			cache := impl.factory()

			sst := SST{SourceName: "src", Scope: "scope", Type: "type"}
			ck := CacheKey{SST: sst, Method: new(sdp.QueryMethod_GET), UniqueAttributeValue: new("prec")}

			// Store an item first.
			item := GenerateRandomItem()
			item.Scope = sst.Scope
			item.Type = sst.Type
			item.Metadata.SourceName = sst.SourceName
			item.GetAttributes().Set("name", "prec")
			cache.StoreItem(ctx, item, 10*time.Second, ck)

			// Then store an error under the same key.
			cache.StoreUnavailableItem(ctx, &sdp.QueryError{
				ErrorType:   sdp.QueryError_NOTFOUND,
				ErrorString: "gone",
				Scope:       sst.Scope,
				SourceName:  sst.SourceName,
				ItemType:    sst.Type,
			}, 10*time.Second, ck)

			hit, _, items, qErr, done := cache.Lookup(ctx, sst.SourceName, sdp.QueryMethod_GET, sst.Scope, sst.Type, "prec", false)
			defer done()

			if !hit {
				t.Fatal("expected hit")
			}
			if qErr == nil {
				t.Fatal("expected error hit (error should take precedence over items)")
			}
			if items != nil {
				t.Errorf("expected nil items when error takes precedence, got %d", len(items))
			}
		})
	}
}

// --- Zero/negative TTL -----------------------------------------------------

func TestCacheContract_ZeroTTLPurgedImmediately(t *testing.T) {
	for _, impl := range cacheImplementations(t) {
		t.Run(impl.name, func(t *testing.T) {
			ctx := t.Context()
			cache := impl.factory()

			item := GenerateRandomItem()
			ck := CacheKeyFromQuery(item.GetMetadata().GetSourceQuery(), item.GetMetadata().GetSourceName())
			cache.StoreItem(ctx, item, 0, ck)

			// A zero-TTL item sets expiry to ~time.Now(). It may survive a
			// Search in the same nanosecond (strict Before check) but must
			// not survive a Purge with a future cutoff.
			stats := cache.Purge(ctx, time.Now().Add(time.Second))
			if stats.NumPurged != 1 {
				t.Errorf("expected 1 purged, got %d", stats.NumPurged)
			}

			hit, _, _, _, done := cache.Lookup(ctx,
				item.GetMetadata().GetSourceName(),
				sdp.QueryMethod_GET,
				item.GetScope(),
				item.GetType(),
				item.UniqueAttributeValue(),
				false,
			)
			defer done()

			if hit {
				t.Error("expected miss after purging zero-TTL item")
			}
		})
	}
}

// --- Multiple error types --------------------------------------------------

func TestCacheContract_StoreUnavailableItemTypes(t *testing.T) {
	errorTypes := []sdp.QueryError_ErrorType{
		sdp.QueryError_NOTFOUND,
		sdp.QueryError_NOSCOPE,
		sdp.QueryError_TIMEOUT,
		sdp.QueryError_OTHER,
	}

	for _, impl := range cacheImplementations(t) {
		t.Run(impl.name, func(t *testing.T) {
			ctx := t.Context()
			cache := impl.factory()

			for i, et := range errorTypes {
				t.Run(et.String(), func(t *testing.T) {
					sst := SST{
						SourceName: fmt.Sprintf("src-%d", i),
						Scope:      "scope",
						Type:       "type",
					}
					ck := CacheKey{SST: sst, Method: new(sdp.QueryMethod_GET), UniqueAttributeValue: new("q")}

					qErr := &sdp.QueryError{
						ErrorType:   et,
						ErrorString: fmt.Sprintf("err %s", et),
						Scope:       sst.Scope,
						SourceName:  sst.SourceName,
						ItemType:    sst.Type,
					}
					cache.StoreUnavailableItem(ctx, qErr, 10*time.Second, ck)

					hit, _, items, retErr, done := cache.Lookup(ctx, sst.SourceName, sdp.QueryMethod_GET, sst.Scope, sst.Type, "q", false)
					defer done()

					if !hit {
						t.Fatal("expected hit for cached error")
					}
					if items != nil {
						t.Errorf("expected nil items, got %d", len(items))
					}
					if retErr == nil || retErr.GetErrorType() != et {
						t.Errorf("error type: got %v, want %v", retErr.GetErrorType(), et)
					}
				})
			}
		})
	}
}
