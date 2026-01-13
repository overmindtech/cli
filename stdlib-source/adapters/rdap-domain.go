package adapters

import (
	"context"
	"fmt"
	"strings"

	"github.com/openrdap/rdap"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
)

type RdapDomainAdapter struct {
	ClientFac func() *rdap.Client
	Cache     sdpcache.Cache
}

// Type is the type of items that this returns
func (s *RdapDomainAdapter) Type() string {
	return "rdap-domain"
}

// Name Returns the name of the backend
func (s *RdapDomainAdapter) Name() string {
	return "rdap"
}

func (s *RdapDomainAdapter) Metadata() *sdp.AdapterMetadata {
	return rdapDomainMetadata
}

var rdapDomainMetadata = Metadata.Register(&sdp.AdapterMetadata{
	DescriptiveName: "RDAP Domain",
	Type:            "rdap-domain",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		SearchDescription: "Search for a domain record by the domain name e.g. \"www.google.com\"",
		Search:            true,
	},
	PotentialLinks: []string{"dns", "rdap-nameserver", "rdap-entity", "rdap-ip-network"},
	Category:       sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
})

// Weighting of duplicate adapters
func (s *RdapDomainAdapter) Weight() int {
	return 100
}

func (s *RdapDomainAdapter) Scopes() []string {
	return []string{
		"global",
	}
}

func (s *RdapDomainAdapter) Get(ctx context.Context, scope string, query string, ignoreCache bool) (*sdp.Item, error) {
	// While we can't actually run GET queries, we can return them if they are
	// cached
	hit, _, items, sdpErr := s.Cache.Lookup(ctx, s.Name(), sdp.QueryMethod_GET, scope, s.Type(), query, ignoreCache)

	if sdpErr != nil {
		return nil, sdpErr
	}

	if hit {
		if len(items) > 0 {
			return items[0], nil
		}
	}

	return nil, &sdp.QueryError{
		ErrorType:   sdp.QueryError_NOTFOUND,
		ErrorString: "Domains can't be queried by handle, use the SEARCH method instead",
		Scope:       scope,
		SourceName:  s.Name(),
		ItemType:    s.Type(),
	}
}

func (s *RdapDomainAdapter) List(ctx context.Context, scope string, ignoreCache bool) ([]*sdp.Item, error) {
	return nil, &sdp.QueryError{
		ErrorType:   sdp.QueryError_NOTFOUND,
		ErrorString: "Domains listed, use the SEARCH method instead",
		Scope:       scope,
		SourceName:  s.Name(),
		ItemType:    s.Type(),
	}
}

// Search for the most specific domain that contains the specified domain. The
// input should be something like "www.google.com". This will first search for
// "www.google.com", then "google.com", then "com"
func (s *RdapDomainAdapter) Search(ctx context.Context, scope string, query string, ignoreCache bool) ([]*sdp.Item, error) {
	// Strip the trailing dot if it exists
	query = strings.TrimSuffix(query, ".")

	hit, ck, items, sdpErr := s.Cache.Lookup(ctx, s.Name(), sdp.QueryMethod_SEARCH, scope, s.Type(), query, ignoreCache)

	if sdpErr != nil {
		return nil, sdpErr
	}
	if hit {
		return items, nil
	}

	// Split the query into subdomains
	sections := strings.Split(query, ".")

	// Start by querying the whole domain, then go down from there, however
	// don't query for the top-level domain as it won't return anything useful
	for i := range len(sections) - 1 {
		domainName := strings.Join(sections[i:], ".")

		request := &rdap.Request{
			Type:  rdap.DomainRequest,
			Query: domainName,
		}
		request = request.WithContext(ctx)

		response, err := s.ClientFac().Do(request)
		if err != nil {
			// If there was an error, continue to the next domain
			continue
		}

		if response.Object == nil {
			return nil, &sdp.QueryError{
				ErrorType:   sdp.QueryError_NOTFOUND,
				ErrorString: "Empty domain response",
				Scope:       scope,
				SourceName:  s.Name(),
				ItemType:    s.Type(),
			}
		}

		domain, ok := response.Object.(*rdap.Domain)

		if !ok {
			return nil, &sdp.QueryError{
				ErrorType:   sdp.QueryError_OTHER,
				ErrorString: fmt.Sprintf("Unexpected response type %T", response.Object),
				Scope:       scope,
				SourceName:  s.Name(),
				ItemType:    s.Type(),
			}
		}

		attributes, err := sdp.ToAttributesCustom(map[string]interface{}{
			"conformance":     domain.Conformance,
			"events":          domain.Events,
			"handle":          domain.Handle,
			"ldhName":         domain.LDHName,
			"links":           domain.Links,
			"notices":         domain.Notices,
			"objectClassName": domain.ObjectClassName,
			"port43":          domain.Port43,
			"publicIDs":       domain.PublicIDs,
			"remarks":         domain.Remarks,
			"secureDNS":       domain.SecureDNS,
			"status":          domain.Status,
			"unicodeName":     domain.UnicodeName,
			"variants":        domain.Variants,
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

		// Link to nameservers
		for _, nameServer := range domain.Nameservers {
			// Look through the HTTP responses until we find one
			var parsed *RDAPUrl
			for _, httpResponse := range response.HTTP {
				if httpResponse.URL != "" {
					parsed, err = parseRdapUrl(httpResponse.URL)

					if err == nil {
						break
					}
				}
			}

			// Reconstruct the required query URL
			if parsed != nil {
				newURL := parsed.ServerRoot.JoinPath("/nameserver/" + nameServer.LDHName)

				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "rdap-nameserver",
						Method: sdp.QueryMethod_SEARCH,
						Query:  newURL.String(),
						Scope:  "global",
					},
					BlastPropagation: &sdp.BlastPropagation{
						// A change in a name server could affect the domains
						In: true,
						// Domains won't affect the name server
						Out: false,
					},
				})
			}

		}

		// Link to entities

		item.LinkedItemQueries = append(item.LinkedItemQueries, extractEntityLinks(domain.Entities)...)

		// Link to IP Network
		if network := domain.Network; network != nil {
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "rdap-ip-network",
					Method: sdp.QueryMethod_SEARCH,
					Query:  network.StartAddress,
					Scope:  "global",
				},
				BlastPropagation: &sdp.BlastPropagation{
					// do not link through rdap definitions to avoid huge blast radius
					In:  false,
					Out: false,
				},
			})
		}

		if err != nil {
			return nil, err
		}

		s.Cache.StoreItem(ctx, item, RdapCacheDuration, ck)

		return []*sdp.Item{item}, nil
	}

	err := &sdp.QueryError{
		ErrorType:   sdp.QueryError_NOTFOUND,
		ErrorString: fmt.Sprintf("No domain found for %s", query),
		Scope:       scope,
		SourceName:  s.Name(),
		ItemType:    s.Type(),
	}

	s.Cache.StoreError(ctx, err, RdapCacheDuration, ck)

	return nil, err
}
