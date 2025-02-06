package sdpcache

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/btree"
	"github.com/overmindtech/cli/sdp-go"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type IndexValues struct {
	SSTHash              SSTHash
	UniqueAttributeValue string
	Method               sdp.QueryMethod
	Query                string
}

type CacheKey struct {
	SST                  SST // *required
	UniqueAttributeValue *string
	Method               *sdp.QueryMethod
	Query                *string
}

func CacheKeyFromParts(srcName string, method sdp.QueryMethod, scope string, typ string, query string) CacheKey {
	ck := CacheKey{
		SST: SST{
			SourceName: srcName,
			Scope:      scope,
			Type:       typ,
		},
	}

	switch method {
	case sdp.QueryMethod_GET:
		// With a Get query we need just the one specific item, so also
		// filter on uniqueAttributeValue
		ck.UniqueAttributeValue = &query
	case sdp.QueryMethod_LIST:
		// In the case of a find, we just want everything that was found in
		// the last find, so we only care about the method
		ck.Method = &method
	case sdp.QueryMethod_SEARCH:
		// For a search, we only want to get from the cache items that were
		// found using search, and with the exact same query
		ck.Method = &method
		ck.Query = &query
	}

	return ck
}

func CacheKeyFromQuery(q *sdp.Query, srcName string) CacheKey {
	return CacheKeyFromParts(srcName, q.GetMethod(), q.GetScope(), q.GetType(), q.GetQuery())
}

func (ck CacheKey) String() string {
	fields := []string{
		("SourceName=" + ck.SST.SourceName),
		("Scope=" + ck.SST.Scope),
		("Type=" + ck.SST.Type),
	}

	if ck.UniqueAttributeValue != nil {
		fields = append(fields, ("UniqueAttributeValue=" + *ck.UniqueAttributeValue))
	}

	if ck.Method != nil {
		fields = append(fields, ("Method=" + ck.Method.String()))
	}

	if ck.Query != nil {
		fields = append(fields, ("Query=" + *ck.Query))
	}

	return strings.Join(fields, ", ")
}

// ToIndexValues Converts a cache query to a set of index values
func (ck CacheKey) ToIndexValues() IndexValues {
	iv := IndexValues{
		SSTHash: ck.SST.Hash(),
	}

	if ck.Method != nil {
		iv.Method = *ck.Method
	}

	if ck.Query != nil {
		iv.Query = *ck.Query
	}

	if ck.UniqueAttributeValue != nil {
		iv.UniqueAttributeValue = *ck.UniqueAttributeValue
	}

	return iv
}

// Matches Returns whether or not the supplied index values match the
// CacheQuery, excluding the SST since this will have already been validated.
// Note that this only checks values that ave actually been set in the
// CacheQuery
func (ck CacheKey) Matches(i IndexValues) bool {
	// Check for any mismatches on the values that are set
	if ck.Method != nil {
		if *ck.Method != i.Method {
			return false
		}
	}

	if ck.Query != nil {
		if *ck.Query != i.Query {
			return false
		}
	}

	if ck.UniqueAttributeValue != nil {
		if *ck.UniqueAttributeValue != i.UniqueAttributeValue {
			return false
		}
	}

	return true
}

var ErrCacheNotFound = errors.New("not found in cache")

// SST A combination of SourceName, Scope and Type, all of which must be
// provided
type SST struct {
	SourceName string
	Scope      string
	Type       string
}

// Hash Creates a new SST hash from a given SST
func (s SST) Hash() SSTHash {
	h := sha256.New()
	h.Write([]byte(s.SourceName))
	h.Write([]byte(s.Scope))
	h.Write([]byte(s.Type))

	sum := make([]byte, 0)
	sum = h.Sum(sum)

	return SSTHash(fmt.Sprintf("%x", sum))
}

// CachedResult An item including cache metadata
type CachedResult struct {
	// Item is the actual cached item
	Item *sdp.Item

	// Error is the error that we want
	Error error

	// The time at which this item expires
	Expiry time.Time

	// Values that we use for calculating indexes
	IndexValues IndexValues
}

// SSTHash Represents the hash of `SourceName`, `Scope` and `Type`
type SSTHash string

type Cache struct {
	// Minimum amount of time to wait between cache purges
	MinWaitTime time.Duration

	// The timer that is used to trigger the next purge
	purgeTimer *time.Timer

	// The time that the purger will run next
	nextPurge time.Time

	indexes map[SSTHash]*indexSet

	// This index is used to track item expiries, since items can have different
	// expiry durations we need to use a btree here rather than just appending
	// to a slice or something. The purge process uses this to determine what
	// needs deleting, then calls into each specific index to delete as required
	expiryIndex *btree.BTreeG[*CachedResult]

	// Mutex for reading caches
	indexMutex sync.RWMutex

	// Ensures that purge stats like `purgeTimer` and `nextPurge` aren't being
	// modified concurrently
	purgeMutex sync.Mutex
}

func NewCache() *Cache {
	return &Cache{
		indexes:     make(map[SSTHash]*indexSet),
		expiryIndex: newExpiryIndex(),
	}
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
// The CacheKey is always returned, even if the lookup otherwise fails or errors
func (c *Cache) Lookup(ctx context.Context, srcName string, method sdp.QueryMethod, scope string, typ string, query string, ignoreCache bool) (bool, CacheKey, []*sdp.Item, *sdp.QueryError) {
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
		}
	}

	if ignoreCache {
		span.SetAttributes(
			attribute.String("ovm.cache.result", "ignore cache"),
			attribute.Bool("ovm.cache.hit", false),
		)
		return false, ck, nil, nil
	}

	items, err := c.Search(ck)

	if err != nil {
		var qErr *sdp.QueryError
		if errors.Is(err, ErrCacheNotFound) {
			// If nothing was found then execute the search against the sources
			span.SetAttributes(
				attribute.String("ovm.cache.result", "cache miss"),
				attribute.Bool("ovm.cache.hit", false),
			)
			return false, ck, nil, nil
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
			return true, ck, nil, qErr
		} else {
			// If it's an unknown error, convert it to SDP and skip this source
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

			return true, ck, nil, qErr
		}
	}

	if method == sdp.QueryMethod_GET {
		// If the method was Get we should validate that we have
		// only pulled one thing from the cache

		if len(items) < 2 {
			span.SetAttributes(
				attribute.String("ovm.cache.result", "cache hit: 1 item"),
				attribute.Int("ovm.cache.numItems", len(items)),
				attribute.Bool("ovm.cache.hit", true),
			)
			return true, ck, items, nil
		} else {
			span.SetAttributes(
				attribute.String("ovm.cache.result", "cache returned >1 value, purging and continuing"),
				attribute.Int("ovm.cache.numItems", len(items)),
				attribute.Bool("ovm.cache.hit", false),
			)
			c.Delete(ck)
			return false, ck, nil, nil
		}
	}

	span.SetAttributes(
		attribute.String("ovm.cache.result", "cache hit: multiple items"),
		attribute.Int("ovm.cache.numItems", len(items)),
		attribute.Bool("ovm.cache.hit", true),
	)

	return true, ck, items, nil
}

// Search Runs a given query against the cache. If a cached error is found it
// will be returned immediately, if nothing is found a ErrCacheNotFound will
// be returned. Otherwise this will return items that match ALL of the given
// query parameters
func (c *Cache) Search(ck CacheKey) ([]*sdp.Item, error) {
	if c == nil {
		return nil, nil
	}

	items := make([]*sdp.Item, 0)

	results := c.getResults(ck)

	if len(results) == 0 {
		return nil, ErrCacheNotFound
	}

	// If there is an error we want to return that, so we need to range over the
	// results and separate items and errors. This is computationally less
	// efficient than extracting errors inside of `getResults()` but logically
	// it's a lot less complicated since `Delete()` uses the same method but
	// applies different logic
	for _, res := range results {
		if res.Error != nil {
			return nil, res.Error
		}

		// Return a copy of the item so the user can do whatever they want with
		// it
		itemCopy := sdp.Item{}
		res.Item.Copy(&itemCopy)

		items = append(items, &itemCopy)
	}

	return items, nil
}

// Delete Deletes anything that matches the given cache query
func (c *Cache) Delete(ck CacheKey) {
	if c == nil {
		return
	}

	c.deleteResults(c.getResults(ck))
}

// getResults Searches indexes for cached results, doing no other logic. If
// nothing is found an empty slice will be returned.
func (c *Cache) getResults(ck CacheKey) []*CachedResult {
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

// StoreItem Stores an item in the cache. Note that this item must be fully
// populated (including metadata) for indexing to work correctly
func (c *Cache) StoreItem(item *sdp.Item, duration time.Duration, ck CacheKey) {
	if item == nil || c == nil {
		return
	}

	itemCopy := sdp.Item{}
	item.Copy(&itemCopy)

	res := CachedResult{
		Item:   &itemCopy,
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

	c.storeResult(res)
}

// StoreError Stores an error for the given duration.
func (c *Cache) StoreError(err error, duration time.Duration, cacheQuery CacheKey) {
	if c == nil || err == nil {
		return
	}

	res := CachedResult{
		Item:        nil,
		Error:       err,
		Expiry:      time.Now().Add(duration),
		IndexValues: cacheQuery.ToIndexValues(),
	}

	c.storeResult(res)
}

// Clear Delete all data in cache
func (c *Cache) Clear() {
	if c == nil {
		return
	}

	c.indexMutex.Lock()
	defer c.indexMutex.Unlock()

	c.indexes = make(map[SSTHash]*indexSet)
	c.expiryIndex = newExpiryIndex()
}

func (c *Cache) storeResult(res CachedResult) {
	c.indexMutex.Lock()
	defer c.indexMutex.Unlock()

	// Create the index if it doesn't exist
	indexes, ok := c.indexes[res.IndexValues.SSTHash]

	if !ok {
		indexes = newIndexSet()
		c.indexes[res.IndexValues.SSTHash] = indexes
	}

	// Add the item to the indexes
	indexes.methodIndex.ReplaceOrInsert(&res)
	indexes.queryIndex.ReplaceOrInsert(&res)
	indexes.uniqueAttributeValueIndex.ReplaceOrInsert(&res)

	// Add the item to the expiry index
	c.expiryIndex.ReplaceOrInsert(&res)

	// Update the purge time if required
	c.setNextPurgeIfEarlier(res.Expiry)
}

// sortString Returns the string that the cached result should be sorted on.
// This has a prefix of the index value and suffix of the GloballyUniqueName if
// relevant
func sortString(indexValue string, item *sdp.Item) string {
	if item == nil {
		return indexValue
	} else {
		return indexValue + item.GloballyUniqueName()
	}
}

// PurgeStats Stats about the Purge
type PurgeStats struct {
	// How many items were timed out of the cache
	NumPurged int
	// How long purging took overall
	TimeTaken time.Duration
	// The expiry time of the next item to expire. If there are no more items in
	// the cache, this will be nil
	NextExpiry *time.Time
}

// deleteResults Deletes many cached results at once
func (c *Cache) deleteResults(results []*CachedResult) {
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

// Purge Purges all expired items from the cache. The user must pass in the
// `before` time. All items that expired before this will be purged. Usually
// this would be just `time.Now()` however it could be overridden for testing
func (c *Cache) Purge(before time.Time) PurgeStats {
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

// MinWaitDefault The default minimum wait time
const MinWaitDefault = (5 * time.Second)

// GetMinWaitTime Returns the minimum wait time or the default if not set
func (c *Cache) GetMinWaitTime() time.Duration {
	if c == nil {
		return 0
	}

	if c.MinWaitTime == 0 {
		return MinWaitDefault
	}

	return c.MinWaitTime
}

// StartPurger Starts the purge process in the background, it will be cancelled
// when the context is cancelled. The cache will be purged initially, at which
// point the process will sleep until the next time an item expires
func (c *Cache) StartPurger(ctx context.Context) error {
	if c == nil {
		return nil
	}

	c.purgeMutex.Lock()
	if c.purgeTimer == nil {
		c.purgeTimer = time.NewTimer(0)
		c.purgeMutex.Unlock()
	} else {
		c.purgeMutex.Unlock()
		return errors.New("purger already running")
	}

	go func(ctx context.Context) {
		for {
			select {
			case <-c.purgeTimer.C:
				stats := c.Purge(time.Now())

				c.setNextPurgeFromStats(stats)
			case <-ctx.Done():
				c.purgeMutex.Lock()
				defer c.purgeMutex.Unlock()

				c.purgeTimer.Stop()
				c.purgeTimer = nil
				return
			}
		}
	}(ctx)

	return nil
}

// setNextPurgeFromStats Sets when the next purge should run based on the stats of the
// previous purge
func (c *Cache) setNextPurgeFromStats(stats PurgeStats) {
	c.purgeMutex.Lock()
	defer c.purgeMutex.Unlock()

	if stats.NextExpiry == nil {
		// If there is nothing else in the cache, wait basically
		// forever
		c.purgeTimer.Reset(1000 * time.Hour)
		c.nextPurge = time.Now().Add(1000 * time.Hour)
	} else {
		if time.Until(*stats.NextExpiry) < c.GetMinWaitTime() {
			c.purgeTimer.Reset(c.GetMinWaitTime())
			c.nextPurge = time.Now().Add(c.GetMinWaitTime())
		} else {
			c.purgeTimer.Reset(time.Until(*stats.NextExpiry))
			c.nextPurge = *stats.NextExpiry
		}
	}
}

// setNextPurgeIfEarlier Sets the next time the purger will run, if the provided
// time is sooner than the current scheduled purge time. While the purger is
// active this will be constantly updated, however if the purger is sleeping and
// new items are added this method ensures that the purger is woken up
func (c *Cache) setNextPurgeIfEarlier(t time.Time) {
	c.purgeMutex.Lock()
	defer c.purgeMutex.Unlock()

	if t.Before(c.nextPurge) {
		if c.purgeTimer == nil {
			return
		}

		c.purgeTimer.Stop()
		c.nextPurge = t
		c.purgeTimer.Reset(time.Until(t))
	}
}
