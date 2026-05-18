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
	"time"

	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
	"google.golang.org/protobuf/types/known/structpb"
)

const USER_AGENT_VERSION = "0.1"

type HTTPAdapter struct {
	cache    sdpcache.Cache // This is mandatory
	ipPolicy IPPolicy       // nil → defaultIPPolicy (production); tests inject allowLoopbackPolicy
	resolver *net.Resolver  // nil → net.DefaultResolver; tests inject a stub
}

func (s *HTTPAdapter) policy() IPPolicy {
	if s.ipPolicy == nil {
		return defaultIPPolicy{}
	}
	return s.ipPolicy
}

const httpCacheDuration = 5 * time.Minute

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

	// Pre-validate hostname against the SSRF policy. DialContext is the
	// authoritative enforcement point; this early check lets us return a
	// clean error without opening a socket.
	hostname := parsedURL.Hostname()
	if hostname != "" {
		if err := validateHost(ctx, hostname, s.policy(), s.resolver); err != nil {
			ck := sdpcache.CacheKeyFromParts(s.Name(), sdp.QueryMethod_GET, scope, s.Type(), query)
			qe := &sdp.QueryError{
				ErrorType:   sdp.QueryError_OTHER,
				ErrorString: err.Error(),
				Scope:       scope,
			}
			s.cache.StoreUnavailableItem(ctx, qe, httpCacheDuration, ck)
			return nil, qe
		}
	}

	var cacheHit bool
	var ck sdpcache.CacheKey
	var cachedItems []*sdp.Item
	var qErr *sdp.QueryError
	var done func()

	cacheHit, ck, cachedItems, qErr, done = s.cache.Lookup(ctx, s.Name(), sdp.QueryMethod_GET, scope, s.Type(), query, ignoreCache)
	defer done()
	if qErr != nil {
		return nil, qErr
	}
	if cacheHit {
		// Get only caches a single item or NOTFOUND (via StoreUnavailableItem). Guard against empty slice for defensive safety (e.g. cache corruption).
		if len(cachedItems) > 0 {
			return cachedItems[0], nil
		}
		return nil, nil
	}

	client := &http.Client{
		Transport: newSecureTransport(s.policy(), s.resolver),
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
		s.cache.StoreUnavailableItem(ctx, err, httpCacheDuration, ck)
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
		s.cache.StoreUnavailableItem(ctx, err, httpCacheDuration, ck)
		return nil, err
	}

	// Clean up connections once we're done
	defer client.CloseIdleConnections()
	defer res.Body.Close()

	// Treat HTTP 404 and 410 as not-found; cache to avoid repeated requests.
	if res.StatusCode == http.StatusNotFound || res.StatusCode == http.StatusGone {
		notFoundErr := &sdp.QueryError{
			ErrorType:     sdp.QueryError_NOTFOUND,
			ErrorString:   fmt.Sprintf("HTTP %s for %s", res.Status, query),
			Scope:         scope,
			SourceName:    s.Name(),
			ItemType:      s.Type(),
			ResponderName: s.Name(),
		}
		s.cache.StoreUnavailableItem(ctx, notFoundErr, httpCacheDuration, ck)
		return nil, notFoundErr
	}

	// Convert headers from map[string][]string to map[string]string. This means
	// that headers that were returned many times will end up with their values
	// comma-separated
	headersMap := make(map[string]string)
	for header, values := range res.Header {
		headersMap[header] = strings.Join(values, ", ")
	}

	// Convert the attributes from a golang map, to the structure required for
	// the SDP protocol
	attributes, err := sdp.ToAttributes(map[string]any{
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
		s.cache.StoreUnavailableItem(ctx, err, httpCacheDuration, ck)
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

		attributes.Set("tls", map[string]any{
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

				redirectHostname := resolvedURL.Hostname()
				if redirectHostname != "" {
					if err := validateHost(ctx, redirectHostname, s.policy(), s.resolver); err != nil {
						item.Attributes.AttrStruct.Fields["location-error"] = &structpb.Value{
							Kind: &structpb.Value_StringValue{
								StringValue: fmt.Sprintf("redirect blocked: %v", err),
							},
						}
					} else {
						resolvedURL.Fragment = ""
						resolvedURL.RawQuery = ""
						item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   "http",
								Method: sdp.QueryMethod_SEARCH,
								Query:  resolvedURL.String(),
								Scope:  scope,
							},
						})
					}
				}
			}
		}
	}
	s.cache.StoreItem(ctx, &item, httpCacheDuration, ck)
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
		// Return (nil, error) for NOTFOUND so cache hit and fresh lookup behave the same
		return nil, err
	}
	if item == nil {
		// Get can return (nil, nil) on the defensive path when cache reports hit but cachedItems is empty (e.g. cache corruption).
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
