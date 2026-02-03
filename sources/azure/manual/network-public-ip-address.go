package manual

import (
	"context"
	"errors"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v8"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

var NetworkPublicIPAddressLookupByName = shared.NewItemTypeLookup("name", azureshared.NetworkPublicIPAddress)

type networkPublicIPAddressWrapper struct {
	client clients.PublicIPAddressesClient

	*azureshared.ResourceGroupBase
}

func NewNetworkPublicIPAddress(client clients.PublicIPAddressesClient, subscriptionID, resourceGroup string) sources.ListableWrapper {
	return &networkPublicIPAddressWrapper{
		client: client,
		ResourceGroupBase: azureshared.NewResourceGroupBase(
			subscriptionID,
			resourceGroup,
			sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
			azureshared.NetworkPublicIPAddress,
		),
	}
}

// reference: https://learn.microsoft.com/en-us/rest/api/virtualnetwork/public-ip-addresses/list?view=rest-virtualnetwork-2025-03-01&tabs=HTTP
// GET https://management.azure.com/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/publicIPAddresses?api-version=2025-03-01
func (n networkPublicIPAddressWrapper) List(ctx context.Context, scope string) ([]*sdp.Item, *sdp.QueryError) {
	resourceGroup := azureshared.ResourceGroupFromScope(scope)
	if resourceGroup == "" {
		resourceGroup = n.ResourceGroup()
	}
	pager := n.client.List(ctx, resourceGroup)

	var items []*sdp.Item
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, azureshared.QueryError(err, scope, n.Type())
		}

		for _, publicIPAddress := range page.Value {
			if publicIPAddress.Name == nil {
				continue
			}

			item, sdpErr := n.azurePublicIPAddressToSDPItem(publicIPAddress, scope)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}

	return items, nil
}

// reference: https://learn.microsoft.com/en-us/rest/api/virtualnetwork/public-ip-addresses/get?view=rest-virtualnetwork-2025-03-01&tabs=HTTP
// GET https://management.azure.com/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/publicIPAddresses/{publicIpAddressName}?api-version=2025-03-01
func (n networkPublicIPAddressWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) != 1 {
		return nil, azureshared.QueryError(errors.New("query must be exactly one part and be a public IP address name"), scope, n.Type())
	}

	publicIPAddressName := queryParts[0]

	resourceGroup := azureshared.ResourceGroupFromScope(scope)
	if resourceGroup == "" {
		resourceGroup = n.ResourceGroup()
	}
	publicIPAddress, err := n.client.Get(ctx, resourceGroup, publicIPAddressName)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, n.Type())
	}

	return n.azurePublicIPAddressToSDPItem(&publicIPAddress.PublicIPAddress, scope)
}

func (n networkPublicIPAddressWrapper) azurePublicIPAddressToSDPItem(publicIPAddress *armnetwork.PublicIPAddress, scope string) (*sdp.Item, *sdp.QueryError) {
	if publicIPAddress.Name == nil {
		return nil, azureshared.QueryError(errors.New("public IP address name is nil"), scope, n.Type())
	}

	attributes, err := shared.ToAttributesWithExclude(publicIPAddress, "tags")
	if err != nil {
		return nil, azureshared.QueryError(err, scope, n.Type())
	}

	sdpItem := &sdp.Item{
		Type:            azureshared.NetworkPublicIPAddress.String(),
		UniqueAttribute: "name",
		Attributes:      attributes,
		Scope:           scope,
		Tags:            azureshared.ConvertAzureTags(publicIPAddress.Tags),
	}

	// Link to IP address (standard library) if IP address is assigned
	if publicIPAddress.Properties != nil && publicIPAddress.Properties.IPAddress != nil && *publicIPAddress.Properties.IPAddress != "" {
		sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   "ip",
				Method: sdp.QueryMethod_GET,
				Query:  *publicIPAddress.Properties.IPAddress,
				Scope:  "global",
			},
			BlastPropagation: &sdp.BlastPropagation{
				// IPs are always linked
				In:  true,
				Out: true,
			},
		})
	}

	// Link to DNS name (standard library) if FQDN is configured
	if publicIPAddress.Properties != nil && publicIPAddress.Properties.DNSSettings != nil && publicIPAddress.Properties.DNSSettings.Fqdn != nil && *publicIPAddress.Properties.DNSSettings.Fqdn != "" {
		sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   "dns",
				Method: sdp.QueryMethod_SEARCH,
				Query:  *publicIPAddress.Properties.DNSSettings.Fqdn,
				Scope:  "global",
			},
			BlastPropagation: &sdp.BlastPropagation{
				// DNS names are always linked
				In:  true,
				Out: true,
			},
		})
	}

	// Link to Network Interface if IPConfiguration references a network interface
	if publicIPAddress.Properties != nil && publicIPAddress.Properties.IPConfiguration != nil {
		if publicIPAddress.Properties.IPConfiguration.ID != nil {
			ipConfigID := *publicIPAddress.Properties.IPConfiguration.ID
			// Check if this IP configuration belongs to a network interface
			// Format: /subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Network/networkInterfaces/{nicName}/ipConfigurations/{ipConfigName}
			if strings.Contains(ipConfigID, "/networkInterfaces/") {
				nicName := azureshared.ExtractPathParamsFromResourceID(ipConfigID, []string{"networkInterfaces"})
				if len(nicName) > 0 && nicName[0] != "" {
					// Extract scope from the IP configuration ID (may be in different resource group)
					linkedScope := azureshared.ExtractScopeFromResourceID(ipConfigID)
					if linkedScope == "" {
						linkedScope = scope
					}
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.NetworkNetworkInterface.String(),
							Method: sdp.QueryMethod_GET,
							Query:  nicName[0],
							Scope:  linkedScope,
						},
						BlastPropagation: &sdp.BlastPropagation{
							In:  true,  // Network interface IP configuration changes affect the public IP address
							Out: false, // Public IP address changes don't affect the network interface itself
						}, // Public IP address is associated with the network interface's IP configuration
					})
				}
			}
		}
	}

	// Link to linked public IP address
	if publicIPAddress.Properties != nil && publicIPAddress.Properties.LinkedPublicIPAddress != nil {
		if publicIPAddress.Properties.LinkedPublicIPAddress.ID != nil {
			linkedIPID := *publicIPAddress.Properties.LinkedPublicIPAddress.ID
			linkedIPName := azureshared.ExtractResourceName(linkedIPID)
			if linkedIPName != "" {
				// Extract scope from the linked IP address ID (may be in different resource group)
				linkedScope := azureshared.ExtractScopeFromResourceID(linkedIPID)
				if linkedScope == "" {
					linkedScope = scope
				}
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   azureshared.NetworkPublicIPAddress.String(),
						Method: sdp.QueryMethod_GET,
						Query:  linkedIPName,
						Scope:  linkedScope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true, // Linked public IP address changes can affect this public IP address
						Out: true, // This public IP address changes can affect the linked public IP address
					}, // Linked public IP addresses are tightly coupled and affect each other
				})
			}
		}
	}

	// Link to service public IP address
	if publicIPAddress.Properties != nil && publicIPAddress.Properties.ServicePublicIPAddress != nil {
		if publicIPAddress.Properties.ServicePublicIPAddress.ID != nil {
			serviceIPID := *publicIPAddress.Properties.ServicePublicIPAddress.ID
			serviceIPName := azureshared.ExtractResourceName(serviceIPID)
			if serviceIPName != "" {
				// Extract scope from the service IP address ID (may be in different resource group)
				linkedScope := azureshared.ExtractScopeFromResourceID(serviceIPID)
				if linkedScope == "" {
					linkedScope = scope
				}
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   azureshared.NetworkPublicIPAddress.String(),
						Method: sdp.QueryMethod_GET,
						Query:  serviceIPName,
						Scope:  linkedScope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true,  // Service public IP address changes can affect this public IP address
						Out: false, // This public IP address changes don't affect the service public IP address
					}, // Service public IP address is the underlying resource for this public IP address
				})
			}
		}
	}

	// Link to public IP prefix
	if publicIPAddress.Properties != nil && publicIPAddress.Properties.PublicIPPrefix != nil {
		if publicIPAddress.Properties.PublicIPPrefix.ID != nil {
			prefixID := *publicIPAddress.Properties.PublicIPPrefix.ID
			prefixName := azureshared.ExtractResourceName(prefixID)
			if prefixName != "" {
				// Extract scope from the public IP prefix ID (may be in different resource group)
				linkedScope := azureshared.ExtractScopeFromResourceID(prefixID)
				if linkedScope == "" {
					linkedScope = scope
				}
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   azureshared.NetworkPublicIPPrefix.String(),
						Method: sdp.QueryMethod_GET,
						Query:  prefixName,
						Scope:  linkedScope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true,  // Public IP prefix changes can affect this public IP address
						Out: false, // This public IP address changes don't affect the public IP prefix
					}, // Public IP address is allocated from the public IP prefix
				})
			}
		}
	}

	// Link to NAT gateway
	if publicIPAddress.Properties != nil && publicIPAddress.Properties.NatGateway != nil {
		if publicIPAddress.Properties.NatGateway.ID != nil {
			natGatewayID := *publicIPAddress.Properties.NatGateway.ID
			natGatewayName := azureshared.ExtractResourceName(natGatewayID)
			if natGatewayName != "" {
				// Extract scope from the NAT gateway ID (may be in different resource group)
				linkedScope := azureshared.ExtractScopeFromResourceID(natGatewayID)
				if linkedScope == "" {
					linkedScope = scope
				}
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   azureshared.NetworkNatGateway.String(),
						Method: sdp.QueryMethod_GET,
						Query:  natGatewayName,
						Scope:  linkedScope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true,  // NAT gateway changes can affect this public IP address
						Out: false, // This public IP address changes don't affect the NAT gateway
					}, // Public IP address is associated with the NAT gateway for outbound connectivity
				})
			}
		}
	}

	// Link to DDoS protection plan
	if publicIPAddress.Properties != nil && publicIPAddress.Properties.DdosSettings != nil {
		if publicIPAddress.Properties.DdosSettings.DdosProtectionPlan != nil {
			if publicIPAddress.Properties.DdosSettings.DdosProtectionPlan.ID != nil {
				ddosPlanID := *publicIPAddress.Properties.DdosSettings.DdosProtectionPlan.ID
				ddosPlanName := azureshared.ExtractResourceName(ddosPlanID)
				if ddosPlanName != "" {
					// Extract scope from the DDoS protection plan ID (may be in different resource group)
					linkedScope := azureshared.ExtractScopeFromResourceID(ddosPlanID)
					if linkedScope == "" {
						linkedScope = scope
					}
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.NetworkDdosProtectionPlan.String(),
							Method: sdp.QueryMethod_GET,
							Query:  ddosPlanName,
							Scope:  linkedScope,
						},
						BlastPropagation: &sdp.BlastPropagation{
							In:  true,  // DDoS protection plan changes can affect this public IP address protection
							Out: false, // This public IP address changes don't affect the DDoS protection plan
						}, // Public IP address is protected by the DDoS protection plan
					})
				}
			}
		}
	}

	// Link to Load Balancer if IPConfiguration references a load balancer frontend IP configuration
	if publicIPAddress.Properties != nil && publicIPAddress.Properties.IPConfiguration != nil {
		if publicIPAddress.Properties.IPConfiguration.ID != nil {
			ipConfigID := *publicIPAddress.Properties.IPConfiguration.ID
			// Check if this IP configuration belongs to a load balancer
			// Format: /subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Network/loadBalancers/{lbName}/frontendIPConfigurations/{frontendIPConfigName}
			if strings.Contains(ipConfigID, "/loadBalancers/") {
				lbName := azureshared.ExtractPathParamsFromResourceID(ipConfigID, []string{"loadBalancers"})
				if len(lbName) > 0 && lbName[0] != "" {
					// Extract scope from the load balancer ID (may be in different resource group)
					linkedScope := azureshared.ExtractScopeFromResourceID(ipConfigID)
					if linkedScope == "" {
						linkedScope = scope
					}
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.NetworkLoadBalancer.String(),
							Method: sdp.QueryMethod_GET,
							Query:  lbName[0],
							Scope:  linkedScope,
						},
						BlastPropagation: &sdp.BlastPropagation{
							In:  true,  // Load balancer frontend IP configuration changes affect the public IP address
							Out: false, // Public IP address changes don't affect the load balancer itself
						}, // Public IP address is associated with the load balancer's frontend IP configuration
					})
				}
			}
		}
	}

	return sdpItem, nil
}

func (n networkPublicIPAddressWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		NetworkPublicIPAddressLookupByName,
	}
}

func (n networkPublicIPAddressWrapper) PotentialLinks() map[shared.ItemType]bool {
	return map[shared.ItemType]bool{
		azureshared.NetworkNetworkInterface:   true,
		azureshared.NetworkPublicIPAddress:    true,
		azureshared.NetworkPublicIPPrefix:     true,
		azureshared.NetworkNatGateway:         true,
		azureshared.NetworkDdosProtectionPlan: true,
		azureshared.NetworkLoadBalancer:       true,
		stdlib.NetworkIP:                      true,
		stdlib.NetworkDNS:                     true,
	}
}

func (n networkPublicIPAddressWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "azurerm_public_ip.name",
		},
	}
}

// https://learn.microsoft.com/en-us/azure/role-based-access-control/permissions/networking
func (n networkPublicIPAddressWrapper) IAMPermissions() []string {
	return []string{
		"Microsoft.Network/publicIPAddresses/read",
	}
}

func (n networkPublicIPAddressWrapper) PredefinedRole() string {
	return "Reader"
}
