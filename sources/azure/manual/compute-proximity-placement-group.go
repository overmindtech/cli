package manual

import (
	"context"
	"errors"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/shared"
	"github.com/overmindtech/cli/sdpcache"
	"github.com/overmindtech/cli/discovery"
)

var ComputeProximityPlacementGroupLookupByName = shared.NewItemTypeLookup("name", azureshared.ComputeProximityPlacementGroup)

type computeProximityPlacementGroupWrapper struct {
	client clients.ProximityPlacementGroupsClient
	*azureshared.MultiResourceGroupBase
}

func NewComputeProximityPlacementGroup(client clients.ProximityPlacementGroupsClient, resourceGroupScopes []azureshared.ResourceGroupScope) sources.ListableWrapper {
	return &computeProximityPlacementGroupWrapper{
		client: client,
		MultiResourceGroupBase: azureshared.NewMultiResourceGroupBase(
			resourceGroupScopes,
			sdp.AdapterCategory_ADAPTER_CATEGORY_COMPUTE_APPLICATION,
			azureshared.ComputeProximityPlacementGroup,
		),
	}
}

// ref: https://learn.microsoft.com/en-us/rest/api/compute/proximity-placement-groups/list-by-resource-group?view=rest-compute-2025-04-01&tabs=HTTP
func (c computeProximityPlacementGroupWrapper) List(ctx context.Context, scope string) ([]*sdp.Item, *sdp.QueryError) {
	rgScope, err := c.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}
	pager := c.client.ListByResourceGroup(ctx, rgScope.ResourceGroup, nil)
	var items []*sdp.Item
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, azureshared.QueryError(err, scope, c.Type())
		}
		for _, proximityPlacementGroup := range page.Value {
			if proximityPlacementGroup.Name == nil {
				continue
			}
			item, sdpErr := c.azureProximityPlacementGroupToSDPItem(proximityPlacementGroup, scope)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}
	return items, nil
}

func (c computeProximityPlacementGroupWrapper) ListStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string) {
	rgScope, err := c.ResourceGroupScopeFromScope(scope)
	if err != nil {
		stream.SendError(azureshared.QueryError(err, scope, c.Type()))
		return
	}
	pager := c.client.ListByResourceGroup(ctx, rgScope.ResourceGroup, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			stream.SendError(azureshared.QueryError(err, scope, c.Type()))
			return
		}
		for _, proximityPlacementGroup := range page.Value {
			if proximityPlacementGroup.Name == nil {
				continue
			}
			item, sdpErr := c.azureProximityPlacementGroupToSDPItem(proximityPlacementGroup, scope)
			if sdpErr != nil {
				stream.SendError(sdpErr)
				continue
			}
			cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
			stream.SendItem(item)
		}
	}
}

// ref: https://learn.microsoft.com/en-us/rest/api/compute/proximity-placement-groups/get?view=rest-compute-2025-04-01&tabs=HTTP
func (c computeProximityPlacementGroupWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	rgScope, err := c.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}
	if len(queryParts) < 1 {
		return nil, azureshared.QueryError(errors.New("queryParts must be at least 1 and be the proximity placement group name"), scope, c.Type())
	}
	proximityPlacementGroupName := queryParts[0]
	resp, err := c.client.Get(ctx, rgScope.ResourceGroup, proximityPlacementGroupName, nil)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}
	return c.azureProximityPlacementGroupToSDPItem(&resp.ProximityPlacementGroup, scope)
}

func (c computeProximityPlacementGroupWrapper) azureProximityPlacementGroupToSDPItem(proximityPlacementGroup *armcompute.ProximityPlacementGroup, scope string) (*sdp.Item, *sdp.QueryError) {
	if proximityPlacementGroup.Name == nil {
		return nil, azureshared.QueryError(errors.New("proximityPlacementGroupName is nil"), scope, c.Type())
	}
	attributes, err := shared.ToAttributesWithExclude(proximityPlacementGroup, "tags")
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}
	sdpItem := &sdp.Item{
		Type:            azureshared.ComputeProximityPlacementGroup.String(),
		UniqueAttribute: "name",
		Attributes:      attributes,
		Scope:           scope,
		Tags:            azureshared.ConvertAzureTags(proximityPlacementGroup.Tags),
	}

	// Link to Virtual Machines in the proximity placement group
	// Reference: https://learn.microsoft.com/en-us/rest/api/compute/virtual-machines/get
	if proximityPlacementGroup.Properties != nil && proximityPlacementGroup.Properties.VirtualMachines != nil {
		for _, ref := range proximityPlacementGroup.Properties.VirtualMachines {
			if ref != nil && ref.ID != nil {
				vmName := azureshared.ExtractResourceName(*ref.ID)
				if vmName != "" {
					linkedScope := scope
					if extractedScope := azureshared.ExtractScopeFromResourceID(*ref.ID); extractedScope != "" {
						linkedScope = extractedScope
					}
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.ComputeVirtualMachine.String(),
							Method: sdp.QueryMethod_GET,
							Query:  vmName,
							Scope:  linkedScope,
						},
						BlastPropagation: &sdp.BlastPropagation{
							In:  true, // PPG change affects VM placement
							Out: true, // VM add/remove changes PPG membership
						},
					})
				}
			}
		}
	}

	// Link to Availability Sets in the proximity placement group
	// Reference: https://learn.microsoft.com/en-us/rest/api/compute/availability-sets/get
	if proximityPlacementGroup.Properties != nil && proximityPlacementGroup.Properties.AvailabilitySets != nil {
		for _, ref := range proximityPlacementGroup.Properties.AvailabilitySets {
			if ref != nil && ref.ID != nil {
				avSetName := azureshared.ExtractResourceName(*ref.ID)
				if avSetName != "" {
					linkedScope := scope
					if extractedScope := azureshared.ExtractScopeFromResourceID(*ref.ID); extractedScope != "" {
						linkedScope = extractedScope
					}
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.ComputeAvailabilitySet.String(),
							Method: sdp.QueryMethod_GET,
							Query:  avSetName,
							Scope:  linkedScope,
						},
						BlastPropagation: &sdp.BlastPropagation{
							In:  true, // PPG change affects Availability Set placement
							Out: true, // Availability Set add/remove changes PPG membership
						},
					})
				}
			}
		}
	}

	// Link to Virtual Machine Scale Sets in the proximity placement group
	// Reference: https://learn.microsoft.com/en-us/rest/api/compute/virtual-machine-scale-sets/get
	if proximityPlacementGroup.Properties != nil && proximityPlacementGroup.Properties.VirtualMachineScaleSets != nil {
		for _, ref := range proximityPlacementGroup.Properties.VirtualMachineScaleSets {
			if ref != nil && ref.ID != nil {
				vmssName := azureshared.ExtractResourceName(*ref.ID)
				if vmssName != "" {
					linkedScope := scope
					if extractedScope := azureshared.ExtractScopeFromResourceID(*ref.ID); extractedScope != "" {
						linkedScope = extractedScope
					}
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.ComputeVirtualMachineScaleSet.String(),
							Method: sdp.QueryMethod_GET,
							Query:  vmssName,
							Scope:  linkedScope,
						},
						BlastPropagation: &sdp.BlastPropagation{
							In:  true, // PPG change affects VMSS placement
							Out: true, // VMSS add/remove changes PPG membership
						},
					})
				}
			}
		}
	}

	return sdpItem, nil
}

func (c computeProximityPlacementGroupWrapper) PotentialLinks() map[shared.ItemType]bool {
	return map[shared.ItemType]bool{
		azureshared.ComputeVirtualMachine:         true,
		azureshared.ComputeAvailabilitySet:        true,
		azureshared.ComputeVirtualMachineScaleSet: true,
	}
}

// ref: https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/proximity_placement_group
func (c computeProximityPlacementGroupWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "azurerm_proximity_placement_group.name",
		},
	}
}

func (c computeProximityPlacementGroupWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		ComputeProximityPlacementGroupLookupByName,
	}
}
