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
)

var NetworkDNSVirtualNetworkLinkLookupByName = shared.NewItemTypeLookup("name", azureshared.NetworkDNSVirtualNetworkLink)

type networkDNSVirtualNetworkLinkWrapper struct {
	client clients.VirtualNetworkLinksClient

	*azureshared.MultiResourceGroupBase
}

func NewNetworkDNSVirtualNetworkLink(client clients.VirtualNetworkLinksClient, resourceGroupScopes []azureshared.ResourceGroupScope) sources.SearchableWrapper {
	return &networkDNSVirtualNetworkLinkWrapper{
		client: client,
		MultiResourceGroupBase: azureshared.NewMultiResourceGroupBase(
			resourceGroupScopes,
			sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
			azureshared.NetworkDNSVirtualNetworkLink,
		),
	}
}

// ref: https://learn.microsoft.com/en-us/rest/api/dns/privatednszones/virtualnetworklinks/get
func (c networkDNSVirtualNetworkLinkWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 2 {
		return nil, azureshared.QueryError(errors.New("Get requires 2 query parts: privateZoneName and virtualNetworkLinkName"), scope, c.Type())
	}
	privateZoneName := queryParts[0]
	linkName := queryParts[1]
	if privateZoneName == "" || linkName == "" {
		return nil, azureshared.QueryError(errors.New("privateZoneName and virtualNetworkLinkName cannot be empty"), scope, c.Type())
	}

	rgScope, err := c.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}
	resp, err := c.client.Get(ctx, rgScope.ResourceGroup, privateZoneName, linkName, nil)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}
	return c.azureVirtualNetworkLinkToSDPItem(&resp.VirtualNetworkLink, privateZoneName, scope)
}

func (c networkDNSVirtualNetworkLinkWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		NetworkPrivateDNSZoneLookupByName,
		NetworkDNSVirtualNetworkLinkLookupByName,
	}
}

// ref: https://learn.microsoft.com/en-us/rest/api/dns/privatednszones/virtualnetworklinks/list
func (c networkDNSVirtualNetworkLinkWrapper) Search(ctx context.Context, scope string, queryParts ...string) ([]*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 1 {
		return nil, azureshared.QueryError(errors.New("Search requires 1 query part: privateZoneName"), scope, c.Type())
	}
	privateZoneName := queryParts[0]
	if privateZoneName == "" {
		return nil, azureshared.QueryError(errors.New("privateZoneName cannot be empty"), scope, c.Type())
	}

	rgScope, err := c.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}
	pager := c.client.NewListPager(rgScope.ResourceGroup, privateZoneName, nil)

	var items []*sdp.Item
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, azureshared.QueryError(err, scope, c.Type())
		}
		for _, link := range page.Value {
			if link == nil || link.Name == nil {
				continue
			}
			item, sdpErr := c.azureVirtualNetworkLinkToSDPItem(link, privateZoneName, scope)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}
	return items, nil
}

func (c networkDNSVirtualNetworkLinkWrapper) SearchStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string, queryParts ...string) {
	if len(queryParts) < 1 {
		stream.SendError(azureshared.QueryError(errors.New("Search requires 1 query part: privateZoneName"), scope, c.Type()))
		return
	}
	privateZoneName := queryParts[0]
	if privateZoneName == "" {
		stream.SendError(azureshared.QueryError(errors.New("privateZoneName cannot be empty"), scope, c.Type()))
		return
	}

	rgScope, err := c.ResourceGroupScopeFromScope(scope)
	if err != nil {
		stream.SendError(azureshared.QueryError(err, scope, c.Type()))
		return
	}
	pager := c.client.NewListPager(rgScope.ResourceGroup, privateZoneName, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			stream.SendError(azureshared.QueryError(err, scope, c.Type()))
			return
		}
		for _, link := range page.Value {
			if link == nil || link.Name == nil {
				continue
			}
			item, sdpErr := c.azureVirtualNetworkLinkToSDPItem(link, privateZoneName, scope)
			if sdpErr != nil {
				stream.SendError(sdpErr)
				continue
			}
			cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
			stream.SendItem(item)
		}
	}
}

func (c networkDNSVirtualNetworkLinkWrapper) SearchLookups() []sources.ItemTypeLookups {
	return []sources.ItemTypeLookups{
		{NetworkPrivateDNSZoneLookupByName},
	}
}

func (c networkDNSVirtualNetworkLinkWrapper) azureVirtualNetworkLinkToSDPItem(link *armprivatedns.VirtualNetworkLink, privateZoneName, scope string) (*sdp.Item, *sdp.QueryError) {
	if link.Name == nil {
		return nil, azureshared.QueryError(errors.New("virtual network link name is nil"), scope, c.Type())
	}

	attributes, err := shared.ToAttributesWithExclude(link, "tags")
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}

	uniqueAttr := shared.CompositeLookupKey(privateZoneName, *link.Name)
	if err := attributes.Set("uniqueAttr", uniqueAttr); err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}

	sdpItem := &sdp.Item{
		Type:            azureshared.NetworkDNSVirtualNetworkLink.String(),
		UniqueAttribute: "uniqueAttr",
		Attributes:      attributes,
		Scope:           scope,
		Tags:            azureshared.ConvertAzureTags(link.Tags),
	}

	// Health from provisioning state
	if link.Properties != nil && link.Properties.ProvisioningState != nil {
		switch *link.Properties.ProvisioningState {
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

	// Link to parent Private DNS Zone
	sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   azureshared.NetworkPrivateDNSZone.String(),
			Method: sdp.QueryMethod_GET,
			Query:  privateZoneName,
			Scope:  scope,
		},
	})

	// Link to the Virtual Network referenced by this link
	if link.Properties != nil && link.Properties.VirtualNetwork != nil && link.Properties.VirtualNetwork.ID != nil {
		vnetName := azureshared.ExtractResourceName(*link.Properties.VirtualNetwork.ID)
		if vnetName != "" {
			linkedScope := scope
			if extractedScope := azureshared.ExtractScopeFromResourceID(*link.Properties.VirtualNetwork.ID); extractedScope != "" {
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

	return sdpItem, nil
}

func (c networkDNSVirtualNetworkLinkWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		azureshared.NetworkPrivateDNSZone,
		azureshared.NetworkVirtualNetwork,
	)
}

// ref: https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/private_dns_zone_virtual_network_link
func (c networkDNSVirtualNetworkLinkWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_SEARCH,
			TerraformQueryMap: "azurerm_private_dns_zone_virtual_network_link.id",
		},
	}
}

// ref: https://learn.microsoft.com/en-us/azure/role-based-access-control/resource-provider-operations#microsoftnetwork
func (c networkDNSVirtualNetworkLinkWrapper) IAMPermissions() []string {
	return []string{
		"Microsoft.Network/privateDnsZones/virtualNetworkLinks/read",
	}
}

func (c networkDNSVirtualNetworkLinkWrapper) PredefinedRole() string {
	return "Reader"
}
