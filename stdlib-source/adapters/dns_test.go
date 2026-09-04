package adapters

import (
	"context"
	"errors"
	"fmt"
	"net"
	"syscall"
	"testing"
	"time"

	"github.com/miekg/dns"
	"github.com/overmindtech/cli/go/discovery"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
)

const (
	testDNSAddressName = "address.test"
	testDNSAliasName   = "alias.test"
	testDNSMissingName = "missing.test"
	testDNSIPv4        = "192.0.2.10"
	testDNSIPv6        = "2001:db8::10"
)

func newTestDNSServer(t *testing.T) string {
	t.Helper()

	var lc net.ListenConfig
	packetConn, err := lc.ListenPacket(t.Context(), "udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen for test DNS server: %v", err)
	}

	handler := dns.HandlerFunc(func(w dns.ResponseWriter, request *dns.Msg) {
		response := new(dns.Msg)
		response.SetReply(request)
		response.Authoritative = true

		if len(request.Question) != 1 {
			response.SetRcode(request, dns.RcodeFormatError)
		} else {
			question := request.Question[0]
			switch {
			case question.Name == dns.Fqdn(testDNSAddressName) && question.Qtype == dns.TypeA:
				response.Answer = append(response.Answer, &dns.A{
					Hdr: dns.RR_Header{
						Name:   question.Name,
						Rrtype: dns.TypeA,
						Class:  dns.ClassINET,
						Ttl:    60,
					},
					A: net.ParseIP(testDNSIPv4).To4(),
				})
			case question.Name == dns.Fqdn(testDNSAddressName) && question.Qtype == dns.TypeAAAA:
				response.Answer = append(response.Answer, &dns.AAAA{
					Hdr: dns.RR_Header{
						Name:   question.Name,
						Rrtype: dns.TypeAAAA,
						Class:  dns.ClassINET,
						Ttl:    60,
					},
					AAAA: net.ParseIP(testDNSIPv6),
				})
			case question.Name == dns.Fqdn(testDNSAliasName):
				response.Answer = append(response.Answer, &dns.CNAME{
					Hdr: dns.RR_Header{
						Name:   question.Name,
						Rrtype: dns.TypeCNAME,
						Class:  dns.ClassINET,
						Ttl:    60,
					},
					Target: dns.Fqdn(testDNSAddressName),
				})
			case question.Name == "10.2.0.192.in-addr.arpa." && question.Qtype == dns.TypePTR:
				response.Answer = append(response.Answer, &dns.PTR{
					Hdr: dns.RR_Header{
						Name:   question.Name,
						Rrtype: dns.TypePTR,
						Class:  dns.ClassINET,
						Ttl:    60,
					},
					Ptr: dns.Fqdn(testDNSAddressName),
				})
			default:
				response.SetRcode(request, dns.RcodeNameError)
			}
		}

		if err := w.WriteMsg(response); err != nil {
			t.Errorf("write test DNS response: %v", err)
		}
	})

	server := &dns.Server{
		PacketConn: packetConn,
		Handler:    handler,
	}
	go func() {
		if err := server.ActivateAndServe(); err != nil && !errors.Is(err, net.ErrClosed) {
			t.Errorf("serve test DNS: %v", err)
		}
	}()
	t.Cleanup(func() {
		if err := server.Shutdown(); err != nil {
			t.Errorf("shutdown test DNS server: %v", err)
		}
	})

	return packetConn.LocalAddr().String()
}

func TestSearch(t *testing.T) {
	t.Parallel()

	server := newTestDNSServer(t)
	s := DNSAdapter{
		cache:   sdpcache.NewNoOpCache(),
		Servers: []string{server},
	}

	t.Run("with a bad DNS name", func(t *testing.T) {
		_, err := s.Search(context.Background(), "global", testDNSMissingName, false)
		if err == nil {
			t.Error("expected error for non-existent name")
		}
		var qe *sdp.QueryError
		if !errors.As(err, &qe) || qe.GetErrorType() != sdp.QueryError_NOTFOUND {
			t.Errorf("expected NOTFOUND error, got %v", err)
		}
	})

	t.Run("with an address record", func(t *testing.T) {
		items, err := s.Search(context.Background(), "global", testDNSAddressName, false)

		if err != nil {
			t.Error(err)
		}

		if len(items) != 1 {
			t.Errorf("expected 1 item, got %v", len(items))
		}

		// Make sure both fixed address records are linked.
		var foundV4 bool
		var foundV6 bool
		for _, item := range items {
			for _, q := range item.GetLinkedItemQueries() {
				if q.GetQuery().GetQuery() == testDNSIPv4 {
					foundV4 = true
				}
				if q.GetQuery().GetQuery() == testDNSIPv6 {
					foundV6 = true
				}
			}
		}

		if !foundV4 {
			t.Errorf("could not find %s in linked item queries", testDNSIPv4)
		}
		if !foundV6 {
			t.Errorf("could not find %s in linked item queries", testDNSIPv6)
		}

		discovery.TestValidateItems(t, items)
	})

	t.Run("Search returns same NOTFOUND for first and second call", func(t *testing.T) {
		// First call (fresh NOTFOUND) and second call (cached NOTFOUND) must return the same: nil items, same error
		cache := sdpcache.NewMemoryCache()
		cachedSrc := DNSAdapter{cache: cache, Servers: s.Servers}
		query := testDNSMissingName

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
		items, err := s.Search(context.Background(), "global", testDNSIPv4, false)

		if err != nil {
			t.Error(err)
		}

		// Reverse lookup resolves to the same fixed address record.
		var foundV4 bool
		var foundV6 bool
		for _, item := range items {
			for _, q := range item.GetLinkedItemQueries() {
				if q.GetQuery().GetQuery() == testDNSIPv4 {
					foundV4 = true
				}
				if q.GetQuery().GetQuery() == testDNSIPv6 {
					foundV6 = true
				}
			}
		}

		if !foundV4 {
			t.Errorf("could not find %s in linked item queries", testDNSIPv4)
		}
		if !foundV6 {
			t.Errorf("could not find %s in linked item queries", testDNSIPv6)
		}

		discovery.TestValidateItems(t, items)
	})
}

func TestDnsGet(t *testing.T) {
	t.Parallel()

	server := newTestDNSServer(t)

	src := DNSAdapter{
		cache:   sdpcache.NewNoOpCache(),
		Servers: []string{server},
	}

	t.Run("working request", func(t *testing.T) {
		item, err := src.Get(context.Background(), "global", testDNSAddressName, false)

		if err != nil {
			t.Fatal(err)
		}

		discovery.TestValidateItem(t, item)
	})

	t.Run("bad dns entry", func(t *testing.T) {
		_, err := src.Get(context.Background(), "global", testDNSMissingName, false)

		if err == nil {
			t.Error("expected error but got nil")
		}

		if _, ok := errors.AsType[*sdp.QueryError](err); !ok {
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
		_, err := src.Get(context.Background(), "something.local.test", testDNSMissingName, false)

		if err == nil {
			t.Error("expected error but got nil")
		}

		if _, ok := errors.AsType[*sdp.QueryError](err); !ok {
			t.Errorf("expected error type to be *sdp.QueryError, got %T", err)
		}
	})

	t.Run("with a CNAME", func(t *testing.T) {
		// When we do a Get on a CNAME, I wan it to work, but only return the
		// first thing
		item, err := src.Get(context.Background(), "global", testDNSAliasName, false)

		if err != nil {
			t.Fatal(err)
		}

		target := item.GetAttributes().GetAttrStruct().GetFields()["target"].GetStringValue()
		if target != testDNSAddressName {
			t.Errorf("expected target to be %s, got %v", testDNSAddressName, target)
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

	server := newTestDNSServer(t)
	src := DNSAdapter{
		cache:   sdpcache.NewNoOpCache(),
		Servers: []string{server},
	}

	// Create a context with a very long deadline to ensure Search creates its own timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// Use a valid, fast DNS query to verify the timeout wrapper doesn't break normal operation
	items, err := src.Search(ctx, "global", testDNSAddressName, false)

	// Should succeed with the fast query
	if err != nil {
		t.Errorf("expected no error for valid query, got: %v", err)
	}

	// Should return at least one item for this known DNS name
	if len(items) == 0 {
		t.Errorf("expected at least one DNS item for %s, got %d", testDNSAddressName, len(items))
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

type timeoutNetError struct{}

func (timeoutNetError) Error() string   { return "i/o timeout" }
func (timeoutNetError) Timeout() bool   { return true }
func (timeoutNetError) Temporary() bool { return true }

func wrapSyscall(err error) error {
	return fmt.Errorf("read udp 169.254.169.253:53: %w", &net.OpError{
		Op:  "read",
		Net: "udp",
		Err: err,
	})
}

func TestRetryDNSQueryTransportFailover(t *testing.T) {
	t.Parallel()

	success := []*sdp.Item{{Type: ItemType, UniqueAttribute: UniqueAttribute, Scope: "global"}}
	servers := []string{"169.254.169.253:53", "1.1.1.1:53"}

	unreachableErrs := []struct {
		name string
		err  error
	}{
		{"ECONNREFUSED", syscall.ECONNREFUSED},
		{"ENETUNREACH", syscall.ENETUNREACH},
		{"EHOSTUNREACH", syscall.EHOSTUNREACH},
	}

	for _, tc := range unreachableErrs {
		t.Run("first server "+tc.name+" then second succeeds", func(t *testing.T) {
			t.Parallel()

			d := DNSAdapter{
				cache:   sdpcache.NewNoOpCache(),
				Servers: servers,
			}
			var attempted []string
			items, err := d.retryDNSQuery(t.Context(), func(_ context.Context, server string) ([]*sdp.Item, error) {
				attempted = append(attempted, server)
				if server == servers[0] {
					return nil, wrapSyscall(tc.err)
				}
				return success, nil
			})
			if err != nil {
				t.Fatalf("expected success after failover, got error: %v", err)
			}
			if len(items) != 1 {
				t.Errorf("expected 1 item, got %d", len(items))
			}
			wantAttempted := []string{servers[0], servers[1]}
			if len(attempted) != len(wantAttempted) {
				t.Fatalf("expected attempted %v, got %v", wantAttempted, attempted)
			}
			for i, got := range attempted {
				if got != wantAttempted[i] {
					t.Errorf("attempted[%d]: expected %q, got %q", i, wantAttempted[i], got)
				}
			}
		})
	}

	t.Run("NOTFOUND from first server does not call second", func(t *testing.T) {
		t.Parallel()

		d := DNSAdapter{
			cache:   sdpcache.NewNoOpCache(),
			Servers: servers,
		}
		notFound := &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOTFOUND,
			ErrorString: "no A or AAAA records found",
			Scope:       "global",
			SourceName:  d.Name(),
			ItemType:    d.Type(),
		}
		var attempted []string
		_, err := d.retryDNSQuery(t.Context(), func(_ context.Context, server string) ([]*sdp.Item, error) {
			attempted = append(attempted, server)
			if server == servers[0] {
				return nil, notFound
			}
			t.Error("second server should not be queried after NOTFOUND")
			return success, nil
		})
		if err == nil {
			t.Fatal("expected NOTFOUND error, got nil")
		}
		var qe *sdp.QueryError
		if !errors.As(err, &qe) || qe.GetErrorType() != sdp.QueryError_NOTFOUND {
			t.Errorf("expected NOTFOUND, got %v", err)
		}
		if len(attempted) != 1 || attempted[0] != servers[0] {
			t.Errorf("expected only first server, got %v", attempted)
		}
	})

	t.Run("other non-transport errors do not rotate", func(t *testing.T) {
		t.Parallel()

		d := DNSAdapter{
			cache:   sdpcache.NewNoOpCache(),
			Servers: servers,
		}
		appErr := errors.New("servfail")
		var attempted []string
		_, err := d.retryDNSQuery(t.Context(), func(_ context.Context, server string) ([]*sdp.Item, error) {
			attempted = append(attempted, server)
			if server == servers[0] {
				return nil, appErr
			}
			t.Error("second server should not be queried after application error")
			return success, nil
		})
		if !errors.Is(err, appErr) {
			t.Errorf("expected %v, got %v", appErr, err)
		}
		if len(attempted) != 1 || attempted[0] != servers[0] {
			t.Errorf("expected only first server, got %v", attempted)
		}
	})

	t.Run("every server refused fails after one pass", func(t *testing.T) {
		t.Parallel()

		d := DNSAdapter{
			cache:   sdpcache.NewNoOpCache(),
			Servers: []string{"10.0.0.1:53", "10.0.0.2:53", "10.0.0.3:53"},
		}
		var attempted []string
		start := time.Now()
		_, err := d.retryDNSQuery(t.Context(), func(_ context.Context, server string) ([]*sdp.Item, error) {
			attempted = append(attempted, server)
			return nil, wrapSyscall(syscall.ECONNREFUSED)
		})
		elapsed := time.Since(start)
		if elapsed > 2*time.Second {
			t.Errorf("expected one pass to finish quickly, took %v", elapsed)
		}
		if !errors.Is(err, syscall.ECONNREFUSED) {
			t.Errorf("expected last error to be ECONNREFUSED, got %v", err)
		}
		if len(attempted) != 3 {
			t.Errorf("expected 3 attempts (one per server), got %d: %v", len(attempted), attempted)
		}
		for i, got := range attempted {
			if got != d.Servers[i] {
				t.Errorf("attempted[%d]: expected %q, got %q", i, d.Servers[i], got)
			}
		}
	})

	t.Run("caller context cancel returns promptly", func(t *testing.T) {
		t.Parallel()

		d := DNSAdapter{
			cache:   sdpcache.NewNoOpCache(),
			Servers: servers,
		}
		ctx, cancel := context.WithCancel(t.Context())
		go func() {
			time.Sleep(50 * time.Millisecond)
			cancel()
		}()
		start := time.Now()
		_, err := d.retryDNSQuery(ctx, func(ctx context.Context, _ string) ([]*sdp.Item, error) {
			<-ctx.Done()
			return nil, ctx.Err()
		})
		elapsed := time.Since(start)
		if elapsed > 2*time.Second {
			t.Errorf("expected cancel to return promptly, took %v", elapsed)
		}
		if !errors.Is(err, context.Canceled) {
			t.Errorf("expected context.Canceled, got %v", err)
		}
	})

	t.Run("timeout on first server rotates to second", func(t *testing.T) {
		t.Parallel()

		d := DNSAdapter{
			cache:   sdpcache.NewNoOpCache(),
			Servers: servers,
		}

		timeoutCases := []struct {
			name string
			err  error
		}{
			{"DeadlineExceeded", context.DeadlineExceeded},
			{"net.Error Timeout", timeoutNetError{}},
		}
		for _, tc := range timeoutCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				var attempted []string
				items, err := d.retryDNSQuery(t.Context(), func(_ context.Context, server string) ([]*sdp.Item, error) {
					attempted = append(attempted, server)
					if server == servers[0] {
						return nil, tc.err
					}
					return success, nil
				})
				if err != nil {
					t.Fatalf("expected success after timeout failover, got error: %v", err)
				}
				if len(items) != 1 {
					t.Errorf("expected 1 item, got %d", len(items))
				}
				if len(attempted) < 2 || attempted[0] != servers[0] || attempted[1] != servers[1] {
					t.Errorf("expected first then second server, got %v", attempted)
				}
			})
		}
	})
}
