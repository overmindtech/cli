package manual

import (
	"context"
	"errors"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v8"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/shared"
	"github.com/overmindtech/cli/sources/stdlib"
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

func (n networkVirtualNetworkWrapper) List(ctx context.Context, scope string) ([]*sdp.Item, *sdp.QueryError) {
	resourceGroup := azureshared.ResourceGroupFromScope(scope)
	if resourceGroup == "" {
		resourceGroup = n.ResourceGroup()
	}
	pager := n.client.NewListPager(resourceGroup, nil)

	var items []*sdp.Item
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, azureshared.QueryError(err, scope, n.Type())
		}

		for _, network := range page.Value {
			item, sdpErr := n.azureVirtualNetworkToSDPItem(network, scope)
			if sdpErr != nil {
				return nil, sdpErr
			}

			items = append(items, item)
		}
	}

	return items, nil
}

func (n networkVirtualNetworkWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 1 {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "Get requires 1 query part: virtualNetworkName",
			Scope:       scope,
			ItemType:    n.Type(),
		}
	}

	virtualNetworkName := queryParts[0]

	resourceGroup := azureshared.ResourceGroupFromScope(scope)
	if resourceGroup == "" {
		resourceGroup = n.ResourceGroup()
	}
	resp, err := n.client.Get(ctx, resourceGroup, virtualNetworkName, nil)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, n.Type())
	}

	return n.azureVirtualNetworkToSDPItem(&resp.VirtualNetwork, scope)
}

func (n networkVirtualNetworkWrapper) azureVirtualNetworkToSDPItem(network *armnetwork.VirtualNetwork, scope string) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(network)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, n.Type())
	}

	if network.Name == nil {
		return nil, azureshared.QueryError(errors.New("network name is nil"), scope, n.Type())
	}

	sdpItem := &sdp.Item{
		Type:            azureshared.NetworkVirtualNetwork.String(),
		UniqueAttribute: "name",
		Attributes:      attributes,
		Scope:           scope,
		Tags:            azureshared.ConvertAzureTags(network.Tags),
	}

	sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   azureshared.NetworkSubnet.String(),
			Method: sdp.QueryMethod_SEARCH,
			Scope:  scope,
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
			Scope:  scope,
			Query:  *network.Name, // List virtual network peerings in the virtual network
		},
		BlastPropagation: &sdp.BlastPropagation{
			In:  false, // Peering changes don't affect the Virtual Network itself
			Out: true,  // Virtual Network changes (especially deletion) affect peerings
		},
	})

	// Link to DDoS protection plan
	// Reference: https://learn.microsoft.com/en-us/rest/api/virtualnetwork/ddos-protection-plans/get
	if network.Properties != nil && network.Properties.DdosProtectionPlan != nil && network.Properties.DdosProtectionPlan.ID != nil {
		ddosPlanID := *network.Properties.DdosProtectionPlan.ID
		ddosPlanName := azureshared.ExtractResourceName(ddosPlanID)
		if ddosPlanName != "" {
			scope := n.DefaultScope()
			// Check if DDoS protection plan is in a different resource group
			if extractedScope := azureshared.ExtractScopeFromResourceID(ddosPlanID); extractedScope != "" {
				scope = extractedScope
			}
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   azureshared.NetworkDdosProtectionPlan.String(),
					Method: sdp.QueryMethod_GET,
					Query:  ddosPlanName,
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,  // If DDoS protection plan changes → Virtual Network protection affected (In: true)
					Out: false, // If Virtual Network is deleted → DDoS protection plan remains (Out: false)
				},
			})
		}
	}

	// Link to resources from subnets
	if network.Properties != nil && network.Properties.Subnets != nil {
		for _, subnet := range network.Properties.Subnets {
			if subnet == nil || subnet.Properties == nil {
				continue
			}

			// Link to Network Security Group from subnet
			// Reference: https://learn.microsoft.com/en-us/rest/api/virtualnetwork/network-security-groups/get
			if subnet.Properties.NetworkSecurityGroup != nil && subnet.Properties.NetworkSecurityGroup.ID != nil {
				nsgID := *subnet.Properties.NetworkSecurityGroup.ID
				nsgName := azureshared.ExtractResourceName(nsgID)
				if nsgName != "" {
					scope := n.DefaultScope()
					// Check if NSG is in a different resource group
					if extractedScope := azureshared.ExtractScopeFromResourceID(nsgID); extractedScope != "" {
						scope = extractedScope
					}
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.NetworkNetworkSecurityGroup.String(),
							Method: sdp.QueryMethod_GET,
							Query:  nsgName,
							Scope:  scope,
						},
						BlastPropagation: &sdp.BlastPropagation{
							In:  true,  // If NSG changes → Subnet security rules affected (In: true)
							Out: false, // If Virtual Network is deleted → NSG remains (Out: false)
						},
					})
				}
			}

			// Link to Route Table from subnet
			// Reference: https://learn.microsoft.com/en-us/rest/api/virtualnetwork/route-tables/get
			if subnet.Properties.RouteTable != nil && subnet.Properties.RouteTable.ID != nil {
				routeTableID := *subnet.Properties.RouteTable.ID
				routeTableName := azureshared.ExtractResourceName(routeTableID)
				if routeTableName != "" {
					scope := n.DefaultScope()
					// Check if Route Table is in a different resource group
					if extractedScope := azureshared.ExtractScopeFromResourceID(routeTableID); extractedScope != "" {
						scope = extractedScope
					}
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.NetworkRouteTable.String(),
							Method: sdp.QueryMethod_GET,
							Query:  routeTableName,
							Scope:  scope,
						},
						BlastPropagation: &sdp.BlastPropagation{
							In:  true,  // If Route Table changes → Subnet routing affected (In: true)
							Out: false, // If Virtual Network is deleted → Route Table remains (Out: false)
						},
					})
				}
			}

			// Link to NAT Gateway from subnet
			// Reference: https://learn.microsoft.com/en-us/rest/api/virtualnetwork/nat-gateways/get
			if subnet.Properties.NatGateway != nil && subnet.Properties.NatGateway.ID != nil {
				natGatewayID := *subnet.Properties.NatGateway.ID
				natGatewayName := azureshared.ExtractResourceName(natGatewayID)
				if natGatewayName != "" {
					scope := n.DefaultScope()
					// Check if NAT Gateway is in a different resource group
					if extractedScope := azureshared.ExtractScopeFromResourceID(natGatewayID); extractedScope != "" {
						scope = extractedScope
					}
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.NetworkNatGateway.String(),
							Method: sdp.QueryMethod_GET,
							Query:  natGatewayName,
							Scope:  scope,
						},
						BlastPropagation: &sdp.BlastPropagation{
							In:  true,  // If NAT Gateway changes → Subnet outbound connectivity affected (In: true)
							Out: false, // If Virtual Network is deleted → NAT Gateway remains (Out: false)
						},
					})
				}
			}

			// Link to Private Endpoints from subnet (read-only references)
			// Reference: https://learn.microsoft.com/en-us/rest/api/virtualnetwork/private-endpoints/get
			if subnet.Properties.PrivateEndpoints != nil {
				for _, privateEndpoint := range subnet.Properties.PrivateEndpoints {
					if privateEndpoint != nil && privateEndpoint.ID != nil {
						privateEndpointID := *privateEndpoint.ID
						privateEndpointName := azureshared.ExtractResourceName(privateEndpointID)
						if privateEndpointName != "" {
							scope := n.DefaultScope()
							// Check if Private Endpoint is in a different resource group
							if extractedScope := azureshared.ExtractScopeFromResourceID(privateEndpointID); extractedScope != "" {
								scope = extractedScope
							}
							sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
								Query: &sdp.Query{
									Type:   azureshared.NetworkPrivateEndpoint.String(),
									Method: sdp.QueryMethod_GET,
									Query:  privateEndpointName,
									Scope:  scope,
								},
								BlastPropagation: &sdp.BlastPropagation{
									In:  true,  // If Private Endpoint changes → Subnet connectivity affected (In: true)
									Out: false, // If Virtual Network is deleted → Private Endpoint may become invalid (Out: false, but could be true)
								},
							})
						}
					}
				}
			}
		}
	}

	// Link to remote Virtual Networks from peerings
	// Reference: https://learn.microsoft.com/en-us/rest/api/virtualnetwork/virtual-networks/get
	if network.Properties != nil && network.Properties.VirtualNetworkPeerings != nil {
		for _, peering := range network.Properties.VirtualNetworkPeerings {
			if peering != nil && peering.Properties != nil && peering.Properties.RemoteVirtualNetwork != nil && peering.Properties.RemoteVirtualNetwork.ID != nil {
				remoteVNetID := *peering.Properties.RemoteVirtualNetwork.ID
				remoteVNetName := azureshared.ExtractResourceName(remoteVNetID)
				if remoteVNetName != "" {
					scope := n.DefaultScope()
					// Check if remote Virtual Network is in a different resource group or subscription
					if extractedScope := azureshared.ExtractScopeFromResourceID(remoteVNetID); extractedScope != "" {
						scope = extractedScope
					}
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.NetworkVirtualNetwork.String(),
							Method: sdp.QueryMethod_GET,
							Query:  remoteVNetName,
							Scope:  scope,
						},
						BlastPropagation: &sdp.BlastPropagation{
							In:  true, // If remote VNet changes → Peering connectivity affected (In: true)
							Out: true, // If this VNet changes → Remote VNet peering affected (Out: true)
						},
					})
				}
			}
		}
	}

	// Link to default public NAT Gateway (VNet-level)
	// Reference: https://learn.microsoft.com/en-us/rest/api/virtualnetwork/nat-gateways/get
	if network.Properties != nil && network.Properties.DefaultPublicNatGateway != nil && network.Properties.DefaultPublicNatGateway.ID != nil {
		natGatewayID := *network.Properties.DefaultPublicNatGateway.ID
		natGatewayName := azureshared.ExtractResourceName(natGatewayID)
		if natGatewayName != "" {
			scope := n.DefaultScope()
			if extractedScope := azureshared.ExtractScopeFromResourceID(natGatewayID); extractedScope != "" {
				scope = extractedScope
			}
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   azureshared.NetworkNatGateway.String(),
					Method: sdp.QueryMethod_GET,
					Query:  natGatewayName,
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,  // If NAT Gateway changes → VNet outbound connectivity affected (In: true)
					Out: false, // If Virtual Network is deleted → NAT Gateway remains (Out: false)
				},
			})
		}
	}

	// Link DHCP DNS servers to stdlib ip (IP addresses) or stdlib dns (hostnames)
	// Reference: DhcpOptions contains DNS servers available to VMs in the VNet
	if network.Properties != nil && network.Properties.DhcpOptions != nil && network.Properties.DhcpOptions.DNSServers != nil {
		for _, dnsServerPtr := range network.Properties.DhcpOptions.DNSServers {
			if dnsServerPtr == nil {
				continue
			}
			appendDNSServerLinkIfValid(&sdpItem.LinkedItemQueries, *dnsServerPtr, "AzureProvidedDNS")
		}
	}

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
		azureshared.NetworkDdosProtectionPlan,
		azureshared.NetworkNatGateway,
		azureshared.NetworkNetworkSecurityGroup,
		azureshared.NetworkRouteTable,
		azureshared.NetworkPrivateEndpoint,
		azureshared.NetworkVirtualNetwork,
		stdlib.NetworkIP,
		stdlib.NetworkDNS,
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
	return "Reader" // there is no predefined role for virtual networks, so we use the most restrictive role (Reader)
}
