package adapters

import (
	"context"
	"crypto/tls"
	"encoding/pem"
	"fmt"
	"net"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
)

const USER_AGENT_VERSION = "0.1"

type HTTPAdapter struct {
	cache       *sdpcache.Cache // The sdpcache of this adapter
	cacheInitMu sync.Mutex      // Mutex to ensure cache is only initialised once
}

const httpCacheDuration = 5 * time.Minute

func (s *HTTPAdapter) ensureCache() {
	s.cacheInitMu.Lock()
	defer s.cacheInitMu.Unlock()

	if s.cache == nil {
		s.cache = sdpcache.NewCache()
	}
}

func (s *HTTPAdapter) Cache() *sdpcache.Cache {
	s.ensureCache()
	return s.cache
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
		Get:            true,
		GetDescription: "A HTTP endpoint to run a `HEAD` request against",
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

	s.ensureCache()
	cacheHit, ck, cachedItems, qErr := s.cache.Lookup(ctx, s.Name(), sdp.QueryMethod_GET, scope, s.Type(), query, ignoreCache)
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
		s.cache.StoreError(err, httpCacheDuration, ck)
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
		s.cache.StoreError(err, httpCacheDuration, ck)
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
		s.cache.StoreError(err, httpCacheDuration, ck)
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
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "http",
					Method: sdp.QueryMethod_GET,
					Query:  loc,
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

	s.cache.StoreItem(&item, httpCacheDuration, ck)

	return &item, nil
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
