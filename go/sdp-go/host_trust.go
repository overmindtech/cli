package sdp

import (
	"fmt"
	"net"
	"net/url"
	"slices"
	"strings"
)

var trustedDomainSuffixes = []string{
	".overmind.tech",
	".overmind-demo.com",
}

var trustedExactDomains = []string{
	"overmind.tech",
	"overmind-demo.com",
}

// IsTrustedHost reports whether the given host (without port) belongs
// to a known Overmind domain (*.overmind.tech, *.overmind-demo.com) or is a
// local address. Callers should prompt for explicit user confirmation before
// sending credentials to untrusted hosts.
func IsTrustedHost(hostname string) bool {
	hostname = strings.ToLower(hostname)

	if IsLocalHost(hostname) {
		return true
	}

	if slices.Contains(trustedExactDomains, hostname) {
		return true
	}

	for _, suffix := range trustedDomainSuffixes {
		if strings.HasSuffix(hostname, suffix) {
			return true
		}
	}

	return false
}

// IsLocalHost reports whether the given host (without port) resolves
// to a loopback address. HTTP (non-TLS) is only acceptable for local hosts.
func IsLocalHost(hostname string) bool {
	if hostname == "localhost" {
		return true
	}
	ip := net.ParseIP(hostname)
	return ip != nil && ip.IsLoopback()
}

// ValidateAppURL parses appURLString and enforces that non-local hosts use
// HTTPS. It returns the parsed URL or an error.
func ValidateAppURL(appURLString string) (*url.URL, error) {
	appURL, err := url.Parse(appURLString)
	if err != nil {
		return nil, fmt.Errorf("invalid app URL %q: %w", appURLString, err)
	}

	if !IsLocalHost(appURL.Hostname()) && appURL.Scheme != "https" {
		return nil, fmt.Errorf(
			"HTTPS is required for non-local hosts (got %s://%s); "+
				"use https:// or target localhost for development",
			appURL.Scheme, appURL.Host,
		)
	}

	return appURL, nil
}
