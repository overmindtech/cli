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

var ComputeCapacityReservationGroupLookupByName = shared.NewItemTypeLookup("name", azureshared.ComputeCapacityReservationGroup)

type computeCapacityReservationGroupWrapper struct {
	client clients.CapacityReservationGroupsClient
	*azureshared.MultiResourceGroupBase
}

// NewComputeCapacityReservationGroup creates a new computeCapacityReservationGroupWrapper instance.
func NewComputeCapacityReservationGroup(client clients.CapacityReservationGroupsClient, resourceGroupScopes []azureshared.ResourceGroupScope) sources.ListableWrapper {
	return &computeCapacityReservationGroupWrapper{
		client: client,
		MultiResourceGroupBase: azureshared.NewMultiResourceGroupBase(
			resourceGroupScopes,
			sdp.AdapterCategory_ADAPTER_CATEGORY_COMPUTE_APPLICATION,
			azureshared.ComputeCapacityReservationGroup,
		),
	}
}

func capacityReservationGroupGetOptions() *armcompute.CapacityReservationGroupsClientGetOptions {
	return nil
}

func capacityReservationGroupListOptions() *armcompute.CapacityReservationGroupsClientListByResourceGroupOptions {
	expand := armcompute.ExpandTypesForGetCapacityReservationGroupsVirtualMachinesRef
	return &armcompute.CapacityReservationGroupsClientListByResourceGroupOptions{
		Expand: &expand,
	}
}

// ref: https://learn.microsoft.com/en-us/rest/api/compute/capacity-reservation-groups/get?view=rest-compute-2025-04-01&tabs=HTTP
func (c *computeCapacityReservationGroupWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) != 1 {
		return nil, azureshared.QueryError(errors.New("queryParts must be exactly 1 and be the capacity reservation group name"), scope, c.Type())
	}
	capacityReservationGroupName := queryParts[0]
	if capacityReservationGroupName == "" {
		return nil, azureshared.QueryError(errors.New("capacity reservation group name cannot be empty"), scope, c.Type())
	}
	rgScope, err := c.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}
	capacityReservationGroup, err := c.client.Get(ctx, rgScope.ResourceGroup, capacityReservationGroupName, capacityReservationGroupGetOptions())
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}
	return c.azureCapacityReservationGroupToSDPItem(&capacityReservationGroup.CapacityReservationGroup, scope)
}

// ref:https://learn.microsoft.com/en-us/rest/api/compute/capacity-reservation-groups/list-by-resource-group?view=rest-compute-2025-04-01&tabs=HTTP
func (c *computeCapacityReservationGroupWrapper) List(ctx context.Context, scope string) ([]*sdp.Item, *sdp.QueryError) {
	rgScope, err := c.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}
	pager := c.client.NewListByResourceGroupPager(rgScope.ResourceGroup, capacityReservationGroupListOptions())

	var items []*sdp.Item
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, azureshared.QueryError(err, scope, c.Type())
		}
		for _, capacityReservationGroup := range page.Value {
			if capacityReservationGroup.Name == nil {
				continue
			}
			item, sdpErr := c.azureCapacityReservationGroupToSDPItem(capacityReservationGroup, scope)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}
	return items, nil
}

func (c *computeCapacityReservationGroupWrapper) ListStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string) {
	rgScope, err := c.ResourceGroupScopeFromScope(scope)
	if err != nil {
		stream.SendError(azureshared.QueryError(err, scope, c.Type()))
		return
	}
	pager := c.client.NewListByResourceGroupPager(rgScope.ResourceGroup, capacityReservationGroupListOptions())
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			stream.SendError(azureshared.QueryError(err, scope, c.Type()))
			return
		}
		for _, capacityReservationGroup := range page.Value {
			if capacityReservationGroup.Name == nil {
				continue
			}
			item, sdpErr := c.azureCapacityReservationGroupToSDPItem(capacityReservationGroup, scope)
			if sdpErr != nil {
				stream.SendError(sdpErr)
				continue
			}
			cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
			stream.SendItem(item)
		}
	}
}

func (c *computeCapacityReservationGroupWrapper) azureCapacityReservationGroupToSDPItem(capacityReservationGroup *armcompute.CapacityReservationGroup, scope string) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(capacityReservationGroup, "tags")
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}

	linkedItemQueries := make([]*sdp.LinkedItemQuery, 0)

	if capacityReservationGroup.Properties != nil {
		groupName := ""
		if capacityReservationGroup.Name != nil {
			groupName = *capacityReservationGroup.Name
		}

		// Child resource: capacity reservations in this group (have their own GET/LIST endpoints)
		if capacityReservationGroup.Properties.CapacityReservations != nil && groupName != "" {
			for _, ref := range capacityReservationGroup.Properties.CapacityReservations {
				if ref == nil || ref.ID == nil || *ref.ID == "" {
					continue
				}
				reservationName := azureshared.ExtractResourceName(*ref.ID)
				if reservationName == "" {
					continue
				}
				linkedItemQueries = append(linkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   azureshared.ComputeCapacityReservation.String(),
						Method: sdp.QueryMethod_GET,
						Query:  shared.CompositeLookupKey(groupName, reservationName),
						Scope:  scope,
					},
				})
			}
		}

		// External resource: VMs associated with this capacity reservation group
		if capacityReservationGroup.Properties.VirtualMachinesAssociated != nil {
			for _, ref := range capacityReservationGroup.Properties.VirtualMachinesAssociated {
				if ref == nil || ref.ID == nil || *ref.ID == "" {
					continue
				}
				vmName := azureshared.ExtractResourceName(*ref.ID)
				if vmName == "" {
					continue
				}
				linkScope := scope
				if extractedScope := azureshared.ExtractScopeFromResourceID(*ref.ID); extractedScope != "" {
					linkScope = extractedScope
				}
				linkedItemQueries = append(linkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   azureshared.ComputeVirtualMachine.String(),
						Method: sdp.QueryMethod_GET,
						Query:  vmName,
						Scope:  linkScope,
					},
				})
			}
		}
	}

	sdpItem := &sdp.Item{
		Type:              azureshared.ComputeCapacityReservationGroup.String(),
		UniqueAttribute:   "name",
		Attributes:        attributes,
		Scope:             scope,
		Tags:              azureshared.ConvertAzureTags(capacityReservationGroup.Tags),
		LinkedItemQueries: linkedItemQueries,
	}

	return sdpItem, nil
}

func (c *computeCapacityReservationGroupWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		ComputeCapacityReservationGroupLookupByName,
	}
}

func (c *computeCapacityReservationGroupWrapper) PotentialLinks() map[shared.ItemType]bool {
	return map[shared.ItemType]bool{
		azureshared.ComputeCapacityReservation: true,
		azureshared.ComputeVirtualMachine:      true,
	}
}

// ref: https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/capacity_reservation_group
func (c *computeCapacityReservationGroupWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "azurerm_capacity_reservation_group.name",
		},
	}
}

func (c *computeCapacityReservationGroupWrapper) IAMPermissions() []string {
	return []string{
		"Microsoft.Compute/capacityReservationGroups/read",
	}
}

func (c *computeCapacityReservationGroupWrapper) PredefinedRole() string {
	return "Reader"
}
