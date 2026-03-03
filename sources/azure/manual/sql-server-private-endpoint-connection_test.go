package manual_test

import (
	"context"
	"errors"
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

type mockSQLServerPrivateEndpointConnectionsPager struct {
	pages []armsql.PrivateEndpointConnectionsClientListByServerResponse
	index int
}

func (m *mockSQLServerPrivateEndpointConnectionsPager) More() bool {
	return m.index < len(m.pages)
}

func (m *mockSQLServerPrivateEndpointConnectionsPager) NextPage(ctx context.Context) (armsql.PrivateEndpointConnectionsClientListByServerResponse, error) {
	if m.index >= len(m.pages) {
		return armsql.PrivateEndpointConnectionsClientListByServerResponse{}, errors.New("no more pages")
	}
	page := m.pages[m.index]
	m.index++
	return page, nil
}

type testSQLServerPrivateEndpointConnectionsClient struct {
	*mocks.MockSQLServerPrivateEndpointConnectionsClient
	pager clients.SQLServerPrivateEndpointConnectionsPager
}

func (t *testSQLServerPrivateEndpointConnectionsClient) ListByServer(ctx context.Context, resourceGroupName, serverName string) clients.SQLServerPrivateEndpointConnectionsPager {
	return t.pager
}

func TestSQLServerPrivateEndpointConnection(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"
	serverName := "test-sql-server"
	connectionName := "test-pec"

	t.Run("Get", func(t *testing.T) {
		conn := createAzureSQLServerPrivateEndpointConnection(connectionName, "")

		mockClient := mocks.NewMockSQLServerPrivateEndpointConnectionsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, serverName, connectionName).Return(
			armsql.PrivateEndpointConnectionsClientGetResponse{
				PrivateEndpointConnection: *conn,
			}, nil)

		testClient := &testSQLServerPrivateEndpointConnectionsClient{MockSQLServerPrivateEndpointConnectionsClient: mockClient}
		wrapper := manual.NewSQLServerPrivateEndpointConnection(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(serverName, connectionName)
		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.SQLServerPrivateEndpointConnection.String() {
			t.Errorf("Expected type %s, got %s", azureshared.SQLServerPrivateEndpointConnection, sdpItem.GetType())
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

			foundSQLServer := false
			for _, lq := range linkedQueries {
				if lq.GetQuery().GetType() == azureshared.SQLServer.String() {
					foundSQLServer = true
					if lq.GetQuery().GetMethod() != sdp.QueryMethod_GET {
						t.Errorf("Expected SQLServer link method GET, got %v", lq.GetQuery().GetMethod())
					}
					if lq.GetQuery().GetQuery() != serverName {
						t.Errorf("Expected SQLServer query %s, got %s", serverName, lq.GetQuery().GetQuery())
					}
				}
			}
			if !foundSQLServer {
				t.Error("Expected linked query to SQLServer")
			}
		})
	})

	t.Run("Get_WithPrivateEndpointLink", func(t *testing.T) {
		peID := "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/privateEndpoints/test-pe"
		conn := createAzureSQLServerPrivateEndpointConnection(connectionName, peID)

		mockClient := mocks.NewMockSQLServerPrivateEndpointConnectionsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, serverName, connectionName).Return(
			armsql.PrivateEndpointConnectionsClientGetResponse{
				PrivateEndpointConnection: *conn,
			}, nil)

		testClient := &testSQLServerPrivateEndpointConnectionsClient{MockSQLServerPrivateEndpointConnectionsClient: mockClient}
		wrapper := manual.NewSQLServerPrivateEndpointConnection(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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
				break
			}
		}
		if !foundPrivateEndpoint {
			t.Error("Expected linked query to NetworkPrivateEndpoint when PrivateEndpoint ID is set")
		}
	})

	t.Run("GetWithInsufficientQueryParts", func(t *testing.T) {
		mockClient := mocks.NewMockSQLServerPrivateEndpointConnectionsClient(ctrl)
		testClient := &testSQLServerPrivateEndpointConnectionsClient{MockSQLServerPrivateEndpointConnectionsClient: mockClient}

		wrapper := manual.NewSQLServerPrivateEndpointConnection(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], serverName, true)
		if qErr == nil {
			t.Error("Expected error when providing insufficient query parts, but got nil")
		}
	})

	t.Run("Search", func(t *testing.T) {
		conn1 := createAzureSQLServerPrivateEndpointConnection("pec-1", "")
		conn2 := createAzureSQLServerPrivateEndpointConnection("pec-2", "")

		mockClient := mocks.NewMockSQLServerPrivateEndpointConnectionsClient(ctrl)
		mockPager := &mockSQLServerPrivateEndpointConnectionsPager{
			pages: []armsql.PrivateEndpointConnectionsClientListByServerResponse{
				{
					PrivateEndpointConnectionListResult: armsql.PrivateEndpointConnectionListResult{
						Value: []*armsql.PrivateEndpointConnection{conn1, conn2},
					},
				},
			},
		}

		testClient := &testSQLServerPrivateEndpointConnectionsClient{
			MockSQLServerPrivateEndpointConnectionsClient: mockClient,
			pager: mockPager,
		}

		wrapper := manual.NewSQLServerPrivateEndpointConnection(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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
			if item.GetType() != azureshared.SQLServerPrivateEndpointConnection.String() {
				t.Errorf("Expected type %s, got %s", azureshared.SQLServerPrivateEndpointConnection, item.GetType())
			}
		}
	})

	t.Run("Search_NilNameSkipped", func(t *testing.T) {
		validConn := createAzureSQLServerPrivateEndpointConnection("valid-pec", "")

		mockClient := mocks.NewMockSQLServerPrivateEndpointConnectionsClient(ctrl)
		mockPager := &mockSQLServerPrivateEndpointConnectionsPager{
			pages: []armsql.PrivateEndpointConnectionsClientListByServerResponse{
				{
					PrivateEndpointConnectionListResult: armsql.PrivateEndpointConnectionListResult{
						Value: []*armsql.PrivateEndpointConnection{
							{Name: nil},
							validConn,
						},
					},
				},
			},
		}

		testClient := &testSQLServerPrivateEndpointConnectionsClient{
			MockSQLServerPrivateEndpointConnectionsClient: mockClient,
			pager: mockPager,
		}

		wrapper := manual.NewSQLServerPrivateEndpointConnection(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support Search operation")
		}

		sdpItems, err := searchable.Search(ctx, wrapper.Scopes()[0], serverName, true)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if len(sdpItems) != 1 {
			t.Fatalf("Expected 1 item (nil name skipped), got: %d", len(sdpItems))
		}
		if sdpItems[0].UniqueAttributeValue() != shared.CompositeLookupKey(serverName, "valid-pec") {
			t.Errorf("Expected unique value %s, got %s", shared.CompositeLookupKey(serverName, "valid-pec"), sdpItems[0].UniqueAttributeValue())
		}
	})

	t.Run("Search_InvalidQueryParts", func(t *testing.T) {
		mockClient := mocks.NewMockSQLServerPrivateEndpointConnectionsClient(ctrl)
		testClient := &testSQLServerPrivateEndpointConnectionsClient{MockSQLServerPrivateEndpointConnectionsClient: mockClient}

		wrapper := manual.NewSQLServerPrivateEndpointConnection(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		_, qErr := wrapper.Search(ctx, wrapper.Scopes()[0])
		if qErr == nil {
			t.Error("Expected error when providing no query parts, but got nil")
		}
	})

	t.Run("ErrorHandling_Get", func(t *testing.T) {
		expectedErr := errors.New("private endpoint connection not found")

		mockClient := mocks.NewMockSQLServerPrivateEndpointConnectionsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, serverName, "nonexistent-pec").Return(
			armsql.PrivateEndpointConnectionsClientGetResponse{}, expectedErr)

		testClient := &testSQLServerPrivateEndpointConnectionsClient{MockSQLServerPrivateEndpointConnectionsClient: mockClient}
		wrapper := manual.NewSQLServerPrivateEndpointConnection(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(serverName, "nonexistent-pec")
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr == nil {
			t.Error("Expected error when getting non-existent private endpoint connection, but got nil")
		}
	})

	t.Run("PotentialLinks", func(t *testing.T) {
		wrapper := manual.NewSQLServerPrivateEndpointConnection(nil, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		links := wrapper.PotentialLinks()
		if !links[azureshared.SQLServer] {
			t.Error("Expected SQLServer in PotentialLinks")
		}
		if !links[azureshared.NetworkPrivateEndpoint] {
			t.Error("Expected NetworkPrivateEndpoint in PotentialLinks")
		}
	})
}

func createAzureSQLServerPrivateEndpointConnection(connectionName, privateEndpointID string) *armsql.PrivateEndpointConnection {
	ready := armsql.PrivateEndpointProvisioningStateReady
	approved := armsql.PrivateLinkServiceConnectionStateStatusApproved
	conn := &armsql.PrivateEndpointConnection{
		ID:   new("/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.Sql/servers/test-sql-server/privateEndpointConnections/" + connectionName),
		Name: new(connectionName),
		Type: new("Microsoft.Sql/servers/privateEndpointConnections"),
		Properties: &armsql.PrivateEndpointConnectionProperties{
			ProvisioningState: &ready,
			PrivateLinkServiceConnectionState: &armsql.PrivateLinkServiceConnectionStateProperty{
				Status: &approved,
			},
		},
	}
	if privateEndpointID != "" {
		conn.Properties.PrivateEndpoint = &armsql.PrivateEndpointProperty{
			ID: new(privateEndpointID),
		}
	}
	return conn
}
