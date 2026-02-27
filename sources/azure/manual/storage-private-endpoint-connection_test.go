package manual_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage/v3"
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

type mockPrivateEndpointConnectionsPager struct {
	pages []armstorage.PrivateEndpointConnectionsClientListResponse
	index int
}

func (m *mockPrivateEndpointConnectionsPager) More() bool {
	return m.index < len(m.pages)
}

func (m *mockPrivateEndpointConnectionsPager) NextPage(ctx context.Context) (armstorage.PrivateEndpointConnectionsClientListResponse, error) {
	if m.index >= len(m.pages) {
		return armstorage.PrivateEndpointConnectionsClientListResponse{}, errors.New("no more pages")
	}
	page := m.pages[m.index]
	m.index++
	return page, nil
}

type testStoragePrivateEndpointConnectionsClient struct {
	*mocks.MockStoragePrivateEndpointConnectionsClient
	pager clients.PrivateEndpointConnectionsPager
}

func (t *testStoragePrivateEndpointConnectionsClient) List(ctx context.Context, resourceGroupName, accountName string) clients.PrivateEndpointConnectionsPager {
	return t.pager
}

func TestStoragePrivateEndpointConnection(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"
	accountName := "teststorageaccount"
	connectionName := "test-pec"

	t.Run("Get", func(t *testing.T) {
		conn := createAzureStoragePrivateEndpointConnection(connectionName, "")

		mockClient := mocks.NewMockStoragePrivateEndpointConnectionsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, accountName, connectionName).Return(
			armstorage.PrivateEndpointConnectionsClientGetResponse{
				PrivateEndpointConnection: *conn,
			}, nil)

		testClient := &testStoragePrivateEndpointConnectionsClient{MockStoragePrivateEndpointConnectionsClient: mockClient}
		wrapper := manual.NewStoragePrivateEndpointConnection(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(accountName, connectionName)
		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.StoragePrivateEndpointConnection.String() {
			t.Errorf("Expected type %s, got %s", azureshared.StoragePrivateEndpointConnection, sdpItem.GetType())
		}

		if sdpItem.GetUniqueAttribute() != "uniqueAttr" {
			t.Errorf("Expected unique attribute 'uniqueAttr', got %s", sdpItem.GetUniqueAttribute())
		}

		if sdpItem.UniqueAttributeValue() != shared.CompositeLookupKey(accountName, connectionName) {
			t.Errorf("Expected unique attribute value %s, got %s", shared.CompositeLookupKey(accountName, connectionName), sdpItem.UniqueAttributeValue())
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

			foundStorageAccount := false
			for _, lq := range linkedQueries {
				if lq.GetQuery().GetType() == azureshared.StorageAccount.String() {
					foundStorageAccount = true
					if lq.GetQuery().GetMethod() != sdp.QueryMethod_GET {
						t.Errorf("Expected StorageAccount link method GET, got %v", lq.GetQuery().GetMethod())
					}
					if lq.GetQuery().GetQuery() != accountName {
						t.Errorf("Expected StorageAccount query %s, got %s", accountName, lq.GetQuery().GetQuery())
					}
				}
			}
			if !foundStorageAccount {
				t.Error("Expected linked query to StorageAccount")
			}
		})
	})

	t.Run("Get_WithPrivateEndpointLink", func(t *testing.T) {
		peID := "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/privateEndpoints/test-pe"
		conn := createAzureStoragePrivateEndpointConnection(connectionName, peID)

		mockClient := mocks.NewMockStoragePrivateEndpointConnectionsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, accountName, connectionName).Return(
			armstorage.PrivateEndpointConnectionsClientGetResponse{
				PrivateEndpointConnection: *conn,
			}, nil)

		testClient := &testStoragePrivateEndpointConnectionsClient{MockStoragePrivateEndpointConnectionsClient: mockClient}
		wrapper := manual.NewStoragePrivateEndpointConnection(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(accountName, connectionName)
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
		mockClient := mocks.NewMockStoragePrivateEndpointConnectionsClient(ctrl)
		testClient := &testStoragePrivateEndpointConnectionsClient{MockStoragePrivateEndpointConnectionsClient: mockClient}

		wrapper := manual.NewStoragePrivateEndpointConnection(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], accountName, true)
		if qErr == nil {
			t.Error("Expected error when providing insufficient query parts, but got nil")
		}
	})

	t.Run("Search", func(t *testing.T) {
		conn1 := createAzureStoragePrivateEndpointConnection("pec-1", "")
		conn2 := createAzureStoragePrivateEndpointConnection("pec-2", "")

		mockClient := mocks.NewMockStoragePrivateEndpointConnectionsClient(ctrl)
		mockPager := &mockPrivateEndpointConnectionsPager{
			pages: []armstorage.PrivateEndpointConnectionsClientListResponse{
				{
					PrivateEndpointConnectionListResult: armstorage.PrivateEndpointConnectionListResult{
						Value: []*armstorage.PrivateEndpointConnection{conn1, conn2},
					},
				},
			},
		}

		testClient := &testStoragePrivateEndpointConnectionsClient{
			MockStoragePrivateEndpointConnectionsClient: mockClient,
			pager: mockPager,
		}

		wrapper := manual.NewStoragePrivateEndpointConnection(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support Search operation")
		}

		sdpItems, err := searchable.Search(ctx, wrapper.Scopes()[0], accountName, true)
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
			if item.GetType() != azureshared.StoragePrivateEndpointConnection.String() {
				t.Errorf("Expected type %s, got %s", azureshared.StoragePrivateEndpointConnection, item.GetType())
			}
		}
	})

	t.Run("Search_NilNameSkipped", func(t *testing.T) {
		validConn := createAzureStoragePrivateEndpointConnection("valid-pec", "")

		mockClient := mocks.NewMockStoragePrivateEndpointConnectionsClient(ctrl)
		mockPager := &mockPrivateEndpointConnectionsPager{
			pages: []armstorage.PrivateEndpointConnectionsClientListResponse{
				{
					PrivateEndpointConnectionListResult: armstorage.PrivateEndpointConnectionListResult{
						Value: []*armstorage.PrivateEndpointConnection{
							{Name: nil},
							validConn,
						},
					},
				},
			},
		}

		testClient := &testStoragePrivateEndpointConnectionsClient{
			MockStoragePrivateEndpointConnectionsClient: mockClient,
			pager: mockPager,
		}

		wrapper := manual.NewStoragePrivateEndpointConnection(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support Search operation")
		}

		sdpItems, err := searchable.Search(ctx, wrapper.Scopes()[0], accountName, true)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if len(sdpItems) != 1 {
			t.Fatalf("Expected 1 item (nil name skipped), got: %d", len(sdpItems))
		}
		if sdpItems[0].UniqueAttributeValue() != shared.CompositeLookupKey(accountName, "valid-pec") {
			t.Errorf("Expected unique value %s, got %s", shared.CompositeLookupKey(accountName, "valid-pec"), sdpItems[0].UniqueAttributeValue())
		}
	})

	t.Run("Search_InvalidQueryParts", func(t *testing.T) {
		mockClient := mocks.NewMockStoragePrivateEndpointConnectionsClient(ctrl)
		testClient := &testStoragePrivateEndpointConnectionsClient{MockStoragePrivateEndpointConnectionsClient: mockClient}

		wrapper := manual.NewStoragePrivateEndpointConnection(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		_, qErr := wrapper.Search(ctx, wrapper.Scopes()[0])
		if qErr == nil {
			t.Error("Expected error when providing no query parts, but got nil")
		}
	})

	t.Run("ErrorHandling_Get", func(t *testing.T) {
		expectedErr := errors.New("private endpoint connection not found")

		mockClient := mocks.NewMockStoragePrivateEndpointConnectionsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, accountName, "nonexistent-pec").Return(
			armstorage.PrivateEndpointConnectionsClientGetResponse{}, expectedErr)

		testClient := &testStoragePrivateEndpointConnectionsClient{MockStoragePrivateEndpointConnectionsClient: mockClient}
		wrapper := manual.NewStoragePrivateEndpointConnection(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(accountName, "nonexistent-pec")
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr == nil {
			t.Error("Expected error when getting non-existent private endpoint connection, but got nil")
		}
	})

	t.Run("PotentialLinks", func(t *testing.T) {
		wrapper := manual.NewStoragePrivateEndpointConnection(nil, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		links := wrapper.PotentialLinks()
		if !links[azureshared.StorageAccount] {
			t.Error("Expected StorageAccount in PotentialLinks")
		}
		if !links[azureshared.NetworkPrivateEndpoint] {
			t.Error("Expected NetworkPrivateEndpoint in PotentialLinks")
		}
	})
}

func createAzureStoragePrivateEndpointConnection(connectionName, privateEndpointID string) *armstorage.PrivateEndpointConnection {
	conn := &armstorage.PrivateEndpointConnection{
		ID:   new("/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.Storage/storageAccounts/teststorageaccount/privateEndpointConnections/" + connectionName),
		Name: new(connectionName),
		Type: new("Microsoft.Storage/storageAccounts/privateEndpointConnections"),
		Properties: &armstorage.PrivateEndpointConnectionProperties{
			ProvisioningState: to.Ptr(armstorage.PrivateEndpointConnectionProvisioningStateSucceeded),
			PrivateLinkServiceConnectionState: &armstorage.PrivateLinkServiceConnectionState{
				Status: to.Ptr(armstorage.PrivateEndpointServiceConnectionStatusApproved),
			},
		},
	}
	if privateEndpointID != "" {
		conn.Properties.PrivateEndpoint = &armstorage.PrivateEndpoint{
			ID: new(privateEndpointID),
		}
	}
	return conn
}
