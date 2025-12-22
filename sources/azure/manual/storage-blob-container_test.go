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

// mockBlobContainersPager is a simple mock implementation of BlobContainersPager
type mockBlobContainersPager struct {
	pages []armstorage.BlobContainersClientListResponse
	index int
}

func (m *mockBlobContainersPager) More() bool {
	return m.index < len(m.pages)
}

func (m *mockBlobContainersPager) NextPage(ctx context.Context) (armstorage.BlobContainersClientListResponse, error) {
	if m.index >= len(m.pages) {
		return armstorage.BlobContainersClientListResponse{}, errors.New("no more pages")
	}
	page := m.pages[m.index]
	m.index++
	return page, nil
}

// errorBlobContainersPager is a mock pager that always returns an error
type errorBlobContainersPager struct{}

func (e *errorBlobContainersPager) More() bool {
	return true // Always return true so NextPage will be called
}

func (e *errorBlobContainersPager) NextPage(ctx context.Context) (armstorage.BlobContainersClientListResponse, error) {
	return armstorage.BlobContainersClientListResponse{}, errors.New("pager error")
}

// testBlobContainersClient wraps the mock to implement the correct interface
type testBlobContainersClient struct {
	*mocks.MockBlobContainersClient
	pager clients.BlobContainersPager
}

func (t *testBlobContainersClient) List(ctx context.Context, resourceGroupName, accountName string) clients.BlobContainersPager {
	return t.pager
}

func TestStorageBlobContainer(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"
	storageAccountName := "teststorageaccount"
	containerName := "test-container"

	t.Run("Get", func(t *testing.T) {
		container := createAzureBlobContainer(containerName)

		mockClient := mocks.NewMockBlobContainersClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, storageAccountName, containerName).Return(
			armstorage.BlobContainersClientGetResponse{
				BlobContainer: *container,
			}, nil)

		testClient := &testBlobContainersClient{MockBlobContainersClient: mockClient}
		wrapper := manual.NewStorageBlobContainer(testClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper)

		// Get requires storageAccountName and containerName as query parts
		query := storageAccountName + shared.QuerySeparator + containerName
		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.StorageBlobContainer.String() {
			t.Errorf("Expected type %s, got %s", azureshared.StorageBlobContainer, sdpItem.GetType())
		}

		if sdpItem.GetUniqueAttribute() != "name" {
			t.Errorf("Expected unique attribute 'name', got %s", sdpItem.GetUniqueAttribute())
		}

		if sdpItem.UniqueAttributeValue() != containerName {
			t.Errorf("Expected unique attribute value %s, got %s", containerName, sdpItem.UniqueAttributeValue())
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
		mockClient := mocks.NewMockBlobContainersClient(ctrl)
		testClient := &testBlobContainersClient{MockBlobContainersClient: mockClient}

		wrapper := manual.NewStorageBlobContainer(testClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper)

		// Test with insufficient query parts (only storage account name)
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], storageAccountName, true)
		if qErr == nil {
			t.Error("Expected error when providing insufficient query parts, but got nil")
		}
	})

	t.Run("Search", func(t *testing.T) {
		container1 := createAzureBlobContainer("container-1")
		container2 := createAzureBlobContainer("container-2")

		mockClient := mocks.NewMockBlobContainersClient(ctrl)
		mockPager := &mockBlobContainersPager{
			pages: []armstorage.BlobContainersClientListResponse{
				{
					ListContainerItems: armstorage.ListContainerItems{
						Value: []*armstorage.ListContainerItem{
							{
								ID:   container1.ID,
								Name: container1.Name,
								Type: container1.Type,
								Properties: &armstorage.ContainerProperties{
									PublicAccess: container1.ContainerProperties.PublicAccess,
								},
								Etag: container1.Etag,
							},
							{
								ID:   container2.ID,
								Name: container2.Name,
								Type: container2.Type,
								Properties: &armstorage.ContainerProperties{
									PublicAccess: container2.ContainerProperties.PublicAccess,
								},
								Etag: container2.Etag,
							},
						},
					},
				},
			},
		}

		// The mock returns *runtime.Pager, but we need to work with BlobContainersPager
		// We'll use a type assertion approach - create a wrapper that implements the interface
		testClient := &testBlobContainersClient{
			MockBlobContainersClient: mockClient,
			pager:                    mockPager,
		}

		wrapper := manual.NewStorageBlobContainer(testClient, subscriptionID, resourceGroup)
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

			if item.GetType() != azureshared.StorageBlobContainer.String() {
				t.Errorf("Expected type %s, got %s", azureshared.StorageBlobContainer, item.GetType())
			}
		}
	})

	t.Run("Search_InvalidQueryParts", func(t *testing.T) {
		// This test verifies that the wrapper's Search method validates query parts
		// We test it directly on the wrapper since the adapter may handle empty queries differently
		mockClient := mocks.NewMockBlobContainersClient(ctrl)
		testClient := &testBlobContainersClient{MockBlobContainersClient: mockClient}

		wrapper := manual.NewStorageBlobContainer(testClient, subscriptionID, resourceGroup)

		// Test Search directly with no query parts - should return error before calling List
		_, qErr := wrapper.Search(ctx)
		if qErr == nil {
			t.Error("Expected error when providing no query parts, but got nil")
		}
	})

	t.Run("Search_ContainerWithNilName", func(t *testing.T) {
		mockClient := mocks.NewMockBlobContainersClient(ctrl)
		mockPager := &mockBlobContainersPager{
			pages: []armstorage.BlobContainersClientListResponse{
				{
					ListContainerItems: armstorage.ListContainerItems{
						Value: []*armstorage.ListContainerItem{
							{
								// Container with nil name should be skipped
								Name: nil,
							},
							{
								ID:   to.Ptr("/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.Storage/storageAccounts/teststorageaccount/blobServices/default/containers/valid-container"),
								Name: to.Ptr("valid-container"),
								Type: to.Ptr("Microsoft.Storage/storageAccounts/blobServices/containers"),
							},
						},
					},
				},
			},
		}

		testClient := &testBlobContainersClient{
			MockBlobContainersClient: mockClient,
			pager:                    mockPager,
		}

		wrapper := manual.NewStorageBlobContainer(testClient, subscriptionID, resourceGroup)
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

		if sdpItems[0].UniqueAttributeValue() != "valid-container" {
			t.Errorf("Expected container name 'valid-container', got %s", sdpItems[0].UniqueAttributeValue())
		}
	})

	t.Run("ErrorHandling_Get", func(t *testing.T) {
		expectedErr := errors.New("container not found")

		mockClient := mocks.NewMockBlobContainersClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, storageAccountName, "nonexistent-container").Return(
			armstorage.BlobContainersClientGetResponse{}, expectedErr)

		testClient := &testBlobContainersClient{MockBlobContainersClient: mockClient}
		wrapper := manual.NewStorageBlobContainer(testClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper)

		query := storageAccountName + shared.QuerySeparator + "nonexistent-container"
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr == nil {
			t.Error("Expected error when getting non-existent container, but got nil")
		}
	})

	t.Run("ErrorHandling_Search", func(t *testing.T) {
		mockClient := mocks.NewMockBlobContainersClient(ctrl)
		// Create a pager that returns an error when NextPage is called
		errorPager := &errorBlobContainersPager{}

		testClient := &testBlobContainersClient{
			MockBlobContainersClient: mockClient,
			pager:                    errorPager,
		}

		wrapper := manual.NewStorageBlobContainer(testClient, subscriptionID, resourceGroup)
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

// createAzureBlobContainer creates a mock Azure blob container for testing
func createAzureBlobContainer(containerName string) *armstorage.BlobContainer {
	return &armstorage.BlobContainer{
		ID:   to.Ptr("/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.Storage/storageAccounts/teststorageaccount/blobServices/default/containers/" + containerName),
		Name: to.Ptr(containerName),
		Type: to.Ptr("Microsoft.Storage/storageAccounts/blobServices/containers"),
		ContainerProperties: &armstorage.ContainerProperties{
			PublicAccess: to.Ptr(armstorage.PublicAccessNone),
		},
		Etag: to.Ptr("\"0x8D1234567890ABC\""),
	}
}
