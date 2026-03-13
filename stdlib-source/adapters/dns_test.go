package adapters

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/overmindtech/cli/go/discovery"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
)

func TestSearch(t *testing.T) {
	t.Parallel()

	s := DNSAdapter{
		cache: sdpcache.NewNoOpCache(),
		Servers: []string{
			"1.1.1.1:53",
			"8.8.8.8:53",
		},
	}

	t.Run("with a bad DNS name", func(t *testing.T) {
		_, err := s.Search(context.Background(), "global", "not.real.overmind.tech", false)
		if err == nil {
			t.Error("expected error for non-existent name")
		}
		var qe *sdp.QueryError
		if !errors.As(err, &qe) || qe.GetErrorType() != sdp.QueryError_NOTFOUND {
			t.Errorf("expected NOTFOUND error, got %v", err)
		}
	})

	t.Run("with one.one.one.one", func(t *testing.T) {
		items, err := s.Search(context.Background(), "global", "one.one.one.one", false)

		if err != nil {
			t.Error(err)
		}

		if len(items) != 1 {
			t.Errorf("expected 1 item, got %v", len(items))
		}

		// Make sure 1.1.1.1 is in there
		var foundV4 bool
		var foundV6 bool
		for _, item := range items {
			for _, q := range item.GetLinkedItemQueries() {
				if q.GetQuery().GetQuery() == "1.1.1.1" {
					foundV4 = true
				}
				if q.GetQuery().GetQuery() == "2606:4700:4700::1111" {
					foundV6 = true
				}
			}
		}

		if !foundV4 {
			t.Error("could not find 1.1.1.1 in linked item queries")
		}
		if !foundV6 {
			t.Error("could not find 2606:4700:4700::1111 in linked item queries")
		}

		discovery.TestValidateItems(t, items)
	})

	t.Run("Search returns same NOTFOUND for first and second call", func(t *testing.T) {
		// First call (fresh NOTFOUND) and second call (cached NOTFOUND) must return the same: nil items, same error
		cache := sdpcache.NewMemoryCache()
		cachedSrc := DNSAdapter{cache: cache, Servers: s.Servers}
		query := "not.real.overmind.tech"

		first, err1 := cachedSrc.Search(context.Background(), "global", query, false)
		if err1 == nil {
			t.Fatal("first Search: expected NOTFOUND error, got nil")
		}
		if first != nil {
			t.Errorf("first Search: expected nil items, got len=%d", len(first))
		}
		var qe *sdp.QueryError
		if !errors.As(err1, &qe) || qe.GetErrorType() != sdp.QueryError_NOTFOUND {
			t.Errorf("first Search: expected NOTFOUND, got %v", err1)
		}
		firstErrStr := err1.Error()

		second, err2 := cachedSrc.Search(context.Background(), "global", query, false)
		if err2 == nil {
			t.Fatal("second Search: expected NOTFOUND error, got nil")
		}
		if second != nil {
			t.Errorf("second Search: expected nil items, got len=%d", len(second))
		}
		if !errors.As(err2, &qe) || qe.GetErrorType() != sdp.QueryError_NOTFOUND {
			t.Errorf("second Search: expected NOTFOUND, got %v", err2)
		}
		if err2.Error() != firstErrStr {
			t.Errorf("first and second Search must return same error message: first %q, second %q", firstErrStr, err2.Error())
		}
	})

	t.Run("with an IP and therefore reverse DNS", func(t *testing.T) {
		s.ReverseLookup = true
		items, err := s.Search(context.Background(), "global", "1.1.1.1", false)

		if err != nil {
			t.Error(err)
		}

		// Make sure 1.1.1.1 is in there
		var foundV4 bool
		var foundV6 bool
		for _, item := range items {
			for _, q := range item.GetLinkedItemQueries() {
				if q.GetQuery().GetQuery() == "1.1.1.1" {
					foundV4 = true
				}
				if q.GetQuery().GetQuery() == "2606:4700:4700::1111" {
					foundV6 = true
				}
			}
		}

		if !foundV4 {
			t.Error("could not find 1.1.1.1 in linked item queries")
		}
		if !foundV6 {
			t.Error("could not find 2606:4700:4700::1111 in linked item queries")
		}

		discovery.TestValidateItems(t, items)
	})
}

func TestDnsGet(t *testing.T) {
	t.Parallel()

	var conn net.Conn
	var err error

	// Check that we actually have an internet connection, if not there is not
	// point running this test
	dialer := &net.Dialer{}
	conn, err = dialer.DialContext(t.Context(), "tcp", "one.one.one.one:443")
	conn.Close()

	if err != nil {
		t.Skip("No internet connection detected")
	}

	src := DNSAdapter{
		cache: sdpcache.NewNoOpCache(),
	}

	t.Run("working request", func(t *testing.T) {
		item, err := src.Get(context.Background(), "global", "one.one.one.one", false)

		if err != nil {
			t.Fatal(err)
		}

		discovery.TestValidateItem(t, item)
	})

	t.Run("bad dns entry", func(t *testing.T) {
		_, err := src.Get(context.Background(), "global", "something.does.not.exist.please.testing", false)

		if err == nil {
			t.Error("expected error but got nil")
		}

		var e *sdp.QueryError
		if !errors.As(err, &e) {
			t.Errorf("expected error type to be *sdp.QueryError, got %T", err)
		}
	})

	t.Run("GET returns NOTFOUND when cache has NOTFOUND", func(t *testing.T) {
		cache := sdpcache.NewMemoryCache()
		cachedSrc := DNSAdapter{cache: cache}
		query := "cached.notfound.get.example"

		// Pre-seed cache with NOTFOUND (simulates a previous Get that got 0 records)
		ck := sdpcache.CacheKeyFromParts(cachedSrc.Name(), sdp.QueryMethod_GET, "global", cachedSrc.Type(), query)
		notFoundErr := &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOTFOUND,
			ErrorString: "no DNS records found",
			Scope:       "global",
			SourceName:  cachedSrc.Name(),
			ItemType:    cachedSrc.Type(),
		}
		cache.StoreUnavailableItem(context.Background(), notFoundErr, dnsCacheDuration, ck)

		// Get should return cached NOTFOUND without doing a DNS lookup
		item, err := cachedSrc.Get(context.Background(), "global", query, false)
		if item != nil {
			t.Errorf("expected nil item, got %v", item)
		}
		if err == nil {
			t.Fatal("expected NOTFOUND error, got nil")
		}
		var qErr *sdp.QueryError
		if !errors.As(err, &qErr) || qErr.GetErrorType() != sdp.QueryError_NOTFOUND {
			t.Errorf("expected NOTFOUND, got %v", err)
		}

		// Second Get: should still return cached NOTFOUND (same response as first)
		firstErrStr := err.Error()
		item, err = cachedSrc.Get(context.Background(), "global", query, false)
		if item != nil {
			t.Errorf("second Get: expected nil item, got %v", item)
		}
		if err == nil {
			t.Fatal("second Get: expected NOTFOUND error, got nil")
		}
		if !errors.As(err, &qErr) || qErr.GetErrorType() != sdp.QueryError_NOTFOUND {
			t.Errorf("second Get: expected NOTFOUND, got %v", err)
		}
		if err.Error() != firstErrStr {
			t.Errorf("first and second Get must return same error message: first %q, second %q", firstErrStr, err.Error())
		}
	})

	t.Run("bad scope", func(t *testing.T) {
		_, err := src.Get(context.Background(), "something.local.test", "something.does.not.exist.please.testing", false)

		if err == nil {
			t.Error("expected error but got nil")
		}

		var e *sdp.QueryError
		if !errors.As(err, &e) {
			t.Errorf("expected error type to be *sdp.QueryError, got %T", err)
		}
	})

	t.Run("with a CNAME", func(t *testing.T) {
		// When we do a Get on a CNAME, I wan it to work, but only return the
		// first thing
		item, err := src.Get(context.Background(), "global", "www.github.com", false)

		if err != nil {
			t.Fatal(err)
		}

		target := item.GetAttributes().GetAttrStruct().GetFields()["target"].GetStringValue()
		if target != "github.com" {
			t.Errorf("expected target to be github.com, got %v", target)
		}

		t.Log(item)
	})
}

// TestGetTimeout verifies that Get enforces the maximum timeout by checking
// that the adapter's timeout takes precedence over a longer caller timeout
func TestGetTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping timeout test in short mode")
	}

	src := DNSAdapter{
		cache: sdpcache.NewNoOpCache(),
		// Use a non-existent DNS server to force timeout
		Servers: []string{"192.0.2.1:53"}, // TEST-NET-1, guaranteed to be unroutable
	}

	// Create a context with a very long deadline to verify adapter's internal timeout takes precedence
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	start := time.Now()
	_, err := src.Get(ctx, "global", "test.example.com", false)
	elapsed := time.Since(start)

	// The operation should fail (no response from DNS server)
	if err == nil {
		t.Error("expected error but got nil")
	}

	// The operation should complete around the maxOperationTimeout (30s), not the caller's 10 minutes
	// Allow generous buffer for CI variance and different network behaviors
	if elapsed > 35*time.Second {
		t.Errorf("Get took %v, expected around 30s (max 35s for variance), timeout may not be properly enforced", elapsed)
	}

	// Don't assert minimum duration as TEST-NET may fail fast in some environments
	// The key assertion is that it completes in ~30s, not 10 minutes
}

// TestSearchTimeoutContext verifies that Search properly wraps the context with a timeout
func TestSearchTimeoutContext(t *testing.T) {
	t.Parallel()

	src := DNSAdapter{
		cache: sdpcache.NewNoOpCache(),
	}

	// Create a context with a very long deadline to ensure Search creates its own timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// Use a valid, fast DNS query to verify the timeout wrapper doesn't break normal operation
	items, err := src.Search(ctx, "global", "one.one.one.one", false)

	// Should succeed with the fast query
	if err != nil {
		t.Errorf("expected no error for valid query, got: %v", err)
	}

	// Should return at least one item for this known DNS name
	if len(items) == 0 {
		t.Error("expected at least one DNS item for one.one.one.one")
	}
}

// TestListBehavior verifies that List returns an empty slice without making DNS queries
func TestListBehavior(t *testing.T) {
	t.Parallel()

	src := DNSAdapter{
		cache: sdpcache.NewNoOpCache(),
	}

	ctx := context.Background()

	// List should return an empty slice without making any DNS queries
	items, err := src.List(ctx, "global", false)

	// List should succeed with empty results
	if err != nil {
		t.Errorf("expected no error but got: %v", err)
	}

	if len(items) != 0 {
		t.Errorf("expected empty list, got %d items", len(items))
	}
}

// TestTimeoutShorterThanCaller verifies that a short caller timeout is respected
func TestTimeoutShorterThanCaller(t *testing.T) {
	t.Parallel()

	src := DNSAdapter{
		cache: sdpcache.NewNoOpCache(),
		// Use a non-existent DNS server to force timeout
		Servers: []string{"192.0.2.1:53"}, // TEST-NET-1, guaranteed to be unroutable
	}

	// Create a context with a 2s deadline (shorter than the adapter's 30s max)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	start := time.Now()
	_, err := src.Get(ctx, "global", "test.example.com", false)
	elapsed := time.Since(start)

	// The operation should fail (no response from DNS server)
	if err == nil {
		t.Error("expected error but got nil")
	}

	// The operation should complete in roughly 2 seconds (the caller's timeout), not 30s
	// Allow some buffer for processing time (4s max)
	if elapsed > 4*time.Second {
		t.Errorf("Get took %v, expected around 2s (max 4s)", elapsed)
	}

	// Verify it's a context deadline exceeded error
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected context.DeadlineExceeded error, got: %v", err)
	}
}
