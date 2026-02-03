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

func (n networkNetworkInterfaceWrapper) List(ctx context.Context, scope string) ([]*sdp.Item, *sdp.QueryError) {
	resourceGroup := azureshared.ResourceGroupFromScope(scope)
	if resourceGroup == "" {
		resourceGroup = n.ResourceGroup()
	}
	pager := n.client.List(ctx, resourceGroup)

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

	// Private endpoint (read-only reference when NIC is used by a private endpoint)
	if networkInterface.Properties != nil && networkInterface.Properties.PrivateEndpoint != nil &&
		networkInterface.Properties.PrivateEndpoint.ID != nil {
		peName := azureshared.ExtractResourceName(*networkInterface.Properties.PrivateEndpoint.ID)
		if peName != "" {
			scope := azureshared.ExtractScopeFromResourceID(*networkInterface.Properties.PrivateEndpoint.ID)
			if scope == "" {
				scope = n.DefaultScope()
			}
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   azureshared.NetworkPrivateEndpoint.String(),
					Method: sdp.QueryMethod_GET,
					Query:  peName,
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true, // Private endpoint changes affect the NIC's role
					Out: true, // NIC changes affect the private endpoint's connectivity
				},
			})
		}
	}

	// Private Link Service (when this NIC is the frontend of a private link service)
	if networkInterface.Properties != nil && networkInterface.Properties.PrivateLinkService != nil &&
		networkInterface.Properties.PrivateLinkService.ID != nil {
		plsName := azureshared.ExtractResourceName(*networkInterface.Properties.PrivateLinkService.ID)
		if plsName != "" {
			scope := azureshared.ExtractScopeFromResourceID(*networkInterface.Properties.PrivateLinkService.ID)
			if scope == "" {
				scope = n.DefaultScope()
			}
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   azureshared.NetworkPrivateLinkService.String(),
					Method: sdp.QueryMethod_GET,
					Query:  plsName,
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true, // Private link service changes affect the NIC
					Out: true, // NIC changes affect the private link service
				},
			})
		}
	}

	// DSCP configuration (read-only reference)
	if networkInterface.Properties != nil && networkInterface.Properties.DscpConfiguration != nil &&
		networkInterface.Properties.DscpConfiguration.ID != nil {
		dscpName := azureshared.ExtractResourceName(*networkInterface.Properties.DscpConfiguration.ID)
		if dscpName != "" {
			scope := azureshared.ExtractScopeFromResourceID(*networkInterface.Properties.DscpConfiguration.ID)
			if scope == "" {
				scope = n.DefaultScope()
			}
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   azureshared.NetworkDscpConfiguration.String(),
					Method: sdp.QueryMethod_GET,
					Query:  dscpName,
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,  // DSCP config changes affect NIC QoS
					Out: false, // NIC changes don't affect the DSCP configuration resource
				},
			})
		}
	}

	// Tap configurations (child resource; list by NIC name)
	sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   azureshared.NetworkNetworkInterfaceTapConfiguration.String(),
			Method: sdp.QueryMethod_SEARCH,
			Query:  *networkInterface.Name,
			Scope:  n.DefaultScope(),
		},
		BlastPropagation: &sdp.BlastPropagation{
			In:  false, // Tap config changes don't affect the NIC itself
			Out: true,  // NIC changes (e.g. deletion) affect tap configurations
		},
	})

	// IP configuration references: subnet, public IP, private IP (stdlib), ASGs, LB pools/rules, App Gateway pools, gateway LB, VNet taps
	if networkInterface.Properties != nil && networkInterface.Properties.IPConfigurations != nil {
		for _, ipConfig := range networkInterface.Properties.IPConfigurations {
			if ipConfig == nil || ipConfig.Properties == nil {
				continue
			}
			props := ipConfig.Properties

			// Subnet
			if props.Subnet != nil && props.Subnet.ID != nil {
				subnetParams := azureshared.ExtractPathParamsFromResourceID(*props.Subnet.ID, []string{"virtualNetworks", "subnets"})
				if len(subnetParams) >= 2 {
					vnetName, subnetName := subnetParams[0], subnetParams[1]
					scope := azureshared.ExtractScopeFromResourceID(*props.Subnet.ID)
					if scope == "" {
						scope = n.DefaultScope()
					}
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.NetworkSubnet.String(),
							Method: sdp.QueryMethod_GET,
							Query:  shared.CompositeLookupKey(vnetName, subnetName),
							Scope:  scope,
						},
						BlastPropagation: &sdp.BlastPropagation{
							In:  true,  // Subnet changes (e.g. address space) affect the NIC's IP config
							Out: false, // NIC changes don't affect the subnet resource
						},
					})
				}
			}

			// Public IP address
			if props.PublicIPAddress != nil && props.PublicIPAddress.ID != nil {
				pipName := azureshared.ExtractResourceName(*props.PublicIPAddress.ID)
				if pipName != "" {
					scope := azureshared.ExtractScopeFromResourceID(*props.PublicIPAddress.ID)
					if scope == "" {
						scope = n.DefaultScope()
					}
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.NetworkPublicIPAddress.String(),
							Method: sdp.QueryMethod_GET,
							Query:  pipName,
							Scope:  scope,
						},
						BlastPropagation: &sdp.BlastPropagation{
							In:  true, // Public IP changes affect the NIC's connectivity
							Out: true, // NIC detachment affects the public IP's association
						},
					})
				}
			}

			// Private IP address -> stdlib ip
			if props.PrivateIPAddress != nil && *props.PrivateIPAddress != "" {
				addr := *props.PrivateIPAddress
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   stdlib.NetworkIP.String(),
						Method: sdp.QueryMethod_GET,
						Query:  addr,
						Scope:  "global",
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				})
			}

			// Application security groups
			if props.ApplicationSecurityGroups != nil {
				for _, asg := range props.ApplicationSecurityGroups {
					if asg != nil && asg.ID != nil {
						asgName := azureshared.ExtractResourceName(*asg.ID)
						if asgName != "" {
							scope := azureshared.ExtractScopeFromResourceID(*asg.ID)
							if scope == "" {
								scope = n.DefaultScope()
							}
							sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
								Query: &sdp.Query{
									Type:   azureshared.NetworkApplicationSecurityGroup.String(),
									Method: sdp.QueryMethod_GET,
									Query:  asgName,
									Scope:  scope,
								},
								BlastPropagation: &sdp.BlastPropagation{
									In:  true,  // ASG rule changes affect the NIC's effective rules
									Out: false, // NIC changes don't affect the ASG
								},
							})
						}
					}
				}
			}

			// Load balancer backend address pools
			if props.LoadBalancerBackendAddressPools != nil {
				for _, pool := range props.LoadBalancerBackendAddressPools {
					if pool != nil && pool.ID != nil {
						params := azureshared.ExtractPathParamsFromResourceID(*pool.ID, []string{"loadBalancers", "backendAddressPools"})
						if len(params) >= 2 {
							scope := azureshared.ExtractScopeFromResourceID(*pool.ID)
							if scope == "" {
								scope = n.DefaultScope()
							}
							sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
								Query: &sdp.Query{
									Type:   azureshared.NetworkLoadBalancerBackendAddressPool.String(),
									Method: sdp.QueryMethod_GET,
									Query:  shared.CompositeLookupKey(params[0], params[1]),
									Scope:  scope,
								},
								BlastPropagation: &sdp.BlastPropagation{
									In:  true, // Pool config changes affect which backends receive traffic
									Out: true, // NIC removal affects the pool's members
								},
							})
						}
					}
				}
			}

			// Load balancer inbound NAT rules
			if props.LoadBalancerInboundNatRules != nil {
				for _, rule := range props.LoadBalancerInboundNatRules {
					if rule != nil && rule.ID != nil {
						params := azureshared.ExtractPathParamsFromResourceID(*rule.ID, []string{"loadBalancers", "inboundNatRules"})
						if len(params) >= 2 {
							scope := azureshared.ExtractScopeFromResourceID(*rule.ID)
							if scope == "" {
								scope = n.DefaultScope()
							}
							sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
								Query: &sdp.Query{
									Type:   azureshared.NetworkLoadBalancerInboundNatRule.String(),
									Method: sdp.QueryMethod_GET,
									Query:  shared.CompositeLookupKey(params[0], params[1]),
									Scope:  scope,
								},
								BlastPropagation: &sdp.BlastPropagation{
									In:  true, // NAT rule changes affect the NIC
									Out: true, // NIC removal affects the NAT rule's target
								},
							})
						}
					}
				}
			}

			// Application Gateway backend address pools
			if props.ApplicationGatewayBackendAddressPools != nil {
				for _, pool := range props.ApplicationGatewayBackendAddressPools {
					if pool != nil && pool.ID != nil {
						params := azureshared.ExtractPathParamsFromResourceID(*pool.ID, []string{"applicationGateways", "backendAddressPools"})
						if len(params) >= 2 {
							scope := azureshared.ExtractScopeFromResourceID(*pool.ID)
							if scope == "" {
								scope = n.DefaultScope()
							}
							sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
								Query: &sdp.Query{
									Type:   azureshared.NetworkApplicationGatewayBackendAddressPool.String(),
									Method: sdp.QueryMethod_GET,
									Query:  shared.CompositeLookupKey(params[0], params[1]),
									Scope:  scope,
								},
								BlastPropagation: &sdp.BlastPropagation{
									In:  true, // App GW pool changes affect backend targets
									Out: true, // NIC removal affects the pool's members
								},
							})
						}
					}
				}
			}

			// Gateway Load Balancer (frontend IP config reference)
			if props.GatewayLoadBalancer != nil && props.GatewayLoadBalancer.ID != nil {
				params := azureshared.ExtractPathParamsFromResourceID(*props.GatewayLoadBalancer.ID, []string{"loadBalancers", "frontendIPConfigurations"})
				if len(params) >= 2 {
					scope := azureshared.ExtractScopeFromResourceID(*props.GatewayLoadBalancer.ID)
					if scope == "" {
						scope = n.DefaultScope()
					}
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.NetworkLoadBalancerFrontendIPConfiguration.String(),
							Method: sdp.QueryMethod_GET,
							Query:  shared.CompositeLookupKey(params[0], params[1]),
							Scope:  scope,
						},
						BlastPropagation: &sdp.BlastPropagation{
							In:  true, // Gateway LB frontend changes affect traffic path
							Out: true, // NIC changes affect the gateway LB association
						},
					})
				}
			}

			// Virtual Network Taps
			if props.VirtualNetworkTaps != nil {
				for _, tap := range props.VirtualNetworkTaps {
					if tap != nil && tap.ID != nil {
						tapName := azureshared.ExtractResourceName(*tap.ID)
						if tapName != "" {
							scope := azureshared.ExtractScopeFromResourceID(*tap.ID)
							if scope == "" {
								scope = n.DefaultScope()
							}
							sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
								Query: &sdp.Query{
									Type:   azureshared.NetworkVirtualNetworkTap.String(),
									Method: sdp.QueryMethod_GET,
									Query:  tapName,
									Scope:  scope,
								},
								BlastPropagation: &sdp.BlastPropagation{
									In:  true, // Tap config changes affect what is mirrored
									Out: true, // NIC removal affects the tap's sources
								},
							})
						}
					}
				}
			}
		}
	}

	// DNS settings: link IPs to stdlib.NetworkIP and hostnames to stdlib.NetworkDNS
	if networkInterface.Properties != nil && networkInterface.Properties.DNSSettings != nil {
		dns := networkInterface.Properties.DNSSettings
		if dns.DNSServers != nil {
			for _, srv := range dns.DNSServers {
				if srv == nil {
					continue
				}
				appendDNSServerLinkIfValid(&sdpItem.LinkedItemQueries, *srv, "AzureProvidedDNS")
			}
		}
		if dns.InternalDNSNameLabel != nil && *dns.InternalDNSNameLabel != "" {
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   stdlib.NetworkDNS.String(),
					Method: sdp.QueryMethod_SEARCH,
					Query:  *dns.InternalDNSNameLabel,
					Scope:  "global",
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: true,
				},
			})
		}
		if dns.InternalFqdn != nil && *dns.InternalFqdn != "" {
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   stdlib.NetworkDNS.String(),
					Method: sdp.QueryMethod_SEARCH,
					Query:  *dns.InternalFqdn,
					Scope:  "global",
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: true,
				},
			})
		}
	}

	return sdpItem, nil
}

func (n networkNetworkInterfaceWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) != 1 {
		return nil, azureshared.QueryError(errors.New("query must be exactly one part and be a network interface name"), n.DefaultScope(), n.Type())
	}
	networkInterfaceName := queryParts[0]

	resourceGroup := azureshared.ResourceGroupFromScope(scope)
	if resourceGroup == "" {
		resourceGroup = n.ResourceGroup()
	}
	networkInterface, err := n.client.Get(ctx, resourceGroup, networkInterfaceName)
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
		azureshared.NetworkVirtualNetwork:                       true,
		azureshared.ComputeVirtualMachine:                       true,
		azureshared.NetworkNetworkSecurityGroup:                 true,
		azureshared.NetworkNetworkInterfaceIPConfiguration:      true,
		azureshared.NetworkNetworkInterfaceTapConfiguration:     true,
		azureshared.NetworkSubnet:                               true,
		azureshared.NetworkPublicIPAddress:                      true,
		azureshared.NetworkPrivateEndpoint:                      true,
		azureshared.NetworkPrivateLinkService:                   true,
		azureshared.NetworkDscpConfiguration:                    true,
		azureshared.NetworkApplicationSecurityGroup:             true,
		azureshared.NetworkLoadBalancerBackendAddressPool:       true,
		azureshared.NetworkLoadBalancerInboundNatRule:           true,
		azureshared.NetworkApplicationGatewayBackendAddressPool: true,
		azureshared.NetworkLoadBalancerFrontendIPConfiguration:  true,
		azureshared.NetworkVirtualNetworkTap:                    true,
		stdlib.NetworkIP:                                        true,
		stdlib.NetworkDNS:                                       true,
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
