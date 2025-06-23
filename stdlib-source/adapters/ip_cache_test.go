package adapters

import (
	"net"
	"testing"
	"time"
)

func TestIPCaching(t *testing.T) {
	cache := NewIPCache[string]()

	// Store a number of ranges
	ranges := []struct {
		Range string
		Value string
	}{
		{
			Range: "10.0.0.0/24",
			Value: "super-local",
		},
		{
			// Goes up to 10.0.63.255
			Range: "10.0.0.0/18",
			Value: "semi-local",
		},
		{
			Range: "10.0.0.0/8",
			Value: "local",
		},
	}

	for _, r := range ranges {
		_, network, err := net.ParseCIDR(r.Range)

		if err != nil {
			t.Fatal(err)
		}

		cache.Store(network, r.Value, 10*time.Minute)
	}

	expectations := []struct {
		IP    string
		Value string
	}{
		{
			IP:    "10.0.0.1",
			Value: "super-local",
		},
		{
			IP:    "10.0.20.20",
			Value: "semi-local",
		},
		{
			IP:    "10.23.54.76",
			Value: "local",
		},
	}

	for _, e := range expectations {
		ip := net.ParseIP(e.IP)

		value, ok := cache.SearchIP(ip)

		if !ok {
			t.Fatal("Expected to find a value")
		}

		if value != e.Value {
			t.Errorf("Expected to find %v, got %v", e.Value, value)
		}
	}

	// Test for something that should not exist
	ip := net.ParseIP("86.4.78.2")

	_, ok := cache.SearchIP(ip)

	if ok {
		t.Error("Expected not to find a value for a public IP")
	}
}

func TestIPCachePurge(t *testing.T) {
	cache := NewIPCache[string]()

	start := time.Now()

	// Store a number of ranges
	_, a, _ := net.ParseCIDR("10.0.0.0/24")
	cache.Store(a, "super-local", 1*time.Second)
	_, b, _ := net.ParseCIDR("10.0.0.0/18")
	cache.Store(b, "semi-local", 2*time.Second)
	_, c, _ := net.ParseCIDR("10.0.0.0/8")
	cache.Store(c, "local", 3*time.Second)

	// Lookup a local IP, this should be served from the most local cache
	// entry
	result, found := cache.SearchIP(net.ParseIP("10.0.0.1"))

	if !found {
		t.Fatal("Expected to find a value")
	}

	if result != "super-local" {
		t.Errorf("Expected to find super-local, got %v", result)
	}

	// Expire the first (most specific) entry
	numExpired := cache.Expire(start.Add(1100 * time.Millisecond))

	if numExpired != 1 {
		t.Errorf("Expected 1 entry to expire, got %v", numExpired)
	}

	// Lookup a local IP, this should be served from the next most local cache
	// entry
	result, found = cache.SearchIP(net.ParseIP("10.0.0.1"))

	if !found {
		t.Fatal("Expected to find a value")
	}

	if result != "semi-local" {
		t.Errorf("Expected to find semi-local, got %v", result)
	}

	// Expire the second entry
	numExpired = cache.Expire(start.Add(2100 * time.Millisecond))

	if numExpired != 1 {
		t.Errorf("Expected 1 entry to expire, got %v", numExpired)
	}

	// Lookup a local IP, this should be served from the local entry
	result, found = cache.SearchIP(net.ParseIP("10.0.0.1"))

	if !found {
		t.Fatal("Expected to find a value")
	}

	if result != "local" {
		t.Errorf("Expected to find local, got %v", result)
	}

	// Expire the third entry
	numExpired = cache.Expire(start.Add(3100 * time.Millisecond))

	if numExpired != 1 {
		t.Errorf("Expected 1 entry to expire, got %v", numExpired)
	}

	// Lookup a local IP the cache should now be empty
	_, found = cache.SearchIP(net.ParseIP("10.0.0.1"))

	if found {
		t.Fatal("Expected not to find a value")
	}
}

func TestParseIPWithCIDR(t *testing.T) {
	ip := net.ParseIP("10.0.0.1/32")

	t.Log(ip)
}
