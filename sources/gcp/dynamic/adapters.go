package dynamic

import (
	"fmt"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/overmindtech/cli/discovery"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

var adaptersByScope = map[gcpshared.Scope]map[shared.ItemType]gcpshared.AdapterMeta{}

func init() {
	adaptersByScope = make(map[gcpshared.Scope]map[shared.ItemType]gcpshared.AdapterMeta)
	for sdpItemType, meta := range gcpshared.SDPAssetTypeToAdapterMeta {
		if _, ok := adaptersByScope[meta.Scope]; !ok {
			adaptersByScope[meta.Scope] = make(map[shared.ItemType]gcpshared.AdapterMeta)
		}
		adaptersByScope[meta.Scope][sdpItemType] = meta
	}

}

// Adapters returns a list of discovery.Adapters for the given project ID, token, regions, and zones.
func Adapters(projectID string, token string, regions []string, zones []string, linker *gcpshared.Linker, manualAdapters map[string]bool) ([]discovery.Adapter, error) {
	var adapters []discovery.Adapter

	// Project level adapters
	for sdpItemType, meta := range adaptersByScope[gcpshared.ScopeProject] {
		if _, ok := manualAdapters[sdpItemType.String()]; ok {
			// Skip, because we have a manual adapter for this item type
			continue
		}

		getEndpointBaseURL, err := meta.GetEndpointBaseURLFunc(projectID)
		if err != nil {
			return nil, err
		}

		cfg := &AdapterConfig{
			ProjectID:           projectID,
			Scope:               projectID,
			Token:               token,
			GetURLFunc:          getEndpointBaseURL,
			SDPAssetType:        sdpItemType,
			SDPAdapterCategory:  meta.SDPAdapterCategory,
			TerraformMappings:   SDPAssetTypeToTerraformMappings[sdpItemType].Mappings,
			Linker:              linker,
			HTTPClient:          otelhttp.DefaultClient,
			UniqueAttributeKeys: meta.UniqueAttributeKeys,
		}

		if meta.ListEndpointFunc != nil {
			listEndpoint, err := meta.ListEndpointFunc(projectID)
			if err != nil {
				return nil, err
			}

			adapters = append(adapters, NewListableAdapter(listEndpoint, cfg))

			continue
		}

		adapters = append(adapters, NewAdapter(cfg))
	}

	// Regional adapters
	for _, region := range regions {
		for sdpItemType, meta := range adaptersByScope[gcpshared.ScopeRegional] {
			if _, ok := manualAdapters[sdpItemType.String()]; ok {
				// Skip, because we have a manual adapter for this item type
				continue
			}

			getEndpointBaseURL, err := meta.GetEndpointBaseURLFunc(projectID, region)
			if err != nil {
				return nil, err
			}

			scope := fmt.Sprintf("%s.%s", projectID, region)

			cfg := &AdapterConfig{
				ProjectID:           projectID,
				Scope:               scope,
				Token:               token,
				GetURLFunc:          getEndpointBaseURL,
				SDPAssetType:        sdpItemType,
				SDPAdapterCategory:  meta.SDPAdapterCategory,
				TerraformMappings:   SDPAssetTypeToTerraformMappings[sdpItemType].Mappings,
				Linker:              linker,
				HTTPClient:          otelhttp.DefaultClient,
				UniqueAttributeKeys: meta.UniqueAttributeKeys,
			}

			if meta.ListEndpointFunc != nil {
				listEndpoint, err := meta.ListEndpointFunc(projectID, region)
				if err != nil {
					return nil, err
				}
				adapters = append(adapters, NewListableAdapter(listEndpoint, cfg))

				continue
			}

			adapters = append(adapters, NewAdapter(cfg))
		}
	}

	// Zonal adapters
	for _, zone := range zones {
		for sdpItemType, meta := range adaptersByScope[gcpshared.ScopeZonal] {
			if _, ok := manualAdapters[sdpItemType.String()]; ok {
				// Skip, because we have a manual adapter for this item type
				continue
			}

			getEndpointBaseURL, err := meta.GetEndpointBaseURLFunc(projectID, zone)
			if err != nil {
				return nil, err
			}

			scope := fmt.Sprintf("%s.%s", projectID, zone)

			cfg := &AdapterConfig{
				ProjectID:           projectID,
				Scope:               scope,
				Token:               token,
				GetURLFunc:          getEndpointBaseURL,
				SDPAssetType:        sdpItemType,
				SDPAdapterCategory:  meta.SDPAdapterCategory,
				TerraformMappings:   SDPAssetTypeToTerraformMappings[sdpItemType].Mappings,
				Linker:              linker,
				HTTPClient:          otelhttp.DefaultClient,
				UniqueAttributeKeys: meta.UniqueAttributeKeys,
			}
			if meta.ListEndpointFunc != nil {
				listEndpoint, err := meta.ListEndpointFunc(projectID, zone)
				if err != nil {
					return nil, err
				}
				adapters = append(adapters, NewListableAdapter(listEndpoint, cfg))

				continue
			}

			adapters = append(adapters, NewAdapter(cfg))
		}
	}

	return adapters, nil
}
