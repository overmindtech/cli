package sdpcache

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/tracing"
	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// DefaultShardCount is the number of independent BoltDB shards. 17 is prime
// (avoids hash collision clustering) and distributes ~345 stdlib goroutines to
// ~20 per shard, making BoltDB's single-writer lock no longer a bottleneck.
const DefaultShardCount = 17

// ShardedCache implements the Cache interface by distributing entries across N
// independent BoltCache instances. Shard selection uses FNV-32a of the item
// identity (SSTHash + UniqueAttributeValue), so writes within a single adapter
// type (e.g. DNS in stdlib) spread evenly across all shards.
//
// GET queries route to exactly one shard. LIST/SEARCH queries fan out to all
// shards in parallel and merge results. pendingWork deduplication lives at the
// ShardedCache level to prevent duplicate API calls across the fan-out.
type ShardedCache struct {
	shards []*BoltCache
	dir    string

	// pendingWork lives at the ShardedCache level so that deduplication spans
	// the entire cache, not individual shards.
	pending *pendingWork
}

var _ Cache = (*ShardedCache)(nil)

// NewShardedCache creates N BoltCache instances in dir (shard-00.db through
// shard-{N-1}.db) using goroutine fan-out to avoid N× startup latency.
func NewShardedCache(dir string, shardCount int, opts ...BoltCacheOption) (*ShardedCache, error) {
	if shardCount <= 0 {
		return nil, fmt.Errorf("shard count must be positive, got %d", shardCount)
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create shard directory: %w", err)
	}

	shards := make([]*BoltCache, shardCount)
	errs := make([]error, shardCount)

	var wg sync.WaitGroup
	for i := range shardCount {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			path := filepath.Join(dir, fmt.Sprintf("shard-%02d.db", idx))
			c, err := NewBoltCache(path, opts...)
			if err != nil {
				errs[idx] = fmt.Errorf("shard %d: %w", idx, err)
				return
			}
			shards[idx] = c
		}(i)
	}
	wg.Wait()

	// If any shard failed, close the ones that succeeded and return the error.
	for _, err := range errs {
		if err != nil {
			for _, s := range shards {
				if s != nil {
					_ = s.CloseAndDestroy()
				}
			}
			return nil, err
		}
	}

	return &ShardedCache{
		shards:  shards,
		dir:     dir,
		pending: newPendingWork(),
	}, nil
}

// shardFor returns the shard index for a given item identity.
func (sc *ShardedCache) shardFor(sstHash SSTHash, uav string) int {
	h := fnv.New32a()
	_, _ = h.Write([]byte(sstHash))
	_, _ = h.Write([]byte(uav))
	return int(h.Sum32()) % len(sc.shards)
}

// Lookup performs a cache lookup, routing GET queries to a single shard and
// LIST/SEARCH queries to all shards via parallel fan-out.
func (sc *ShardedCache) Lookup(ctx context.Context, srcName string, method sdp.QueryMethod, scope string, typ string, query string, ignoreCache bool) (bool, CacheKey, []*sdp.Item, *sdp.QueryError, func()) {
	ctx, span := tracing.Tracer().Start(ctx, "ShardedCache.Lookup",
		trace.WithAttributes(
			attribute.String("ovm.cache.sourceName", srcName),
			attribute.String("ovm.cache.method", method.String()),
			attribute.String("ovm.cache.scope", scope),
			attribute.String("ovm.cache.type", typ),
			attribute.String("ovm.cache.query", query),
			attribute.Bool("ovm.cache.ignoreCache", ignoreCache),
			attribute.Int("ovm.cache.shardCount", len(sc.shards)),
		),
	)
	defer span.End()

	ck := CacheKeyFromParts(srcName, method, scope, typ, query)

	if ignoreCache {
		span.SetAttributes(
			attribute.String("ovm.cache.result", "ignore cache"),
			attribute.Bool("ovm.cache.hit", false),
		)
		return false, ck, nil, nil, noopDone
	}

	items, err := sc.searchByKey(ctx, ck)

	if err != nil {
		var qErr *sdp.QueryError
		if errors.Is(err, ErrCacheNotFound) {
			shouldWork, entry := sc.pending.StartWork(ck.String())
			if shouldWork {
				span.SetAttributes(
					attribute.String("ovm.cache.result", "cache miss"),
					attribute.Bool("ovm.cache.hit", false),
					attribute.Bool("ovm.cache.workPending", false),
				)
				return false, ck, nil, nil, sc.createDoneFunc(ck)
			}

			pendingWaitStart := time.Now()
			ok := sc.pending.Wait(ctx, entry)
			pendingWaitDuration := time.Since(pendingWaitStart)
			span.SetAttributes(
				attribute.Float64("ovm.cache.pendingWaitDuration_ms", float64(pendingWaitDuration.Milliseconds())),
				attribute.Bool("ovm.cache.pendingWaitSuccess", ok),
			)

			if !ok {
				span.SetAttributes(
					attribute.String("ovm.cache.result", "pending work cancelled or timeout"),
					attribute.Bool("ovm.cache.hit", false),
				)
				return false, ck, nil, nil, noopDone
			}

			items, recheckErr := sc.searchByKey(ctx, ck)
			if recheckErr != nil {
				if errors.Is(recheckErr, ErrCacheNotFound) {
					span.SetAttributes(
						attribute.String("ovm.cache.result", "pending work completed but cache still empty"),
						attribute.Bool("ovm.cache.hit", false),
					)
					return false, ck, nil, nil, noopDone
				}
				var recheckQErr *sdp.QueryError
				if errors.As(recheckErr, &recheckQErr) {
					span.SetAttributes(
						attribute.String("ovm.cache.result", "cache hit from pending work: error"),
						attribute.Bool("ovm.cache.hit", true),
					)
					return true, ck, nil, recheckQErr, noopDone
				}
				span.SetAttributes(
					attribute.String("ovm.cache.result", "unexpected error on re-check"),
					attribute.Bool("ovm.cache.hit", false),
				)
				return false, ck, nil, nil, noopDone
			}

			span.SetAttributes(
				attribute.String("ovm.cache.result", "cache hit from pending work"),
				attribute.Int("ovm.cache.numItems", len(items)),
				attribute.Bool("ovm.cache.hit", true),
			)
			return true, ck, items, nil, noopDone
		} else if errors.As(err, &qErr) {
			if qErr.GetErrorType() == sdp.QueryError_NOTFOUND {
				span.SetAttributes(attribute.String("ovm.cache.result", "cache hit: item not found"))
			} else {
				span.SetAttributes(
					attribute.String("ovm.cache.result", "cache hit: QueryError"),
					attribute.String("ovm.cache.error", err.Error()),
				)
			}
			span.SetAttributes(attribute.Bool("ovm.cache.hit", true))
			return true, ck, nil, qErr, noopDone
		} else {
			qErr = &sdp.QueryError{
				ErrorType:   sdp.QueryError_OTHER,
				ErrorString: err.Error(),
				Scope:       scope,
				SourceName:  srcName,
				ItemType:    typ,
			}
			span.SetAttributes(
				attribute.String("ovm.cache.error", err.Error()),
				attribute.String("ovm.cache.result", "cache hit: unknown QueryError"),
				attribute.Bool("ovm.cache.hit", true),
			)
			return true, ck, nil, qErr, noopDone
		}
	}

	if method == sdp.QueryMethod_GET {
		if len(items) < 2 {
			span.SetAttributes(
				attribute.String("ovm.cache.result", "cache hit: 1 item"),
				attribute.Int("ovm.cache.numItems", len(items)),
				attribute.Bool("ovm.cache.hit", true),
			)
			return true, ck, items, nil, noopDone
		}
		span.SetAttributes(
			attribute.String("ovm.cache.result", "cache returned >1 value, purging and continuing"),
			attribute.Int("ovm.cache.numItems", len(items)),
			attribute.Bool("ovm.cache.hit", false),
		)
		sc.Delete(ck)
		return false, ck, nil, nil, noopDone
	}

	span.SetAttributes(
		attribute.String("ovm.cache.result", "cache hit: multiple items"),
		attribute.Int("ovm.cache.numItems", len(items)),
		attribute.Bool("ovm.cache.hit", true),
	)
	return true, ck, items, nil, noopDone
}

// searchByKey routes GET queries to a single shard and LIST/SEARCH/unspecified
// queries to all shards via fan-out.
func (sc *ShardedCache) searchByKey(ctx context.Context, ck CacheKey) ([]*sdp.Item, error) {
	span := trace.SpanFromContext(ctx)

	if ck.UniqueAttributeValue != nil {
		idx := sc.shardFor(ck.SST.Hash(), *ck.UniqueAttributeValue)
		span.SetAttributes(
			attribute.Int("ovm.cache.shardIndex", idx),
			attribute.Bool("ovm.cache.fanOut", false),
		)
		return sc.shards[idx].Search(ctx, ck)
	}

	return sc.searchAll(ctx, ck)
}

// searchAll fans out a search to all shards in parallel and merges results.
func (sc *ShardedCache) searchAll(ctx context.Context, ck CacheKey) ([]*sdp.Item, error) {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attribute.Bool("ovm.cache.fanOut", true))

	type result struct {
		items []*sdp.Item
		err   error
		dur   time.Duration
	}
	results := make([]result, len(sc.shards))

	var wg sync.WaitGroup
	for i, shard := range sc.shards {
		wg.Add(1)
		go func(i int, shard *BoltCache) {
			defer wg.Done()
			start := time.Now()
			items, err := shard.Search(ctx, ck)
			results[i] = result{items: items, err: err, dur: time.Since(start)}
		}(i, shard)
	}
	wg.Wait()

	var (
		allItems         []*sdp.Item
		maxDur           time.Duration
		shardsWithResult int
		firstErr         error
		allNotFound      = true
	)

	for _, r := range results {
		if r.dur > maxDur {
			maxDur = r.dur
		}
		if r.err != nil {
			if errors.Is(r.err, ErrCacheNotFound) {
				continue
			}
			allNotFound = false
			if firstErr == nil {
				firstErr = r.err
			}
			continue
		}
		allNotFound = false
		if len(r.items) > 0 {
			shardsWithResult++
			allItems = append(allItems, r.items...)
		}
	}

	span.SetAttributes(
		attribute.Float64("ovm.cache.fanOutMaxMs", float64(maxDur.Milliseconds())),
		attribute.Int("ovm.cache.shardsWithResults", shardsWithResult),
	)

	if firstErr != nil {
		return nil, firstErr
	}

	if allNotFound {
		return nil, ErrCacheNotFound
	}

	return allItems, nil
}

// StoreItem routes the item to one shard based on its UniqueAttributeValue.
func (sc *ShardedCache) StoreItem(ctx context.Context, item *sdp.Item, duration time.Duration, ck CacheKey) {
	if item == nil {
		return
	}

	sstHash := ck.SST.Hash()
	uav := item.UniqueAttributeValue()
	idx := sc.shardFor(sstHash, uav)

	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attribute.Int("ovm.cache.shardIndex", idx))

	sc.shards[idx].StoreItem(ctx, item, duration, ck)
}

// StoreUnavailableItem routes the error based on the CacheKey:
//   - GET errors (UniqueAttributeValue set) go to the same shard a GET Lookup would query.
//   - LIST/SEARCH errors go to shard 0 as a deterministic default; fan-out reads will find them.
func (sc *ShardedCache) StoreUnavailableItem(ctx context.Context, err error, duration time.Duration, ck CacheKey) {
	if err == nil {
		return
	}

	var idx int
	if ck.UniqueAttributeValue != nil {
		idx = sc.shardFor(ck.SST.Hash(), *ck.UniqueAttributeValue)
	}

	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attribute.Int("ovm.cache.shardIndex", idx))

	sc.shards[idx].StoreUnavailableItem(ctx, err, duration, ck)
}

// Delete fans out to all shards.
func (sc *ShardedCache) Delete(ck CacheKey) {
	var wg sync.WaitGroup
	for _, shard := range sc.shards {
		wg.Add(1)
		go func(s *BoltCache) {
			defer wg.Done()
			s.Delete(ck)
		}(shard)
	}
	wg.Wait()
}

// Clear fans out to all shards.
func (sc *ShardedCache) Clear() {
	var wg sync.WaitGroup
	for _, shard := range sc.shards {
		wg.Add(1)
		go func(s *BoltCache) {
			defer wg.Done()
			s.Clear()
		}(shard)
	}
	wg.Wait()
}

// Purge fans out to all shards in parallel and aggregates PurgeStats.
// TimeTaken reflects wall-clock time of the parallel fan-out, not the sum of
// per-shard durations.
func (sc *ShardedCache) Purge(ctx context.Context, before time.Time) PurgeStats {
	type result struct {
		stats PurgeStats
	}
	results := make([]result, len(sc.shards))

	start := time.Now()

	var wg sync.WaitGroup
	for i, shard := range sc.shards {
		wg.Add(1)
		go func(i int, s *BoltCache) {
			defer wg.Done()
			results[i] = result{stats: s.Purge(ctx, before)}
		}(i, shard)
	}
	wg.Wait()

	combined := PurgeStats{
		TimeTaken: time.Since(start),
	}
	for _, r := range results {
		combined.NumPurged += r.stats.NumPurged
		if r.stats.NextExpiry != nil {
			if combined.NextExpiry == nil || r.stats.NextExpiry.Before(*combined.NextExpiry) {
				combined.NextExpiry = r.stats.NextExpiry
			}
		}
	}
	return combined
}

// GetMinWaitTime returns the minimum wait time from the first shard.
func (sc *ShardedCache) GetMinWaitTime() time.Duration {
	if len(sc.shards) == 0 {
		return 0
	}
	return sc.shards[0].GetMinWaitTime()
}

// StartPurger starts a purger on each shard independently.
func (sc *ShardedCache) StartPurger(ctx context.Context) {
	for _, shard := range sc.shards {
		shard.StartPurger(ctx)
	}
}

// CloseAndDestroy closes and destroys all shard files in parallel, then removes
// the shard directory.
func (sc *ShardedCache) CloseAndDestroy() error {
	errs := make([]error, len(sc.shards))

	var wg sync.WaitGroup
	for i, shard := range sc.shards {
		wg.Add(1)
		go func(i int, s *BoltCache) {
			defer wg.Done()
			errs[i] = s.CloseAndDestroy()
		}(i, shard)
	}
	wg.Wait()

	for _, err := range errs {
		if err != nil {
			return err
		}
	}

	return os.RemoveAll(sc.dir)
}

// createDoneFunc returns a done function that calls pending.Complete for the
// given cache key. Safe to call multiple times (idempotent via sync.Once).
func (sc *ShardedCache) createDoneFunc(ck CacheKey) func() {
	if sc == nil || sc.pending == nil {
		return noopDone
	}
	key := ck.String()
	var once sync.Once
	return func() {
		once.Do(func() {
			sc.pending.Complete(key)
		})
	}
}

// newShardedCacheForProduction is used by NewCache to create a production
// ShardedCache with appropriate defaults. It logs and falls back to MemoryCache
// on failure.
func newShardedCacheForProduction(ctx context.Context) Cache {
	dir, err := os.MkdirTemp("", "sdpcache-shards-*")
	if err != nil {
		sentry.CaptureException(err)
		log.WithError(err).Error("Failed to create temp dir for ShardedCache, using memory cache instead")
		cache := NewMemoryCache()
		cache.StartPurger(ctx)
		return cache
	}

	perShardThreshold := int64(1*1024*1024*1024) / int64(DefaultShardCount)

	cache, err := NewShardedCache(
		dir,
		DefaultShardCount,
		WithMinWaitTime(30*time.Second),
		WithCompactThreshold(perShardThreshold),
	)
	if err != nil {
		sentry.CaptureException(err)
		log.WithError(err).Error("Failed to create ShardedCache, using memory cache instead")
		_ = os.RemoveAll(dir)
		memCache := NewMemoryCache()
		memCache.StartPurger(ctx)
		return memCache
	}

	cache.StartPurger(ctx)
	return cache
}
