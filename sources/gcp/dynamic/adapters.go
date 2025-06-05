package dynamic

import (
	"fmt"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/overmindtech/cli/discovery"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

var adaptersByScope = map[Scope]map[shared.ItemType]AdapterMeta{}

func init() {
	adaptersByScope = make(map[Scope]map[shared.ItemType]AdapterMeta)
	for sdpItemType, meta := range sdpAssetTypeToAdapterMeta {
		if _, ok := adaptersByScope[meta.Scope]; !ok {
			adaptersByScope[meta.Scope] = make(map[shared.ItemType]AdapterMeta)
		}
		adaptersByScope[meta.Scope][sdpItemType] = meta
	}

}

// Adapters returns a list of discovery.Adapters for the given project ID, token, regions, and zones.
func Adapters(projectID string, token string, regions []string, zones []string, linker *gcpshared.Linker) ([]discovery.Adapter, error) {
	var adapters []discovery.Adapter

	// Global adapters
	for sdpItemType, meta := range adaptersByScope[ScopeGlobal] {
		getEndpointBaseURL, err := meta.GetEndpointBaseURLFunc(projectID)
		if err != nil {
			return nil, err
		}

		cfg := &AdapterConfig{
			ProjectID:          projectID,
			Scope:              projectID,
			Token:              token,
			GetBaseURL:         getEndpointBaseURL,
			SDPAssetType:       sdpItemType,
			SDPAdapterCategory: meta.SDPAdapterCategory,
			TerraformMappings:  SDPAssetTypeToTerraformMappings[sdpItemType],
			Linker:             linker,
			HTTPClient:         otelhttp.DefaultClient,
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
		for sdpItemType, meta := range adaptersByScope[ScopeRegional] {
			getEndpointBaseURL, err := meta.GetEndpointBaseURLFunc(projectID, region)
			if err != nil {
				return nil, err
			}

			scope := fmt.Sprintf("%s.%s", projectID, region)

			cfg := &AdapterConfig{
				ProjectID:          projectID,
				Scope:              scope,
				Token:              token,
				GetBaseURL:         getEndpointBaseURL,
				SDPAssetType:       sdpItemType,
				SDPAdapterCategory: meta.SDPAdapterCategory,
				TerraformMappings:  SDPAssetTypeToTerraformMappings[sdpItemType],
				Linker:             linker,
				HTTPClient:         otelhttp.DefaultClient,
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
		for sdpItemType, meta := range adaptersByScope[ScopeZonal] {
			getEndpointBaseURL, err := meta.GetEndpointBaseURLFunc(projectID, zone)
			if err != nil {
				return nil, err
			}

			scope := fmt.Sprintf("%s.%s", projectID, zone)

			cfg := &AdapterConfig{
				ProjectID:          projectID,
				Scope:              scope,
				Token:              token,
				GetBaseURL:         getEndpointBaseURL,
				SDPAssetType:       sdpItemType,
				SDPAdapterCategory: meta.SDPAdapterCategory,
				TerraformMappings:  SDPAssetTypeToTerraformMappings[sdpItemType],
				Linker:             linker,
				HTTPClient:         otelhttp.DefaultClient,
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
