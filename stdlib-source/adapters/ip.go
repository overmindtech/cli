package adapters

import (
	"context"
	"fmt"
	"net"

	"github.com/overmindtech/cli/sdp-go"
)

// IPAdapter struct on which all methods are registered
type IPAdapter struct{}

// Type is the type of items that this returns
func (bc *IPAdapter) Type() string {
	return "ip"
}

// Name Returns the name of the backend
func (bc *IPAdapter) Name() string {
	return "stdlib-ip"
}

// Weighting of duplicate adapters
func (s *IPAdapter) Weight() int {
	return 100
}

func (s *IPAdapter) Metadata() *sdp.AdapterMetadata {
	return ipMetadata
}

var ipMetadata = Metadata.Register(&sdp.AdapterMetadata{
	DescriptiveName: "IP Address",
	Type:            "ip",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:            true,
		GetDescription: "An ipv4 or ipv6 address",
	},
	PotentialLinks: []string{"dns", "rdap-ip-network"},
	Category:       sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
})

// List of scopes that this adapter is capable of find items for
func (s *IPAdapter) Scopes() []string {
	return []string{
		// This supports all scopes since there might be local IPs that need
		// to have a different scope. E.g. 127.0.0.1 is a different logical
		// address per computer since it refers to "itself" This means we
		// definitely don't want all thing that reference 127.0.0.1 linked
		// together, only those in the same scope
		//
		// TODO: Make a recommendation for what the scope should be when
		// looking up an IP in the local range. It's possible that an org could
		// have the address (10.2.56.1) assigned to many devices (hopefully not,
		// but I have seen it happen) and we would therefore want those IPs to
		// have different scopes as they don't refer to the same thing
		sdp.WILDCARD,
	}
}

// Get gets information about a single IP This expects an IP in a format that
// can be parsed by net.ParseIP() such as "192.0.2.1", "2001:db8::68" or
// "::ffff:192.0.2.1". It returns some useful information about that IP but this
// is all just information that is inherent in the IP itself, it doesn't look
// anything up externally
//
// The purpose of this is mainly to provide a node in the graph that many things
// can be linked to, rather than being particularly useful on its own
func (bc *IPAdapter) Get(ctx context.Context, scope string, query string, ignoreCache bool) (*sdp.Item, error) {
	var ip net.IP
	var err error
	var attributes *sdp.ItemAttributes
	var isGlobalIP bool

	ip = net.ParseIP(query)

	if ip == nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOTFOUND,
			ErrorString: fmt.Sprintf("%v is not a valid IP", query),
			Scope:       scope,
		}
	}

	isGlobalIP = IsGlobalScopeIP(ip)

	// If the query was executed with a wildcard, and the scope is global, we
	// might was well set it. If it's not then we have no way to determine the
	// scope so we need to return an error
	if scope == sdp.WILDCARD {
		if isGlobalIP {
			scope = "global"
		} else {
			return nil, &sdp.QueryError{
				ErrorType:   sdp.QueryError_NOTFOUND,
				ErrorString: fmt.Sprintf("%v is not a globally-unique IP and therefore could exist in every scope. Query with a wildcard does not work for non-global IPs", query),
				Scope:       scope,
			}
		}
	}

	if scope == "global" {
		if !IsGlobalScopeIP(ip) {
			return nil, &sdp.QueryError{
				ErrorType:   sdp.QueryError_NOTFOUND,
				ErrorString: fmt.Sprintf("%v is not a valid ip withing the global scope. It must be request with some other scope", query),
				Scope:       scope,
			}
		}
	} else {
		// If the scope is non-global, ensure that the IP is not globally unique unique
		if IsGlobalScopeIP(ip) {
			return nil, &sdp.QueryError{
				ErrorType:   sdp.QueryError_NOTFOUND,
				ErrorString: fmt.Sprintf("%v is a globally-unique IP and therefore only exists in the global scope. Note that private IP ranges are also considered 'global' for convenience", query),
				Scope:       scope,
			}
		}
	}

	attributes, err = sdp.ToAttributes(map[string]interface{}{
		"ip":                      ip.String(),
		"unspecified":             ip.IsUnspecified(),
		"loopback":                ip.IsLoopback(),
		"private":                 ip.IsPrivate(),
		"multicast":               ip.IsMulticast(),
		"interfaceLocalMulticast": ip.IsInterfaceLocalMulticast(),
		"linkLocalMulticast":      ip.IsLinkLocalMulticast(),
		"linkLocalUnicast":        ip.IsLinkLocalUnicast(),
	})

	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: err.Error(),
			Scope:       scope,
		}
	}

	return &sdp.Item{
		Type:            "ip",
		UniqueAttribute: "ip",
		Attributes:      attributes,
		Scope:           scope,
		LinkedItemQueries: []*sdp.LinkedItemQuery{
			// Reverse DNS
			{
				Query: &sdp.Query{
					Type:   "dns",
					Method: sdp.QueryMethod_SEARCH,
					Query:  ip.String(),
					Scope:  "global",
				},
				BlastPropagation: &sdp.BlastPropagation{
					// DNS always linked
					In:  true,
					Out: true,
				},
			},
			{
				// RDAP
				Query: &sdp.Query{
					Type:   "rdap-ip-network",
					Method: sdp.QueryMethod_SEARCH,
					Query:  ip.String(),
					Scope:  "global",
				},
				BlastPropagation: &sdp.BlastPropagation{
					// Changing the network will affect the IP
					In: true,
					// The IP won't affect the network
					Out: false,
				},
			},
		},
	}, nil
}

// List Returns an empty list as returning all possible IP addresses would be
// unproductive
func (bc *IPAdapter) List(ctx context.Context, scope string, ignoreCache bool) ([]*sdp.Item, error) {
	if scope != "global" {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: "IP queries only supported in global scope",
			Scope:       scope,
		}
	}

	return make([]*sdp.Item, 0), nil
}

// IsGlobalScopeIP Returns whether or not the IP should be considered valid
// withing the global scope according to the following logic:
//
// Non-Global:
//
// * LinkLocalMulticast
// * LinkLocalUnicast
// * InterfaceLocalMulticast
// * Loopback
//
// Global:
//
// * Private
// * Other (All non-reserved addresses)
func IsGlobalScopeIP(ip net.IP) bool {
	return !ip.IsLinkLocalMulticast() && !ip.IsLinkLocalUnicast() && !ip.IsInterfaceLocalMulticast() && !ip.IsLoopback()
}
