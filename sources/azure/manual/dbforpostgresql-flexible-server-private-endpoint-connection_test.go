package manual_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/postgresql/armpostgresqlflexibleservers/v5"
	"go.uber.org/mock/gomock"

	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	"github.com/overmindtech/cli/sources/azure/manual"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/azure/shared/mocks"
	"github.com/overmindtech/cli/sources/shared"
)

type mockDBforPostgreSQLFlexibleServerPrivateEndpointConnectionsPager struct {
	pages []armpostgresqlflexibleservers.PrivateEndpointConnectionsClientListByServerResponse
	index int
}

func (m *mockDBforPostgreSQLFlexibleServerPrivateEndpointConnectionsPager) More() bool {
	return m.index < len(m.pages)
}

func (m *mockDBforPostgreSQLFlexibleServerPrivateEndpointConnectionsPager) NextPage(ctx context.Context) (armpostgresqlflexibleservers.PrivateEndpointConnectionsClientListByServerResponse, error) {
	if m.index >= len(m.pages) {
		return armpostgresqlflexibleservers.PrivateEndpointConnectionsClientListByServerResponse{}, errors.New("no more pages")
	}
	page := m.pages[m.index]
	m.index++
	return page, nil
}

type testDBforPostgreSQLFlexibleServerPrivateEndpointConnectionsClient struct {
	*mocks.MockDBforPostgreSQLFlexibleServerPrivateEndpointConnectionsClient
	pager clients.DBforPostgreSQLFlexibleServerPrivateEndpointConnectionsPager
}

func (t *testDBforPostgreSQLFlexibleServerPrivateEndpointConnectionsClient) ListByServer(ctx context.Context, resourceGroupName, serverName string) clients.DBforPostgreSQLFlexibleServerPrivateEndpointConnectionsPager {
	return t.pager
}

func TestDBforPostgreSQLFlexibleServerPrivateEndpointConnection(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"
	serverName := "test-pg-server"
	connectionName := "test-pec"

	t.Run("Get", func(t *testing.T) {
		conn := createAzureDBforPostgreSQLFlexibleServerPrivateEndpointConnection(connectionName, "")

		mockClient := mocks.NewMockDBforPostgreSQLFlexibleServerPrivateEndpointConnectionsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, serverName, connectionName).Return(
			armpostgresqlflexibleservers.PrivateEndpointConnectionsClientGetResponse{
				PrivateEndpointConnection: *conn,
			}, nil)

		testClient := &testDBforPostgreSQLFlexibleServerPrivateEndpointConnectionsClient{MockDBforPostgreSQLFlexibleServerPrivateEndpointConnectionsClient: mockClient}
		wrapper := manual.NewDBforPostgreSQLFlexibleServerPrivateEndpointConnection(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(serverName, connectionName)
		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.DBforPostgreSQLFlexibleServerPrivateEndpointConnection.String() {
			t.Errorf("Expected type %s, got %s", azureshared.DBforPostgreSQLFlexibleServerPrivateEndpointConnection, sdpItem.GetType())
		}

		if sdpItem.GetUniqueAttribute() != "uniqueAttr" {
			t.Errorf("Expected unique attribute 'uniqueAttr', got %s", sdpItem.GetUniqueAttribute())
		}

		if sdpItem.UniqueAttributeValue() != shared.CompositeLookupKey(serverName, connectionName) {
			t.Errorf("Expected unique attribute value %s, got %s", shared.CompositeLookupKey(serverName, connectionName), sdpItem.UniqueAttributeValue())
		}

		if sdpItem.GetScope() != subscriptionID+"."+resourceGroup {
			t.Errorf("Expected scope %s, got %s", subscriptionID+"."+resourceGroup, sdpItem.GetScope())
		}

		if err := sdpItem.Validate(); err != nil {
			t.Fatalf("Expected no validation error, got: %v", err)
		}

		t.Run("StaticTests", func(t *testing.T) {
			linkedQueries := sdpItem.GetLinkedItemQueries()
			if len(linkedQueries) < 1 {
				t.Fatalf("Expected at least 1 linked query, got: %d", len(linkedQueries))
			}

			foundFlexibleServer := false
			for _, lq := range linkedQueries {
				if lq.GetQuery().GetType() == azureshared.DBforPostgreSQLFlexibleServer.String() {
					foundFlexibleServer = true
					if lq.GetQuery().GetMethod() != sdp.QueryMethod_GET {
						t.Errorf("Expected DBforPostgreSQLFlexibleServer link method GET, got %v", lq.GetQuery().GetMethod())
					}
					if lq.GetQuery().GetQuery() != serverName {
						t.Errorf("Expected DBforPostgreSQLFlexibleServer query %s, got %s", serverName, lq.GetQuery().GetQuery())
					}
				}
			}
			if !foundFlexibleServer {
				t.Error("Expected linked query to DBforPostgreSQLFlexibleServer")
			}
		})
	})

	t.Run("Get_WithPrivateEndpointLink", func(t *testing.T) {
		peID := "/subscriptions/test-subscription/resourceGroups/other-rg/providers/Microsoft.Network/privateEndpoints/test-pe"
		conn := createAzureDBforPostgreSQLFlexibleServerPrivateEndpointConnection(connectionName, peID)

		mockClient := mocks.NewMockDBforPostgreSQLFlexibleServerPrivateEndpointConnectionsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, serverName, connectionName).Return(
			armpostgresqlflexibleservers.PrivateEndpointConnectionsClientGetResponse{
				PrivateEndpointConnection: *conn,
			}, nil)

		testClient := &testDBforPostgreSQLFlexibleServerPrivateEndpointConnectionsClient{MockDBforPostgreSQLFlexibleServerPrivateEndpointConnectionsClient: mockClient}
		wrapper := manual.NewDBforPostgreSQLFlexibleServerPrivateEndpointConnection(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(serverName, connectionName)
		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		foundPrivateEndpoint := false
		for _, lq := range sdpItem.GetLinkedItemQueries() {
			if lq.GetQuery().GetType() == azureshared.NetworkPrivateEndpoint.String() {
				foundPrivateEndpoint = true
				if lq.GetQuery().GetQuery() != "test-pe" {
					t.Errorf("Expected NetworkPrivateEndpoint query 'test-pe', got %s", lq.GetQuery().GetQuery())
				}
			}
		}
		if !foundPrivateEndpoint {
			t.Error("Expected linked query to NetworkPrivateEndpoint when PrivateEndpoint ID is set")
		}
	})

	t.Run("Search", func(t *testing.T) {
		conn1 := createAzureDBforPostgreSQLFlexibleServerPrivateEndpointConnection("pec-1", "")
		conn2 := createAzureDBforPostgreSQLFlexibleServerPrivateEndpointConnection("pec-2", "")

		mockClient := mocks.NewMockDBforPostgreSQLFlexibleServerPrivateEndpointConnectionsClient(ctrl)
		mockPager := &mockDBforPostgreSQLFlexibleServerPrivateEndpointConnectionsPager{
			pages: []armpostgresqlflexibleservers.PrivateEndpointConnectionsClientListByServerResponse{
				{
					PrivateEndpointConnectionList: armpostgresqlflexibleservers.PrivateEndpointConnectionList{
						Value: []*armpostgresqlflexibleservers.PrivateEndpointConnection{conn1, conn2},
					},
				},
			},
		}
		testClient := &testDBforPostgreSQLFlexibleServerPrivateEndpointConnectionsClient{
			MockDBforPostgreSQLFlexibleServerPrivateEndpointConnectionsClient: mockClient,
			pager: mockPager,
		}

		wrapper := manual.NewDBforPostgreSQLFlexibleServerPrivateEndpointConnection(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		items, qErr := wrapper.Search(ctx, subscriptionID+"."+resourceGroup, serverName)
		if qErr != nil {
			t.Fatalf("Search failed: %v", qErr)
		}
		if len(items) != 2 {
			t.Errorf("Expected 2 items, got %d", len(items))
		}
		for _, item := range items {
			if item.GetType() != azureshared.DBforPostgreSQLFlexibleServerPrivateEndpointConnection.String() {
				t.Errorf("Expected type %s, got %s", azureshared.DBforPostgreSQLFlexibleServerPrivateEndpointConnection, item.GetType())
			}
		}
	})

	t.Run("Search_NilNameSkipped", func(t *testing.T) {
		validConn := createAzureDBforPostgreSQLFlexibleServerPrivateEndpointConnection("valid-pec", "")

		mockClient := mocks.NewMockDBforPostgreSQLFlexibleServerPrivateEndpointConnectionsClient(ctrl)
		mockPager := &mockDBforPostgreSQLFlexibleServerPrivateEndpointConnectionsPager{
			pages: []armpostgresqlflexibleservers.PrivateEndpointConnectionsClientListByServerResponse{
				{
					PrivateEndpointConnectionList: armpostgresqlflexibleservers.PrivateEndpointConnectionList{
						Value: []*armpostgresqlflexibleservers.PrivateEndpointConnection{
							nil,
							{Name: nil},
							validConn,
						},
					},
				},
			},
		}
		testClient := &testDBforPostgreSQLFlexibleServerPrivateEndpointConnectionsClient{
			MockDBforPostgreSQLFlexibleServerPrivateEndpointConnectionsClient: mockClient,
			pager: mockPager,
		}

		wrapper := manual.NewDBforPostgreSQLFlexibleServerPrivateEndpointConnection(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		items, qErr := wrapper.Search(ctx, subscriptionID+"."+resourceGroup, serverName)
		if qErr != nil {
			t.Fatalf("Search failed: %v", qErr)
		}
		if len(items) != 1 {
			t.Errorf("Expected 1 item (nil names skipped), got %d", len(items))
		}
	})

	t.Run("GetWithInsufficientQueryParts", func(t *testing.T) {
		mockClient := mocks.NewMockDBforPostgreSQLFlexibleServerPrivateEndpointConnectionsClient(ctrl)
		testClient := &testDBforPostgreSQLFlexibleServerPrivateEndpointConnectionsClient{MockDBforPostgreSQLFlexibleServerPrivateEndpointConnectionsClient: mockClient}
		wrapper := manual.NewDBforPostgreSQLFlexibleServerPrivateEndpointConnection(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], serverName, true)
		if qErr == nil {
			t.Error("Expected error when query has only serverName")
		}
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		expectedErr := errors.New("connection not found")
		mockClient := mocks.NewMockDBforPostgreSQLFlexibleServerPrivateEndpointConnectionsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, serverName, connectionName).Return(
			armpostgresqlflexibleservers.PrivateEndpointConnectionsClientGetResponse{}, expectedErr)

		testClient := &testDBforPostgreSQLFlexibleServerPrivateEndpointConnectionsClient{MockDBforPostgreSQLFlexibleServerPrivateEndpointConnectionsClient: mockClient}
		wrapper := manual.NewDBforPostgreSQLFlexibleServerPrivateEndpointConnection(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(serverName, connectionName)
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr == nil {
			t.Fatal("Expected error when Get fails")
		}
	})

	t.Run("PotentialLinks", func(t *testing.T) {
		wrapper := manual.NewDBforPostgreSQLFlexibleServerPrivateEndpointConnection(nil, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		links := wrapper.PotentialLinks()
		if !links[azureshared.DBforPostgreSQLFlexibleServer] {
			t.Error("Expected PotentialLinks to include DBforPostgreSQLFlexibleServer")
		}
		if !links[azureshared.NetworkPrivateEndpoint] {
			t.Error("Expected PotentialLinks to include NetworkPrivateEndpoint")
		}
	})
}

func createAzureDBforPostgreSQLFlexibleServerPrivateEndpointConnection(connectionName, privateEndpointID string) *armpostgresqlflexibleservers.PrivateEndpointConnection {
	state := armpostgresqlflexibleservers.PrivateEndpointConnectionProvisioningStateSucceeded
	conn := &armpostgresqlflexibleservers.PrivateEndpointConnection{
		ID:   new("/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.DBforPostgreSQL/flexibleServers/test-pg-server/privateEndpointConnections/" + connectionName),
		Name: new(connectionName),
		Type: new("Microsoft.DBforPostgreSQL/flexibleServers/privateEndpointConnections"),
		Properties: &armpostgresqlflexibleservers.PrivateEndpointConnectionProperties{
			ProvisioningState: &state,
		},
	}
	if privateEndpointID != "" {
		conn.Properties.PrivateEndpoint = &armpostgresqlflexibleservers.PrivateEndpoint{
			ID: new(privateEndpointID),
		}
	}
	return conn
}
