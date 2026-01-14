package manual

import (
	"context"
	"errors"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/dns/armdns"
	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

var NetworkZoneLookupByName = shared.NewItemTypeLookup("name", azureshared.NetworkZone)

type networkZoneWrapper struct {
	client clients.ZonesClient

	*azureshared.ResourceGroupBase
}

func NewNetworkZone(client clients.ZonesClient, subscriptionID, resourceGroup string) sources.ListableWrapper {
	return &networkZoneWrapper{
		client: client,
		ResourceGroupBase: azureshared.NewResourceGroupBase(
			subscriptionID,
			resourceGroup,
			sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
			azureshared.NetworkZone,
		),
	}
}

func (n networkZoneWrapper) List(ctx context.Context) ([]*sdp.Item, *sdp.QueryError) {
	pager := n.client.NewListByResourceGroupPager(n.ResourceGroup(), nil)

	var items []*sdp.Item
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, azureshared.QueryError(err, n.DefaultScope(), n.Type())
		}

		for _, zone := range page.Value {
			if zone.Name == nil {
				continue
			}
			item, sdpErr := n.azureZoneToSDPItem(zone)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}

	return items, nil
}

// ref: https://learn.microsoft.com/en-us/rest/api/dns/zones/list-by-resource-group?view=rest-dns-2018-05-01&tabs=HTTP
func (n networkZoneWrapper) ListStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey) {
	pager := n.client.NewListByResourceGroupPager(n.ResourceGroup(), nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			stream.SendError(azureshared.QueryError(err, n.DefaultScope(), n.Type()))
			return
		}
		for _, zone := range page.Value {
			if zone.Name == nil {
				continue
			}
			item, sdpErr := n.azureZoneToSDPItem(zone)
			if sdpErr != nil {
				stream.SendError(sdpErr)
				continue
			}
			cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
			stream.SendItem(item)
		}
	}
}

func (n networkZoneWrapper) azureZoneToSDPItem(zone *armdns.Zone) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(zone, "tags")
	if err != nil {
		return nil, azureshared.QueryError(err, n.DefaultScope(), n.Type())
	}

	if zone.Name == nil {
		return nil, azureshared.QueryError(errors.New("zone name is nil"), n.DefaultScope(), n.Type())
	}

	sdpItem := &sdp.Item{
		Type:            azureshared.NetworkZone.String(),
		UniqueAttribute: "name",
		Attributes:      attributes,
		Scope:           n.DefaultScope(),
		Tags:            azureshared.ConvertAzureTags(zone.Tags),
	}

	zoneName := *zone.Name

	// Link to DNS name (standard library) for the zone name itself
	// The zone name is a DNS name and should be linked to verify proper delegation and show the public DNS view
	if zoneName != "" {
		sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   "dns",
				Method: sdp.QueryMethod_SEARCH,
				Query:  zoneName,
				Scope:  "global",
			},
			BlastPropagation: &sdp.BlastPropagation{
				// DNS names are always linked
				In:  true,
				Out: true,
			},
		})
	}

	// Link to Virtual Networks from RegistrationVirtualNetworks (external resources)
	// Reference: https://learn.microsoft.com/en-us/rest/api/virtualnetwork/virtual-networks/get
	if zone.Properties != nil && zone.Properties.RegistrationVirtualNetworks != nil {
		for _, vnetRef := range zone.Properties.RegistrationVirtualNetworks {
			if vnetRef != nil && vnetRef.ID != nil {
				vnetName := azureshared.ExtractResourceName(*vnetRef.ID)
				if vnetName != "" {
					// Extract subscription ID and resource group from the resource ID to determine scope
					scope := azureshared.ExtractScopeFromResourceID(*vnetRef.ID)
					if scope == "" {
						scope = n.DefaultScope()
					}

					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.NetworkVirtualNetwork.String(),
							Method: sdp.QueryMethod_GET,
							Query:  vnetName,
							Scope:  scope,
						},
						BlastPropagation: &sdp.BlastPropagation{
							// DNS zone depends on virtual network for registration
							// If virtual network is deleted/modified, DNS zone registration may fail
							In:  true,
							Out: false,
						}, // Virtual network provides registration capability for the DNS zone
					})
				}
			}
		}
	}

	// Link to Virtual Networks from ResolutionVirtualNetworks (external resources)
	// Reference: https://learn.microsoft.com/en-us/rest/api/virtualnetwork/virtual-networks/get
	if zone.Properties != nil && zone.Properties.ResolutionVirtualNetworks != nil {
		for _, vnetRef := range zone.Properties.ResolutionVirtualNetworks {
			if vnetRef != nil && vnetRef.ID != nil {
				vnetName := azureshared.ExtractResourceName(*vnetRef.ID)
				if vnetName != "" {
					// Extract subscription ID and resource group from the resource ID to determine scope
					scope := azureshared.ExtractScopeFromResourceID(*vnetRef.ID)
					if scope == "" {
						scope = n.DefaultScope()
					}

					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.NetworkVirtualNetwork.String(),
							Method: sdp.QueryMethod_GET,
							Query:  vnetName,
							Scope:  scope,
						},
						BlastPropagation: &sdp.BlastPropagation{
							// DNS zone depends on virtual network for resolution
							// If virtual network is deleted/modified, DNS zone resolution may fail
							In:  true,
							Out: false,
						}, // Virtual network provides resolution capability for the DNS zone
					})
				}
			}
		}
	}

	// Link to DNS Record Sets (child resource)
	// Reference: https://learn.microsoft.com/en-us/rest/api/dns/record-sets/list-by-dns-zone
	// Record sets can be listed by zone name, so we use SEARCH method
	// The zone name is available, which is sufficient to list record sets
	sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   azureshared.NetworkDNSRecordSet.String(),
			Method: sdp.QueryMethod_SEARCH,
			Query:  zoneName,
			Scope:  n.DefaultScope(),
		},
		BlastPropagation: &sdp.BlastPropagation{
			// Record sets are child resources of the DNS zone
			// Changes to record sets affect DNS resolution for the zone
			In:  true,
			Out: true,
		}, // Record sets are tightly coupled with the DNS zone; bidirectional dependency
	})

	// Link to DNS names (standard library) from NameServers
	// Reference: DNS name servers are external resources
	if zone.Properties != nil && zone.Properties.NameServers != nil {
		for _, nameServer := range zone.Properties.NameServers {
			if nameServer != nil && *nameServer != "" {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "dns",
						Method: sdp.QueryMethod_SEARCH,
						Query:  *nameServer,
						Scope:  "global",
					},
					BlastPropagation: &sdp.BlastPropagation{
						// DNS names are always linked
						In:  true,
						Out: true,
					},
				})
			}
		}
	}

	return sdpItem, nil
}

// ref: https://learn.microsoft.com/en-us/rest/api/dns/zones/get?view=rest-dns-2018-05-01&tabs=HTTP
func (n networkZoneWrapper) Get(ctx context.Context, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 1 {
		return nil, azureshared.QueryError(errors.New("query must be exactly one part and be a zone name"), n.DefaultScope(), n.Type())
	}
	zoneName := queryParts[0]

	zone, err := n.client.Get(ctx, n.ResourceGroup(), zoneName, nil)
	if err != nil {
		return nil, azureshared.QueryError(err, n.DefaultScope(), n.Type())
	}

	return n.azureZoneToSDPItem(&zone.Zone)
}

func (n networkZoneWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		NetworkZoneLookupByName,
	}
}

// ref https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/dns_zone
func (n networkZoneWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "azurerm_dns_zone.name",
		},
	}
}

func (n networkZoneWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		azureshared.NetworkVirtualNetwork,
		azureshared.NetworkDNSRecordSet,
		stdlib.NetworkDNS,
	)
}

// ref: https://learn.microsoft.com/en-us/azure/role-based-access-control/permissions/networking
func (n networkZoneWrapper) IAMPermissions() []string {
	return []string{
		"Microsoft.Network/dnszones/read",
	}
}

func (n networkZoneWrapper) PredefinedRole() string {
	return "Reader"
}
