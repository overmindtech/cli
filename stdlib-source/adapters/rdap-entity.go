package adapters

import (
	"context"
	"fmt"
	"net/url"

	"github.com/openrdap/rdap"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
)

type RdapEntityAdapter struct {
	ClientFac func() *rdap.Client
	Cache     *sdpcache.Cache
}

// Type is the type of items that this returns
func (s *RdapEntityAdapter) Type() string {
	return "rdap-entity"
}

// Name Returns the name of the backend
func (s *RdapEntityAdapter) Name() string {
	return "rdap"
}

// Weighting of duplicate adapters
func (s *RdapEntityAdapter) Weight() int {
	return 100
}

func (s *RdapEntityAdapter) Metadata() *sdp.AdapterMetadata {
	return rdapEntityMetadata
}

var rdapEntityMetadata = Metadata.Register(&sdp.AdapterMetadata{
	DescriptiveName: "RDAP Entity",
	Type:            "rdap-entity",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		Search:            true,
		GetDescription:    "Get an entity by its handle. This method is discouraged as it's not reliable since entity bootstrapping isn't comprehensive",
		SearchDescription: "Search for an entity by its URL e.g. https://rdap.apnic.net/entity/AIC3-AP",
	},
	PotentialLinks: []string{"rdap-asn"},
	Category:       sdp.AdapterCategory_ADAPTER_CATEGORY_SECURITY,
})

func (s *RdapEntityAdapter) Scopes() []string {
	return []string{
		"global",
	}
}

// Gets an entity by its handle, note that this might not work as entity
// bootstrapping in RDAP isn't comprehensive and might not be able to find the
// correct registry to search
func (s *RdapEntityAdapter) Get(ctx context.Context, scope string, query string, ignoreCache bool) (*sdp.Item, error) {
	hit, ck, items, sdpErr := s.Cache.Lookup(ctx, s.Name(), sdp.QueryMethod_GET, scope, s.Type(), query, ignoreCache)

	if sdpErr != nil {
		return nil, sdpErr
	}
	if hit {
		if len(items) > 0 {
			return items[0], nil
		}
	}

	return s.runEntityRequest(ctx, query, nil, scope, ck)
}

func (s *RdapEntityAdapter) List(ctx context.Context, scope string, ignoreCache bool) ([]*sdp.Item, error) {
	return nil, nil
}

// Search for an entity by its URL e.g. https://rdap.apnic.net/entity/AIC3-AP.
// This is required because despite the work on bootstrapping in RFC 8521 it's
// still not reliable enough to always resolve entities. However when we get
// linked to an entity it should always have a link to itself, so we should be
// able to do a lookup using that which will also tell us which server to use
// for the lookup
func (s *RdapEntityAdapter) Search(ctx context.Context, scope string, query string, ignoreCache bool) ([]*sdp.Item, error) {
	hit, ck, items, sdpErr := s.Cache.Lookup(ctx, s.Name(), sdp.QueryMethod_SEARCH, scope, s.Type(), query, ignoreCache)

	if sdpErr != nil {
		return nil, sdpErr
	}
	if hit {
		return items, nil
	}

	// Parse the URL
	parsed, err := parseRdapUrl(query)

	if err != nil {
		return nil, err
	}

	if parsed.Type != "entity" {
		return nil, fmt.Errorf("Expected URL to lookup entity, got %s", parsed.Type)
	}

	// Run the entity request
	item, err := s.runEntityRequest(ctx, parsed.Query, parsed.ServerRoot, scope, ck)

	if err != nil {
		return nil, err
	}

	return []*sdp.Item{item}, nil
}

// Runs the entity request and converts into the SDP version of an entity
func (s *RdapEntityAdapter) runEntityRequest(ctx context.Context, query string, server *url.URL, scope string, cacheKey sdpcache.CacheKey) (*sdp.Item, error) {
	request := &rdap.Request{
		Type:   rdap.EntityRequest,
		Query:  query,
		Server: server,
	}
	request = request.WithContext(ctx)

	response, err := s.ClientFac().Do(request)

	if err != nil {
		err = wrapRdapError(err)

		s.Cache.StoreError(err, RdapCacheDuration, cacheKey)

		return nil, err
	}

	if response.Object == nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOTFOUND,
			Scope:       scope,
			ErrorString: fmt.Sprintf("No entity found for %s", query),
			ItemType:    s.Type(),
			SourceName:  s.Name(),
		}
	}

	entity, ok := response.Object.(*rdap.Entity)

	if !ok {
		return nil, fmt.Errorf("Expected Entity, got %T", response.Object)
	}

	attributes, err := sdp.ToAttributesCustom(map[string]interface{}{
		"asEventActor":    entity.AsEventActor,
		"conformance":     entity.Conformance,
		"events":          entity.Events,
		"handle":          entity.Handle,
		"links":           entity.Links,
		"notices":         entity.Notices,
		"objectClassName": entity.ObjectClassName,
		"port43":          entity.Port43,
		"publicIDs":       entity.PublicIDs,
		"remarks":         entity.Remarks,
		"roles":           entity.Roles,
		"status":          entity.Status,
		"vCard":           entity.VCard,
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

	// Link to related entities
	item.LinkedItemQueries = extractEntityLinks(entity.Entities)

	// Don't link to related networks as there are entities with hundreds of
	// networks and there isn't a reasonable use case that would involve
	// traversing these

	// Link to related ASNs
	for _, autnum := range entity.Autnums {

		item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   "rdap-asn",
				Method: sdp.QueryMethod_GET,
				Query:  autnum.Handle,
				Scope:  scope,
			},
			BlastPropagation: &sdp.BlastPropagation{
				// The ASN won't affect the entity
				In: false,
				// The entity could maybe affect the ASN? Change this if it
				// causes issues
				Out: true,
			},
		})
	}

	s.Cache.StoreItem(item, RdapCacheDuration, cacheKey)

	return item, nil
}
