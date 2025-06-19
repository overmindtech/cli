package dynamic

import (
	"fmt"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/overmindtech/cli/discovery"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

type typeOfAdapter string

const (
	Standard           typeOfAdapter = "standard"
	Listable           typeOfAdapter = "listable"
	Searchable         typeOfAdapter = "searchable"
	SearchableListable typeOfAdapter = "searchableListable"
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

		adapter, err := makeAdapter(meta, cfg, projectID)
		if err != nil {
			return nil, fmt.Errorf("failed to add adapter for %s: %w", sdpItemType, err)
		}

		adapters = append(adapters, adapter)
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

			adapter, err := makeAdapter(meta, cfg, projectID, region)
			if err != nil {
				return nil, fmt.Errorf("failed to add adapter for %s in region %s: %w", sdpItemType, region, err)
			}

			adapters = append(adapters, adapter)
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

			adapter, err := makeAdapter(meta, cfg, projectID, zone)
			if err != nil {
				return nil, fmt.Errorf("failed to add adapter for %s in zone %s: %w", sdpItemType, zone, err)
			}

			adapters = append(adapters, adapter)
		}
	}

	return adapters, nil
}

func adapterType(meta gcpshared.AdapterMeta) typeOfAdapter {
	if meta.ListEndpointFunc != nil && meta.SearchEndpointFunc == nil {
		return Listable
	}

	if meta.SearchEndpointFunc != nil && meta.ListEndpointFunc == nil {
		return Searchable
	}

	if meta.ListEndpointFunc != nil && meta.SearchEndpointFunc != nil {
		return SearchableListable
	}

	return Standard
}

func makeAdapter(meta gcpshared.AdapterMeta, cfg *AdapterConfig, opts ...string) (discovery.Adapter, error) {
	switch adapterType(meta) {
	case SearchableListable:
		searchEndpointFunc, err := meta.SearchEndpointFunc(opts...)
		if err != nil {
			return nil, err
		}

		listEndpoint, err := meta.ListEndpointFunc(opts...)
		if err != nil {
			return nil, err
		}

		return NewSearchableListableAdapter(searchEndpointFunc, listEndpoint, cfg), nil
	case Searchable:
		searchEndpointFunc, err := meta.SearchEndpointFunc(opts...)
		if err != nil {
			return nil, err
		}

		return NewSearchableAdapter(searchEndpointFunc, cfg), nil
	case Listable:
		listEndpoint, err := meta.ListEndpointFunc(opts...)
		if err != nil {
			return nil, err
		}

		return NewListableAdapter(listEndpoint, cfg), nil
	case Standard:
		return NewAdapter(cfg), nil
	default:
		return nil, fmt.Errorf("unknown adapter type %s", adapterType(meta))
	}
}
