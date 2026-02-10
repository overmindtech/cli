package manual

import (
	"context"
	"errors"

	armcompute "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"
	discovery "github.com/overmindtech/workspace/discovery"
	"github.com/overmindtech/workspace/sdp-go"
	sdpcache "github.com/overmindtech/workspace/sdpcache"
	sources "github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/shared"
)

var ComputeDiskAccessLookupByName = shared.NewItemTypeLookup("name", azureshared.ComputeDiskAccess)

type computeDiskAccessWrapper struct {
	client clients.DiskAccessesClient
	*azureshared.MultiResourceGroupBase
}

func NewComputeDiskAccess(client clients.DiskAccessesClient, resourceGroupScopes []azureshared.ResourceGroupScope) sources.ListableWrapper {
	return &computeDiskAccessWrapper{
		client: client,
		MultiResourceGroupBase: azureshared.NewMultiResourceGroupBase(
			resourceGroupScopes,
			sdp.AdapterCategory_ADAPTER_CATEGORY_STORAGE,
			azureshared.ComputeDiskAccess,
		),
	}
}

func (c *computeDiskAccessWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) != 1 {
		return nil, azureshared.QueryError(errors.New("queryParts must be exactly 1 and be the disk access name"), scope, c.Type())
	}
	diskAccessName := queryParts[0]
	if diskAccessName == "" {
		return nil, azureshared.QueryError(errors.New("disk access name cannot be empty"), scope, c.Type())
	}
	rgScope, err := c.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}
	diskAccess, err := c.client.Get(ctx, rgScope.ResourceGroup, diskAccessName, nil)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}
	return c.azureDiskAccessToSDPItem(&diskAccess.DiskAccess, scope)
}

func (c *computeDiskAccessWrapper) List(ctx context.Context, scope string) ([]*sdp.Item, *sdp.QueryError) {
	rgScope, err := c.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}
	pager := c.client.NewListByResourceGroupPager(rgScope.ResourceGroup, nil)

	var items []*sdp.Item
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, azureshared.QueryError(err, scope, c.Type())
		}
		for _, diskAccess := range page.Value {
			if diskAccess.Name == nil {
				continue
			}
			item, sdpErr := c.azureDiskAccessToSDPItem(diskAccess, scope)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}
	return items, nil
}

func (c *computeDiskAccessWrapper) ListStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string) {
	rgScope, err := c.ResourceGroupScopeFromScope(scope)
	if err != nil {
		stream.SendError(azureshared.QueryError(err, scope, c.Type()))
		return
	}
	pager := c.client.NewListByResourceGroupPager(rgScope.ResourceGroup, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			stream.SendError(azureshared.QueryError(err, scope, c.Type()))
			return
		}
		for _, diskAccess := range page.Value {
			if diskAccess.Name == nil {
				continue
			}
			item, sdpErr := c.azureDiskAccessToSDPItem(diskAccess, scope)
			if sdpErr != nil {
				stream.SendError(sdpErr)
				continue
			}
			cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
			stream.SendItem(item)
		}
	}
}
func (c *computeDiskAccessWrapper) azureDiskAccessToSDPItem(diskAccess *armcompute.DiskAccess, scope string) (*sdp.Item, *sdp.QueryError) {
	if diskAccess.Name == nil {
		return nil, azureshared.QueryError(errors.New("name is nil"), scope, c.Type())
	}
	attributes, err := shared.ToAttributesWithExclude(diskAccess, "tags")
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}
	sdpItem := &sdp.Item{
		Type:              azureshared.ComputeDiskAccess.String(),
		UniqueAttribute:   "name",
		Attributes:        attributes,
		Scope:             scope,
		Tags:              azureshared.ConvertAzureTags(diskAccess.Tags),
		LinkedItemQueries: []*sdp.LinkedItemQuery{},
	}

	// Link to Private Endpoint Connections (child resource)
	// Reference: https://learn.microsoft.com/en-us/rest/api/compute/disk-accesses/list-private-endpoint-connections
	// GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Compute/diskAccesses/{diskAccessName}/privateEndpointConnections
	sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   azureshared.ComputeDiskAccessPrivateEndpointConnection.String(),
			Method: sdp.QueryMethod_SEARCH,
			Query:  *diskAccess.Name,
			Scope:  scope,
		},
		BlastPropagation: &sdp.BlastPropagation{
			In:  true, // Disk access changes affect private endpoint connections
			Out: true, // Private endpoint connection state affects disk access connectivity
		},
	})

	// Link to Network Private Endpoints (external resources) from PrivateEndpointConnections
	// Reference: https://learn.microsoft.com/en-us/rest/api/virtualnetwork/private-endpoints/get
	if diskAccess.Properties != nil && diskAccess.Properties.PrivateEndpointConnections != nil {
		for _, peConnection := range diskAccess.Properties.PrivateEndpointConnections {
			if peConnection.Properties != nil && peConnection.Properties.PrivateEndpoint != nil && peConnection.Properties.PrivateEndpoint.ID != nil {
				privateEndpointID := *peConnection.Properties.PrivateEndpoint.ID
				privateEndpointName := azureshared.ExtractResourceName(privateEndpointID)
				if privateEndpointName != "" {
					extractedScope := azureshared.ExtractScopeFromResourceID(privateEndpointID)
					if extractedScope == "" {
						extractedScope = scope
					}
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.NetworkPrivateEndpoint.String(),
							Method: sdp.QueryMethod_GET,
							Query:  privateEndpointName,
							Scope:  extractedScope,
						},
						BlastPropagation: &sdp.BlastPropagation{
							In:  true, // Private endpoint changes affect disk access private connectivity
							Out: true, // Disk access deletion or config changes may affect the private endpoint connection state
						},
					})
				}
			}
		}
	}

	return sdpItem, nil
}

func (c *computeDiskAccessWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		ComputeDiskAccessLookupByName,
	}
}

func (c *computeDiskAccessWrapper) PotentialLinks() map[shared.ItemType]bool {
	return map[shared.ItemType]bool{
		azureshared.ComputeDiskAccessPrivateEndpointConnection: true,
		azureshared.NetworkPrivateEndpoint:                     true,
	}
}

func (c *computeDiskAccessWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "azurerm_disk_access.name",
		},
	}
}
