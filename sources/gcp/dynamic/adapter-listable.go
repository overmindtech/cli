package dynamic

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

// ListableAdapter implements discovery.ListableAdapter for GCP dynamic adapters.
type ListableAdapter struct {
	listEndpoint string
	Adapter
}

// NewListableAdapter creates a new GCP dynamic adapter.
func NewListableAdapter(listEndpoint string, config *AdapterConfig) (discovery.ListableAdapter, error) {
	a := Adapter{
		projectID:           config.ProjectID,
		scope:               config.Scope,
		httpCli:             config.HTTPClient,
		cache:               sdpcache.NewCache(),
		getURLFunc:          config.GetURLFunc,
		sdpAssetType:        config.SDPAssetType,
		sdpAdapterCategory:  config.SDPAdapterCategory,
		terraformMappings:   config.TerraformMappings,
		linker:              config.Linker,
		potentialLinks:      potentialLinksFromBlasts(config.SDPAssetType, gcpshared.BlastPropagations),
		uniqueAttributeKeys: config.UniqueAttributeKeys,
	}

	if a.httpCli == nil {
		gcpHTTPCliWithOtel, err := gcpshared.GCPHTTPClientWithOtel()
		if err != nil {
			return nil, err
		}

		a.httpCli = gcpHTTPCliWithOtel
	}

	return ListableAdapter{
		listEndpoint: listEndpoint,
		Adapter:      a,
	}, nil
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
	if scope != g.scope {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: fmt.Sprintf("requested scope %v does not match any adapter scope %v", scope, g.Scopes()),
		}
	}

	cacheHit, ck, cachedItems, qErr := g.cache.Lookup(
		ctx,
		g.Name(),
		sdp.QueryMethod_LIST,
		scope,
		g.Type(),
		"",
		ignoreCache,
	)
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

	items, err := aggregateSDPItems(ctx, g.Adapter, g.listEndpoint)
	if err != nil {
		return nil, err
	}

	for _, item := range items {
		g.cache.StoreItem(item, DefaultCacheDuration, ck)
	}

	return items, nil
}

func (g ListableAdapter) ListStream(ctx context.Context, scope string, ignoreCache bool, stream discovery.QueryResultStream) {
	if scope != g.scope {
		stream.SendError(&sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: fmt.Sprintf("requested scope %v does not match any adapter scope %v", scope, g.Scopes()),
		})
		return
	}

	cacheHit, ck, cachedItems, qErr := g.cache.Lookup(
		ctx,
		g.Name(),
		sdp.QueryMethod_LIST,
		scope,
		g.Type(),
		"",
		ignoreCache,
	)
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

	streamSDPItems(ctx, g.Adapter, g.listEndpoint, stream, g.cache, ck)
}
