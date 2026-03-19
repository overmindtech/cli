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

var NetworkLoadBalancerFrontendIPConfigurationLookupByUniqueAttr = shared.NewItemTypeLookup("uniqueAttr", azureshared.NetworkLoadBalancerFrontendIPConfiguration)

type networkLoadBalancerFrontendIPConfigurationWrapper struct {
	client clients.LoadBalancerFrontendIPConfigurationsClient

	*azureshared.MultiResourceGroupBase
}

func NewNetworkLoadBalancerFrontendIPConfiguration(client clients.LoadBalancerFrontendIPConfigurationsClient, resourceGroupScopes []azureshared.ResourceGroupScope) sources.SearchableWrapper {
	return &networkLoadBalancerFrontendIPConfigurationWrapper{
		client: client,
		MultiResourceGroupBase: azureshared.NewMultiResourceGroupBase(
			resourceGroupScopes,
			sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
			azureshared.NetworkLoadBalancerFrontendIPConfiguration,
		),
	}
}

func (c networkLoadBalancerFrontendIPConfigurationWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 2 {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "Get requires 2 query parts: loadBalancerName and frontendIPConfigurationName",
			Scope:       scope,
			ItemType:    c.Type(),
		}
	}
	loadBalancerName := queryParts[0]
	frontendIPConfigName := queryParts[1]

	if loadBalancerName == "" {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "loadBalancerName cannot be empty",
			Scope:       scope,
			ItemType:    c.Type(),
		}
	}
	if frontendIPConfigName == "" {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "frontendIPConfigurationName cannot be empty",
			Scope:       scope,
			ItemType:    c.Type(),
		}
	}

	rgScope, err := c.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}
	resp, err := c.client.Get(ctx, rgScope.ResourceGroup, loadBalancerName, frontendIPConfigName)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}

	return c.azureFrontendIPConfigToSDPItem(&resp.FrontendIPConfiguration, loadBalancerName, scope)
}

func (c networkLoadBalancerFrontendIPConfigurationWrapper) Search(ctx context.Context, scope string, queryParts ...string) ([]*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 1 {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "Search requires 1 query part: loadBalancerName",
			Scope:       scope,
			ItemType:    c.Type(),
		}
	}
	loadBalancerName := queryParts[0]

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

		for _, frontendIPConfig := range page.Value {
			if frontendIPConfig == nil || frontendIPConfig.Name == nil {
				continue
			}
			item, sdpErr := c.azureFrontendIPConfigToSDPItem(frontendIPConfig, loadBalancerName, scope)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}

	return items, nil
}

func (c networkLoadBalancerFrontendIPConfigurationWrapper) SearchStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string, queryParts ...string) {
	if len(queryParts) < 1 {
		stream.SendError(azureshared.QueryError(errors.New("Search requires 1 query part: loadBalancerName"), scope, c.Type()))
		return
	}
	loadBalancerName := queryParts[0]

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
		for _, frontendIPConfig := range page.Value {
			if frontendIPConfig == nil || frontendIPConfig.Name == nil {
				continue
			}
			item, sdpErr := c.azureFrontendIPConfigToSDPItem(frontendIPConfig, loadBalancerName, scope)
			if sdpErr != nil {
				stream.SendError(sdpErr)
				continue
			}
			cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
			stream.SendItem(item)
		}
	}
}

func (c networkLoadBalancerFrontendIPConfigurationWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		NetworkLoadBalancerLookupByName,
		NetworkLoadBalancerFrontendIPConfigurationLookupByUniqueAttr,
	}
}

func (c networkLoadBalancerFrontendIPConfigurationWrapper) SearchLookups() []sources.ItemTypeLookups {
	return []sources.ItemTypeLookups{
		{
			NetworkLoadBalancerLookupByName,
		},
	}
}

func (c networkLoadBalancerFrontendIPConfigurationWrapper) azureFrontendIPConfigToSDPItem(frontendIPConfig *armnetwork.FrontendIPConfiguration, loadBalancerName string, scope string) (*sdp.Item, *sdp.QueryError) {
	if frontendIPConfig.Name == nil {
		return nil, azureshared.QueryError(errors.New("frontend IP configuration name is nil"), scope, c.Type())
	}

	frontendIPConfigName := *frontendIPConfig.Name

	attributes, err := shared.ToAttributesWithExclude(frontendIPConfig, "tags")
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}

	err = attributes.Set("uniqueAttr", shared.CompositeLookupKey(loadBalancerName, frontendIPConfigName))
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}

	sdpItem := &sdp.Item{
		Type:            azureshared.NetworkLoadBalancerFrontendIPConfiguration.String(),
		UniqueAttribute: "uniqueAttr",
		Attributes:      attributes,
		Scope:           scope,
	}

	// Health status from provisioning state
	if frontendIPConfig.Properties != nil && frontendIPConfig.Properties.ProvisioningState != nil {
		switch *frontendIPConfig.Properties.ProvisioningState {
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

	if frontendIPConfig.Properties != nil {
		// Link to Public IP Address
		if frontendIPConfig.Properties.PublicIPAddress != nil && frontendIPConfig.Properties.PublicIPAddress.ID != nil {
			publicIPName := azureshared.ExtractResourceName(*frontendIPConfig.Properties.PublicIPAddress.ID)
			if publicIPName != "" {
				linkedScope := scope
				if extractedScope := azureshared.ExtractScopeFromResourceID(*frontendIPConfig.Properties.PublicIPAddress.ID); extractedScope != "" {
					linkedScope = extractedScope
				}
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   azureshared.NetworkPublicIPAddress.String(),
						Method: sdp.QueryMethod_GET,
						Query:  publicIPName,
						Scope:  linkedScope,
					},
				})
			}
		}

		// Link to Subnet
		if frontendIPConfig.Properties.Subnet != nil && frontendIPConfig.Properties.Subnet.ID != nil {
			subnetID := *frontendIPConfig.Properties.Subnet.ID
			params := azureshared.ExtractPathParamsFromResourceID(subnetID, []string{"virtualNetworks", "subnets"})
			if len(params) >= 2 {
				linkedScope := scope
				if extractedScope := azureshared.ExtractScopeFromResourceID(subnetID); extractedScope != "" {
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

		// Link to Public IP Prefix
		if frontendIPConfig.Properties.PublicIPPrefix != nil && frontendIPConfig.Properties.PublicIPPrefix.ID != nil {
			publicIPPrefixName := azureshared.ExtractResourceName(*frontendIPConfig.Properties.PublicIPPrefix.ID)
			if publicIPPrefixName != "" {
				linkedScope := scope
				if extractedScope := azureshared.ExtractScopeFromResourceID(*frontendIPConfig.Properties.PublicIPPrefix.ID); extractedScope != "" {
					linkedScope = extractedScope
				}
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   azureshared.NetworkPublicIPPrefix.String(),
						Method: sdp.QueryMethod_GET,
						Query:  publicIPPrefixName,
						Scope:  linkedScope,
					},
				})
			}
		}

		// Link to Gateway Load Balancer Frontend IP Configuration
		if frontendIPConfig.Properties.GatewayLoadBalancer != nil && frontendIPConfig.Properties.GatewayLoadBalancer.ID != nil {
			params := azureshared.ExtractPathParamsFromResourceID(*frontendIPConfig.Properties.GatewayLoadBalancer.ID, []string{"loadBalancers", "frontendIPConfigurations"})
			if len(params) >= 2 {
				linkedScope := scope
				if extractedScope := azureshared.ExtractScopeFromResourceID(*frontendIPConfig.Properties.GatewayLoadBalancer.ID); extractedScope != "" {
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

		// Link to Inbound NAT Rules (read-only references)
		for _, natRule := range frontendIPConfig.Properties.InboundNatRules {
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

		// Link to Inbound NAT Pools (read-only references)
		for _, natPool := range frontendIPConfig.Properties.InboundNatPools {
			if natPool != nil && natPool.ID != nil {
				params := azureshared.ExtractPathParamsFromResourceID(*natPool.ID, []string{"loadBalancers", "inboundNatPools"})
				if len(params) >= 2 {
					linkedScope := scope
					if extractedScope := azureshared.ExtractScopeFromResourceID(*natPool.ID); extractedScope != "" {
						linkedScope = extractedScope
					}
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.NetworkLoadBalancerInboundNatPool.String(),
							Method: sdp.QueryMethod_GET,
							Query:  shared.CompositeLookupKey(params[0], params[1]),
							Scope:  linkedScope,
						},
					})
				}
			}
		}

		// Link to Outbound Rules (read-only references)
		for _, outboundRule := range frontendIPConfig.Properties.OutboundRules {
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

		// Link to Load Balancing Rules (read-only references)
		for _, lbRule := range frontendIPConfig.Properties.LoadBalancingRules {
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

		// Link to Private IP Address (stdlib)
		if frontendIPConfig.Properties.PrivateIPAddress != nil && *frontendIPConfig.Properties.PrivateIPAddress != "" {
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   stdlib.NetworkIP.String(),
					Method: sdp.QueryMethod_GET,
					Query:  *frontendIPConfig.Properties.PrivateIPAddress,
					Scope:  "global",
				},
			})
		}
	}

	return sdpItem, nil
}

func (c networkLoadBalancerFrontendIPConfigurationWrapper) PotentialLinks() map[shared.ItemType]bool {
	return map[shared.ItemType]bool{
		azureshared.NetworkLoadBalancer:                        true,
		azureshared.NetworkPublicIPAddress:                     true,
		azureshared.NetworkSubnet:                              true,
		azureshared.NetworkPublicIPPrefix:                      true,
		azureshared.NetworkLoadBalancerFrontendIPConfiguration: true,
		azureshared.NetworkLoadBalancerInboundNatRule:          true,
		azureshared.NetworkLoadBalancerInboundNatPool:          true,
		azureshared.NetworkLoadBalancerOutboundRule:            true,
		azureshared.NetworkLoadBalancerLoadBalancingRule:       true,
		stdlib.NetworkIP:                                       true,
	}
}

// ref: https://learn.microsoft.com/en-us/azure/role-based-access-control/permissions/networking
func (c networkLoadBalancerFrontendIPConfigurationWrapper) IAMPermissions() []string {
	return []string{
		"Microsoft.Network/loadBalancers/frontendIPConfigurations/read",
	}
}

func (c networkLoadBalancerFrontendIPConfigurationWrapper) PredefinedRole() string {
	return "Reader"
}
