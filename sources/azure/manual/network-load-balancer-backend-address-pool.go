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

var NetworkLoadBalancerBackendAddressPoolLookupByUniqueAttr = shared.NewItemTypeLookup("uniqueAttr", azureshared.NetworkLoadBalancerBackendAddressPool)

type networkLoadBalancerBackendAddressPoolWrapper struct {
	client clients.LoadBalancerBackendAddressPoolsClient

	*azureshared.MultiResourceGroupBase
}

func NewNetworkLoadBalancerBackendAddressPool(client clients.LoadBalancerBackendAddressPoolsClient, resourceGroupScopes []azureshared.ResourceGroupScope) sources.SearchableWrapper {
	return &networkLoadBalancerBackendAddressPoolWrapper{
		client: client,
		MultiResourceGroupBase: azureshared.NewMultiResourceGroupBase(
			resourceGroupScopes,
			sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
			azureshared.NetworkLoadBalancerBackendAddressPool,
		),
	}
}

func (c networkLoadBalancerBackendAddressPoolWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 2 {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "Get requires 2 query parts: loadBalancerName and backendAddressPoolName",
			Scope:       scope,
			ItemType:    c.Type(),
		}
	}
	loadBalancerName := queryParts[0]
	backendAddressPoolName := queryParts[1]

	if loadBalancerName == "" {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "loadBalancerName cannot be empty",
			Scope:       scope,
			ItemType:    c.Type(),
		}
	}
	if backendAddressPoolName == "" {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "backendAddressPoolName cannot be empty",
			Scope:       scope,
			ItemType:    c.Type(),
		}
	}

	rgScope, err := c.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}
	resp, err := c.client.Get(ctx, rgScope.ResourceGroup, loadBalancerName, backendAddressPoolName)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}

	return c.azureBackendAddressPoolToSDPItem(&resp.BackendAddressPool, loadBalancerName, scope)
}

func (c networkLoadBalancerBackendAddressPoolWrapper) Search(ctx context.Context, scope string, queryParts ...string) ([]*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 1 {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "Search requires 1 query part: loadBalancerName",
			Scope:       scope,
			ItemType:    c.Type(),
		}
	}
	loadBalancerName := queryParts[0]

	if loadBalancerName == "" {
		return nil, azureshared.QueryError(errors.New("loadBalancerName cannot be empty"), scope, c.Type())
	}

	rgScope, err := c.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}
	pager := c.client.NewListPager(rgScope.ResourceGroup, loadBalancerName)

	var items []*sdp.Item
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, azureshared.QueryError(err, scope, c.Type())
		}

		for _, backendPool := range page.Value {
			if backendPool == nil || backendPool.Name == nil {
				continue
			}
			item, sdpErr := c.azureBackendAddressPoolToSDPItem(backendPool, loadBalancerName, scope)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}

	return items, nil
}

func (c networkLoadBalancerBackendAddressPoolWrapper) SearchStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string, queryParts ...string) {
	if len(queryParts) < 1 {
		stream.SendError(azureshared.QueryError(errors.New("Search requires 1 query part: loadBalancerName"), scope, c.Type()))
		return
	}
	loadBalancerName := queryParts[0]

	if loadBalancerName == "" {
		stream.SendError(azureshared.QueryError(errors.New("loadBalancerName cannot be empty"), scope, c.Type()))
		return
	}

	rgScope, err := c.ResourceGroupScopeFromScope(scope)
	if err != nil {
		stream.SendError(azureshared.QueryError(err, scope, c.Type()))
		return
	}
	pager := c.client.NewListPager(rgScope.ResourceGroup, loadBalancerName)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			stream.SendError(azureshared.QueryError(err, scope, c.Type()))
			return
		}
		for _, backendPool := range page.Value {
			if backendPool == nil || backendPool.Name == nil {
				continue
			}
			item, sdpErr := c.azureBackendAddressPoolToSDPItem(backendPool, loadBalancerName, scope)
			if sdpErr != nil {
				stream.SendError(sdpErr)
				continue
			}
			cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
			stream.SendItem(item)
		}
	}
}

func (c networkLoadBalancerBackendAddressPoolWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		NetworkLoadBalancerLookupByName,
		NetworkLoadBalancerBackendAddressPoolLookupByUniqueAttr,
	}
}

func (c networkLoadBalancerBackendAddressPoolWrapper) SearchLookups() []sources.ItemTypeLookups {
	return []sources.ItemTypeLookups{
		{
			NetworkLoadBalancerLookupByName,
		},
	}
}

func (c networkLoadBalancerBackendAddressPoolWrapper) azureBackendAddressPoolToSDPItem(backendPool *armnetwork.BackendAddressPool, loadBalancerName string, scope string) (*sdp.Item, *sdp.QueryError) {
	if backendPool.Name == nil {
		return nil, azureshared.QueryError(errors.New("backend address pool name is nil"), scope, c.Type())
	}

	backendPoolName := *backendPool.Name

	attributes, err := shared.ToAttributesWithExclude(backendPool, "tags")
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}

	err = attributes.Set("uniqueAttr", shared.CompositeLookupKey(loadBalancerName, backendPoolName))
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}

	sdpItem := &sdp.Item{
		Type:            azureshared.NetworkLoadBalancerBackendAddressPool.String(),
		UniqueAttribute: "uniqueAttr",
		Attributes:      attributes,
		Scope:           scope,
	}

	// Health status from provisioning state
	if backendPool.Properties != nil && backendPool.Properties.ProvisioningState != nil {
		switch *backendPool.Properties.ProvisioningState {
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

	// Link to parent Load Balancer
	sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   azureshared.NetworkLoadBalancer.String(),
			Method: sdp.QueryMethod_GET,
			Query:  loadBalancerName,
			Scope:  scope,
		},
	})

	if backendPool.Properties != nil {
		// Link to Virtual Network (pool level)
		if backendPool.Properties.VirtualNetwork != nil && backendPool.Properties.VirtualNetwork.ID != nil {
			vnetName := azureshared.ExtractResourceName(*backendPool.Properties.VirtualNetwork.ID)
			if vnetName != "" {
				linkedScope := scope
				if extractedScope := azureshared.ExtractScopeFromResourceID(*backendPool.Properties.VirtualNetwork.ID); extractedScope != "" {
					linkedScope = extractedScope
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

		// Link to Inbound NAT Rules (read-only references)
		for _, natRule := range backendPool.Properties.InboundNatRules {
			if natRule != nil && natRule.ID != nil {
				params := azureshared.ExtractPathParamsFromResourceID(*natRule.ID, []string{"loadBalancers", "inboundNatRules"})
				if len(params) >= 2 {
					linkedScope := scope
					if extractedScope := azureshared.ExtractScopeFromResourceID(*natRule.ID); extractedScope != "" {
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

		// Link to Load Balancing Rules (read-only references)
		for _, lbRule := range backendPool.Properties.LoadBalancingRules {
			if lbRule != nil && lbRule.ID != nil {
				params := azureshared.ExtractPathParamsFromResourceID(*lbRule.ID, []string{"loadBalancers", "loadBalancingRules"})
				if len(params) >= 2 {
					linkedScope := scope
					if extractedScope := azureshared.ExtractScopeFromResourceID(*lbRule.ID); extractedScope != "" {
						linkedScope = extractedScope
					}
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.NetworkLoadBalancerLoadBalancingRule.String(),
							Method: sdp.QueryMethod_GET,
							Query:  shared.CompositeLookupKey(params[0], params[1]),
							Scope:  linkedScope,
						},
					})
				}
			}
		}

		// Link to Outbound Rule (single read-only reference)
		if backendPool.Properties.OutboundRule != nil && backendPool.Properties.OutboundRule.ID != nil {
			params := azureshared.ExtractPathParamsFromResourceID(*backendPool.Properties.OutboundRule.ID, []string{"loadBalancers", "outboundRules"})
			if len(params) >= 2 {
				linkedScope := scope
				if extractedScope := azureshared.ExtractScopeFromResourceID(*backendPool.Properties.OutboundRule.ID); extractedScope != "" {
					linkedScope = extractedScope
				}
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   azureshared.NetworkLoadBalancerOutboundRule.String(),
						Method: sdp.QueryMethod_GET,
						Query:  shared.CompositeLookupKey(params[0], params[1]),
						Scope:  linkedScope,
					},
				})
			}
		}

		// Link to Outbound Rules (read-only references array)
		for _, outboundRule := range backendPool.Properties.OutboundRules {
			if outboundRule != nil && outboundRule.ID != nil {
				params := azureshared.ExtractPathParamsFromResourceID(*outboundRule.ID, []string{"loadBalancers", "outboundRules"})
				if len(params) >= 2 {
					linkedScope := scope
					if extractedScope := azureshared.ExtractScopeFromResourceID(*outboundRule.ID); extractedScope != "" {
						linkedScope = extractedScope
					}
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.NetworkLoadBalancerOutboundRule.String(),
							Method: sdp.QueryMethod_GET,
							Query:  shared.CompositeLookupKey(params[0], params[1]),
							Scope:  linkedScope,
						},
					})
				}
			}
		}

		// Link to Backend IP Configurations (Network Interface IP Configurations)
		for _, backendIPConfig := range backendPool.Properties.BackendIPConfigurations {
			if backendIPConfig != nil && backendIPConfig.ID != nil {
				params := azureshared.ExtractPathParamsFromResourceID(*backendIPConfig.ID, []string{"networkInterfaces", "ipConfigurations"})
				if len(params) >= 2 {
					linkedScope := scope
					if extractedScope := azureshared.ExtractScopeFromResourceID(*backendIPConfig.ID); extractedScope != "" {
						linkedScope = extractedScope
					}
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.NetworkNetworkInterfaceIPConfiguration.String(),
							Method: sdp.QueryMethod_GET,
							Query:  shared.CompositeLookupKey(params[0], params[1]),
							Scope:  linkedScope,
						},
					})
				}
			}
		}

		// Link to Backend Addresses (IP addresses, VNets, Subnets, Frontend IP Configs)
		for _, addr := range backendPool.Properties.LoadBalancerBackendAddresses {
			if addr == nil || addr.Properties == nil {
				continue
			}

			// Link to Virtual Network
			if addr.Properties.VirtualNetwork != nil && addr.Properties.VirtualNetwork.ID != nil {
				vnetName := azureshared.ExtractResourceName(*addr.Properties.VirtualNetwork.ID)
				if vnetName != "" {
					linkedScope := scope
					if extractedScope := azureshared.ExtractScopeFromResourceID(*addr.Properties.VirtualNetwork.ID); extractedScope != "" {
						linkedScope = extractedScope
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

			// Link to Subnet
			if addr.Properties.Subnet != nil && addr.Properties.Subnet.ID != nil {
				params := azureshared.ExtractPathParamsFromResourceID(*addr.Properties.Subnet.ID, []string{"virtualNetworks", "subnets"})
				if len(params) >= 2 {
					linkedScope := scope
					if extractedScope := azureshared.ExtractScopeFromResourceID(*addr.Properties.Subnet.ID); extractedScope != "" {
						linkedScope = extractedScope
					}
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.NetworkSubnet.String(),
							Method: sdp.QueryMethod_GET,
							Query:  shared.CompositeLookupKey(params[0], params[1]),
							Scope:  linkedScope,
						},
					})
				}
			}

			// Link to Frontend IP Configuration (regional LB)
			if addr.Properties.LoadBalancerFrontendIPConfiguration != nil && addr.Properties.LoadBalancerFrontendIPConfiguration.ID != nil {
				params := azureshared.ExtractPathParamsFromResourceID(*addr.Properties.LoadBalancerFrontendIPConfiguration.ID, []string{"loadBalancers", "frontendIPConfigurations"})
				if len(params) >= 2 {
					linkedScope := scope
					if extractedScope := azureshared.ExtractScopeFromResourceID(*addr.Properties.LoadBalancerFrontendIPConfiguration.ID); extractedScope != "" {
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

			// Link to Network Interface IP Configuration
			if addr.Properties.NetworkInterfaceIPConfiguration != nil && addr.Properties.NetworkInterfaceIPConfiguration.ID != nil {
				params := azureshared.ExtractPathParamsFromResourceID(*addr.Properties.NetworkInterfaceIPConfiguration.ID, []string{"networkInterfaces", "ipConfigurations"})
				if len(params) >= 2 {
					linkedScope := scope
					if extractedScope := azureshared.ExtractScopeFromResourceID(*addr.Properties.NetworkInterfaceIPConfiguration.ID); extractedScope != "" {
						linkedScope = extractedScope
					}
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.NetworkNetworkInterfaceIPConfiguration.String(),
							Method: sdp.QueryMethod_GET,
							Query:  shared.CompositeLookupKey(params[0], params[1]),
							Scope:  linkedScope,
						},
					})
				}
			}

			// Link to IP Address (stdlib)
			if addr.Properties.IPAddress != nil && *addr.Properties.IPAddress != "" {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   stdlib.NetworkIP.String(),
						Method: sdp.QueryMethod_GET,
						Query:  *addr.Properties.IPAddress,
						Scope:  "global",
					},
				})
			}
		}
	}

	return sdpItem, nil
}

func (c networkLoadBalancerBackendAddressPoolWrapper) PotentialLinks() map[shared.ItemType]bool {
	return map[shared.ItemType]bool{
		azureshared.NetworkLoadBalancer:                        true,
		azureshared.NetworkVirtualNetwork:                      true,
		azureshared.NetworkSubnet:                              true,
		azureshared.NetworkNetworkInterfaceIPConfiguration:     true,
		azureshared.NetworkLoadBalancerInboundNatRule:          true,
		azureshared.NetworkLoadBalancerLoadBalancingRule:       true,
		azureshared.NetworkLoadBalancerOutboundRule:            true,
		azureshared.NetworkLoadBalancerFrontendIPConfiguration: true,
		stdlib.NetworkIP:                                       true,
	}
}

// ref: https://learn.microsoft.com/en-us/azure/role-based-access-control/permissions/networking
func (c networkLoadBalancerBackendAddressPoolWrapper) IAMPermissions() []string {
	return []string{
		"Microsoft.Network/loadBalancers/backendAddressPools/read",
	}
}

func (c networkLoadBalancerBackendAddressPoolWrapper) PredefinedRole() string {
	return "Reader"
}
