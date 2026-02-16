package dynamic

import (
	"context"
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/overmindtech/workspace/discovery"
	"github.com/overmindtech/workspace/sdp-go"
	"github.com/overmindtech/workspace/sdpcache"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/shared"
)

// SearchableAdapter implements discovery.SearchableAdapter for GCP dynamic adapters.
type SearchableAdapter struct {
	customSearchMethodDesc string
	searchEndpointFunc     gcpshared.EndpointFunc
	Adapter
}

// NewSearchableAdapter creates a new GCP dynamic adapter.
func NewSearchableAdapter(searchEndpointFunc gcpshared.EndpointFunc, config *AdapterConfig, customSearchMethodDesc string, cache sdpcache.Cache) discovery.SearchableAdapter {
	return SearchableAdapter{
		customSearchMethodDesc: customSearchMethodDesc,
		searchEndpointFunc:     searchEndpointFunc,

		Adapter: Adapter{
			locations:            config.Locations,
			httpCli:              config.HTTPClient,
			cache:                cache,
			getURLFunc:           config.GetURLFunc,
			sdpAssetType:         config.SDPAssetType,
			sdpAdapterCategory:   config.SDPAdapterCategory,
			terraformMappings:    config.TerraformMappings,
			linker:               config.Linker,
			potentialLinks:       potentialLinksFromBlasts(config.SDPAssetType, gcpshared.BlastPropagations),
			uniqueAttributeKeys:  config.UniqueAttributeKeys,
			iamPermissions:       config.IAMPermissions,
			nameSelector:         config.NameSelector,
			listResponseSelector: config.ListResponseSelector,
		},
	}
}

func (g SearchableAdapter) Metadata() *sdp.AdapterMetadata {
	return &sdp.AdapterMetadata{
		Type:            g.sdpAssetType.String(),
		Category:        g.sdpAdapterCategory,
		DescriptiveName: g.sdpAssetType.Readable(),
		SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
			Get:               true,
			GetDescription:    getDescription(g.sdpAssetType, g.uniqueAttributeKeys),
			Search:            true,
			SearchDescription: searchDescription(g.sdpAssetType, g.uniqueAttributeKeys, g.customSearchMethodDesc),
		},
		TerraformMappings: g.terraformMappings,
		PotentialLinks:    g.potentialLinks,
	}
}

func (g SearchableAdapter) Search(ctx context.Context, scope, query string, ignoreCache bool) ([]*sdp.Item, error) {
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

	// This is a regular SEARCH call
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
			g.cache.StoreError(ctx, err, shared.DefaultCacheDuration, ck)
			return []*sdp.Item{}, nil
		}
		return nil, err
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
		g.cache.StoreError(ctx, notFoundErr, shared.DefaultCacheDuration, ck)
		return items, nil
	}

	for _, item := range items {
		g.cache.StoreItem(ctx, item, shared.DefaultCacheDuration, ck)
	}

	return items, nil
}

func (g SearchableAdapter) SearchStream(ctx context.Context, scope, query string, ignoreCache bool, stream discovery.QueryResultStream) {
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
