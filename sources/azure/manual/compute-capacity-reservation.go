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

var ComputeCapacityReservationLookupByName = shared.NewItemTypeLookup("name", azureshared.ComputeCapacityReservation)

type computeCapacityReservationWrapper struct {
	client clients.CapacityReservationsClient
	*azureshared.MultiResourceGroupBase
}

func NewComputeCapacityReservation(client clients.CapacityReservationsClient, resourceGroupScopes []azureshared.ResourceGroupScope) sources.SearchableWrapper {
	return &computeCapacityReservationWrapper{
		client: client,
		MultiResourceGroupBase: azureshared.NewMultiResourceGroupBase(
			resourceGroupScopes,
			sdp.AdapterCategory_ADAPTER_CATEGORY_COMPUTE_APPLICATION,
			azureshared.ComputeCapacityReservation,
		),
	}
}

func capacityReservationGetOptions() *armcompute.CapacityReservationsClientGetOptions {
	expand := armcompute.CapacityReservationInstanceViewTypesInstanceView
	return &armcompute.CapacityReservationsClientGetOptions{
		Expand: &expand,
	}
}

// ref: https://learn.microsoft.com/en-us/rest/api/compute/capacity-reservations/get?view=rest-compute-2025-04-01&tabs=HTTP
func (c *computeCapacityReservationWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) != 2 {
		return nil, azureshared.QueryError(errors.New("queryParts must be exactly 2: capacity reservation group name and capacity reservation name"), scope, c.Type())
	}
	groupName := queryParts[0]
	if groupName == "" {
		return nil, azureshared.QueryError(errors.New("capacity reservation group name cannot be empty"), scope, c.Type())
	}
	reservationName := queryParts[1]
	if reservationName == "" {
		return nil, azureshared.QueryError(errors.New("capacity reservation name cannot be empty"), scope, c.Type())
	}

	rgScope, err := c.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}
	resp, err := c.client.Get(ctx, rgScope.ResourceGroup, groupName, reservationName, capacityReservationGetOptions())
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}
	return c.azureCapacityReservationToSDPItem(&resp.CapacityReservation, groupName, scope)
}

// ref: https://learn.microsoft.com/en-us/rest/api/compute/capacity-reservations/list-by-capacity-reservation-group?view=rest-compute-2025-04-01&tabs=HTTP
func (c *computeCapacityReservationWrapper) Search(ctx context.Context, scope string, queryParts ...string) ([]*sdp.Item, *sdp.QueryError) {
	if len(queryParts) != 1 {
		return nil, azureshared.QueryError(errors.New("queryParts must be exactly 1: capacity reservation group name"), scope, c.Type())
	}
	groupName := queryParts[0]
	if groupName == "" {
		return nil, azureshared.QueryError(errors.New("capacity reservation group name cannot be empty"), scope, c.Type())
	}

	rgScope, err := c.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}
	pager := c.client.NewListByCapacityReservationGroupPager(rgScope.ResourceGroup, groupName, nil)

	var items []*sdp.Item
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, azureshared.QueryError(err, scope, c.Type())
		}
		for _, res := range page.Value {
			if res == nil || res.Name == nil {
				continue
			}
			item, sdpErr := c.azureCapacityReservationToSDPItem(res, groupName, scope)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}
	return items, nil
}

func (c *computeCapacityReservationWrapper) SearchStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string, queryParts ...string) {
	if len(queryParts) != 1 {
		stream.SendError(azureshared.QueryError(errors.New("queryParts must be exactly 1: capacity reservation group name"), scope, c.Type()))
		return
	}
	groupName := queryParts[0]
	if groupName == "" {
		stream.SendError(azureshared.QueryError(errors.New("capacity reservation group name cannot be empty"), scope, c.Type()))
		return
	}

	rgScope, err := c.ResourceGroupScopeFromScope(scope)
	if err != nil {
		stream.SendError(azureshared.QueryError(err, scope, c.Type()))
		return
	}

	pager := c.client.NewListByCapacityReservationGroupPager(rgScope.ResourceGroup, groupName, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			stream.SendError(azureshared.QueryError(err, scope, c.Type()))
			return
		}
		for _, res := range page.Value {
			if res == nil || res.Name == nil {
				continue
			}
			item, sdpErr := c.azureCapacityReservationToSDPItem(res, groupName, scope)
			if sdpErr != nil {
				stream.SendError(sdpErr)
				continue
			}
			cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
			stream.SendItem(item)
		}
	}
}

func (c *computeCapacityReservationWrapper) azureCapacityReservationToSDPItem(res *armcompute.CapacityReservation, groupName, scope string) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(res, "tags")
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}

	if res.Name == nil {
		return nil, azureshared.QueryError(errors.New("capacity reservation name is nil"), scope, c.Type())
	}
	reservationName := *res.Name
	if reservationName == "" {
		return nil, azureshared.QueryError(errors.New("capacity reservation name cannot be empty"), scope, c.Type())
	}
	if err := attributes.Set("uniqueAttr", shared.CompositeLookupKey(groupName, reservationName)); err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}

	linkedItemQueries := make([]*sdp.LinkedItemQuery, 0)

	// Parent: capacity reservation group
	linkedItemQueries = append(linkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   azureshared.ComputeCapacityReservationGroup.String(),
			Method: sdp.QueryMethod_GET,
			Query:  groupName,
			Scope:  scope,
		},
	})

	// VMs associated with this capacity reservation
	if res.Properties != nil && res.Properties.VirtualMachinesAssociated != nil {
		for _, vmRef := range res.Properties.VirtualMachinesAssociated {
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

	// VMs physically allocated to this capacity reservation (from instance view; only populated when Get uses $expand=instanceView)
	if res.Properties != nil && res.Properties.InstanceView != nil && res.Properties.InstanceView.UtilizationInfo != nil && res.Properties.InstanceView.UtilizationInfo.VirtualMachinesAllocated != nil {
		for _, vmRef := range res.Properties.InstanceView.UtilizationInfo.VirtualMachinesAllocated {
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
		Type:              azureshared.ComputeCapacityReservation.String(),
		UniqueAttribute:   "uniqueAttr",
		Attributes:        attributes,
		Scope:             scope,
		Tags:              azureshared.ConvertAzureTags(res.Tags),
		LinkedItemQueries: linkedItemQueries,
	}

	// Health status from ProvisioningState
	if res.Properties != nil && res.Properties.ProvisioningState != nil {
		state := strings.ToLower(*res.Properties.ProvisioningState)
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

func (c *computeCapacityReservationWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		ComputeCapacityReservationGroupLookupByName,
		ComputeCapacityReservationLookupByName,
	}
}

func (c *computeCapacityReservationWrapper) SearchLookups() []sources.ItemTypeLookups {
	return []sources.ItemTypeLookups{
		{
			ComputeCapacityReservationGroupLookupByName,
		},
	}
}

func (c *computeCapacityReservationWrapper) PotentialLinks() map[shared.ItemType]bool {
	return map[shared.ItemType]bool{
		azureshared.ComputeCapacityReservationGroup: true,
		azureshared.ComputeVirtualMachine:            true,
	}
}

// ref: https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/capacity_reservation
func (c *computeCapacityReservationWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_SEARCH,
			TerraformQueryMap: "azurerm_capacity_reservation.id",
		},
	}
}

func (c *computeCapacityReservationWrapper) IAMPermissions() []string {
	return []string{
		"Microsoft.Compute/capacityReservationGroups/capacityReservations/read",
	}
}

func (c *computeCapacityReservationWrapper) PredefinedRole() string {
	return "Reader"
}
