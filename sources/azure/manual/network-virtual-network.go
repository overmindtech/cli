package manual

import (
	"context"
	"errors"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/shared"
)

var NetworkVirtualNetworkLookupByName = shared.NewItemTypeLookup("name", azureshared.NetworkVirtualNetwork)

type networkVirtualNetworkWrapper struct {
	client clients.VirtualNetworksClient

	*azureshared.ResourceGroupBase
}

// NewNetworkVirtualNetwork creates a new networkVirtualNetworkWrapper instance
func NewNetworkVirtualNetwork(client clients.VirtualNetworksClient, subscriptionID, resourceGroup string) sources.ListableWrapper {
	return &networkVirtualNetworkWrapper{
		client: client,
		ResourceGroupBase: azureshared.NewResourceGroupBase(
			subscriptionID,
			resourceGroup,
			sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
			azureshared.NetworkVirtualNetwork,
		),
	}
}

func (n networkVirtualNetworkWrapper) List(ctx context.Context) ([]*sdp.Item, *sdp.QueryError) {
	pager := n.client.NewListPager(n.ResourceGroup(), nil)

	var items []*sdp.Item
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, azureshared.QueryError(err, n.DefaultScope(), n.Type())
		}

		for _, network := range page.Value {
			item, sdpErr := n.azureVirtualNetworkToSDPItem(network)
			if sdpErr != nil {
				return nil, sdpErr
			}

			items = append(items, item)
		}
	}

	return items, nil
}

func (n networkVirtualNetworkWrapper) Get(ctx context.Context, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 1 {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "Get requires 1 query part: virtualNetworkName",
			Scope:       n.DefaultScope(),
			ItemType:    n.Type(),
		}
	}

	virtualNetworkName := queryParts[0]

	resp, err := n.client.Get(ctx, n.ResourceGroup(), virtualNetworkName, nil)
	if err != nil {
		return nil, azureshared.QueryError(err, n.DefaultScope(), n.Type())
	}

	return n.azureVirtualNetworkToSDPItem(&resp.VirtualNetwork)
}

func (n networkVirtualNetworkWrapper) azureVirtualNetworkToSDPItem(network *armnetwork.VirtualNetwork) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(network)
	if err != nil {
		return nil, azureshared.QueryError(err, n.DefaultScope(), n.Type())
	}

	if network.Name == nil {
		return nil, azureshared.QueryError(errors.New("network name is nil"), n.DefaultScope(), n.Type())
	}

	sdpItem := &sdp.Item{
		Type:            azureshared.NetworkVirtualNetwork.String(),
		UniqueAttribute: "name",
		Attributes:      attributes,
		Scope:           n.DefaultScope(),
		Tags:            azureshared.ConvertAzureTags(network.Tags),
	}

	sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   azureshared.NetworkSubnet.String(),
			Method: sdp.QueryMethod_SEARCH,
			Scope:  n.DefaultScope(),
			Query:  *network.Name, // List subnets in the virtual network
		},
		BlastPropagation: &sdp.BlastPropagation{
			In:  false,
			Out: true,
		},
	})

	sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   azureshared.NetworkVirtualNetworkPeering.String(),
			Method: sdp.QueryMethod_SEARCH,
			Scope:  n.DefaultScope(),
			Query:  *network.Name, // List virtual network peerings in the virtual network
		},
		BlastPropagation: &sdp.BlastPropagation{
			In:  false, // Peering changes don't affect the Virtual Network itself
			Out: true,  // Virtual Network changes (especially deletion) affect peerings
		},
	})
	return sdpItem, nil
}

func (n networkVirtualNetworkWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		NetworkVirtualNetworkLookupByName,
	}
}

func (n networkVirtualNetworkWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		azureshared.NetworkSubnet,
		azureshared.NetworkVirtualNetworkPeering,
	)
}

func (n networkVirtualNetworkWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod: sdp.QueryMethod_GET,
			// https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/virtual_network
			TerraformQueryMap: "azurerm_virtual_network.name",
		},
	}
}

func (n networkVirtualNetworkWrapper) IAMPermissions() []string {
	return []string{
		"Microsoft.Network/virtualNetworks/read",
	}
}

func (n networkVirtualNetworkWrapper) PredefinedRole() string {
	return "Reader" //there is no predefined role for virtual networks, so we use the most restrictive role (Reader)
}
