package shared

import (
	"fmt"
	"strings"
)

// LocationInfo encapsulates location information for GCP resources.
// It provides type-safe handling of different scope types (project, regional, zonal)
// and simplifies scope validation and URL generation.
type LocationInfo struct {
	ProjectID string
	Region    string // Empty for project-level resources
	Zone      string // Empty for project and regional resources
}

// LocationFromScope parses a scope string into a LocationInfo struct.
//
// Supported formats:
//   - Project scope:  "project-id"
//   - Regional scope: "project-id.region" (e.g., "my-project.us-central1")
//   - Zonal scope:    "project-id.zone"   (e.g., "my-project.us-central1-a")
//
// Scope detection uses the dash count in the second component:
//   - 1 dash => region
//   - 2 dashes => zone
func LocationFromScope(scope string) (LocationInfo, error) {
	if scope == "" {
		return LocationInfo{}, fmt.Errorf("scope cannot be empty")
	}

	parts := strings.Split(scope, ".")
	switch len(parts) {
	case 1:
		return LocationInfo{
			ProjectID: parts[0],
		}, nil
	case 2:
		projectID := parts[0]
		location := parts[1]

		switch strings.Count(location, "-") {
		case 1:
			return LocationInfo{
				ProjectID: projectID,
				Region:    location,
			}, nil
		case 2:
			return LocationInfo{
				ProjectID: projectID,
				Region:    ZoneToRegion(location),
				Zone:      location,
			}, nil
		default:
			return LocationInfo{}, fmt.Errorf("invalid location format: %q", location)
		}
	default:
		return LocationInfo{}, fmt.Errorf("invalid scope format: %q", scope)
	}
}

// ToScope converts LocationInfo back to scope string format.
// If Zone is set, returns "project.zone".
// If Region is set but Zone is empty, returns "project.region".
// Otherwise, returns just the project ID.
func (l LocationInfo) ToScope() string {
	if l.Zone != "" {
		return fmt.Sprintf("%s.%s", l.ProjectID, l.Zone)
	}
	if l.Region != "" {
		return fmt.Sprintf("%s.%s", l.ProjectID, l.Region)
	}
	return l.ProjectID
}

// LocationLevel returns the calculated scope type based on Zone and Region fields.
// If Zone is set, returns ZonalLevel.
// If Region is set (but Zone is empty), returns RegionalLevel.
// Otherwise, returns ProjectLevel.
func (l LocationInfo) LocationLevel() LocationLevel {
	if l.Zone != "" {
		return ZonalLevel
	}
	if l.Region != "" {
		return RegionalLevel
	}
	return ProjectLevel
}

// ProjectLevel returns true if this is a project-level location (no region or zone).
func (l LocationInfo) ProjectLevel() bool {
	return l.Zone == "" && l.Region == ""
}

// Regional returns true if this is a regional location (has region but no zone).
func (l LocationInfo) Regional() bool {
	return l.Region != "" && l.Zone == ""
}

// Zonal returns true if this is a zonal location (has zone).
func (l LocationInfo) Zonal() bool {
	return l.Zone != ""
}

// Equals compares two LocationInfo instances for equality.
func (l LocationInfo) Equals(other LocationInfo) bool {
	return l.ProjectID == other.ProjectID &&
		l.Region == other.Region &&
		l.Zone == other.Zone
}

// Validate checks if the LocationInfo has valid values.
func (l LocationInfo) Validate() error {
	if l.ProjectID == "" {
		return fmt.Errorf("project ID cannot be empty")
	}
	// If zone is set, region should be derivable
	if l.Zone != "" && l.Region == "" {
		return fmt.Errorf("zone is set but region is empty")
	}
	return nil
}

// String returns a human-readable representation of the LocationInfo.
func (l LocationInfo) String() string {
	return l.ToScope()
}

// NewProjectLocation creates a LocationInfo for a project-level resource.
func NewProjectLocation(projectID string) LocationInfo {
	return LocationInfo{
		ProjectID: projectID,
	}
}

// NewRegionalLocation creates a LocationInfo for a regional resource.
func NewRegionalLocation(projectID, region string) LocationInfo {
	return LocationInfo{
		ProjectID: projectID,
		Region:    region,
	}
}

// NewZonalLocation creates a LocationInfo for a zonal resource.
func NewZonalLocation(projectID, zone string) LocationInfo {
	return LocationInfo{
		ProjectID: projectID,
		Region:    ZoneToRegion(zone),
		Zone:      zone,
	}
}

// LocationsToScopes converts a slice of LocationInfo to a slice of scope strings.
func LocationsToScopes(locations []LocationInfo) []string {
	scopes := make([]string, 0, len(locations))
	for _, loc := range locations {
		scopes = append(scopes, loc.ToScope())
	}
	return scopes
}

// ValidateScopeForLocations checks if a scope string matches any of the configured locations.
// Returns the matching LocationInfo if found, or an error if the scope is not valid for these locations.
func ValidateScopeForLocations(scope string, locations []LocationInfo) (LocationInfo, error) {
	location, err := LocationFromScope(scope)
	if err != nil {
		return LocationInfo{}, fmt.Errorf("failed to parse scope %s: %w", scope, err)
	}

	for _, loc := range locations {
		if loc.Equals(location) {
			return location, nil
		}
	}
	return LocationInfo{}, fmt.Errorf("scope %s not found in configured locations", scope)
}

// ParseAggregatedListScope parses a scope key from aggregatedList response
// Examples:
//   - "zones/us-central1-a" -> LocationInfo{ProjectID: projectID, Zone: "us-central1-a", Region: "us-central1"}
//   - "regions/us-central1" -> LocationInfo{ProjectID: projectID, Region: "us-central1"}
func ParseAggregatedListScope(projectID, scopeKey string) (LocationInfo, error) {
	parts := strings.Split(scopeKey, "/")
	if len(parts) != 2 {
		return LocationInfo{}, fmt.Errorf("invalid scope key format: %s", scopeKey)
	}

	scopeType := parts[0] // "zones" or "regions"
	locationName := parts[1]

	switch scopeType {
	case "zones":
		return NewZonalLocation(projectID, locationName), nil
	case "regions":
		return NewRegionalLocation(projectID, locationName), nil
	default:
		return LocationInfo{}, fmt.Errorf("unsupported scope type: %s", scopeType)
	}
}
