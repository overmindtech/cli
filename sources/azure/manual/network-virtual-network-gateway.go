package manual

import (
	"context"
	"errors"
	"net"
	"net/url"
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

var NetworkVirtualNetworkGatewayLookupByName = shared.NewItemTypeLookup("name", azureshared.NetworkVirtualNetworkGateway)

type networkVirtualNetworkGatewayWrapper struct {
	client clients.VirtualNetworkGatewaysClient

	*azureshared.MultiResourceGroupBase
}

// NewNetworkVirtualNetworkGateway creates a new networkVirtualNetworkGatewayWrapper instance.
func NewNetworkVirtualNetworkGateway(client clients.VirtualNetworkGatewaysClient, resourceGroupScopes []azureshared.ResourceGroupScope) sources.ListableWrapper {
	return &networkVirtualNetworkGatewayWrapper{
		client: client,
		MultiResourceGroupBase: azureshared.NewMultiResourceGroupBase(
			resourceGroupScopes,
			sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
			azureshared.NetworkVirtualNetworkGateway,
		),
	}
}

func (n networkVirtualNetworkGatewayWrapper) List(ctx context.Context, scope string) ([]*sdp.Item, *sdp.QueryError) {
	rgScope, err := n.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, n.Type())
	}
	pager := n.client.NewListPager(rgScope.ResourceGroup, nil)

	var items []*sdp.Item
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, azureshared.QueryError(err, scope, n.Type())
		}

		for _, gw := range page.Value {
			if gw.Name == nil {
				continue
			}
			item, sdpErr := n.azureVirtualNetworkGatewayToSDPItem(gw, scope)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}

	return items, nil
}

func (n networkVirtualNetworkGatewayWrapper) ListStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string) {
	rgScope, err := n.ResourceGroupScopeFromScope(scope)
	if err != nil {
		stream.SendError(azureshared.QueryError(err, scope, n.Type()))
		return
	}
	pager := n.client.NewListPager(rgScope.ResourceGroup, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			stream.SendError(azureshared.QueryError(err, scope, n.Type()))
			return
		}

		for _, gw := range page.Value {
			if gw.Name == nil {
				continue
			}
			item, sdpErr := n.azureVirtualNetworkGatewayToSDPItem(gw, scope)
			if sdpErr != nil {
				stream.SendError(sdpErr)
				continue
			}
			cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
			stream.SendItem(item)
		}
	}
}

func (n networkVirtualNetworkGatewayWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 1 {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "Get requires 1 query part: virtualNetworkGatewayName",
			Scope:       scope,
			ItemType:    n.Type(),
		}
	}

	gatewayName := queryParts[0]

	rgScope, err := n.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, n.Type())
	}
	resp, err := n.client.Get(ctx, rgScope.ResourceGroup, gatewayName, nil)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, n.Type())
	}

	return n.azureVirtualNetworkGatewayToSDPItem(&resp.VirtualNetworkGateway, scope)
}

func (n networkVirtualNetworkGatewayWrapper) azureVirtualNetworkGatewayToSDPItem(gw *armnetwork.VirtualNetworkGateway, scope string) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(gw, "tags")
	if err != nil {
		return nil, azureshared.QueryError(err, scope, n.Type())
	}

	if gw.Name == nil {
		return nil, azureshared.QueryError(errors.New("virtual network gateway name is nil"), scope, n.Type())
	}

	sdpItem := &sdp.Item{
		Type:               azureshared.NetworkVirtualNetworkGateway.String(),
		UniqueAttribute:    "name",
		Attributes:         attributes,
		Scope:              scope,
		Tags:               azureshared.ConvertAzureTags(gw.Tags),
		LinkedItemQueries:  []*sdp.LinkedItemQuery{},
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

	// Link from IP configurations: subnet, public IP, private IP
	if gw.Properties != nil && gw.Properties.IPConfigurations != nil {
		for _, ipConfig := range gw.Properties.IPConfigurations {
			if ipConfig == nil || ipConfig.Properties == nil {
				continue
			}

			// Subnet (SearchableWrapper: virtualNetworks/{vnet}/subnets/{subnet})
			if ipConfig.Properties.Subnet != nil && ipConfig.Properties.Subnet.ID != nil {
				subnetID := *ipConfig.Properties.Subnet.ID
				params := azureshared.ExtractPathParamsFromResourceID(subnetID, []string{"virtualNetworks", "subnets"})
				if len(params) >= 2 && params[0] != "" && params[1] != "" {
					linkedScope := azureshared.ExtractScopeFromResourceID(subnetID)
					if linkedScope == "" {
						linkedScope = scope
					}
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.NetworkSubnet.String(),
							Method: sdp.QueryMethod_GET,
							Scope:  linkedScope,
							Query:  shared.CompositeLookupKey(params[0], params[1]),
						},
					})
				}
			}

			// Public IP address
			if ipConfig.Properties.PublicIPAddress != nil && ipConfig.Properties.PublicIPAddress.ID != nil {
				pubIPID := *ipConfig.Properties.PublicIPAddress.ID
				pubIPName := azureshared.ExtractResourceName(pubIPID)
				if pubIPName != "" {
					linkedScope := azureshared.ExtractScopeFromResourceID(pubIPID)
					if linkedScope == "" {
						linkedScope = scope
					}
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.NetworkPublicIPAddress.String(),
							Method: sdp.QueryMethod_GET,
							Query:  pubIPName,
							Scope:  linkedScope,
						},
					})
				}
			}

			// Private IP address -> stdlib ip
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

	// Inbound DNS forwarding endpoint (read-only IP)
	if gw.Properties != nil && gw.Properties.InboundDNSForwardingEndpoint != nil && *gw.Properties.InboundDNSForwardingEndpoint != "" {
		sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   stdlib.NetworkIP.String(),
				Method: sdp.QueryMethod_GET,
				Query:  *gw.Properties.InboundDNSForwardingEndpoint,
				Scope:  "global",
			},
		})
	}

	// Gateway default site (Local Network Gateway)
	if gw.Properties != nil && gw.Properties.GatewayDefaultSite != nil && gw.Properties.GatewayDefaultSite.ID != nil {
		localGWID := *gw.Properties.GatewayDefaultSite.ID
		localGWName := azureshared.ExtractResourceName(localGWID)
		if localGWName != "" {
			linkedScope := azureshared.ExtractScopeFromResourceID(localGWID)
			if linkedScope == "" {
				linkedScope = scope
			}
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   azureshared.NetworkLocalNetworkGateway.String(),
					Method: sdp.QueryMethod_GET,
					Query:  localGWName,
					Scope:  linkedScope,
				},
			})
		}
	}

	// Extended location (custom location) when Name is a custom location resource ID
	if gw.ExtendedLocation != nil && gw.ExtendedLocation.Name != nil {
		customLocationID := *gw.ExtendedLocation.Name
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

	// User-assigned managed identities (map keys are ARM resource IDs)
	if gw.Identity != nil && gw.Identity.UserAssignedIdentities != nil {
		for identityID := range gw.Identity.UserAssignedIdentities {
			if identityID == "" {
				continue
			}
			identityName := azureshared.ExtractResourceName(identityID)
			if identityName != "" {
				linkedScope := azureshared.ExtractScopeFromResourceID(identityID)
				if linkedScope == "" {
					linkedScope = scope
				}
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   azureshared.ManagedIdentityUserAssignedIdentity.String(),
						Method: sdp.QueryMethod_GET,
						Query:  identityName,
						Scope:  linkedScope,
					},
				})
			}
		}
	}

	// VNet extended location resource (customer VNet when gateway type is local)
	if gw.Properties != nil && gw.Properties.VNetExtendedLocationResourceID != nil && *gw.Properties.VNetExtendedLocationResourceID != "" {
		vnetID := *gw.Properties.VNetExtendedLocationResourceID
		vnetName := azureshared.ExtractResourceName(vnetID)
		if vnetName != "" {
			linkedScope := azureshared.ExtractScopeFromResourceID(vnetID)
			if linkedScope == "" {
				linkedScope = scope
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

	// VPN client configuration: RADIUS server address(es) (IP or DNS)
	if gw.Properties != nil && gw.Properties.VPNClientConfiguration != nil {
		vpnCfg := gw.Properties.VPNClientConfiguration
		if vpnCfg.RadiusServerAddress != nil && *vpnCfg.RadiusServerAddress != "" {
			appendDNSServerLinkIfValid(&sdpItem.LinkedItemQueries, *vpnCfg.RadiusServerAddress)
		}
		if vpnCfg.RadiusServers != nil {
			for _, radiusServer := range vpnCfg.RadiusServers {
				if radiusServer != nil && radiusServer.RadiusServerAddress != nil && *radiusServer.RadiusServerAddress != "" {
					appendDNSServerLinkIfValid(&sdpItem.LinkedItemQueries, *radiusServer.RadiusServerAddress)
				}
			}
		}
		// AAD authentication URLs (e.g. https://login.microsoftonline.com/{tenant}/) — link DNS hostnames
		for _, s := range []*string{vpnCfg.AADTenant, vpnCfg.AADAudience, vpnCfg.AADIssuer} {
			if s == nil || *s == "" {
				continue
			}
			host := extractHostFromURLOrHostname(*s)
			if host == "" {
				continue
			}
			// Skip if it's an IP address; stdlib ip links are added elsewhere for IPs
			if net.ParseIP(host) != nil {
				continue
			}
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   stdlib.NetworkDNS.String(),
					Method: sdp.QueryMethod_SEARCH,
					Query:  host,
					Scope:  "global",
				},
			})
		}
	}

	// BGP settings: peering address and IP arrays
	if gw.Properties != nil && gw.Properties.BgpSettings != nil {
		bgp := gw.Properties.BgpSettings
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
		if bgp.BgpPeeringAddresses != nil {
			for _, peeringAddr := range bgp.BgpPeeringAddresses {
				if peeringAddr == nil {
					continue
				}
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

	// Virtual Network Gateway Connections (child resource; list by parent gateway name)
	if gw.Name != nil && *gw.Name != "" {
		sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   azureshared.NetworkVirtualNetworkGatewayConnection.String(),
				Method: sdp.QueryMethod_SEARCH,
				Scope:  scope,
				Query:  *gw.Name,
			},
		})
	}

	return sdpItem, nil
}

func (n networkVirtualNetworkGatewayWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		NetworkVirtualNetworkGatewayLookupByName,
	}
}

func (n networkVirtualNetworkGatewayWrapper) PotentialLinks() map[shared.ItemType]bool {
	return map[shared.ItemType]bool{
		azureshared.NetworkSubnet:                           true,
		azureshared.NetworkPublicIPAddress:                  true,
		azureshared.NetworkLocalNetworkGateway:              true,
		azureshared.NetworkVirtualNetworkGatewayConnection:  true,
		azureshared.ExtendedLocationCustomLocation:          true,
		azureshared.ManagedIdentityUserAssignedIdentity:     true,
		azureshared.NetworkVirtualNetwork:                   true,
		stdlib.NetworkIP:                                    true,
		stdlib.NetworkDNS:                                   true,
	}
}

func (n networkVirtualNetworkGatewayWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "azurerm_virtual_network_gateway.name",
		},
	}
}

func (n networkVirtualNetworkGatewayWrapper) IAMPermissions() []string {
	return []string{
		"Microsoft.Network/virtualNetworkGateways/read",
	}
}

func extractHostFromURLOrHostname(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	u, err := url.Parse(s)
	if err != nil {
		return s
	}
	if u.Host != "" {
		return u.Hostname()
	}
	return s
}

func (n networkVirtualNetworkGatewayWrapper) PredefinedRole() string {
	return "Reader"
}
