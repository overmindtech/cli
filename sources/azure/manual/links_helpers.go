package manual

import (
	"net"
	"strings"

	"github.com/overmindtech/cli/go/sdp-go"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

// appendLinkIfValid appends a linked item query when the value passes validation.
// Skips empty strings and any value in skipValues. If createQuery returns a non-nil query, it is appended.
// Use this for reusable link-creation logic with configurable skip rules (e.g. DNS servers, IP/CIDR prefixes).
func appendLinkIfValid(
	queries *[]*sdp.LinkedItemQuery,
	value string,
	skipValues []string,
	createQuery func(string) *sdp.LinkedItemQuery,
) {
	if value == "" {
		return
	}
	for _, skip := range skipValues {
		if value == skip {
			return
		}
	}
	if q := createQuery(value); q != nil {
		*queries = append(*queries, q)
	}
}

// AppendURILinks appends linked item queries for a URI: HTTP link plus DNS or IP link from the host (with deduplication).
// It mutates linkedItemQueries and the dedupe maps. Skips empty or non-http(s) URIs.
func AppendURILinks(
	linkedItemQueries *[]*sdp.LinkedItemQuery,
	uri string,
	linkedDNSHostnames map[string]struct{},
	seenIPs map[string]struct{},
) {
	if uri == "" || (!strings.HasPrefix(uri, "http://") && !strings.HasPrefix(uri, "https://")) {
		return
	}
	*linkedItemQueries = append(*linkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   stdlib.NetworkHTTP.String(),
			Method: sdp.QueryMethod_SEARCH,
			Query:  uri,
			Scope:  "global",
		},
	})
	hostFromURL := azureshared.ExtractDNSFromURL(uri)
	if hostFromURL != "" {
		hostOnly := hostFromURL
		if h, _, err := net.SplitHostPort(hostFromURL); err == nil {
			hostOnly = h
		}
		if net.ParseIP(hostOnly) != nil {
			if _, seen := seenIPs[hostOnly]; !seen {
				seenIPs[hostOnly] = struct{}{}
				*linkedItemQueries = append(*linkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   stdlib.NetworkIP.String(),
						Method: sdp.QueryMethod_GET,
						Query:  hostOnly,
						Scope:  "global",
					},
				})
			}
		} else {
			if _, seen := linkedDNSHostnames[hostOnly]; !seen {
				linkedDNSHostnames[hostOnly] = struct{}{}
				*linkedItemQueries = append(*linkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   stdlib.NetworkDNS.String(),
						Method: sdp.QueryMethod_SEARCH,
						Query:  hostOnly,
						Scope:  "global",
					},
				})
			}
		}
	}
}

// networkIPQuery returns a linked item query for stdlib.NetworkIP.
func networkIPQuery(query string) *sdp.LinkedItemQuery {
	return &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   stdlib.NetworkIP.String(),
			Method: sdp.QueryMethod_GET,
			Query:  query,
			Scope:  "global",
		},
	}
}
