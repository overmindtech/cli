package manual_test

import (
	"context"
	"errors"
	"testing"

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
	"github.com/overmindtech/cli/sources/stdlib"
)

func TestComputeSharedGalleryImage(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	location := "eastus"
	galleryUniqueName := "test-gallery-unique-name"
	imageName := "test-image"

	t.Run("Get", func(t *testing.T) {
		image := createSharedGalleryImage(imageName)

		mockClient := mocks.NewMockSharedGalleryImagesClient(ctrl)
		mockClient.EXPECT().Get(ctx, location, galleryUniqueName, imageName, nil).Return(
			armcompute.SharedGalleryImagesClientGetResponse{
				SharedGalleryImage: *image,
			}, nil)

		wrapper := manual.NewComputeSharedGalleryImage(mockClient, subscriptionID)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(location, galleryUniqueName, imageName)
		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.ComputeSharedGalleryImage.String() {
			t.Errorf("Expected type %s, got %s", azureshared.ComputeSharedGalleryImage.String(), sdpItem.GetType())
		}

		if sdpItem.GetUniqueAttribute() != "uniqueAttr" {
			t.Errorf("Expected unique attribute 'uniqueAttr', got %s", sdpItem.GetUniqueAttribute())
		}

		expectedUnique := shared.CompositeLookupKey(location, galleryUniqueName, imageName)
		if sdpItem.UniqueAttributeValue() != expectedUnique {
			t.Errorf("Expected unique attribute value %s, got %s", expectedUnique, sdpItem.UniqueAttributeValue())
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{ExpectedType: azureshared.ComputeSharedGallery.String(), ExpectedMethod: sdp.QueryMethod_GET, ExpectedQuery: shared.CompositeLookupKey(location, galleryUniqueName), ExpectedScope: subscriptionID},
			}
			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("GetWithURIs", func(t *testing.T) {
		image := createSharedGalleryImageWithURIs(imageName)

		mockClient := mocks.NewMockSharedGalleryImagesClient(ctrl)
		mockClient.EXPECT().Get(ctx, location, galleryUniqueName, imageName, nil).Return(
			armcompute.SharedGalleryImagesClientGetResponse{
				SharedGalleryImage: *image,
			}, nil)

		wrapper := manual.NewComputeSharedGalleryImage(mockClient, subscriptionID)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(location, galleryUniqueName, imageName)
		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{ExpectedType: azureshared.ComputeSharedGallery.String(), ExpectedMethod: sdp.QueryMethod_GET, ExpectedQuery: shared.CompositeLookupKey(location, galleryUniqueName), ExpectedScope: subscriptionID},
				{ExpectedType: stdlib.NetworkHTTP.String(), ExpectedMethod: sdp.QueryMethod_SEARCH, ExpectedQuery: "https://eula.example.com/terms", ExpectedScope: "global"},
				{ExpectedType: stdlib.NetworkDNS.String(), ExpectedMethod: sdp.QueryMethod_SEARCH, ExpectedQuery: "eula.example.com", ExpectedScope: "global"},
				{ExpectedType: stdlib.NetworkHTTP.String(), ExpectedMethod: sdp.QueryMethod_SEARCH, ExpectedQuery: "https://example.com/privacy", ExpectedScope: "global"},
				{ExpectedType: stdlib.NetworkDNS.String(), ExpectedMethod: sdp.QueryMethod_SEARCH, ExpectedQuery: "example.com", ExpectedScope: "global"},
			}
			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("Get_PlainTextEula_NoLinks", func(t *testing.T) {
		image := createSharedGalleryImage(imageName)
		image.Properties.Eula = new("This software is provided as-is. No warranty.")

		mockClient := mocks.NewMockSharedGalleryImagesClient(ctrl)
		mockClient.EXPECT().Get(ctx, location, galleryUniqueName, imageName, nil).Return(
			armcompute.SharedGalleryImagesClientGetResponse{
				SharedGalleryImage: *image,
			}, nil)

		wrapper := manual.NewComputeSharedGalleryImage(mockClient, subscriptionID)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(location, galleryUniqueName, imageName)
		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

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
		image := createSharedGalleryImage(imageName)
		image.Properties.Eula = new("https://example.com/eula")
		image.Properties.PrivacyStatementURI = new("https://example.com/privacy")

		mockClient := mocks.NewMockSharedGalleryImagesClient(ctrl)
		mockClient.EXPECT().Get(ctx, location, galleryUniqueName, imageName, nil).Return(
			armcompute.SharedGalleryImagesClientGetResponse{
				SharedGalleryImage: *image,
			}, nil)

		wrapper := manual.NewComputeSharedGalleryImage(mockClient, subscriptionID)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(location, galleryUniqueName, imageName)
		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		httpCount := 0
		dnsCount := 0
		for _, q := range sdpItem.GetLinkedItemQueries() {
			lq := q.GetQuery()
			if lq != nil {
				if lq.GetType() == stdlib.NetworkHTTP.String() {
					httpCount++
				}
				if lq.GetType() == stdlib.NetworkDNS.String() {
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
		image := createSharedGalleryImage(imageName)
		image.Properties.PrivacyStatementURI = new("https://192.168.1.10:8443/privacy")

		mockClient := mocks.NewMockSharedGalleryImagesClient(ctrl)
		mockClient.EXPECT().Get(ctx, location, galleryUniqueName, imageName, nil).Return(
			armcompute.SharedGalleryImagesClientGetResponse{
				SharedGalleryImage: *image,
			}, nil)

		wrapper := manual.NewComputeSharedGalleryImage(mockClient, subscriptionID)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(location, galleryUniqueName, imageName)
		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		hasIP := false
		for _, q := range sdpItem.GetLinkedItemQueries() {
			lq := q.GetQuery()
			if lq != nil && lq.GetType() == stdlib.NetworkIP.String() {
				hasIP = true
				if lq.GetMethod() != sdp.QueryMethod_GET {
					t.Errorf("Expected NetworkIP link to use GET, got %v", lq.GetMethod())
				}
				if lq.GetScope() != "global" {
					t.Errorf("Expected NetworkIP link scope global, got %s", lq.GetScope())
				}
				if lq.GetQuery() != "192.168.1.10" {
					t.Errorf("Expected NetworkIP link query 192.168.1.10, got %s", lq.GetQuery())
				}
				break
			}
		}
		if !hasIP {
			t.Error("Expected NetworkIP linked query when PrivacyStatementURI host is an IP address")
		}
	})

	t.Run("Get_InvalidQueryParts", func(t *testing.T) {
		mockClient := mocks.NewMockSharedGalleryImagesClient(ctrl)
		wrapper := manual.NewComputeSharedGalleryImage(mockClient, subscriptionID)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], location, true)
		if qErr == nil {
			t.Error("Expected error when Get with wrong number of query parts, but got nil")
		}
	})

	t.Run("Get_EmptyLocation", func(t *testing.T) {
		mockClient := mocks.NewMockSharedGalleryImagesClient(ctrl)
		wrapper := manual.NewComputeSharedGalleryImage(mockClient, subscriptionID)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey("", galleryUniqueName, imageName)
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr == nil {
			t.Error("Expected error when location is empty, but got nil")
		}
	})

	t.Run("Get_EmptyGalleryUniqueName", func(t *testing.T) {
		mockClient := mocks.NewMockSharedGalleryImagesClient(ctrl)
		wrapper := manual.NewComputeSharedGalleryImage(mockClient, subscriptionID)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(location, "", imageName)
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr == nil {
			t.Error("Expected error when gallery unique name is empty, but got nil")
		}
	})

	t.Run("Get_EmptyImageName", func(t *testing.T) {
		mockClient := mocks.NewMockSharedGalleryImagesClient(ctrl)
		wrapper := manual.NewComputeSharedGalleryImage(mockClient, subscriptionID)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(location, galleryUniqueName, "")
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr == nil {
			t.Error("Expected error when image name is empty, but got nil")
		}
	})

	t.Run("Get_ClientError", func(t *testing.T) {
		expectedErr := errors.New("image not found")
		mockClient := mocks.NewMockSharedGalleryImagesClient(ctrl)
		mockClient.EXPECT().Get(ctx, location, galleryUniqueName, "nonexistent", nil).Return(
			armcompute.SharedGalleryImagesClientGetResponse{}, expectedErr)

		wrapper := manual.NewComputeSharedGalleryImage(mockClient, subscriptionID)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(location, galleryUniqueName, "nonexistent")
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr == nil {
			t.Error("Expected error when client returns error, but got nil")
		}
	})

	t.Run("Search", func(t *testing.T) {
		img1 := createSharedGalleryImage("image-1")
		img2 := createSharedGalleryImage("image-2")

		mockClient := mocks.NewMockSharedGalleryImagesClient(ctrl)
		mockPager := newMockSharedGalleryImagesPager([]*armcompute.SharedGalleryImage{img1, img2})
		mockClient.EXPECT().NewListPager(location, galleryUniqueName, nil).Return(mockPager)

		wrapper := manual.NewComputeSharedGalleryImage(mockClient, subscriptionID)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support Search operation")
		}

		searchQuery := shared.CompositeLookupKey(location, galleryUniqueName)
		sdpItems, err := searchable.Search(ctx, wrapper.Scopes()[0], searchQuery, true)
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
		mockClient := mocks.NewMockSharedGalleryImagesClient(ctrl)
		wrapper := manual.NewComputeSharedGalleryImage(mockClient, subscriptionID)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support Search operation")
		}

		searchQuery := shared.CompositeLookupKey(location, galleryUniqueName, imageName)
		_, err := searchable.Search(ctx, wrapper.Scopes()[0], searchQuery, true)
		if err == nil {
			t.Error("Expected error when Search with wrong number of query parts, but got nil")
		}
	})

	t.Run("Search_EmptyLocation", func(t *testing.T) {
		mockClient := mocks.NewMockSharedGalleryImagesClient(ctrl)
		wrapper := manual.NewComputeSharedGalleryImage(mockClient, subscriptionID)

		_, qErr := wrapper.Search(ctx, wrapper.Scopes()[0], "", galleryUniqueName)
		if qErr == nil {
			t.Error("Expected error when location is empty, but got nil")
		}
	})

	t.Run("Search_EmptyGalleryUniqueName", func(t *testing.T) {
		mockClient := mocks.NewMockSharedGalleryImagesClient(ctrl)
		wrapper := manual.NewComputeSharedGalleryImage(mockClient, subscriptionID)

		_, qErr := wrapper.Search(ctx, wrapper.Scopes()[0], location, "")
		if qErr == nil {
			t.Error("Expected error when gallery unique name is empty, but got nil")
		}
	})

	t.Run("Search_PagerError", func(t *testing.T) {
		mockClient := mocks.NewMockSharedGalleryImagesClient(ctrl)
		errorPager := &errorSharedGalleryImagesPager{}
		mockClient.EXPECT().NewListPager(location, galleryUniqueName, nil).Return(errorPager)

		wrapper := manual.NewComputeSharedGalleryImage(mockClient, subscriptionID)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support Search operation")
		}

		searchQuery := shared.CompositeLookupKey(location, galleryUniqueName)
		_, err := searchable.Search(ctx, wrapper.Scopes()[0], searchQuery, true)
		if err == nil {
			t.Error("Expected error when pager returns error, but got nil")
		}
	})

	t.Run("PotentialLinks", func(t *testing.T) {
		mockClient := mocks.NewMockSharedGalleryImagesClient(ctrl)
		wrapper := manual.NewComputeSharedGalleryImage(mockClient, subscriptionID)

		links := wrapper.PotentialLinks()
		expected := map[shared.ItemType]bool{
			azureshared.ComputeSharedGallery: true,
			stdlib.NetworkDNS:                true,
			stdlib.NetworkHTTP:               true,
			stdlib.NetworkIP:                 true,
		}
		for itemType, want := range expected {
			if got := links[itemType]; got != want {
				t.Errorf("PotentialLinks()[%v] = %v, want %v", itemType, got, want)
			}
		}
	})

	t.Run("ImplementsSearchableAdapter", func(t *testing.T) {
		mockClient := mocks.NewMockSharedGalleryImagesClient(ctrl)
		wrapper := manual.NewComputeSharedGalleryImage(mockClient, subscriptionID)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Error("Adapter should implement SearchableAdapter interface")
		}
	})
}

func createSharedGalleryImage(name string) *armcompute.SharedGalleryImage {
	return &armcompute.SharedGalleryImage{
		Name:     new(name),
		Location: new("eastus"),
		Identifier: &armcompute.SharedGalleryIdentifier{
			UniqueID: new("/SharedGalleries/test-gallery-unique-name"),
		},
		Properties: &armcompute.SharedGalleryImageProperties{
			Identifier: &armcompute.GalleryImageIdentifier{
				Publisher: new("test-publisher"),
				Offer:     new("test-offer"),
				SKU:       new("test-sku"),
			},
			OSType:  new(armcompute.OperatingSystemTypesLinux),
			OSState: new(armcompute.OperatingSystemStateTypesGeneralized),
		},
	}
}

func createSharedGalleryImageWithURIs(name string) *armcompute.SharedGalleryImage {
	img := createSharedGalleryImage(name)
	img.Properties.Eula = new("https://eula.example.com/terms")
	img.Properties.PrivacyStatementURI = new("https://example.com/privacy")
	return img
}

type mockSharedGalleryImagesPager struct {
	pages []armcompute.SharedGalleryImagesClientListResponse
	index int
}

func newMockSharedGalleryImagesPager(items []*armcompute.SharedGalleryImage) clients.SharedGalleryImagesPager {
	return &mockSharedGalleryImagesPager{
		pages: []armcompute.SharedGalleryImagesClientListResponse{
			{
				SharedGalleryImageList: armcompute.SharedGalleryImageList{
					Value: items,
				},
			},
		},
		index: 0,
	}
}

func (m *mockSharedGalleryImagesPager) More() bool {
	return m.index < len(m.pages)
}

func (m *mockSharedGalleryImagesPager) NextPage(ctx context.Context) (armcompute.SharedGalleryImagesClientListResponse, error) {
	if m.index >= len(m.pages) {
		return armcompute.SharedGalleryImagesClientListResponse{}, errors.New("no more pages")
	}
	page := m.pages[m.index]
	m.index++
	return page, nil
}

type errorSharedGalleryImagesPager struct{}

func (e *errorSharedGalleryImagesPager) More() bool {
	return true
}

func (e *errorSharedGalleryImagesPager) NextPage(ctx context.Context) (armcompute.SharedGalleryImagesClientListResponse, error) {
	return armcompute.SharedGalleryImagesClientListResponse{}, errors.New("pager error")
}
