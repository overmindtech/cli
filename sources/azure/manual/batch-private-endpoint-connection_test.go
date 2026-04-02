package manual_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/batch/armbatch/v4"
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

type mockBatchPrivateEndpointConnectionPager struct {
	pages []armbatch.PrivateEndpointConnectionClientListByBatchAccountResponse
	index int
}

func (m *mockBatchPrivateEndpointConnectionPager) More() bool {
	return m.index < len(m.pages)
}

func (m *mockBatchPrivateEndpointConnectionPager) NextPage(ctx context.Context) (armbatch.PrivateEndpointConnectionClientListByBatchAccountResponse, error) {
	if m.index >= len(m.pages) {
		return armbatch.PrivateEndpointConnectionClientListByBatchAccountResponse{}, errors.New("no more pages")
	}
	page := m.pages[m.index]
	m.index++
	return page, nil
}

type testBatchPrivateEndpointConnectionClient struct {
	*mocks.MockBatchPrivateEndpointConnectionClient
	pager clients.BatchPrivateEndpointConnectionPager
}

func (t *testBatchPrivateEndpointConnectionClient) ListByBatchAccount(ctx context.Context, resourceGroupName, accountName string) clients.BatchPrivateEndpointConnectionPager {
	return t.pager
}

func TestBatchPrivateEndpointConnection(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"
	accountName := "test-batch-account"
	connectionName := "test-pec"

	t.Run("Get", func(t *testing.T) {
		conn := createAzureBatchPrivateEndpointConnection(connectionName, "")

		mockClient := mocks.NewMockBatchPrivateEndpointConnectionClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, accountName, connectionName).Return(
			armbatch.PrivateEndpointConnectionClientGetResponse{
				PrivateEndpointConnection: *conn,
			}, nil)

		testClient := &testBatchPrivateEndpointConnectionClient{MockBatchPrivateEndpointConnectionClient: mockClient}
		wrapper := manual.NewBatchPrivateEndpointConnection(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(accountName, connectionName)
		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.BatchBatchPrivateEndpointConnection.String() {
			t.Errorf("Expected type %s, got %s", azureshared.BatchBatchPrivateEndpointConnection, sdpItem.GetType())
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

			foundBatchAccount := false
			for _, lq := range linkedQueries {
				if lq.GetQuery().GetType() == azureshared.BatchBatchAccount.String() {
					foundBatchAccount = true
					if lq.GetQuery().GetMethod() != sdp.QueryMethod_GET {
						t.Errorf("Expected BatchAccount link method GET, got %v", lq.GetQuery().GetMethod())
					}
					if lq.GetQuery().GetQuery() != accountName {
						t.Errorf("Expected BatchAccount query %s, got %s", accountName, lq.GetQuery().GetQuery())
					}
				}
			}
			if !foundBatchAccount {
				t.Error("Expected linked query to BatchAccount")
			}
		})
	})

	t.Run("Get_WithPrivateEndpointLink", func(t *testing.T) {
		peID := "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/privateEndpoints/test-pe"
		conn := createAzureBatchPrivateEndpointConnection(connectionName, peID)

		mockClient := mocks.NewMockBatchPrivateEndpointConnectionClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, accountName, connectionName).Return(
			armbatch.PrivateEndpointConnectionClientGetResponse{
				PrivateEndpointConnection: *conn,
			}, nil)

		testClient := &testBatchPrivateEndpointConnectionClient{MockBatchPrivateEndpointConnectionClient: mockClient}
		wrapper := manual.NewBatchPrivateEndpointConnection(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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
		mockClient := mocks.NewMockBatchPrivateEndpointConnectionClient(ctrl)
		testClient := &testBatchPrivateEndpointConnectionClient{MockBatchPrivateEndpointConnectionClient: mockClient}

		wrapper := manual.NewBatchPrivateEndpointConnection(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], accountName, true)
		if qErr == nil {
			t.Error("Expected error when providing insufficient query parts, but got nil")
		}
	})

	t.Run("GetWithEmptyAccountName", func(t *testing.T) {
		mockClient := mocks.NewMockBatchPrivateEndpointConnectionClient(ctrl)
		testClient := &testBatchPrivateEndpointConnectionClient{MockBatchPrivateEndpointConnectionClient: mockClient}

		wrapper := manual.NewBatchPrivateEndpointConnection(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey("", connectionName)
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr == nil {
			t.Error("Expected error when accountName is empty, but got nil")
		}
	})

	t.Run("GetWithEmptyConnectionName", func(t *testing.T) {
		mockClient := mocks.NewMockBatchPrivateEndpointConnectionClient(ctrl)
		testClient := &testBatchPrivateEndpointConnectionClient{MockBatchPrivateEndpointConnectionClient: mockClient}

		wrapper := manual.NewBatchPrivateEndpointConnection(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(accountName, "")
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr == nil {
			t.Error("Expected error when connectionName is empty, but got nil")
		}
	})

	t.Run("Search", func(t *testing.T) {
		conn1 := createAzureBatchPrivateEndpointConnection("pec-1", "")
		conn2 := createAzureBatchPrivateEndpointConnection("pec-2", "")

		mockClient := mocks.NewMockBatchPrivateEndpointConnectionClient(ctrl)
		mockPager := &mockBatchPrivateEndpointConnectionPager{
			pages: []armbatch.PrivateEndpointConnectionClientListByBatchAccountResponse{
				{
					ListPrivateEndpointConnectionsResult: armbatch.ListPrivateEndpointConnectionsResult{
						Value: []*armbatch.PrivateEndpointConnection{conn1, conn2},
					},
				},
			},
		}

		testClient := &testBatchPrivateEndpointConnectionClient{
			MockBatchPrivateEndpointConnectionClient: mockClient,
			pager:                                    mockPager,
		}

		wrapper := manual.NewBatchPrivateEndpointConnection(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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
			if item.GetType() != azureshared.BatchBatchPrivateEndpointConnection.String() {
				t.Errorf("Expected type %s, got %s", azureshared.BatchBatchPrivateEndpointConnection, item.GetType())
			}
		}
	})

	t.Run("SearchStream", func(t *testing.T) {
		conn1 := createAzureBatchPrivateEndpointConnection("pec-1", "")
		conn2 := createAzureBatchPrivateEndpointConnection("pec-2", "")

		mockClient := mocks.NewMockBatchPrivateEndpointConnectionClient(ctrl)
		mockPager := &mockBatchPrivateEndpointConnectionPager{
			pages: []armbatch.PrivateEndpointConnectionClientListByBatchAccountResponse{
				{
					ListPrivateEndpointConnectionsResult: armbatch.ListPrivateEndpointConnectionsResult{
						Value: []*armbatch.PrivateEndpointConnection{conn1, conn2},
					},
				},
			},
		}

		testClient := &testBatchPrivateEndpointConnectionClient{
			MockBatchPrivateEndpointConnectionClient: mockClient,
			pager:                                    mockPager,
		}

		wrapper := manual.NewBatchPrivateEndpointConnection(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		searchStreamable, ok := adapter.(discovery.SearchStreamableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support SearchStream operation")
		}

		var items []*sdp.Item
		var errs []error

		mockItemHandler := func(item *sdp.Item) {
			items = append(items, item)
		}
		mockErrorHandler := func(err error) {
			errs = append(errs, err)
		}

		stream := discovery.NewQueryResultStream(mockItemHandler, mockErrorHandler)

		searchStreamable.SearchStream(ctx, wrapper.Scopes()[0], accountName, true, stream)

		if len(errs) != 0 {
			t.Fatalf("Expected no errors, got: %v", errs)
		}

		if len(items) != 2 {
			t.Fatalf("Expected 2 items, got: %d", len(items))
		}
	})

	t.Run("Search_NilNameSkipped", func(t *testing.T) {
		validConn := createAzureBatchPrivateEndpointConnection("valid-pec", "")

		mockClient := mocks.NewMockBatchPrivateEndpointConnectionClient(ctrl)
		mockPager := &mockBatchPrivateEndpointConnectionPager{
			pages: []armbatch.PrivateEndpointConnectionClientListByBatchAccountResponse{
				{
					ListPrivateEndpointConnectionsResult: armbatch.ListPrivateEndpointConnectionsResult{
						Value: []*armbatch.PrivateEndpointConnection{
							{Name: nil},
							validConn,
						},
					},
				},
			},
		}

		testClient := &testBatchPrivateEndpointConnectionClient{
			MockBatchPrivateEndpointConnectionClient: mockClient,
			pager:                                    mockPager,
		}

		wrapper := manual.NewBatchPrivateEndpointConnection(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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
		mockClient := mocks.NewMockBatchPrivateEndpointConnectionClient(ctrl)
		testClient := &testBatchPrivateEndpointConnectionClient{MockBatchPrivateEndpointConnectionClient: mockClient}

		wrapper := manual.NewBatchPrivateEndpointConnection(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		_, qErr := wrapper.Search(ctx, wrapper.Scopes()[0])
		if qErr == nil {
			t.Error("Expected error when providing no query parts, but got nil")
		}
	})

	t.Run("SearchWithEmptyAccountName", func(t *testing.T) {
		mockClient := mocks.NewMockBatchPrivateEndpointConnectionClient(ctrl)
		testClient := &testBatchPrivateEndpointConnectionClient{MockBatchPrivateEndpointConnectionClient: mockClient}

		wrapper := manual.NewBatchPrivateEndpointConnection(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		_, qErr := wrapper.Search(ctx, wrapper.Scopes()[0], "")
		if qErr == nil {
			t.Error("Expected error when accountName is empty, but got nil")
		}
	})

	t.Run("ErrorHandling_Get", func(t *testing.T) {
		expectedErr := errors.New("private endpoint connection not found")

		mockClient := mocks.NewMockBatchPrivateEndpointConnectionClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, accountName, "nonexistent-pec").Return(
			armbatch.PrivateEndpointConnectionClientGetResponse{}, expectedErr)

		testClient := &testBatchPrivateEndpointConnectionClient{MockBatchPrivateEndpointConnectionClient: mockClient}
		wrapper := manual.NewBatchPrivateEndpointConnection(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(accountName, "nonexistent-pec")
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr == nil {
			t.Error("Expected error when getting non-existent private endpoint connection, but got nil")
		}
	})

	t.Run("PotentialLinks", func(t *testing.T) {
		wrapper := manual.NewBatchPrivateEndpointConnection(nil, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		links := wrapper.PotentialLinks()
		if !links[azureshared.BatchBatchAccount] {
			t.Error("Expected BatchAccount in PotentialLinks")
		}
		if !links[azureshared.NetworkPrivateEndpoint] {
			t.Error("Expected NetworkPrivateEndpoint in PotentialLinks")
		}
	})

	t.Run("HealthMapping", func(t *testing.T) {
		tests := []struct {
			name          string
			state         armbatch.PrivateEndpointConnectionProvisioningState
			expectedHeath sdp.Health
		}{
			{"Succeeded", armbatch.PrivateEndpointConnectionProvisioningStateSucceeded, sdp.Health_HEALTH_OK},
			{"Creating", armbatch.PrivateEndpointConnectionProvisioningStateCreating, sdp.Health_HEALTH_PENDING},
			{"Updating", armbatch.PrivateEndpointConnectionProvisioningStateUpdating, sdp.Health_HEALTH_PENDING},
			{"Deleting", armbatch.PrivateEndpointConnectionProvisioningStateDeleting, sdp.Health_HEALTH_PENDING},
			{"Failed", armbatch.PrivateEndpointConnectionProvisioningStateFailed, sdp.Health_HEALTH_ERROR},
			{"Cancelled", armbatch.PrivateEndpointConnectionProvisioningStateCancelled, sdp.Health_HEALTH_ERROR},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				conn := createAzureBatchPrivateEndpointConnectionWithState(connectionName, tt.state)

				mockClient := mocks.NewMockBatchPrivateEndpointConnectionClient(ctrl)
				mockClient.EXPECT().Get(ctx, resourceGroup, accountName, connectionName).Return(
					armbatch.PrivateEndpointConnectionClientGetResponse{
						PrivateEndpointConnection: *conn,
					}, nil)

				testClient := &testBatchPrivateEndpointConnectionClient{MockBatchPrivateEndpointConnectionClient: mockClient}
				wrapper := manual.NewBatchPrivateEndpointConnection(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
				adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

				query := shared.CompositeLookupKey(accountName, connectionName)
				sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
				if qErr != nil {
					t.Fatalf("Expected no error, got: %v", qErr)
				}

				if sdpItem.GetHealth() != tt.expectedHeath {
					t.Errorf("Expected health %v, got %v", tt.expectedHeath, sdpItem.GetHealth())
				}
			})
		}
	})
}

func createAzureBatchPrivateEndpointConnection(connectionName, privateEndpointID string) *armbatch.PrivateEndpointConnection {
	succeeded := armbatch.PrivateEndpointConnectionProvisioningStateSucceeded
	conn := &armbatch.PrivateEndpointConnection{
		ID:   new("/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.Batch/batchAccounts/test-batch-account/privateEndpointConnections/" + connectionName),
		Name: new(connectionName),
		Type: new("Microsoft.Batch/batchAccounts/privateEndpointConnections"),
		Properties: &armbatch.PrivateEndpointConnectionProperties{
			ProvisioningState: &succeeded,
			PrivateLinkServiceConnectionState: &armbatch.PrivateLinkServiceConnectionState{
				Status: new(armbatch.PrivateLinkServiceConnectionStatusApproved),
			},
		},
		Tags: map[string]*string{
			"env": new("test"),
		},
	}
	if privateEndpointID != "" {
		conn.Properties.PrivateEndpoint = &armbatch.PrivateEndpoint{
			ID: new(privateEndpointID),
		}
	}
	return conn
}

func createAzureBatchPrivateEndpointConnectionWithState(connectionName string, state armbatch.PrivateEndpointConnectionProvisioningState) *armbatch.PrivateEndpointConnection {
	conn := &armbatch.PrivateEndpointConnection{
		ID:   new("/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.Batch/batchAccounts/test-batch-account/privateEndpointConnections/" + connectionName),
		Name: new(connectionName),
		Type: new("Microsoft.Batch/batchAccounts/privateEndpointConnections"),
		Properties: &armbatch.PrivateEndpointConnectionProperties{
			ProvisioningState: &state,
		},
		Tags: map[string]*string{
			"env": new("test"),
		},
	}
	return conn
}
