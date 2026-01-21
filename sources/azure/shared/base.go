package shared

import (
	"fmt"
	"strings"

	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources/shared"
)

// ResourceGroupBase customizes the sources.Base struct for Azure
// It adds the subscription ID and resource group to the base struct
// and makes them available to concrete wrapper implementations.
type ResourceGroupBase struct {
	AzureBase
	resourceGroup string

	*shared.Base
}

// NewResourceGroupBase creates a new ResourceGroupBase struct
func NewResourceGroupBase(
	subscriptionID string,
	resourceGroup string,
	category sdp.AdapterCategory,
	item shared.ItemType,
) *ResourceGroupBase {
	base := &ResourceGroupBase{
		AzureBase: AzureBase{
			subscriptionID: subscriptionID,
		},
		resourceGroup: resourceGroup,
	}
	base.Base = shared.NewBase(
		category,
		item,
		[]string{fmt.Sprintf("%s.%s", base.SubscriptionID(), resourceGroup)},
	)
	return base
}

// ResourceGroup returns the resource group
func (m *ResourceGroupBase) ResourceGroup() string {
	return m.resourceGroup
}

// DefaultScope returns the default scope
// Subscription ID and resource group are used to create the default scope.
func (m *ResourceGroupBase) DefaultScope() string {
	return m.Scopes()[0]
}

// ResourceGroupFromScope returns the resource group from a scope string.
// Scope format is "{subscriptionId}.{resourceGroup}".
func ResourceGroupFromScope(scope string) string {
	if scope == "" {
		return ""
	}
	parts := strings.SplitN(scope, ".", 2)
	if len(parts) < 2 || parts[1] == "" {
		return ""
	}
	return parts[1]
}

// SubscriptionIDFromScope returns the subscription ID from a scope string.
// Scope format is "{subscriptionId}.{resourceGroup}".
func SubscriptionIDFromScope(scope string) string {
	if scope == "" {
		return ""
	}
	parts := strings.SplitN(scope, ".", 2)
	if len(parts) < 2 || parts[0] == "" {
		return ""
	}
	return parts[0]
}

// SubscriptionBase customizes the sources.Base struct for Azure
// It adds the subscription ID to the base struct
// and makes them available to concrete wrapper implementations.
type SubscriptionBase struct {
	AzureBase

	*shared.Base
}

// NewSubscriptionBase creates a new SubscriptionBase struct
func NewSubscriptionBase(
	subscriptionID string,
	category sdp.AdapterCategory,
	item shared.ItemType,
) *SubscriptionBase {
	base := &SubscriptionBase{
		AzureBase: AzureBase{
			subscriptionID: subscriptionID,
		},
	}
	base.Base = shared.NewBase(
		category,
		item,
		[]string{base.SubscriptionID()},
	)
	return base
}

// DefaultScope returns the default scope
// Subscription ID is used to create the default scope.
func (m *SubscriptionBase) DefaultScope() string {
	return m.Scopes()[0]
}

// AzureBase is the base struct for all Azure adapters.
// It contains common fields and methods for Azure resources.
type AzureBase struct {
	subscriptionID string
}

// SubscriptionID returns the subscription ID
func (a *AzureBase) SubscriptionID() string {
	return a.subscriptionID
}
