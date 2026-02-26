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

var NetworkVirtualNetworkPeeringLookupByUniqueAttr = shared.NewItemTypeLookup("uniqueAttr", azureshared.NetworkVirtualNetworkPeering)

type networkVirtualNetworkPeeringWrapper struct {
	client clients.VirtualNetworkPeeringsClient

	*azureshared.MultiResourceGroupBase
}

// NewNetworkVirtualNetworkPeering creates a new networkVirtualNetworkPeeringWrapper instance (SearchableWrapper: child of virtual network).
func NewNetworkVirtualNetworkPeering(client clients.VirtualNetworkPeeringsClient, resourceGroupScopes []azureshared.ResourceGroupScope) sources.SearchableWrapper {
	return &networkVirtualNetworkPeeringWrapper{
		client: client,
		MultiResourceGroupBase: azureshared.NewMultiResourceGroupBase(
			resourceGroupScopes,
			sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
			azureshared.NetworkVirtualNetworkPeering,
		),
	}
}

func (n networkVirtualNetworkPeeringWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 2 {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "Get requires 2 query parts: virtualNetworkName and peeringName",
			Scope:       scope,
			ItemType:    n.Type(),
		}
	}
	virtualNetworkName := queryParts[0]
	peeringName := queryParts[1]
	if peeringName == "" {
		return nil, azureshared.QueryError(errors.New("peering name cannot be empty"), scope, n.Type())
	}

	rgScope, err := n.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, n.Type())
	}
	resp, err := n.client.Get(ctx, rgScope.ResourceGroup, virtualNetworkName, peeringName, nil)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, n.Type())
	}

	return n.azureVirtualNetworkPeeringToSDPItem(&resp.VirtualNetworkPeering, virtualNetworkName, peeringName, scope)
}

func (n networkVirtualNetworkPeeringWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		NetworkVirtualNetworkLookupByName,
		NetworkVirtualNetworkPeeringLookupByUniqueAttr,
	}
}

func (n networkVirtualNetworkPeeringWrapper) Search(ctx context.Context, scope string, queryParts ...string) ([]*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 1 {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "Search requires 1 query part: virtualNetworkName",
			Scope:       scope,
			ItemType:    n.Type(),
		}
	}
	virtualNetworkName := queryParts[0]

	rgScope, err := n.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, n.Type())
	}
	pager := n.client.NewListPager(rgScope.ResourceGroup, virtualNetworkName, nil)

	var items []*sdp.Item
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, azureshared.QueryError(err, scope, n.Type())
		}
		for _, peering := range page.Value {
			if peering == nil || peering.Name == nil {
				continue
			}
			item, sdpErr := n.azureVirtualNetworkPeeringToSDPItem(peering, virtualNetworkName, *peering.Name, scope)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}
	return items, nil
}

func (n networkVirtualNetworkPeeringWrapper) SearchStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string, queryParts ...string) {
	if len(queryParts) < 1 {
		stream.SendError(azureshared.QueryError(errors.New("Search requires 1 query part: virtualNetworkName"), scope, n.Type()))
		return
	}
	virtualNetworkName := queryParts[0]

	rgScope, err := n.ResourceGroupScopeFromScope(scope)
	if err != nil {
		stream.SendError(azureshared.QueryError(err, scope, n.Type()))
		return
	}
	pager := n.client.NewListPager(rgScope.ResourceGroup, virtualNetworkName, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			stream.SendError(azureshared.QueryError(err, scope, n.Type()))
			return
		}
		for _, peering := range page.Value {
			if peering == nil || peering.Name == nil {
				continue
			}
			item, sdpErr := n.azureVirtualNetworkPeeringToSDPItem(peering, virtualNetworkName, *peering.Name, scope)
			if sdpErr != nil {
				stream.SendError(sdpErr)
				continue
			}
			cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
			stream.SendItem(item)
		}
	}
}

func (n networkVirtualNetworkPeeringWrapper) SearchLookups() []sources.ItemTypeLookups {
	return []sources.ItemTypeLookups{
		{
			NetworkVirtualNetworkLookupByName,
		},
	}
}

func (n networkVirtualNetworkPeeringWrapper) azureVirtualNetworkPeeringToSDPItem(peering *armnetwork.VirtualNetworkPeering, virtualNetworkName, peeringName, scope string) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(peering, "tags")
	if err != nil {
		return nil, azureshared.QueryError(err, scope, n.Type())
	}

	err = attributes.Set("uniqueAttr", shared.CompositeLookupKey(virtualNetworkName, peeringName))
	if err != nil {
		return nil, azureshared.QueryError(err, scope, n.Type())
	}

	sdpItem := &sdp.Item{
		Type:            azureshared.NetworkVirtualNetworkPeering.String(),
		UniqueAttribute: "uniqueAttr",
		Attributes:      attributes,
		Scope:           scope,
	}

	// Health status from ProvisioningState
	if peering.Properties != nil && peering.Properties.ProvisioningState != nil {
		switch *peering.Properties.ProvisioningState {
		case armnetwork.ProvisioningStateSucceeded:
			sdpItem.Health = sdp.Health_HEALTH_OK.Enum()
		case armnetwork.ProvisioningStateCreating, armnetwork.ProvisioningStateUpdating, armnetwork.ProvisioningStateDeleting:
			sdpItem.Health = sdp.Health_HEALTH_PENDING.Enum()
		case armnetwork.ProvisioningStateFailed, armnetwork.ProvisioningStateCanceled:
			sdpItem.Health = sdp.Health_HEALTH_ERROR.Enum()
		}
	}

	// Link to parent (local) Virtual Network
	sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   azureshared.NetworkVirtualNetwork.String(),
			Method: sdp.QueryMethod_GET,
			Query:  virtualNetworkName,
			Scope:  scope,
		},
	})

	// Link to remote Virtual Network and remote subnets (selective peering)
	if peering.Properties != nil && peering.Properties.RemoteVirtualNetwork != nil && peering.Properties.RemoteVirtualNetwork.ID != nil {
		remoteVNetID := *peering.Properties.RemoteVirtualNetwork.ID
		remoteVNetName := azureshared.ExtractResourceName(remoteVNetID)
		if remoteVNetName != "" {
			linkedScope := scope
			if extractedScope := azureshared.ExtractScopeFromResourceID(remoteVNetID); extractedScope != "" {
				linkedScope = extractedScope
			}
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   azureshared.NetworkVirtualNetwork.String(),
					Method: sdp.QueryMethod_GET,
					Query:  remoteVNetName,
					Scope:  linkedScope,
				},
			})
			// Link to remote subnets (selective subnet peering)
			if peering.Properties.RemoteSubnetNames != nil {
				for _, name := range peering.Properties.RemoteSubnetNames {
					if name != nil && *name != "" {
						sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   azureshared.NetworkSubnet.String(),
								Method: sdp.QueryMethod_GET,
								Query:  shared.CompositeLookupKey(remoteVNetName, *name),
								Scope:  linkedScope,
							},
						})
					}
				}
			}
		}
	}

	// Link to local subnets (selective subnet peering)
	if peering.Properties != nil && peering.Properties.LocalSubnetNames != nil {
		for _, name := range peering.Properties.LocalSubnetNames {
			if name != nil && *name != "" {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   azureshared.NetworkSubnet.String(),
						Method: sdp.QueryMethod_GET,
						Query:  shared.CompositeLookupKey(virtualNetworkName, *name),
						Scope:  scope,
					},
				})
			}
		}
	}

	return sdpItem, nil
}

func (n networkVirtualNetworkPeeringWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		azureshared.NetworkVirtualNetwork,
		azureshared.NetworkSubnet,
	)
}

// ref: https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/virtual_network_peering
func (n networkVirtualNetworkPeeringWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_SEARCH,
			TerraformQueryMap: "azurerm_virtual_network_peering.id",
		},
	}
}

// ref: https://learn.microsoft.com/en-us/azure/role-based-access-control/permissions-reference#microsoftnetwork
func (n networkVirtualNetworkPeeringWrapper) IAMPermissions() []string {
	return []string{
		"Microsoft.Network/virtualNetworks/virtualNetworkPeerings/read",
	}
}

func (n networkVirtualNetworkPeeringWrapper) PredefinedRole() string {
	return "Reader"
}
