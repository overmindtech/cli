package dynamic

import (
	"context"
	"fmt"
	"net/http"

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
func NewListableAdapter(listEndpoint string, config *AdapterConfig) discovery.ListableAdapter {
	var potentialLinks []string
	if blasts, ok := gcpshared.BlastPropagations[config.SDPAssetType]; ok {
		for item := range blasts {
			potentialLinks = append(potentialLinks, item.String())
		}
	}

	return ListableAdapter{
		listEndpoint: listEndpoint,
		Adapter: Adapter{
			projectID:  config.ProjectID,
			scope:      config.Scope,
			httpCli:    config.HTTPClient,
			getBaseURL: config.GetBaseURL,
			httpHeaders: http.Header{
				"Authorization": []string{"Bearer " + config.Token},
			},
			sdpAssetType:       config.SDPAssetType,
			sdpAdapterCategory: config.SDPAdapterCategory,
			terraformMappings:  config.TerraformMappings,
			linker:             config.Linker,
			potentialLinks:     potentialLinks,
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
			GetDescription:  fmt.Sprintf("Get a %s by its name i.e: zones/<zone>/instances/<instance-name>", g.sdpAssetType),
			List:            true,
			ListDescription: fmt.Sprintf("List all %s within its scopes: %v", g.sdpAssetType, g.Scopes()),
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

	var items []*sdp.Item
	multiResp, err := externalCallMulti(ctx, g.httpCli, g.httpHeaders, g.listEndpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to list items for %s: %w", g.listEndpoint, err)
	}

	for _, resp := range multiResp {
		item, err := externalToSDP(ctx, g.projectID, resp, g.sdpAssetType, g.linker)
		if err != nil {
			return nil, err
		}

		items = append(items, item)
	}

	return items, nil
}
