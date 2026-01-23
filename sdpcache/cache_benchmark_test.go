package sdpcache

import (
	"context"
	"fmt"
	"math/rand"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/overmindtech/cli/sdp-go"
)

const CacheDuration = 10 * time.Second

// NewPopulatedCache Returns a newly populated cache and the CacheQuery that
// matches a randomly selected item in that cache
func NewPopulatedCache(ctx context.Context, numberItems int) (Cache, CacheKey) {
	// Populate the cache
	c := NewCache(ctx)

	var item *sdp.Item
	var exampleCk CacheKey
	exampleIndex := rand.Intn(numberItems)

	for i := range numberItems {
		item = GenerateRandomItem()
		ck := CacheKeyFromQuery(item.GetMetadata().GetSourceQuery(), item.GetMetadata().GetSourceName())

		if i == exampleIndex {
			exampleCk = ck
		}

		c.StoreItem(ctx, item, CacheDuration, ck)
	}

	return c, exampleCk
}

// NewPopulatedCacheWithListItems populates a cache with items that share the same
// SST (Source, Scope, Type) for LIST query benchmarking. All items will be returned
// when searching with a LIST query for the given SST.
func NewPopulatedCacheWithListItems(cache Cache, numberItems int, sst SST) CacheKey {
	listMethod := sdp.QueryMethod_LIST
	ck := CacheKey{SST: sst, Method: &listMethod}

	for i := range numberItems {
		item := GenerateRandomItem()
		item.Scope = sst.Scope
		item.Type = sst.Type
		item.Metadata.SourceName = sst.SourceName

		// Ensure each item has a unique attribute value to prevent overwrites
		// Format: "item-{index}" to guarantee uniqueness
		uniqueValue := fmt.Sprintf("item-%d", i)
		item.GetAttributes().Set("name", uniqueValue)

		cache.StoreItem(context.Background(), item, CacheDuration, ck)
	}

	return ck
}

// NewPopulatedCacheWithMultipleBuckets creates a cache with multiple SST buckets
// to enable realistic concurrent access patterns where different goroutines hit
// different buckets.
func NewPopulatedCacheWithMultipleBuckets(cache Cache, itemsPerBucket, numBuckets int) []CacheKey {
	keys := make([]CacheKey, numBuckets)
	listMethod := sdp.QueryMethod_LIST

	for bucketIdx := range numBuckets {
		sst := SST{
			SourceName: "test-source",
			Scope:      fmt.Sprintf("scope-%d", bucketIdx),
			Type:       "test-type",
		}

		keys[bucketIdx] = CacheKey{SST: sst, Method: &listMethod}

		for i := range itemsPerBucket {
			item := GenerateRandomItem()
				item.Scope = sst.Scope
				item.Type = sst.Type
				item.Metadata.SourceName = sst.SourceName
				uniqueValue := fmt.Sprintf("bucket-%d-item-%d", bucketIdx, i)
				item.GetAttributes().Set("name", uniqueValue)

			cache.StoreItem(context.Background(), item, CacheDuration, keys[bucketIdx])
		}
	}

	return keys
}

func BenchmarkCache1SingleItem(b *testing.B) {
	c, query := NewPopulatedCache(b.Context(), 1)

	var err error

	b.ResetTimer()

	for range b.N {
		// Search for a single item
		_, err = c.Search(context.Background(), query)

		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCache10SingleItem(b *testing.B) {
	c, query := NewPopulatedCache(b.Context(), 10)

	var err error

	b.ResetTimer()

	for range b.N {
		// Search for a single item
		_, err = c.Search(context.Background(), query)

		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCache100SingleItem(b *testing.B) {
	c, query := NewPopulatedCache(b.Context(), 100)

	var err error

	b.ResetTimer()

	for range b.N {
		// Search for a single item
		_, err = c.Search(context.Background(), query)

		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCache1000SingleItem(b *testing.B) {
	c, query := NewPopulatedCache(b.Context(), 1000)

	var err error

	b.ResetTimer()

	for range b.N {
		// Search for a single item
		_, err = c.Search(context.Background(), query)

		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCache10_000SingleItem(b *testing.B) {
	c, query := NewPopulatedCache(b.Context(), 10_000)

	var err error

	b.ResetTimer()

	for range b.N {
		// Search for a single item
		_, err = c.Search(context.Background(), query)

		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkListQueryLookup benchmarks LIST query performance using the Lookup method,
// which includes the full production path with pending work deduplication logic.
// This provides a more realistic benchmark of end-to-end LIST query performance.
func BenchmarkListQueryLookup(b *testing.B) {
	implementations := cacheImplementations(b)
	cacheSizes := []int{10, 100, 1_000, 10_000}

	for _, impl := range implementations {
		b.Run(impl.name, func(b *testing.B) {
			for _, size := range cacheSizes {
				b.Run(fmt.Sprintf("%d_items", size), func(b *testing.B) {
					// Setup
					cache := impl.factory()
					sst := SST{
						SourceName: "test-source",
						Scope:      "test-scope",
						Type:       "test-type",
					}
					_ = NewPopulatedCacheWithListItems(cache, size, sst)

					b.ResetTimer()
					b.ReportAllocs()

					// Benchmark
					for range b.N {
						hit, _, items, qErr := cache.Lookup(
							b.Context(),
							sst.SourceName,
							sdp.QueryMethod_LIST,
							sst.Scope,
							sst.Type,
							"",
							false, // ignoreCache
						)
						if qErr != nil {
							b.Fatalf("unexpected query error: %v", qErr)
						}
						if !hit {
							b.Fatal("expected cache hit, got miss")
						}
						if len(items) != size {
							b.Fatalf("expected %d items, got %d", size, len(items))
						}
					}
				})
			}
		})
	}
}

// BenchmarkListQueryConcurrent benchmarks LIST query performance under high concurrency.
// This simulates production scenarios where hundreds of goroutines hit the cache simultaneously.
func BenchmarkListQueryConcurrent(b *testing.B) {
	implementations := cacheImplementations(b)

	// Test configuration similar to production
	cacheSize := 5_000 // Similar to production's largest bucket
	concurrencyLevels := []int{10, 50, 100, 250, 500}

	for _, impl := range implementations {
		b.Run(impl.name, func(b *testing.B) {
			for _, concurrency := range concurrencyLevels {
				b.Run(fmt.Sprintf("%d_concurrent", concurrency), func(b *testing.B) {
					// Setup: Create cache with multiple buckets for realistic access patterns
					cache := impl.factory()
					numBuckets := 10 // Multiple buckets to spread queries
					itemsPerBucket := cacheSize / numBuckets
					cacheKeys := NewPopulatedCacheWithMultipleBuckets(cache, itemsPerBucket, numBuckets)

					b.ResetTimer()
					b.ReportAllocs()
					b.SetParallelism(concurrency / runtime.GOMAXPROCS(0)) // Scale to desired concurrency

					// Benchmark: Each goroutine randomly queries one of the buckets
					b.RunParallel(func(pb *testing.PB) {
						for pb.Next() {
							// Randomly select a bucket to query
							bucketIdx := rand.Intn(numBuckets)
							ck := cacheKeys[bucketIdx]

							// Use Lookup() to match production behavior
							hit, _, items, qErr := cache.Lookup(
								b.Context(),
								ck.SST.SourceName,
								sdp.QueryMethod_LIST,
								ck.SST.Scope,
								ck.SST.Type,
								"",
								false, // ignoreCache
							)
							if qErr != nil {
								b.Errorf("unexpected query error: %v", qErr)
								return
							}
							if !hit {
								b.Error("expected cache hit, got miss")
								return
							}
							if len(items) != itemsPerBucket {
								b.Errorf("expected %d items, got %d", itemsPerBucket, len(items))
								return
							}
						}
					})
				})
			}
		})
	}
}

// BenchmarkListQueryConcurrentSameKey benchmarks worst-case contention where all
// goroutines query the same cache key simultaneously. This tests pending work
// deduplication and maximum lock contention.
func BenchmarkListQueryConcurrentSameKey(b *testing.B) {
	implementations := cacheImplementations(b)

	cacheSize := 5_000
	concurrencyLevels := []int{10, 50, 100, 250, 500}

	for _, impl := range implementations {
		b.Run(impl.name, func(b *testing.B) {
			for _, concurrency := range concurrencyLevels {
				b.Run(fmt.Sprintf("%d_concurrent", concurrency), func(b *testing.B) {
					// Setup: Single SST bucket that all goroutines will hit
					cache := impl.factory()
					sst := SST{
						SourceName: "test-source",
						Scope:      "test-scope",
						Type:       "test-type",
					}
					_ = NewPopulatedCacheWithListItems(cache, cacheSize, sst)

					b.ResetTimer()
					b.ReportAllocs()
					b.SetParallelism(concurrency / runtime.GOMAXPROCS(0))

					// Benchmark: All goroutines hit the same key
					b.RunParallel(func(pb *testing.PB) {
						for pb.Next() {
							// Use Lookup() to match production behavior
							hit, _, items, qErr := cache.Lookup(
								b.Context(),
								sst.SourceName,
								sdp.QueryMethod_LIST,
								sst.Scope,
								sst.Type,
								"",
								false, // ignoreCache
							)
							if qErr != nil {
								b.Errorf("unexpected query error: %v", qErr)
								return
							}
							if !hit {
								b.Error("expected cache hit, got miss")
								return
							}
							if len(items) != cacheSize {
								b.Errorf("expected %d items, got %d", cacheSize, len(items))
								return
							}
						}
					})
				})
			}
		})
	}
}

// BenchmarkPendingWorkContention tests cache behavior when many concurrent goroutines
// all call Lookup() for the same cache key simultaneously. This simulates the production
// scenario where hundreds of goroutines wait in pending.Wait() for a single slow
// aggregatedList operation to complete.
func BenchmarkPendingWorkContention(b *testing.B) {
	// Test parameters matching production scenarios
	concurrencyLevels := []int{100, 200, 400, 500}
	fetchDurations := []time.Duration{1 * time.Second, 5 * time.Second, 10 * time.Second}
	resultSizes := []int{100, 1000, 5000}

	for _, impl := range cacheImplementations(b) {
		b.Run(impl.name, func(b *testing.B) {
			for _, concurrency := range concurrencyLevels {
				b.Run(fmt.Sprintf("concurrency=%d", concurrency), func(b *testing.B) {
					for _, fetchDuration := range fetchDurations {
						b.Run(fmt.Sprintf("fetchDuration=%s", fetchDuration), func(b *testing.B) {
							for _, resultSize := range resultSizes {
								b.Run(fmt.Sprintf("resultSize=%d", resultSize), func(b *testing.B) {
									// Run the actual benchmark
									benchmarkPendingWorkContentionScenario(
										b,
										impl.factory,
										concurrency,
										fetchDuration,
										resultSize,
									)
								})
							}
						})
					}
				})
			}
		})
	}
}

// benchmarkPendingWorkContentionScenario runs a single pending work contention scenario
func benchmarkPendingWorkContentionScenario(
	b *testing.B,
	cacheFactory func() Cache,
	concurrency int,
	fetchDuration time.Duration,
	resultSize int,
) {
	b.ReportAllocs()

	// Create a fresh cache for this test
	cache := cacheFactory()
	defer func() {
		if closer, ok := cache.(interface{ Close() error }); ok {
			closer.Close()
		}
	}()

	// Define the shared cache key that all goroutines will use
	sst := SST{
		SourceName: "test-source",
		Scope:      "test-scope-*",
		Type:       "test-type",
	}
	listMethod := sdp.QueryMethod_LIST
	sharedCacheKey := CacheKey{SST: sst, Method: &listMethod}

	// Track timing metrics across all goroutines
	var (
		firstStartTime   time.Time
		firstCompleteTime time.Time
		lastCompleteTime  time.Time
		timingMutex       sync.Mutex
	)

	// Atomic flag to detect the first goroutine (the one that does the work)
	var firstGoroutine atomic.Bool

	// Use a start barrier to ensure all goroutines begin simultaneously
	startBarrier := make(chan struct{})

	b.ResetTimer()

	for range b.N {
		// Clear cache between iterations
		cache.Clear()

		// Reset state
		firstGoroutine.Store(false)
		firstStartTime = time.Time{}
		firstCompleteTime = time.Time{}
		lastCompleteTime = time.Time{}

		var wg sync.WaitGroup
		wg.Add(concurrency)

		// Spawn all goroutines
		for range concurrency {
			go func() {
				defer wg.Done()

				// Wait for start signal to ensure simultaneous execution
				<-startBarrier

				startTime := time.Now()

				// Call Lookup - this is where the contention happens
				hit, _, items, qErr := cache.Lookup(
					b.Context(),
					sst.SourceName,
					sdp.QueryMethod_LIST,
					sst.Scope,
					sst.Type,
					"",
					false, // ignoreCache
				)

				endTime := time.Now()

				// Check if this goroutine was the first one (the worker)
				isFirst := firstGoroutine.CompareAndSwap(false, true)

				if isFirst {
					// This goroutine got the cache miss and needs to do the work
					if hit {
						b.Errorf("First goroutine should get cache miss, got hit")
						return
					}

					// Record when work started
					timingMutex.Lock()
					firstStartTime = startTime
					timingMutex.Unlock()

					// Simulate slow fetch operation (like aggregatedList)
					time.Sleep(fetchDuration)

					// Store items in cache (simulating results from aggregatedList)
					// Note: The first StoreItem() will call pending.Complete() and wake waiters
					for itemIdx := range resultSize {
						item := GenerateRandomItem()
						item.Scope = sst.Scope
						item.Type = sst.Type
						item.Metadata.SourceName = sst.SourceName
						item.GetAttributes().Set("name", fmt.Sprintf("item-%d", itemIdx))

						cache.StoreItem(b.Context(), item, CacheDuration, sharedCacheKey)
					}

					// Record when work completed
					timingMutex.Lock()
					firstCompleteTime = time.Now()
					timingMutex.Unlock()
				} else {
					// This goroutine should have waited in pending.Wait() and then got a cache hit
					// Note: It might get partial results if it wakes up while the first goroutine
					// is still storing items (since StoreItem calls pending.Complete on first item)
					if !hit {
						b.Errorf("Waiting goroutine should get cache hit after pending work completes, got miss")
						return
					}
					if qErr != nil {
						b.Errorf("Waiting goroutine got error: %v", qErr)
						return
					}
					if len(items) == 0 {
						b.Errorf("Waiting goroutine got cache hit but no items")
						return
					}
					// Don't check exact count - waiters may get partial results
				}

				// Track when each goroutine completes
				timingMutex.Lock()
				if lastCompleteTime.IsZero() || endTime.After(lastCompleteTime) {
					lastCompleteTime = endTime
				}
				timingMutex.Unlock()
			}()
		}

		// Release all goroutines simultaneously
		close(startBarrier)

		// Wait for all goroutines to complete
		wg.Wait()

		// Calculate and report metrics for this iteration
		if !firstStartTime.IsZero() && !firstCompleteTime.IsZero() && !lastCompleteTime.IsZero() {
			workDuration := firstCompleteTime.Sub(firstStartTime)
			totalDuration := lastCompleteTime.Sub(firstStartTime)
			maxWaitTime := lastCompleteTime.Sub(firstCompleteTime)

			// Report metrics
			b.ReportMetric(workDuration.Seconds(), "work_duration_sec")
			b.ReportMetric(totalDuration.Seconds(), "total_duration_sec")
			b.ReportMetric(maxWaitTime.Seconds(), "max_wait_sec")
			b.ReportMetric(float64(concurrency-1), "waiting_goroutines")

			// Calculate efficiency: ideally, waiters should return immediately after work completes
			// A ratio close to 1.0 means waiters waited approximately the work duration
			waitToWorkRatio := totalDuration.Seconds() / workDuration.Seconds()
			b.ReportMetric(waitToWorkRatio, "wait_to_work_ratio")
		}

		// Recreate start barrier for next iteration
		startBarrier = make(chan struct{})
	}

	b.StopTimer()
}

// BenchmarkConcurrentMultiKeyWrites tests cache behavior when many concurrent goroutines
// call Lookup() with DIFFERENT cache keys, all get cache misses, and all write results
// concurrently to the same BoltDB file. This simulates the production scenario where
// a wildcard query is expanded into 620+ separate queries with different scopes.
func BenchmarkConcurrentMultiKeyWrites(b *testing.B) {
	// Test parameters matching production scenarios
	concurrencyLevels := []int{100, 200, 400, 600}
	itemsPerGoroutine := []int{10, 100, 500}
	fetchDurations := []time.Duration{100 * time.Millisecond, 1 * time.Second, 5 * time.Second}

	for _, impl := range cacheImplementations(b) {
		b.Run(impl.name, func(b *testing.B) {
			for _, concurrency := range concurrencyLevels {
				b.Run(fmt.Sprintf("concurrency=%d", concurrency), func(b *testing.B) {
					for _, itemsPerGoroutine := range itemsPerGoroutine {
						b.Run(fmt.Sprintf("itemsPerGoroutine=%d", itemsPerGoroutine), func(b *testing.B) {
							for _, fetchDuration := range fetchDurations {
								b.Run(fmt.Sprintf("fetchDuration=%s", fetchDuration), func(b *testing.B) {
									// Run the actual benchmark
									benchmarkConcurrentMultiKeyWritesScenario(
										b,
										impl.factory,
										concurrency,
										itemsPerGoroutine,
										fetchDuration,
									)
								})
							}
						})
					}
				})
			}
		})
	}
}

// benchmarkConcurrentMultiKeyWritesScenario runs a single concurrent multi-key write scenario
func benchmarkConcurrentMultiKeyWritesScenario(
	b *testing.B,
	cacheFactory func() Cache,
	concurrency int,
	itemsPerGoroutine int,
	fetchDuration time.Duration,
) {
	b.ReportAllocs()

	// Create a fresh cache for this test
	cache := cacheFactory()
	defer func() {
		if closer, ok := cache.(interface{ Close() error }); ok {
			closer.Close()
		}
	}()

	// Generate unique cache keys for each goroutine (different scopes)
	cacheKeys := make([]CacheKey, concurrency)
	listMethod := sdp.QueryMethod_LIST
	for i := range concurrency {
		cacheKeys[i] = CacheKey{
			SST: SST{
				SourceName: "test-source",
				Scope:      fmt.Sprintf("scope-%d", i), // Different scope = different cache key
				Type:       "test-type",
			},
			Method: &listMethod,
		}
	}

	// Track timing metrics
	var (
		goroutineStartTimes   []time.Time
		goroutineEndTimes     []time.Time
		timesMutex            sync.Mutex
		totalStoreItemCalls   atomic.Int64
	)

	// Use a start barrier to ensure all goroutines begin simultaneously
	startBarrier := make(chan struct{})

	b.ResetTimer()

	for range b.N {
		// Clear cache between iterations
		cache.Clear()

		// Reset metrics
		goroutineStartTimes = make([]time.Time, 0, concurrency)
		goroutineEndTimes = make([]time.Time, 0, concurrency)
		totalStoreItemCalls.Store(0)

		var wg sync.WaitGroup
		wg.Add(concurrency)

		// Spawn all goroutines
		for g := range concurrency {
			goroutineIdx := g
			go func() {
				defer wg.Done()

				// Wait for start signal to ensure simultaneous execution
				<-startBarrier

				startTime := time.Now()

				// Track start time
				timesMutex.Lock()
				goroutineStartTimes = append(goroutineStartTimes, startTime)
				timesMutex.Unlock()

				// Call Lookup with unique cache key - should be a cache miss
				myCacheKey := cacheKeys[goroutineIdx]
				hit, _, _, qErr := cache.Lookup(
					b.Context(),
					myCacheKey.SST.SourceName,
					sdp.QueryMethod_LIST,
					myCacheKey.SST.Scope,
					myCacheKey.SST.Type,
					"",
					false, // ignoreCache
				)

				if hit {
					b.Errorf("Expected cache miss for goroutine %d, got hit", goroutineIdx)
					return
				}
				if qErr != nil {
					b.Errorf("Unexpected error for goroutine %d: %v", goroutineIdx, qErr)
					return
				}

				// Simulate slow fetch operation (like aggregatedList API call)
				time.Sleep(fetchDuration)

				// Store multiple items (simulating API results)
				for itemIdx := range itemsPerGoroutine {
					item := GenerateRandomItem()
					item.Scope = myCacheKey.SST.Scope
					item.Type = myCacheKey.SST.Type
					item.Metadata.SourceName = myCacheKey.SST.SourceName
					item.GetAttributes().Set("name", fmt.Sprintf("goroutine-%d-item-%d", goroutineIdx, itemIdx))

					cache.StoreItem(b.Context(), item, CacheDuration, myCacheKey)
					totalStoreItemCalls.Add(1)
				}

				endTime := time.Now()

				// Track end time
				timesMutex.Lock()
				goroutineEndTimes = append(goroutineEndTimes, endTime)
				timesMutex.Unlock()
			}()
		}

		// Release all goroutines simultaneously
		close(startBarrier)

		// Wait for all goroutines to complete
		wg.Wait()

		// Calculate and report metrics for this iteration
		if len(goroutineStartTimes) > 0 && len(goroutineEndTimes) > 0 {
			// Find earliest start and latest end
			earliestStart := goroutineStartTimes[0]
			latestEnd := goroutineEndTimes[0]

			for _, t := range goroutineStartTimes {
				if t.Before(earliestStart) {
					earliestStart = t
				}
			}
			for _, t := range goroutineEndTimes {
				if t.After(latestEnd) {
					latestEnd = t
				}
			}

			totalDuration := latestEnd.Sub(earliestStart)
			totalWrites := totalStoreItemCalls.Load()
			writeThroughput := float64(totalWrites) / totalDuration.Seconds()

			// Calculate average goroutine duration
			var totalGoroutineDuration time.Duration
			for idx := range goroutineStartTimes {
				if idx < len(goroutineEndTimes) {
					totalGoroutineDuration += goroutineEndTimes[idx].Sub(goroutineStartTimes[idx])
				}
			}
			avgGoroutineDuration := totalGoroutineDuration / time.Duration(len(goroutineStartTimes))

			// Report metrics
			b.ReportMetric(totalDuration.Seconds(), "total_duration_sec")
			b.ReportMetric(avgGoroutineDuration.Seconds(), "avg_goroutine_sec")
			b.ReportMetric(float64(concurrency), "concurrent_writers")
			b.ReportMetric(float64(totalWrites), "total_store_calls")
			b.ReportMetric(writeThroughput, "writes_per_sec")
		}

		// Recreate start barrier for next iteration
		startBarrier = make(chan struct{})
	}

	b.StopTimer()
}
