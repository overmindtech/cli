package shared

import (
	"fmt"

	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources/shared"
)

// ResourceGroupScope represents a subscription and resource group pair.
// It is used by multi-scope adapters to handle multiple resource groups.
type ResourceGroupScope struct {
	SubscriptionID string
	ResourceGroup   string
}

// NewResourceGroupScope creates a ResourceGroupScope for the given subscription and resource group.
func NewResourceGroupScope(subscriptionID, resourceGroup string) ResourceGroupScope {
	return ResourceGroupScope{
		SubscriptionID: subscriptionID,
		ResourceGroup:   resourceGroup,
	}
}

// ToScope returns the scope string in format "{subscriptionId}.{resourceGroup}".
func (r ResourceGroupScope) ToScope() string {
	return fmt.Sprintf("%s.%s", r.SubscriptionID, r.ResourceGroup)
}

// MultiResourceGroupBase provides shared multi-scope behavior for resource-group-scoped adapters.
// One adapter instance handles all resource groups in resourceGroupScopes.
type MultiResourceGroupBase struct {
	resourceGroupScopes []ResourceGroupScope
	*shared.Base
}

// NewMultiResourceGroupBase creates a MultiResourceGroupBase that supports multiple resource group scopes.
func NewMultiResourceGroupBase(
	resourceGroupScopes []ResourceGroupScope,
	category sdp.AdapterCategory,
	item shared.ItemType,
) *MultiResourceGroupBase {
	if len(resourceGroupScopes) == 0 {
		panic("NewMultiResourceGroupBase: resourceGroupScopes cannot be empty")
	}

	scopeStrings := make([]string, 0, len(resourceGroupScopes))
	for _, rgScope := range resourceGroupScopes {
		scopeStrings = append(scopeStrings, rgScope.ToScope())
	}

	return &MultiResourceGroupBase{
		resourceGroupScopes: resourceGroupScopes,
		Base:                shared.NewBase(category, item, scopeStrings),
	}
}

// ResourceGroupScopeFromScope parses a scope string and returns the matching ResourceGroupScope
// if it is one of the adapter's configured scopes.
func (m *MultiResourceGroupBase) ResourceGroupScopeFromScope(scope string) (ResourceGroupScope, error) {
	subscriptionID := SubscriptionIDFromScope(scope)
	resourceGroup := ResourceGroupFromScope(scope)
	if subscriptionID == "" || resourceGroup == "" {
		return ResourceGroupScope{}, fmt.Errorf("invalid scope format %q: expected subscriptionId.resourceGroup", scope)
	}

	rgScope := NewResourceGroupScope(subscriptionID, resourceGroup)
	for _, s := range m.resourceGroupScopes {
		if s.SubscriptionID == rgScope.SubscriptionID && s.ResourceGroup == rgScope.ResourceGroup {
			return rgScope, nil
		}
	}
	return ResourceGroupScope{}, fmt.Errorf("scope %s not found in adapter resource group scopes", scope)
}

// ResourceGroupScopes returns the configured resource group scopes for this adapter.
func (m *MultiResourceGroupBase) ResourceGroupScopes() []ResourceGroupScope {
	return m.resourceGroupScopes
}

// DefaultScope returns the first scope (for compatibility where a single default is needed).
func (m *MultiResourceGroupBase) DefaultScope() string {
	return m.Scopes()[0]
}
