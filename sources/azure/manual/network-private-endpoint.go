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

var NetworkPrivateEndpointLookupByName = shared.NewItemTypeLookup("name", azureshared.NetworkPrivateEndpoint)

type networkPrivateEndpointWrapper struct {
	client clients.PrivateEndpointsClient

	*azureshared.MultiResourceGroupBase
}

func NewNetworkPrivateEndpoint(client clients.PrivateEndpointsClient, resourceGroupScopes []azureshared.ResourceGroupScope) sources.ListableWrapper {
	return &networkPrivateEndpointWrapper{
		client: client,
		MultiResourceGroupBase: azureshared.NewMultiResourceGroupBase(
			resourceGroupScopes,
			sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
			azureshared.NetworkPrivateEndpoint,
		),
	}
}

func (n networkPrivateEndpointWrapper) List(ctx context.Context, scope string) ([]*sdp.Item, *sdp.QueryError) {
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
		for _, pe := range page.Value {
			if pe.Name == nil {
				continue
			}
			item, sdpErr := n.azurePrivateEndpointToSDPItem(pe, scope)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}
	return items, nil
}

func (n networkPrivateEndpointWrapper) ListStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string) {
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
		for _, pe := range page.Value {
			if pe.Name == nil {
				continue
			}
			item, sdpErr := n.azurePrivateEndpointToSDPItem(pe, scope)
			if sdpErr != nil {
				stream.SendError(sdpErr)
				continue
			}
			cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
			stream.SendItem(item)
		}
	}
}

func (n networkPrivateEndpointWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) != 1 {
		return nil, azureshared.QueryError(errors.New("query must be a private endpoint name"), scope, n.Type())
	}
	name := queryParts[0]
	if name == "" {
		return nil, azureshared.QueryError(errors.New("private endpoint name cannot be empty"), scope, n.Type())
	}

	rgScope, err := n.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, n.Type())
	}
	resp, err := n.client.Get(ctx, rgScope.ResourceGroup, name)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, n.Type())
	}
	return n.azurePrivateEndpointToSDPItem(&resp.PrivateEndpoint, scope)
}

func (n networkPrivateEndpointWrapper) azurePrivateEndpointToSDPItem(pe *armnetwork.PrivateEndpoint, scope string) (*sdp.Item, *sdp.QueryError) {
	if pe.Name == nil {
		return nil, azureshared.QueryError(errors.New("private endpoint name is nil"), scope, n.Type())
	}
	attributes, err := shared.ToAttributesWithExclude(pe, "tags")
	if err != nil {
		return nil, azureshared.QueryError(err, scope, n.Type())
	}

	sdpItem := &sdp.Item{
		Type:            azureshared.NetworkPrivateEndpoint.String(),
		UniqueAttribute: "name",
		Attributes:      attributes,
		Scope:           scope,
		Tags:            azureshared.ConvertAzureTags(pe.Tags),
	}

	// Health status from ProvisioningState
	if pe.Properties != nil && pe.Properties.ProvisioningState != nil {
		switch *pe.Properties.ProvisioningState {
		case armnetwork.ProvisioningStateSucceeded:
			sdpItem.Health = sdp.Health_HEALTH_OK.Enum()
		case armnetwork.ProvisioningStateCreating, armnetwork.ProvisioningStateUpdating, armnetwork.ProvisioningStateDeleting:
			sdpItem.Health = sdp.Health_HEALTH_PENDING.Enum()
		case armnetwork.ProvisioningStateFailed, armnetwork.ProvisioningStateCanceled:
			sdpItem.Health = sdp.Health_HEALTH_ERROR.Enum()
		}
	}

	// Link to Subnet and parent VirtualNetwork
	if pe.Properties != nil && pe.Properties.Subnet != nil && pe.Properties.Subnet.ID != nil {
		subnetParams := azureshared.ExtractPathParamsFromResourceID(*pe.Properties.Subnet.ID, []string{"virtualNetworks", "subnets"})
		if len(subnetParams) >= 2 {
			vnetName, subnetName := subnetParams[0], subnetParams[1]
			linkedScope := azureshared.ExtractScopeFromResourceID(*pe.Properties.Subnet.ID)
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

	// Link to NetworkInterfaces (read-only array of NICs created for this private endpoint)
	if pe.Properties != nil && pe.Properties.NetworkInterfaces != nil {
		for _, iface := range pe.Properties.NetworkInterfaces {
			if iface != nil && iface.ID != nil {
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
	}

	// Link to ApplicationSecurityGroups
	if pe.Properties != nil && pe.Properties.ApplicationSecurityGroups != nil {
		for _, asg := range pe.Properties.ApplicationSecurityGroups {
			if asg != nil && asg.ID != nil {
				asgName := azureshared.ExtractResourceName(*asg.ID)
				if asgName != "" {
					linkedScope := azureshared.ExtractScopeFromResourceID(*asg.ID)
					if linkedScope == "" {
						linkedScope = scope
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

	// Link IPConfigurations[].Properties.PrivateIPAddress to stdlib ip (GET, global)
	if pe.Properties != nil && pe.Properties.IPConfigurations != nil {
		for _, ipConfig := range pe.Properties.IPConfigurations {
			if ipConfig == nil || ipConfig.Properties == nil || ipConfig.Properties.PrivateIPAddress == nil {
				continue
			}
			if *ipConfig.Properties.PrivateIPAddress != "" {
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

	// Link to Private Link Services from PrivateLinkServiceConnections and ManualPrivateLinkServiceConnections
	if pe.Properties != nil {
		seenPLS := make(map[string]struct{})
		for _, conns := range [][]*armnetwork.PrivateLinkServiceConnection{
			pe.Properties.PrivateLinkServiceConnections,
			pe.Properties.ManualPrivateLinkServiceConnections,
		} {
			for _, conn := range conns {
				if conn == nil || conn.Properties == nil || conn.Properties.PrivateLinkServiceID == nil {
					continue
				}
				plsID := *conn.Properties.PrivateLinkServiceID
				if plsID == "" {
					continue
				}
				if _, ok := seenPLS[plsID]; ok {
					continue
				}
				seenPLS[plsID] = struct{}{}
				plsName := azureshared.ExtractResourceName(plsID)
				if plsName == "" {
					continue
				}
				linkedScope := azureshared.ExtractScopeFromResourceID(plsID)
				if linkedScope == "" {
					linkedScope = scope
				}
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   azureshared.NetworkPrivateLinkService.String(),
						Method: sdp.QueryMethod_GET,
						Query:  plsName,
						Scope:  linkedScope,
					},
				})
			}
		}
	}

	// Link CustomDnsConfigs: Fqdn -> stdlib dns (SEARCH, global), IPAddresses -> stdlib ip (GET, global)
	if pe.Properties != nil && pe.Properties.CustomDNSConfigs != nil {
		for _, dnsConfig := range pe.Properties.CustomDNSConfigs {
			if dnsConfig == nil {
				continue
			}
			if dnsConfig.Fqdn != nil && *dnsConfig.Fqdn != "" {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   stdlib.NetworkDNS.String(),
						Method: sdp.QueryMethod_SEARCH,
						Query:  *dnsConfig.Fqdn,
						Scope:  "global",
					},
				})
			}
			if dnsConfig.IPAddresses != nil {
				for _, ip := range dnsConfig.IPAddresses {
					if ip != nil && *ip != "" {
						sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   stdlib.NetworkIP.String(),
								Method: sdp.QueryMethod_GET,
								Query:  *ip,
								Scope:  "global",
							},
						})
					}
				}
			}
		}
	}

	return sdpItem, nil
}

func (n networkPrivateEndpointWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		NetworkPrivateEndpointLookupByName,
	}
}

func (n networkPrivateEndpointWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		azureshared.NetworkSubnet,
		azureshared.NetworkVirtualNetwork,
		azureshared.NetworkNetworkInterface,
		azureshared.NetworkApplicationSecurityGroup,
		azureshared.NetworkPrivateLinkService,
		stdlib.NetworkIP,
		stdlib.NetworkDNS,
	)
}

// ref: https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/private_endpoint
func (n networkPrivateEndpointWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "azurerm_private_endpoint.name",
		},
	}
}

// ref: https://learn.microsoft.com/en-us/azure/role-based-access-control/permissions-reference#microsoftnetwork
func (n networkPrivateEndpointWrapper) IAMPermissions() []string {
	return []string{
		"Microsoft.Network/privateEndpoints/read",
	}
}

func (n networkPrivateEndpointWrapper) PredefinedRole() string {
	return "Network Contributor"
}
