package dynamic

import (
	"context"
	"fmt"
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
	customSearchMethodDescription string
	searchEndpointFunc            gcpshared.EndpointFunc
	ListableAdapter
}

// NewSearchableListableAdapter creates a new GCP dynamic adapter.
func NewSearchableListableAdapter(searchURLFunc gcpshared.EndpointFunc, listEndpoint string, config *AdapterConfig, customSearchMethodDesc string) (SearchableListableDiscoveryAdapter, error) {
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

	return SearchableListableAdapter{
		customSearchMethodDescription: customSearchMethodDesc,
		searchEndpointFunc:            searchURLFunc,
		ListableAdapter: ListableAdapter{
			listEndpoint: listEndpoint,
			Adapter:      a,
		},
	}, nil
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
		return terraformMappingViaSearch(ctx, g.Adapter, query)
	}

	searchEndpoint := g.searchEndpointFunc(query)
	if searchEndpoint == "" {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: fmt.Sprintf("no search endpoint found for query \"%s\". %s", query, g.Metadata().GetSupportedQueryMethods().GetSearchDescription()),
		}
	}

	return aggregateSDPItems(ctx, g.Adapter, searchEndpoint)
}

func (g SearchableListableAdapter) SearchStream(ctx context.Context, scope, query string, ignoreCache bool, stream discovery.QueryResultStream) {
	if scope != g.scope {
		stream.SendError(&sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: fmt.Sprintf("requested scope %v does not match any adapter scope %v", scope, g.Scopes()),
		})
		return
	}

	if strings.HasPrefix(query, "projects/") {
		// This must be a terraform query in the format of:
		// projects/{{project}}/datasets/{{dataset}}/tables/{{name}}
		// projects/{{project}}/serviceAccounts/{{account}}/keys/{{key}}
		items, err := terraformMappingViaSearch(ctx, g.Adapter, query)
		if err != nil {
			stream.SendError(&sdp.QueryError{
				ErrorType:   sdp.QueryError_OTHER,
				ErrorString: fmt.Sprintf("failed to execute terraform mapping search for query \"%s\": %v", query, err),
			})
			return
		}

		// There should only be one item in the result, so we can send it directly
		stream.SendItem(items[0])
		return
	}

	searchURL := g.searchEndpointFunc(query)
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

	streamSDPItems(ctx, g.Adapter, searchURL, stream)
}
