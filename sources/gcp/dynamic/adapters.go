package dynamic

import (
	"fmt"
	"net/http"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdpcache"
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

// Adapters returns a list of discovery.Adapters for the given project ID, regions, and zones.
// Each adapter type is created once and handles all locations of its scope type.
func Adapters(projectID string, regions []string, zones []string, linker *gcpshared.Linker, httpCli *http.Client, manualAdapters map[string]bool, cache sdpcache.Cache) ([]discovery.Adapter, error) {
	var adapters []discovery.Adapter

	// Group adapters by location level
	adaptersByLevel := make(map[gcpshared.LocationLevel]map[shared.ItemType]gcpshared.AdapterMeta)
	for sdpItemType, meta := range gcpshared.SDPAssetTypeToAdapterMeta {
		if meta.InDevelopment {
			// Skip adapters that are in development
			// This is useful for testing new adapters without exposing them to production
			continue
		}
		if _, ok := adaptersByLevel[meta.LocationLevel]; !ok {
			adaptersByLevel[meta.LocationLevel] = make(map[shared.ItemType]gcpshared.AdapterMeta)
		}
		adaptersByLevel[meta.LocationLevel][sdpItemType] = meta
	}

	// Build LocationInfo slices for each location level
	projectLocation := []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)}

	regionLocations := make([]gcpshared.LocationInfo, 0, len(regions))
	for _, region := range regions {
		regionLocations = append(regionLocations, gcpshared.NewRegionalLocation(projectID, region))
	}

	zoneLocations := make([]gcpshared.LocationInfo, 0, len(zones))
	for _, zone := range zones {
		zoneLocations = append(zoneLocations, gcpshared.NewZonalLocation(projectID, zone))
	}

	// Create project-level adapters (one per type)
	for sdpItemType := range adaptersByLevel[gcpshared.ProjectLevel] {
		if _, ok := manualAdapters[sdpItemType.String()]; ok {
			// Skip, because we have a manual adapter for this item type
			continue
		}

		adapter, err := MakeAdapter(sdpItemType, linker, httpCli, cache, projectLocation)
		if err != nil {
			return nil, fmt.Errorf("failed to add adapter for %s: %w", sdpItemType, err)
		}

		adapters = append(adapters, adapter)
	}

	// Create regional adapters (one per type, handling all regions)
	if len(regionLocations) > 0 {
		for sdpItemType := range adaptersByLevel[gcpshared.RegionalLevel] {
			if _, ok := manualAdapters[sdpItemType.String()]; ok {
				// Skip, because we have a manual adapter for this item type
				continue
			}

			adapter, err := MakeAdapter(sdpItemType, linker, httpCli, cache, regionLocations)
			if err != nil {
				return nil, fmt.Errorf("failed to add adapter for %s: %w", sdpItemType, err)
			}

			adapters = append(adapters, adapter)
		}
	}

	// Create zonal adapters (one per type, handling all zones)
	if len(zoneLocations) > 0 {
		for sdpItemType := range adaptersByLevel[gcpshared.ZonalLevel] {
			if _, ok := manualAdapters[sdpItemType.String()]; ok {
				// Skip, because we have a manual adapter for this item type
				continue
			}

			adapter, err := MakeAdapter(sdpItemType, linker, httpCli, cache, zoneLocations)
			if err != nil {
				return nil, fmt.Errorf("failed to add adapter for %s: %w", sdpItemType, err)
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
// It accepts a slice of LocationInfo representing all locations this adapter should handle.
func MakeAdapter(sdpItemType shared.ItemType, linker *gcpshared.Linker, httpCli *http.Client, cache sdpcache.Cache, locations []gcpshared.LocationInfo) (discovery.Adapter, error) {
	meta, ok := gcpshared.SDPAssetTypeToAdapterMeta[sdpItemType]
	if !ok {
		return nil, fmt.Errorf("no adapter metadata found for item type %s", sdpItemType.String())
	}

	// Validate that all locations match the adapter's expected scope type
	for _, loc := range locations {
		if loc.LocationLevel() != meta.LocationLevel {
			return nil, fmt.Errorf("location %s has scope %s, expected %s", loc.ToScope(), loc.LocationLevel(), meta.LocationLevel)
		}
	}

	cfg := &AdapterConfig{
		Locations:            locations,
		GetURLFunc:           meta.GetEndpointFunc,
		SDPAssetType:         sdpItemType,
		SDPAdapterCategory:   meta.SDPAdapterCategory,
		TerraformMappings:    gcpshared.SDPAssetTypeToTerraformMappings[sdpItemType].Mappings,
		Linker:               linker,
		HTTPClient:           httpCli,
		UniqueAttributeKeys:  meta.UniqueAttributeKeys,
		IAMPermissions:       meta.IAMPermissions,
		NameSelector:         meta.NameSelector,
		ListResponseSelector: meta.ListResponseSelector,
	}

	switch adapterType(meta) {
	case SearchableListable:
		return NewSearchableListableAdapter(meta.SearchEndpointFunc, meta.ListEndpointFunc, cfg, meta.SearchDescription, cache), nil
	case Searchable:
		return NewSearchableAdapter(meta.SearchEndpointFunc, cfg, meta.SearchDescription, cache), nil
	case Listable:
		return NewListableAdapter(meta.ListEndpointFunc, cfg, cache), nil
	case Standard:
		return NewAdapter(cfg, cache), nil
	default:
		return nil, fmt.Errorf("unknown adapter type %s", adapterType(meta))
	}
}
