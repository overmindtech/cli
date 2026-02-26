package manual

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
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

// mockGalleryApplicationVersionsPager is a mock pager for ListByGalleryApplication.
type mockGalleryApplicationVersionsPager struct {
	pages []armcompute.GalleryApplicationVersionsClientListByGalleryApplicationResponse
	index int
}

func (m *mockGalleryApplicationVersionsPager) More() bool {
	return m.index < len(m.pages)
}

func (m *mockGalleryApplicationVersionsPager) NextPage(ctx context.Context) (armcompute.GalleryApplicationVersionsClientListByGalleryApplicationResponse, error) {
	if m.index >= len(m.pages) {
		return armcompute.GalleryApplicationVersionsClientListByGalleryApplicationResponse{}, errors.New("no more pages")
	}
	page := m.pages[m.index]
	m.index++
	return page, nil
}

// errorGalleryApplicationVersionsPager is a mock pager that always returns an error.
type errorGalleryApplicationVersionsPager struct{}

func (e *errorGalleryApplicationVersionsPager) More() bool {
	return true
}

func (e *errorGalleryApplicationVersionsPager) NextPage(ctx context.Context) (armcompute.GalleryApplicationVersionsClientListByGalleryApplicationResponse, error) {
	return armcompute.GalleryApplicationVersionsClientListByGalleryApplicationResponse{}, errors.New("pager error")
}

// testGalleryApplicationVersionsClient wraps the mock and returns a pager from NewListByGalleryApplicationPager.
type testGalleryApplicationVersionsClient struct {
	*MockGalleryApplicationVersionsClient
	pager clients.GalleryApplicationVersionsPager
}

// NewListByGalleryApplicationPager returns the test pager so we don't need to mock this call.
func (t *testGalleryApplicationVersionsClient) NewListByGalleryApplicationPager(resourceGroupName, galleryName, galleryApplicationName string, options *armcompute.GalleryApplicationVersionsClientListByGalleryApplicationOptions) clients.GalleryApplicationVersionsPager {
	if t.pager != nil {
		return t.pager
	}
	return t.MockGalleryApplicationVersionsClient.NewListByGalleryApplicationPager(resourceGroupName, galleryName, galleryApplicationName, options)
}

func createAzureGalleryApplicationVersion(versionName string) *armcompute.GalleryApplicationVersion {
	return &armcompute.GalleryApplicationVersion{
		Name:     new(versionName),
		Location: new("eastus"),
		Tags: map[string]*string{
			"env": new("test"),
		},
		Properties: &armcompute.GalleryApplicationVersionProperties{
			PublishingProfile: &armcompute.GalleryApplicationVersionPublishingProfile{
				Source: &armcompute.UserArtifactSource{
					MediaLink: new("https://mystorageaccount.blob.core.windows.net/packages/app.zip"),
				},
			},
		},
	}
}

func createAzureGalleryApplicationVersionWithLinks(versionName, subscriptionID, resourceGroup string) *armcompute.GalleryApplicationVersion {
	v := createAzureGalleryApplicationVersion(versionName)
	v.Properties.PublishingProfile.Source.DefaultConfigurationLink = new("https://mystorageaccount.blob.core.windows.net/config/default.json")
	desID := "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Compute/diskEncryptionSets/test-des"
	v.Properties.PublishingProfile.TargetRegions = []*armcompute.TargetRegion{
		{
			Name: new("eastus"),
			Encryption: &armcompute.EncryptionImages{
				OSDiskImage: &armcompute.OSDiskImageEncryption{
					DiskEncryptionSetID: new(desID),
				},
			},
		},
	}
	return v
}

func TestComputeGalleryApplicationVersion(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"
	scope := subscriptionID + "." + resourceGroup
	galleryName := "test-gallery"
	galleryApplicationName := "test-app"
	galleryApplicationVersionName := "1.0.0"

	t.Run("Get", func(t *testing.T) {
		version := createAzureGalleryApplicationVersion(galleryApplicationVersionName)

		mockClient := NewMockGalleryApplicationVersionsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, galleryName, galleryApplicationName, galleryApplicationVersionName, nil).Return(
			armcompute.GalleryApplicationVersionsClientGetResponse{
				GalleryApplicationVersion: *version,
			}, nil)

		wrapper := NewComputeGalleryApplicationVersion(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(galleryName, galleryApplicationName, galleryApplicationVersionName)
		sdpItem, qErr := adapter.Get(ctx, scope, query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.ComputeGalleryApplicationVersion.String() {
			t.Errorf("Expected type %s, got %s", azureshared.ComputeGalleryApplicationVersion.String(), sdpItem.GetType())
		}

		if sdpItem.GetUniqueAttribute() != "uniqueAttr" {
			t.Errorf("Expected unique attribute 'uniqueAttr', got %s", sdpItem.GetUniqueAttribute())
		}

		expectedUnique := shared.CompositeLookupKey(galleryName, galleryApplicationName, galleryApplicationVersionName)
		if sdpItem.UniqueAttributeValue() != expectedUnique {
			t.Errorf("Expected unique attribute value %s, got %s", expectedUnique, sdpItem.UniqueAttributeValue())
		}

		if sdpItem.GetTags()["env"] != "test" {
			t.Errorf("Expected tag env=test, got: %v", sdpItem.GetTags()["env"])
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{ExpectedType: azureshared.ComputeGallery.String(), ExpectedMethod: sdp.QueryMethod_GET, ExpectedQuery: galleryName, ExpectedScope: scope},
				{ExpectedType: azureshared.ComputeGalleryApplication.String(), ExpectedMethod: sdp.QueryMethod_GET, ExpectedQuery: shared.CompositeLookupKey(galleryName, galleryApplicationName), ExpectedScope: scope},
				{ExpectedType: azureshared.StorageAccount.String(), ExpectedMethod: sdp.QueryMethod_GET, ExpectedQuery: "mystorageaccount", ExpectedScope: scope},
				{ExpectedType: azureshared.StorageBlobContainer.String(), ExpectedMethod: sdp.QueryMethod_GET, ExpectedQuery: shared.CompositeLookupKey("mystorageaccount", "packages"), ExpectedScope: scope},
				{ExpectedType: stdlib.NetworkHTTP.String(), ExpectedMethod: sdp.QueryMethod_SEARCH, ExpectedQuery: "https://mystorageaccount.blob.core.windows.net/packages/app.zip", ExpectedScope: "global"},
				{ExpectedType: stdlib.NetworkDNS.String(), ExpectedMethod: sdp.QueryMethod_SEARCH, ExpectedQuery: "mystorageaccount.blob.core.windows.net", ExpectedScope: "global"},
			}
			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("GetWithLinkedResources", func(t *testing.T) {
		version := createAzureGalleryApplicationVersionWithLinks(galleryApplicationVersionName, subscriptionID, resourceGroup)

		mockClient := NewMockGalleryApplicationVersionsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, galleryName, galleryApplicationName, galleryApplicationVersionName, nil).Return(
			armcompute.GalleryApplicationVersionsClientGetResponse{
				GalleryApplicationVersion: *version,
			}, nil)

		wrapper := NewComputeGalleryApplicationVersion(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(galleryName, galleryApplicationName, galleryApplicationVersionName)
		sdpItem, qErr := adapter.Get(ctx, scope, query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{ExpectedType: azureshared.ComputeGallery.String(), ExpectedMethod: sdp.QueryMethod_GET, ExpectedQuery: galleryName, ExpectedScope: scope},
				{ExpectedType: azureshared.ComputeGalleryApplication.String(), ExpectedMethod: sdp.QueryMethod_GET, ExpectedQuery: shared.CompositeLookupKey(galleryName, galleryApplicationName), ExpectedScope: scope},
				{ExpectedType: azureshared.StorageAccount.String(), ExpectedMethod: sdp.QueryMethod_GET, ExpectedQuery: "mystorageaccount", ExpectedScope: scope},
				{ExpectedType: azureshared.StorageBlobContainer.String(), ExpectedMethod: sdp.QueryMethod_GET, ExpectedQuery: shared.CompositeLookupKey("mystorageaccount", "packages"), ExpectedScope: scope},
				{ExpectedType: azureshared.StorageBlobContainer.String(), ExpectedMethod: sdp.QueryMethod_GET, ExpectedQuery: shared.CompositeLookupKey("mystorageaccount", "config"), ExpectedScope: scope},
				{ExpectedType: stdlib.NetworkHTTP.String(), ExpectedMethod: sdp.QueryMethod_SEARCH, ExpectedQuery: "https://mystorageaccount.blob.core.windows.net/packages/app.zip", ExpectedScope: "global"},
				{ExpectedType: stdlib.NetworkDNS.String(), ExpectedMethod: sdp.QueryMethod_SEARCH, ExpectedQuery: "mystorageaccount.blob.core.windows.net", ExpectedScope: "global"},
				{ExpectedType: stdlib.NetworkHTTP.String(), ExpectedMethod: sdp.QueryMethod_SEARCH, ExpectedQuery: "https://mystorageaccount.blob.core.windows.net/config/default.json", ExpectedScope: "global"},
				{ExpectedType: azureshared.ComputeDiskEncryptionSet.String(), ExpectedMethod: sdp.QueryMethod_GET, ExpectedQuery: "test-des", ExpectedScope: scope},
			}
			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("Get_InvalidQueryParts", func(t *testing.T) {
		mockClient := NewMockGalleryApplicationVersionsClient(ctrl)
		wrapper := NewComputeGalleryApplicationVersion(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		// Adapter expects query to split into 3 parts (gallery, application, version); single part is invalid
		_, qErr := adapter.Get(ctx, scope, galleryName, true)
		if qErr == nil {
			t.Error("Expected error when Get with wrong number of query parts, but got nil")
		}
	})

	t.Run("Get_EmptyGalleryName", func(t *testing.T) {
		mockClient := NewMockGalleryApplicationVersionsClient(ctrl)
		wrapper := NewComputeGalleryApplicationVersion(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey("", galleryApplicationName, galleryApplicationVersionName)
		_, qErr := adapter.Get(ctx, scope, query, true)
		if qErr == nil {
			t.Error("Expected error when gallery name is empty, but got nil")
		}
	})

	t.Run("Get_ClientError", func(t *testing.T) {
		expectedErr := errors.New("version not found")
		mockClient := NewMockGalleryApplicationVersionsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, galleryName, galleryApplicationName, "nonexistent", nil).Return(
			armcompute.GalleryApplicationVersionsClientGetResponse{}, expectedErr)

		wrapper := NewComputeGalleryApplicationVersion(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(galleryName, galleryApplicationName, "nonexistent")
		_, qErr := adapter.Get(ctx, scope, query, true)
		if qErr == nil {
			t.Error("Expected error when client returns error, but got nil")
		}
	})

	t.Run("Get_NonBlobURL_NoStorageLinks", func(t *testing.T) {
		// MediaLink that is not Azure Blob Storage must not create StorageAccount/StorageBlobContainer links.
		version := createAzureGalleryApplicationVersion(galleryApplicationVersionName)
		version.Properties.PublishingProfile.Source.MediaLink = new("https://example.com/artifacts/app.zip")

		mockClient := NewMockGalleryApplicationVersionsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, galleryName, galleryApplicationName, galleryApplicationVersionName, nil).Return(
			armcompute.GalleryApplicationVersionsClientGetResponse{
				GalleryApplicationVersion: *version,
			}, nil)

		wrapper := NewComputeGalleryApplicationVersion(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(galleryName, galleryApplicationName, galleryApplicationVersionName)
		sdpItem, qErr := adapter.Get(ctx, scope, query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		for _, q := range sdpItem.GetLinkedItemQueries() {
			query := q.GetQuery()
			if query == nil {
				continue
			}
			typ := query.GetType()
			if typ == azureshared.StorageAccount.String() || typ == azureshared.StorageBlobContainer.String() {
				t.Errorf("Non-blob URL must not create storage links; found linked query type %s with query %s", typ, query.GetQuery())
			}
		}
		// Should still have NetworkHTTP and NetworkDNS for the URL
		hasHTTP := false
		hasDNS := false
		for _, q := range sdpItem.GetLinkedItemQueries() {
			query := q.GetQuery()
			if query != nil {
				if query.GetType() == stdlib.NetworkHTTP.String() {
					hasHTTP = true
				}
				if query.GetType() == stdlib.NetworkDNS.String() {
					hasDNS = true
				}
			}
		}
		if !hasHTTP {
			t.Error("Expected NetworkHTTP linked query for the media URL")
		}
		if !hasDNS {
			t.Error("Expected NetworkDNS linked query for the media URL hostname")
		}
	})

	t.Run("Get_IPHost_EmitsIPLink", func(t *testing.T) {
		// When MediaLink or DefaultConfigurationLink has a literal IP host, emit stdlib.NetworkIP link (GET, global), not DNS.
		version := createAzureGalleryApplicationVersion(galleryApplicationVersionName)
		version.Properties.PublishingProfile.Source.MediaLink = new("https://192.168.1.10:8443/artifacts/app.zip")

		mockClient := NewMockGalleryApplicationVersionsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, galleryName, galleryApplicationName, galleryApplicationVersionName, nil).Return(
			armcompute.GalleryApplicationVersionsClientGetResponse{
				GalleryApplicationVersion: *version,
			}, nil)

		wrapper := NewComputeGalleryApplicationVersion(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(galleryName, galleryApplicationName, galleryApplicationVersionName)
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
			t.Error("Expected NetworkIP linked query when MediaLink host is an IP address")
		}
	})

	t.Run("Search", func(t *testing.T) {
		v1 := createAzureGalleryApplicationVersion("1.0.0")
		v2 := createAzureGalleryApplicationVersion("1.0.1")

		mockClient := NewMockGalleryApplicationVersionsClient(ctrl)
		pages := []armcompute.GalleryApplicationVersionsClientListByGalleryApplicationResponse{
			{
				GalleryApplicationVersionList: armcompute.GalleryApplicationVersionList{
					Value: []*armcompute.GalleryApplicationVersion{v1, v2},
				},
			},
		}
		mockPager := &mockGalleryApplicationVersionsPager{pages: pages}
		testClient := &testGalleryApplicationVersionsClient{
			MockGalleryApplicationVersionsClient: mockClient,
			pager:                                mockPager,
		}

		wrapper := NewComputeGalleryApplicationVersion(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support Search operation")
		}

		searchQuery := shared.CompositeLookupKey(galleryName, galleryApplicationName)
		sdpItems, err := searchable.Search(ctx, scope, searchQuery, true)
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
		mockClient := NewMockGalleryApplicationVersionsClient(ctrl)
		wrapper := NewComputeGalleryApplicationVersion(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support Search operation")
		}

		_, err := searchable.Search(ctx, scope, galleryName, true)
		if err == nil {
			t.Error("Expected error when Search with wrong number of query parts, but got nil")
		}
	})

	t.Run("Search_EmptyGalleryName", func(t *testing.T) {
		mockClient := NewMockGalleryApplicationVersionsClient(ctrl)
		wrapper := NewComputeGalleryApplicationVersion(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		_, qErr := wrapper.Search(ctx, scope, "", galleryApplicationName)
		if qErr == nil {
			t.Error("Expected error when gallery name is empty, but got nil")
		}
	})

	t.Run("Search_PagerError", func(t *testing.T) {
		mockClient := NewMockGalleryApplicationVersionsClient(ctrl)
		errorPager := &errorGalleryApplicationVersionsPager{}
		testClient := &testGalleryApplicationVersionsClient{
			MockGalleryApplicationVersionsClient: mockClient,
			pager:                                errorPager,
		}

		wrapper := NewComputeGalleryApplicationVersion(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support Search operation")
		}

		searchQuery := shared.CompositeLookupKey(galleryName, galleryApplicationName)
		_, err := searchable.Search(ctx, scope, searchQuery, true)
		if err == nil {
			t.Error("Expected error when pager returns error, but got nil")
		}
	})

	t.Run("PotentialLinks", func(t *testing.T) {
		mockClient := NewMockGalleryApplicationVersionsClient(ctrl)
		wrapper := NewComputeGalleryApplicationVersion(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		links := wrapper.PotentialLinks()
		expected := map[shared.ItemType]bool{
			azureshared.ComputeGallery:            true,
			azureshared.ComputeGalleryApplication: true,
			azureshared.ComputeDiskEncryptionSet:  true,
			azureshared.StorageAccount:            true,
			azureshared.StorageBlobContainer:      true,
			stdlib.NetworkDNS:                     true,
			stdlib.NetworkHTTP:                    true,
			stdlib.NetworkIP:                      true,
		}
		for itemType, want := range expected {
			if got := links[itemType]; got != want {
				t.Errorf("PotentialLinks()[%v] = %v, want %v", itemType, got, want)
			}
		}
	})

	t.Run("ImplementsSearchableAdapter", func(t *testing.T) {
		mockClient := NewMockGalleryApplicationVersionsClient(ctrl)
		wrapper := NewComputeGalleryApplicationVersion(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Error("Adapter should implement SearchableAdapter interface")
		}
	})
}
