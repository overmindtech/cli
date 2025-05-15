package example

import (
	"fmt"

	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources/shared"
)

// Base customizes the sources.Base struct
// It adds the project ID and zone to the base struct
// and makes them available to concrete wrapper implementations.
type Base struct {
	projectID string
	zone      string

	*shared.Base
}

// NewBase creates a new Base struct
func NewBase(
	projectID string,
	zone string,
	category sdp.AdapterCategory,
	item shared.ItemType,
) *Base {
	return &Base{
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
func (m *Base) ProjectID() string {
	return m.projectID
}

// Zone returns the zone
func (m *Base) Zone() string {
	return m.zone
}
