package adapters

import (
	"context"
	"fmt"
	"net"

	"github.com/openrdap/rdap"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
)

type RdapIPNetworkAdapter struct {
	ClientFac func() *rdap.Client
	Cache     *sdpcache.Cache
	IPCache   *IPCache[*rdap.IPNetwork]
}

// Type is the type of items that this returns
func (s *RdapIPNetworkAdapter) Type() string {
	return "rdap-ip-network"
}

// Name Returns the name of the adapter
func (s *RdapIPNetworkAdapter) Name() string {
	return "rdap"
}

// Weighting of duplicate adapters
func (s *RdapIPNetworkAdapter) Weight() int {
	return 100
}

func (s *RdapIPNetworkAdapter) Scopes() []string {
	return []string{
		"global",
	}
}

func (s *RdapIPNetworkAdapter) Metadata() *sdp.AdapterMetadata {
	return rdapIPNetworkMetadata
}

var rdapIPNetworkMetadata = Metadata.Register(&sdp.AdapterMetadata{
	DescriptiveName: "RDAP IP Network",
	Type:            "rdap-ip-network",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Search:            true,
		SearchDescription: "Search for the most specific network that contains the specified IP or CIDR",
	},
	PotentialLinks: []string{"rdap-entity"},
	Category:       sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
})

func (s *RdapIPNetworkAdapter) Get(ctx context.Context, scope string, query string, ignoreCache bool) (*sdp.Item, error) {
	hit, _, items, sdpErr := s.Cache.Lookup(ctx, s.Name(), sdp.QueryMethod_GET, scope, s.Type(), query, ignoreCache)

	if sdpErr != nil {
		return nil, sdpErr
	}

	if hit {
		if len(items) > 0 {
			return items[0], nil
		}
	}
	// This adapter doesn't technically support the GET method (since you can't
	// use the handle to query for an IP network)
	return nil, &sdp.QueryError{
		ErrorType:   sdp.QueryError_NOTFOUND,
		Scope:       scope,
		ErrorString: "IP networks can't be queried by handle, use the SEARCH method instead",
	}
}

func (s *RdapIPNetworkAdapter) List(ctx context.Context, scope string, ignoreCache bool) ([]*sdp.Item, error) {
	return nil, &sdp.QueryError{
		ErrorType:   sdp.QueryError_NOTFOUND,
		Scope:       scope,
		ErrorString: "IP networks cannot be listed, use the SEARCH method instead",
	}
}

// Search for the most specific network that contains the specified IP or CIDR
func (s *RdapIPNetworkAdapter) Search(ctx context.Context, scope string, query string, ignoreCache bool) ([]*sdp.Item, error) {
	hit, ck, items, sdpErr := s.Cache.Lookup(ctx, s.Name(), sdp.QueryMethod_SEARCH, scope, s.Type(), query, ignoreCache)

	if sdpErr != nil {
		return nil, sdpErr
	}
	if hit {
		return items, nil
	}

	// Second layer of caching means that we cn look up an IP, and if there is
	// anything in the cache that covers a range that IP is in, it will hit
	// the cache
	var ipNetwork *rdap.IPNetwork

	// See which type of argument we have and parse it
	if ip := net.ParseIP(query); ip != nil {
		// Check if the IP is in the cache
		ipNetwork, hit = s.IPCache.SearchIP(ip)
	} else if _, network, err := net.ParseCIDR(query); err == nil {
		// Check if the CIDR is in the cache
		ipNetwork, hit = s.IPCache.SearchCIDR(network)
	} else {
		return nil, fmt.Errorf("Invalid IP or CIDR: %v", query)
	}

	if !hit {
		// If we didn't hit the cache, then actually execute the query
		request := &rdap.Request{
			Type:  rdap.IPRequest,
			Query: query,
		}
		request = request.WithContext(ctx)

		response, err := s.ClientFac().Do(request)

		if err != nil {
			err = wrapRdapError(err)

			s.Cache.StoreError(err, RdapCacheDuration, ck)

			return nil, err
		}

		if response.Object == nil {
			return nil, &sdp.QueryError{
				ErrorType:   sdp.QueryError_NOTFOUND,
				Scope:       scope,
				ErrorString: fmt.Sprintf("No IP Network found for %s", query),
				SourceName:  s.Name(),
			}
		}

		var ok bool

		ipNetwork, ok = response.Object.(*rdap.IPNetwork)

		if !ok {
			return nil, fmt.Errorf("Expected IPNetwork, got %T", response.Object)
		}

		// Calculate the CIDR for this network
		network, err := calculateNetwork(ipNetwork.StartAddress, ipNetwork.EndAddress)

		if err != nil {
			return nil, err
		}

		// Cache this network
		s.IPCache.Store(network, ipNetwork, RdapCacheDuration)
	}

	attributes, err := sdp.ToAttributesCustom(map[string]interface{}{
		"conformance":     ipNetwork.Conformance,
		"country":         ipNetwork.Country,
		"endAddress":      ipNetwork.EndAddress,
		"events":          ipNetwork.Events,
		"handle":          ipNetwork.Handle,
		"ipVersion":       ipNetwork.IPVersion,
		"links":           ipNetwork.Links,
		"name":            ipNetwork.Name,
		"notices":         ipNetwork.Notices,
		"objectClassName": ipNetwork.ObjectClassName,
		"parentHandle":    ipNetwork.ParentHandle,
		"port43":          ipNetwork.Port43,
		"remarks":         ipNetwork.Remarks,
		"startAddress":    ipNetwork.StartAddress,
		"status":          ipNetwork.Status,
		"type":            ipNetwork.Type,
	}, true, RDAPTransforms)

	if err != nil {
		return nil, err
	}

	item := &sdp.Item{
		Type:            s.Type(),
		UniqueAttribute: "handle",
		Attributes:      attributes,
		Scope:           scope,
	}

	// Loop over the entities and create linkedin item queries
	item.LinkedItemQueries = extractEntityLinks(ipNetwork.Entities)

	s.Cache.StoreItem(item, RdapCacheDuration, ck)

	return []*sdp.Item{item}, nil
}

// Calculates the network (like a CIDR) from a given start and end IP
func calculateNetwork(startIP, endIP string) (*net.IPNet, error) {
	// Parse start and end IP addresses
	start := net.ParseIP(startIP)
	if start == nil {
		return nil, fmt.Errorf("Invalid start IP address: %s", startIP)
	}

	end := net.ParseIP(endIP)
	if end == nil {
		return nil, fmt.Errorf("Invalid end IP address: %s", endIP)
	}

	// Calculate the CIDR prefix length
	var prefixLen int
	for i := range start {
		startByte := start[i]
		endByte := end[i]

		if startByte != endByte {
			// Find the differing bit position
			diffBit := startByte ^ endByte

			// Count the number of consecutive zero bits in the differing byte
			for j := 7; j >= 0; j-- {
				if (diffBit & (1 << uint(j))) != 0 {
					break
				}
				prefixLen++
			}
			break
		}

		prefixLen += 8
	}

	mask := net.CIDRMask(int(prefixLen), 128)

	// Calculate the network address
	network := net.IPNet{
		IP:   start,
		Mask: mask,
	}

	return &network, nil
}
