package dynamic

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

type SearchableListableDiscoveryAdapter interface {
	discovery.SearchableAdapter
	discovery.ListableAdapter
}

// SearchableListableAdapter implements discovery.SearchableAdapter for GCP dynamic adapters.
type SearchableListableAdapter struct {
	searchEndpointFunc gcpshared.EndpointFunc
	ListableAdapter
}

// NewSearchableListableAdapter creates a new GCP dynamic adapter.
func NewSearchableListableAdapter(searchURLFunc gcpshared.EndpointFunc, listEndpoint string, config *AdapterConfig) SearchableListableDiscoveryAdapter {
	return SearchableListableAdapter{
		searchEndpointFunc: searchURLFunc,
		ListableAdapter: ListableAdapter{
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
			GetDescription:    getDescription(g.sdpAssetType, g.scope, g.uniqueAttributeKeys),
			Search:            true,
			SearchDescription: searchDescription(g.sdpAssetType, g.scope, g.uniqueAttributeKeys),
			List:              true,
			ListDescription:   listDescription(g.sdpAssetType, g.scope),
		},
		TerraformMappings: g.terraformMappings,
		PotentialLinks:    g.potentialLinks,
	}
}

func (g SearchableListableAdapter) Search(ctx context.Context, scope, query string, ignoreCache bool) ([]*sdp.Item, error) {
	if scope != g.scope {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: fmt.Sprintf("requested scope %v does not match any adapter scope %v", scope, g.Scopes()),
		}
	}
	searchEndpoint := g.searchEndpointFunc(query)
	if searchEndpoint == "" {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: fmt.Sprintf("no search endpoint found for query \"%s\". %s", query, g.Metadata().GetSupportedQueryMethods().GetSearchDescription()),
		}
	}

	var items []*sdp.Item
	itemsSelector := g.uniqueAttributeKeys[len(g.uniqueAttributeKeys)-1] // Use the last key as the item selector

	if strings.HasPrefix(query, "projects/") {
		// This is a single item query for terraform search method mappings.
		// See: https://linear.app/overmind/issue/ENG-580/handle-terraform-mappings-in-search-method
		resp, err := externalCallSingle(ctx, g.httpCli, g.httpHeaders, searchEndpoint)
		if err != nil {
			return nil, err
		}

		item, err := externalToSDP(ctx, g.projectID, g.scope, g.uniqueAttributeKeys, resp, g.sdpAssetType, g.linker)
		if err != nil {
			return nil, err
		}

		return append(items, item), nil
	}

	multiResp, err := externalCallMulti(ctx, itemsSelector, g.httpCli, g.httpHeaders, searchEndpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve items for %s: %w", searchEndpoint, err)
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
