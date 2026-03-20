package sdpcache

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/overmindtech/cli/go/sdp-go"
)

// noopDone is a reusable no-op done function returned when no cleanup is needed
var noopDone = func() {}

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

// Cache provides operations for caching SDP query results (items and errors).
//
// # Lookup state matrix
//
// Lookup returns (hit bool, ck CacheKey, items []*sdp.Item, qErr *sdp.QueryError, done func()).
// The return values follow one of three states:
//
//   - Miss:      hit=false, items=nil,    qErr=nil    — no cached data
//   - Item hit:  hit=true,  len(items)>0, qErr=nil    — cached items found
//   - Error hit: hit=true,  items=nil,    qErr!=nil   — cached error found
//
// # done() contract
//
// On a cache miss the returned done function MUST be called after storing
// results (or deciding to store nothing). It releases the pending-work slot
// so that waiting goroutines can proceed. The done function is idempotent
// (safe to call multiple times). On a cache hit or for goroutines that were
// waiting, done is a no-op.
//
// # ignoreCache
//
// When ignoreCache=true, Lookup always returns a miss without checking the
// cache or registering pending work. The returned done is a no-op.
//
// # GET cardinality
//
// If a GET lookup finds more than one cached item for the same key, the
// cache treats the data as inconsistent, purges the key, and returns a miss.
//
// # Item ordering
//
// The order of items returned from Lookup or any fan-out search is
// implementation-defined and must not be relied upon by callers.
//
// # Error precedence
//
// If both items and an error are cached under the same CacheKey, the error
// takes precedence: Lookup returns an error hit with nil items.
//
// # TTL handling
//
// There is no minimum TTL floor. A zero or negative duration stores the
// entry with an expiry at (or before) the current time. The entry will
// not survive a Purge(ctx, time.Now()) call and will be skipped by
// subsequent searches once the clock advances past the stored expiry.
//
// # Copy semantics
//
// Stored items are copied; mutating an item after StoreItem will not alter
// the cached copy.
type Cache interface {
	// Lookup performs a cache lookup for the given query parameters.
	// See the Cache-level doc for the state matrix, done() obligations,
	// ignoreCache semantics, and GET cardinality rules.
	Lookup(ctx context.Context, srcName string, method sdp.QueryMethod, scope string, typ string, query string, ignoreCache bool) (bool, CacheKey, []*sdp.Item, *sdp.QueryError, func())

	// StoreItem stores an item in the cache with the specified TTL.
	// The item is deep-copied before storage; the caller retains ownership
	// of the original. Storing under the same CacheKey overwrites any
	// previous entry with matching IndexValues.
	StoreItem(ctx context.Context, item *sdp.Item, duration time.Duration, ck CacheKey)

	// StoreUnavailableItem stores an error in the cache with the specified TTL.
	// A subsequent Lookup for the same key returns an error hit.
	StoreUnavailableItem(ctx context.Context, err error, duration time.Duration, ck CacheKey)

	// Delete removes all entries whose IndexValues match the supplied
	// CacheKey. Because CacheKey fields are optional, omitting Method or
	// UniqueAttributeValue acts as a wildcard across those dimensions.
	Delete(ck CacheKey)

	// Clear removes every entry from the cache.
	Clear()

	// Purge removes entries that expired before the given time.
	// Returns PurgeStats with the count of purged entries and the next
	// expiry time (nil when the cache is empty after purging).
	Purge(ctx context.Context, before time.Time) PurgeStats

	// GetMinWaitTime returns the minimum interval between automatic purge
	// cycles. Stateful implementations return a positive duration;
	// NoOpCache returns 0.
	GetMinWaitTime() time.Duration

	// StartPurger starts a background goroutine that periodically calls
	// Purge. The goroutine exits when ctx is cancelled.
	StartPurger(ctx context.Context)
}

// NoOpCache is a cache implementation that does nothing.
// It can be used in tests or when caching is not desired, avoiding nil checks.
type NoOpCache struct{}

var _ Cache = (*NoOpCache)(nil)

// NewNoOpCache creates a new no-op cache that implements the Cache interface
// but performs no operations. Useful for testing or when caching is disabled.
func NewNoOpCache() Cache {
	return &NoOpCache{}
}

// Lookup always returns a cache miss
func (n *NoOpCache) Lookup(ctx context.Context, srcName string, method sdp.QueryMethod, scope string, typ string, query string, ignoreCache bool) (bool, CacheKey, []*sdp.Item, *sdp.QueryError, func()) {
	ck := CacheKeyFromParts(srcName, method, scope, typ, query)
	return false, ck, nil, nil, noopDone
}

// StoreItem does nothing
func (n *NoOpCache) StoreItem(ctx context.Context, item *sdp.Item, duration time.Duration, ck CacheKey) {
	// No-op
}

// StoreUnavailableItem does nothing
func (n *NoOpCache) StoreUnavailableItem(ctx context.Context, err error, duration time.Duration, ck CacheKey) {
	// No-op
}

// Delete does nothing
func (n *NoOpCache) Delete(ck CacheKey) {
	// No-op
}

// Clear does nothing
func (n *NoOpCache) Clear() {
	// No-op
}

// Purge returns empty stats
func (n *NoOpCache) Purge(ctx context.Context, before time.Time) PurgeStats {
	return PurgeStats{}
}

// GetMinWaitTime returns 0
func (n *NoOpCache) GetMinWaitTime() time.Duration {
	return 0
}

// StartPurger does nothing
func (n *NoOpCache) StartPurger(ctx context.Context) {
}

// NewCache creates a new cache. This function returns a Cache interface backed
// by a ShardedCache (N independent BoltDB files) for write concurrency.
// The passed context will be used to start the purger.
func NewCache(ctx context.Context) Cache {
	return newShardedCacheForProduction(ctx)
}
