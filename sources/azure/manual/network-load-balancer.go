package manual

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

var NetworkLoadBalancerLookupByName = shared.NewItemTypeLookup("name", azureshared.NetworkLoadBalancer)

type networkLoadBalancerWrapper struct {
	client clients.LoadBalancersClient

	*azureshared.ResourceGroupBase
}

func NewNetworkLoadBalancer(client clients.LoadBalancersClient, subscriptionID, resourceGroup string) sources.ListableWrapper {
	return &networkLoadBalancerWrapper{
		client: client,
		ResourceGroupBase: azureshared.NewResourceGroupBase(
			subscriptionID,
			resourceGroup,
			sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
			azureshared.NetworkLoadBalancer,
		),
	}
}

func (n networkLoadBalancerWrapper) List(ctx context.Context, scope string) ([]*sdp.Item, *sdp.QueryError) {
	resourceGroup := azureshared.ResourceGroupFromScope(scope)
	if resourceGroup == "" {
		resourceGroup = n.ResourceGroup()
	}
	pager := n.client.List(resourceGroup)

	var items []*sdp.Item
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, azureshared.QueryError(err, scope, n.Type())
		}

		for _, loadBalancer := range page.Value {
			if loadBalancer.Name == nil {
				continue
			}

			item, sdpErr := n.azureLoadBalancerToSDPItem(loadBalancer, scope)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}

	return items, nil
}

func (n networkLoadBalancerWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) != 1 {
		return nil, azureshared.QueryError(errors.New("query must be a load balancer name"), scope, n.Type())
	}

	loadBalancerName := queryParts[0]
	if loadBalancerName == "" {
		return nil, azureshared.QueryError(errors.New("load balancer name cannot be empty"), scope, n.Type())
	}

	resourceGroup := azureshared.ResourceGroupFromScope(scope)
	if resourceGroup == "" {
		resourceGroup = n.ResourceGroup()
	}
	resp, err := n.client.Get(ctx, resourceGroup, loadBalancerName)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, n.Type())
	}

	item, sdpErr := n.azureLoadBalancerToSDPItem(&resp.LoadBalancer, scope)
	if sdpErr != nil {
		return nil, sdpErr
	}

	return item, nil
}

func (n networkLoadBalancerWrapper) azureLoadBalancerToSDPItem(loadBalancer *armnetwork.LoadBalancer, scope string) (*sdp.Item, *sdp.QueryError) {
	if loadBalancer.Name == nil {
		return nil, azureshared.QueryError(errors.New("load balancer name is nil"), scope, n.Type())
	}

	attributes, err := shared.ToAttributesWithExclude(loadBalancer, "tags")
	if err != nil {
		return nil, azureshared.QueryError(err, scope, n.Type())
	}

	sdpItem := &sdp.Item{
		Type:            azureshared.NetworkLoadBalancer.String(),
		UniqueAttribute: "name",
		Attributes:      attributes,
		Scope:           scope,
		Tags:            azureshared.ConvertAzureTags(loadBalancer.Tags),
	}

	loadBalancerName := *loadBalancer.Name

	// Process FrontendIPConfigurations (Child Resource)
	// Reference: https://learn.microsoft.com/en-us/rest/api/load-balancer/load-balancer-frontend-ip-configurations/get?view=rest-load-balancer-2025-03-01&tabs=HTTP
	if loadBalancer.Properties != nil && loadBalancer.Properties.FrontendIPConfigurations != nil {
		for _, frontendIPConfig := range loadBalancer.Properties.FrontendIPConfigurations {
			if frontendIPConfig.Name != nil {
				// Link to FrontendIPConfiguration child resource
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   azureshared.NetworkLoadBalancerFrontendIPConfiguration.String(),
						Method: sdp.QueryMethod_GET,
						Query:  shared.CompositeLookupKey(loadBalancerName, *frontendIPConfig.Name),
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true, // FrontendIPConfiguration changes affect the load balancer's frontend configuration
						Out: true, // Load balancer changes (like deletion) affect the frontend IP configuration
					}, // FrontendIPConfiguration is a child resource of the Load Balancer; bidirectional dependency
				})
			}

			if frontendIPConfig.Properties != nil {
				// Link to Public IP Address if referenced
				// Reference: https://learn.microsoft.com/en-us/rest/api/virtualnetwork/public-ip-addresses/get?view=rest-virtualnetwork-2025-03-01&tabs=HTTP
				if frontendIPConfig.Properties.PublicIPAddress != nil && frontendIPConfig.Properties.PublicIPAddress.ID != nil {
					publicIPName := azureshared.ExtractResourceName(*frontendIPConfig.Properties.PublicIPAddress.ID)
					if publicIPName != "" {
						// Extract subscription ID and resource group from the resource ID to determine scope
						resourceID := *frontendIPConfig.Properties.PublicIPAddress.ID
						parts := strings.Split(strings.Trim(resourceID, "/"), "/")
						linkedScope := scope
						if len(parts) >= 4 && parts[0] == "subscriptions" && parts[2] == "resourceGroups" {
							subscriptionID := parts[1]
							resourceGroup := parts[3]
							linkedScope = fmt.Sprintf("%s.%s", subscriptionID, resourceGroup)
						}

						sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   azureshared.NetworkPublicIPAddress.String(),
								Method: sdp.QueryMethod_GET,
								Query:  publicIPName,
								Scope:  linkedScope,
							},
							BlastPropagation: &sdp.BlastPropagation{
								In:  true,  // Public IP changes (like deletion or reassignment) affect the load balancer's frontend
								Out: false, // Load balancer changes don't affect the public IP address itself
							}, // Public IP provides the frontend IP for the load balancer
						})
					}
				}

				// Link to Subnet if referenced
				// Reference: https://learn.microsoft.com/en-us/rest/api/virtualnetwork/subnets/get?view=rest-virtualnetwork-2025-03-01&tabs=HTTP
				if frontendIPConfig.Properties.Subnet != nil && frontendIPConfig.Properties.Subnet.ID != nil {
					subnetID := *frontendIPConfig.Properties.Subnet.ID
					// Subnet ID format: /subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Network/virtualNetworks/{vnet}/subnets/{subnet}
					parts := strings.Split(strings.Trim(subnetID, "/"), "/")
					if len(parts) >= 10 && parts[0] == "subscriptions" && parts[2] == "resourceGroups" && parts[4] == "providers" && parts[5] == "Microsoft.Network" && parts[6] == "virtualNetworks" && parts[8] == "subnets" {
						subscriptionID := parts[1]
						resourceGroup := parts[3]
						vnetName := parts[7]
						subnetName := parts[9]
						scope := fmt.Sprintf("%s.%s", subscriptionID, resourceGroup)

						sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   azureshared.NetworkSubnet.String(),
								Method: sdp.QueryMethod_GET, // Field is an ID, so use GET with composite lookup key
								Query:  shared.CompositeLookupKey(vnetName, subnetName),
								Scope:  scope,
							},
							BlastPropagation: &sdp.BlastPropagation{
								In:  true,  // Subnet changes (like address space modifications) affect the load balancer's network configuration
								Out: false, // Load balancer changes don't affect the subnet itself
							}, // Subnet provides the network location for the load balancer's frontend
						})
					}
				}

				// Link to IP address (standard library) if private IP address is assigned
				if frontendIPConfig.Properties.PrivateIPAddress != nil && *frontendIPConfig.Properties.PrivateIPAddress != "" {
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   "ip",
							Method: sdp.QueryMethod_GET,
							Query:  *frontendIPConfig.Properties.PrivateIPAddress,
							Scope:  "global",
						},
						BlastPropagation: &sdp.BlastPropagation{
							// IPs are always linked
							In:  true,
							Out: true,
						},
					})
				}
			}
		}
	}

	// Process BackendAddressPools (Child Resource)
	// Reference: https://learn.microsoft.com/en-us/rest/api/load-balancer/backend-address-pools/get
	if loadBalancer.Properties != nil && loadBalancer.Properties.BackendAddressPools != nil {
		for _, backendPool := range loadBalancer.Properties.BackendAddressPools {
			if backendPool.Name != nil {
				// Link to BackendAddressPool child resource
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   azureshared.NetworkLoadBalancerBackendAddressPool.String(),
						Method: sdp.QueryMethod_GET,
						Query:  shared.CompositeLookupKey(loadBalancerName, *backendPool.Name),
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true, // BackendAddressPool changes affect which backends receive traffic
						Out: true, // Load balancer changes (like deletion) affect the backend address pool
					}, // BackendAddressPool is a child resource of the Load Balancer; bidirectional dependency
				})
			}
		}
	}

	// Process InboundNatRules (Child Resource)
	// Reference: https://learn.microsoft.com/en-us/rest/api/load-balancer/inbound-nat-rules/get?view=rest-load-balancer-2025-03-01&tabs=HTTP
	if loadBalancer.Properties != nil && loadBalancer.Properties.InboundNatRules != nil {
		for _, natRule := range loadBalancer.Properties.InboundNatRules {
			if natRule.Name != nil {
				// Link to InboundNatRule child resource
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   azureshared.NetworkLoadBalancerInboundNatRule.String(),
						Method: sdp.QueryMethod_GET,
						Query:  shared.CompositeLookupKey(loadBalancerName, *natRule.Name),
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true, // InboundNatRule changes affect the load balancer's NAT configuration
						Out: true, // Load balancer changes (like deletion) affect the NAT rules
					}, // InboundNatRule is a child resource of the Load Balancer; bidirectional dependency
				})
			}

			// Link to Network Interface via BackendIPConfiguration
			if natRule.Properties != nil && natRule.Properties.BackendIPConfiguration != nil && natRule.Properties.BackendIPConfiguration.ID != nil {
				// BackendIPConfiguration.ID points to a Network Interface IP Configuration
				// Format: /subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Network/networkInterfaces/{nic}/ipConfigurations/{ipConfig}
				backendIPConfigID := *natRule.Properties.BackendIPConfiguration.ID
				parts := strings.Split(strings.Trim(backendIPConfigID, "/"), "/")
				if len(parts) >= 8 && parts[0] == "subscriptions" && parts[2] == "resourceGroups" && parts[4] == "providers" && parts[5] == "Microsoft.Network" && parts[6] == "networkInterfaces" {
					subscriptionID := parts[1]
					resourceGroup := parts[3]
					nicName := parts[7]
					scope := fmt.Sprintf("%s.%s", subscriptionID, resourceGroup)

					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.NetworkNetworkInterface.String(),
							Method: sdp.QueryMethod_GET,
							Query:  nicName,
							Scope:  scope,
						},
						BlastPropagation: &sdp.BlastPropagation{
							In:  true, // Network interface changes affect NAT rule routing
							Out: true, // NAT rule changes affect which network interface receives inbound traffic
						}, // Inbound NAT rules map traffic to specific network interfaces; bidirectional operational dependency
					})
				}
			}
		}
	}

	// Process LoadBalancingRules (Child Resource)
	// Reference: https://learn.microsoft.com/en-us/rest/api/load-balancer/load-balancer-load-balancing-rules/get?view=rest-load-balancer-2025-03-01&tabs=HTTP
	if loadBalancer.Properties != nil && loadBalancer.Properties.LoadBalancingRules != nil {
		for _, lbRule := range loadBalancer.Properties.LoadBalancingRules {
			if lbRule.Name != nil {
				// Link to LoadBalancingRule child resource
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   azureshared.NetworkLoadBalancerLoadBalancingRule.String(),
						Method: sdp.QueryMethod_GET,
						Query:  shared.CompositeLookupKey(loadBalancerName, *lbRule.Name),
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true, // LoadBalancingRule changes affect how traffic is distributed
						Out: true, // Load balancer changes (like deletion) affect the load balancing rules
					}, // LoadBalancingRule is a child resource of the Load Balancer; bidirectional dependency
				})
			}
		}
	}

	// Process Probes (Child Resource)
	// Reference: https://learn.microsoft.com/en-us/rest/api/load-balancer/load-balancer-probes/get?view=rest-load-balancer-2025-03-01&tabs=HTTP
	if loadBalancer.Properties != nil && loadBalancer.Properties.Probes != nil {
		for _, probe := range loadBalancer.Properties.Probes {
			if probe.Name != nil {
				// Link to Probe child resource
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   azureshared.NetworkLoadBalancerProbe.String(),
						Method: sdp.QueryMethod_GET,
						Query:  shared.CompositeLookupKey(loadBalancerName, *probe.Name),
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true, // Probe changes affect health monitoring of backend instances
						Out: true, // Load balancer changes (like deletion) affect the probes
					}, // Probe is a child resource of the Load Balancer; bidirectional dependency
				})
			}
		}
	}

	// Process OutboundRules (Child Resource)
	// Reference: https://learn.microsoft.com/en-us/rest/api/load-balancer/load-balancer-outbound-rules/get?view=rest-load-balancer-2025-03-01&tabs=HTTP
	if loadBalancer.Properties != nil && loadBalancer.Properties.OutboundRules != nil {
		for _, outboundRule := range loadBalancer.Properties.OutboundRules {
			if outboundRule.Name != nil {
				// Link to OutboundRule child resource
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   azureshared.NetworkLoadBalancerOutboundRule.String(),
						Method: sdp.QueryMethod_GET,
						Query:  shared.CompositeLookupKey(loadBalancerName, *outboundRule.Name),
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true, // OutboundRule changes affect outbound connectivity configuration
						Out: true, // Load balancer changes (like deletion) affect the outbound rules
					}, // OutboundRule is a child resource of the Load Balancer; bidirectional dependency
				})
			}
		}
	}

	// Process InboundNatPools (Child Resource)
	// Reference: https://learn.microsoft.com/en-us/rest/api/load-balancer/inbound-nat-pools/get
	if loadBalancer.Properties != nil && loadBalancer.Properties.InboundNatPools != nil {
		for _, natPool := range loadBalancer.Properties.InboundNatPools {
			if natPool.Name != nil {
				// Link to InboundNatPool child resource
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   azureshared.NetworkLoadBalancerInboundNatPool.String(),
						Method: sdp.QueryMethod_GET,
						Query:  shared.CompositeLookupKey(loadBalancerName, *natPool.Name),
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true, // InboundNatPool changes affect NAT pool configuration
						Out: true, // Load balancer changes (like deletion) affect the NAT pools
					}, // InboundNatPool is a child resource of the Load Balancer; bidirectional dependency
				})
			}
		}
	}

	return sdpItem, nil
}

// ref: https://learn.microsoft.com/en-us/rest/api/load-balancer/load-balancers/get?view=rest-load-balancer-2025-03-01&tabs=HTTP
func (n networkLoadBalancerWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		NetworkLoadBalancerLookupByName,
	}
}

func (n networkLoadBalancerWrapper) PotentialLinks() map[shared.ItemType]bool {
	return map[shared.ItemType]bool{
		// Child resources
		azureshared.NetworkLoadBalancerFrontendIPConfiguration: true,
		azureshared.NetworkLoadBalancerBackendAddressPool:      true,
		azureshared.NetworkLoadBalancerInboundNatRule:          true,
		azureshared.NetworkLoadBalancerLoadBalancingRule:       true,
		azureshared.NetworkLoadBalancerProbe:                   true,
		azureshared.NetworkLoadBalancerOutboundRule:            true,
		azureshared.NetworkLoadBalancerInboundNatPool:          true,
		// External resources
		azureshared.NetworkPublicIPAddress:  true,
		azureshared.NetworkSubnet:           true,
		azureshared.NetworkNetworkInterface: true,
		// Standard library resources
		stdlib.NetworkIP: true,
	}
}

// ref: https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/lb
func (n networkLoadBalancerWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "azurerm_lb.name",
		},
	}
}

// ref; https://learn.microsoft.com/en-us/azure/role-based-access-control/permissions/networking
func (n networkLoadBalancerWrapper) IAMPermissions() []string {
	return []string{
		"Microsoft.Network/loadBalancers/read",
	}
}

func (n networkLoadBalancerWrapper) PredefinedRole() string {
	return "Reader"
}
