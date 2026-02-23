package manual_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"
	"go.uber.org/mock/gomock"

	"github.com/overmindtech/cli/go/discovery"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	"github.com/overmindtech/cli/sources/azure/manual"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/azure/shared/mocks"
	"github.com/overmindtech/cli/sources/shared"
)

func TestComputeGallery(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"
	scope := subscriptionID + "." + resourceGroup

	t.Run("Get", func(t *testing.T) {
		galleryName := "test-gallery"
		gallery := createAzureGallery(galleryName)

		mockClient := mocks.NewMockGalleriesClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, galleryName, nil).Return(
			armcompute.GalleriesClientGetResponse{
				Gallery: *gallery,
			}, nil)

		wrapper := manual.NewComputeGallery(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, scope, galleryName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.ComputeGallery.String() {
			t.Errorf("Expected type %s, got %s", azureshared.ComputeGallery.String(), sdpItem.GetType())
		}

		if sdpItem.GetUniqueAttribute() != "name" {
			t.Errorf("Expected unique attribute 'name', got %s", sdpItem.GetUniqueAttribute())
		}

		if sdpItem.UniqueAttributeValue() != galleryName {
			t.Errorf("Expected unique attribute value %s, got %s", galleryName, sdpItem.UniqueAttributeValue())
		}

		if sdpItem.GetTags()["env"] != "test" {
			t.Errorf("Expected tag 'env=test', got: %v", sdpItem.GetTags()["env"])
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					ExpectedType:   azureshared.ComputeGalleryImage.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  galleryName,
					ExpectedScope:  scope,
				},
			}
			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("List", func(t *testing.T) {
		gallery1 := createAzureGallery("test-gallery-1")
		gallery2 := createAzureGallery("test-gallery-2")

		mockClient := mocks.NewMockGalleriesClient(ctrl)
		mockPager := newMockGalleriesPager(ctrl, []*armcompute.Gallery{gallery1, gallery2})

		mockClient.EXPECT().NewListByResourceGroupPager(resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewComputeGallery(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		listable, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support List operation")
		}

		sdpItems, err := listable.List(ctx, scope, true)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if len(sdpItems) != 2 {
			t.Fatalf("Expected 2 items, got: %d", len(sdpItems))
		}

		for _, item := range sdpItems {
			if item.Validate() != nil {
				t.Fatalf("Expected no validation error, got: %v", item.Validate())
			}
		}
	})

	t.Run("ListStream", func(t *testing.T) {
		gallery1 := createAzureGallery("test-gallery-1")
		gallery2 := createAzureGallery("test-gallery-2")

		mockClient := mocks.NewMockGalleriesClient(ctrl)
		mockPager := newMockGalleriesPager(ctrl, []*armcompute.Gallery{gallery1, gallery2})

		mockClient.EXPECT().NewListByResourceGroupPager(resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewComputeGallery(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		wg := &sync.WaitGroup{}
		wg.Add(2)

		var items []*sdp.Item
		mockItemHandler := func(item *sdp.Item) {
			items = append(items, item)
			wg.Done()
		}

		var errs []error
		mockErrorHandler := func(err error) {
			errs = append(errs, err)
		}

		stream := discovery.NewQueryResultStream(mockItemHandler, mockErrorHandler)

		listStreamable, ok := adapter.(discovery.ListStreamableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support ListStream operation")
		}

		listStreamable.ListStream(ctx, scope, true, stream)
		wg.Wait()

		if len(errs) != 0 {
			t.Fatalf("Expected no errors, got: %v", errs)
		}

		if len(items) != 2 {
			t.Fatalf("Expected 2 items, got: %d", len(items))
		}
	})

	t.Run("ListWithNilName", func(t *testing.T) {
		gallery1 := createAzureGallery("test-gallery-1")
		galleryNilName := &armcompute.Gallery{
			Name:     nil,
			Location: to.Ptr("eastus"),
			Tags: map[string]*string{
				"env": to.Ptr("test"),
			},
		}

		mockClient := mocks.NewMockGalleriesClient(ctrl)
		mockPager := newMockGalleriesPager(ctrl, []*armcompute.Gallery{gallery1, galleryNilName})

		mockClient.EXPECT().NewListByResourceGroupPager(resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewComputeGallery(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		listable, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support List operation")
		}

		sdpItems, err := listable.List(ctx, scope, true)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if len(sdpItems) != 1 {
			t.Fatalf("Expected 1 item (nil name skipped), got: %d", len(sdpItems))
		}
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		expectedErr := errors.New("gallery not found")

		mockClient := mocks.NewMockGalleriesClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, "nonexistent-gallery", nil).Return(
			armcompute.GalleriesClientGetResponse{}, expectedErr)

		wrapper := manual.NewComputeGallery(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, scope, "nonexistent-gallery", true)
		if qErr == nil {
			t.Error("Expected error when getting non-existent gallery, but got nil")
		}
	})

	t.Run("GetWithEmptyName", func(t *testing.T) {
		mockClient := mocks.NewMockGalleriesClient(ctrl)

		wrapper := manual.NewComputeGallery(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, scope, "", true)
		if qErr == nil {
			t.Error("Expected error when getting gallery with empty name, but got nil")
		}
	})

	t.Run("GetWithInsufficientQueryParts", func(t *testing.T) {
		mockClient := mocks.NewMockGalleriesClient(ctrl)

		wrapper := manual.NewComputeGallery(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		_, qErr := wrapper.Get(ctx, scope)
		if qErr == nil {
			t.Error("Expected error when getting gallery with insufficient query parts, but got nil")
		}
	})
}

func createAzureGallery(galleryName string) *armcompute.Gallery {
	return &armcompute.Gallery{
		Name:     to.Ptr(galleryName),
		Location: to.Ptr("eastus"),
		Tags: map[string]*string{
			"env":     to.Ptr("test"),
			"project": to.Ptr("testing"),
		},
		Properties: &armcompute.GalleryProperties{
			Description: to.Ptr("Test shared image gallery"),
			Identifier: &armcompute.GalleryIdentifier{
				UniqueName: to.Ptr("unique-" + galleryName),
			},
			ProvisioningState: to.Ptr(armcompute.GalleryProvisioningStateSucceeded),
		},
	}
}

type mockGalleriesPager struct {
	ctrl  *gomock.Controller
	items []*armcompute.Gallery
	index int
	more  bool
}

func newMockGalleriesPager(ctrl *gomock.Controller, items []*armcompute.Gallery) clients.GalleriesPager {
	return &mockGalleriesPager{
		ctrl:  ctrl,
		items: items,
		index: 0,
		more:  len(items) > 0,
	}
}

func (m *mockGalleriesPager) More() bool {
	return m.more
}

func (m *mockGalleriesPager) NextPage(ctx context.Context) (armcompute.GalleriesClientListByResourceGroupResponse, error) {
	if m.index >= len(m.items) {
		m.more = false
		return armcompute.GalleriesClientListByResourceGroupResponse{
			GalleryList: armcompute.GalleryList{
				Value: []*armcompute.Gallery{},
			},
		}, nil
	}

	item := m.items[m.index]
	m.index++
	m.more = m.index < len(m.items)

	return armcompute.GalleriesClientListByResourceGroupResponse{
		GalleryList: armcompute.GalleryList{
			Value: []*armcompute.Gallery{item},
		},
	}, nil
}
