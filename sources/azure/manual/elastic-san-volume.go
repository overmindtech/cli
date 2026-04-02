package manual

import (
	"context"
	"errors"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/elasticsan/armelasticsan"
	"github.com/overmindtech/cli/go/discovery"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

var ElasticSanVolumeLookupByName = shared.NewItemTypeLookup("name", azureshared.ElasticSanVolume)

type elasticSanVolumeWrapper struct {
	client clients.ElasticSanVolumeClient
	*azureshared.MultiResourceGroupBase
}

func NewElasticSanVolume(client clients.ElasticSanVolumeClient, resourceGroupScopes []azureshared.ResourceGroupScope) sources.SearchableWrapper {
	return &elasticSanVolumeWrapper{
		client: client,
		MultiResourceGroupBase: azureshared.NewMultiResourceGroupBase(
			resourceGroupScopes,
			sdp.AdapterCategory_ADAPTER_CATEGORY_STORAGE,
			azureshared.ElasticSanVolume,
		),
	}
}

func (e elasticSanVolumeWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 3 {
		return nil, azureshared.QueryError(errors.New("Get requires 3 query parts: elasticSanName, volumeGroupName, and volumeName"), scope, e.Type())
	}
	elasticSanName := queryParts[0]
	if elasticSanName == "" {
		return nil, azureshared.QueryError(errors.New("elasticSanName cannot be empty"), scope, e.Type())
	}
	volumeGroupName := queryParts[1]
	if volumeGroupName == "" {
		return nil, azureshared.QueryError(errors.New("volumeGroupName cannot be empty"), scope, e.Type())
	}
	volumeName := queryParts[2]
	if volumeName == "" {
		return nil, azureshared.QueryError(errors.New("volumeName cannot be empty"), scope, e.Type())
	}

	rgScope, err := e.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, e.Type())
	}
	resp, err := e.client.Get(ctx, rgScope.ResourceGroup, elasticSanName, volumeGroupName, volumeName, nil)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, e.Type())
	}
	return e.azureVolumeToSDPItem(&resp.Volume, elasticSanName, volumeGroupName, volumeName, scope)
}

func (e elasticSanVolumeWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		ElasticSanLookupByName,
		ElasticSanVolumeGroupLookupByName,
		ElasticSanVolumeLookupByName,
	}
}

func (e elasticSanVolumeWrapper) Search(ctx context.Context, scope string, queryParts ...string) ([]*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 2 {
		return nil, azureshared.QueryError(errors.New("Search requires 2 query parts: elasticSanName and volumeGroupName"), scope, e.Type())
	}
	elasticSanName := queryParts[0]
	if elasticSanName == "" {
		return nil, azureshared.QueryError(errors.New("elasticSanName cannot be empty"), scope, e.Type())
	}
	volumeGroupName := queryParts[1]
	if volumeGroupName == "" {
		return nil, azureshared.QueryError(errors.New("volumeGroupName cannot be empty"), scope, e.Type())
	}

	rgScope, err := e.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, e.Type())
	}
	pager := e.client.NewListByVolumeGroupPager(rgScope.ResourceGroup, elasticSanName, volumeGroupName, nil)

	var items []*sdp.Item
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, azureshared.QueryError(err, scope, e.Type())
		}
		for _, vol := range page.Value {
			if vol.Name == nil {
				continue
			}
			item, sdpErr := e.azureVolumeToSDPItem(vol, elasticSanName, volumeGroupName, *vol.Name, scope)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}
	return items, nil
}

func (e elasticSanVolumeWrapper) SearchStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string, queryParts ...string) {
	if len(queryParts) < 2 {
		stream.SendError(azureshared.QueryError(errors.New("Search requires 2 query parts: elasticSanName and volumeGroupName"), scope, e.Type()))
		return
	}
	elasticSanName := queryParts[0]
	if elasticSanName == "" {
		stream.SendError(azureshared.QueryError(errors.New("elasticSanName cannot be empty"), scope, e.Type()))
		return
	}
	volumeGroupName := queryParts[1]
	if volumeGroupName == "" {
		stream.SendError(azureshared.QueryError(errors.New("volumeGroupName cannot be empty"), scope, e.Type()))
		return
	}

	rgScope, err := e.ResourceGroupScopeFromScope(scope)
	if err != nil {
		stream.SendError(azureshared.QueryError(err, scope, e.Type()))
		return
	}
	pager := e.client.NewListByVolumeGroupPager(rgScope.ResourceGroup, elasticSanName, volumeGroupName, nil)

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			stream.SendError(azureshared.QueryError(err, scope, e.Type()))
			return
		}
		for _, vol := range page.Value {
			if vol.Name == nil {
				continue
			}
			item, sdpErr := e.azureVolumeToSDPItem(vol, elasticSanName, volumeGroupName, *vol.Name, scope)
			if sdpErr != nil {
				stream.SendError(sdpErr)
				continue
			}
			cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
			stream.SendItem(item)
		}
	}
}

func (e elasticSanVolumeWrapper) SearchLookups() []sources.ItemTypeLookups {
	return []sources.ItemTypeLookups{
		{ElasticSanLookupByName, ElasticSanVolumeGroupLookupByName},
	}
}

func (e elasticSanVolumeWrapper) azureVolumeToSDPItem(vol *armelasticsan.Volume, elasticSanName, volumeGroupName, volumeName, scope string) (*sdp.Item, *sdp.QueryError) {
	if vol.Name == nil {
		return nil, azureshared.QueryError(errors.New("volume name is nil"), scope, e.Type())
	}
	attributes, err := shared.ToAttributesWithExclude(vol, "tags")
	if err != nil {
		return nil, azureshared.QueryError(err, scope, e.Type())
	}
	err = attributes.Set("uniqueAttr", shared.CompositeLookupKey(elasticSanName, volumeGroupName, volumeName))
	if err != nil {
		return nil, azureshared.QueryError(err, scope, e.Type())
	}

	item := &sdp.Item{
		Type:              azureshared.ElasticSanVolume.String(),
		UniqueAttribute:   "uniqueAttr",
		Attributes:        attributes,
		Scope:             scope,
		LinkedItemQueries: []*sdp.LinkedItemQuery{},
	}

	// Link to parent Elastic SAN
	item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   azureshared.ElasticSan.String(),
			Method: sdp.QueryMethod_GET,
			Query:  elasticSanName,
			Scope:  scope,
		},
	})

	// Link to parent Volume Group
	item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   azureshared.ElasticSanVolumeGroup.String(),
			Method: sdp.QueryMethod_GET,
			Query:  shared.CompositeLookupKey(elasticSanName, volumeGroupName),
			Scope:  scope,
		},
	})

	if vol.Properties != nil {
		// Link to source resource (snapshot or volume) via CreationData.SourceID
		if vol.Properties.CreationData != nil && vol.Properties.CreationData.SourceID != nil && *vol.Properties.CreationData.SourceID != "" {
			sourceID := *vol.Properties.CreationData.SourceID
			// Determine the type based on the resource ID path
			// Azure REST API uses /snapshots/ for Elastic SAN volume snapshots
			if strings.Contains(sourceID, "/snapshots/") {
				// It's a snapshot - extract elasticSanName, volumeGroupName, snapshotName
				params := azureshared.ExtractPathParamsFromResourceID(sourceID, []string{"elasticSans", "volumegroups", "snapshots"})
				if len(params) >= 3 && params[0] != "" && params[1] != "" && params[2] != "" {
					linkedScope := azureshared.ExtractScopeFromResourceID(sourceID)
					if linkedScope == "" {
						linkedScope = scope
					}
					item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.ElasticSanVolumeSnapshot.String(),
							Method: sdp.QueryMethod_GET,
							Query:  shared.CompositeLookupKey(params[0], params[1], params[2]),
							Scope:  linkedScope,
						},
					})
				}
			} else if strings.Contains(sourceID, "/volumes/") {
				// It's a volume - extract elasticSanName, volumeGroupName, volumeName
				params := azureshared.ExtractPathParamsFromResourceID(sourceID, []string{"elasticSans", "volumegroups", "volumes"})
				if len(params) >= 3 && params[0] != "" && params[1] != "" && params[2] != "" {
					linkedScope := azureshared.ExtractScopeFromResourceID(sourceID)
					if linkedScope == "" {
						linkedScope = scope
					}
					item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.ElasticSanVolume.String(),
							Method: sdp.QueryMethod_GET,
							Query:  shared.CompositeLookupKey(params[0], params[1], params[2]),
							Scope:  linkedScope,
						},
					})
				}
			}
		}

		// Link to managed-by resource via ManagedBy.ResourceID
		if vol.Properties.ManagedBy != nil && vol.Properties.ManagedBy.ResourceID != nil && *vol.Properties.ManagedBy.ResourceID != "" {
			managedByID := *vol.Properties.ManagedBy.ResourceID
			// ManagedBy can reference different resource types (e.g., AKS clusters, VMs)
			// We'll use the generic resource name extraction and link appropriately
			linkedScope := azureshared.ExtractScopeFromResourceID(managedByID)
			if linkedScope == "" {
				linkedScope = scope
			}

			// Detect the resource type based on the path
			if strings.Contains(managedByID, "/virtualMachines/") {
				vmName := azureshared.ExtractResourceName(managedByID)
				if vmName != "" {
					item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.ComputeVirtualMachine.String(),
							Method: sdp.QueryMethod_GET,
							Query:  vmName,
							Scope:  linkedScope,
						},
					})
				}
			}
			// Add other resource types as needed
		}

		// Link to storage target DNS/hostname if available
		if vol.Properties.StorageTarget != nil {
			if vol.Properties.StorageTarget.TargetPortalHostname != nil && *vol.Properties.StorageTarget.TargetPortalHostname != "" {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   stdlib.NetworkDNS.String(),
						Method: sdp.QueryMethod_SEARCH,
						Query:  *vol.Properties.StorageTarget.TargetPortalHostname,
						Scope:  "global",
					},
				})
			}
		}
	}

	// Health from provisioning state
	if vol.Properties != nil && vol.Properties.ProvisioningState != nil {
		switch *vol.Properties.ProvisioningState {
		case armelasticsan.ProvisioningStatesSucceeded:
			item.Health = sdp.Health_HEALTH_OK.Enum()
		case armelasticsan.ProvisioningStatesCreating, armelasticsan.ProvisioningStatesUpdating, armelasticsan.ProvisioningStatesDeleting,
			armelasticsan.ProvisioningStatesPending, armelasticsan.ProvisioningStatesRestoring:
			item.Health = sdp.Health_HEALTH_PENDING.Enum()
		case armelasticsan.ProvisioningStatesFailed, armelasticsan.ProvisioningStatesCanceled,
			armelasticsan.ProvisioningStatesDeleted, armelasticsan.ProvisioningStatesInvalid:
			item.Health = sdp.Health_HEALTH_ERROR.Enum()
		default:
			item.Health = sdp.Health_HEALTH_UNKNOWN.Enum()
		}
	}

	return item, nil
}

func (e elasticSanVolumeWrapper) PotentialLinks() map[shared.ItemType]bool {
	return map[shared.ItemType]bool{
		azureshared.ElasticSan:               true,
		azureshared.ElasticSanVolumeGroup:    true,
		azureshared.ElasticSanVolumeSnapshot: true,
		azureshared.ElasticSanVolume:         true,
		azureshared.ComputeVirtualMachine:    true,
		stdlib.NetworkDNS:                    true,
	}
}

// ref: https://learn.microsoft.com/en-us/azure/role-based-access-control/resource-provider-operations#microsoftelasticsan
func (e elasticSanVolumeWrapper) IAMPermissions() []string {
	return []string{
		"Microsoft.ElasticSan/elasticSans/volumegroups/volumes/read",
	}
}

func (e elasticSanVolumeWrapper) PredefinedRole() string {
	return "Reader"
}
