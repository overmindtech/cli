package manual

import (
	"net"

	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources/stdlib"
)

// appendDNSServerLinkIfValid appends a linked item query for a DNS server string:
// stdlib.NetworkIP for IP addresses, stdlib.NetworkDNS for hostnames.
// Skips empty strings and any value in skipValues (e.g. "AzureProvidedDNS" for Azure managed DNS).
func appendDNSServerLinkIfValid(queries *[]*sdp.LinkedItemQuery, server string, skipValues ...string) {
	appendLinkIfValid(queries, server, skipValues, func(s string) *sdp.LinkedItemQuery {
		if net.ParseIP(s) != nil {
			return networkIPQuery(s)
		}
		return &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   stdlib.NetworkDNS.String(),
				Method: sdp.QueryMethod_SEARCH,
				Query:  s,
				Scope:  "global",
			},
			BlastPropagation: &sdp.BlastPropagation{In: true, Out: true},
		}
	})
}
