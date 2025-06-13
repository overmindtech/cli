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

	return ListableAdapter{
		listEndpoint: listEndpoint,
		Adapter: Adapter{
			projectID:  config.ProjectID,
			scope:      config.Scope,
			httpCli:    config.HTTPClient,
			getURLFunc: config.GetURLFunc,
			httpHeaders: http.Header{
				"Authorization": []string{"Bearer " + config.Token},
			},
			sdpAssetType:        config.SDPAssetType,
			sdpAdapterCategory:  config.SDPAdapterCategory,
			terraformMappings:   config.TerraformMappings,
			linker:              config.Linker,
			potentialLinks:      potentialLinksFromBlasts(config.SDPAssetType, gcpshared.BlastPropagations),
			uniqueAttributeKeys: config.UniqueAttributeKeys,
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
	itemsSelector := g.uniqueAttributeKeys[len(g.uniqueAttributeKeys)-1] // Use the last key as the item selector
	multiResp, err := externalCallMulti(ctx, itemsSelector, g.httpCli, g.httpHeaders, g.listEndpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to list items for %s: %w", g.listEndpoint, err)
	}

	for _, resp := range multiResp {
		item, err := externalToSDP(ctx, g.projectID, g.scope, g.uniqueAttributeKeys, resp, g.sdpAssetType, g.linker)
		if err != nil {
			return nil, err
		}

		items = append(items, item)
	}

	return items, nil
}
