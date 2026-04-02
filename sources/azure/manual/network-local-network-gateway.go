package manual

import (
	"context"
	"errors"
	"net"

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

var NetworkLocalNetworkGatewayLookupByName = shared.NewItemTypeLookup("name", azureshared.NetworkLocalNetworkGateway)

type networkLocalNetworkGatewayWrapper struct {
	client clients.LocalNetworkGatewaysClient

	*azureshared.MultiResourceGroupBase
}

// NewNetworkLocalNetworkGateway creates a new networkLocalNetworkGatewayWrapper instance.
func NewNetworkLocalNetworkGateway(client clients.LocalNetworkGatewaysClient, resourceGroupScopes []azureshared.ResourceGroupScope) sources.ListableWrapper {
	return &networkLocalNetworkGatewayWrapper{
		client: client,
		MultiResourceGroupBase: azureshared.NewMultiResourceGroupBase(
			resourceGroupScopes,
			sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
			azureshared.NetworkLocalNetworkGateway,
		),
	}
}

// List retrieves all local network gateways in a scope.
// ref: https://learn.microsoft.com/en-us/rest/api/network-gateway/local-network-gateways/list
func (c networkLocalNetworkGatewayWrapper) List(ctx context.Context, scope string) ([]*sdp.Item, *sdp.QueryError) {
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
		for _, gw := range page.Value {
			if gw.Name == nil {
				continue
			}
			item, sdpErr := c.azureLocalNetworkGatewayToSDPItem(gw, scope)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}

	return items, nil
}

// ListStream streams all local network gateways in a scope.
func (c networkLocalNetworkGatewayWrapper) ListStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string) {
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

		for _, gw := range page.Value {
			if gw.Name == nil {
				continue
			}
			item, sdpErr := c.azureLocalNetworkGatewayToSDPItem(gw, scope)
			if sdpErr != nil {
				stream.SendError(sdpErr)
				continue
			}
			cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
			stream.SendItem(item)
		}
	}
}

// Get retrieves a single local network gateway by name.
// ref: https://learn.microsoft.com/en-us/rest/api/network-gateway/local-network-gateways/get
func (c networkLocalNetworkGatewayWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 1 {
		return nil, azureshared.QueryError(errors.New("queryParts must be at least 1 and be the local network gateway name"), scope, c.Type())
	}
	gatewayName := queryParts[0]
	if gatewayName == "" {
		return nil, azureshared.QueryError(errors.New("localNetworkGatewayName cannot be empty"), scope, c.Type())
	}

	rgScope, err := c.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}

	result, err := c.client.Get(ctx, rgScope.ResourceGroup, gatewayName, nil)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}

	return c.azureLocalNetworkGatewayToSDPItem(&result.LocalNetworkGateway, scope)
}

func (c networkLocalNetworkGatewayWrapper) azureLocalNetworkGatewayToSDPItem(gw *armnetwork.LocalNetworkGateway, scope string) (*sdp.Item, *sdp.QueryError) {
	if gw.Name == nil {
		return nil, azureshared.QueryError(errors.New("local network gateway name is nil"), scope, c.Type())
	}

	attributes, err := shared.ToAttributesWithExclude(gw, "tags")
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}

	sdpItem := &sdp.Item{
		Type:              azureshared.NetworkLocalNetworkGateway.String(),
		UniqueAttribute:   "name",
		Attributes:        attributes,
		Scope:             scope,
		Tags:              azureshared.ConvertAzureTags(gw.Tags),
		LinkedItemQueries: []*sdp.LinkedItemQuery{},
	}

	// Health from provisioning state
	if gw.Properties != nil && gw.Properties.ProvisioningState != nil {
		switch *gw.Properties.ProvisioningState {
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

	// Gateway IP address (on-premises VPN device IP)
	if gw.Properties != nil && gw.Properties.GatewayIPAddress != nil && *gw.Properties.GatewayIPAddress != "" {
		sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   stdlib.NetworkIP.String(),
				Method: sdp.QueryMethod_GET,
				Query:  *gw.Properties.GatewayIPAddress,
				Scope:  "global",
			},
		})
	}

	// FQDN (if used instead of IP address for the on-premises device)
	if gw.Properties != nil && gw.Properties.Fqdn != nil && *gw.Properties.Fqdn != "" {
		sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   stdlib.NetworkDNS.String(),
				Method: sdp.QueryMethod_SEARCH,
				Query:  *gw.Properties.Fqdn,
				Scope:  "global",
			},
		})
	}

	// BGP settings
	if gw.Properties != nil && gw.Properties.BgpSettings != nil {
		bgp := gw.Properties.BgpSettings

		// BgpPeeringAddress - can be IP or hostname
		if bgp.BgpPeeringAddress != nil && *bgp.BgpPeeringAddress != "" {
			if net.ParseIP(*bgp.BgpPeeringAddress) != nil {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   stdlib.NetworkIP.String(),
						Method: sdp.QueryMethod_GET,
						Query:  *bgp.BgpPeeringAddress,
						Scope:  "global",
					},
				})
			} else {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   stdlib.NetworkDNS.String(),
						Method: sdp.QueryMethod_SEARCH,
						Query:  *bgp.BgpPeeringAddress,
						Scope:  "global",
					},
				})
			}
		}

		// BgpPeeringAddresses array
		if bgp.BgpPeeringAddresses != nil {
			for _, peeringAddr := range bgp.BgpPeeringAddresses {
				if peeringAddr == nil {
					continue
				}
				// DefaultBgpIPAddresses
				for _, ipStr := range peeringAddr.DefaultBgpIPAddresses {
					if ipStr != nil && *ipStr != "" {
						sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   stdlib.NetworkIP.String(),
								Method: sdp.QueryMethod_GET,
								Query:  *ipStr,
								Scope:  "global",
							},
						})
					}
				}
				// CustomBgpIPAddresses
				for _, ipStr := range peeringAddr.CustomBgpIPAddresses {
					if ipStr != nil && *ipStr != "" {
						sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   stdlib.NetworkIP.String(),
								Method: sdp.QueryMethod_GET,
								Query:  *ipStr,
								Scope:  "global",
							},
						})
					}
				}
				// TunnelIPAddresses
				for _, ipStr := range peeringAddr.TunnelIPAddresses {
					if ipStr != nil && *ipStr != "" {
						sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   stdlib.NetworkIP.String(),
								Method: sdp.QueryMethod_GET,
								Query:  *ipStr,
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

func (c networkLocalNetworkGatewayWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		NetworkLocalNetworkGatewayLookupByName,
	}
}

func (c networkLocalNetworkGatewayWrapper) PotentialLinks() map[shared.ItemType]bool {
	return map[shared.ItemType]bool{
		stdlib.NetworkIP:  true,
		stdlib.NetworkDNS: true,
	}
}

// IAMPermissions returns the Azure RBAC permissions required to read this resource.
// ref: https://learn.microsoft.com/en-us/azure/role-based-access-control/resource-provider-operations#microsoftnetwork
func (c networkLocalNetworkGatewayWrapper) IAMPermissions() []string {
	return []string{
		"Microsoft.Network/localNetworkGateways/read",
	}
}

// PredefinedRole returns the Azure built-in role that grants the required permissions.
func (c networkLocalNetworkGatewayWrapper) PredefinedRole() string {
	return "Reader"
}
