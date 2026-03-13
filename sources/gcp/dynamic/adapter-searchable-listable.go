package dynamic

import (
	"context"
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/overmindtech/cli/go/discovery"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
	"github.com/overmindtech/cli/sources"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

type SearchableListableDiscoveryAdapter interface {
	discovery.SearchableAdapter
	discovery.ListableAdapter
}

// SearchableListableAdapter implements discovery.SearchableAdapter for GCP dynamic adapters.
type SearchableListableAdapter struct {
	customSearchMethodDescription string
	searchEndpointFunc            gcpshared.EndpointFunc
	searchFilterFunc              gcpshared.SearchFilterFunc
	ListableAdapter
}

// NewSearchableListableAdapter creates a new GCP dynamic adapter.
func NewSearchableListableAdapter(searchURLFunc gcpshared.EndpointFunc, listEndpointFunc gcpshared.ListEndpointFunc, config *AdapterConfig, customSearchMethodDesc string, cache sdpcache.Cache) SearchableListableDiscoveryAdapter {
	return SearchableListableAdapter{
		customSearchMethodDescription: customSearchMethodDesc,
		searchEndpointFunc:            searchURLFunc,
		searchFilterFunc:              config.SearchFilterFunc,
		ListableAdapter: ListableAdapter{
			listEndpointFunc: listEndpointFunc,
			Adapter: Adapter{
				locations:            config.Locations,
				httpCli:              config.HTTPClient,
				cache:                cache,
				getURLFunc:           config.GetURLFunc,
				sdpAssetType:         config.SDPAssetType,
				sdpAdapterCategory:   config.SDPAdapterCategory,
				terraformMappings:    config.TerraformMappings,
				linker:               config.Linker,
				potentialLinks:       potentialLinksFromLinkRules(config.SDPAssetType, gcpshared.LinkRules),
				uniqueAttributeKeys:  config.UniqueAttributeKeys,
				iamPermissions:       config.IAMPermissions,
				nameSelector:         config.NameSelector,
				listResponseSelector: config.ListResponseSelector,
			},
		},
	}
}

func (g SearchableListableAdapter) Metadata() *sdp.AdapterMetadata {
	return &sdp.AdapterMetadata{
		Type:            g.sdpAssetType.String(),
		Category:        g.sdpAdapterCategory,
		DescriptiveName: g.sdpAssetType.Readable(),
		SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
			Get:               true,
			GetDescription:    getDescription(g.sdpAssetType, g.uniqueAttributeKeys),
			Search:            true,
			SearchDescription: searchDescription(g.sdpAssetType, g.uniqueAttributeKeys, g.customSearchMethodDescription),
			List:              true,
			ListDescription:   listDescription(g.sdpAssetType),
		},
		TerraformMappings: g.terraformMappings,
		PotentialLinks:    g.potentialLinks,
	}
}

func (g SearchableListableAdapter) Search(ctx context.Context, scope, query string, ignoreCache bool) ([]*sdp.Item, error) {
	location, err := g.validateScope(scope)
	if err != nil {
		return nil, err
	}

	cacheHit, ck, cachedItems, qErr, done := g.cache.Lookup(
		ctx,
		g.Name(),
		sdp.QueryMethod_SEARCH,
		scope,
		g.Type(),
		query,
		ignoreCache,
	)
	defer done()

	if qErr != nil {
		// For better semantics, convert cached NOTFOUND into empty result
		if qErr.GetErrorType() == sdp.QueryError_NOTFOUND {
			return []*sdp.Item{}, nil
		}
		log.WithContext(ctx).WithFields(log.Fields{
			"ovm.source.type":      "gcp",
			"ovm.source.adapter":   g.Name(),
			"ovm.source.scope":     scope,
			"ovm.source.method":    sdp.QueryMethod_SEARCH.String(),
			"ovm.source.cache-key": ck,
		}).WithError(qErr).Info("returning cached query error")
		return nil, qErr
	}

	if cacheHit {
		return cachedItems, nil
	}

	if strings.HasPrefix(query, "projects/") {
		// This must be a terraform query in the format of:
		// projects/{{project}}/datasets/{{dataset}}/tables/{{name}}
		// projects/{{project}}/serviceAccounts/{{account}}/keys/{{key}}
		return terraformMappingViaSearch(ctx, g.Adapter, query, location, g.cache, ck)
	}

	searchEndpoint := g.searchEndpointFunc(query, location)
	if searchEndpoint == "" {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: fmt.Sprintf("no search endpoint found for query \"%s\". %s", query, g.Metadata().GetSupportedQueryMethods().GetSearchDescription()),
		}
	}

	items, err := aggregateSDPItems(ctx, g.Adapter, searchEndpoint, location)
	if err != nil {
		if sources.IsNotFound(err) {
			g.cache.StoreUnavailableItem(ctx, err, shared.DefaultCacheDuration, ck)
			return []*sdp.Item{}, nil
		}
		return nil, err
	}

	if g.searchFilterFunc != nil {
		filtered := make([]*sdp.Item, 0, len(items))
		for _, item := range items {
			if g.searchFilterFunc(query, item) {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}

	if len(items) == 0 {
		// Cache not-found when no items were found
		notFoundErr := &sdp.QueryError{
			ErrorType:     sdp.QueryError_NOTFOUND,
			ErrorString:   fmt.Sprintf("no %s found for search query '%s'", g.Type(), query),
			Scope:         scope,
			SourceName:    g.Name(),
			ItemType:      g.Type(),
			ResponderName: g.Name(),
		}
		g.cache.StoreUnavailableItem(ctx, notFoundErr, shared.DefaultCacheDuration, ck)
		return items, nil
	}

	for _, item := range items {
		g.cache.StoreItem(ctx, item, shared.DefaultCacheDuration, ck)
	}

	return items, nil
}

func (g SearchableListableAdapter) SearchStream(ctx context.Context, scope, query string, ignoreCache bool, stream discovery.QueryResultStream) {
	// When a post-filter is configured, fall back to the non-streaming Search
	// so we can filter before sending items to the stream.
	if g.searchFilterFunc != nil {
		items, err := g.Search(ctx, scope, query, ignoreCache)
		if err != nil {
			stream.SendError(err)
			return
		}
		for _, item := range items {
			stream.SendItem(item)
		}
		return
	}

	location, err := g.validateScope(scope)
	if err != nil {
		stream.SendError(err)
		return
	}

	cacheHit, ck, cachedItems, qErr, done := g.cache.Lookup(
		ctx,
		g.Name(),
		sdp.QueryMethod_SEARCH,
		scope,
		g.Type(),
		query,
		ignoreCache,
	)
	defer done()

	if qErr != nil {
		// For better semantics, convert cached NOTFOUND into empty result
		if qErr.GetErrorType() == sdp.QueryError_NOTFOUND {
			return
		}
		log.WithContext(ctx).WithFields(log.Fields{
			"ovm.source.type":      "gcp",
			"ovm.source.adapter":   g.Name(),
			"ovm.source.scope":     scope,
			"ovm.source.method":    sdp.QueryMethod_SEARCH.String(),
			"ovm.source.cache-key": ck,
		}).WithError(qErr).Info("returning cached query error")
		stream.SendError(qErr)
		return
	}

	if cacheHit {
		for _, item := range cachedItems {
			stream.SendItem(item)
		}

		return
	}

	if strings.HasPrefix(query, "projects/") {
		// This must be a terraform query in the format of:
		// projects/{{project}}/datasets/{{dataset}}/tables/{{name}}
		// projects/{{project}}/serviceAccounts/{{account}}/keys/{{key}}
		items, err := terraformMappingViaSearch(ctx, g.Adapter, query, location, g.cache, ck)
		if err != nil {
			stream.SendError(&sdp.QueryError{
				ErrorType:   sdp.QueryError_OTHER,
				ErrorString: fmt.Sprintf("failed to execute terraform mapping search for query \"%s\": %v", query, err),
			})
			return
		}
		if len(items) == 0 {
			// NOTFOUND: terraformMappingViaSearch returns ([], nil); send nothing (matches cached NOTFOUND behaviour)
			return
		}
		g.cache.StoreItem(ctx, items[0], shared.DefaultCacheDuration, ck)

		// There should only be one item in the result, so we can send it directly
		stream.SendItem(items[0])
		return
	}

	searchURL := g.searchEndpointFunc(query, location)
	if searchURL == "" {
		stream.SendError(&sdp.QueryError{
			ErrorType: sdp.QueryError_OTHER,
			ErrorString: fmt.Sprintf(
				"failed to construct the URL for the query \"%s\". SEARCH method description: %s",
				query,
				g.Metadata().GetSupportedQueryMethods().GetSearchDescription(),
			),
		})
		return
	}

	streamSDPItems(ctx, g.Adapter, searchURL, location, stream, g.cache, ck)
}
