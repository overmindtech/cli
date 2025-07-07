package shared

import (
	"fmt"

	"github.com/overmindtech/cli/sdp-go"
)

// Base is a struct that holds fundamental pieces for creating an adapter.
type Base struct {
	category sdp.AdapterCategory
	itemType ItemType
	scopes   []string
}

// NewBase creates a new Base instance with the provided parameters and options.
func NewBase(
	category sdp.AdapterCategory,
	item ItemType,
	scopes []string,
) *Base {
	base := &Base{
		category: category,
		itemType: item,
		scopes:   scopes,
	}

	return base
}

// Category returns the adapter category.
func (u *Base) Category() sdp.AdapterCategory {
	return u.category
}

// Type returns a string representation of the type, combining source family, API, and resource.
func (u *Base) Type() string {
	return u.itemType.String()
}

// Name returns the name of the adapter.
func (u *Base) Name() string {
	return fmt.Sprintf("%s-adapter", u.Type())
}

// PotentialLinks returns a map of potential links for the itemType.
func (*Base) PotentialLinks() map[ItemType]bool {
	return nil
}

// AdapterMetadata returns the adapter metadata.
// This can be created from the wrapper.
// Otherwise, it will be generated when transforming the wrapper to an adapter.
func (u *Base) AdapterMetadata() *sdp.AdapterMetadata {
	return nil
}

// TerraformMappings returns a slice of Terraform mappings for the itemType.
// This is optional.
func (u *Base) TerraformMappings() []*sdp.TerraformMapping {
	return nil
}

// Scopes returns a slice of strings representing the scopes for the itemType.
func (u *Base) Scopes() []string {
	return u.scopes
}

// ItemType returns the itemType which the adapter is created for.
func (u *Base) ItemType() ItemType {
	return u.itemType
}

// IAMPermissions returns a slice of IAM permissions required for the adapter.
// This is optional, not all adapters will implement this.
func (u *Base) IAMPermissions() []string {
	return nil
}
