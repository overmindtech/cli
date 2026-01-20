package manual_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/sql/armsql"
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

// mockSqlDatabasesPager is a simple mock implementation of SqlDatabasesPager
type mockSqlDatabasesPager struct {
	pages []armsql.DatabasesClientListByServerResponse
	index int
}

func (m *mockSqlDatabasesPager) More() bool {
	return m.index < len(m.pages)
}

func (m *mockSqlDatabasesPager) NextPage(ctx context.Context) (armsql.DatabasesClientListByServerResponse, error) {
	if m.index >= len(m.pages) {
		return armsql.DatabasesClientListByServerResponse{}, errors.New("no more pages")
	}
	page := m.pages[m.index]
	m.index++
	return page, nil
}

// errorSqlDatabasesPager is a mock pager that always returns an error
type errorSqlDatabasesPager struct{}

func (e *errorSqlDatabasesPager) More() bool {
	return true // Always return true so NextPage will be called
}

func (e *errorSqlDatabasesPager) NextPage(ctx context.Context) (armsql.DatabasesClientListByServerResponse, error) {
	return armsql.DatabasesClientListByServerResponse{}, errors.New("pager error")
}

// testSqlDatabasesClient wraps the mock to implement the correct interface
type testSqlDatabasesClient struct {
	*mocks.MockSqlDatabasesClient
	pager clients.SqlDatabasesPager
}

func (t *testSqlDatabasesClient) ListByServer(ctx context.Context, resourceGroupName, serverName string) clients.SqlDatabasesPager {
	return t.pager
}

func TestSqlDatabase(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"
	serverName := "test-server"
	databaseName := "test-database"

	t.Run("Get", func(t *testing.T) {
		database := createAzureSqlDatabase(serverName, databaseName, "")

		mockClient := mocks.NewMockSqlDatabasesClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, serverName, databaseName).Return(
			armsql.DatabasesClientGetResponse{
				Database: *database,
			}, nil)

		testClient := &testSqlDatabasesClient{MockSqlDatabasesClient: mockClient}
		wrapper := manual.NewSqlDatabase(testClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		// Get requires serverName and databaseName as query parts
		query := shared.CompositeLookupKey(serverName, databaseName)
		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.SQLDatabase.String() {
			t.Errorf("Expected type %s, got %s", azureshared.SQLDatabase, sdpItem.GetType())
		}

		if sdpItem.GetUniqueAttribute() != "uniqueAttr" {
			t.Errorf("Expected unique attribute 'uniqueAttr', got %s", sdpItem.GetUniqueAttribute())
		}

		expectedUniqueAttrValue := shared.CompositeLookupKey(serverName, databaseName)
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
					// SQLServer link
					ExpectedType:   azureshared.SQLServer.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  serverName,
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
			}

			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("Get_WithElasticPool", func(t *testing.T) {
		elasticPoolID := "/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.Sql/servers/test-server/elasticPools/test-pool"
		database := createAzureSqlDatabase(serverName, databaseName, elasticPoolID)

		mockClient := mocks.NewMockSqlDatabasesClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, serverName, databaseName).Return(
			armsql.DatabasesClientGetResponse{
				Database: *database,
			}, nil)

		testClient := &testSqlDatabasesClient{MockSqlDatabasesClient: mockClient}
		wrapper := manual.NewSqlDatabase(testClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := serverName + shared.QuerySeparator + databaseName
		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					// SQLServer link
					ExpectedType:   azureshared.SQLServer.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  serverName,
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					// SQLElasticPool link
					ExpectedType:   azureshared.SQLElasticPool.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-pool",
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
			}

			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("Get_InvalidQueryParts", func(t *testing.T) {
		mockClient := mocks.NewMockSqlDatabasesClient(ctrl)
		testClient := &testSqlDatabasesClient{MockSqlDatabasesClient: mockClient}

		wrapper := manual.NewSqlDatabase(testClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		// Test with insufficient query parts (only server name)
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], serverName, true)
		if qErr == nil {
			t.Error("Expected error when providing insufficient query parts, but got nil")
		}
	})

	t.Run("Search", func(t *testing.T) {
		database1 := createAzureSqlDatabase(serverName, "database-1", "")
		database2 := createAzureSqlDatabase(serverName, "database-2", "")

		mockClient := mocks.NewMockSqlDatabasesClient(ctrl)
		mockPager := &mockSqlDatabasesPager{
			pages: []armsql.DatabasesClientListByServerResponse{
				{
					DatabaseListResult: armsql.DatabaseListResult{
						Value: []*armsql.Database{database1, database2},
					},
				},
			},
		}

		testClient := &testSqlDatabasesClient{
			MockSqlDatabasesClient: mockClient,
			pager:                  mockPager,
		}

		wrapper := manual.NewSqlDatabase(testClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support Search operation")
		}

		sdpItems, err := searchable.Search(ctx, wrapper.Scopes()[0], serverName, true)
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

			if item.GetType() != azureshared.SQLDatabase.String() {
				t.Errorf("Expected type %s, got %s", azureshared.SQLDatabase, item.GetType())
			}
		}
	})

	t.Run("Search_WithNilName", func(t *testing.T) {
		database1 := createAzureSqlDatabase(serverName, "database-1", "")
		database2 := &armsql.Database{
			Name:     nil, // Database with nil name should be skipped
			Location: to.Ptr("eastus"),
			Tags: map[string]*string{
				"env": to.Ptr("test"),
			},
			ID: to.Ptr("/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.Sql/servers/test-server/databases/database-2"),
			Properties: &armsql.DatabaseProperties{
				Status: to.Ptr(armsql.DatabaseStatusOnline),
			},
		}

		mockClient := mocks.NewMockSqlDatabasesClient(ctrl)
		mockPager := &mockSqlDatabasesPager{
			pages: []armsql.DatabasesClientListByServerResponse{
				{
					DatabaseListResult: armsql.DatabaseListResult{
						Value: []*armsql.Database{database1, database2},
					},
				},
			},
		}

		testClient := &testSqlDatabasesClient{
			MockSqlDatabasesClient: mockClient,
			pager:                  mockPager,
		}

		wrapper := manual.NewSqlDatabase(testClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support Search operation")
		}

		sdpItems, err := searchable.Search(ctx, wrapper.Scopes()[0], serverName, true)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		// Should only return 1 item (database with nil name is skipped)
		if len(sdpItems) != 1 {
			t.Fatalf("Expected 1 item (nil name filtered out), got: %d", len(sdpItems))
		}

		if sdpItems[0].UniqueAttributeValue() != shared.CompositeLookupKey(serverName, "database-1") {
			t.Fatalf("Expected database name 'database-1', got: %s", sdpItems[0].UniqueAttributeValue())
		}
	})

	t.Run("Search_InvalidQueryParts", func(t *testing.T) {
		mockClient := mocks.NewMockSqlDatabasesClient(ctrl)
		testClient := &testSqlDatabasesClient{MockSqlDatabasesClient: mockClient}

		wrapper := manual.NewSqlDatabase(testClient, subscriptionID, resourceGroup)

		// Test Search directly with no query parts - should return error before calling ListByServer
		_, qErr := wrapper.Search(ctx, wrapper.Scopes()[0])
		if qErr == nil {
			t.Error("Expected error when providing no query parts, but got nil")
		}
	})

	t.Run("ErrorHandling_Get", func(t *testing.T) {
		expectedErr := errors.New("database not found")

		mockClient := mocks.NewMockSqlDatabasesClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, serverName, "nonexistent-database").Return(
			armsql.DatabasesClientGetResponse{}, expectedErr)

		testClient := &testSqlDatabasesClient{MockSqlDatabasesClient: mockClient}
		wrapper := manual.NewSqlDatabase(testClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := serverName + shared.QuerySeparator + "nonexistent-database"
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr == nil {
			t.Error("Expected error when getting non-existent database, but got nil")
		}
	})

	t.Run("ErrorHandling_Search", func(t *testing.T) {
		mockClient := mocks.NewMockSqlDatabasesClient(ctrl)
		// Create a pager that returns an error when NextPage is called
		errorPager := &errorSqlDatabasesPager{}

		testClient := &testSqlDatabasesClient{
			MockSqlDatabasesClient: mockClient,
			pager:                  errorPager,
		}

		wrapper := manual.NewSqlDatabase(testClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support Search operation")
		}

		_, err := searchable.Search(ctx, wrapper.Scopes()[0], serverName, true)
		// The Search implementation should return an error when pager.NextPage returns an error
		if err == nil {
			t.Error("Expected error from pager when NextPage returns an error, but got nil")
		}
	})

	t.Run("InterfaceCompliance", func(t *testing.T) {
		mockClient := mocks.NewMockSqlDatabasesClient(ctrl)
		testClient := &testSqlDatabasesClient{MockSqlDatabasesClient: mockClient}
		wrapper := manual.NewSqlDatabase(testClient, subscriptionID, resourceGroup)

		// Cast to sources.Wrapper to access interface methods
		w := wrapper.(sources.Wrapper)

		// Verify IAMPermissions
		permissions := w.IAMPermissions()
		if len(permissions) == 0 {
			t.Error("Expected IAMPermissions to return at least one permission")
		}
		expectedPermission := "Microsoft.Sql/servers/databases/read"
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

		// Verify PotentialLinks
		potentialLinks := w.PotentialLinks()
		if len(potentialLinks) == 0 {
			t.Error("Expected PotentialLinks to return at least one link")
		}
		/* //todo: uncomment when sql server adapter and elastic pool adapter are made
		if !potentialLinks[azureshared.SQLServer] {
			t.Error("Expected PotentialLinks to include SQLServer")
		}
		if !potentialLinks[azureshared.SQLElasticPool] {
			t.Error("Expected PotentialLinks to include SQLElasticPool")
		}*/

		// Verify TerraformMappings
		mappings := w.TerraformMappings()
		if len(mappings) == 0 {
			t.Error("Expected TerraformMappings to return at least one mapping")
		}
		foundMapping := false
		for _, mapping := range mappings {
			if mapping.GetTerraformQueryMap() == "azurerm_mssql_database.id" {
				foundMapping = true
				break
			}
		}
		if !foundMapping {
			t.Error("Expected TerraformMappings to include 'azurerm_mssql_database.id' mapping")
		}
	})
}

// createAzureSqlDatabase creates a mock Azure SQL database for testing
func createAzureSqlDatabase(serverName, databaseName, elasticPoolID string) *armsql.Database {
	databaseID := "/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.Sql/servers/" + serverName + "/databases/" + databaseName

	db := &armsql.Database{
		Name:     to.Ptr(databaseName),
		Location: to.Ptr("eastus"),
		Tags: map[string]*string{
			"env":     to.Ptr("test"),
			"project": to.Ptr("testing"),
		},
		ID: to.Ptr(databaseID),
		Properties: &armsql.DatabaseProperties{
			Status: to.Ptr(armsql.DatabaseStatusOnline),
		},
	}

	if elasticPoolID != "" {
		db.Properties.ElasticPoolID = to.Ptr(elasticPoolID)
	}

	return db
}
