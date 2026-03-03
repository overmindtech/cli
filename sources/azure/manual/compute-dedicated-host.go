package manual

import (
	"context"
	"errors"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"
	"github.com/overmindtech/cli/go/discovery"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/shared"
)

var ComputeDedicatedHostLookupByName = shared.NewItemTypeLookup("name", azureshared.ComputeDedicatedHost)

type computeDedicatedHostWrapper struct {
	client clients.DedicatedHostsClient
	*azureshared.MultiResourceGroupBase
}

func NewComputeDedicatedHost(client clients.DedicatedHostsClient, resourceGroupScopes []azureshared.ResourceGroupScope) sources.SearchableWrapper {
	return &computeDedicatedHostWrapper{
		client: client,
		MultiResourceGroupBase: azureshared.NewMultiResourceGroupBase(
			resourceGroupScopes,
			sdp.AdapterCategory_ADAPTER_CATEGORY_COMPUTE_APPLICATION,
			azureshared.ComputeDedicatedHost,
		),
	}
}

// ref: https://learn.microsoft.com/en-us/rest/api/compute/dedicated-hosts/get?view=rest-compute-2025-04-01&tabs=HTTP
func (c *computeDedicatedHostWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) != 2 {
		return nil, azureshared.QueryError(errors.New("queryParts must be exactly 2: dedicated host group name and dedicated host name"), scope, c.Type())
	}
	hostGroupName := queryParts[0]
	if hostGroupName == "" {
		return nil, azureshared.QueryError(errors.New("dedicated host group name cannot be empty"), scope, c.Type())
	}
	hostName := queryParts[1]
	if hostName == "" {
		return nil, azureshared.QueryError(errors.New("dedicated host name cannot be empty"), scope, c.Type())
	}

	rgScope, err := c.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}
	resp, err := c.client.Get(ctx, rgScope.ResourceGroup, hostGroupName, hostName, nil)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}
	return c.azureDedicatedHostToSDPItem(&resp.DedicatedHost, hostGroupName, scope)
}

// ref: https://learn.microsoft.com/en-us/rest/api/compute/dedicated-hosts/list-by-host-group?view=rest-compute-2025-04-01&tabs=HTTP
func (c *computeDedicatedHostWrapper) Search(ctx context.Context, scope string, queryParts ...string) ([]*sdp.Item, *sdp.QueryError) {
	if len(queryParts) != 1 {
		return nil, azureshared.QueryError(errors.New("queryParts must be exactly 1: dedicated host group name"), scope, c.Type())
	}
	hostGroupName := queryParts[0]
	if hostGroupName == "" {
		return nil, azureshared.QueryError(errors.New("dedicated host group name cannot be empty"), scope, c.Type())
	}

	rgScope, err := c.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}
	pager := c.client.NewListByHostGroupPager(rgScope.ResourceGroup, hostGroupName, nil)

	var items []*sdp.Item
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, azureshared.QueryError(err, scope, c.Type())
		}
		for _, host := range page.Value {
			if host == nil || host.Name == nil {
				continue
			}
			item, sdpErr := c.azureDedicatedHostToSDPItem(host, hostGroupName, scope)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}
	return items, nil
}

func (c *computeDedicatedHostWrapper) SearchStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string, queryParts ...string) {
	if len(queryParts) != 1 {
		stream.SendError(azureshared.QueryError(errors.New("queryParts must be exactly 1: dedicated host group name"), scope, c.Type()))
		return
	}
	hostGroupName := queryParts[0]
	if hostGroupName == "" {
		stream.SendError(azureshared.QueryError(errors.New("dedicated host group name cannot be empty"), scope, c.Type()))
		return
	}

	rgScope, err := c.ResourceGroupScopeFromScope(scope)
	if err != nil {
		stream.SendError(azureshared.QueryError(err, scope, c.Type()))
		return
	}

	pager := c.client.NewListByHostGroupPager(rgScope.ResourceGroup, hostGroupName, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			stream.SendError(azureshared.QueryError(err, scope, c.Type()))
			return
		}
		for _, host := range page.Value {
			if host == nil || host.Name == nil {
				continue
			}
			item, sdpErr := c.azureDedicatedHostToSDPItem(host, hostGroupName, scope)
			if sdpErr != nil {
				stream.SendError(sdpErr)
				continue
			}
			cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
			stream.SendItem(item)
		}
	}
}

func (c *computeDedicatedHostWrapper) azureDedicatedHostToSDPItem(host *armcompute.DedicatedHost, hostGroupName, scope string) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(host, "tags")
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}

	if host.Name == nil {
		return nil, azureshared.QueryError(errors.New("dedicated host name is nil"), scope, c.Type())
	}
	hostName := *host.Name
	if hostName == "" {
		return nil, azureshared.QueryError(errors.New("dedicated host name cannot be empty"), scope, c.Type())
	}
	if err := attributes.Set("uniqueAttr", shared.CompositeLookupKey(hostGroupName, hostName)); err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}

	linkedItemQueries := make([]*sdp.LinkedItemQuery, 0)

	// Parent: dedicated host group
	linkedItemQueries = append(linkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   azureshared.ComputeDedicatedHostGroup.String(),
			Method: sdp.QueryMethod_GET,
			Query:  hostGroupName,
			Scope:  scope,
		},
	})

	// VMs deployed on this dedicated host
	if host.Properties != nil && host.Properties.VirtualMachines != nil {
		for _, vmRef := range host.Properties.VirtualMachines {
			if vmRef == nil || vmRef.ID == nil || *vmRef.ID == "" {
				continue
			}
			vmName := azureshared.ExtractResourceName(*vmRef.ID)
			if vmName == "" {
				continue
			}
			vmScope := scope
			if linkScope := azureshared.ExtractScopeFromResourceID(*vmRef.ID); linkScope != "" {
				vmScope = linkScope
			}
			linkedItemQueries = append(linkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   azureshared.ComputeVirtualMachine.String(),
					Method: sdp.QueryMethod_GET,
					Query:  vmName,
					Scope:  vmScope,
				},
			})
		}
	}

	sdpItem := &sdp.Item{
		Type:              azureshared.ComputeDedicatedHost.String(),
		UniqueAttribute:   "uniqueAttr",
		Attributes:        attributes,
		Scope:             scope,
		Tags:              azureshared.ConvertAzureTags(host.Tags),
		LinkedItemQueries: linkedItemQueries,
	}

	// Health status from ProvisioningState
	if host.Properties != nil && host.Properties.ProvisioningState != nil {
		state := strings.ToLower(*host.Properties.ProvisioningState)
		switch state {
		case "succeeded":
			sdpItem.Health = sdp.Health_HEALTH_OK.Enum()
		case "creating", "updating", "deleting":
			sdpItem.Health = sdp.Health_HEALTH_PENDING.Enum()
		case "failed", "canceled":
			sdpItem.Health = sdp.Health_HEALTH_ERROR.Enum()
		}
	}

	return sdpItem, nil
}

func (c *computeDedicatedHostWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		ComputeDedicatedHostGroupLookupByName,
		ComputeDedicatedHostLookupByName,
	}
}

func (c *computeDedicatedHostWrapper) SearchLookups() []sources.ItemTypeLookups {
	return []sources.ItemTypeLookups{
		{
			ComputeDedicatedHostGroupLookupByName,
		},
	}
}

func (c *computeDedicatedHostWrapper) PotentialLinks() map[shared.ItemType]bool {
	return map[shared.ItemType]bool{
		azureshared.ComputeDedicatedHostGroup: true,
		azureshared.ComputeVirtualMachine:     true,
	}
}

// ref: https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/dedicated_host
func (c *computeDedicatedHostWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_SEARCH,
			TerraformQueryMap: "azurerm_dedicated_host.id",
		},
	}
}

func (c *computeDedicatedHostWrapper) IAMPermissions() []string {
	return []string{
		"Microsoft.Compute/hostGroups/hosts/read",
	}
}

func (c *computeDedicatedHostWrapper) PredefinedRole() string {
	return "Reader"
}
