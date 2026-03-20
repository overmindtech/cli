package sdpcache

import (
	"testing"
	"time"

	"github.com/overmindtech/cli/go/sdp-go"
)

func TestNoOpCacheLookupAlwaysMiss(t *testing.T) {
	cache := NewNoOpCache()

	hit, ck, items, qErr, done := cache.Lookup(
		t.Context(),
		"test-source",
		sdp.QueryMethod_GET,
		"test-scope",
		"test-type",
		"test-query",
		false,
	)

	if hit {
		t.Fatal("expected miss, got hit")
	}
	if qErr != nil {
		t.Fatalf("expected nil error, got %v", qErr)
	}
	if len(items) != 0 {
		t.Fatalf("expected no items, got %d", len(items))
	}

	expected := CacheKeyFromParts("test-source", sdp.QueryMethod_GET, "test-scope", "test-type", "test-query")
	if ck.String() != expected.String() {
		t.Fatalf("expected cache key %q, got %q", expected.String(), ck.String())
	}

	// done() should be a no-op and idempotent.
	done()
	done()
}

func TestNoOpCacheIgnoresAllMutations(t *testing.T) {
	cache := NewNoOpCache()
	ctx := t.Context()

	item := GenerateRandomItem()
	ck := CacheKeyFromQuery(item.GetMetadata().GetSourceQuery(), item.GetMetadata().GetSourceName())

	cache.StoreItem(ctx, item, time.Second, ck)
	cache.StoreUnavailableItem(ctx, &sdp.QueryError{
		ErrorType:   sdp.QueryError_OTHER,
		ErrorString: "noop",
	}, time.Second, ck)
	cache.Delete(ck)
	cache.Clear()
	cache.StartPurger(ctx)

	hit, _, items, qErr, done := cache.Lookup(
		ctx,
		item.GetMetadata().GetSourceName(),
		item.GetMetadata().GetSourceQuery().GetMethod(),
		item.GetMetadata().GetSourceQuery().GetScope(),
		item.GetMetadata().GetSourceQuery().GetType(),
		item.GetMetadata().GetSourceQuery().GetQuery(),
		false,
	)
	defer done()

	if hit {
		t.Fatal("expected miss after no-op mutations, got hit")
	}
	if qErr != nil {
		t.Fatalf("expected nil error after no-op mutations, got %v", qErr)
	}
	if len(items) != 0 {
		t.Fatalf("expected no items after no-op mutations, got %d", len(items))
	}
}

func TestNoOpCachePurgeAndMinWaitDefaults(t *testing.T) {
	cache := NewNoOpCache()

	if got := cache.GetMinWaitTime(); got != 0 {
		t.Fatalf("expected min wait time 0, got %v", got)
	}

	stats := cache.Purge(t.Context(), time.Now())
	if stats.NumPurged != 0 {
		t.Fatalf("expected NumPurged=0, got %d", stats.NumPurged)
	}
	if stats.NextExpiry != nil {
		t.Fatalf("expected NextExpiry=nil, got %v", stats.NextExpiry)
	}
}
