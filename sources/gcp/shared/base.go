package shared

import (
	"fmt"

	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources/shared"
)

// ZoneBase provides shared multi-scope behavior for zonal adapters.
type ZoneBase struct {
	locations []LocationInfo
	*shared.Base
}

// NewZoneBase creates a ZoneBase that supports multiple zones.
func NewZoneBase(locations []LocationInfo, category sdp.AdapterCategory, item shared.ItemType) *ZoneBase {
	for _, location := range locations {
		if !location.Zonal() {
			panic(fmt.Sprintf("NewZoneBase: location %s is not zonal", location.ToScope()))
		}
	}

	scopes := make([]string, 0, len(locations))
	for _, location := range locations {
		scopes = append(scopes, location.ToScope())
	}

	return &ZoneBase{
		locations: locations,
		Base:      shared.NewBase(category, item, scopes),
	}
}

// PredefinedRole implements the sources.WithPredefinedRole interface.
// Individual adapters must override this method.
func (z *ZoneBase) PredefinedRole() string {
	panic("PredefinedRole not implemented - adapter must override this method")
}

// LocationFromScope parses a scope string into a zonal LocationInfo.
func (z *ZoneBase) LocationFromScope(scope string) (LocationInfo, error) {
	location, err := LocationFromScope(scope)
	if err != nil {
		return LocationInfo{}, fmt.Errorf("failed to parse scope %s: %w", scope, err)
	}
	if !location.Zonal() {
		return LocationInfo{}, fmt.Errorf("scope %s is not zonal", scope)
	}
	for _, loc := range z.locations {
		if location.Equals(loc) {
			return location, nil
		}
	}
	return LocationInfo{}, fmt.Errorf("scope %s not found in adapter locations", scope)
}

// ZoneFromScope returns a zone string from the scope for backward compatibility.
func (z *ZoneBase) ZoneFromScope(scope string) (string, error) {
	location, err := z.LocationFromScope(scope)
	if err != nil {
		return "", err
	}
	return location.Zone, nil
}

// RegionBase provides shared multi-scope behavior for regional adapters.
type RegionBase struct {
	locations []LocationInfo
	*shared.Base
}

// NewRegionBase creates a RegionBase that supports multiple regions.
func NewRegionBase(locations []LocationInfo, category sdp.AdapterCategory, item shared.ItemType) *RegionBase {
	for _, location := range locations {
		if !location.Regional() {
			panic(fmt.Sprintf("NewRegionBase: location %s is not regional", location.ToScope()))
		}
	}

	scopes := make([]string, 0, len(locations))
	for _, location := range locations {
		scopes = append(scopes, location.ToScope())
	}

	return &RegionBase{
		locations: locations,
		Base:      shared.NewBase(category, item, scopes),
	}
}

// PredefinedRole implements the sources.WithPredefinedRole interface.
// Individual adapters must override this method.
func (r *RegionBase) PredefinedRole() string {
	panic("PredefinedRole not implemented - adapter must override this method")
}

// LocationFromScope parses a scope string into a regional LocationInfo.
func (r *RegionBase) LocationFromScope(scope string) (LocationInfo, error) {
	location, err := LocationFromScope(scope)
	if err != nil {
		return LocationInfo{}, fmt.Errorf("failed to parse scope %s: %w", scope, err)
	}
	if !location.Regional() {
		return LocationInfo{}, fmt.Errorf("scope %s is not regional", scope)
	}
	for _, loc := range r.locations {
		if location.Equals(loc) {
			return location, nil
		}
	}
	return LocationInfo{}, fmt.Errorf("scope %s not found in adapter locations", scope)
}

// RegionFromScope returns a region string from the scope for backward compatibility.
func (r *RegionBase) RegionFromScope(scope string) (string, error) {
	location, err := r.LocationFromScope(scope)
	if err != nil {
		return "", err
	}
	return location.Region, nil
}

// ProjectBase provides shared behavior for project-scoped adapters.
type ProjectBase struct {
	locations []LocationInfo
	*shared.Base
}

// NewProjectBase creates a ProjectBase that supports multiple projects.
func NewProjectBase(locations []LocationInfo, category sdp.AdapterCategory, item shared.ItemType) *ProjectBase {
	return NewProjectBaseFromLocations(locations, category, item)
}

// NewProjectBase creates a ProjectBase that supports multiple projects.
func NewProjectBaseFromLocations(locations []LocationInfo, category sdp.AdapterCategory, item shared.ItemType) *ProjectBase {
	scopes := make([]string, 0, len(locations))
	for _, location := range locations {
		scopes = append(scopes, location.ToScope())
	}

	return &ProjectBase{
		locations: locations,
		Base:      shared.NewBase(category, item, scopes),
	}
}

// PredefinedRole implements the sources.WithPredefinedRole interface.
// Individual adapters must override this method.
func (p *ProjectBase) PredefinedRole() string {
	panic("PredefinedRole not implemented - adapter must override this method")
}

// LocationFromScope parses a scope string into a project LocationInfo.
func (p *ProjectBase) LocationFromScope(scope string) (LocationInfo, error) {
	location, err := LocationFromScope(scope)
	if err != nil {
		return LocationInfo{}, fmt.Errorf("failed to parse scope %s: %w", scope, err)
	}
	if !location.ProjectLevel() {
		return LocationInfo{}, fmt.Errorf("scope %s is not project-level", scope)
	}
	for _, loc := range p.locations {
		if location.Equals(loc) {
			return location, nil
		}
	}
	return LocationInfo{}, fmt.Errorf("scope %s not found in adapter locations", scope)
}
