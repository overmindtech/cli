package adapters

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"
)

// ErrIPBlocked is returned by IPPolicy implementations when an IP is in a
// disallowed range. Callers use errors.Is to detect policy rejections vs.
// transport errors.
var ErrIPBlocked = errors.New("ip address blocked by SSRF policy")

// IPPolicy decides whether an IP address may be dialed. Implementations are
// expected to be cheap, allocation-free, and safe for concurrent use.
type IPPolicy interface {
	CheckIP(ip net.IP) error
}

// cgnatRange is RFC 6598 100.64.0.0/10 — not covered by net.IP.IsPrivate.
var cgnatRange = &net.IPNet{
	IP:   net.IPv4(100, 64, 0, 0),
	Mask: net.CIDRMask(10, 32),
}

// defaultIPPolicy is the production policy. It blocks the IP ranges that
// ENG-4205 enumerates: loopback, link-local, RFC1918 private, RFC6598 CGNAT,
// and the IPv6 equivalents (loopback, link-local, ULA fc00::/7).
type defaultIPPolicy struct{}

func (defaultIPPolicy) CheckIP(ip net.IP) error {
	if ip == nil {
		return fmt.Errorf("%w: nil ip", ErrIPBlocked)
	}
	// Normalise IPv4-mapped IPv6 (::ffff:a.b.c.d) so a single check covers both.
	if v4 := ip.To4(); v4 != nil {
		ip = v4
	}
	switch {
	case ip.IsUnspecified():
		return fmt.Errorf("%w: %s is unspecified", ErrIPBlocked, ip)
	case ip.IsLoopback():
		return fmt.Errorf("%w: %s is loopback", ErrIPBlocked, ip)
	case ip.IsLinkLocalUnicast():
		return fmt.Errorf("%w: %s is link-local", ErrIPBlocked, ip)
	case ip.IsPrivate():
		return fmt.Errorf("%w: %s is private", ErrIPBlocked, ip)
	case ip.To4() != nil && cgnatRange.Contains(ip):
		return fmt.Errorf("%w: %s is carrier-grade NAT", ErrIPBlocked, ip)
	}
	return nil
}

// allowLoopbackPolicy is a test-only policy that wraps defaultIPPolicy but
// permits loopback (127.0.0.0/8 and ::1). This lets test files construct
// HTTPAdapter instances that can still talk to httptest.NewServer on localhost.
type allowLoopbackPolicy struct{}

func (allowLoopbackPolicy) CheckIP(ip net.IP) error {
	if ip == nil {
		return fmt.Errorf("%w: nil ip", ErrIPBlocked)
	}
	if v4 := ip.To4(); v4 != nil {
		ip = v4
	}
	if ip.IsLoopback() {
		return nil
	}
	return (defaultIPPolicy{}).CheckIP(ip)
}

// validateHost checks a hostname against the given IPPolicy. If hostname is an
// IP literal it checks directly; otherwise it resolves via the supplied
// resolver and checks every returned address.
func validateHost(ctx context.Context, hostname string, policy IPPolicy, resolver *net.Resolver) error {
	if ip := net.ParseIP(hostname); ip != nil {
		return policy.CheckIP(ip)
	}

	if resolver == nil {
		resolver = net.DefaultResolver
	}
	ips, err := resolver.LookupIPAddr(ctx, hostname)
	if err != nil {
		return fmt.Errorf("dns resolution failed for %s: %w", hostname, err)
	}
	for _, ipAddr := range ips {
		if err := policy.CheckIP(ipAddr.IP); err != nil {
			return fmt.Errorf("hostname %s resolves to blocked address: %w", hostname, err)
		}
	}
	return nil
}

// newSecureTransport builds an *http.Transport with a DialContext hook that
// enforces the given IPPolicy at connection time (preventing DNS rebinding).
func newSecureTransport(policy IPPolicy, resolver *net.Resolver) *http.Transport {
	if resolver == nil {
		resolver = net.DefaultResolver
	}
	base := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}
	return &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true, //nolint:gosec // adapter inspects TLS certificate details via HEAD request, not trusting the content
		},
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}
			// Resolve once here, validate each candidate, and dial the
			// validated IP literal — not the hostname — so DNS cannot
			// rebind between validation and connect.
			ips, err := resolver.LookupIPAddr(ctx, host)
			if err != nil {
				return nil, err
			}
			var lastErr error
			for _, ipAddr := range ips {
				if perr := policy.CheckIP(ipAddr.IP); perr != nil {
					lastErr = perr
					continue
				}
				conn, derr := base.DialContext(ctx, network, net.JoinHostPort(ipAddr.IP.String(), port))
				if derr == nil {
					return conn, nil
				}
				lastErr = derr
				if errors.Is(derr, context.Canceled) || errors.Is(derr, context.DeadlineExceeded) {
					return nil, derr
				}
			}
			if lastErr == nil {
				lastErr = fmt.Errorf("no addresses resolved for %s", host)
			}
			return nil, fmt.Errorf("failed to dial %s: %w", host, lastErr)
		},
	}
}
