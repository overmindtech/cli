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
	"github.com/overmindtech/cli/sources/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

// mockGalleryImagesPager is a mock pager for ListByGallery.
type mockGalleryImagesPager struct {
	pages []armcompute.GalleryImagesClientListByGalleryResponse
	index int
}

func (m *mockGalleryImagesPager) More() bool {
	return m.index < len(m.pages)
}

func (m *mockGalleryImagesPager) NextPage(ctx context.Context) (armcompute.GalleryImagesClientListByGalleryResponse, error) {
	if m.index >= len(m.pages) {
		return armcompute.GalleryImagesClientListByGalleryResponse{}, errors.New("no more pages")
	}
	page := m.pages[m.index]
	m.index++
	return page, nil
}

// errorGalleryImagesPager is a mock pager that always returns an error.
type errorGalleryImagesPager struct{}

func (e *errorGalleryImagesPager) More() bool {
	return true
}

func (e *errorGalleryImagesPager) NextPage(ctx context.Context) (armcompute.GalleryImagesClientListByGalleryResponse, error) {
	return armcompute.GalleryImagesClientListByGalleryResponse{}, errors.New("pager error")
}

// testGalleryImagesClient wraps the mock and returns a pager from NewListByGalleryPager.
type testGalleryImagesClient struct {
	*MockGalleryImagesClient
	pager clients.GalleryImagesPager
}

// NewListByGalleryPager returns the test pager so we don't need to mock this call.
func (t *testGalleryImagesClient) NewListByGalleryPager(resourceGroupName, galleryName string, options *armcompute.GalleryImagesClientListByGalleryOptions) clients.GalleryImagesPager {
	if t.pager != nil {
		return t.pager
	}
	return t.MockGalleryImagesClient.NewListByGalleryPager(resourceGroupName, galleryName, options)
}

func createAzureGalleryImage(imageName string) *armcompute.GalleryImage {
	return &armcompute.GalleryImage{
		Name:     to.Ptr(imageName),
		Location: to.Ptr("eastus"),
		Tags: map[string]*string{
			"env": to.Ptr("test"),
		},
		Properties: &armcompute.GalleryImageProperties{
			Identifier: &armcompute.GalleryImageIdentifier{
				Publisher: to.Ptr("test-publisher"),
				Offer:     to.Ptr("test-offer"),
				SKU:       to.Ptr("test-sku"),
			},
			OSType:  to.Ptr(armcompute.OperatingSystemTypesLinux),
			OSState: to.Ptr(armcompute.OperatingSystemStateTypesGeneralized),
		},
	}
}

func createAzureGalleryImageWithURIs(imageName string) *armcompute.GalleryImage {
	img := createAzureGalleryImage(imageName)
	img.Properties.Eula = to.Ptr("https://eula.example.com/terms")
	img.Properties.PrivacyStatementURI = to.Ptr("https://example.com/privacy")
	img.Properties.ReleaseNoteURI = to.Ptr("https://releases.example.com/notes")
	return img
}

func TestComputeGalleryImage(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"
	scope := subscriptionID + "." + resourceGroup
	galleryName := "test-gallery"
	galleryImageName := "test-image"

	t.Run("Get", func(t *testing.T) {
		image := createAzureGalleryImage(galleryImageName)

		mockClient := NewMockGalleryImagesClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, galleryName, galleryImageName, nil).Return(
			armcompute.GalleryImagesClientGetResponse{
				GalleryImage: *image,
			}, nil)

		wrapper := NewComputeGalleryImage(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(galleryName, galleryImageName)
		sdpItem, qErr := adapter.Get(ctx, scope, query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.ComputeGalleryImage.String() {
			t.Errorf("Expected type %s, got %s", azureshared.ComputeGalleryImage.String(), sdpItem.GetType())
		}

		if sdpItem.GetUniqueAttribute() != "uniqueAttr" {
			t.Errorf("Expected unique attribute 'uniqueAttr', got %s", sdpItem.GetUniqueAttribute())
		}

		expectedUnique := shared.CompositeLookupKey(galleryName, galleryImageName)
		if sdpItem.UniqueAttributeValue() != expectedUnique {
			t.Errorf("Expected unique attribute value %s, got %s", expectedUnique, sdpItem.UniqueAttributeValue())
		}

		if sdpItem.GetTags()["env"] != "test" {
			t.Errorf("Expected tag env=test, got: %v", sdpItem.GetTags()["env"])
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{ExpectedType: azureshared.ComputeGallery.String(), ExpectedMethod: sdp.QueryMethod_GET, ExpectedQuery: galleryName, ExpectedScope: scope},
			}
			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("GetWithURIs", func(t *testing.T) {
		image := createAzureGalleryImageWithURIs(galleryImageName)

		mockClient := NewMockGalleryImagesClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, galleryName, galleryImageName, nil).Return(
			armcompute.GalleryImagesClientGetResponse{
				GalleryImage: *image,
			}, nil)

		wrapper := NewComputeGalleryImage(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(galleryName, galleryImageName)
		sdpItem, qErr := adapter.Get(ctx, scope, query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{ExpectedType: azureshared.ComputeGallery.String(), ExpectedMethod: sdp.QueryMethod_GET, ExpectedQuery: galleryName, ExpectedScope: scope},
				{ExpectedType: stdlib.NetworkHTTP.String(), ExpectedMethod: sdp.QueryMethod_SEARCH, ExpectedQuery: "https://eula.example.com/terms", ExpectedScope: "global"},
				{ExpectedType: stdlib.NetworkDNS.String(), ExpectedMethod: sdp.QueryMethod_SEARCH, ExpectedQuery: "eula.example.com", ExpectedScope: "global"},
				{ExpectedType: stdlib.NetworkHTTP.String(), ExpectedMethod: sdp.QueryMethod_SEARCH, ExpectedQuery: "https://example.com/privacy", ExpectedScope: "global"},
				{ExpectedType: stdlib.NetworkDNS.String(), ExpectedMethod: sdp.QueryMethod_SEARCH, ExpectedQuery: "example.com", ExpectedScope: "global"},
				{ExpectedType: stdlib.NetworkHTTP.String(), ExpectedMethod: sdp.QueryMethod_SEARCH, ExpectedQuery: "https://releases.example.com/notes", ExpectedScope: "global"},
				{ExpectedType: stdlib.NetworkDNS.String(), ExpectedMethod: sdp.QueryMethod_SEARCH, ExpectedQuery: "releases.example.com", ExpectedScope: "global"},
			}
			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("Get_PlainTextEula_NoLinks", func(t *testing.T) {
		image := createAzureGalleryImage(galleryImageName)
		image.Properties.Eula = to.Ptr("This software is provided as-is. No warranty.")

		mockClient := NewMockGalleryImagesClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, galleryName, galleryImageName, nil).Return(
			armcompute.GalleryImagesClientGetResponse{
				GalleryImage: *image,
			}, nil)

		wrapper := NewComputeGalleryImage(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(galleryName, galleryImageName)
		sdpItem, qErr := adapter.Get(ctx, scope, query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		// Plain-text Eula should not generate HTTP/DNS/IP links
		for _, q := range sdpItem.GetLinkedItemQueries() {
			lq := q.GetQuery()
			if lq == nil {
				continue
			}
			typ := lq.GetType()
			if typ == stdlib.NetworkHTTP.String() || typ == stdlib.NetworkDNS.String() || typ == stdlib.NetworkIP.String() {
				t.Errorf("Plain-text Eula must not create network links; found linked query type %s with query %s", typ, lq.GetQuery())
			}
		}
	})

	t.Run("Get_SameHostDeduplication", func(t *testing.T) {
		image := createAzureGalleryImage(galleryImageName)
		image.Properties.PrivacyStatementURI = to.Ptr("https://example.com/privacy")
		image.Properties.ReleaseNoteURI = to.Ptr("https://example.com/release-notes")

		mockClient := NewMockGalleryImagesClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, galleryName, galleryImageName, nil).Return(
			armcompute.GalleryImagesClientGetResponse{
				GalleryImage: *image,
			}, nil)

		wrapper := NewComputeGalleryImage(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(galleryName, galleryImageName)
		sdpItem, qErr := adapter.Get(ctx, scope, query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		// Should have 2 HTTP links (one per URI) but only 1 DNS link (same hostname)
		httpCount := 0
		dnsCount := 0
		for _, q := range sdpItem.GetLinkedItemQueries() {
			query := q.GetQuery()
			if query != nil {
				if query.GetType() == stdlib.NetworkHTTP.String() {
					httpCount++
				}
				if query.GetType() == stdlib.NetworkDNS.String() {
					dnsCount++
				}
			}
		}
		if httpCount != 2 {
			t.Errorf("Expected 2 HTTP links, got %d", httpCount)
		}
		if dnsCount != 1 {
			t.Errorf("Expected 1 DNS link (deduped), got %d", dnsCount)
		}
	})

	t.Run("Get_IPHost_EmitsIPLink", func(t *testing.T) {
		image := createAzureGalleryImage(galleryImageName)
		image.Properties.PrivacyStatementURI = to.Ptr("https://192.168.1.10:8443/privacy")

		mockClient := NewMockGalleryImagesClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, galleryName, galleryImageName, nil).Return(
			armcompute.GalleryImagesClientGetResponse{
				GalleryImage: *image,
			}, nil)

		wrapper := NewComputeGalleryImage(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(galleryName, galleryImageName)
		sdpItem, qErr := adapter.Get(ctx, scope, query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		hasIP := false
		for _, q := range sdpItem.GetLinkedItemQueries() {
			query := q.GetQuery()
			if query != nil && query.GetType() == stdlib.NetworkIP.String() {
				hasIP = true
				if query.GetMethod() != sdp.QueryMethod_GET {
					t.Errorf("Expected NetworkIP link to use GET, got %v", query.GetMethod())
				}
				if query.GetScope() != "global" {
					t.Errorf("Expected NetworkIP link scope global, got %s", query.GetScope())
				}
				if query.GetQuery() != "192.168.1.10" {
					t.Errorf("Expected NetworkIP link query 192.168.1.10, got %s", query.GetQuery())
				}
				break
			}
		}
		if !hasIP {
			t.Error("Expected NetworkIP linked query when PrivacyStatementURI host is an IP address")
		}
	})

	t.Run("Get_InvalidQueryParts", func(t *testing.T) {
		mockClient := NewMockGalleryImagesClient(ctrl)
		wrapper := NewComputeGalleryImage(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		// Adapter expects query to split into 2 parts (gallery, image); single part is invalid
		_, qErr := adapter.Get(ctx, scope, galleryName, true)
		if qErr == nil {
			t.Error("Expected error when Get with wrong number of query parts, but got nil")
		}
	})

	t.Run("Get_EmptyGalleryName", func(t *testing.T) {
		mockClient := NewMockGalleryImagesClient(ctrl)
		wrapper := NewComputeGalleryImage(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey("", galleryImageName)
		_, qErr := adapter.Get(ctx, scope, query, true)
		if qErr == nil {
			t.Error("Expected error when gallery name is empty, but got nil")
		}
	})

	t.Run("Get_EmptyImageName", func(t *testing.T) {
		mockClient := NewMockGalleryImagesClient(ctrl)
		wrapper := NewComputeGalleryImage(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(galleryName, "")
		_, qErr := adapter.Get(ctx, scope, query, true)
		if qErr == nil {
			t.Error("Expected error when image name is empty, but got nil")
		}
	})

	t.Run("Get_ClientError", func(t *testing.T) {
		expectedErr := errors.New("image not found")
		mockClient := NewMockGalleryImagesClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, galleryName, "nonexistent", nil).Return(
			armcompute.GalleryImagesClientGetResponse{}, expectedErr)

		wrapper := NewComputeGalleryImage(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(galleryName, "nonexistent")
		_, qErr := adapter.Get(ctx, scope, query, true)
		if qErr == nil {
			t.Error("Expected error when client returns error, but got nil")
		}
	})

	t.Run("Search", func(t *testing.T) {
		img1 := createAzureGalleryImage("image-1")
		img2 := createAzureGalleryImage("image-2")

		mockClient := NewMockGalleryImagesClient(ctrl)
		pages := []armcompute.GalleryImagesClientListByGalleryResponse{
			{
				GalleryImageList: armcompute.GalleryImageList{
					Value: []*armcompute.GalleryImage{img1, img2},
				},
			},
		}
		mockPager := &mockGalleryImagesPager{pages: pages}
		testClient := &testGalleryImagesClient{
			MockGalleryImagesClient: mockClient,
			pager:                   mockPager,
		}

		wrapper := NewComputeGalleryImage(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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
		mockClient := NewMockGalleryImagesClient(ctrl)
		wrapper := NewComputeGalleryImage(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support Search operation")
		}

		// Search expects exactly 1 query part; giving 2 is invalid
		searchQuery := shared.CompositeLookupKey(galleryName, galleryImageName)
		_, err := searchable.Search(ctx, scope, searchQuery, true)
		if err == nil {
			t.Error("Expected error when Search with wrong number of query parts, but got nil")
		}
	})

	t.Run("Search_EmptyGalleryName", func(t *testing.T) {
		mockClient := NewMockGalleryImagesClient(ctrl)
		wrapper := NewComputeGalleryImage(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		_, qErr := wrapper.Search(ctx, scope, "")
		if qErr == nil {
			t.Error("Expected error when gallery name is empty, but got nil")
		}
	})

	t.Run("Search_PagerError", func(t *testing.T) {
		mockClient := NewMockGalleryImagesClient(ctrl)
		errorPager := &errorGalleryImagesPager{}
		testClient := &testGalleryImagesClient{
			MockGalleryImagesClient: mockClient,
			pager:                   errorPager,
		}

		wrapper := NewComputeGalleryImage(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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
		mockClient := NewMockGalleryImagesClient(ctrl)
		wrapper := NewComputeGalleryImage(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		links := wrapper.PotentialLinks()
		expected := map[shared.ItemType]bool{
			azureshared.ComputeGallery: true,
			stdlib.NetworkDNS:          true,
			stdlib.NetworkHTTP:         true,
			stdlib.NetworkIP:           true,
		}
		for itemType, want := range expected {
			if got := links[itemType]; got != want {
				t.Errorf("PotentialLinks()[%v] = %v, want %v", itemType, got, want)
			}
		}
	})

	t.Run("ImplementsSearchableAdapter", func(t *testing.T) {
		mockClient := NewMockGalleryImagesClient(ctrl)
		wrapper := NewComputeGalleryImage(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Error("Adapter should implement SearchableAdapter interface")
		}
	})
}
