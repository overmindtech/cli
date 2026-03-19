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

var NetworkNatGatewayLookupByName = shared.NewItemTypeLookup("name", azureshared.NetworkNatGateway)

type networkNatGatewayWrapper struct {
	client clients.NatGatewaysClient

	*azureshared.MultiResourceGroupBase
}

// NewNetworkNatGateway creates a new networkNatGatewayWrapper instance.
func NewNetworkNatGateway(client clients.NatGatewaysClient, resourceGroupScopes []azureshared.ResourceGroupScope) sources.ListableWrapper {
	return &networkNatGatewayWrapper{
		client: client,
		MultiResourceGroupBase: azureshared.NewMultiResourceGroupBase(
			resourceGroupScopes,
			sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
			azureshared.NetworkNatGateway,
		),
	}
}

func (n networkNatGatewayWrapper) List(ctx context.Context, scope string) ([]*sdp.Item, *sdp.QueryError) {
	rgScope, err := n.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, n.Type())
	}
	pager := n.client.NewListPager(rgScope.ResourceGroup, nil)

	var items []*sdp.Item
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, azureshared.QueryError(err, scope, n.Type())
		}

		for _, ng := range page.Value {
			if ng.Name == nil {
				continue
			}
			item, sdpErr := n.azureNatGatewayToSDPItem(ng, scope)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}

	return items, nil
}

func (n networkNatGatewayWrapper) ListStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string) {
	rgScope, err := n.ResourceGroupScopeFromScope(scope)
	if err != nil {
		stream.SendError(azureshared.QueryError(err, scope, n.Type()))
		return
	}
	pager := n.client.NewListPager(rgScope.ResourceGroup, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			stream.SendError(azureshared.QueryError(err, scope, n.Type()))
			return
		}

		for _, ng := range page.Value {
			if ng.Name == nil {
				continue
			}
			item, sdpErr := n.azureNatGatewayToSDPItem(ng, scope)
			if sdpErr != nil {
				stream.SendError(sdpErr)
				continue
			}
			cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
			stream.SendItem(item)
		}
	}
}

func (n networkNatGatewayWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 1 {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "Get requires 1 query part: natGatewayName",
			Scope:       scope,
			ItemType:    n.Type(),
		}
	}

	natGatewayName := queryParts[0]

	rgScope, err := n.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, n.Type())
	}
	resp, err := n.client.Get(ctx, rgScope.ResourceGroup, natGatewayName, nil)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, n.Type())
	}

	return n.azureNatGatewayToSDPItem(&resp.NatGateway, scope)
}

func (n networkNatGatewayWrapper) azureNatGatewayToSDPItem(ng *armnetwork.NatGateway, scope string) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(ng, "tags")
	if err != nil {
		return nil, azureshared.QueryError(err, scope, n.Type())
	}

	if ng.Name == nil {
		return nil, azureshared.QueryError(errors.New("nat gateway name is nil"), scope, n.Type())
	}

	sdpItem := &sdp.Item{
		Type:              azureshared.NetworkNatGateway.String(),
		UniqueAttribute:   "name",
		Attributes:        attributes,
		Scope:             scope,
		Tags:              azureshared.ConvertAzureTags(ng.Tags),
		LinkedItemQueries: []*sdp.LinkedItemQuery{},
	}

	// Health from provisioning state
	if ng.Properties != nil && ng.Properties.ProvisioningState != nil {
		switch *ng.Properties.ProvisioningState {
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

	// Linked resources from Properties
	if ng.Properties == nil {
		return sdpItem, nil
	}
	props := ng.Properties

	// Public IP addresses (V4 and V6)
	for _, refs := range [][]*armnetwork.SubResource{props.PublicIPAddresses, props.PublicIPAddressesV6} {
		for _, ref := range refs {
			if ref != nil && ref.ID != nil {
				refID := *ref.ID
				refName := azureshared.ExtractResourceName(refID)
				if refName != "" {
					linkedScope := azureshared.ExtractScopeFromResourceID(refID)
					if linkedScope == "" {
						linkedScope = scope
					}
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.NetworkPublicIPAddress.String(),
							Method: sdp.QueryMethod_GET,
							Query:  refName,
							Scope:  linkedScope,
						},
					})
				}
			}
		}
	}

	// Public IP prefixes (V4 and V6)
	for _, refs := range [][]*armnetwork.SubResource{props.PublicIPPrefixes, props.PublicIPPrefixesV6} {
		for _, ref := range refs {
			if ref != nil && ref.ID != nil {
				refID := *ref.ID
				refName := azureshared.ExtractResourceName(refID)
				if refName != "" {
					linkedScope := azureshared.ExtractScopeFromResourceID(refID)
					if linkedScope == "" {
						linkedScope = scope
					}
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.NetworkPublicIPPrefix.String(),
							Method: sdp.QueryMethod_GET,
							Query:  refName,
							Scope:  linkedScope,
						},
					})
				}
			}
		}
	}

	// Subnets (read-only references: subnets using this NAT gateway)
	for _, ref := range props.Subnets {
		if ref != nil && ref.ID != nil {
			subnetID := *ref.ID
			params := azureshared.ExtractPathParamsFromResourceID(subnetID, []string{"virtualNetworks", "subnets"})
			if len(params) >= 2 && params[0] != "" && params[1] != "" {
				linkedScope := azureshared.ExtractScopeFromResourceID(subnetID)
				if linkedScope == "" {
					linkedScope = scope
				}
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   azureshared.NetworkSubnet.String(),
						Method: sdp.QueryMethod_GET,
						Scope:  linkedScope,
						Query:  shared.CompositeLookupKey(params[0], params[1]),
					},
				})
			}
		}
	}

	// Source virtual network
	if props.SourceVirtualNetwork != nil && props.SourceVirtualNetwork.ID != nil {
		vnetID := *props.SourceVirtualNetwork.ID
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

	return sdpItem, nil
}

func (n networkNatGatewayWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		NetworkNatGatewayLookupByName,
	}
}

func (n networkNatGatewayWrapper) PotentialLinks() map[shared.ItemType]bool {
	return map[shared.ItemType]bool{
		azureshared.NetworkPublicIPAddress: true,
		azureshared.NetworkPublicIPPrefix:  true,
		azureshared.NetworkSubnet:          true,
		azureshared.NetworkVirtualNetwork:  true,
	}
}

func (n networkNatGatewayWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "azurerm_nat_gateway.name",
		},
	}
}

func (n networkNatGatewayWrapper) IAMPermissions() []string {
	return []string{
		"Microsoft.Network/natGateways/read",
	}
}

func (n networkNatGatewayWrapper) PredefinedRole() string {
	return "Reader"
}
