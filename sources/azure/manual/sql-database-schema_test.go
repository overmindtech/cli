package manual_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/sql/armsql/v2"
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

// mockSqlDatabaseSchemasPager is a simple mock implementation of SqlDatabaseSchemasPager
type mockSqlDatabaseSchemasPager struct {
	pages []armsql.DatabaseSchemasClientListByDatabaseResponse
	index int
}

func (m *mockSqlDatabaseSchemasPager) More() bool {
	return m.index < len(m.pages)
}

func (m *mockSqlDatabaseSchemasPager) NextPage(ctx context.Context) (armsql.DatabaseSchemasClientListByDatabaseResponse, error) {
	if m.index >= len(m.pages) {
		return armsql.DatabaseSchemasClientListByDatabaseResponse{}, errors.New("no more pages")
	}
	page := m.pages[m.index]
	m.index++
	return page, nil
}

// errorSqlDatabaseSchemasPager is a mock pager that always returns an error
type errorSqlDatabaseSchemasPager struct{}

func (e *errorSqlDatabaseSchemasPager) More() bool {
	return true // Always return true so NextPage will be called
}

func (e *errorSqlDatabaseSchemasPager) NextPage(ctx context.Context) (armsql.DatabaseSchemasClientListByDatabaseResponse, error) {
	return armsql.DatabaseSchemasClientListByDatabaseResponse{}, errors.New("pager error")
}

// testSqlDatabaseSchemasClient wraps the mock to implement the correct interface
type testSqlDatabaseSchemasClient struct {
	*mocks.MockSqlDatabaseSchemasClient
	pager clients.SqlDatabaseSchemasPager
}

func (t *testSqlDatabaseSchemasClient) ListByDatabase(ctx context.Context, resourceGroupName, serverName, databaseName string) clients.SqlDatabaseSchemasPager {
	return t.pager
}

func TestSqlDatabaseSchema(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"
	serverName := "test-server"
	databaseName := "test-database"
	schemaName := "dbo"

	t.Run("Get", func(t *testing.T) {
		schema := createAzureDatabaseSchema(serverName, databaseName, schemaName)

		mockClient := mocks.NewMockSqlDatabaseSchemasClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, serverName, databaseName, schemaName).Return(
			armsql.DatabaseSchemasClientGetResponse{
				DatabaseSchema: *schema,
			}, nil)

		testClient := &testSqlDatabaseSchemasClient{MockSqlDatabaseSchemasClient: mockClient}
		wrapper := manual.NewSqlDatabaseSchema(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		// Get requires serverName, databaseName, and schemaName as query parts
		query := shared.CompositeLookupKey(serverName, databaseName, schemaName)
		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.SQLDatabaseSchema.String() {
			t.Errorf("Expected type %s, got %s", azureshared.SQLDatabaseSchema, sdpItem.GetType())
		}

		if sdpItem.GetUniqueAttribute() != "uniqueAttr" {
			t.Errorf("Expected unique attribute 'uniqueAttr', got %s", sdpItem.GetUniqueAttribute())
		}

		expectedUniqueAttrValue := shared.CompositeLookupKey(serverName, databaseName, schemaName)
		if sdpItem.UniqueAttributeValue() != expectedUniqueAttrValue {
			t.Errorf("Expected unique attribute value %s, got %s", expectedUniqueAttrValue, sdpItem.UniqueAttributeValue())
		}

		if sdpItem.GetScope() != subscriptionID+"."+resourceGroup {
			t.Errorf("Expected scope %s, got %s", subscriptionID+"."+resourceGroup, sdpItem.GetScope())
		}

		// Validate the item
		if err := sdpItem.Validate(); err != nil {
			t.Fatalf("Expected no validation error, got: %v", err)
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					// SQLDatabase parent link
					ExpectedType:   azureshared.SQLDatabase.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey(serverName, databaseName),
					ExpectedScope:  subscriptionID + "." + resourceGroup,
				},
			}

			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("Get_InvalidQueryParts", func(t *testing.T) {
		mockClient := mocks.NewMockSqlDatabaseSchemasClient(ctrl)
		testClient := &testSqlDatabaseSchemasClient{MockSqlDatabaseSchemasClient: mockClient}

		wrapper := manual.NewSqlDatabaseSchema(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		// Test with insufficient query parts (only server and database name)
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], shared.CompositeLookupKey(serverName, databaseName), true)
		if qErr == nil {
			t.Error("Expected error when providing insufficient query parts, but got nil")
		}
	})

	t.Run("GetWithEmptyServerName", func(t *testing.T) {
		mockClient := mocks.NewMockSqlDatabaseSchemasClient(ctrl)
		testClient := &testSqlDatabaseSchemasClient{MockSqlDatabaseSchemasClient: mockClient}

		wrapper := manual.NewSqlDatabaseSchema(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		// Test with empty server name
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], shared.CompositeLookupKey("", databaseName, schemaName), true)
		if qErr == nil {
			t.Error("Expected error when providing empty server name, but got nil")
		}
	})

	t.Run("GetWithEmptyDatabaseName", func(t *testing.T) {
		mockClient := mocks.NewMockSqlDatabaseSchemasClient(ctrl)
		testClient := &testSqlDatabaseSchemasClient{MockSqlDatabaseSchemasClient: mockClient}

		wrapper := manual.NewSqlDatabaseSchema(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		// Test with empty database name
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], shared.CompositeLookupKey(serverName, "", schemaName), true)
		if qErr == nil {
			t.Error("Expected error when providing empty database name, but got nil")
		}
	})

	t.Run("GetWithEmptySchemaName", func(t *testing.T) {
		mockClient := mocks.NewMockSqlDatabaseSchemasClient(ctrl)
		testClient := &testSqlDatabaseSchemasClient{MockSqlDatabaseSchemasClient: mockClient}

		wrapper := manual.NewSqlDatabaseSchema(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		// Test with empty schema name
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], shared.CompositeLookupKey(serverName, databaseName, ""), true)
		if qErr == nil {
			t.Error("Expected error when providing empty schema name, but got nil")
		}
	})

	t.Run("Search", func(t *testing.T) {
		schema1 := createAzureDatabaseSchema(serverName, databaseName, "dbo")
		schema2 := createAzureDatabaseSchema(serverName, databaseName, "sys")

		mockClient := mocks.NewMockSqlDatabaseSchemasClient(ctrl)
		mockPager := &mockSqlDatabaseSchemasPager{
			pages: []armsql.DatabaseSchemasClientListByDatabaseResponse{
				{
					DatabaseSchemaListResult: armsql.DatabaseSchemaListResult{
						Value: []*armsql.DatabaseSchema{schema1, schema2},
					},
				},
			},
		}

		testClient := &testSqlDatabaseSchemasClient{
			MockSqlDatabaseSchemasClient: mockClient,
			pager:                        mockPager,
		}

		wrapper := manual.NewSqlDatabaseSchema(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support Search operation")
		}

		sdpItems, err := searchable.Search(ctx, wrapper.Scopes()[0], shared.CompositeLookupKey(serverName, databaseName), true)
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

			if item.GetType() != azureshared.SQLDatabaseSchema.String() {
				t.Errorf("Expected type %s, got %s", azureshared.SQLDatabaseSchema, item.GetType())
			}
		}
	})

	t.Run("Search_WithNilName", func(t *testing.T) {
		schema1 := createAzureDatabaseSchema(serverName, databaseName, "dbo")
		schema2 := &armsql.DatabaseSchema{
			Name: nil, // Schema with nil name should be skipped
			ID:   new("/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.Sql/servers/test-server/databases/test-database/schemas/nil-schema"),
		}

		mockClient := mocks.NewMockSqlDatabaseSchemasClient(ctrl)
		mockPager := &mockSqlDatabaseSchemasPager{
			pages: []armsql.DatabaseSchemasClientListByDatabaseResponse{
				{
					DatabaseSchemaListResult: armsql.DatabaseSchemaListResult{
						Value: []*armsql.DatabaseSchema{schema1, schema2},
					},
				},
			},
		}

		testClient := &testSqlDatabaseSchemasClient{
			MockSqlDatabaseSchemasClient: mockClient,
			pager:                        mockPager,
		}

		wrapper := manual.NewSqlDatabaseSchema(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support Search operation")
		}

		sdpItems, err := searchable.Search(ctx, wrapper.Scopes()[0], shared.CompositeLookupKey(serverName, databaseName), true)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		// Should only return 1 item (schema with nil name is skipped)
		if len(sdpItems) != 1 {
			t.Fatalf("Expected 1 item (nil name filtered out), got: %d", len(sdpItems))
		}

		if sdpItems[0].UniqueAttributeValue() != shared.CompositeLookupKey(serverName, databaseName, "dbo") {
			t.Fatalf("Expected schema name 'dbo', got: %s", sdpItems[0].UniqueAttributeValue())
		}
	})

	t.Run("Search_InvalidQueryParts", func(t *testing.T) {
		mockClient := mocks.NewMockSqlDatabaseSchemasClient(ctrl)
		testClient := &testSqlDatabaseSchemasClient{MockSqlDatabaseSchemasClient: mockClient}

		wrapper := manual.NewSqlDatabaseSchema(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		// Test Search directly with insufficient query parts - should return error before calling ListByDatabase
		_, qErr := wrapper.Search(ctx, wrapper.Scopes()[0], serverName)
		if qErr == nil {
			t.Error("Expected error when providing insufficient query parts, but got nil")
		}
	})

	t.Run("SearchWithEmptyServerName", func(t *testing.T) {
		mockClient := mocks.NewMockSqlDatabaseSchemasClient(ctrl)
		testClient := &testSqlDatabaseSchemasClient{MockSqlDatabaseSchemasClient: mockClient}

		wrapper := manual.NewSqlDatabaseSchema(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		// Test Search with empty server name
		_, qErr := wrapper.Search(ctx, wrapper.Scopes()[0], "", databaseName)
		if qErr == nil {
			t.Error("Expected error when providing empty server name, but got nil")
		}
	})

	t.Run("SearchWithEmptyDatabaseName", func(t *testing.T) {
		mockClient := mocks.NewMockSqlDatabaseSchemasClient(ctrl)
		testClient := &testSqlDatabaseSchemasClient{MockSqlDatabaseSchemasClient: mockClient}

		wrapper := manual.NewSqlDatabaseSchema(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		// Test Search with empty database name
		_, qErr := wrapper.Search(ctx, wrapper.Scopes()[0], serverName, "")
		if qErr == nil {
			t.Error("Expected error when providing empty database name, but got nil")
		}
	})

	t.Run("ErrorHandling_Get", func(t *testing.T) {
		expectedErr := errors.New("schema not found")

		mockClient := mocks.NewMockSqlDatabaseSchemasClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, serverName, databaseName, "nonexistent-schema").Return(
			armsql.DatabaseSchemasClientGetResponse{}, expectedErr)

		testClient := &testSqlDatabaseSchemasClient{MockSqlDatabaseSchemasClient: mockClient}
		wrapper := manual.NewSqlDatabaseSchema(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(serverName, databaseName, "nonexistent-schema")
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr == nil {
			t.Error("Expected error when getting non-existent schema, but got nil")
		}
	})

	t.Run("ErrorHandling_Search", func(t *testing.T) {
		mockClient := mocks.NewMockSqlDatabaseSchemasClient(ctrl)
		// Create a pager that returns an error when NextPage is called
		errorPager := &errorSqlDatabaseSchemasPager{}

		testClient := &testSqlDatabaseSchemasClient{
			MockSqlDatabaseSchemasClient: mockClient,
			pager:                        errorPager,
		}

		wrapper := manual.NewSqlDatabaseSchema(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support Search operation")
		}

		_, err := searchable.Search(ctx, wrapper.Scopes()[0], shared.CompositeLookupKey(serverName, databaseName), true)
		if err == nil {
			t.Error("Expected error from pager when NextPage returns an error, but got nil")
		}
	})

	t.Run("InterfaceCompliance", func(t *testing.T) {
		mockClient := mocks.NewMockSqlDatabaseSchemasClient(ctrl)
		testClient := &testSqlDatabaseSchemasClient{MockSqlDatabaseSchemasClient: mockClient}
		wrapper := manual.NewSqlDatabaseSchema(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		// Cast to sources.Wrapper to access interface methods
		w := wrapper.(sources.Wrapper)

		// Verify IAMPermissions
		permissions := w.IAMPermissions()
		if len(permissions) == 0 {
			t.Error("Expected IAMPermissions to return at least one permission")
		}
		expectedPermission := "Microsoft.Sql/servers/databases/schemas/read"
		found := slices.Contains(permissions, expectedPermission)
		if !found {
			t.Errorf("Expected IAMPermissions to include %s", expectedPermission)
		}

		// Verify PotentialLinks
		potentialLinks := w.PotentialLinks()
		if len(potentialLinks) == 0 {
			t.Error("Expected PotentialLinks to return at least one link")
		}
		if !potentialLinks[azureshared.SQLDatabase] {
			t.Error("Expected PotentialLinks to include SQLDatabase")
		}
	})
}

// createAzureDatabaseSchema creates a mock Azure database schema for testing
func createAzureDatabaseSchema(serverName, databaseName, schemaName string) *armsql.DatabaseSchema {
	schemaID := "/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.Sql/servers/" + serverName + "/databases/" + databaseName + "/schemas/" + schemaName

	return &armsql.DatabaseSchema{
		Name: new(schemaName),
		ID:   new(schemaID),
		Type: new("Microsoft.Sql/servers/databases/schemas"),
	}
}
