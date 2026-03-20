package sdpcache

import (
	"context"
	"sync"
	"time"

	"github.com/google/btree"
	"github.com/overmindtech/cli/go/sdp-go"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/proto"
)

type MemoryCache struct {
	purger

	indexes map[SSTHash]*indexSet

	// This index is used to track item expiries, since items can have different
	// expiry durations we need to use a btree here rather than just appending
	// to a slice or something. The purge process uses this to determine what
	// needs deleting, then calls into each specific index to delete as required
	expiryIndex *btree.BTreeG[*CachedResult]

	// Mutex for reading caches
	indexMutex sync.RWMutex

	// Tracks in-flight lookups to prevent duplicate work when multiple
	// goroutines request the same cache key simultaneously
	pending *pendingWork

	lookup *lookupCoordinator
}

var _ Cache = (*MemoryCache)(nil)

// NewMemoryCache creates a new in-memory cache implementation.
func NewMemoryCache() *MemoryCache {
	pending := newPendingWork()
	c := &MemoryCache{
		indexes:     make(map[SSTHash]*indexSet),
		expiryIndex: newExpiryIndex(),
		pending:     pending,
		lookup:      newLookupCoordinator(pending),
	}
	c.purgeFunc = c.Purge
	return c
}

func newExpiryIndex() *btree.BTreeG[*CachedResult] {
	return btree.NewG(2, func(a, b *CachedResult) bool {
		return a.Expiry.Before(b.Expiry)
	})
}

type indexSet struct {
	uniqueAttributeValueIndex *btree.BTreeG[*CachedResult]
	methodIndex               *btree.BTreeG[*CachedResult]
	queryIndex                *btree.BTreeG[*CachedResult]
}

func newIndexSet() *indexSet {
	return &indexSet{
		uniqueAttributeValueIndex: btree.NewG(2, func(a, b *CachedResult) bool {
			return sortString(a.IndexValues.UniqueAttributeValue, a.Item) < sortString(b.IndexValues.UniqueAttributeValue, b.Item)
		}),
		methodIndex: btree.NewG(2, func(a, b *CachedResult) bool {
			return sortString(a.IndexValues.Method.String(), a.Item) < sortString(b.IndexValues.Method.String(), b.Item)
		}),
		queryIndex: btree.NewG(2, func(a, b *CachedResult) bool {
			return sortString(a.IndexValues.Query, a.Item) < sortString(b.IndexValues.Query, b.Item)
		}),
	}
}

// Lookup returns true/false whether or not the cache has a result for the given
// query. If there are results, they will be returned as slice of `sdp.Item`s or
// an `*sdp.QueryError`.
// The CacheKey is always returned, even if the lookup otherwise fails or errors.
func (c *MemoryCache) Lookup(ctx context.Context, srcName string, method sdp.QueryMethod, scope string, typ string, query string, ignoreCache bool) (bool, CacheKey, []*sdp.Item, *sdp.QueryError, func()) {
	span := trace.SpanFromContext(ctx)
	ck := CacheKeyFromParts(srcName, method, scope, typ, query)

	if c == nil {
		span.SetAttributes(
			attribute.String("ovm.cache.result", "cache not initialised"),
			attribute.Bool("ovm.cache.hit", false),
		)
		return false, ck, nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "cache has not been initialised",
			Scope:       scope,
			SourceName:  srcName,
			ItemType:    typ,
		}, noopDone
	}

	if ignoreCache {
		span.SetAttributes(
			attribute.String("ovm.cache.result", "ignore cache"),
			attribute.Bool("ovm.cache.hit", false),
		)
		return false, ck, nil, nil, noopDone
	}

	lookup := c.lookup
	if lookup == nil {
		lookup = newLookupCoordinator(c.pending)
	}

	hit, items, qErr, done := lookup.Lookup(
		ctx,
		c,
		ck,
		method,
	)
	return hit, ck, items, qErr, done
}

// Search performs a lower-level search using a CacheKey.
// This bypasses pending-work deduplication and is used by lookupCoordinator.
func (c *MemoryCache) Search(ctx context.Context, ck CacheKey) ([]*sdp.Item, error) {
	return c.search(ctx, ck)
}

// search performs a lower-level search using a CacheKey.
func (c *MemoryCache) search(ctx context.Context, ck CacheKey) ([]*sdp.Item, error) {
	if c == nil {
		return nil, nil
	}

	items := make([]*sdp.Item, 0)

	results := c.getResults(ck)

	if len(results) == 0 {
		return nil, ErrCacheNotFound
	}

	now := time.Now()

	// If there is an error we want to return that, so we need to range over the
	// results and separate items and errors. This is computationally less
	// efficient than extracting errors inside of `getResults()` but logically
	// it's a lot less complicated since `Delete()` uses the same method but
	// applies different logic
	for _, res := range results {
		// Check if the cached result has expired
		if res.Expiry.Before(now) {
			// Skip expired results
			continue
		}

		if res.Error != nil {
			return nil, res.Error
		}

		// Return a copy of the item so the user can do whatever they want with it
		itemCopy := proto.Clone(res.Item).(*sdp.Item)

		items = append(items, itemCopy)
	}

	// If all results were expired, return cache not found
	if len(items) == 0 {
		return nil, ErrCacheNotFound
	}

	return items, nil
}

// Delete deletes anything that matches the given cache query.
func (c *MemoryCache) Delete(ck CacheKey) {
	if c == nil {
		return
	}

	c.deleteResults(c.getResults(ck))
}

// getResults searches indexes for cached results, doing no other logic. If
// nothing is found an empty slice will be returned.
func (c *MemoryCache) getResults(ck CacheKey) []*CachedResult {
	c.indexMutex.RLock()
	defer c.indexMutex.RUnlock()

	results := make([]*CachedResult, 0)

	// Get the relevant set of indexes based on the SST Hash
	sstHash := ck.SST.Hash()
	indexes, exists := c.indexes[sstHash]
	pivot := CachedResult{
		IndexValues: IndexValues{
			SSTHash: sstHash,
		},
	}

	if !exists {
		// If we don't have a set of indexes then it definitely doesn't exist
		return results
	}

	// Start with the most specific index and fall back to the least specific.
	// Checking all matching items and returning. These is no need to check all
	// indexes since they all have the same content
	if ck.UniqueAttributeValue != nil {
		pivot.IndexValues.UniqueAttributeValue = *ck.UniqueAttributeValue

		indexes.uniqueAttributeValueIndex.AscendGreaterOrEqual(&pivot, func(result *CachedResult) bool {
			if *ck.UniqueAttributeValue == result.IndexValues.UniqueAttributeValue {
				if ck.Matches(result.IndexValues) {
					results = append(results, result)
				}

				// Always return true so that we continue to iterate
				return true
			}

			return false
		})

		return results
	}

	if ck.Query != nil {
		pivot.IndexValues.Query = *ck.Query

		indexes.queryIndex.AscendGreaterOrEqual(&pivot, func(result *CachedResult) bool {
			if *ck.Query == result.IndexValues.Query {
				if ck.Matches(result.IndexValues) {
					results = append(results, result)
				}

				// Always return true so that we continue to iterate
				return true
			}

			return false
		})

		return results
	}

	if ck.Method != nil {
		pivot.IndexValues.Method = *ck.Method

		indexes.methodIndex.AscendGreaterOrEqual(&pivot, func(result *CachedResult) bool {
			if *ck.Method == result.IndexValues.Method {
				// If the methods match, check the rest
				if ck.Matches(result.IndexValues) {
					results = append(results, result)
				}

				// Always return true so that we continue to iterate
				return true
			}

			return false
		})

		return results
	}

	// If nothing other than SST has been set then return everything
	indexes.methodIndex.Ascend(func(result *CachedResult) bool {
		results = append(results, result)

		return true
	})

	return results
}

// StoreItem stores an item in the cache. Note that this item must be fully
// populated (including metadata) for indexing to work correctly.
func (c *MemoryCache) StoreItem(ctx context.Context, item *sdp.Item, duration time.Duration, ck CacheKey) {
	if item == nil || c == nil {
		return
	}

	itemCopy := proto.Clone(item).(*sdp.Item)

	res := CachedResult{
		Item:   itemCopy,
		Error:  nil,
		Expiry: time.Now().Add(duration),
		IndexValues: IndexValues{
			UniqueAttributeValue: itemCopy.UniqueAttributeValue(),
		},
	}

	if ck.Method != nil {
		res.IndexValues.Method = *ck.Method
	}
	if ck.Query != nil {
		res.IndexValues.Query = *ck.Query
	}

	res.IndexValues.SSTHash = ck.SST.Hash()

	c.storeResult(ctx, res)
}

// StoreUnavailableItem stores an error for the given duration.
func (c *MemoryCache) StoreUnavailableItem(ctx context.Context, err error, duration time.Duration, cacheQuery CacheKey) {
	if c == nil || err == nil {
		return
	}

	res := CachedResult{
		Item:        nil,
		Error:       err,
		Expiry:      time.Now().Add(duration),
		IndexValues: cacheQuery.ToIndexValues(),
	}

	c.storeResult(ctx, res)
}

// Clear deletes all data in cache.
func (c *MemoryCache) Clear() {
	if c == nil {
		return
	}

	c.indexMutex.Lock()
	defer c.indexMutex.Unlock()

	c.indexes = make(map[SSTHash]*indexSet)
	c.expiryIndex = newExpiryIndex()
}

func (c *MemoryCache) storeResult(ctx context.Context, res CachedResult) {
	c.indexMutex.Lock()
	defer c.indexMutex.Unlock()

	// Create the index if it doesn't exist
	indexes, ok := c.indexes[res.IndexValues.SSTHash]

	if !ok {
		indexes = newIndexSet()
		c.indexes[res.IndexValues.SSTHash] = indexes
	}

	// Add the item to the indexes and check if we're overwriting an unexpired entry
	// We only need to check one index since they all reference the same CachedResult
	oldResult, replaced := indexes.methodIndex.ReplaceOrInsert(&res)
	indexes.queryIndex.ReplaceOrInsert(&res)
	indexes.uniqueAttributeValueIndex.ReplaceOrInsert(&res)

	// Get the current span to add attributes
	span := trace.SpanFromContext(ctx)

	// Check if we overwrote an entry that hasn't expired yet
	// This indicates potential thundering-herd issues where multiple identical
	// queries are executed concurrently instead of waiting for the first result
	overwritten := false
	if replaced && oldResult != nil {
		now := time.Now()
		if oldResult.Expiry.After(now) {
			overwritten = true
			timeUntilExpiry := oldResult.Expiry.Sub(now)

			// Build attributes for the overwrite event
			attrs := []attribute.KeyValue{
				attribute.Bool("ovm.cache.unexpired_overwrite", true),
				attribute.String("ovm.cache.time_until_expiry", timeUntilExpiry.String()),
				attribute.String("ovm.cache.sst_hash", string(res.IndexValues.SSTHash)),
				attribute.String("ovm.cache.query_method", res.IndexValues.Method.String()),
			}

			if res.Item != nil {
				attrs = append(attrs,
					attribute.String("ovm.cache.item_type", res.Item.GetType()),
					attribute.String("ovm.cache.item_scope", res.Item.GetScope()),
				)
			}

			if res.IndexValues.Query != "" {
				attrs = append(attrs, attribute.String("ovm.cache.query", res.IndexValues.Query))
			}

			if res.IndexValues.UniqueAttributeValue != "" {
				attrs = append(attrs, attribute.String("ovm.cache.unique_attribute", res.IndexValues.UniqueAttributeValue))
			}

			span.SetAttributes(attrs...)
		}
	}

	// Always set the overwrite attribute, even if false, for consistent tracking
	if !overwritten {
		span.SetAttributes(attribute.Bool("ovm.cache.unexpired_overwrite", false))
	}

	// Add the item to the expiry index
	c.expiryIndex.ReplaceOrInsert(&res)

	// Update the purge time if required
	c.setNextPurgeIfEarlier(res.Expiry)
}

// sortString returns the string that the cached result should be sorted on.
// This has a prefix of the index value and suffix of the GloballyUniqueName if
// relevant.
func sortString(indexValue string, item *sdp.Item) string {
	if item == nil {
		return indexValue
	}
	return indexValue + item.GloballyUniqueName()
}

// deleteResults deletes many cached results at once.
func (c *MemoryCache) deleteResults(results []*CachedResult) {
	c.indexMutex.Lock()
	defer c.indexMutex.Unlock()

	for _, res := range results {
		if indexSet, ok := c.indexes[res.IndexValues.SSTHash]; ok {
			// For each expired item, delete it from all of the indexes that it will be in
			if indexSet.methodIndex != nil {
				indexSet.methodIndex.Delete(res)
			}
			if indexSet.queryIndex != nil {
				indexSet.queryIndex.Delete(res)
			}
			if indexSet.uniqueAttributeValueIndex != nil {
				indexSet.uniqueAttributeValueIndex.Delete(res)
			}
		}

		c.expiryIndex.Delete(res)
	}
}

// Purge purges all expired items from the cache. The user must pass in the
// `before` time. All items that expired before this will be purged. Usually
// this would be just `time.Now()` however it could be overridden for testing.
func (c *MemoryCache) Purge(ctx context.Context, before time.Time) PurgeStats {
	if c == nil {
		return PurgeStats{}
	}

	// Store the current time rather than calling it a million times
	start := time.Now()

	var nextExpiry *time.Time

	expired := make([]*CachedResult, 0)

	// Look through the expiry cache and work out what has expired
	c.indexMutex.RLock()
	c.expiryIndex.Ascend(func(res *CachedResult) bool {
		if res.Expiry.Before(before) {
			expired = append(expired, res)

			return true
		}

		// Take note of the next expiry so we can schedule the next run
		nextExpiry = &res.Expiry

		// As soon as hit this we'll stop ascending
		return false
	})
	c.indexMutex.RUnlock()

	c.deleteResults(expired)

	return PurgeStats{
		NumPurged:  len(expired),
		TimeTaken:  time.Since(start),
		NextExpiry: nextExpiry,
	}
}
