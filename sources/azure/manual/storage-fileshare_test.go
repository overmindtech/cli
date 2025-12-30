package manual_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage/v2"
	"go.uber.org/mock/gomock"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	"github.com/overmindtech/cli/sources/azure/manual"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/azure/shared/mocks"
	"github.com/overmindtech/cli/sources/shared"
)

// mockFileSharesPager is a simple mock implementation of FileSharesPager
type mockFileSharesPager struct {
	pages []armstorage.FileSharesClientListResponse
	index int
}

func (m *mockFileSharesPager) More() bool {
	return m.index < len(m.pages)
}

func (m *mockFileSharesPager) NextPage(ctx context.Context) (armstorage.FileSharesClientListResponse, error) {
	if m.index >= len(m.pages) {
		return armstorage.FileSharesClientListResponse{}, errors.New("no more pages")
	}
	page := m.pages[m.index]
	m.index++
	return page, nil
}

// errorFileSharesPager is a mock pager that always returns an error
type errorFileSharesPager struct{}

func (e *errorFileSharesPager) More() bool {
	return true // Always return true so NextPage will be called
}

func (e *errorFileSharesPager) NextPage(ctx context.Context) (armstorage.FileSharesClientListResponse, error) {
	return armstorage.FileSharesClientListResponse{}, errors.New("pager error")
}

// testFileSharesClient wraps the mock to implement the correct interface
type testFileSharesClient struct {
	*mocks.MockFileSharesClient
	pager clients.FileSharesPager
}

func (t *testFileSharesClient) List(ctx context.Context, resourceGroupName, accountName string) clients.FileSharesPager {
	return t.pager
}

func TestStorageFileShare(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"
	storageAccountName := "teststorageaccount"
	shareName := "test-share"

	t.Run("Get", func(t *testing.T) {
		fileShare := createAzureFileShare(shareName)

		mockClient := mocks.NewMockFileSharesClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, storageAccountName, shareName).Return(
			armstorage.FileSharesClientGetResponse{
				FileShare: *fileShare,
			}, nil)

		testClient := &testFileSharesClient{MockFileSharesClient: mockClient}
		wrapper := manual.NewStorageFileShare(testClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper)

		// Get requires storageAccountName and shareName as query parts
		query := shared.CompositeLookupKey(storageAccountName, shareName)
		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.StorageFileShare.String() {
			t.Errorf("Expected type %s, got %s", azureshared.StorageFileShare, sdpItem.GetType())
		}

		if sdpItem.GetUniqueAttribute() != "uniqueAttr" {
			t.Errorf("Expected unique attribute 'uniqueAttr', got %s", sdpItem.GetUniqueAttribute())
		}

		if sdpItem.UniqueAttributeValue() != shared.CompositeLookupKey(storageAccountName, shareName) {
			t.Errorf("Expected unique attribute value %s, got %s", shared.CompositeLookupKey(storageAccountName, shareName), sdpItem.UniqueAttributeValue())
		}

		if sdpItem.GetScope() != subscriptionID+"."+resourceGroup {
			t.Errorf("Expected scope %s, got %s", subscriptionID+"."+resourceGroup, sdpItem.GetScope())
		}

		// Validate the item
		if err := sdpItem.Validate(); err != nil {
			t.Fatalf("Expected no validation error, got: %v", err)
		}

		t.Run("StaticTests", func(t *testing.T) {
			// Verify linked item queries
			linkedQueries := sdpItem.GetLinkedItemQueries()
			if len(linkedQueries) != 1 {
				t.Fatalf("Expected 1 linked query, got: %d", len(linkedQueries))
			}

			linkedQuery := linkedQueries[0]
			if linkedQuery.GetQuery().GetType() != azureshared.StorageAccount.String() {
				t.Errorf("Expected linked query type %s, got %s", azureshared.StorageAccount, linkedQuery.GetQuery().GetType())
			}
			if linkedQuery.GetQuery().GetMethod() != sdp.QueryMethod_GET {
				t.Errorf("Expected linked query method GET, got %s", linkedQuery.GetQuery().GetMethod())
			}
			if linkedQuery.GetQuery().GetQuery() != storageAccountName {
				t.Errorf("Expected linked query %s, got %s", storageAccountName, linkedQuery.GetQuery().GetQuery())
			}
			if linkedQuery.GetBlastPropagation().GetIn() != true {
				t.Error("Expected BlastPropagation.In to be true")
			}
			if linkedQuery.GetBlastPropagation().GetOut() != false {
				t.Error("Expected BlastPropagation.Out to be false")
			}
		})
	})

	t.Run("Get_InvalidQueryParts", func(t *testing.T) {
		mockClient := mocks.NewMockFileSharesClient(ctrl)
		testClient := &testFileSharesClient{MockFileSharesClient: mockClient}

		wrapper := manual.NewStorageFileShare(testClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper)

		// Test with insufficient query parts (only storage account name)
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], storageAccountName, true)
		if qErr == nil {
			t.Error("Expected error when providing insufficient query parts, but got nil")
		}
	})

	t.Run("Search", func(t *testing.T) {
		share1 := createAzureFileShare("share-1")
		share2 := createAzureFileShare("share-2")

		mockClient := mocks.NewMockFileSharesClient(ctrl)
		mockPager := &mockFileSharesPager{
			pages: []armstorage.FileSharesClientListResponse{
				{
					FileShareItems: armstorage.FileShareItems{
						Value: []*armstorage.FileShareItem{
							{
								ID:   share1.ID,
								Name: share1.Name,
								Type: share1.Type,
								Properties: &armstorage.FileShareProperties{
									AccessTier: share1.FileShareProperties.AccessTier,
									ShareQuota: share1.FileShareProperties.ShareQuota,
								},
								Etag: share1.Etag,
							},
							{
								ID:   share2.ID,
								Name: share2.Name,
								Type: share2.Type,
								Properties: &armstorage.FileShareProperties{
									AccessTier: share2.FileShareProperties.AccessTier,
									ShareQuota: share2.FileShareProperties.ShareQuota,
								},
								Etag: share2.Etag,
							},
						},
					},
				},
			},
		}

		testClient := &testFileSharesClient{
			MockFileSharesClient: mockClient,
			pager:                mockPager,
		}

		wrapper := manual.NewStorageFileShare(testClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper)

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support Search operation")
		}

		sdpItems, err := searchable.Search(ctx, wrapper.Scopes()[0], storageAccountName, true)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if len(sdpItems) != 2 {
			t.Fatalf("Expected 2 items, got: %d", len(sdpItems))
		}

		for _, item := range sdpItems {
			if err := item.Validate(); err != nil {
				t.Fatalf("Expected no validation error, got: %v", err)
			}

			if item.GetType() != azureshared.StorageFileShare.String() {
				t.Errorf("Expected type %s, got %s", azureshared.StorageFileShare, item.GetType())
			}
		}
	})

	t.Run("Search_InvalidQueryParts", func(t *testing.T) {
		// This test verifies that the wrapper's Search method validates query parts
		// We test it directly on the wrapper since the adapter may handle empty queries differently
		mockClient := mocks.NewMockFileSharesClient(ctrl)
		testClient := &testFileSharesClient{MockFileSharesClient: mockClient}

		wrapper := manual.NewStorageFileShare(testClient, subscriptionID, resourceGroup)

		// Test Search directly with no query parts - should return error before calling List
		_, qErr := wrapper.Search(ctx)
		if qErr == nil {
			t.Error("Expected error when providing no query parts, but got nil")
		}
	})

	t.Run("Search_ShareWithNilName", func(t *testing.T) {
		mockClient := mocks.NewMockFileSharesClient(ctrl)
		mockPager := &mockFileSharesPager{
			pages: []armstorage.FileSharesClientListResponse{
				{
					FileShareItems: armstorage.FileShareItems{
						Value: []*armstorage.FileShareItem{
							{
								// Share with nil name should be skipped
								Name: nil,
							},
							{
								ID:   to.Ptr("/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.Storage/storageAccounts/teststorageaccount/fileServices/default/shares/valid-share"),
								Name: to.Ptr("valid-share"),
								Type: to.Ptr("Microsoft.Storage/storageAccounts/fileServices/shares"),
							},
						},
					},
				},
			},
		}

		testClient := &testFileSharesClient{
			MockFileSharesClient: mockClient,
			pager:                mockPager,
		}

		wrapper := manual.NewStorageFileShare(testClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper)

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support Search operation")
		}

		sdpItems, err := searchable.Search(ctx, wrapper.Scopes()[0], storageAccountName, true)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		// Should only return 1 item (the one with a valid name)
		if len(sdpItems) != 1 {
			t.Fatalf("Expected 1 item, got: %d", len(sdpItems))
		}

		if sdpItems[0].UniqueAttributeValue() != shared.CompositeLookupKey(storageAccountName, "valid-share") {
			t.Errorf("Expected share name 'teststorageaccount|valid-share', got %s", sdpItems[0].UniqueAttributeValue())
		}
	})

	t.Run("ErrorHandling_Get", func(t *testing.T) {
		expectedErr := errors.New("file share not found")

		mockClient := mocks.NewMockFileSharesClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, storageAccountName, "nonexistent-share").Return(
			armstorage.FileSharesClientGetResponse{}, expectedErr)

		testClient := &testFileSharesClient{MockFileSharesClient: mockClient}
		wrapper := manual.NewStorageFileShare(testClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper)

		query := storageAccountName + shared.QuerySeparator + "nonexistent-share"
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr == nil {
			t.Error("Expected error when getting non-existent file share, but got nil")
		}
	})

	t.Run("ErrorHandling_Search", func(t *testing.T) {
		mockClient := mocks.NewMockFileSharesClient(ctrl)
		// Create a pager that returns an error when NextPage is called
		errorPager := &errorFileSharesPager{}

		testClient := &testFileSharesClient{
			MockFileSharesClient: mockClient,
			pager:                errorPager,
		}

		wrapper := manual.NewStorageFileShare(testClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper)

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support Search operation")
		}

		_, err := searchable.Search(ctx, wrapper.Scopes()[0], "test-account", true)
		// The Search implementation should return an error when pager.NextPage returns an error
		// Errors from NextPage are converted to QueryError by the implementation
		if err == nil {
			t.Error("Expected error from pager when NextPage returns an error, but got nil")
		}
	})
}

// createAzureFileShare creates a mock Azure file share for testing
func createAzureFileShare(shareName string) *armstorage.FileShare {
	return &armstorage.FileShare{
		ID:   to.Ptr("/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.Storage/storageAccounts/teststorageaccount/fileServices/default/shares/" + shareName),
		Name: to.Ptr(shareName),
		Type: to.Ptr("Microsoft.Storage/storageAccounts/fileServices/shares"),
		FileShareProperties: &armstorage.FileShareProperties{
			AccessTier: to.Ptr(armstorage.ShareAccessTierHot),
			ShareQuota: to.Ptr(int32(5120)), // 5GB
		},
		Etag: to.Ptr("\"0x8D1234567890ABC\""),
	}
}
