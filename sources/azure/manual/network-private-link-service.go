package manual

import (
	"context"
	"errors"
	"strings"

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

var NetworkPrivateLinkServiceLookupByName = shared.NewItemTypeLookup("name", azureshared.NetworkPrivateLinkService)

type networkPrivateLinkServiceWrapper struct {
	client clients.PrivateLinkServicesClient

	*azureshared.MultiResourceGroupBase
}

func NewNetworkPrivateLinkService(client clients.PrivateLinkServicesClient, resourceGroupScopes []azureshared.ResourceGroupScope) sources.ListableWrapper {
	return &networkPrivateLinkServiceWrapper{
		client: client,
		MultiResourceGroupBase: azureshared.NewMultiResourceGroupBase(
			resourceGroupScopes,
			sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
			azureshared.NetworkPrivateLinkService,
		),
	}
}

func (n networkPrivateLinkServiceWrapper) List(ctx context.Context, scope string) ([]*sdp.Item, *sdp.QueryError) {
	rgScope, err := n.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, n.Type())
	}
	pager := n.client.List(rgScope.ResourceGroup)

	var items []*sdp.Item
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, azureshared.QueryError(err, scope, n.Type())
		}
		for _, pls := range page.Value {
			if pls.Name == nil {
				continue
			}
			item, sdpErr := n.azurePrivateLinkServiceToSDPItem(pls, scope)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}
	return items, nil
}

func (n networkPrivateLinkServiceWrapper) ListStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string) {
	rgScope, err := n.ResourceGroupScopeFromScope(scope)
	if err != nil {
		stream.SendError(azureshared.QueryError(err, scope, n.Type()))
		return
	}
	pager := n.client.List(rgScope.ResourceGroup)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			stream.SendError(azureshared.QueryError(err, scope, n.Type()))
			return
		}
		for _, pls := range page.Value {
			if pls.Name == nil {
				continue
			}
			item, sdpErr := n.azurePrivateLinkServiceToSDPItem(pls, scope)
			if sdpErr != nil {
				stream.SendError(sdpErr)
				continue
			}
			cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
			stream.SendItem(item)
		}
	}
}

func (n networkPrivateLinkServiceWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) != 1 {
		return nil, azureshared.QueryError(errors.New("query must be a private link service name"), scope, n.Type())
	}
	serviceName := queryParts[0]
	if serviceName == "" {
		return nil, azureshared.QueryError(errors.New("private link service name cannot be empty"), scope, n.Type())
	}

	rgScope, err := n.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, n.Type())
	}
	resp, err := n.client.Get(ctx, rgScope.ResourceGroup, serviceName)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, n.Type())
	}
	return n.azurePrivateLinkServiceToSDPItem(&resp.PrivateLinkService, scope)
}

func (n networkPrivateLinkServiceWrapper) azurePrivateLinkServiceToSDPItem(pls *armnetwork.PrivateLinkService, scope string) (*sdp.Item, *sdp.QueryError) {
	if pls.Name == nil {
		return nil, azureshared.QueryError(errors.New("private link service name is nil"), scope, n.Type())
	}
	attributes, err := shared.ToAttributesWithExclude(pls, "tags")
	if err != nil {
		return nil, azureshared.QueryError(err, scope, n.Type())
	}

	sdpItem := &sdp.Item{
		Type:            azureshared.NetworkPrivateLinkService.String(),
		UniqueAttribute: "name",
		Attributes:      attributes,
		Scope:           scope,
		Tags:            azureshared.ConvertAzureTags(pls.Tags),
	}

	// Health status from ProvisioningState
	if pls.Properties != nil && pls.Properties.ProvisioningState != nil {
		switch *pls.Properties.ProvisioningState {
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

	// Link to Custom Location when ExtendedLocation.Name is a custom location resource ID
	if pls.ExtendedLocation != nil && pls.ExtendedLocation.Name != nil {
		customLocationID := *pls.ExtendedLocation.Name
		if strings.Contains(customLocationID, "customLocations") {
			customLocationName := azureshared.ExtractResourceName(customLocationID)
			if customLocationName != "" {
				linkedScope := azureshared.ExtractScopeFromResourceID(customLocationID)
				if linkedScope == "" {
					linkedScope = scope
				}
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   azureshared.ExtendedLocationCustomLocation.String(),
						Method: sdp.QueryMethod_GET,
						Query:  customLocationName,
						Scope:  linkedScope,
					},
				})
			}
		}
	}

	if pls.Properties != nil {
		// Link to IPConfigurations[].Properties.Subnet and PrivateIPAddress
		if pls.Properties.IPConfigurations != nil {
			for _, ipConfig := range pls.Properties.IPConfigurations {
				if ipConfig == nil || ipConfig.Properties == nil {
					continue
				}
				// Link to Subnet and VirtualNetwork
				if ipConfig.Properties.Subnet != nil && ipConfig.Properties.Subnet.ID != nil {
					subnetParams := azureshared.ExtractPathParamsFromResourceID(*ipConfig.Properties.Subnet.ID, []string{"virtualNetworks", "subnets"})
					if len(subnetParams) >= 2 {
						vnetName, subnetName := subnetParams[0], subnetParams[1]
						linkedScope := azureshared.ExtractScopeFromResourceID(*ipConfig.Properties.Subnet.ID)
						if linkedScope == "" {
							linkedScope = scope
						}
						sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   azureshared.NetworkSubnet.String(),
								Method: sdp.QueryMethod_GET,
								Query:  shared.CompositeLookupKey(vnetName, subnetName),
								Scope:  linkedScope,
							},
						})
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
				// Link to PrivateIPAddress
				if ipConfig.Properties.PrivateIPAddress != nil && *ipConfig.Properties.PrivateIPAddress != "" {
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   stdlib.NetworkIP.String(),
							Method: sdp.QueryMethod_GET,
							Query:  *ipConfig.Properties.PrivateIPAddress,
							Scope:  "global",
						},
					})
				}
			}
		}

		// Link to LoadBalancerFrontendIPConfigurations
		if pls.Properties.LoadBalancerFrontendIPConfigurations != nil {
			for _, lbFrontendIPConfig := range pls.Properties.LoadBalancerFrontendIPConfigurations {
				if lbFrontendIPConfig == nil || lbFrontendIPConfig.ID == nil {
					continue
				}
				params := azureshared.ExtractPathParamsFromResourceID(*lbFrontendIPConfig.ID, []string{"loadBalancers", "frontendIPConfigurations"})
				if len(params) >= 2 {
					lbName, frontendIPConfigName := params[0], params[1]
					linkedScope := azureshared.ExtractScopeFromResourceID(*lbFrontendIPConfig.ID)
					if linkedScope == "" {
						linkedScope = scope
					}
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.NetworkLoadBalancerFrontendIPConfiguration.String(),
							Method: sdp.QueryMethod_GET,
							Query:  shared.CompositeLookupKey(lbName, frontendIPConfigName),
							Scope:  linkedScope,
						},
					})
					// Also link to the parent LoadBalancer
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.NetworkLoadBalancer.String(),
							Method: sdp.QueryMethod_GET,
							Query:  lbName,
							Scope:  linkedScope,
						},
					})
				}
			}
		}

		// Link to NetworkInterfaces (read-only array)
		if pls.Properties.NetworkInterfaces != nil {
			for _, iface := range pls.Properties.NetworkInterfaces {
				if iface == nil || iface.ID == nil {
					continue
				}
				nicName := azureshared.ExtractResourceName(*iface.ID)
				if nicName != "" {
					linkedScope := azureshared.ExtractScopeFromResourceID(*iface.ID)
					if linkedScope == "" {
						linkedScope = scope
					}
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.NetworkNetworkInterface.String(),
							Method: sdp.QueryMethod_GET,
							Query:  nicName,
							Scope:  linkedScope,
						},
					})
				}
			}
		}

		// Link to PrivateEndpointConnections[].PrivateEndpoint
		if pls.Properties.PrivateEndpointConnections != nil {
			for _, peConn := range pls.Properties.PrivateEndpointConnections {
				if peConn == nil || peConn.Properties == nil || peConn.Properties.PrivateEndpoint == nil || peConn.Properties.PrivateEndpoint.ID == nil {
					continue
				}
				peName := azureshared.ExtractResourceName(*peConn.Properties.PrivateEndpoint.ID)
				if peName != "" {
					linkedScope := azureshared.ExtractScopeFromResourceID(*peConn.Properties.PrivateEndpoint.ID)
					if linkedScope == "" {
						linkedScope = scope
					}
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.NetworkPrivateEndpoint.String(),
							Method: sdp.QueryMethod_GET,
							Query:  peName,
							Scope:  linkedScope,
						},
					})
				}
			}
		}

		// Link to Fqdns as DNS names
		if pls.Properties.Fqdns != nil {
			for _, fqdn := range pls.Properties.Fqdns {
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

		// Link to DestinationIPAddress
		if pls.Properties.DestinationIPAddress != nil && *pls.Properties.DestinationIPAddress != "" {
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   stdlib.NetworkIP.String(),
					Method: sdp.QueryMethod_GET,
					Query:  *pls.Properties.DestinationIPAddress,
					Scope:  "global",
				},
			})
		}

		// Link to Alias (read-only DNS-resolvable name for the private link service)
		if pls.Properties.Alias != nil && *pls.Properties.Alias != "" {
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   stdlib.NetworkDNS.String(),
					Method: sdp.QueryMethod_SEARCH,
					Query:  *pls.Properties.Alias,
					Scope:  "global",
				},
			})
		}
	}

	return sdpItem, nil
}

func (n networkPrivateLinkServiceWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		NetworkPrivateLinkServiceLookupByName,
	}
}

func (n networkPrivateLinkServiceWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		azureshared.NetworkSubnet,
		azureshared.NetworkVirtualNetwork,
		azureshared.NetworkLoadBalancerFrontendIPConfiguration,
		azureshared.NetworkLoadBalancer,
		azureshared.NetworkNetworkInterface,
		azureshared.NetworkPrivateEndpoint,
		azureshared.ExtendedLocationCustomLocation,
		stdlib.NetworkIP,
		stdlib.NetworkDNS,
	)
}

// ref: https://learn.microsoft.com/en-us/azure/role-based-access-control/resource-provider-operations#microsoftnetwork
func (n networkPrivateLinkServiceWrapper) IAMPermissions() []string {
	return []string{
		"Microsoft.Network/privateLinkServices/read",
	}
}

func (n networkPrivateLinkServiceWrapper) PredefinedRole() string {
	return "Network Contributor"
}
