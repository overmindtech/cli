package dynamic

import (
	"context"
	"fmt"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
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
			GetDescription:  getDescription(g.sdpAssetType, g.scope, g.uniqueAttributeKeys),
			List:            true,
			ListDescription: listDescription(g.sdpAssetType, g.scope),
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

	return aggregateSDPItems(ctx, g.Adapter, g.listEndpoint)
}

func (g ListableAdapter) ListStream(ctx context.Context, scope string, ignoreCache bool, stream discovery.QueryResultStream) {
	if scope != g.scope {
		stream.SendError(&sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: fmt.Sprintf("requested scope %v does not match any adapter scope %v", scope, g.Scopes()),
		})
		return
	}

	streamSDPItems(ctx, g.Adapter, g.listEndpoint, stream)
}
