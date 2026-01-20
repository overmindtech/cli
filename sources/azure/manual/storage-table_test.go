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

// mockTablesPager is a simple mock implementation of TablesPager
type mockTablesPager struct {
	pages []armstorage.TableClientListResponse
	index int
}

func (m *mockTablesPager) More() bool {
	return m.index < len(m.pages)
}

func (m *mockTablesPager) NextPage(ctx context.Context) (armstorage.TableClientListResponse, error) {
	if m.index >= len(m.pages) {
		return armstorage.TableClientListResponse{}, errors.New("no more pages")
	}
	page := m.pages[m.index]
	m.index++
	return page, nil
}

// errorTablesPager is a mock pager that always returns an error
type errorTablesPager struct{}

func (e *errorTablesPager) More() bool {
	return true // Always return true so NextPage will be called
}

func (e *errorTablesPager) NextPage(ctx context.Context) (armstorage.TableClientListResponse, error) {
	return armstorage.TableClientListResponse{}, errors.New("pager error")
}

// testTablesClient wraps the mock to implement the correct interface
type testTablesClient struct {
	*mocks.MockTablesClient
	pager clients.TablesPager
}

func (t *testTablesClient) List(ctx context.Context, resourceGroupName, accountName string) clients.TablesPager {
	return t.pager
}

func TestStorageTables(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"
	storageAccountName := "teststorageaccount"
	tableName := "test-table"

	t.Run("Get", func(t *testing.T) {
		table := createAzureTable(tableName)

		mockClient := mocks.NewMockTablesClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, storageAccountName, tableName).Return(
			armstorage.TableClientGetResponse{
				Table: *table,
			}, nil)

		testClient := &testTablesClient{MockTablesClient: mockClient}
		wrapper := manual.NewStorageTable(testClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		// Get requires storageAccountName and tableName as query parts
		query := storageAccountName + shared.QuerySeparator + tableName
		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.StorageTable.String() {
			t.Errorf("Expected type %s, got %s", azureshared.StorageTable, sdpItem.GetType())
		}

		if sdpItem.GetUniqueAttribute() != "uniqueAttr" {
			t.Errorf("Expected unique attribute 'uniqueAttr', got %s", sdpItem.GetUniqueAttribute())
		}

		expectedID := shared.CompositeLookupKey(storageAccountName, tableName)
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
		mockClient := mocks.NewMockTablesClient(ctrl)
		testClient := &testTablesClient{MockTablesClient: mockClient}

		wrapper := manual.NewStorageTable(testClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		// Test with insufficient query parts (only storage account name)
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], storageAccountName, true)
		if qErr == nil {
			t.Error("Expected error when providing insufficient query parts, but got nil")
		}
	})

	t.Run("Search", func(t *testing.T) {
		table1 := createAzureTable("table-1")
		table2 := createAzureTable("table-2")

		mockClient := mocks.NewMockTablesClient(ctrl)
		mockPager := &mockTablesPager{
			pages: []armstorage.TableClientListResponse{
				{
					ListTableResource: armstorage.ListTableResource{
						Value: []*armstorage.Table{
							{
								ID:              table1.ID,
								Name:            table1.Name,
								Type:            table1.Type,
								TableProperties: table1.TableProperties,
							},
							{
								ID:              table2.ID,
								Name:            table2.Name,
								Type:            table2.Type,
								TableProperties: table2.TableProperties,
							},
						},
					},
				},
			},
		}

		testClient := &testTablesClient{
			MockTablesClient: mockClient,
			pager:            mockPager,
		}

		wrapper := manual.NewStorageTable(testClient, subscriptionID, resourceGroup)
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

			if item.GetType() != azureshared.StorageTable.String() {
				t.Errorf("Expected type %s, got %s", azureshared.StorageTable, item.GetType())
			}
		}
	})

	t.Run("Search_InvalidQueryParts", func(t *testing.T) {
		// This test verifies that the wrapper's Search method validates query parts
		// We test it directly on the wrapper since the adapter may handle empty queries differently
		mockClient := mocks.NewMockTablesClient(ctrl)
		testClient := &testTablesClient{MockTablesClient: mockClient}

		wrapper := manual.NewStorageTable(testClient, subscriptionID, resourceGroup)

		// Test Search directly with no query parts - should return error before calling List
		_, qErr := wrapper.Search(ctx, wrapper.Scopes()[0])
		if qErr == nil {
			t.Error("Expected error when providing no query parts, but got nil")
		}
	})

	t.Run("Search_TableWithNilName", func(t *testing.T) {
		validTable := createAzureTable("valid-table")
		mockClient := mocks.NewMockTablesClient(ctrl)
		mockPager := &mockTablesPager{
			pages: []armstorage.TableClientListResponse{
				{
					ListTableResource: armstorage.ListTableResource{
						Value: []*armstorage.Table{
							{
								// Table with nil name should be skipped
								Name: nil,
							},
							{
								ID:              validTable.ID,
								Name:            validTable.Name,
								Type:            validTable.Type,
								TableProperties: validTable.TableProperties,
							},
						},
					},
				},
			},
		}

		testClient := &testTablesClient{
			MockTablesClient: mockClient,
			pager:            mockPager,
		}

		wrapper := manual.NewStorageTable(testClient, subscriptionID, resourceGroup)
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

		expectedID := shared.CompositeLookupKey(storageAccountName, "valid-table")
		if sdpItems[0].UniqueAttributeValue() != expectedID {
			t.Errorf("Expected table ID %s, got %s", expectedID, sdpItems[0].UniqueAttributeValue())
		}
	})

	t.Run("ErrorHandling_Get", func(t *testing.T) {
		expectedErr := errors.New("table not found")

		mockClient := mocks.NewMockTablesClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, storageAccountName, "nonexistent-table").Return(
			armstorage.TableClientGetResponse{}, expectedErr)

		testClient := &testTablesClient{MockTablesClient: mockClient}
		wrapper := manual.NewStorageTable(testClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := storageAccountName + shared.QuerySeparator + "nonexistent-table"
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr == nil {
			t.Error("Expected error when getting non-existent table, but got nil")
		}
	})

	t.Run("ErrorHandling_Search", func(t *testing.T) {
		mockClient := mocks.NewMockTablesClient(ctrl)
		// Create a pager that returns an error when NextPage is called
		errorPager := &errorTablesPager{}

		testClient := &testTablesClient{
			MockTablesClient: mockClient,
			pager:            errorPager,
		}

		wrapper := manual.NewStorageTable(testClient, subscriptionID, resourceGroup)
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
		mockClient := mocks.NewMockTablesClient(ctrl)
		testClient := &testTablesClient{MockTablesClient: mockClient}
		wrapper := manual.NewStorageTable(testClient, subscriptionID, resourceGroup)

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
		mockClient := mocks.NewMockTablesClient(ctrl)
		testClient := &testTablesClient{MockTablesClient: mockClient}
		wrapper := manual.NewStorageTable(testClient, subscriptionID, resourceGroup)

		links := wrapper.PotentialLinks()
		if len(links) == 0 {
			t.Error("Expected potential links to be defined")
		}

		if !links[azureshared.StorageAccount] {
			t.Error("Expected StorageAccount to be in potential links")
		}
	})

	t.Run("TerraformMappings", func(t *testing.T) {
		mockClient := mocks.NewMockTablesClient(ctrl)
		testClient := &testTablesClient{MockTablesClient: mockClient}
		wrapper := manual.NewStorageTable(testClient, subscriptionID, resourceGroup)

		mappings := wrapper.TerraformMappings()
		if len(mappings) == 0 {
			t.Fatal("Expected TerraformMappings to be defined")
		}

		// Verify we have the correct mapping for azurerm_storage_table.id
		foundIDMapping := false
		for _, mapping := range mappings {
			if mapping.GetTerraformQueryMap() == "azurerm_storage_table.id" {
				foundIDMapping = true
				if mapping.GetTerraformMethod() != sdp.QueryMethod_SEARCH {
					t.Errorf("Expected TerraformMethod to be SEARCH for id mapping, got %s", mapping.GetTerraformMethod())
				}
			}
		}

		if !foundIDMapping {
			t.Error("Expected TerraformMappings to include 'azurerm_storage_table.id' mapping")
		}

		// Verify we only have one mapping (the id mapping)
		if len(mappings) != 1 {
			t.Errorf("Expected 1 TerraformMapping, got %d", len(mappings))
		}
	})

	t.Run("IAMPermissions", func(t *testing.T) {
		mockClient := mocks.NewMockTablesClient(ctrl)
		testClient := &testTablesClient{MockTablesClient: mockClient}
		wrapper := manual.NewStorageTable(testClient, subscriptionID, resourceGroup)

		permissions := wrapper.IAMPermissions()
		if len(permissions) == 0 {
			t.Error("Expected IAMPermissions to be defined")
		}

		expectedPermission := "Microsoft.Storage/storageAccounts/tableServices/tables/read"
		found := false
		for _, perm := range permissions {
			if perm == expectedPermission {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected IAMPermissions to include %s", expectedPermission)
		}
	})
}

// createAzureTable creates a mock Azure table for testing
func createAzureTable(tableName string) *armstorage.Table {
	return &armstorage.Table{
		ID:              to.Ptr("/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.Storage/storageAccounts/teststorageaccount/tableServices/default/tables/" + tableName),
		Name:            to.Ptr(tableName),
		Type:            to.Ptr("Microsoft.Storage/storageAccounts/tableServices/tables"),
		TableProperties: &armstorage.TableProperties{},
	}
}
