package dynamic

import (
	"context"
	"fmt"
	"strings"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

// SearchableAdapter implements discovery.SearchableAdapter for GCP dynamic adapters.
type SearchableAdapter struct {
	customSearchMethodDesc string
	searchURLFunc          gcpshared.EndpointFunc
	Adapter
}

// NewSearchableAdapter creates a new GCP dynamic adapter.
func NewSearchableAdapter(searchURLFunc gcpshared.EndpointFunc, config *AdapterConfig, customSearchMethodDesc string) (discovery.SearchableAdapter, error) {
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

	return SearchableAdapter{
		customSearchMethodDesc: customSearchMethodDesc,
		searchURLFunc:          searchURLFunc,
		Adapter:                a,
	}, nil
}

func (g SearchableAdapter) Metadata() *sdp.AdapterMetadata {
	return &sdp.AdapterMetadata{
		Type:            g.sdpAssetType.String(),
		Category:        g.sdpAdapterCategory,
		DescriptiveName: g.sdpAssetType.Readable(),
		SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
			Get:               true,
			GetDescription:    getDescription(g.sdpAssetType, g.scope, g.uniqueAttributeKeys),
			Search:            true,
			SearchDescription: searchDescription(g.sdpAssetType, g.scope, g.uniqueAttributeKeys, g.customSearchMethodDesc),
		},
		TerraformMappings: g.terraformMappings,
		PotentialLinks:    g.potentialLinks,
	}
}

func (g SearchableAdapter) Search(ctx context.Context, scope, query string, ignoreCache bool) ([]*sdp.Item, error) {
	if scope != g.scope {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: fmt.Sprintf("requested scope %v does not match any adapter scope %v", scope, g.Scopes()),
		}
	}

	if strings.HasPrefix(query, "projects/") {
		// This must be a terraform query in the format of:
		// projects/{{project}}/datasets/{{dataset}}/tables/{{name}}
		// projects/{{project}}/serviceAccounts/{{account}}/keys/{{key}}
		//
		// Extract the relevant parts from the query
		// We need to extract the path parameters based on the number of unique attribute keys
		// From projects/{{project}}/serviceAccounts/{{account}}/keys/{{key}}
		// we get: ["account", "key"]
		// if the unique attribute keys are ["serviceAccounts", "keys"]
		queryParts := gcpshared.ExtractPathParamsWithCount(query, len(g.uniqueAttributeKeys))
		if len(queryParts) != len(g.uniqueAttributeKeys) {
			return nil, &sdp.QueryError{
				ErrorType: sdp.QueryError_OTHER,
				ErrorString: fmt.Sprintf(
					"failed to handle terraform mapping from query %s for %s",
					query,
					g.sdpAssetType,
				),
			}
		}

		// Reconstruct the query from the parts with default separator
		// For example, if the unique attribute keys are ["serviceAccounts", "keys"]
		// and the query parts are ["account", "key"], we get "account|key"
		query = strings.Join(queryParts, shared.QuerySeparator)

		// We use the GET endpoint for this query. Because the terraform mappings are for single items,
		url := g.getURLFunc(query)
		if url == "" {
			return nil, &sdp.QueryError{
				ErrorType: sdp.QueryError_OTHER,
				ErrorString: fmt.Sprintf(
					"failed to construct the URL for the query \"%s\". SEARCH method description: %s",
					query,
					g.Metadata().GetSupportedQueryMethods().GetSearchDescription(),
				),
			}
		}

		resp, err := externalCallSingle(ctx, g.httpCli, url)
		if err != nil {
			return nil, err
		}

		item, err := externalToSDP(ctx, g.projectID, g.scope, g.uniqueAttributeKeys, resp, g.sdpAssetType, g.linker)
		if err != nil {
			return nil, err
		}

		return []*sdp.Item{item}, nil
	}

	// This is a regular SEARCH call
	url := g.searchURLFunc(query)
	if url == "" {
		return nil, &sdp.QueryError{
			ErrorType: sdp.QueryError_OTHER,
			ErrorString: fmt.Sprintf(
				"failed to construct the URL for the query \"%s\". SEARCH method description: %s",
				query,
				g.Metadata().GetSupportedQueryMethods().GetSearchDescription(),
			),
		}
	}

	var items []*sdp.Item
	itemsSelector := g.uniqueAttributeKeys[len(g.uniqueAttributeKeys)-1] // Use the last key as the item selector

	multiResp, err := externalCallMulti(ctx, itemsSelector, g.httpCli, url)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve items for %s: %w", url, err)
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
