package manual

import (
	"context"
	"errors"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v6"
	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/shared"
)

var ComputeAvailabilitySetLookupByName = shared.NewItemTypeLookup("name", azureshared.ComputeAvailabilitySet)

type computeAvailabilitySetWrapper struct {
	client clients.AvailabilitySetsClient

	*azureshared.ResourceGroupBase
}

func NewComputeAvailabilitySet(client clients.AvailabilitySetsClient, subscriptionID, resourceGroup string) sources.ListableWrapper {
	return &computeAvailabilitySetWrapper{
		client: client,
		ResourceGroupBase: azureshared.NewResourceGroupBase(
			subscriptionID,
			resourceGroup,
			sdp.AdapterCategory_ADAPTER_CATEGORY_COMPUTE_APPLICATION,
			azureshared.ComputeAvailabilitySet,
		),
	}
}

// ref: https://learn.microsoft.com/en-us/rest/api/compute/availability-sets/list?view=rest-compute-2025-04-01&tabs=HTTP
func (c computeAvailabilitySetWrapper) List(ctx context.Context) ([]*sdp.Item, *sdp.QueryError) {
	pager := c.client.NewListPager(c.ResourceGroup(), nil)

	var items []*sdp.Item
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, azureshared.QueryError(err, c.DefaultScope(), c.Type())
		}
		for _, availabilitySet := range page.Value {
			if availabilitySet.Name == nil {
				continue
			}
			item, sdpErr := c.azureAvailabilitySetToSDPItem(availabilitySet)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}

	return items, nil
}

func (c computeAvailabilitySetWrapper) ListStream(ctx context.Context, stream discovery.QueryResultStream, cache *sdpcache.Cache, cacheKey sdpcache.CacheKey) {
	pager := c.client.NewListPager(c.ResourceGroup(), nil)

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			stream.SendError(azureshared.QueryError(err, c.DefaultScope(), c.Type()))
			return
		}

		for _, availabilitySet := range page.Value {
			if availabilitySet.Name == nil {
				continue
			}
			var sdpErr *sdp.QueryError
			var item *sdp.Item
			item, sdpErr = c.azureAvailabilitySetToSDPItem(availabilitySet)
			if sdpErr != nil {
				stream.SendError(sdpErr)
				continue
			}
			cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
			stream.SendItem(item)
		}
	}
}

// ref : https://learn.microsoft.com/en-us/rest/api/compute/availability-sets/get?view=rest-compute-2025-04-01&tabs=HTTP
func (c computeAvailabilitySetWrapper) Get(ctx context.Context, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 1 {
		return nil, azureshared.QueryError(errors.New("queryParts must be at least 1 and be the availability set name"), c.DefaultScope(), c.Type())
	}
	availabilitySetName := queryParts[0]
	if availabilitySetName == "" {
		return nil, azureshared.QueryError(errors.New("availabilitySetName cannot be empty"), c.DefaultScope(), c.Type())
	}

	availabilitySet, err := c.client.Get(ctx, c.ResourceGroup(), availabilitySetName, nil)
	if err != nil {
		return nil, azureshared.QueryError(err, c.DefaultScope(), c.Type())
	}
	return c.azureAvailabilitySetToSDPItem(&availabilitySet.AvailabilitySet)
}

func (c computeAvailabilitySetWrapper) azureAvailabilitySetToSDPItem(availabilitySet *armcompute.AvailabilitySet) (*sdp.Item, *sdp.QueryError) {
	if availabilitySet.Name == nil {
		return nil, azureshared.QueryError(errors.New("availabilitySetName is nil"), c.DefaultScope(), c.Type())
	}
	attributes, err := shared.ToAttributesWithExclude(availabilitySet, "tags")
	if err != nil {
		return nil, azureshared.QueryError(err, c.DefaultScope(), c.Type())
	}

	sdpItem := &sdp.Item{
		Type:            azureshared.ComputeAvailabilitySet.String(),
		UniqueAttribute: "name",
		Attributes:      attributes,
		Scope:           c.DefaultScope(),
		Tags:            azureshared.ConvertAzureTags(availabilitySet.Tags),
	}

	// Link to Proximity Placement Group
	// Reference: https://learn.microsoft.com/en-us/rest/api/compute/proximity-placement-groups/get
	if availabilitySet.Properties != nil && availabilitySet.Properties.ProximityPlacementGroup != nil && availabilitySet.Properties.ProximityPlacementGroup.ID != nil {
		ppgName := azureshared.ExtractResourceName(*availabilitySet.Properties.ProximityPlacementGroup.ID)
		if ppgName != "" {
			scope := c.DefaultScope()
			// Check if Proximity Placement Group is in a different resource group
			if extractedScope := azureshared.ExtractScopeFromResourceID(*availabilitySet.Properties.ProximityPlacementGroup.ID); extractedScope != "" {
				scope = extractedScope
			}
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   azureshared.ComputeProximityPlacementGroup.String(),
					Method: sdp.QueryMethod_GET,
					Query:  ppgName,
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,  // If PPG changes → Availability Set placement changes (In: true)
					Out: false, // If Availability Set is deleted → PPG remains (Out: false)
				},
			})
		}
	}

	// Link to Virtual Machines
	// Reference: https://learn.microsoft.com/en-us/rest/api/compute/virtual-machines/get
	if availabilitySet.Properties != nil && availabilitySet.Properties.VirtualMachines != nil {
		for _, vmRef := range availabilitySet.Properties.VirtualMachines {
			if vmRef != nil && vmRef.ID != nil {
				vmName := azureshared.ExtractResourceName(*vmRef.ID)
				if vmName != "" {
					scope := c.DefaultScope()
					// Check if Virtual Machine is in a different resource group
					if extractedScope := azureshared.ExtractScopeFromResourceID(*vmRef.ID); extractedScope != "" {
						scope = extractedScope
					}
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.ComputeVirtualMachine.String(),
							Method: sdp.QueryMethod_GET,
							Query:  vmName,
							Scope:  scope,
						},
						BlastPropagation: &sdp.BlastPropagation{
							In:  true,  // If VM changes → Availability Set membership changes (In: true)
							Out: false, // If Availability Set is deleted → VMs remain but lose availability set association (Out: false)
						},
					})
				}
			}
		}
	}

	return sdpItem, nil
}

func (c computeAvailabilitySetWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		ComputeAvailabilitySetLookupByName,
	}
}

func (c computeAvailabilitySetWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		azureshared.ComputeProximityPlacementGroup,
		azureshared.ComputeVirtualMachine,
	)
}

// ref: https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/availability_set
func (c computeAvailabilitySetWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "azurerm_availability_set.name",
		},
	}
}

// ref: https://learn.microsoft.com/en-us/azure/role-based-access-control/permissions/compute#microsoftcompute
func (c computeAvailabilitySetWrapper) IAMPermissions() []string {
	return []string{
		"Microsoft.Compute/availabilitySets/read",
	}
}

// ref: https://learn.microsoft.com/en-us/azure/role-based-access-control/built-in-roles/compute
func (c computeAvailabilitySetWrapper) PredefinedRole() string {
	return "Reader" //there is no predefined role for availability sets, so we use the most restrictive role (Reader)
}
