package manual

import (
	"context"
	"errors"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/privatedns/armprivatedns"
	"github.com/overmindtech/cli/go/discovery"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

var NetworkPrivateDNSZoneLookupByName = shared.NewItemTypeLookup("name", azureshared.NetworkPrivateDNSZone)

type networkPrivateDNSZoneWrapper struct {
	client clients.PrivateDNSZonesClient

	*azureshared.MultiResourceGroupBase
}

func NewNetworkPrivateDNSZone(client clients.PrivateDNSZonesClient, resourceGroupScopes []azureshared.ResourceGroupScope) sources.ListableWrapper {
	return &networkPrivateDNSZoneWrapper{
		client: client,
		MultiResourceGroupBase: azureshared.NewMultiResourceGroupBase(
			resourceGroupScopes,
			sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
			azureshared.NetworkPrivateDNSZone,
		),
	}
}

func (n networkPrivateDNSZoneWrapper) List(ctx context.Context, scope string) ([]*sdp.Item, *sdp.QueryError) {
	rgScope, err := n.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, n.Type())
	}
	pager := n.client.NewListByResourceGroupPager(rgScope.ResourceGroup, nil)

	var items []*sdp.Item
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, azureshared.QueryError(err, scope, n.Type())
		}
		for _, zone := range page.Value {
			if zone.Name == nil {
				continue
			}
			item, sdpErr := n.azurePrivateZoneToSDPItem(zone, scope)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}
	return items, nil
}

func (n networkPrivateDNSZoneWrapper) ListStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string) {
	rgScope, err := n.ResourceGroupScopeFromScope(scope)
	if err != nil {
		stream.SendError(azureshared.QueryError(err, scope, n.Type()))
		return
	}
	pager := n.client.NewListByResourceGroupPager(rgScope.ResourceGroup, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			stream.SendError(azureshared.QueryError(err, scope, n.Type()))
			return
		}
		for _, zone := range page.Value {
			if zone.Name == nil {
				continue
			}
			item, sdpErr := n.azurePrivateZoneToSDPItem(zone, scope)
			if sdpErr != nil {
				stream.SendError(sdpErr)
				continue
			}
			cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
			stream.SendItem(item)
		}
	}
}

func (n networkPrivateDNSZoneWrapper) azurePrivateZoneToSDPItem(zone *armprivatedns.PrivateZone, scope string) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(zone, "tags")
	if err != nil {
		return nil, azureshared.QueryError(err, scope, n.Type())
	}
	if zone.Name == nil {
		return nil, azureshared.QueryError(errors.New("zone name is nil"), scope, n.Type())
	}

	sdpItem := &sdp.Item{
		Type:            azureshared.NetworkPrivateDNSZone.String(),
		UniqueAttribute: "name",
		Attributes:      attributes,
		Scope:           scope,
		Tags:            azureshared.ConvertAzureTags(zone.Tags),
	}

	// Health from provisioning state
	if zone.Properties != nil && zone.Properties.ProvisioningState != nil {
		switch *zone.Properties.ProvisioningState {
		case armprivatedns.ProvisioningStateSucceeded:
			sdpItem.Health = sdp.Health_HEALTH_OK.Enum()
		case armprivatedns.ProvisioningStateCreating, armprivatedns.ProvisioningStateUpdating, armprivatedns.ProvisioningStateDeleting:
			sdpItem.Health = sdp.Health_HEALTH_PENDING.Enum()
		case armprivatedns.ProvisioningStateFailed, armprivatedns.ProvisioningStateCanceled:
			sdpItem.Health = sdp.Health_HEALTH_ERROR.Enum()
		default:
			sdpItem.Health = sdp.Health_HEALTH_UNKNOWN.Enum()
		}
	}

	zoneName := *zone.Name

	// Link to DNS name (standard library) for the zone name
	if zoneName != "" {
		sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   stdlib.NetworkDNS.String(),
				Method: sdp.QueryMethod_SEARCH,
				Query:  zoneName,
				Scope:  "global",
			},
		})
	}

	// Link to Virtual Network Links (child resource of Private DNS Zone)
	// Reference: https://learn.microsoft.com/en-us/rest/api/dns/privatednszones/virtualnetworklinks/list
	// Virtual network links can be listed by zone name, so we use SEARCH method
	if zoneName != "" {
		sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   azureshared.NetworkDNSVirtualNetworkLink.String(),
				Method: sdp.QueryMethod_SEARCH,
				Query:  zoneName,
				Scope:  scope,
			},
		})
	}

	// Link to DNS Record Sets (child resource of Private DNS Zone)
	// Reference: https://learn.microsoft.com/en-us/rest/api/dns/privatednszones/recordsets/list
	// Record sets (A, AAAA, CNAME, MX, PTR, SOA, SRV, TXT) can be listed by zone name, so we use SEARCH method
	if zoneName != "" {
		sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   azureshared.NetworkDNSRecordSet.String(),
				Method: sdp.QueryMethod_SEARCH,
				Query:  zoneName,
				Scope:  scope,
			},
		})
	}

	return sdpItem, nil
}

// ref: https://learn.microsoft.com/en-us/rest/api/dns/privatednszones/get
func (n networkPrivateDNSZoneWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 1 {
		return nil, azureshared.QueryError(errors.New("query must be exactly one part (private zone name)"), scope, n.Type())
	}
	zoneName := queryParts[0]
	if zoneName == "" {
		return nil, azureshared.QueryError(errors.New("private zone name cannot be empty"), scope, n.Type())
	}

	rgScope, err := n.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, n.Type())
	}
	resp, err := n.client.Get(ctx, rgScope.ResourceGroup, zoneName, nil)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, n.Type())
	}
	return n.azurePrivateZoneToSDPItem(&resp.PrivateZone, scope)
}

func (n networkPrivateDNSZoneWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		NetworkPrivateDNSZoneLookupByName,
	}
}

func (n networkPrivateDNSZoneWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		azureshared.NetworkDNSRecordSet,
		azureshared.NetworkDNSVirtualNetworkLink,
		stdlib.NetworkDNS,
	)
}

// ref: https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/private_dns_zone
func (n networkPrivateDNSZoneWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "azurerm_private_dns_zone.name",
		},
	}
}

// ref: https://learn.microsoft.com/en-us/azure/role-based-access-control/resource-provider-operations#microsoftnetwork
func (n networkPrivateDNSZoneWrapper) IAMPermissions() []string {
	return []string{
		"Microsoft.Network/privateDnsZones/read",
	}
}

func (n networkPrivateDNSZoneWrapper) PredefinedRole() string {
	return "Reader"
}
