package adapters

import (
	"context"
	"crypto/tls"
	"encoding/pem"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
	"google.golang.org/protobuf/types/known/structpb"
)

const USER_AGENT_VERSION = "0.1"

// linkLocalRange represents the IPv4 link-local address range (169.254.0.0/16)
// This includes the EC2 metadata service IP (169.254.169.254) and is blocked
// to prevent DNS rebinding attacks and unauthorized metadata service access.
var linkLocalRange = &net.IPNet{
	IP:   net.IPv4(169, 254, 0, 0),
	Mask: net.CIDRMask(16, 32),
}

// isLinkLocalIP checks if an IP address is in the link-local range (169.254.0.0/16)
func isLinkLocalIP(ip net.IP) bool {
	if ip == nil {
		return false
	}
	// Convert IPv4-mapped IPv6 addresses to IPv4
	ip = ip.To4()
	if ip == nil {
		return false
	}
	return linkLocalRange.Contains(ip)
}

// validateHostname checks if a hostname resolves to a link-local IP address.
// This prevents DNS rebinding attacks where a hostname resolves to the EC2
// metadata service or other link-local addresses.
func validateHostname(ctx context.Context, hostname string) error {
	// First check if the hostname is already an IP address
	if ip := net.ParseIP(hostname); ip != nil {
		if isLinkLocalIP(ip) {
			return fmt.Errorf("access to link-local address range (169.254.0.0/16) is blocked for security reasons")
		}
		return nil
	}

	// Resolve the hostname to check if it resolves to a link-local IP
	resolver := net.DefaultResolver
	ips, err := resolver.LookupIPAddr(ctx, hostname)
	if err != nil {
		// If DNS resolution fails, we can't validate, but we should still
		// allow the request to proceed (it will fail later if needed)
		// This prevents blocking legitimate requests due to transient DNS issues
		//nolint:nilerr // Intentionally allowing request to proceed if DNS resolution fails
		return nil
	}

	// Check all resolved IPs
	for _, ipAddr := range ips {
		if isLinkLocalIP(ipAddr.IP) {
			return fmt.Errorf("hostname %s resolves to link-local address %s (169.254.0.0/16), which is blocked for security reasons", hostname, ipAddr.IP)
		}
	}

	return nil
}

type HTTPAdapter struct {
	cacheField sdpcache.Cache // The cache for this adapter (set during creation, can be nil for tests)
}

const httpCacheDuration = 5 * time.Minute

var (
	noOpCacheHTTPOnce sync.Once
	noOpCacheHTTP     sdpcache.Cache
)

func (s *HTTPAdapter) Cache() sdpcache.Cache {
	if s.cacheField == nil {
		noOpCacheHTTPOnce.Do(func() {
			noOpCacheHTTP = sdpcache.NewNoOpCache()
		})
		return noOpCacheHTTP
	}
	return s.cacheField
}

// Type The type of items that this adapter is capable of finding
func (s *HTTPAdapter) Type() string {
	return "http"
}

// Descriptive name for the adapter, used in logging and metadata
func (s *HTTPAdapter) Name() string {
	return "stdlib-http"
}

// Metadata Returns metadata about the adapter
func (s *HTTPAdapter) Metadata() *sdp.AdapterMetadata {
	return httpMetadata
}

var httpMetadata = Metadata.Register(&sdp.AdapterMetadata{
	DescriptiveName: "HTTP Endpoint",
	Type:            "http",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		GetDescription:    "A HTTP endpoint to run a `HEAD` request against",
		Search:            true,
		SearchDescription: "A HTTP URL to search for. Query parameters and fragments will be stripped from the URL before processing.",
	},
	Category:       sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
	PotentialLinks: []string{"ip", "dns", "certificate", "http"},
})

// List of scopes that this adapter is capable of find items for. If the
// adapter supports all scopes the special value `AllScopes` ("*")
// should be used
func (s *HTTPAdapter) Scopes() []string {
	return []string{
		"global", // This is a reserved word meaning that the items should be considered globally unique
	}
}

// Get Get a single item with a given scope and query. The item returned
// should have a UniqueAttributeValue that matches the `query` parameter. The
// ctx parameter contains a golang Context object which should be used to allow
// this adapter to timeout or be cancelled when executing potentially
// long-running actions
func (s *HTTPAdapter) Get(ctx context.Context, scope string, query string, ignoreCache bool) (*sdp.Item, error) {
	if scope != "global" {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: "http is only supported in the 'global' scope",
			Scope:       scope,
		}
	}

	// Validate that the URL doesn't contain query parameters or fragments
	parsedURL, err := url.Parse(query)
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: fmt.Sprintf("invalid URL: %v", err),
			Scope:       scope,
		}
	}

	if parsedURL.RawQuery != "" || parsedURL.Fragment != "" {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "GET method requires clean URLs without query parameters or fragments. Use SEARCH method for URLs with query parameters or fragments.",
			Scope:       scope,
		}
	}

	// Validate hostname to prevent access to link-local addresses (including EC2 metadata service)
	hostname := parsedURL.Hostname()
	if hostname != "" {
		if err := validateHostname(ctx, hostname); err != nil {
			ck := sdpcache.CacheKeyFromParts(s.Name(), sdp.QueryMethod_GET, scope, s.Type(), query)
			err = &sdp.QueryError{
				ErrorType:   sdp.QueryError_OTHER,
				ErrorString: err.Error(),
				Scope:       scope,
			}
			s.Cache().StoreError(ctx, err, httpCacheDuration, ck)
			return nil, err
		}
	}

	var cacheHit bool
	var ck sdpcache.CacheKey
	var cachedItems []*sdp.Item
	var qErr *sdp.QueryError
	var done func()

	cacheHit, ck, cachedItems, qErr, done = s.Cache().Lookup(ctx, s.Name(), sdp.QueryMethod_GET, scope, s.Type(), query, ignoreCache)
	defer done()
	if qErr != nil {
		return nil, qErr
	}
	if cacheHit {
		if len(cachedItems) > 0 {
			return cachedItems[0], nil
		} else {
			return nil, nil
		}
	}

	// Create a client that skips TLS verification since we will want to get the
	// details of the TLS connection rather than stop if it's not trusted. Since
	// we are only running a HEAD request this is unlikely to be a problem
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true, //nolint:gosec // This is fine for a HEAD request
		},
	}
	client := &http.Client{
		Transport: tr,
		// Don't follow redirects, just return the status code directly
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodHead, query, http.NoBody)
	if err != nil {
		err = &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: err.Error(),
			Scope:       scope,
		}
		s.Cache().StoreError(ctx, err, httpCacheDuration, ck)
		return nil, err
	}

	req.Header.Add("User-Agent", fmt.Sprintf("Overmind/%v (%v/%v)", USER_AGENT_VERSION, runtime.GOOS, runtime.GOARCH))
	req.Header.Add("Accept", "*/*")

	var res *http.Response

	res, err = client.Do(req)

	if err != nil {
		err = &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: err.Error(),
			Scope:       scope,
		}
		s.Cache().StoreError(ctx, err, httpCacheDuration, ck)
		return nil, err
	}

	// Clean up connections once we're done
	defer client.CloseIdleConnections()

	// Convert headers from map[string][]string to map[string]string. This means
	// that headers that were returned many times will end up with their values
	// comma-separated
	headersMap := make(map[string]string)
	for header, values := range res.Header {
		headersMap[header] = strings.Join(values, ", ")
	}

	// Convert the attributes from a golang map, to the structure required for
	// the SDP protocol
	attributes, err := sdp.ToAttributes(map[string]interface{}{
		"name":             query,
		"status":           res.StatusCode,
		"statusString":     res.Status,
		"proto":            res.Proto,
		"headers":          headersMap,
		"transferEncoding": res.Request.TransferEncoding,
	})

	if err != nil {
		err = &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: err.Error(),
			Scope:       scope,
		}
		s.Cache().StoreError(ctx, err, httpCacheDuration, ck)
		return nil, err
	}

	item := sdp.Item{
		Type:            "http",
		UniqueAttribute: "name",
		Attributes:      attributes,
		Scope:           "global",
	}

	if ip := net.ParseIP(req.URL.Hostname()); ip != nil {
		// If the host is an IP, add a linked item to that IP address
		item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   "ip",
				Method: sdp.QueryMethod_GET,
				Query:  ip.String(),
				Scope:  "global",
			},
			BlastPropagation: &sdp.BlastPropagation{
				// IPs always linked
				In:  true,
				Out: true,
			},
		})
	} else {
		// If the host is not an ip, try to resolve via DNS
		item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   "dns",
				Method: sdp.QueryMethod_SEARCH,
				Query:  req.URL.Hostname(),
				Scope:  "global",
			},
			BlastPropagation: &sdp.BlastPropagation{
				// DNS always linked
				In:  true,
				Out: true,
			},
		})
	}

	if tlsState := res.TLS; tlsState != nil {
		var version string

		// Extract TLS version as a string
		switch tlsState.Version {
		case tls.VersionTLS10:
			version = "TLSv1.0"
		case tls.VersionTLS11:
			version = "TLSv1.1"
		case tls.VersionTLS12:
			version = "TLSv1.2"
		case tls.VersionTLS13:
			version = "TLSv1.3"
		default:
			version = "unknown"
		}

		attributes.Set("tls", map[string]interface{}{
			"version":     version,
			"certificate": CertToName(tlsState.PeerCertificates[0]),
			"serverName":  tlsState.ServerName,
		})

		if len(tlsState.PeerCertificates) > 0 {
			// Create a PEM bundle and then linked item request
			var certs []string

			for _, cert := range tlsState.PeerCertificates {
				block := pem.Block{
					Type:  "CERTIFICATE",
					Bytes: cert.Raw,
				}

				certs = append(certs, string(pem.EncodeToMemory(&block)))
			}

			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "certificate",
					Method: sdp.QueryMethod_SEARCH,
					Query:  strings.Join(certs, "\n"),
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					// Changing the cert will affect the HTTP endpoint
					In: true,
					// The HTTP endpoint won't affect the cert
					Out: false,
				},
			})
		}
	}
	// Detect redirect and add a linked item for the redirect target
	if res.StatusCode >= 300 && res.StatusCode < 400 {
		if loc := res.Header.Get("Location"); loc != "" {
			item.Attributes.AttrStruct.Fields["location"] = &structpb.Value{
				Kind: &structpb.Value_StringValue{
					StringValue: loc,
				},
			}
			locURL, err := url.Parse(loc)
			if err != nil {
				item.Attributes.AttrStruct.Fields["location-error"] = &structpb.Value{
					Kind: &structpb.Value_StringValue{
						StringValue: err.Error(),
					},
				}
			} else {
				// Resolve relative URLs against the original request URL
				resolvedURL := parsedURL.ResolveReference(locURL)

				// Validate redirect target to prevent redirects to link-local addresses
				redirectHostname := resolvedURL.Hostname()
				if redirectHostname != "" {
					if err := validateHostname(ctx, redirectHostname); err != nil {
						// Don't fail the entire request, but mark the redirect as invalid
						item.Attributes.AttrStruct.Fields["location-error"] = &structpb.Value{
							Kind: &structpb.Value_StringValue{
								StringValue: fmt.Sprintf("redirect blocked: %v", err),
							},
						}
					} else {
						// Don't include query string and fragment in the linked item.
						// This leads to too many items, like auth redirect errors, that
						// do not provide value.
						resolvedURL.Fragment = ""
						resolvedURL.RawQuery = ""
						item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   "http",
								Method: sdp.QueryMethod_SEARCH,
								Query:  resolvedURL.String(),
								Scope:  scope,
							},
							BlastPropagation: &sdp.BlastPropagation{
								// Redirects are tightly coupled
								In:  true,
								Out: true,
							},
						})
					}
				}
			}
		}
	}
	s.Cache().StoreItem(ctx, &item, httpCacheDuration, ck)
	return &item, nil
}

// Search takes a URL, strips query parameters and fragments, and returns the HTTP item
func (s *HTTPAdapter) Search(ctx context.Context, scope string, query string, ignoreCache bool) ([]*sdp.Item, error) {
	if scope != "global" {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: "http is only supported in the 'global' scope",
			Scope:       scope,
		}
	}

	// Parse the URL and strip query parameters and fragments
	parsedURL, err := url.Parse(query)
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: fmt.Sprintf("invalid URL: %v", err),
			Scope:       scope,
		}
	}

	// Strip query parameters and fragments
	parsedURL.RawQuery = ""
	parsedURL.Fragment = ""
	cleanURL := parsedURL.String()

	// Use the existing Get method to retrieve the item
	item, err := s.Get(ctx, scope, cleanURL, ignoreCache)
	if err != nil {
		return nil, err
	}

	if item == nil {
		return []*sdp.Item{}, nil
	}

	return []*sdp.Item{item}, nil
}

// List is not implemented for HTTP as this would require scanning infinitely many
// endpoints or something, doesn't really make sense
func (s *HTTPAdapter) List(ctx context.Context, scope string, ignoreCache bool) ([]*sdp.Item, error) {
	items := make([]*sdp.Item, 0)

	return items, nil
}

// Weight Returns the priority weighting of items returned by this adapter.
// This is used to resolve conflicts where two adapters of the same type
// return an item for a GET request. In this instance only one item can be
// sen on, so the one with the higher weight value will win.
func (s *HTTPAdapter) Weight() int {
	return 100
}
