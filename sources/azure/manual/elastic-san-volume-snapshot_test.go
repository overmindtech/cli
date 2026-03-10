package manual_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/elasticsan/armelasticsan"
	"go.uber.org/mock/gomock"

	"github.com/overmindtech/cli/go/discovery"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/manual"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/azure/shared/mocks"
	"github.com/overmindtech/cli/sources/shared"
)

// mockElasticSanVolumeSnapshotPager is a simple mock implementation of ElasticSanVolumeSnapshotPager
type mockElasticSanVolumeSnapshotPager struct {
	pages []armelasticsan.VolumeSnapshotsClientListByVolumeGroupResponse
	index int
}

func (m *mockElasticSanVolumeSnapshotPager) More() bool {
	return m.index < len(m.pages)
}

func (m *mockElasticSanVolumeSnapshotPager) NextPage(ctx context.Context) (armelasticsan.VolumeSnapshotsClientListByVolumeGroupResponse, error) {
	if m.index >= len(m.pages) {
		return armelasticsan.VolumeSnapshotsClientListByVolumeGroupResponse{}, errors.New("no more pages")
	}
	page := m.pages[m.index]
	m.index++
	return page, nil
}

func createAzureElasticSanSnapshot(name string) *armelasticsan.Snapshot {
	provisioningState := armelasticsan.ProvisioningStatesSucceeded
	return &armelasticsan.Snapshot{
		ID:   new("/subscriptions/sub/resourceGroups/rg/providers/Microsoft.ElasticSan/elasticSans/es/volumegroups/vg/snapshots/" + name),
		Name: new(name),
		Type: new("Microsoft.ElasticSan/elasticSans/volumegroups/snapshots"),
		Properties: &armelasticsan.SnapshotProperties{
			ProvisioningState: &provisioningState,
			CreationData:      &armelasticsan.SnapshotCreationData{},
		},
	}
}

func TestElasticSanVolumeSnapshot(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"
	elasticSanName := "test-elastic-san"
	volumeGroupName := "test-volume-group"
	snapshotName := "test-snapshot"

	t.Run("Get", func(t *testing.T) {
		snapshot := createAzureElasticSanSnapshot(snapshotName)

		mockClient := mocks.NewMockElasticSanVolumeSnapshotClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, elasticSanName, volumeGroupName, snapshotName, nil).Return(
			armelasticsan.VolumeSnapshotsClientGetResponse{
				Snapshot: *snapshot,
			}, nil)

		wrapper := manual.NewElasticSanVolumeSnapshot(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(elasticSanName, volumeGroupName, snapshotName)
		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.ElasticSanVolumeSnapshot.String() {
			t.Errorf("Expected type %s, got %s", azureshared.ElasticSanVolumeSnapshot.String(), sdpItem.GetType())
		}

		if sdpItem.GetUniqueAttribute() != "uniqueAttr" {
			t.Errorf("Expected unique attribute 'uniqueAttr', got %s", sdpItem.GetUniqueAttribute())
		}

		expectedUnique := shared.CompositeLookupKey(elasticSanName, volumeGroupName, snapshotName)
		if sdpItem.UniqueAttributeValue() != expectedUnique {
			t.Errorf("Expected unique attribute value %s, got %s", expectedUnique, sdpItem.UniqueAttributeValue())
		}

		if sdpItem.GetScope() != subscriptionID+"."+resourceGroup {
			t.Errorf("Expected scope %s, got %s", subscriptionID+"."+resourceGroup, sdpItem.GetScope())
		}

		if err := sdpItem.Validate(); err != nil {
			t.Fatalf("Expected no validation error, got: %v", err)
		}

		t.Run("StaticTests", func(t *testing.T) {
			scope := subscriptionID + "." + resourceGroup
			queryTests := shared.QueryTests{
				{ExpectedType: azureshared.ElasticSan.String(), ExpectedMethod: sdp.QueryMethod_GET, ExpectedQuery: elasticSanName, ExpectedScope: scope},
				{ExpectedType: azureshared.ElasticSanVolumeGroup.String(), ExpectedMethod: sdp.QueryMethod_GET, ExpectedQuery: shared.CompositeLookupKey(elasticSanName, volumeGroupName), ExpectedScope: scope},
			}
			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("GetWithInsufficientQueryParts", func(t *testing.T) {
		mockClient := mocks.NewMockElasticSanVolumeSnapshotClient(ctrl)
		wrapper := manual.NewElasticSanVolumeSnapshot(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], elasticSanName, true)
		if qErr == nil {
			t.Error("Expected error when providing insufficient query parts, but got nil")
		}
	})

	t.Run("GetWithEmptyName", func(t *testing.T) {
		mockClient := mocks.NewMockElasticSanVolumeSnapshotClient(ctrl)
		wrapper := manual.NewElasticSanVolumeSnapshot(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(elasticSanName, volumeGroupName, "")
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr == nil {
			t.Error("Expected error when snapshot name is empty, but got nil")
		}
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		mockClient := mocks.NewMockElasticSanVolumeSnapshotClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, elasticSanName, volumeGroupName, "nonexistent", nil).Return(
			armelasticsan.VolumeSnapshotsClientGetResponse{}, errors.New("snapshot not found"))

		wrapper := manual.NewElasticSanVolumeSnapshot(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(elasticSanName, volumeGroupName, "nonexistent")
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr == nil {
			t.Error("Expected error when resource not found, but got nil")
		}
	})

	t.Run("Search", func(t *testing.T) {
		snapshot1 := createAzureElasticSanSnapshot("snap-1")
		snapshot2 := createAzureElasticSanSnapshot("snap-2")

		mockClient := mocks.NewMockElasticSanVolumeSnapshotClient(ctrl)
		mockPager := &mockElasticSanVolumeSnapshotPager{
			pages: []armelasticsan.VolumeSnapshotsClientListByVolumeGroupResponse{
				{
					SnapshotList: armelasticsan.SnapshotList{
						Value: []*armelasticsan.Snapshot{snapshot1, snapshot2},
					},
				},
			},
		}
		mockClient.EXPECT().ListByVolumeGroup(ctx, resourceGroup, elasticSanName, volumeGroupName, nil).Return(mockPager)

		wrapper := manual.NewElasticSanVolumeSnapshot(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support Search operation")
		}

		query := shared.CompositeLookupKey(elasticSanName, volumeGroupName)
		items, err := searchable.Search(ctx, wrapper.Scopes()[0], query, true)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if len(items) != 2 {
			t.Fatalf("Expected 2 items, got %d", len(items))
		}
		for _, item := range items {
			if err := item.Validate(); err != nil {
				t.Fatalf("Expected no validation error, got: %v", err)
			}
		}
	})

	t.Run("SearchStream", func(t *testing.T) {
		snapshot := createAzureElasticSanSnapshot("stream-snap")
		mockClient := mocks.NewMockElasticSanVolumeSnapshotClient(ctrl)
		mockPager := &mockElasticSanVolumeSnapshotPager{
			pages: []armelasticsan.VolumeSnapshotsClientListByVolumeGroupResponse{
				{
					SnapshotList: armelasticsan.SnapshotList{
						Value: []*armelasticsan.Snapshot{snapshot},
					},
				},
			},
		}
		mockClient.EXPECT().ListByVolumeGroup(ctx, resourceGroup, elasticSanName, volumeGroupName, nil).Return(mockPager)

		wrapper := manual.NewElasticSanVolumeSnapshot(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		streamable, ok := adapter.(discovery.SearchStreamableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support SearchStream operation")
		}

		query := shared.CompositeLookupKey(elasticSanName, volumeGroupName)
		stream := discovery.NewRecordingQueryResultStream()
		streamable.SearchStream(ctx, wrapper.Scopes()[0], query, true, stream)
		items := stream.GetItems()
		if len(items) != 1 {
			t.Fatalf("Expected 1 item from stream, got %d", len(items))
		}
		if items[0].GetType() != azureshared.ElasticSanVolumeSnapshot.String() {
			t.Errorf("Expected type %s, got %s", azureshared.ElasticSanVolumeSnapshot.String(), items[0].GetType())
		}
	})
}
