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

var NetworkVirtualNetworkGatewayConnectionLookupByName = shared.NewItemTypeLookup("name", azureshared.NetworkVirtualNetworkGatewayConnection)

type networkVirtualNetworkGatewayConnectionWrapper struct {
	client clients.VirtualNetworkGatewayConnectionsClient

	*azureshared.MultiResourceGroupBase
}

// NewNetworkVirtualNetworkGatewayConnection creates a new networkVirtualNetworkGatewayConnectionWrapper instance.
func NewNetworkVirtualNetworkGatewayConnection(client clients.VirtualNetworkGatewayConnectionsClient, resourceGroupScopes []azureshared.ResourceGroupScope) sources.ListableWrapper {
	return &networkVirtualNetworkGatewayConnectionWrapper{
		client: client,
		MultiResourceGroupBase: azureshared.NewMultiResourceGroupBase(
			resourceGroupScopes,
			sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
			azureshared.NetworkVirtualNetworkGatewayConnection,
		),
	}
}

// ref: https://learn.microsoft.com/en-us/rest/api/network-gateway/virtual-network-gateway-connections/list
func (c networkVirtualNetworkGatewayConnectionWrapper) List(ctx context.Context, scope string) ([]*sdp.Item, *sdp.QueryError) {
	rgScope, err := c.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}
	pager := c.client.NewListPager(rgScope.ResourceGroup, nil)

	var items []*sdp.Item
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, azureshared.QueryError(err, scope, c.Type())
		}
		for _, conn := range page.Value {
			if conn.Name == nil {
				continue
			}
			item, sdpErr := c.azureVirtualNetworkGatewayConnectionToSDPItem(conn, scope)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}

	return items, nil
}

func (c networkVirtualNetworkGatewayConnectionWrapper) ListStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string) {
	rgScope, err := c.ResourceGroupScopeFromScope(scope)
	if err != nil {
		stream.SendError(azureshared.QueryError(err, scope, c.Type()))
		return
	}
	pager := c.client.NewListPager(rgScope.ResourceGroup, nil)

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			stream.SendError(azureshared.QueryError(err, scope, c.Type()))
			return
		}

		for _, conn := range page.Value {
			if conn.Name == nil {
				continue
			}
			item, sdpErr := c.azureVirtualNetworkGatewayConnectionToSDPItem(conn, scope)
			if sdpErr != nil {
				stream.SendError(sdpErr)
				continue
			}
			cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
			stream.SendItem(item)
		}
	}
}

// ref: https://learn.microsoft.com/en-us/rest/api/network-gateway/virtual-network-gateway-connections/get
func (c networkVirtualNetworkGatewayConnectionWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 1 {
		return nil, azureshared.QueryError(errors.New("queryParts must be at least 1 and be the connection name"), scope, c.Type())
	}
	connectionName := queryParts[0]
	if connectionName == "" {
		return nil, azureshared.QueryError(errors.New("connectionName cannot be empty"), scope, c.Type())
	}

	rgScope, err := c.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}
	result, err := c.client.Get(ctx, rgScope.ResourceGroup, connectionName, nil)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}
	return c.azureVirtualNetworkGatewayConnectionToSDPItem(&result.VirtualNetworkGatewayConnection, scope)
}

func (c networkVirtualNetworkGatewayConnectionWrapper) azureVirtualNetworkGatewayConnectionToSDPItem(conn *armnetwork.VirtualNetworkGatewayConnection, scope string) (*sdp.Item, *sdp.QueryError) {
	if conn.Name == nil {
		return nil, azureshared.QueryError(errors.New("connection name is nil"), scope, c.Type())
	}

	attributes, err := shared.ToAttributesWithExclude(conn, "tags")
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}

	sdpItem := &sdp.Item{
		Type:              azureshared.NetworkVirtualNetworkGatewayConnection.String(),
		UniqueAttribute:   "name",
		Attributes:        attributes,
		Scope:             scope,
		Tags:              azureshared.ConvertAzureTags(conn.Tags),
		LinkedItemQueries: []*sdp.LinkedItemQuery{},
	}

	// Health from provisioning state
	if conn.Properties != nil && conn.Properties.ProvisioningState != nil {
		switch *conn.Properties.ProvisioningState {
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

	if conn.Properties == nil {
		return sdpItem, nil
	}

	// VirtualNetworkGateway1 (required)
	if conn.Properties.VirtualNetworkGateway1 != nil && conn.Properties.VirtualNetworkGateway1.ID != nil {
		gwID := *conn.Properties.VirtualNetworkGateway1.ID
		gwName := azureshared.ExtractResourceName(gwID)
		if gwName != "" {
			linkedScope := azureshared.ExtractScopeFromResourceID(gwID)
			if linkedScope == "" {
				linkedScope = scope
			}
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   azureshared.NetworkVirtualNetworkGateway.String(),
					Method: sdp.QueryMethod_GET,
					Query:  gwName,
					Scope:  linkedScope,
				},
			})
		}
	}

	// VirtualNetworkGateway2 (optional - for VNet-to-VNet connections)
	if conn.Properties.VirtualNetworkGateway2 != nil && conn.Properties.VirtualNetworkGateway2.ID != nil {
		gw2ID := *conn.Properties.VirtualNetworkGateway2.ID
		gw2Name := azureshared.ExtractResourceName(gw2ID)
		if gw2Name != "" {
			linkedScope := azureshared.ExtractScopeFromResourceID(gw2ID)
			if linkedScope == "" {
				linkedScope = scope
			}
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   azureshared.NetworkVirtualNetworkGateway.String(),
					Method: sdp.QueryMethod_GET,
					Query:  gw2Name,
					Scope:  linkedScope,
				},
			})
		}
	}

	// LocalNetworkGateway2 (optional - for Site-to-Site connections)
	if conn.Properties.LocalNetworkGateway2 != nil && conn.Properties.LocalNetworkGateway2.ID != nil {
		lgwID := *conn.Properties.LocalNetworkGateway2.ID
		lgwName := azureshared.ExtractResourceName(lgwID)
		if lgwName != "" {
			linkedScope := azureshared.ExtractScopeFromResourceID(lgwID)
			if linkedScope == "" {
				linkedScope = scope
			}
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   azureshared.NetworkLocalNetworkGateway.String(),
					Method: sdp.QueryMethod_GET,
					Query:  lgwName,
					Scope:  linkedScope,
				},
			})
		}
	}

	// Peer (ExpressRoute circuit peering)
	// Path: expressRouteCircuits/{circuitName}/peerings/{peeringName}
	if conn.Properties.Peer != nil && conn.Properties.Peer.ID != nil {
		peerID := *conn.Properties.Peer.ID
		params := azureshared.ExtractPathParamsFromResourceID(peerID, []string{"expressRouteCircuits", "peerings"})
		if len(params) >= 2 && params[0] != "" && params[1] != "" {
			linkedScope := azureshared.ExtractScopeFromResourceID(peerID)
			if linkedScope == "" {
				linkedScope = scope
			}
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   azureshared.NetworkExpressRouteCircuitPeering.String(),
					Method: sdp.QueryMethod_GET,
					Query:  shared.CompositeLookupKey(params[0], params[1]),
					Scope:  linkedScope,
				},
			})
		}
	}

	// EgressNatRules (NAT rules for outbound traffic)
	// Path: virtualNetworkGateways/{gwName}/natRules/{ruleName}
	if conn.Properties.EgressNatRules != nil {
		for _, natRule := range conn.Properties.EgressNatRules {
			if natRule != nil && natRule.ID != nil {
				natRuleID := *natRule.ID
				params := azureshared.ExtractPathParamsFromResourceID(natRuleID, []string{"virtualNetworkGateways", "natRules"})
				if len(params) >= 2 && params[0] != "" && params[1] != "" {
					linkedScope := azureshared.ExtractScopeFromResourceID(natRuleID)
					if linkedScope == "" {
						linkedScope = scope
					}
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.NetworkVirtualNetworkGatewayNatRule.String(),
							Method: sdp.QueryMethod_GET,
							Query:  shared.CompositeLookupKey(params[0], params[1]),
							Scope:  linkedScope,
						},
					})
				}
			}
		}
	}

	// IngressNatRules (NAT rules for inbound traffic)
	// Path: virtualNetworkGateways/{gwName}/natRules/{ruleName}
	if conn.Properties.IngressNatRules != nil {
		for _, natRule := range conn.Properties.IngressNatRules {
			if natRule != nil && natRule.ID != nil {
				natRuleID := *natRule.ID
				params := azureshared.ExtractPathParamsFromResourceID(natRuleID, []string{"virtualNetworkGateways", "natRules"})
				if len(params) >= 2 && params[0] != "" && params[1] != "" {
					linkedScope := azureshared.ExtractScopeFromResourceID(natRuleID)
					if linkedScope == "" {
						linkedScope = scope
					}
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.NetworkVirtualNetworkGatewayNatRule.String(),
							Method: sdp.QueryMethod_GET,
							Query:  shared.CompositeLookupKey(params[0], params[1]),
							Scope:  linkedScope,
						},
					})
				}
			}
		}
	}

	// GatewayCustomBgpIPAddresses - link to custom BGP IP addresses and IP configurations
	if conn.Properties.GatewayCustomBgpIPAddresses != nil {
		for _, bgpConfig := range conn.Properties.GatewayCustomBgpIPAddresses {
			if bgpConfig == nil {
				continue
			}
			// Custom BGP IP address
			if bgpConfig.CustomBgpIPAddress != nil && *bgpConfig.CustomBgpIPAddress != "" {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   stdlib.NetworkIP.String(),
						Method: sdp.QueryMethod_GET,
						Query:  *bgpConfig.CustomBgpIPAddress,
						Scope:  "global",
					},
				})
			}
			// IPConfigurationID - reference to VirtualNetworkGateway IP configuration
			if bgpConfig.IPConfigurationID != nil && *bgpConfig.IPConfigurationID != "" {
				ipConfigID := *bgpConfig.IPConfigurationID
				params := azureshared.ExtractPathParamsFromResourceID(ipConfigID, []string{"virtualNetworkGateways", "ipConfigurations"})
				if len(params) >= 2 && params[0] != "" && params[1] != "" {
					linkedScope := azureshared.ExtractScopeFromResourceID(ipConfigID)
					if linkedScope == "" {
						linkedScope = scope
					}
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.NetworkVirtualNetworkGatewayIPConfiguration.String(),
							Method: sdp.QueryMethod_GET,
							Query:  shared.CompositeLookupKey(params[0], params[1]),
							Scope:  linkedScope,
						},
					})
				}
			}
		}
	}

	// TunnelProperties - tunnel IP addresses and BGP peering addresses
	if conn.Properties.TunnelProperties != nil {
		for _, tunnel := range conn.Properties.TunnelProperties {
			if tunnel == nil {
				continue
			}
			// Tunnel IP address
			if tunnel.TunnelIPAddress != nil && *tunnel.TunnelIPAddress != "" {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   stdlib.NetworkIP.String(),
						Method: sdp.QueryMethod_GET,
						Query:  *tunnel.TunnelIPAddress,
						Scope:  "global",
					},
				})
			}
			// BGP peering address
			if tunnel.BgpPeeringAddress != nil && *tunnel.BgpPeeringAddress != "" {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   stdlib.NetworkIP.String(),
						Method: sdp.QueryMethod_GET,
						Query:  *tunnel.BgpPeeringAddress,
						Scope:  "global",
					},
				})
			}
		}
	}

	return sdpItem, nil
}

func (c networkVirtualNetworkGatewayConnectionWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		NetworkVirtualNetworkGatewayConnectionLookupByName,
	}
}

func (c networkVirtualNetworkGatewayConnectionWrapper) PotentialLinks() map[shared.ItemType]bool {
	return map[shared.ItemType]bool{
		azureshared.NetworkVirtualNetworkGateway:                true,
		azureshared.NetworkLocalNetworkGateway:                  true,
		azureshared.NetworkExpressRouteCircuitPeering:           true,
		azureshared.NetworkVirtualNetworkGatewayNatRule:         true,
		azureshared.NetworkVirtualNetworkGatewayIPConfiguration: true,
		stdlib.NetworkIP: true,
	}
}

// ref: https://learn.microsoft.com/en-us/azure/role-based-access-control/resource-provider-operations#microsoftnetwork
func (c networkVirtualNetworkGatewayConnectionWrapper) IAMPermissions() []string {
	return []string{
		"Microsoft.Network/connections/read",
	}
}

func (c networkVirtualNetworkGatewayConnectionWrapper) PredefinedRole() string {
	return "Reader"
}
