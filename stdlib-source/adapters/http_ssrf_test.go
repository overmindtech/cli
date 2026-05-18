package adapters

import (
	"context"
	"errors"
	"net"
	"testing"
)

func TestDefaultIPPolicy(t *testing.T) {
	t.Parallel()
	policy := defaultIPPolicy{}

	tests := []struct {
		name    string
		ip      string
		blocked bool
		reason  string
	}{
		// Unspecified (0.0.0.0 reaches localhost on Linux/macOS)
		{"ipv4 unspecified 0.0.0.0", "0.0.0.0", true, "unspecified"},
		{"ipv6 unspecified ::", "::", true, "unspecified"},

		// IPv4 loopback
		{"ipv4 loopback 127.0.0.1", "127.0.0.1", true, "loopback"},
		{"ipv4 loopback 127.255.255.254", "127.255.255.254", true, "loopback"},

		// IPv4 link-local
		{"ipv4 link-local 169.254.0.1", "169.254.0.1", true, "link-local"},
		{"ipv4 link-local metadata 169.254.169.254", "169.254.169.254", true, "link-local"},

		// IPv4 private (RFC1918)
		{"ipv4 private 10.0.0.1", "10.0.0.1", true, "private"},
		{"ipv4 private 172.16.0.1", "172.16.0.1", true, "private"},
		{"ipv4 private 172.31.255.254", "172.31.255.254", true, "private"},
		{"ipv4 private 192.168.1.1", "192.168.1.1", true, "private"},

		// IPv4 carrier-grade NAT (RFC6598)
		{"ipv4 CGNAT 100.64.0.1", "100.64.0.1", true, "carrier-grade NAT"},
		{"ipv4 CGNAT 100.127.255.254", "100.127.255.254", true, "carrier-grade NAT"},

		// IPv4 public (should be allowed)
		{"ipv4 public 8.8.8.8", "8.8.8.8", false, ""},
		{"ipv4 public 1.1.1.1", "1.1.1.1", false, ""},
		{"ipv4 public 100.128.0.1", "100.128.0.1", false, ""},
		{"ipv4 just-outside-private 172.32.0.1", "172.32.0.1", false, ""},

		// IPv6 loopback
		{"ipv6 loopback ::1", "::1", true, "loopback"},

		// IPv6 link-local
		{"ipv6 link-local fe80::1", "fe80::1", true, "link-local"},

		// IPv6 unique-local (ULA fc00::/7)
		{"ipv6 ULA fd00::1", "fd00::1", true, "private"},
		{"ipv6 ULA fc00::1", "fc00::1", true, "private"},

		// IPv6 public (should be allowed)
		{"ipv6 public 2001:4860:4860::8888", "2001:4860:4860::8888", false, ""},

		// IPv4-mapped IPv6 — should be unwrapped and checked against v4 rules
		{"ipv4-mapped-ipv6 loopback ::ffff:127.0.0.1", "::ffff:127.0.0.1", true, "loopback"},
		{"ipv4-mapped-ipv6 private ::ffff:10.0.0.1", "::ffff:10.0.0.1", true, "private"},
		{"ipv4-mapped-ipv6 link-local ::ffff:169.254.169.254", "::ffff:169.254.169.254", true, "link-local"},
		{"ipv4-mapped-ipv6 CGNAT ::ffff:100.64.0.1", "::ffff:100.64.0.1", true, "carrier-grade NAT"},
		{"ipv4-mapped-ipv6 public ::ffff:8.8.8.8", "::ffff:8.8.8.8", false, ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ip := net.ParseIP(tc.ip)
			if ip == nil {
				t.Fatalf("failed to parse IP %q", tc.ip)
			}
			err := policy.CheckIP(ip)
			if tc.blocked {
				if err == nil {
					t.Errorf("expected IP %s to be blocked (%s), got nil error", tc.ip, tc.reason)
				} else if !errors.Is(err, ErrIPBlocked) {
					t.Errorf("expected ErrIPBlocked for %s, got: %v", tc.ip, err)
				}
			} else {
				if err != nil {
					t.Errorf("expected IP %s to be allowed, got error: %v", tc.ip, err)
				}
			}
		})
	}

	t.Run("nil IP", func(t *testing.T) {
		t.Parallel()
		err := policy.CheckIP(nil)
		if !errors.Is(err, ErrIPBlocked) {
			t.Errorf("expected ErrIPBlocked for nil IP, got: %v", err)
		}
	})
}

func TestAllowLoopbackPolicy(t *testing.T) {
	t.Parallel()
	policy := allowLoopbackPolicy{}

	tests := []struct {
		name    string
		ip      string
		blocked bool
	}{
		// Loopback is allowed
		{"ipv4 loopback 127.0.0.1", "127.0.0.1", false},
		{"ipv6 loopback ::1", "::1", false},

		// Everything else still blocked
		{"ipv4 private 10.0.0.1", "10.0.0.1", true},
		{"ipv4 link-local 169.254.169.254", "169.254.169.254", true},
		{"ipv4 CGNAT 100.64.0.1", "100.64.0.1", true},
		{"ipv6 ULA fd00::1", "fd00::1", true},

		// Public still allowed
		{"ipv4 public 8.8.8.8", "8.8.8.8", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ip := net.ParseIP(tc.ip)
			if ip == nil {
				t.Fatalf("failed to parse IP %q", tc.ip)
			}
			err := policy.CheckIP(ip)
			if tc.blocked && err == nil {
				t.Errorf("expected IP %s to be blocked, got nil error", tc.ip)
			}
			if !tc.blocked && err != nil {
				t.Errorf("expected IP %s to be allowed, got error: %v", tc.ip, err)
			}
		})
	}

	t.Run("nil IP", func(t *testing.T) {
		t.Parallel()
		err := policy.CheckIP(nil)
		if !errors.Is(err, ErrIPBlocked) {
			t.Errorf("expected ErrIPBlocked for nil IP, got: %v", err)
		}
	})
}

func TestValidateHost(t *testing.T) {
	t.Parallel()
	policy := defaultIPPolicy{}

	t.Run("IP literal blocked", func(t *testing.T) {
		t.Parallel()
		err := validateHost(context.Background(), "10.0.0.1", policy, nil)
		if !errors.Is(err, ErrIPBlocked) {
			t.Errorf("expected ErrIPBlocked for 10.0.0.1, got: %v", err)
		}
	})

	t.Run("IP literal allowed", func(t *testing.T) {
		t.Parallel()
		err := validateHost(context.Background(), "8.8.8.8", policy, nil)
		if err != nil {
			t.Errorf("expected 8.8.8.8 to be allowed, got: %v", err)
		}
	})

	t.Run("unresolvable hostname returns error", func(t *testing.T) {
		t.Parallel()
		err := validateHost(context.Background(), "this-hostname-does-not-exist.invalid", policy, nil)
		if err == nil {
			t.Error("expected error for unresolvable hostname, got nil")
		}
	})
}
