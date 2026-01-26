package dynamic

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
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
			Cache:                cache,
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

	cacheHit, ck, cachedItems, qErr, done := g.GetCache().Lookup(
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
		log.WithContext(ctx).WithFields(log.Fields{
			"ovm.source.type":      "gcp",
			"ovm.source.adapter":   g.Name(),
			"ovm.source.scope":     scope,
			"ovm.source.method":    sdp.QueryMethod_LIST.String(),
			"ovm.source.cache-key": ck,
		}).WithError(qErr).Error("failed to lookup item in cache")
	}

	if cacheHit {
		return cachedItems, nil
	}

	listURL, err := g.listEndpointFunc(location)
	if err != nil {
		err := &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: fmt.Sprintf("failed to construct list endpoint: %v", err),
		}
		g.GetCache().StoreError(ctx, err, shared.DefaultCacheDuration, ck)
		return nil, err
	}

	items, err := aggregateSDPItems(ctx, g.Adapter, listURL, location)
	if err != nil {
		g.GetCache().StoreError(ctx, err, shared.DefaultCacheDuration, ck)
		return nil, err
	}

	for _, item := range items {
		g.GetCache().StoreItem(ctx, item, shared.DefaultCacheDuration, ck)
	}

	return items, nil
}

func (g ListableAdapter) ListStream(ctx context.Context, scope string, ignoreCache bool, stream discovery.QueryResultStream) {
	location, err := g.validateScope(scope)
	if err != nil {
		stream.SendError(err)
		return
	}

	cacheHit, ck, cachedItems, qErr, done := g.GetCache().Lookup(
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
		log.WithContext(ctx).WithFields(log.Fields{
			"ovm.source.type":      "gcp",
			"ovm.source.adapter":   g.Name(),
			"ovm.source.scope":     scope,
			"ovm.source.method":    sdp.QueryMethod_LIST.String(),
			"ovm.source.cache-key": ck,
		}).WithError(qErr).Error("failed to lookup item in cache")
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

	streamSDPItems(ctx, g.Adapter, listURL, location, stream, g.GetCache(), ck)
}
