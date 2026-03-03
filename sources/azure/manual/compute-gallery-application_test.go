package manual

import (
	"context"
	"errors"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"
	"go.uber.org/mock/gomock"

	"github.com/overmindtech/cli/go/discovery"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/azure/shared/mocks"
	"github.com/overmindtech/cli/sources/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

// mockGalleryApplicationsPager is a mock pager for ListByGallery.
type mockGalleryApplicationsPager struct {
	pages []armcompute.GalleryApplicationsClientListByGalleryResponse
	index int
}

func (m *mockGalleryApplicationsPager) More() bool {
	return m.index < len(m.pages)
}

func (m *mockGalleryApplicationsPager) NextPage(ctx context.Context) (armcompute.GalleryApplicationsClientListByGalleryResponse, error) {
	if m.index >= len(m.pages) {
		return armcompute.GalleryApplicationsClientListByGalleryResponse{}, errors.New("no more pages")
	}
	page := m.pages[m.index]
	m.index++
	return page, nil
}

// errorGalleryApplicationsPager is a mock pager that always returns an error.
type errorGalleryApplicationsPager struct{}

func (e *errorGalleryApplicationsPager) More() bool {
	return true
}

func (e *errorGalleryApplicationsPager) NextPage(ctx context.Context) (armcompute.GalleryApplicationsClientListByGalleryResponse, error) {
	return armcompute.GalleryApplicationsClientListByGalleryResponse{}, errors.New("pager error")
}

// testGalleryApplicationsClient wraps the mock and returns a pager from NewListByGalleryPager.
type testGalleryApplicationsClient struct {
	*mocks.MockGalleryApplicationsClient
	pager clients.GalleryApplicationsPager
}

func (t *testGalleryApplicationsClient) NewListByGalleryPager(resourceGroupName, galleryName string, options *armcompute.GalleryApplicationsClientListByGalleryOptions) clients.GalleryApplicationsPager {
	if t.pager != nil {
		return t.pager
	}
	return t.MockGalleryApplicationsClient.NewListByGalleryPager(resourceGroupName, galleryName, options)
}

func createAzureGalleryApplication(applicationName string) *armcompute.GalleryApplication {
	return &armcompute.GalleryApplication{
		Name:     new(applicationName),
		Location: new("eastus"),
		Tags: map[string]*string{
			"env": new("test"),
		},
		Properties: &armcompute.GalleryApplicationProperties{
			SupportedOSType: to.Ptr(armcompute.OperatingSystemTypesWindows),
			Description:     new("Test gallery application"),
		},
	}
}

func TestComputeGalleryApplication(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"
	scope := subscriptionID + "." + resourceGroup
	galleryName := "test-gallery"
	galleryApplicationName := "test-application"

	t.Run("Get", func(t *testing.T) {
		app := createAzureGalleryApplication(galleryApplicationName)

		mockClient := mocks.NewMockGalleryApplicationsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, galleryName, galleryApplicationName, nil).Return(
			armcompute.GalleryApplicationsClientGetResponse{
				GalleryApplication: *app,
			}, nil)

		wrapper := NewComputeGalleryApplication(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(galleryName, galleryApplicationName)
		sdpItem, qErr := adapter.Get(ctx, scope, query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.ComputeGalleryApplication.String() {
			t.Errorf("Expected type %s, got %s", azureshared.ComputeGalleryApplication.String(), sdpItem.GetType())
		}

		if sdpItem.GetUniqueAttribute() != "uniqueAttr" {
			t.Errorf("Expected unique attribute 'uniqueAttr', got %s", sdpItem.GetUniqueAttribute())
		}

		expectedUnique := shared.CompositeLookupKey(galleryName, galleryApplicationName)
		if sdpItem.UniqueAttributeValue() != expectedUnique {
			t.Errorf("Expected unique attribute value %s, got %s", expectedUnique, sdpItem.UniqueAttributeValue())
		}

		if sdpItem.GetTags()["env"] != "test" {
			t.Errorf("Expected tag env=test, got: %v", sdpItem.GetTags()["env"])
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{ExpectedType: azureshared.ComputeGallery.String(), ExpectedMethod: sdp.QueryMethod_GET, ExpectedQuery: galleryName, ExpectedScope: scope},
				{ExpectedType: azureshared.ComputeGalleryApplicationVersion.String(), ExpectedMethod: sdp.QueryMethod_SEARCH, ExpectedQuery: shared.CompositeLookupKey(galleryName, galleryApplicationName), ExpectedScope: scope},
			}
			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("Get_InvalidQueryParts", func(t *testing.T) {
		mockClient := mocks.NewMockGalleryApplicationsClient(ctrl)
		wrapper := NewComputeGalleryApplication(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, scope, galleryName, true)
		if qErr == nil {
			t.Error("Expected error when Get with wrong number of query parts, but got nil")
		}
	})

	t.Run("Get_EmptyGalleryName", func(t *testing.T) {
		mockClient := mocks.NewMockGalleryApplicationsClient(ctrl)
		wrapper := NewComputeGalleryApplication(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey("", galleryApplicationName)
		_, qErr := adapter.Get(ctx, scope, query, true)
		if qErr == nil {
			t.Error("Expected error when gallery name is empty, but got nil")
		}
	})

	t.Run("Get_EmptyApplicationName", func(t *testing.T) {
		mockClient := mocks.NewMockGalleryApplicationsClient(ctrl)
		wrapper := NewComputeGalleryApplication(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(galleryName, "")
		_, qErr := adapter.Get(ctx, scope, query, true)
		if qErr == nil {
			t.Error("Expected error when gallery application name is empty, but got nil")
		}
	})

	t.Run("Get_ClientError", func(t *testing.T) {
		expectedErr := errors.New("application not found")
		mockClient := mocks.NewMockGalleryApplicationsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, galleryName, "nonexistent", nil).Return(
			armcompute.GalleryApplicationsClientGetResponse{}, expectedErr)

		wrapper := NewComputeGalleryApplication(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(galleryName, "nonexistent")
		_, qErr := adapter.Get(ctx, scope, query, true)
		if qErr == nil {
			t.Error("Expected error when client returns error, but got nil")
		}
	})

	t.Run("Search", func(t *testing.T) {
		app1 := createAzureGalleryApplication("app-1")
		app2 := createAzureGalleryApplication("app-2")

		mockClient := mocks.NewMockGalleryApplicationsClient(ctrl)
		pages := []armcompute.GalleryApplicationsClientListByGalleryResponse{
			{
				GalleryApplicationList: armcompute.GalleryApplicationList{
					Value: []*armcompute.GalleryApplication{app1, app2},
				},
			},
		}
		mockPager := &mockGalleryApplicationsPager{pages: pages}
		testClient := &testGalleryApplicationsClient{
			MockGalleryApplicationsClient: mockClient,
			pager:                         mockPager,
		}

		wrapper := NewComputeGalleryApplication(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support Search operation")
		}

		sdpItems, err := searchable.Search(ctx, scope, galleryName, true)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if len(sdpItems) != 2 {
			t.Fatalf("Expected 2 items, got: %d", len(sdpItems))
		}

		for _, item := range sdpItems {
			if err := item.Validate(); err != nil {
				t.Errorf("Expected valid item, got: %v", err)
			}
		}
	})

	t.Run("Search_InvalidQueryParts", func(t *testing.T) {
		mockClient := mocks.NewMockGalleryApplicationsClient(ctrl)
		wrapper := NewComputeGalleryApplication(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support Search operation")
		}

		searchQuery := shared.CompositeLookupKey(galleryName, galleryApplicationName)
		_, err := searchable.Search(ctx, scope, searchQuery, true)
		if err == nil {
			t.Error("Expected error when Search with wrong number of query parts, but got nil")
		}
	})

	t.Run("Search_EmptyGalleryName", func(t *testing.T) {
		mockClient := mocks.NewMockGalleryApplicationsClient(ctrl)
		wrapper := NewComputeGalleryApplication(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		_, qErr := wrapper.Search(ctx, scope, "")
		if qErr == nil {
			t.Error("Expected error when gallery name is empty, but got nil")
		}
	})

	t.Run("Search_PagerError", func(t *testing.T) {
		mockClient := mocks.NewMockGalleryApplicationsClient(ctrl)
		errorPager := &errorGalleryApplicationsPager{}
		testClient := &testGalleryApplicationsClient{
			MockGalleryApplicationsClient: mockClient,
			pager:                         errorPager,
		}

		wrapper := NewComputeGalleryApplication(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support Search operation")
		}

		_, err := searchable.Search(ctx, scope, galleryName, true)
		if err == nil {
			t.Error("Expected error when pager returns error, but got nil")
		}
	})

	t.Run("PotentialLinks", func(t *testing.T) {
		mockClient := mocks.NewMockGalleryApplicationsClient(ctrl)
		wrapper := NewComputeGalleryApplication(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		links := wrapper.PotentialLinks()
		expected := map[shared.ItemType]bool{
			azureshared.ComputeGallery:                   true,
			azureshared.ComputeGalleryApplicationVersion: true,
			stdlib.NetworkDNS:                            true,
			stdlib.NetworkHTTP:                           true,
			stdlib.NetworkIP:                             true,
		}
		for itemType, want := range expected {
			if got := links[itemType]; got != want {
				t.Errorf("PotentialLinks()[%v] = %v, want %v", itemType, got, want)
			}
		}
	})

	t.Run("ImplementsSearchableAdapter", func(t *testing.T) {
		mockClient := mocks.NewMockGalleryApplicationsClient(ctrl)
		wrapper := NewComputeGalleryApplication(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Error("Adapter should implement SearchableAdapter interface")
		}
	})
}
