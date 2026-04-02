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
	"github.com/overmindtech/cli/sources/stdlib"
)

// mockElasticSanVolumePager is a simple mock implementation of ElasticSanVolumePager
type mockElasticSanVolumePager struct {
	pages []armelasticsan.VolumesClientListByVolumeGroupResponse
	index int
}

func (m *mockElasticSanVolumePager) More() bool {
	return m.index < len(m.pages)
}

func (m *mockElasticSanVolumePager) NextPage(ctx context.Context) (armelasticsan.VolumesClientListByVolumeGroupResponse, error) {
	if m.index >= len(m.pages) {
		return armelasticsan.VolumesClientListByVolumeGroupResponse{}, errors.New("no more pages")
	}
	page := m.pages[m.index]
	m.index++
	return page, nil
}

func createAzureElasticSanVolume(name string) *armelasticsan.Volume {
	provisioningState := armelasticsan.ProvisioningStatesSucceeded
	sizeGiB := int64(100)
	return &armelasticsan.Volume{
		ID:   new("/subscriptions/sub/resourceGroups/rg/providers/Microsoft.ElasticSan/elasticSans/es/volumegroups/vg/volumes/" + name),
		Name: new(name),
		Type: new("Microsoft.ElasticSan/elasticSans/volumegroups/volumes"),
		Properties: &armelasticsan.VolumeProperties{
			SizeGiB:           &sizeGiB,
			ProvisioningState: &provisioningState,
		},
	}
}

func createAzureElasticSanVolumeWithLinks(name string) *armelasticsan.Volume {
	vol := createAzureElasticSanVolume(name)
	vol.Properties.StorageTarget = &armelasticsan.IscsiTargetInfo{
		TargetPortalHostname: new("test-san.region.elasticsan.azure.net"),
		TargetIqn:            new("iqn.2022-05.net.azure.elasticsan:test"),
		TargetPortalPort:     new(int32(3260)),
	}
	vol.Properties.CreationData = &armelasticsan.SourceCreationData{
		CreateSource: new(armelasticsan.VolumeCreateOptionVolumeSnapshot),
		SourceID:     new("/subscriptions/sub/resourceGroups/rg/providers/Microsoft.ElasticSan/elasticSans/es/volumegroups/vg/snapshots/snap1"),
	}
	return vol
}

func TestElasticSanVolume(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"
	elasticSanName := "test-elastic-san"
	volumeGroupName := "test-volume-group"
	volumeName := "test-volume"

	t.Run("Get", func(t *testing.T) {
		vol := createAzureElasticSanVolume(volumeName)

		mockClient := mocks.NewMockElasticSanVolumeClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, elasticSanName, volumeGroupName, volumeName, nil).Return(
			armelasticsan.VolumesClientGetResponse{
				Volume: *vol,
			}, nil)

		wrapper := manual.NewElasticSanVolume(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(elasticSanName, volumeGroupName, volumeName)
		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.ElasticSanVolume.String() {
			t.Errorf("Expected type %s, got %s", azureshared.ElasticSanVolume.String(), sdpItem.GetType())
		}

		if sdpItem.GetUniqueAttribute() != "uniqueAttr" {
			t.Errorf("Expected unique attribute 'uniqueAttr', got %s", sdpItem.GetUniqueAttribute())
		}

		expectedUnique := shared.CompositeLookupKey(elasticSanName, volumeGroupName, volumeName)
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

	t.Run("GetWithLinks", func(t *testing.T) {
		vol := createAzureElasticSanVolumeWithLinks(volumeName)

		mockClient := mocks.NewMockElasticSanVolumeClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, elasticSanName, volumeGroupName, volumeName, nil).Return(
			armelasticsan.VolumesClientGetResponse{
				Volume: *vol,
			}, nil)

		wrapper := manual.NewElasticSanVolume(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(elasticSanName, volumeGroupName, volumeName)
		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		t.Run("StaticTests", func(t *testing.T) {
			scope := subscriptionID + "." + resourceGroup
			queryTests := shared.QueryTests{
				{ExpectedType: azureshared.ElasticSan.String(), ExpectedMethod: sdp.QueryMethod_GET, ExpectedQuery: elasticSanName, ExpectedScope: scope},
				{ExpectedType: azureshared.ElasticSanVolumeGroup.String(), ExpectedMethod: sdp.QueryMethod_GET, ExpectedQuery: shared.CompositeLookupKey(elasticSanName, volumeGroupName), ExpectedScope: scope},
				{ExpectedType: azureshared.ElasticSanVolumeSnapshot.String(), ExpectedMethod: sdp.QueryMethod_GET, ExpectedQuery: shared.CompositeLookupKey("es", "vg", "snap1"), ExpectedScope: "sub.rg"},
				{ExpectedType: stdlib.NetworkDNS.String(), ExpectedMethod: sdp.QueryMethod_SEARCH, ExpectedQuery: "test-san.region.elasticsan.azure.net", ExpectedScope: "global"},
			}
			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("GetWithInsufficientQueryParts", func(t *testing.T) {
		mockClient := mocks.NewMockElasticSanVolumeClient(ctrl)
		wrapper := manual.NewElasticSanVolume(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		// Only 2 query parts - missing volumeName
		query := shared.CompositeLookupKey(elasticSanName, volumeGroupName)
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr == nil {
			t.Error("Expected error when providing insufficient query parts, but got nil")
		}
	})

	t.Run("GetWithEmptyElasticSanName", func(t *testing.T) {
		mockClient := mocks.NewMockElasticSanVolumeClient(ctrl)
		wrapper := manual.NewElasticSanVolume(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey("", volumeGroupName, volumeName)
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr == nil {
			t.Error("Expected error when elasticSanName is empty, but got nil")
		}
	})

	t.Run("GetWithEmptyVolumeGroupName", func(t *testing.T) {
		mockClient := mocks.NewMockElasticSanVolumeClient(ctrl)
		wrapper := manual.NewElasticSanVolume(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(elasticSanName, "", volumeName)
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr == nil {
			t.Error("Expected error when volumeGroupName is empty, but got nil")
		}
	})

	t.Run("GetWithEmptyVolumeName", func(t *testing.T) {
		mockClient := mocks.NewMockElasticSanVolumeClient(ctrl)
		wrapper := manual.NewElasticSanVolume(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(elasticSanName, volumeGroupName, "")
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr == nil {
			t.Error("Expected error when volumeName is empty, but got nil")
		}
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		mockClient := mocks.NewMockElasticSanVolumeClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, elasticSanName, volumeGroupName, "nonexistent", nil).Return(
			armelasticsan.VolumesClientGetResponse{}, errors.New("volume not found"))

		wrapper := manual.NewElasticSanVolume(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(elasticSanName, volumeGroupName, "nonexistent")
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr == nil {
			t.Error("Expected error when resource not found, but got nil")
		}
	})

	t.Run("Search", func(t *testing.T) {
		vol1 := createAzureElasticSanVolume("vol-1")
		vol2 := createAzureElasticSanVolume("vol-2")

		mockClient := mocks.NewMockElasticSanVolumeClient(ctrl)
		mockPager := &mockElasticSanVolumePager{
			pages: []armelasticsan.VolumesClientListByVolumeGroupResponse{
				{
					VolumeList: armelasticsan.VolumeList{
						Value: []*armelasticsan.Volume{vol1, vol2},
					},
				},
			},
		}
		mockClient.EXPECT().NewListByVolumeGroupPager(resourceGroup, elasticSanName, volumeGroupName, nil).Return(mockPager)

		wrapper := manual.NewElasticSanVolume(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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

	t.Run("SearchWithEmptyElasticSanName", func(t *testing.T) {
		mockClient := mocks.NewMockElasticSanVolumeClient(ctrl)
		wrapper := manual.NewElasticSanVolume(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support Search operation")
		}

		query := shared.CompositeLookupKey("", volumeGroupName)
		_, err := searchable.Search(ctx, wrapper.Scopes()[0], query, true)
		if err == nil {
			t.Error("Expected error when elasticSanName is empty, but got nil")
		}
	})

	t.Run("SearchWithEmptyVolumeGroupName", func(t *testing.T) {
		mockClient := mocks.NewMockElasticSanVolumeClient(ctrl)
		wrapper := manual.NewElasticSanVolume(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support Search operation")
		}

		query := shared.CompositeLookupKey(elasticSanName, "")
		_, err := searchable.Search(ctx, wrapper.Scopes()[0], query, true)
		if err == nil {
			t.Error("Expected error when volumeGroupName is empty, but got nil")
		}
	})

	t.Run("SearchStream", func(t *testing.T) {
		vol := createAzureElasticSanVolume("stream-vol")
		mockClient := mocks.NewMockElasticSanVolumeClient(ctrl)
		mockPager := &mockElasticSanVolumePager{
			pages: []armelasticsan.VolumesClientListByVolumeGroupResponse{
				{
					VolumeList: armelasticsan.VolumeList{
						Value: []*armelasticsan.Volume{vol},
					},
				},
			},
		}
		mockClient.EXPECT().NewListByVolumeGroupPager(resourceGroup, elasticSanName, volumeGroupName, nil).Return(mockPager)

		wrapper := manual.NewElasticSanVolume(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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
		if items[0].GetType() != azureshared.ElasticSanVolume.String() {
			t.Errorf("Expected type %s, got %s", azureshared.ElasticSanVolume.String(), items[0].GetType())
		}
	})
}
