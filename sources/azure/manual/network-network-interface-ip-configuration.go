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
	"github.com/overmindtech/cli/sources/stdlib"
)

var NetworkNetworkInterfaceIPConfigurationLookupByName = shared.NewItemTypeLookup("name", azureshared.NetworkNetworkInterfaceIPConfiguration)

type networkNetworkInterfaceIPConfigurationWrapper struct {
	client clients.InterfaceIPConfigurationsClient

	*azureshared.MultiResourceGroupBase
}

func NewNetworkNetworkInterfaceIPConfiguration(client clients.InterfaceIPConfigurationsClient, resourceGroupScopes []azureshared.ResourceGroupScope) sources.SearchableWrapper {
	return &networkNetworkInterfaceIPConfigurationWrapper{
		client: client,
		MultiResourceGroupBase: azureshared.NewMultiResourceGroupBase(
			resourceGroupScopes,
			sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
			azureshared.NetworkNetworkInterfaceIPConfiguration,
		),
	}
}

func (n networkNetworkInterfaceIPConfigurationWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 2 {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "Get requires 2 query parts: networkInterfaceName and ipConfigurationName",
			Scope:       scope,
			ItemType:    n.Type(),
		}
	}
	networkInterfaceName := queryParts[0]
	ipConfigurationName := queryParts[1]

	if networkInterfaceName == "" {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "networkInterfaceName cannot be empty",
			Scope:       scope,
			ItemType:    n.Type(),
		}
	}
	if ipConfigurationName == "" {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "ipConfigurationName cannot be empty",
			Scope:       scope,
			ItemType:    n.Type(),
		}
	}

	rgScope, err := n.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, n.Type())
	}

	resp, err := n.client.Get(ctx, rgScope.ResourceGroup, networkInterfaceName, ipConfigurationName)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, n.Type())
	}

	return n.azureIPConfigurationToSDPItem(&resp.InterfaceIPConfiguration, networkInterfaceName, scope)
}

func (n networkNetworkInterfaceIPConfigurationWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		NetworkNetworkInterfaceLookupByName,
		NetworkNetworkInterfaceIPConfigurationLookupByName,
	}
}

func (n networkNetworkInterfaceIPConfigurationWrapper) Search(ctx context.Context, scope string, queryParts ...string) ([]*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 1 {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "Search requires 1 query part: networkInterfaceName",
			Scope:       scope,
			ItemType:    n.Type(),
		}
	}
	networkInterfaceName := queryParts[0]

	if networkInterfaceName == "" {
		return nil, azureshared.QueryError(errors.New("networkInterfaceName cannot be empty"), scope, n.Type())
	}

	rgScope, err := n.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, n.Type())
	}

	pager := n.client.List(ctx, rgScope.ResourceGroup, networkInterfaceName)

	var items []*sdp.Item
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, azureshared.QueryError(err, scope, n.Type())
		}

		for _, ipConfig := range page.Value {
			if ipConfig.Name == nil {
				continue
			}

			item, sdpErr := n.azureIPConfigurationToSDPItem(ipConfig, networkInterfaceName, scope)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}

	return items, nil
}

func (n networkNetworkInterfaceIPConfigurationWrapper) SearchStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string, queryParts ...string) {
	if len(queryParts) < 1 {
		stream.SendError(azureshared.QueryError(errors.New("SearchStream requires 1 query part: networkInterfaceName"), scope, n.Type()))
		return
	}
	networkInterfaceName := queryParts[0]

	if networkInterfaceName == "" {
		stream.SendError(azureshared.QueryError(errors.New("networkInterfaceName cannot be empty"), scope, n.Type()))
		return
	}

	rgScope, err := n.ResourceGroupScopeFromScope(scope)
	if err != nil {
		stream.SendError(azureshared.QueryError(err, scope, n.Type()))
		return
	}

	pager := n.client.List(ctx, rgScope.ResourceGroup, networkInterfaceName)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			stream.SendError(azureshared.QueryError(err, scope, n.Type()))
			return
		}
		for _, ipConfig := range page.Value {
			if ipConfig.Name == nil {
				continue
			}
			item, sdpErr := n.azureIPConfigurationToSDPItem(ipConfig, networkInterfaceName, scope)
			if sdpErr != nil {
				stream.SendError(sdpErr)
				continue
			}
			cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
			stream.SendItem(item)
		}
	}
}

func (n networkNetworkInterfaceIPConfigurationWrapper) SearchLookups() []sources.ItemTypeLookups {
	return []sources.ItemTypeLookups{
		{
			NetworkNetworkInterfaceLookupByName,
		},
	}
}

// ref: https://learn.microsoft.com/en-us/rest/api/virtualnetwork/network-interface-ip-configurations/get
func (n networkNetworkInterfaceIPConfigurationWrapper) azureIPConfigurationToSDPItem(ipConfig *armnetwork.InterfaceIPConfiguration, networkInterfaceName, scope string) (*sdp.Item, *sdp.QueryError) {
	if ipConfig.Name == nil {
		return nil, azureshared.QueryError(errors.New("IP configuration name is nil"), scope, n.Type())
	}

	attributes, err := shared.ToAttributesWithExclude(ipConfig)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, n.Type())
	}

	err = attributes.Set("uniqueAttr", shared.CompositeLookupKey(networkInterfaceName, *ipConfig.Name))
	if err != nil {
		return nil, azureshared.QueryError(err, scope, n.Type())
	}

	sdpItem := &sdp.Item{
		Type:            azureshared.NetworkNetworkInterfaceIPConfiguration.String(),
		UniqueAttribute: "uniqueAttr",
		Attributes:      attributes,
		Scope:           scope,
	}

	// Health status based on provisioning state
	if ipConfig.Properties != nil && ipConfig.Properties.ProvisioningState != nil {
		switch *ipConfig.Properties.ProvisioningState {
		case armnetwork.ProvisioningStateSucceeded:
			sdpItem.Health = sdp.Health_HEALTH_OK.Enum()
		case armnetwork.ProvisioningStateUpdating, armnetwork.ProvisioningStateDeleting, armnetwork.ProvisioningStateCreating:
			sdpItem.Health = sdp.Health_HEALTH_PENDING.Enum()
		case armnetwork.ProvisioningStateFailed, armnetwork.ProvisioningStateCanceled:
			sdpItem.Health = sdp.Health_HEALTH_ERROR.Enum()
		default:
			sdpItem.Health = sdp.Health_HEALTH_UNKNOWN.Enum()
		}
	}

	// Link back to parent NetworkInterface
	sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   azureshared.NetworkNetworkInterface.String(),
			Method: sdp.QueryMethod_GET,
			Query:  networkInterfaceName,
			Scope:  scope,
		},
	})

	if ipConfig.Properties != nil {
		props := ipConfig.Properties

		// Subnet link
		if props.Subnet != nil && props.Subnet.ID != nil {
			subnetParams := azureshared.ExtractPathParamsFromResourceID(*props.Subnet.ID, []string{"virtualNetworks", "subnets"})
			if len(subnetParams) >= 2 {
				vnetName, subnetName := subnetParams[0], subnetParams[1]
				linkedScope := scope
				if extractedScope := azureshared.ExtractScopeFromResourceID(*props.Subnet.ID); extractedScope != "" {
					linkedScope = extractedScope
				}
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   azureshared.NetworkSubnet.String(),
						Method: sdp.QueryMethod_GET,
						Query:  shared.CompositeLookupKey(vnetName, subnetName),
						Scope:  linkedScope,
					},
				})
			}
		}

		// Public IP address link
		if props.PublicIPAddress != nil && props.PublicIPAddress.ID != nil {
			pipName := azureshared.ExtractResourceName(*props.PublicIPAddress.ID)
			if pipName != "" {
				linkedScope := scope
				if extractedScope := azureshared.ExtractScopeFromResourceID(*props.PublicIPAddress.ID); extractedScope != "" {
					linkedScope = extractedScope
				}
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   azureshared.NetworkPublicIPAddress.String(),
						Method: sdp.QueryMethod_GET,
						Query:  pipName,
						Scope:  linkedScope,
					},
				})
			}
		}

		// Private IP address -> stdlib ip
		if props.PrivateIPAddress != nil && *props.PrivateIPAddress != "" {
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   stdlib.NetworkIP.String(),
					Method: sdp.QueryMethod_GET,
					Query:  *props.PrivateIPAddress,
					Scope:  "global",
				},
			})
		}

		// Application security groups
		if props.ApplicationSecurityGroups != nil {
			for _, asg := range props.ApplicationSecurityGroups {
				if asg != nil && asg.ID != nil {
					asgName := azureshared.ExtractResourceName(*asg.ID)
					if asgName != "" {
						linkedScope := scope
						if extractedScope := azureshared.ExtractScopeFromResourceID(*asg.ID); extractedScope != "" {
							linkedScope = extractedScope
						}
						sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   azureshared.NetworkApplicationSecurityGroup.String(),
								Method: sdp.QueryMethod_GET,
								Query:  asgName,
								Scope:  linkedScope,
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
						linkedScope := scope
						if extractedScope := azureshared.ExtractScopeFromResourceID(*pool.ID); extractedScope != "" {
							linkedScope = extractedScope
						}
						sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   azureshared.NetworkLoadBalancerBackendAddressPool.String(),
								Method: sdp.QueryMethod_GET,
								Query:  shared.CompositeLookupKey(params[0], params[1]),
								Scope:  linkedScope,
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
						linkedScope := scope
						if extractedScope := azureshared.ExtractScopeFromResourceID(*rule.ID); extractedScope != "" {
							linkedScope = extractedScope
						}
						sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   azureshared.NetworkLoadBalancerInboundNatRule.String(),
								Method: sdp.QueryMethod_GET,
								Query:  shared.CompositeLookupKey(params[0], params[1]),
								Scope:  linkedScope,
							},
						})
					}
				}
			}
		}

		// Application gateway backend address pools
		if props.ApplicationGatewayBackendAddressPools != nil {
			for _, pool := range props.ApplicationGatewayBackendAddressPools {
				if pool != nil && pool.ID != nil {
					params := azureshared.ExtractPathParamsFromResourceID(*pool.ID, []string{"applicationGateways", "backendAddressPools"})
					if len(params) >= 2 {
						linkedScope := scope
						if extractedScope := azureshared.ExtractScopeFromResourceID(*pool.ID); extractedScope != "" {
							linkedScope = extractedScope
						}
						sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   azureshared.NetworkApplicationGatewayBackendAddressPool.String(),
								Method: sdp.QueryMethod_GET,
								Query:  shared.CompositeLookupKey(params[0], params[1]),
								Scope:  linkedScope,
							},
						})
					}
				}
			}
		}

		// Gateway load balancer (frontend IP config reference)
		if props.GatewayLoadBalancer != nil && props.GatewayLoadBalancer.ID != nil {
			params := azureshared.ExtractPathParamsFromResourceID(*props.GatewayLoadBalancer.ID, []string{"loadBalancers", "frontendIPConfigurations"})
			if len(params) >= 2 {
				linkedScope := scope
				if extractedScope := azureshared.ExtractScopeFromResourceID(*props.GatewayLoadBalancer.ID); extractedScope != "" {
					linkedScope = extractedScope
				}
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   azureshared.NetworkLoadBalancerFrontendIPConfiguration.String(),
						Method: sdp.QueryMethod_GET,
						Query:  shared.CompositeLookupKey(params[0], params[1]),
						Scope:  linkedScope,
					},
				})
			}
		}

		// Virtual network taps
		if props.VirtualNetworkTaps != nil {
			for _, tap := range props.VirtualNetworkTaps {
				if tap != nil && tap.ID != nil {
					tapName := azureshared.ExtractResourceName(*tap.ID)
					if tapName != "" {
						linkedScope := scope
						if extractedScope := azureshared.ExtractScopeFromResourceID(*tap.ID); extractedScope != "" {
							linkedScope = extractedScope
						}
						sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   azureshared.NetworkVirtualNetworkTap.String(),
								Method: sdp.QueryMethod_GET,
								Query:  tapName,
								Scope:  linkedScope,
							},
						})
					}
				}
			}
		}

		// PrivateLinkConnectionProperties - FQDNs
		if props.PrivateLinkConnectionProperties != nil && props.PrivateLinkConnectionProperties.Fqdns != nil {
			for _, fqdn := range props.PrivateLinkConnectionProperties.Fqdns {
				if fqdn != nil && *fqdn != "" {
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   stdlib.NetworkDNS.String(),
							Method: sdp.QueryMethod_SEARCH,
							Query:  *fqdn,
							Scope:  "global",
						},
					})
				}
			}
		}
	}

	return sdpItem, nil
}

func (n networkNetworkInterfaceIPConfigurationWrapper) PotentialLinks() map[shared.ItemType]bool {
	return map[shared.ItemType]bool{
		azureshared.NetworkNetworkInterface:                     true,
		azureshared.NetworkSubnet:                               true,
		azureshared.NetworkPublicIPAddress:                      true,
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

// ref: https://learn.microsoft.com/en-us/azure/role-based-access-control/resource-provider-operations#microsoftnetwork
func (n networkNetworkInterfaceIPConfigurationWrapper) IAMPermissions() []string {
	return []string{
		"Microsoft.Network/networkInterfaces/ipConfigurations/read",
	}
}

func (n networkNetworkInterfaceIPConfigurationWrapper) PredefinedRole() string {
	return "Reader"
}
