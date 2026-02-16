package dynamic

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"

	"github.com/overmindtech/workspace/discovery"
	"github.com/overmindtech/workspace/sdp-go"
	"github.com/overmindtech/workspace/sdpcache"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/shared"
)

// ListableAdapter implements discovery.ListableAdapter for GCP dynamic adapters.
type ListableAdapter struct {
	listEndpointFunc gcpshared.ListEndpointFunc
	Adapter
}

// NewListableAdapter creates a new GCP dynamic adapter.
func NewListableAdapter(listEndpointFunc gcpshared.ListEndpointFunc, config *AdapterConfig, cache sdpcache.Cache) discovery.ListableAdapter {
	return ListableAdapter{
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
	}
}

func (g ListableAdapter) Metadata() *sdp.AdapterMetadata {
	return &sdp.AdapterMetadata{
		Type:            g.sdpAssetType.String(),
		Category:        g.sdpAdapterCategory,
		DescriptiveName: g.sdpAssetType.Readable(),
		SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
			Get:             true,
			GetDescription:  getDescription(g.sdpAssetType, g.uniqueAttributeKeys),
			List:            true,
			ListDescription: listDescription(g.sdpAssetType),
		},
		TerraformMappings: g.terraformMappings,
		PotentialLinks:    g.potentialLinks,
	}
}

func (g ListableAdapter) List(ctx context.Context, scope string, ignoreCache bool) ([]*sdp.Item, error) {
	location, err := g.validateScope(scope)
	if err != nil {
		return nil, err
	}

	cacheHit, ck, cachedItems, qErr, done := g.cache.Lookup(
		ctx,
		g.Name(),
		sdp.QueryMethod_LIST,
		scope,
		g.Type(),
		"",
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
			"ovm.source.method":    sdp.QueryMethod_LIST.String(),
			"ovm.source.cache-key": ck,
		}).WithError(qErr).Info("returning cached query error")
		return nil, qErr
	}

	if cacheHit {
		return cachedItems, nil
	}

	listURL, err := g.listEndpointFunc(location)
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: fmt.Sprintf("failed to construct list endpoint: %v", err),
		}
	}

	items, err := aggregateSDPItems(ctx, g.Adapter, listURL, location)
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
			ErrorString:   fmt.Sprintf("no %s found in scope %s", g.Type(), scope),
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

func (g ListableAdapter) ListStream(ctx context.Context, scope string, ignoreCache bool, stream discovery.QueryResultStream) {
	location, err := g.validateScope(scope)
	if err != nil {
		stream.SendError(err)
		return
	}

	cacheHit, ck, cachedItems, qErr, done := g.cache.Lookup(
		ctx,
		g.Name(),
		sdp.QueryMethod_LIST,
		scope,
		g.Type(),
		"",
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
			"ovm.source.method":    sdp.QueryMethod_LIST.String(),
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

	listURL, err := g.listEndpointFunc(location)
	if err != nil {
		stream.SendError(&sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: fmt.Sprintf("failed to construct list endpoint: %v", err),
		})
		return
	}

	streamSDPItems(ctx, g.Adapter, listURL, location, stream, g.cache, ck)
}
