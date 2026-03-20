package sdpcache

import (
	"context"
	"time"

	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/tracing"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// BoltCache wraps boltStore and delegates Lookup control-flow to
// lookupCoordinator. Purge scheduling lives here; boltStore only handles
// storage and purge execution.
type BoltCache struct {
	purger

	*boltStore
	pending *pendingWork
	lookup  *lookupCoordinator
}

// assert interface
var _ Cache = (*BoltCache)(nil)

// NewBoltCache creates a new BoltCache at the specified path.
// If a cache file already exists at the path, it will be opened and used.
// The existing file will be automatically handled by the purge process,
// which removes expired items. No explicit cleanup is needed on startup.
func NewBoltCache(path string, opts ...BoltCacheOption) (*BoltCache, error) {
	store, err := newBoltCacheStore(path, opts...)
	if err != nil {
		return nil, err
	}

	pending := newPendingWork()
	c := &BoltCache{
		boltStore: store,
		pending:   pending,
		lookup:    newLookupCoordinator(pending),
	}
	c.purgeFunc = c.boltStore.Purge
	return c, nil
}

// Lookup performs a cache lookup for the given query parameters.
func (c *BoltCache) Lookup(ctx context.Context, srcName string, method sdp.QueryMethod, scope string, typ string, query string, ignoreCache bool) (bool, CacheKey, []*sdp.Item, *sdp.QueryError, func()) {
	ctx, span := tracing.Tracer().Start(ctx, "BoltCache.Lookup",
		trace.WithAttributes(
			attribute.String("ovm.cache.sourceName", srcName),
			attribute.String("ovm.cache.method", method.String()),
			attribute.String("ovm.cache.scope", scope),
			attribute.String("ovm.cache.type", typ),
			attribute.String("ovm.cache.query", query),
			attribute.Bool("ovm.cache.ignoreCache", ignoreCache),
		),
	)
	defer span.End()

	ck := CacheKeyFromParts(srcName, method, scope, typ, query)

	if c == nil || c.boltStore == nil {
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

	// Set disk usage metrics
	c.setDiskUsageAttributes(span)

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

// StoreItem delegates to boltStore and pokes the purge timer.
func (c *BoltCache) StoreItem(ctx context.Context, item *sdp.Item, duration time.Duration, ck CacheKey) {
	if item == nil {
		return
	}
	c.boltStore.StoreItem(ctx, item, duration, ck)
	c.setNextPurgeIfEarlier(time.Now().Add(duration))
}

// StoreUnavailableItem delegates to boltStore and pokes the purge timer.
func (c *BoltCache) StoreUnavailableItem(ctx context.Context, err error, duration time.Duration, ck CacheKey) {
	if err == nil {
		return
	}
	c.boltStore.StoreUnavailableItem(ctx, err, duration, ck)
	c.setNextPurgeIfEarlier(time.Now().Add(duration))
}
