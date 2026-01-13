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
	"github.com/overmindtech/cli/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	"github.com/overmindtech/cli/sources/azure/manual"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/azure/shared/mocks"
	"github.com/overmindtech/cli/sources/shared"
)

// mockQueuesPager is a simple mock implementation of QueuesPager
type mockQueuesPager struct {
	pages []armstorage.QueueClientListResponse
	index int
}

func (m *mockQueuesPager) More() bool {
	return m.index < len(m.pages)
}

func (m *mockQueuesPager) NextPage(ctx context.Context) (armstorage.QueueClientListResponse, error) {
	if m.index >= len(m.pages) {
		return armstorage.QueueClientListResponse{}, errors.New("no more pages")
	}
	page := m.pages[m.index]
	m.index++
	return page, nil
}

// errorQueuesPager is a mock pager that always returns an error
type errorQueuesPager struct{}

func (e *errorQueuesPager) More() bool {
	return true // Always return true so NextPage will be called
}

func (e *errorQueuesPager) NextPage(ctx context.Context) (armstorage.QueueClientListResponse, error) {
	return armstorage.QueueClientListResponse{}, errors.New("pager error")
}

// testQueuesClient wraps the mock to implement the correct interface
type testQueuesClient struct {
	*mocks.MockQueuesClient
	pager clients.QueuesPager
}

func (t *testQueuesClient) List(ctx context.Context, resourceGroupName, accountName string) clients.QueuesPager {
	return t.pager
}

func TestStorageQueues(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"
	storageAccountName := "teststorageaccount"
	queueName := "test-queue"

	t.Run("Get", func(t *testing.T) {
		queue := createAzureQueue(queueName)

		mockClient := mocks.NewMockQueuesClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, storageAccountName, queueName).Return(
			armstorage.QueueClientGetResponse{
				Queue: *queue,
			}, nil)

		testClient := &testQueuesClient{MockQueuesClient: mockClient}
		wrapper := manual.NewStorageQueues(testClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		// Get requires storageAccountName and queueName as query parts
		query := storageAccountName + shared.QuerySeparator + queueName
		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.StorageQueue.String() {
			t.Errorf("Expected type %s, got %s", azureshared.StorageQueue, sdpItem.GetType())
		}

		if sdpItem.GetUniqueAttribute() != "uniqueAttr" {
			t.Errorf("Expected unique attribute 'uniqueAttr', got %s", sdpItem.GetUniqueAttribute())
		}

		expectedID := shared.CompositeLookupKey(storageAccountName, queueName)
		if sdpItem.UniqueAttributeValue() != expectedID {
			t.Errorf("Expected unique attribute value %s, got %s", expectedID, sdpItem.UniqueAttributeValue())
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
		mockClient := mocks.NewMockQueuesClient(ctrl)
		testClient := &testQueuesClient{MockQueuesClient: mockClient}

		wrapper := manual.NewStorageQueues(testClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		// Test with insufficient query parts (only storage account name)
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], storageAccountName, true)
		if qErr == nil {
			t.Error("Expected error when providing insufficient query parts, but got nil")
		}
	})

	t.Run("Search", func(t *testing.T) {
		queue1 := createAzureQueue("queue-1")
		queue2 := createAzureQueue("queue-2")

		mockClient := mocks.NewMockQueuesClient(ctrl)
		mockPager := &mockQueuesPager{
			pages: []armstorage.QueueClientListResponse{
				{
					ListQueueResource: armstorage.ListQueueResource{
						Value: []*armstorage.ListQueue{
							{
								ID:   queue1.ID,
								Name: queue1.Name,
								Type: queue1.Type,
								QueueProperties: &armstorage.ListQueueProperties{
									Metadata: queue1.QueueProperties.Metadata,
								},
							},
							{
								ID:   queue2.ID,
								Name: queue2.Name,
								Type: queue2.Type,
								QueueProperties: &armstorage.ListQueueProperties{
									Metadata: queue2.QueueProperties.Metadata,
								},
							},
						},
					},
				},
			},
		}

		testClient := &testQueuesClient{
			MockQueuesClient: mockClient,
			pager:            mockPager,
		}

		wrapper := manual.NewStorageQueues(testClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

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

			if item.GetType() != azureshared.StorageQueue.String() {
				t.Errorf("Expected type %s, got %s", azureshared.StorageQueue, item.GetType())
			}
		}
	})

	t.Run("Search_InvalidQueryParts", func(t *testing.T) {
		// This test verifies that the wrapper's Search method validates query parts
		// We test it directly on the wrapper since the adapter may handle empty queries differently
		mockClient := mocks.NewMockQueuesClient(ctrl)
		testClient := &testQueuesClient{MockQueuesClient: mockClient}

		wrapper := manual.NewStorageQueues(testClient, subscriptionID, resourceGroup)

		// Test Search directly with no query parts - should return error before calling List
		_, qErr := wrapper.Search(ctx)
		if qErr == nil {
			t.Error("Expected error when providing no query parts, but got nil")
		}
	})

	t.Run("Search_QueueWithNilName", func(t *testing.T) {
		validQueue := createAzureQueue("valid-queue")
		mockClient := mocks.NewMockQueuesClient(ctrl)
		mockPager := &mockQueuesPager{
			pages: []armstorage.QueueClientListResponse{
				{
					ListQueueResource: armstorage.ListQueueResource{
						Value: []*armstorage.ListQueue{
							{
								// Queue with nil name should be skipped
								Name: nil,
							},
							{
								ID:   validQueue.ID,
								Name: validQueue.Name,
								Type: validQueue.Type,
								QueueProperties: &armstorage.ListQueueProperties{
									Metadata: validQueue.QueueProperties.Metadata,
								},
							},
						},
					},
				},
			},
		}

		testClient := &testQueuesClient{
			MockQueuesClient: mockClient,
			pager:            mockPager,
		}

		wrapper := manual.NewStorageQueues(testClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

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

		expectedID := shared.CompositeLookupKey(storageAccountName, "valid-queue")
		if sdpItems[0].UniqueAttributeValue() != expectedID {
			t.Errorf("Expected queue ID %s, got %s", expectedID, sdpItems[0].UniqueAttributeValue())
		}
	})

	t.Run("ErrorHandling_Get", func(t *testing.T) {
		expectedErr := errors.New("queue not found")

		mockClient := mocks.NewMockQueuesClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, storageAccountName, "nonexistent-queue").Return(
			armstorage.QueueClientGetResponse{}, expectedErr)

		testClient := &testQueuesClient{MockQueuesClient: mockClient}
		wrapper := manual.NewStorageQueues(testClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := storageAccountName + shared.QuerySeparator + "nonexistent-queue"
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr == nil {
			t.Error("Expected error when getting non-existent queue, but got nil")
		}
	})

	t.Run("ErrorHandling_Search", func(t *testing.T) {
		mockClient := mocks.NewMockQueuesClient(ctrl)
		// Create a pager that returns an error when NextPage is called
		errorPager := &errorQueuesPager{}

		testClient := &testQueuesClient{
			MockQueuesClient: mockClient,
			pager:            errorPager,
		}

		wrapper := manual.NewStorageQueues(testClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support Search operation")
		}

		_, err := searchable.Search(ctx, wrapper.Scopes()[0], storageAccountName, true)
		// The Search implementation should return an error when pager.NextPage returns an error
		// Errors from NextPage are converted to QueryError by the implementation
		if err == nil {
			t.Error("Expected error from pager when NextPage returns an error, but got nil")
		}
	})

	t.Run("InterfaceCompliance", func(t *testing.T) {
		mockClient := mocks.NewMockQueuesClient(ctrl)
		testClient := &testQueuesClient{MockQueuesClient: mockClient}
		wrapper := manual.NewStorageQueues(testClient, subscriptionID, resourceGroup)

		// Verify wrapper implements SearchableWrapper (it's returned as this type)
		if wrapper == nil {
			t.Error("Wrapper should not be nil")
		}

		// Verify adapter implements SearchableAdapter
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())
		_, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Error("Adapter should implement SearchableAdapter interface")
		}
	})

	t.Run("PotentialLinks", func(t *testing.T) {
		mockClient := mocks.NewMockQueuesClient(ctrl)
		testClient := &testQueuesClient{MockQueuesClient: mockClient}
		wrapper := manual.NewStorageQueues(testClient, subscriptionID, resourceGroup)

		links := wrapper.PotentialLinks()
		if len(links) == 0 {
			t.Error("Expected potential links to be defined")
		}

		if !links[azureshared.StorageAccount] {
			t.Error("Expected StorageAccount to be in potential links")
		}
	})

	t.Run("TerraformMappings", func(t *testing.T) {
		mockClient := mocks.NewMockQueuesClient(ctrl)
		testClient := &testQueuesClient{MockQueuesClient: mockClient}
		wrapper := manual.NewStorageQueues(testClient, subscriptionID, resourceGroup)

		mappings := wrapper.TerraformMappings()
		if len(mappings) == 0 {
			t.Fatal("Expected TerraformMappings to be defined")
		}

		// Verify we have the correct mapping for azurerm_storage_queue.id
		foundIDMapping := false
		for _, mapping := range mappings {
			if mapping.GetTerraformQueryMap() == "azurerm_storage_queue.id" {
				foundIDMapping = true
				if mapping.GetTerraformMethod() != sdp.QueryMethod_SEARCH {
					t.Errorf("Expected TerraformMethod to be SEARCH for id mapping, got %s", mapping.GetTerraformMethod())
				}
			}
		}

		if !foundIDMapping {
			t.Error("Expected TerraformMappings to include 'azurerm_storage_queue.id' mapping")
		}

		// Verify we only have one mapping (the id mapping)
		if len(mappings) != 1 {
			t.Errorf("Expected 1 TerraformMapping, got %d", len(mappings))
		}
	})
}

// createAzureQueue creates a mock Azure queue for testing
func createAzureQueue(queueName string) *armstorage.Queue {
	return &armstorage.Queue{
		ID:   to.Ptr("/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.Storage/storageAccounts/teststorageaccount/queueServices/default/queues/" + queueName),
		Name: to.Ptr(queueName),
		Type: to.Ptr("Microsoft.Storage/storageAccounts/queueServices/queues"),
		QueueProperties: &armstorage.QueueProperties{
			Metadata: map[string]*string{
				"env":     to.Ptr("test"),
				"project": to.Ptr("testing"),
			},
		},
	}
}
