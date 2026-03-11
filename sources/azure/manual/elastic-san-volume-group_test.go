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

// mockElasticSanVolumeGroupPager is a simple mock implementation of ElasticSanVolumeGroupPager
type mockElasticSanVolumeGroupPager struct {
	pages []armelasticsan.VolumeGroupsClientListByElasticSanResponse
	index int
}

func (m *mockElasticSanVolumeGroupPager) More() bool {
	return m.index < len(m.pages)
}

func (m *mockElasticSanVolumeGroupPager) NextPage(ctx context.Context) (armelasticsan.VolumeGroupsClientListByElasticSanResponse, error) {
	if m.index >= len(m.pages) {
		return armelasticsan.VolumeGroupsClientListByElasticSanResponse{}, errors.New("no more pages")
	}
	page := m.pages[m.index]
	m.index++
	return page, nil
}

func createAzureElasticSanVolumeGroup(name string) *armelasticsan.VolumeGroup {
	provisioningState := armelasticsan.ProvisioningStatesSucceeded
	return &armelasticsan.VolumeGroup{
		ID:   new("/subscriptions/sub/resourceGroups/rg/providers/Microsoft.ElasticSan/elasticSans/es/volumegroups/" + name),
		Name: new(name),
		Type: new("Microsoft.ElasticSan/elasticSans/volumegroups"),
		Properties: &armelasticsan.VolumeGroupProperties{
			ProvisioningState: &provisioningState,
		},
	}
}

func TestElasticSanVolumeGroup(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"
	elasticSanName := "test-elastic-san"
	volumeGroupName := "test-volume-group"

	t.Run("Get", func(t *testing.T) {
		vg := createAzureElasticSanVolumeGroup(volumeGroupName)

		mockClient := mocks.NewMockElasticSanVolumeGroupClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, elasticSanName, volumeGroupName, nil).Return(
			armelasticsan.VolumeGroupsClientGetResponse{
				VolumeGroup: *vg,
			}, nil)

		wrapper := manual.NewElasticSanVolumeGroup(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(elasticSanName, volumeGroupName)
		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.ElasticSanVolumeGroup.String() {
			t.Errorf("Expected type %s, got %s", azureshared.ElasticSanVolumeGroup.String(), sdpItem.GetType())
		}

		if sdpItem.GetUniqueAttribute() != "uniqueAttr" {
			t.Errorf("Expected unique attribute 'uniqueAttr', got %s", sdpItem.GetUniqueAttribute())
		}

		expectedUnique := shared.CompositeLookupKey(elasticSanName, volumeGroupName)
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
				{ExpectedType: azureshared.ElasticSanVolumeSnapshot.String(), ExpectedMethod: sdp.QueryMethod_SEARCH, ExpectedQuery: shared.CompositeLookupKey(elasticSanName, volumeGroupName), ExpectedScope: scope},
				{ExpectedType: azureshared.ElasticSanVolume.String(), ExpectedMethod: sdp.QueryMethod_SEARCH, ExpectedQuery: shared.CompositeLookupKey(elasticSanName, volumeGroupName), ExpectedScope: scope},
			}
			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("GetWithInsufficientQueryParts", func(t *testing.T) {
		mockClient := mocks.NewMockElasticSanVolumeGroupClient(ctrl)
		wrapper := manual.NewElasticSanVolumeGroup(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], elasticSanName, true)
		if qErr == nil {
			t.Error("Expected error when providing insufficient query parts, but got nil")
		}
	})

	t.Run("GetWithEmptyName", func(t *testing.T) {
		mockClient := mocks.NewMockElasticSanVolumeGroupClient(ctrl)
		wrapper := manual.NewElasticSanVolumeGroup(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(elasticSanName, "")
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr == nil {
			t.Error("Expected error when volume group name is empty, but got nil")
		}
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		mockClient := mocks.NewMockElasticSanVolumeGroupClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, elasticSanName, "nonexistent", nil).Return(
			armelasticsan.VolumeGroupsClientGetResponse{}, errors.New("volume group not found"))

		wrapper := manual.NewElasticSanVolumeGroup(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(elasticSanName, "nonexistent")
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr == nil {
			t.Error("Expected error when resource not found, but got nil")
		}
	})

	t.Run("Search", func(t *testing.T) {
		vg1 := createAzureElasticSanVolumeGroup("vg-1")
		vg2 := createAzureElasticSanVolumeGroup("vg-2")

		mockClient := mocks.NewMockElasticSanVolumeGroupClient(ctrl)
		mockPager := &mockElasticSanVolumeGroupPager{
			pages: []armelasticsan.VolumeGroupsClientListByElasticSanResponse{
				{
					VolumeGroupList: armelasticsan.VolumeGroupList{
						Value: []*armelasticsan.VolumeGroup{vg1, vg2},
					},
				},
			},
		}
		mockClient.EXPECT().NewListByElasticSanPager(resourceGroup, elasticSanName, nil).Return(mockPager)

		wrapper := manual.NewElasticSanVolumeGroup(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support Search operation")
		}

		query := elasticSanName
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
		vg := createAzureElasticSanVolumeGroup("stream-vg")
		mockClient := mocks.NewMockElasticSanVolumeGroupClient(ctrl)
		mockPager := &mockElasticSanVolumeGroupPager{
			pages: []armelasticsan.VolumeGroupsClientListByElasticSanResponse{
				{
					VolumeGroupList: armelasticsan.VolumeGroupList{
						Value: []*armelasticsan.VolumeGroup{vg},
					},
				},
			},
		}
		mockClient.EXPECT().NewListByElasticSanPager(resourceGroup, elasticSanName, nil).Return(mockPager)

		wrapper := manual.NewElasticSanVolumeGroup(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		streamable, ok := adapter.(discovery.SearchStreamableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support SearchStream operation")
		}

		query := elasticSanName
		stream := discovery.NewRecordingQueryResultStream()
		streamable.SearchStream(ctx, wrapper.Scopes()[0], query, true, stream)
		items := stream.GetItems()
		if len(items) != 1 {
			t.Fatalf("Expected 1 item from stream, got %d", len(items))
		}
		if items[0].GetType() != azureshared.ElasticSanVolumeGroup.String() {
			t.Errorf("Expected type %s, got %s", azureshared.ElasticSanVolumeGroup.String(), items[0].GetType())
		}
	})
}
