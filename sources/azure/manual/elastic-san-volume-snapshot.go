package manual

import (
	"context"
	"errors"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/elasticsan/armelasticsan"
	"github.com/overmindtech/cli/go/discovery"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/shared"
)

var (
	ElasticSanLookupByName               = shared.NewItemTypeLookup("name", azureshared.ElasticSan)
	ElasticSanVolumeGroupLookupByName    = shared.NewItemTypeLookup("name", azureshared.ElasticSanVolumeGroup)
	ElasticSanVolumeSnapshotLookupByName = shared.NewItemTypeLookup("name", azureshared.ElasticSanVolumeSnapshot)
)

type elasticSanVolumeSnapshotWrapper struct {
	client clients.ElasticSanVolumeSnapshotClient
	*azureshared.MultiResourceGroupBase
}

func NewElasticSanVolumeSnapshot(client clients.ElasticSanVolumeSnapshotClient, resourceGroupScopes []azureshared.ResourceGroupScope) sources.SearchableWrapper {
	return &elasticSanVolumeSnapshotWrapper{
		client: client,
		MultiResourceGroupBase: azureshared.NewMultiResourceGroupBase(
			resourceGroupScopes,
			sdp.AdapterCategory_ADAPTER_CATEGORY_STORAGE,
			azureshared.ElasticSanVolumeSnapshot,
		),
	}
}

func (s elasticSanVolumeSnapshotWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 3 {
		return nil, azureshared.QueryError(errors.New("Get requires 3 query parts: elasticSanName, volumeGroupName and snapshotName"), scope, s.Type())
	}
	elasticSanName := queryParts[0]
	if elasticSanName == "" {
		return nil, azureshared.QueryError(errors.New("elasticSanName cannot be empty"), scope, s.Type())
	}
	volumeGroupName := queryParts[1]
	if volumeGroupName == "" {
		return nil, azureshared.QueryError(errors.New("volumeGroupName cannot be empty"), scope, s.Type())
	}
	snapshotName := queryParts[2]
	if snapshotName == "" {
		return nil, azureshared.QueryError(errors.New("snapshotName cannot be empty"), scope, s.Type())
	}

	rgScope, err := s.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}
	resp, err := s.client.Get(ctx, rgScope.ResourceGroup, elasticSanName, volumeGroupName, snapshotName, nil)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}

	return s.azureSnapshotToSDPItem(&resp.Snapshot, elasticSanName, volumeGroupName, snapshotName, scope)
}

func (s elasticSanVolumeSnapshotWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		ElasticSanLookupByName,
		ElasticSanVolumeGroupLookupByName,
		ElasticSanVolumeSnapshotLookupByName,
	}
}

func (s elasticSanVolumeSnapshotWrapper) Search(ctx context.Context, scope string, queryParts ...string) ([]*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 2 {
		return nil, azureshared.QueryError(errors.New("Search requires 2 query parts: elasticSanName and volumeGroupName"), scope, s.Type())
	}
	elasticSanName := queryParts[0]
	if elasticSanName == "" {
		return nil, azureshared.QueryError(errors.New("elasticSanName cannot be empty"), scope, s.Type())
	}
	volumeGroupName := queryParts[1]
	if volumeGroupName == "" {
		return nil, azureshared.QueryError(errors.New("volumeGroupName cannot be empty"), scope, s.Type())
	}

	rgScope, err := s.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}
	pager := s.client.ListByVolumeGroup(ctx, rgScope.ResourceGroup, elasticSanName, volumeGroupName, nil)

	var items []*sdp.Item
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, azureshared.QueryError(err, scope, s.Type())
		}
		for _, snapshot := range page.Value {
			if snapshot.Name == nil {
				continue
			}
			item, sdpErr := s.azureSnapshotToSDPItem(snapshot, elasticSanName, volumeGroupName, *snapshot.Name, scope)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}
	return items, nil
}

func (s elasticSanVolumeSnapshotWrapper) SearchStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string, queryParts ...string) {
	if len(queryParts) < 2 {
		stream.SendError(azureshared.QueryError(errors.New("Search requires 2 query parts: elasticSanName and volumeGroupName"), scope, s.Type()))
		return
	}
	elasticSanName := queryParts[0]
	if elasticSanName == "" {
		stream.SendError(azureshared.QueryError(errors.New("elasticSanName cannot be empty"), scope, s.Type()))
		return
	}
	volumeGroupName := queryParts[1]
	if volumeGroupName == "" {
		stream.SendError(azureshared.QueryError(errors.New("volumeGroupName cannot be empty"), scope, s.Type()))
		return
	}

	rgScope, err := s.ResourceGroupScopeFromScope(scope)
	if err != nil {
		stream.SendError(azureshared.QueryError(err, scope, s.Type()))
		return
	}
	pager := s.client.ListByVolumeGroup(ctx, rgScope.ResourceGroup, elasticSanName, volumeGroupName, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			stream.SendError(azureshared.QueryError(err, scope, s.Type()))
			return
		}
		for _, snapshot := range page.Value {
			if snapshot.Name == nil {
				continue
			}
			item, sdpErr := s.azureSnapshotToSDPItem(snapshot, elasticSanName, volumeGroupName, *snapshot.Name, scope)
			if sdpErr != nil {
				stream.SendError(sdpErr)
				continue
			}
			cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
			stream.SendItem(item)
		}
	}
}

func (s elasticSanVolumeSnapshotWrapper) SearchLookups() []sources.ItemTypeLookups {
	return []sources.ItemTypeLookups{
		{
			ElasticSanLookupByName,
			ElasticSanVolumeGroupLookupByName,
		},
	}
}

func (s elasticSanVolumeSnapshotWrapper) azureSnapshotToSDPItem(snapshot *armelasticsan.Snapshot, elasticSanName, volumeGroupName, snapshotName, scope string) (*sdp.Item, *sdp.QueryError) {
	if snapshot.Name == nil {
		return nil, azureshared.QueryError(errors.New("snapshot name is nil"), scope, s.Type())
	}
	attributes, err := shared.ToAttributesWithExclude(snapshot, "tags")
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}
	err = attributes.Set("uniqueAttr", shared.CompositeLookupKey(elasticSanName, volumeGroupName, snapshotName))
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}

	sdpItem := &sdp.Item{
		Type:              azureshared.ElasticSanVolumeSnapshot.String(),
		UniqueAttribute:   "uniqueAttr",
		Attributes:        attributes,
		Scope:             scope,
		LinkedItemQueries: []*sdp.LinkedItemQuery{},
	}

	// Link to parent Elastic SAN
	sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   azureshared.ElasticSan.String(),
			Method: sdp.QueryMethod_GET,
			Query:  elasticSanName,
			Scope:  scope,
		},
	})

	// Link to parent Volume Group
	sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   azureshared.ElasticSanVolumeGroup.String(),
			Method: sdp.QueryMethod_GET,
			Query:  shared.CompositeLookupKey(elasticSanName, volumeGroupName),
			Scope:  scope,
		},
	})

	// Link to source volume from CreationData.SourceID
	if snapshot.Properties != nil && snapshot.Properties.CreationData != nil && snapshot.Properties.CreationData.SourceID != nil && *snapshot.Properties.CreationData.SourceID != "" {
		sourceID := *snapshot.Properties.CreationData.SourceID
		parts := azureshared.ExtractPathParamsFromResourceID(sourceID, []string{"elasticSans", "volumegroups", "volumes"})
		if len(parts) >= 3 {
			extractedScope := azureshared.ExtractScopeFromResourceID(sourceID)
			if extractedScope == "" {
				extractedScope = scope
			}
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   azureshared.ElasticSanVolume.String(),
					Method: sdp.QueryMethod_GET,
					Query:  shared.CompositeLookupKey(parts[0], parts[1], parts[2]),
					Scope:  extractedScope,
				},
			})
		}
	}

	if snapshot.Properties != nil && snapshot.Properties.ProvisioningState != nil {
		switch *snapshot.Properties.ProvisioningState {
		case armelasticsan.ProvisioningStatesSucceeded:
			sdpItem.Health = sdp.Health_HEALTH_OK.Enum()
		case armelasticsan.ProvisioningStatesCreating, armelasticsan.ProvisioningStatesUpdating, armelasticsan.ProvisioningStatesDeleting,
			armelasticsan.ProvisioningStatesPending, armelasticsan.ProvisioningStatesRestoring:
			sdpItem.Health = sdp.Health_HEALTH_PENDING.Enum()
		case armelasticsan.ProvisioningStatesFailed, armelasticsan.ProvisioningStatesCanceled,
			armelasticsan.ProvisioningStatesDeleted, armelasticsan.ProvisioningStatesInvalid:
			sdpItem.Health = sdp.Health_HEALTH_ERROR.Enum()
		default:
			sdpItem.Health = sdp.Health_HEALTH_UNKNOWN.Enum()
		}
	}

	return sdpItem, nil
}

func (s elasticSanVolumeSnapshotWrapper) PotentialLinks() map[shared.ItemType]bool {
	return map[shared.ItemType]bool{
		azureshared.ElasticSan:            true,
		azureshared.ElasticSanVolumeGroup: true,
		azureshared.ElasticSanVolume:      true,
	}
}

func (s elasticSanVolumeSnapshotWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "azurerm_elastic_san_volume_snapshot.id",
		},
	}
}

func (s elasticSanVolumeSnapshotWrapper) IAMPermissions() []string {
	return []string{
		"Microsoft.ElasticSan/elasticSans/volumegroups/snapshots/read",
	}
}

func (s elasticSanVolumeSnapshotWrapper) PredefinedRole() string {
	return "Reader"
}
