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
	GCPBase
	zone string

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
		GCPBase: GCPBase{
			projectID: projectID,
		},
		zone: zone,
		Base: shared.NewBase(
			category,
			item,
			[]string{fmt.Sprintf("%s.%s", projectID, zone)},
		),
	}
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
	GCPBase
	region string

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
		GCPBase: GCPBase{
			projectID: projectID,
		},
		region: region,
		Base: shared.NewBase(
			category,
			item,
			[]string{fmt.Sprintf("%s.%s", projectID, region)},
		),
	}
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
	GCPBase

	*shared.Base
}

// NewProjectBase creates a new ProjectBase struct
func NewProjectBase(
	projectID string,
	category sdp.AdapterCategory,
	item shared.ItemType,
) *ProjectBase {
	return &ProjectBase{
		GCPBase: GCPBase{
			projectID: projectID,
		},
		Base: shared.NewBase(
			category,
			item,
			[]string{projectID},
		),
	}
}

// DefaultScope returns the default scope
// Project ID is used to create the default scope.
func (m *ProjectBase) DefaultScope() string {
	return m.Scopes()[0]
}

type GCPBase struct {
	projectID string
}

func (g *GCPBase) PredefinedRole() string {
	panic("Predefined role not implemented")
}

func (g *GCPBase) ProjectID() string {
	return g.projectID
}
