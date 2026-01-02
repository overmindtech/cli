package manual_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/postgresql/armpostgresqlflexibleservers"
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

// mockPostgreSQLDatabasesPager is a simple mock implementation of PostgreSQLDatabasesPager
type mockPostgreSQLDatabasesPager struct {
	pages []armpostgresqlflexibleservers.DatabasesClientListByServerResponse
	index int
}

func (m *mockPostgreSQLDatabasesPager) More() bool {
	return m.index < len(m.pages)
}

func (m *mockPostgreSQLDatabasesPager) NextPage(ctx context.Context) (armpostgresqlflexibleservers.DatabasesClientListByServerResponse, error) {
	if m.index >= len(m.pages) {
		return armpostgresqlflexibleservers.DatabasesClientListByServerResponse{}, errors.New("no more pages")
	}
	page := m.pages[m.index]
	m.index++
	return page, nil
}

// errorPostgreSQLDatabasesPager is a mock pager that always returns an error
type errorPostgreSQLDatabasesPager struct{}

func (e *errorPostgreSQLDatabasesPager) More() bool {
	return true // Always return true so NextPage will be called
}

func (e *errorPostgreSQLDatabasesPager) NextPage(ctx context.Context) (armpostgresqlflexibleservers.DatabasesClientListByServerResponse, error) {
	return armpostgresqlflexibleservers.DatabasesClientListByServerResponse{}, errors.New("pager error")
}

// testPostgreSQLDatabasesClient wraps the mock to implement the correct interface
type testPostgreSQLDatabasesClient struct {
	*mocks.MockPostgreSQLDatabasesClient
	pager clients.PostgreSQLDatabasesPager
}

func (t *testPostgreSQLDatabasesClient) ListByServer(ctx context.Context, resourceGroupName, serverName string) clients.PostgreSQLDatabasesPager {
	return t.pager
}

func TestDBforPostgreSQLDatabase(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"
	serverName := "test-server"
	databaseName := "test-database"

	t.Run("Get", func(t *testing.T) {
		database := createAzurePostgreSQLDatabase(serverName, databaseName)

		mockClient := mocks.NewMockPostgreSQLDatabasesClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, serverName, databaseName).Return(
			armpostgresqlflexibleservers.DatabasesClientGetResponse{
				Database: *database,
			}, nil)

		testClient := &testPostgreSQLDatabasesClient{MockPostgreSQLDatabasesClient: mockClient}
		wrapper := manual.NewDBforPostgreSQLDatabase(testClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper)

		// Get requires serverName and databaseName as query parts
		query := shared.CompositeLookupKey(serverName, databaseName)
		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.DBforPostgreSQLDatabase.String() {
			t.Errorf("Expected type %s, got %s", azureshared.DBforPostgreSQLDatabase, sdpItem.GetType())
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
					// PostgreSQL Flexible Server link
					ExpectedType:   azureshared.DBforPostgreSQLFlexibleServer.String(),
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

	t.Run("Get_InvalidQueryParts", func(t *testing.T) {
		mockClient := mocks.NewMockPostgreSQLDatabasesClient(ctrl)
		testClient := &testPostgreSQLDatabasesClient{MockPostgreSQLDatabasesClient: mockClient}

		wrapper := manual.NewDBforPostgreSQLDatabase(testClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper)

		// Test with insufficient query parts (only server name)
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], serverName, true)
		if qErr == nil {
			t.Error("Expected error when providing insufficient query parts, but got nil")
		}
	})

	t.Run("Search", func(t *testing.T) {
		database1 := createAzurePostgreSQLDatabase(serverName, "database-1")
		database2 := createAzurePostgreSQLDatabase(serverName, "database-2")

		mockClient := mocks.NewMockPostgreSQLDatabasesClient(ctrl)
		mockPager := &mockPostgreSQLDatabasesPager{
			pages: []armpostgresqlflexibleservers.DatabasesClientListByServerResponse{
				{
					DatabaseListResult: armpostgresqlflexibleservers.DatabaseListResult{
						Value: []*armpostgresqlflexibleservers.Database{database1, database2},
					},
				},
			},
		}

		testClient := &testPostgreSQLDatabasesClient{
			MockPostgreSQLDatabasesClient: mockClient,
			pager:                          mockPager,
		}

		wrapper := manual.NewDBforPostgreSQLDatabase(testClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper)

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

			if item.GetType() != azureshared.DBforPostgreSQLDatabase.String() {
				t.Errorf("Expected type %s, got %s", azureshared.DBforPostgreSQLDatabase, item.GetType())
			}
		}
	})

	t.Run("Search_WithNilName", func(t *testing.T) {
		database1 := createAzurePostgreSQLDatabase(serverName, "database-1")
		database2 := &armpostgresqlflexibleservers.Database{
			Name: nil, // Database with nil name should be skipped
			Properties: &armpostgresqlflexibleservers.DatabaseProperties{
				Charset:   to.Ptr("UTF8"),
				Collation: to.Ptr("en_US.utf8"),
			},
		}

		mockClient := mocks.NewMockPostgreSQLDatabasesClient(ctrl)
		mockPager := &mockPostgreSQLDatabasesPager{
			pages: []armpostgresqlflexibleservers.DatabasesClientListByServerResponse{
				{
					DatabaseListResult: armpostgresqlflexibleservers.DatabaseListResult{
						Value: []*armpostgresqlflexibleservers.Database{database1, database2},
					},
				},
			},
		}

		testClient := &testPostgreSQLDatabasesClient{
			MockPostgreSQLDatabasesClient: mockClient,
			pager:                          mockPager,
		}

		wrapper := manual.NewDBforPostgreSQLDatabase(testClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper)

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
		mockClient := mocks.NewMockPostgreSQLDatabasesClient(ctrl)
		testClient := &testPostgreSQLDatabasesClient{MockPostgreSQLDatabasesClient: mockClient}

		wrapper := manual.NewDBforPostgreSQLDatabase(testClient, subscriptionID, resourceGroup)

		// Test Search directly with no query parts - should return error before calling ListByServer
		_, qErr := wrapper.Search(ctx)
		if qErr == nil {
			t.Error("Expected error when providing no query parts, but got nil")
		}
	})

	t.Run("ErrorHandling_Get", func(t *testing.T) {
		expectedErr := errors.New("database not found")

		mockClient := mocks.NewMockPostgreSQLDatabasesClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, serverName, "nonexistent-database").Return(
			armpostgresqlflexibleservers.DatabasesClientGetResponse{}, expectedErr)

		testClient := &testPostgreSQLDatabasesClient{MockPostgreSQLDatabasesClient: mockClient}
		wrapper := manual.NewDBforPostgreSQLDatabase(testClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper)

		query := shared.CompositeLookupKey(serverName, "nonexistent-database")
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr == nil {
			t.Error("Expected error when getting non-existent database, but got nil")
		}
	})

	t.Run("ErrorHandling_Search", func(t *testing.T) {
		mockClient := mocks.NewMockPostgreSQLDatabasesClient(ctrl)
		// Create a pager that returns an error when NextPage is called
		errorPager := &errorPostgreSQLDatabasesPager{}

		testClient := &testPostgreSQLDatabasesClient{
			MockPostgreSQLDatabasesClient: mockClient,
			pager:                          errorPager,
		}

		wrapper := manual.NewDBforPostgreSQLDatabase(testClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper)

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support Search operation")
		}

		_, err := searchable.Search(ctx, wrapper.Scopes()[0], serverName, true)
		// The Search implementation should return an error when pager.NextPage returns an error
		if err == nil {
			t.Error("Expected error when pager returns error, but got nil")
		}
	})
}

// createAzurePostgreSQLDatabase creates a mock Azure PostgreSQL Database for testing
func createAzurePostgreSQLDatabase(serverName, databaseName string) *armpostgresqlflexibleservers.Database {
	databaseID := "/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.DBforPostgreSQL/flexibleServers/" + serverName + "/databases/" + databaseName

	return &armpostgresqlflexibleservers.Database{
		Name: to.Ptr(databaseName),
		ID:   to.Ptr(databaseID),
		Properties: &armpostgresqlflexibleservers.DatabaseProperties{
			Charset:   to.Ptr("UTF8"),
			Collation: to.Ptr("en_US.utf8"),
		},
	}
}

