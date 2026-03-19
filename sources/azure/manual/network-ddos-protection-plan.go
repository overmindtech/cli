package manual

import (
	"context"
	"errors"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v9"
	"github.com/overmindtech/cli/go/discovery"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/shared"
)

var NetworkDdosProtectionPlanLookupByName = shared.NewItemTypeLookup("name", azureshared.NetworkDdosProtectionPlan)

type networkDdosProtectionPlanWrapper struct {
	client clients.DdosProtectionPlansClient

	*azureshared.MultiResourceGroupBase
}

func NewNetworkDdosProtectionPlan(client clients.DdosProtectionPlansClient, resourceGroupScopes []azureshared.ResourceGroupScope) sources.ListableWrapper {
	return &networkDdosProtectionPlanWrapper{
		client: client,
		MultiResourceGroupBase: azureshared.NewMultiResourceGroupBase(
			resourceGroupScopes,
			sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
			azureshared.NetworkDdosProtectionPlan,
		),
	}
}

// ref: https://learn.microsoft.com/en-us/rest/api/virtualnetwork/ddos-protection-plans/list-by-resource-group
func (n networkDdosProtectionPlanWrapper) List(ctx context.Context, scope string) ([]*sdp.Item, *sdp.QueryError) {
	rgScope, err := n.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, n.Type())
	}
	pager := n.client.NewListByResourceGroupPager(rgScope.ResourceGroup, nil)

	var items []*sdp.Item
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, azureshared.QueryError(err, scope, n.Type())
		}
		for _, plan := range page.Value {
			if plan.Name == nil {
				continue
			}
			item, sdpErr := n.azureDdosProtectionPlanToSDPItem(plan, scope)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}
	return items, nil
}

func (n networkDdosProtectionPlanWrapper) ListStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string) {
	rgScope, err := n.ResourceGroupScopeFromScope(scope)
	if err != nil {
		stream.SendError(azureshared.QueryError(err, scope, n.Type()))
		return
	}
	pager := n.client.NewListByResourceGroupPager(rgScope.ResourceGroup, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			stream.SendError(azureshared.QueryError(err, scope, n.Type()))
			return
		}
		for _, plan := range page.Value {
			if plan.Name == nil {
				continue
			}
			item, sdpErr := n.azureDdosProtectionPlanToSDPItem(plan, scope)
			if sdpErr != nil {
				stream.SendError(sdpErr)
				continue
			}
			cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
			stream.SendItem(item)
		}
	}
}

// ref: https://learn.microsoft.com/en-us/rest/api/virtualnetwork/ddos-protection-plans/get
func (n networkDdosProtectionPlanWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) != 1 {
		return nil, azureshared.QueryError(errors.New("query must be exactly one part (DDoS protection plan name)"), scope, n.Type())
	}
	planName := queryParts[0]
	if planName == "" {
		return nil, azureshared.QueryError(errors.New("DDoS protection plan name cannot be empty"), scope, n.Type())
	}

	rgScope, err := n.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, n.Type())
	}
	resp, err := n.client.Get(ctx, rgScope.ResourceGroup, planName, nil)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, n.Type())
	}
	return n.azureDdosProtectionPlanToSDPItem(&resp.DdosProtectionPlan, scope)
}

func (n networkDdosProtectionPlanWrapper) azureDdosProtectionPlanToSDPItem(plan *armnetwork.DdosProtectionPlan, scope string) (*sdp.Item, *sdp.QueryError) {
	if plan.Name == nil {
		return nil, azureshared.QueryError(errors.New("DDoS protection plan name is nil"), scope, n.Type())
	}

	attributes, err := shared.ToAttributesWithExclude(plan, "tags")
	if err != nil {
		return nil, azureshared.QueryError(err, scope, n.Type())
	}

	sdpItem := &sdp.Item{
		Type:              azureshared.NetworkDdosProtectionPlan.String(),
		UniqueAttribute:   "name",
		Attributes:        attributes,
		Scope:             scope,
		Tags:              azureshared.ConvertAzureTags(plan.Tags),
		LinkedItemQueries: []*sdp.LinkedItemQuery{},
	}

	if plan.Properties != nil {
		// Link to each associated virtual network
		for _, ref := range plan.Properties.VirtualNetworks {
			if ref != nil && ref.ID != nil {
				vnetID := *ref.ID
				vnetName := azureshared.ExtractResourceName(vnetID)
				if vnetName != "" {
					linkedScope := azureshared.ExtractScopeFromResourceID(vnetID)
					if linkedScope == "" {
						linkedScope = scope
					}
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.NetworkVirtualNetwork.String(),
							Method: sdp.QueryMethod_GET,
							Query:  vnetName,
							Scope:  linkedScope,
						},
					})
				}
			}
		}
		// Link to each associated public IP address
		for _, ref := range plan.Properties.PublicIPAddresses {
			if ref != nil && ref.ID != nil {
				publicIPID := *ref.ID
				publicIPName := azureshared.ExtractResourceName(publicIPID)
				if publicIPName != "" {
					linkedScope := azureshared.ExtractScopeFromResourceID(publicIPID)
					if linkedScope == "" {
						linkedScope = scope
					}
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.NetworkPublicIPAddress.String(),
							Method: sdp.QueryMethod_GET,
							Query:  publicIPName,
							Scope:  linkedScope,
						},
					})
				}
			}
		}
	}

	// Health from provisioning state
	if plan.Properties != nil && plan.Properties.ProvisioningState != nil {
		switch *plan.Properties.ProvisioningState {
		case armnetwork.ProvisioningStateSucceeded:
			sdpItem.Health = sdp.Health_HEALTH_OK.Enum()
		case armnetwork.ProvisioningStateCreating, armnetwork.ProvisioningStateUpdating, armnetwork.ProvisioningStateDeleting:
			sdpItem.Health = sdp.Health_HEALTH_PENDING.Enum()
		case armnetwork.ProvisioningStateFailed, armnetwork.ProvisioningStateCanceled:
			sdpItem.Health = sdp.Health_HEALTH_ERROR.Enum()
		default:
			sdpItem.Health = sdp.Health_HEALTH_UNKNOWN.Enum()
		}
	}

	return sdpItem, nil
}

func (n networkDdosProtectionPlanWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		NetworkDdosProtectionPlanLookupByName,
	}
}

func (n networkDdosProtectionPlanWrapper) PotentialLinks() map[shared.ItemType]bool {
	return map[shared.ItemType]bool{
		azureshared.NetworkVirtualNetwork:  true,
		azureshared.NetworkPublicIPAddress: true,
	}
}

func (n networkDdosProtectionPlanWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "azurerm_network_ddos_protection_plan.name",
		},
	}
}

// https://learn.microsoft.com/en-us/azure/role-based-access-control/resource-provider-operations#microsoftnetwork
func (n networkDdosProtectionPlanWrapper) IAMPermissions() []string {
	return []string{
		"Microsoft.Network/ddosProtectionPlans/read",
	}
}

func (n networkDdosProtectionPlanWrapper) PredefinedRole() string {
	return "Reader"
}
