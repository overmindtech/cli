package dynamic

import (
	"fmt"
	"net/http"

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

// Adapters returns a list of discovery.Adapters for the given project ID, regions, and zones.
func Adapters(projectID string, regions []string, zones []string, linker *gcpshared.Linker, httpCli *http.Client, manualAdapters map[string]bool) ([]discovery.Adapter, error) {
	var adapters []discovery.Adapter

	// Project level adapters
	for sdpItemType, meta := range adaptersByScope[gcpshared.ScopeProject] {
		if _, ok := manualAdapters[sdpItemType.String()]; ok {
			// Skip, because we have a manual adapter for this item type
			continue
		}

		if meta.InDevelopment {
			// Skip adapters that are in development
			// This is useful for testing new adapters without exposing them to production
			continue
		}

		adapter, err := MakeAdapter(sdpItemType, meta, linker, httpCli, projectID)
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

			if meta.InDevelopment {
				// Skip adapters that are in development
				// This is useful for testing new adapters without exposing them to production
				continue
			}

			adapter, err := MakeAdapter(sdpItemType, meta, linker, httpCli, projectID, region)
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

			if meta.InDevelopment {
				// Skip adapters that are in development
				// This is useful for testing new adapters without exposing them to production
				continue
			}

			adapter, err := MakeAdapter(sdpItemType, meta, linker, httpCli, projectID, zone)
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

// MakeAdapter creates a new GCP dynamic adapter based on the provided SDP item type and metadata.
// It expects the scope components (project ID, region, zone) to be passed as options.
// It assumes that the first option is always the project ID, and subsequent options depend on the scope type.
// Possible options are:
// - For project scope: project ID
// - For regional scope: project ID and region
// - For zonal scope: project ID, region, and zone
func MakeAdapter(sdpItemType shared.ItemType, meta gcpshared.AdapterMeta, linker *gcpshared.Linker, httpCli *http.Client, opts ...string) (discovery.Adapter, error) {
	getEndpointBaseURL, err := meta.GetEndpointBaseURLFunc(opts...)
	if err != nil {
		return nil, err
	}

	scope := makeScope(meta, opts...)
	if scope == "" {
		return nil, fmt.Errorf("invalid scope for adapter %s with options %v", sdpItemType.String(), opts)
	}

	cfg := &AdapterConfig{
		ProjectID:           opts[0],
		Scope:               scope,
		GetURLFunc:          getEndpointBaseURL,
		SDPAssetType:        sdpItemType,
		SDPAdapterCategory:  meta.SDPAdapterCategory,
		TerraformMappings:   SDPAssetTypeToTerraformMappings[sdpItemType].Mappings,
		Linker:              linker,
		HTTPClient:          httpCli,
		UniqueAttributeKeys: meta.UniqueAttributeKeys,
		IAMPermissions:      meta.IAMPermissions,
	}

	// Add IAM permissions to the global map
	for _, perm := range meta.IAMPermissions {
		gcpshared.IAMPermissions[perm] = true
	}

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

		return NewSearchableListableAdapter(searchEndpointFunc, listEndpoint, cfg, meta.SearchDescription)
	case Searchable:
		searchEndpointFunc, err := meta.SearchEndpointFunc(opts...)
		if err != nil {
			return nil, err
		}

		return NewSearchableAdapter(searchEndpointFunc, cfg, meta.SearchDescription)
	case Listable:
		listEndpoint, err := meta.ListEndpointFunc(opts...)
		if err != nil {
			return nil, err
		}

		return NewListableAdapter(listEndpoint, cfg)
	case Standard:
		return NewAdapter(cfg)
	default:
		return nil, fmt.Errorf("unknown adapter type %s", adapterType(meta))
	}
}

// makeScope constructs the scope string based on the provided metadata and options.
// It uses the first option as the project ID, and for regional or zonal scopes, it appends the region or zone.
// For example:
// - For project scope: opts[0] (project ID)
// - For regional scope: opts[0] (project ID) + opts[1] (region)
// - For zonal scope: opts[0] (project ID) + opts[1] (zone)
// The second option can only be region or zone, depending on the scope type.
func makeScope(meta gcpshared.AdapterMeta, opts ...string) string {
	switch meta.Scope {
	case gcpshared.ScopeProject:
		return opts[0]
	case gcpshared.ScopeRegional, gcpshared.ScopeZonal:
		return fmt.Sprintf("%s.%s", opts[0], opts[1])
	default:
		return ""
	}
}
