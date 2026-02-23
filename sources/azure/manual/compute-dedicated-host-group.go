package manual

import (
	"context"
	"errors"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"
	"github.com/overmindtech/cli/go/discovery"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/shared"
)

var ComputeDedicatedHostGroupLookupByName = shared.NewItemTypeLookup("name", azureshared.ComputeDedicatedHostGroup)

type computeDedicatedHostGroupWrapper struct {
	client clients.DedicatedHostGroupsClient
	*azureshared.MultiResourceGroupBase
}

func NewComputeDedicatedHostGroup(client clients.DedicatedHostGroupsClient, resourceGroupScopes []azureshared.ResourceGroupScope) sources.ListableWrapper {
	return &computeDedicatedHostGroupWrapper{
		client: client,
		MultiResourceGroupBase: azureshared.NewMultiResourceGroupBase(
			resourceGroupScopes,
			sdp.AdapterCategory_ADAPTER_CATEGORY_COMPUTE_APPLICATION,
			azureshared.ComputeDedicatedHostGroup,
		),
	}
}

// ref: https://learn.microsoft.com/en-us/rest/api/compute/dedicated-host-groups/get?view=rest-compute-2025-04-01&tabs=HTTP
func (c *computeDedicatedHostGroupWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) != 1 {
		return nil, azureshared.QueryError(errors.New("queryParts must be exactly 1 and be the dedicated host group name"), scope, c.Type())
	}
	dedicatedHostGroupName := queryParts[0]
	if dedicatedHostGroupName == "" {
		return nil, azureshared.QueryError(errors.New("dedicated host group name cannot be empty"), scope, c.Type())
	}
	rgScope, err := c.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}
	dedicatedHostGroup, err := c.client.Get(ctx, rgScope.ResourceGroup, dedicatedHostGroupName, nil)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}
	return c.azureDedicatedHostGroupToSDPItem(&dedicatedHostGroup.DedicatedHostGroup, scope)
}

// ref: https://learn.microsoft.com/en-us/rest/api/compute/dedicated-host-groups/list-by-resource-group?view=rest-compute-2025-04-01&tabs=HTTP
func (c *computeDedicatedHostGroupWrapper) List(ctx context.Context, scope string) ([]*sdp.Item, *sdp.QueryError) {
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
		for _, dedicatedHostGroup := range page.Value {
			if dedicatedHostGroup.Name == nil {
				continue
			}
			item, sdpErr := c.azureDedicatedHostGroupToSDPItem(dedicatedHostGroup, scope)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}
	return items, nil
}

func (c *computeDedicatedHostGroupWrapper) ListStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string) {
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
		for _, dedicatedHostGroup := range page.Value {
			if dedicatedHostGroup.Name == nil {
				continue
			}
			item, sdpErr := c.azureDedicatedHostGroupToSDPItem(dedicatedHostGroup, scope)
			if sdpErr != nil {
				stream.SendError(sdpErr)
				continue
			}
			cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
			stream.SendItem(item)
		}
	}
}

func (c *computeDedicatedHostGroupWrapper) azureDedicatedHostGroupToSDPItem(dedicatedHostGroup *armcompute.DedicatedHostGroup, scope string) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(dedicatedHostGroup, "tags")
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}

	linkedItemQueries := make([]*sdp.LinkedItemQuery, 0)
	if dedicatedHostGroup.Properties != nil && dedicatedHostGroup.Properties.Hosts != nil && dedicatedHostGroup.Name != nil {
		hostGroupName := *dedicatedHostGroup.Name
		for _, hostRef := range dedicatedHostGroup.Properties.Hosts {
			if hostRef == nil || hostRef.ID == nil || *hostRef.ID == "" {
				continue
			}
			hostName := azureshared.ExtractResourceName(*hostRef.ID)
			if hostName == "" {
				continue
			}
			linkedItemQueries = append(linkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   azureshared.ComputeDedicatedHost.String(),
					Method: sdp.QueryMethod_GET,
					Query:  shared.CompositeLookupKey(hostGroupName, hostName),
					Scope:  scope,
				},
			})
		}
	}

	sdpItem := &sdp.Item{
		Type:              azureshared.ComputeDedicatedHostGroup.String(),
		UniqueAttribute:   "name",
		Attributes:        attributes,
		Scope:             scope,
		Tags:              azureshared.ConvertAzureTags(dedicatedHostGroup.Tags),
		LinkedItemQueries: linkedItemQueries,
	}
	return sdpItem, nil
}

func (c *computeDedicatedHostGroupWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		ComputeDedicatedHostGroupLookupByName,
	}
}

func (c *computeDedicatedHostGroupWrapper) PotentialLinks() map[shared.ItemType]bool {
	return map[shared.ItemType]bool{
		azureshared.ComputeDedicatedHost: true,
	}
}

// ref: https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/dedicated_host_group
func (c *computeDedicatedHostGroupWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "azurerm_dedicated_host_group.name",
		},
	}
}

func (c *computeDedicatedHostGroupWrapper) IAMPermissions() []string {
	return []string{
		"Microsoft.Compute/hostGroups/read",
	}
}

func (c *computeDedicatedHostGroupWrapper) PredefinedRole() string {
	return "Reader"
}
