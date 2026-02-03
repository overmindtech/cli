package manual

import (
	"github.com/overmindtech/cli/sdp-go"
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

// networkIPQuery returns a linked item query for stdlib.NetworkIP.
func networkIPQuery(query string) *sdp.LinkedItemQuery {
	return &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   stdlib.NetworkIP.String(),
			Method: sdp.QueryMethod_GET,
			Query:  query,
			Scope:  "global",
		},
		BlastPropagation: &sdp.BlastPropagation{In: true, Out: true},
	}
}
