package shared

import (
	"fmt"

	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources/shared"
)

// ZoneBase customizes the sources.ZoneBase struct for GCP
// It adds the project ID and zone to the base struct
// and makes them available to concrete wrapper implementations.
type ZoneBase struct {
	projectID string
	zone      string

	*shared.Base
}

// NewZoneBase creates a new ZoneBase struct
func NewZoneBase(
	projectID string,
	zone string,
	category sdp.AdapterCategory,
	item shared.ItemType,
) *ZoneBase {
	return &ZoneBase{
		projectID: projectID,
		zone:      zone,
		Base: shared.NewBase(
			category,
			item,
			[]string{fmt.Sprintf("%s.%s", projectID, zone)},
		),
	}
}

// ProjectID returns the project ID
func (m *ZoneBase) ProjectID() string {
	return m.projectID
}

// Zone returns the zone
func (m *ZoneBase) Zone() string {
	return m.zone
}

// DefaultScope returns the default scope
// Project ID and zone are used to create the default scope.
func (m *ZoneBase) DefaultScope() string {
	return m.Scopes()[0]
}

// RegionBase customizes the sources.Base struct for GCP
// It adds the project ID and region to the base struct
// and makes them available to concrete wrapper implementations.
type RegionBase struct {
	projectID string
	region    string

	*shared.Base
}

// NewRegionBase creates a new RegionBase struct
func NewRegionBase(
	projectID string,
	region string,
	category sdp.AdapterCategory,
	item shared.ItemType,
) *RegionBase {
	return &RegionBase{
		projectID: projectID,
		region:    region,
		Base: shared.NewBase(
			category,
			item,
			[]string{fmt.Sprintf("%s.%s", projectID, region)},
		),
	}
}

// ProjectID returns the project ID
func (m *RegionBase) ProjectID() string {
	return m.projectID
}

// Region returns the region
func (m *RegionBase) Region() string {
	return m.region
}

// DefaultScope returns the default scope
// Project ID and region are used to create the default scope.
func (m *RegionBase) DefaultScope() string {
	return m.Scopes()[0]
}

// ProjectBase customizes the sources.Base struct for GCP
// It adds the project ID to the base struct
// and makes them available to concrete wrapper implementations.
type ProjectBase struct {
	projectID string

	*shared.Base
}

// NewProjectBase creates a new ProjectBase struct
func NewProjectBase(
	projectID string,
	category sdp.AdapterCategory,
	item shared.ItemType,
) *ProjectBase {
	return &ProjectBase{
		projectID: projectID,
		Base: shared.NewBase(
			category,
			item,
			[]string{projectID},
		),
	}
}

// ProjectID returns the project ID
func (m *ProjectBase) ProjectID() string {
	return m.projectID
}

// DefaultScope returns the default scope
// Project ID is used to create the default scope.
func (m *ProjectBase) DefaultScope() string {
	return m.Scopes()[0]
}
