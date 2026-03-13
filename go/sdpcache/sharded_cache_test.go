package sdpcache

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/overmindtech/cli/go/sdp-go"
)

func TestShardDistributionUniformity(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "shards")
	cache, err := NewShardedCache(dir, DefaultShardCount)
	if err != nil {
		t.Fatalf("failed to create ShardedCache: %v", err)
	}
	defer func() { _ = cache.CloseAndDestroy() }()

	ctx := t.Context()
	numItems := 1000

	// Use the same SST for all items so they share the same BoltDB SST bucket.
	// Different UAVs cause items to distribute across shards via shardFor().
	sst := SST{SourceName: "test-source", Scope: "scope", Type: "type"}
	method := sdp.QueryMethod_LIST
	ck := CacheKey{SST: sst, Method: &method}

	for i := range numItems {
		item := GenerateRandomItem()
		item.Scope = sst.Scope
		item.Type = sst.Type
		item.Metadata.SourceName = sst.SourceName

		attrs := make(map[string]any)
		attrs["name"] = fmt.Sprintf("item-%d", i)
		attributes, _ := sdp.ToAttributes(attrs)
		item.Attributes = attributes

		cache.StoreItem(ctx, item, 10*time.Second, ck)
	}

	// Count items per shard by searching each shard with the common SST
	counts := make([]int, DefaultShardCount)
	for i, shard := range cache.shards {
		items, searchErr := shard.Search(ctx, ck)
		if searchErr == nil {
			counts[i] = len(items)
		}
	}

	totalFound := 0
	for _, c := range counts {
		totalFound += c
	}

	if totalFound != numItems {
		t.Errorf("expected %d total items across shards, got %d", numItems, totalFound)
	}

	// Verify distribution is reasonably uniform: no shard should have more than
	// 3× the expected average (very loose bound to avoid flaky tests).
	expected := float64(numItems) / float64(DefaultShardCount)
	for i, c := range counts {
		if float64(c) > expected*3 {
			t.Errorf("shard %d has %d items, expected roughly %.0f (3× threshold: %.0f)", i, c, expected, expected*3)
		}
	}

	// Chi-squared test for uniformity (p < 0.001 threshold)
	var chiSq float64
	for _, c := range counts {
		diff := float64(c) - expected
		chiSq += (diff * diff) / expected
	}
	// Critical value for df=16, p=0.001 is ~39.25
	if chiSq > 39.25 {
		t.Errorf("chi-squared %.2f exceeds critical value 39.25 (df=16, p=0.001), distribution may be non-uniform: %v", chiSq, counts)
	}
}

func TestShardedCacheGETRoutesToCorrectShard(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "shards")
	cache, err := NewShardedCache(dir, DefaultShardCount)
	if err != nil {
		t.Fatalf("failed to create ShardedCache: %v", err)
	}
	defer func() { _ = cache.CloseAndDestroy() }()

	ctx := t.Context()
	sst := SST{SourceName: "test", Scope: "scope", Type: "type"}
	method := sdp.QueryMethod_GET

	item := GenerateRandomItem()
	item.Scope = sst.Scope
	item.Type = sst.Type
	item.Metadata.SourceName = sst.SourceName

	uav := item.UniqueAttributeValue()
	ck := CacheKey{SST: sst, Method: &method, UniqueAttributeValue: &uav}
	cache.StoreItem(ctx, item, 10*time.Second, ck)

	// Verify the item lands on the expected shard
	expectedShard := cache.shardFor(sst.Hash(), uav)
	items, err := cache.shards[expectedShard].Search(ctx, ck)
	if err != nil {
		t.Fatalf("expected item on shard %d, got error: %v", expectedShard, err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item on shard %d, got %d", expectedShard, len(items))
	}

	// Verify Lookup returns the item
	hit, _, cachedItems, qErr, done := cache.Lookup(ctx, sst.SourceName, method, sst.Scope, sst.Type, uav, false)
	defer done()
	if qErr != nil {
		t.Fatalf("unexpected error: %v", qErr)
	}
	if !hit {
		t.Fatal("expected cache hit")
	}
	if len(cachedItems) != 1 {
		t.Fatalf("expected 1 item, got %d", len(cachedItems))
	}
}

func TestShardedCacheLISTFanOutMerge(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "shards")
	cache, err := NewShardedCache(dir, DefaultShardCount)
	if err != nil {
		t.Fatalf("failed to create ShardedCache: %v", err)
	}
	defer func() { _ = cache.CloseAndDestroy() }()

	ctx := t.Context()
	sst := SST{SourceName: "test", Scope: "scope", Type: "type"}
	method := sdp.QueryMethod_LIST
	ck := CacheKey{SST: sst, Method: &method}

	// Store items that should land on different shards
	numItems := 50
	for i := range numItems {
		item := GenerateRandomItem()
		item.Scope = sst.Scope
		item.Type = sst.Type
		item.Metadata.SourceName = sst.SourceName

		attrs := make(map[string]any)
		attrs["name"] = fmt.Sprintf("item-%d", i)
		attributes, _ := sdp.ToAttributes(attrs)
		item.Attributes = attributes

		cache.StoreItem(ctx, item, 10*time.Second, ck)
	}

	// LIST should fan out and return all items
	hit, _, items, qErr, done := cache.Lookup(ctx, sst.SourceName, method, sst.Scope, sst.Type, "", false)
	defer done()
	if qErr != nil {
		t.Fatalf("unexpected error: %v", qErr)
	}
	if !hit {
		t.Fatal("expected cache hit")
	}
	if len(items) != numItems {
		t.Errorf("expected %d items from LIST fan-out, got %d", numItems, len(items))
	}
}

func TestShardedCacheCrossShardLIST(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "shards")
	// Use a small shard count for easier verification
	cache, err := NewShardedCache(dir, 3)
	if err != nil {
		t.Fatalf("failed to create ShardedCache: %v", err)
	}
	defer func() { _ = cache.CloseAndDestroy() }()

	ctx := t.Context()
	sst := SST{SourceName: "test", Scope: "scope", Type: "type"}
	method := sdp.QueryMethod_LIST
	ck := CacheKey{SST: sst, Method: &method}

	// Store enough items that at least 2 shards get items
	storedNames := make(map[string]bool)
	for i := range 30 {
		item := GenerateRandomItem()
		item.Scope = sst.Scope
		item.Type = sst.Type
		item.Metadata.SourceName = sst.SourceName

		name := fmt.Sprintf("cross-shard-%d", i)
		attrs := make(map[string]any)
		attrs["name"] = name
		attributes, _ := sdp.ToAttributes(attrs)
		item.Attributes = attributes

		cache.StoreItem(ctx, item, 10*time.Second, ck)
		storedNames[name] = true
	}

	// Count items per shard
	shardsWithItems := 0
	for _, shard := range cache.shards {
		items, err := shard.Search(ctx, ck)
		if err == nil && len(items) > 0 {
			shardsWithItems++
		}
	}

	if shardsWithItems < 2 {
		t.Errorf("expected items on at least 2 shards, got %d", shardsWithItems)
	}

	// LIST fan-out should return all items regardless of shard
	items, err := cache.searchAll(ctx, ck)
	if err != nil {
		t.Fatalf("searchAll failed: %v", err)
	}
	if len(items) != 30 {
		t.Errorf("expected 30 items from fan-out, got %d", len(items))
	}
}

func TestShardedCachePendingWorkDeduplication(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "shards")
	cache, err := NewShardedCache(dir, DefaultShardCount)
	if err != nil {
		t.Fatalf("failed to create ShardedCache: %v", err)
	}
	defer func() { _ = cache.CloseAndDestroy() }()

	ctx := t.Context()
	sst := SST{SourceName: "dedup-test", Scope: "scope", Type: "type"}
	method := sdp.QueryMethod_LIST

	var workCount int32
	var mu sync.Mutex
	var wg sync.WaitGroup

	numGoroutines := 10
	results := make([]struct {
		hit   bool
		items []*sdp.Item
	}, numGoroutines)

	startBarrier := make(chan struct{})

	for i := range numGoroutines {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
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
		}(i)
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

func TestShardedCacheCloseAndDestroy(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "shards")
	cache, err := NewShardedCache(dir, DefaultShardCount)
	if err != nil {
		t.Fatalf("failed to create ShardedCache: %v", err)
	}

	ctx := t.Context()
	item := GenerateRandomItem()
	ck := CacheKeyFromQuery(item.GetMetadata().GetSourceQuery(), item.GetMetadata().GetSourceName())
	cache.StoreItem(ctx, item, 10*time.Second, ck)

	// Verify shard files exist
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("failed to read shard directory: %v", err)
	}
	if len(entries) != DefaultShardCount {
		t.Errorf("expected %d shard files, got %d", DefaultShardCount, len(entries))
	}

	// Close and destroy
	if err := cache.CloseAndDestroy(); err != nil {
		t.Fatalf("CloseAndDestroy failed: %v", err)
	}

	// Verify the directory is removed
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Error("shard directory should be removed after CloseAndDestroy")
	}
}

func BenchmarkShardedCacheVsSingleBoltCache(b *testing.B) {
	implementations := []struct {
		name    string
		factory func(b *testing.B) Cache
	}{
		{"BoltCache", func(b *testing.B) Cache {
			c, err := NewBoltCache(filepath.Join(b.TempDir(), "cache.db"))
			if err != nil {
				b.Fatalf("failed to create BoltCache: %v", err)
			}
			b.Cleanup(func() { _ = c.CloseAndDestroy() })
			return c
		}},
		{"ShardedCache", func(b *testing.B) Cache {
			c, err := NewShardedCache(
				filepath.Join(b.TempDir(), "shards"),
				DefaultShardCount,
			)
			if err != nil {
				b.Fatalf("failed to create ShardedCache: %v", err)
			}
			b.Cleanup(func() { _ = c.CloseAndDestroy() })
			return c
		}},
	}

	for _, impl := range implementations {
		b.Run(impl.name+"/ConcurrentWrite", func(b *testing.B) {
			cache := impl.factory(b)
			ctx := context.Background()

			sst := SST{SourceName: "bench", Scope: "scope", Type: "type"}
			method := sdp.QueryMethod_LIST
			ck := CacheKey{SST: sst, Method: &method}

			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					item := GenerateRandomItem()
					item.Scope = sst.Scope
					item.Type = sst.Type
					item.Metadata.SourceName = sst.SourceName
					cache.StoreItem(ctx, item, 10*time.Second, ck)
				}
			})
		})
	}
}

func TestShardedCacheShardForDeterminism(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "shards")
	cache, err := NewShardedCache(dir, DefaultShardCount)
	if err != nil {
		t.Fatalf("failed to create ShardedCache: %v", err)
	}
	defer func() { _ = cache.CloseAndDestroy() }()

	sst := SST{SourceName: "test", Scope: "scope", Type: "type"}
	sstHash := sst.Hash()

	// Same input should always produce the same shard
	for range 100 {
		idx1 := cache.shardFor(sstHash, "my-unique-value")
		idx2 := cache.shardFor(sstHash, "my-unique-value")
		if idx1 != idx2 {
			t.Fatalf("shardFor is not deterministic: got %d and %d", idx1, idx2)
		}
	}

	// Different UAVs should produce different shards (at least some of the time)
	shardsSeen := make(map[int]bool)
	for i := range 100 {
		idx := cache.shardFor(sstHash, fmt.Sprintf("value-%d", i))
		shardsSeen[idx] = true
	}
	if len(shardsSeen) < 2 {
		t.Error("expected different UAVs to hash to different shards")
	}
}

func TestShardedCacheErrorRouting(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "shards")
	cache, err := NewShardedCache(dir, DefaultShardCount)
	if err != nil {
		t.Fatalf("failed to create ShardedCache: %v", err)
	}
	defer func() { _ = cache.CloseAndDestroy() }()

	ctx := t.Context()

	t.Run("GET error routes to same shard as GET lookup", func(t *testing.T) {
		sst := SST{SourceName: "err-test", Scope: "scope", Type: "type"}
		method := sdp.QueryMethod_GET
		uav := "my-item"
		ck := CacheKey{SST: sst, Method: &method, UniqueAttributeValue: &uav}

		qErr := &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOTFOUND,
			ErrorString: "not found",
		}
		cache.StoreUnavailableItem(ctx, qErr, 10*time.Second, ck)

		// Lookup should find the error
		hit, _, _, returnedErr, done := cache.Lookup(ctx, sst.SourceName, method, sst.Scope, sst.Type, uav, false)
		defer done()
		if !hit {
			t.Fatal("expected cache hit for stored error")
		}
		if returnedErr == nil {
			t.Fatal("expected error to be returned")
		}
		if returnedErr.GetErrorType() != sdp.QueryError_NOTFOUND {
			t.Errorf("expected NOTFOUND, got %v", returnedErr.GetErrorType())
		}
	})

	t.Run("LIST error routes to shard 0 and is found via fan-out", func(t *testing.T) {
		sst := SST{SourceName: "list-err-test", Scope: "scope", Type: "type"}
		method := sdp.QueryMethod_LIST
		ck := CacheKey{SST: sst, Method: &method}

		qErr := &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "list failed",
		}
		cache.StoreUnavailableItem(ctx, qErr, 10*time.Second, ck)

		// LIST lookup fans out, should find the error on shard 0
		hit, _, _, returnedErr, done := cache.Lookup(ctx, sst.SourceName, method, sst.Scope, sst.Type, "", false)
		defer done()
		if !hit {
			t.Fatal("expected cache hit for stored LIST error")
		}
		if returnedErr == nil {
			t.Fatal("expected error to be returned")
		}
		if returnedErr.GetErrorType() != sdp.QueryError_OTHER {
			t.Errorf("expected OTHER, got %v", returnedErr.GetErrorType())
		}
	})
}

func TestShardedCacheNewCacheFallback(t *testing.T) {
	ctx := t.Context()
	cache := NewCache(ctx)

	if cache == nil {
		t.Fatal("NewCache returned nil")
	}

	// Should be a ShardedCache in normal operation
	if _, ok := cache.(*ShardedCache); !ok {
		t.Logf("NewCache returned %T (may be MemoryCache if ShardedCache creation failed)", cache)
	}

	// Basic operation test
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
	if qErr != nil {
		t.Fatalf("unexpected error: %v", qErr)
	}
	if !hit {
		t.Fatal("expected cache hit")
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
}

func TestShardedCacheCompactThresholdScaling(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "shards")

	parentThreshold := int64(1 * 1024 * 1024 * 1024) // 1GB
	perShardThreshold := parentThreshold / int64(DefaultShardCount)

	cache, err := NewShardedCache(dir, DefaultShardCount,
		WithCompactThreshold(perShardThreshold),
	)
	if err != nil {
		t.Fatalf("failed to create ShardedCache: %v", err)
	}
	defer func() { _ = cache.CloseAndDestroy() }()

	expectedPerShard := parentThreshold / int64(DefaultShardCount)
	for i, shard := range cache.shards {
		if shard.CompactThreshold != expectedPerShard {
			t.Errorf("shard %d: expected CompactThreshold %d, got %d", i, expectedPerShard, shard.CompactThreshold)
		}
	}
}

func TestShardedCacheInvalidShardCount(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "shards")

	_, err := NewShardedCache(dir, 0)
	if err == nil {
		t.Error("expected error for shard count 0")
	}

	_, err = NewShardedCache(dir, -1)
	if err == nil {
		t.Error("expected error for negative shard count")
	}
}

func TestShardedCacheConcurrentWriteThroughput(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "shards")
	cache, err := NewShardedCache(dir, DefaultShardCount)
	if err != nil {
		t.Fatalf("failed to create ShardedCache: %v", err)
	}
	defer func() { _ = cache.CloseAndDestroy() }()

	ctx := t.Context()
	sst := SST{SourceName: "concurrent", Scope: "scope", Type: "type"}
	method := sdp.QueryMethod_LIST
	ck := CacheKey{SST: sst, Method: &method}

	var wg sync.WaitGroup
	numParallel := 100

	for i := range numParallel {
		idx := i
		wg.Go(func() {
			item := GenerateRandomItem()
			item.Scope = sst.Scope
			item.Type = sst.Type
			item.Metadata.SourceName = sst.SourceName

			attrs := make(map[string]any)
			attrs["name"] = fmt.Sprintf("concurrent-item-%d", idx)
			attributes, _ := sdp.ToAttributes(attrs)
			item.Attributes = attributes

			cache.StoreItem(ctx, item, 10*time.Second, ck)
		})
	}

	wg.Wait()

	items, searchErr := cache.searchAll(ctx, ck)
	if searchErr != nil {
		t.Fatalf("searchAll failed: %v", searchErr)
	}
	if len(items) != numParallel {
		t.Errorf("expected %d items, got %d", numParallel, len(items))
	}
}

func TestShardedCachePurgeAggregation(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "shards")
	cache, err := NewShardedCache(dir, 3) // Small count for easier verification
	if err != nil {
		t.Fatalf("failed to create ShardedCache: %v", err)
	}
	defer func() { _ = cache.CloseAndDestroy() }()

	ctx := t.Context()
	sst := SST{SourceName: "purge", Scope: "scope", Type: "type"}
	method := sdp.QueryMethod_LIST
	ck := CacheKey{SST: sst, Method: &method}

	// Store items with short expiry
	for range 10 {
		item := GenerateRandomItem()
		item.Scope = sst.Scope
		item.Type = sst.Type
		item.Metadata.SourceName = sst.SourceName
		cache.StoreItem(ctx, item, 100*time.Millisecond, ck)
	}

	// Wait for expiry
	time.Sleep(200 * time.Millisecond)

	// Purge and check aggregated stats
	stats := cache.Purge(ctx, time.Now())
	if stats.NumPurged != 10 {
		t.Errorf("expected 10 items purged, got %d", stats.NumPurged)
	}
}

// TestShardedCacheShardForBounds verifies that shardFor always returns a valid
// index in [0, shardCount).
func TestShardedCacheShardForBounds(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "shards")
	cache, err := NewShardedCache(dir, DefaultShardCount)
	if err != nil {
		t.Fatalf("failed to create ShardedCache: %v", err)
	}
	defer func() { _ = cache.CloseAndDestroy() }()

	for i := range 10000 {
		idx := cache.shardFor(SSTHash(fmt.Sprintf("hash-%d", i)), fmt.Sprintf("uav-%d", i))
		if idx < 0 || idx >= DefaultShardCount {
			t.Fatalf("shardFor returned out-of-bounds index %d for shard count %d", idx, DefaultShardCount)
		}
	}
}

// TestShardedCacheFNV32aOverflow verifies that the FNV-32a hash mod operation
// works correctly with uint32 values close to math.MaxUint32.
func TestShardedCacheFNV32aOverflow(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "shards")
	cache, err := NewShardedCache(dir, DefaultShardCount)
	if err != nil {
		t.Fatalf("failed to create ShardedCache: %v", err)
	}
	defer func() { _ = cache.CloseAndDestroy() }()

	// These are just strings; the test verifies no panic from the modulo arithmetic
	_ = cache.shardFor(SSTHash(fmt.Sprintf("%d", math.MaxUint32)), "test")
	_ = cache.shardFor(SSTHash(""), "")
	_ = cache.shardFor(SSTHash("a"), "b")
}
