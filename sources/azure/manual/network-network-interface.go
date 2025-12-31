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

var NetworkNetworkInterfaceLookupByName = shared.NewItemTypeLookup("name", azureshared.NetworkNetworkInterface)

type networkNetworkInterfaceWrapper struct {
	client clients.NetworkInterfacesClient

	*azureshared.ResourceGroupBase
}

func NewNetworkNetworkInterface(client clients.NetworkInterfacesClient, subscriptionID, resourceGroup string) sources.ListableWrapper {
	return &networkNetworkInterfaceWrapper{
		client: client,
		ResourceGroupBase: azureshared.NewResourceGroupBase(
			subscriptionID,
			resourceGroup,
			sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
			azureshared.NetworkNetworkInterface,
		),
	}
}

func (n networkNetworkInterfaceWrapper) List(ctx context.Context) ([]*sdp.Item, *sdp.QueryError) {
	pager := n.client.List(ctx, n.ResourceGroup())

	var items []*sdp.Item
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, azureshared.QueryError(err, n.DefaultScope(), n.Type())
		}

		for _, networkInterface := range page.Value {
			item, sdpErr := n.azureNetworkInterfaceToSDPItem(networkInterface)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}

	return items, nil
}

// reference: https://learn.microsoft.com/en-us/rest/api/virtualnetwork/network-interfaces/get?view=rest-virtualnetwork-2025-03-01&tabs=HTTP#response
func (n networkNetworkInterfaceWrapper) azureNetworkInterfaceToSDPItem(networkInterface *armnetwork.Interface) (*sdp.Item, *sdp.QueryError) {
	if networkInterface.Name == nil {
		return nil, azureshared.QueryError(errors.New("network interface name is nil"), n.DefaultScope(), n.Type())
	}

	attributes, err := shared.ToAttributesWithExclude(networkInterface, "tags")
	if err != nil {
		return nil, azureshared.QueryError(err, n.DefaultScope(), n.Type())
	}

	sdpItem := &sdp.Item{
		Type:            azureshared.NetworkNetworkInterface.String(),
		UniqueAttribute: "name",
		Attributes:      attributes,
		Scope:           n.DefaultScope(),
		Tags:            azureshared.ConvertAzureTags(networkInterface.Tags),
	}

	// Add IP configuration link (name is guaranteed to be non-nil due to validation above)
	sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   azureshared.NetworkNetworkInterfaceIPConfiguration.String(),
			Method: sdp.QueryMethod_SEARCH,
			Query:  *networkInterface.Name,
			Scope:  n.DefaultScope(),
		},
		BlastPropagation: &sdp.BlastPropagation{
			In:  false, // IP configuration changes don't affect the network interface itself
			Out: true,  // Network interface changes (especially deletion) affect IP configurations
		}, // IP configurations are child resources of the network interface
	})

	if networkInterface.Properties != nil && networkInterface.Properties.VirtualMachine != nil {
		if networkInterface.Properties.VirtualMachine.ID != nil {
			vmName := azureshared.ExtractResourceName(*networkInterface.Properties.VirtualMachine.ID)
			if vmName != "" {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   azureshared.ComputeVirtualMachine.String(),
						Method: sdp.QueryMethod_GET,
						Query:  vmName,
						Scope:  n.DefaultScope(),
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  false, // VM changes (like deletion) may detach the interface but don't delete it
						Out: true,  // Network interface changes/deletion directly affect VM network connectivity
					}, // Network interface provides connectivity to the VM; bidirectional operational dependency
				})
			}
		}
	}

	if networkInterface.Properties != nil && networkInterface.Properties.NetworkSecurityGroup != nil {
		if networkInterface.Properties.NetworkSecurityGroup.ID != nil {
			nsgName := azureshared.ExtractResourceName(*networkInterface.Properties.NetworkSecurityGroup.ID)
			if nsgName != "" {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   azureshared.NetworkNetworkSecurityGroup.String(),
						Method: sdp.QueryMethod_GET,
						Query:  nsgName,
						Scope:  n.DefaultScope(),
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true,  // NSG rule changes affect the network interface's security and traffic flow
						Out: false, // Network interface changes don't affect the NSG itself
					}, // NSG controls security rules applied to the network interface
				})
			}
		}
	}

	return sdpItem, nil
}

func (n networkNetworkInterfaceWrapper) Get(ctx context.Context, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) != 1 {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "query must be a network interface name",
			Scope:       n.DefaultScope(),
			ItemType:    n.Type(),
		}
	}
	networkInterfaceName := queryParts[0]

	networkInterface, err := n.client.Get(ctx, n.ResourceGroup(), networkInterfaceName)
	if err != nil {
		return nil, azureshared.QueryError(err, n.DefaultScope(), n.Type())
	}

	return n.azureNetworkInterfaceToSDPItem(&networkInterface.Interface)
}

func (n networkNetworkInterfaceWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		NetworkNetworkInterfaceLookupByName,
	}
}

func (n networkNetworkInterfaceWrapper) PotentialLinks() map[shared.ItemType]bool {
	return map[shared.ItemType]bool{
		azureshared.NetworkVirtualNetwork:                  true,
		azureshared.ComputeVirtualMachine:                  true,
		azureshared.NetworkNetworkSecurityGroup:            false, //TODO: Create adapter for network security groups
		azureshared.NetworkNetworkInterfaceIPConfiguration: false, //TODO: Create adapter for network interface IP configurations
	}
}

func (n networkNetworkInterfaceWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "azurerm_network_interface.name",
		},
	}
}

func (n networkNetworkInterfaceWrapper) IAMPermissions() []string {
	return []string{
		"Microsoft.Network/networkInterfaces/read",
	}
}

func (n networkNetworkInterfaceWrapper) PredefinedRole() string {
	return "Reader"
}
