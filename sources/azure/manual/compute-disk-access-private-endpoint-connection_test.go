package manual_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"
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

type mockComputeDiskAccessPrivateEndpointConnectionsPager struct {
	pages []armcompute.DiskAccessesClientListPrivateEndpointConnectionsResponse
	index int
}

func (m *mockComputeDiskAccessPrivateEndpointConnectionsPager) More() bool {
	return m.index < len(m.pages)
}

func (m *mockComputeDiskAccessPrivateEndpointConnectionsPager) NextPage(ctx context.Context) (armcompute.DiskAccessesClientListPrivateEndpointConnectionsResponse, error) {
	if m.index >= len(m.pages) {
		return armcompute.DiskAccessesClientListPrivateEndpointConnectionsResponse{}, errors.New("no more pages")
	}
	page := m.pages[m.index]
	m.index++
	return page, nil
}

type testComputeDiskAccessPrivateEndpointConnectionsClient struct {
	*mocks.MockComputeDiskAccessPrivateEndpointConnectionsClient
	pager clients.ComputeDiskAccessPrivateEndpointConnectionsPager
}

func (t *testComputeDiskAccessPrivateEndpointConnectionsClient) NewListPrivateEndpointConnectionsPager(resourceGroupName string, diskAccessName string, options *armcompute.DiskAccessesClientListPrivateEndpointConnectionsOptions) clients.ComputeDiskAccessPrivateEndpointConnectionsPager {
	return t.pager
}

func TestComputeDiskAccessPrivateEndpointConnection(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"
	diskAccessName := "test-disk-access"
	connectionName := "test-pec"

	t.Run("Get", func(t *testing.T) {
		conn := createAzureComputeDiskAccessPrivateEndpointConnection(connectionName, "")

		mockClient := mocks.NewMockComputeDiskAccessPrivateEndpointConnectionsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, diskAccessName, connectionName).Return(
			armcompute.DiskAccessesClientGetAPrivateEndpointConnectionResponse{
				PrivateEndpointConnection: *conn,
			}, nil)

		testClient := &testComputeDiskAccessPrivateEndpointConnectionsClient{MockComputeDiskAccessPrivateEndpointConnectionsClient: mockClient}
		wrapper := manual.NewComputeDiskAccessPrivateEndpointConnection(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(diskAccessName, connectionName)
		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.ComputeDiskAccessPrivateEndpointConnection.String() {
			t.Errorf("Expected type %s, got %s", azureshared.ComputeDiskAccessPrivateEndpointConnection.String(), sdpItem.GetType())
		}

		if sdpItem.GetUniqueAttribute() != "uniqueAttr" {
			t.Errorf("Expected unique attribute 'uniqueAttr', got %s", sdpItem.GetUniqueAttribute())
		}

		if sdpItem.UniqueAttributeValue() != shared.CompositeLookupKey(diskAccessName, connectionName) {
			t.Errorf("Expected unique attribute value %s, got %s", shared.CompositeLookupKey(diskAccessName, connectionName), sdpItem.UniqueAttributeValue())
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

			foundDiskAccess := false
			for _, lq := range linkedQueries {
				if lq.GetQuery().GetType() == azureshared.ComputeDiskAccess.String() {
					foundDiskAccess = true
					if lq.GetQuery().GetMethod() != sdp.QueryMethod_GET {
						t.Errorf("Expected ComputeDiskAccess link method GET, got %v", lq.GetQuery().GetMethod())
					}
					if lq.GetQuery().GetQuery() != diskAccessName {
						t.Errorf("Expected ComputeDiskAccess query %s, got %s", diskAccessName, lq.GetQuery().GetQuery())
					}
				}
			}
			if !foundDiskAccess {
				t.Error("Expected linked query to ComputeDiskAccess")
			}
		})
	})

	t.Run("Get_WithPrivateEndpointLink", func(t *testing.T) {
		peID := "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/privateEndpoints/test-pe"
		conn := createAzureComputeDiskAccessPrivateEndpointConnection(connectionName, peID)

		mockClient := mocks.NewMockComputeDiskAccessPrivateEndpointConnectionsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, diskAccessName, connectionName).Return(
			armcompute.DiskAccessesClientGetAPrivateEndpointConnectionResponse{
				PrivateEndpointConnection: *conn,
			}, nil)

		testClient := &testComputeDiskAccessPrivateEndpointConnectionsClient{MockComputeDiskAccessPrivateEndpointConnectionsClient: mockClient}
		wrapper := manual.NewComputeDiskAccessPrivateEndpointConnection(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(diskAccessName, connectionName)
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
		mockClient := mocks.NewMockComputeDiskAccessPrivateEndpointConnectionsClient(ctrl)
		testClient := &testComputeDiskAccessPrivateEndpointConnectionsClient{MockComputeDiskAccessPrivateEndpointConnectionsClient: mockClient}

		wrapper := manual.NewComputeDiskAccessPrivateEndpointConnection(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], diskAccessName, true)
		if qErr == nil {
			t.Error("Expected error when providing insufficient query parts, but got nil")
		}
	})

	t.Run("Search", func(t *testing.T) {
		conn1 := createAzureComputeDiskAccessPrivateEndpointConnection("pec-1", "")
		conn2 := createAzureComputeDiskAccessPrivateEndpointConnection("pec-2", "")

		mockClient := mocks.NewMockComputeDiskAccessPrivateEndpointConnectionsClient(ctrl)
		mockPager := &mockComputeDiskAccessPrivateEndpointConnectionsPager{
			pages: []armcompute.DiskAccessesClientListPrivateEndpointConnectionsResponse{
				{
					PrivateEndpointConnectionListResult: armcompute.PrivateEndpointConnectionListResult{
						Value: []*armcompute.PrivateEndpointConnection{conn1, conn2},
					},
				},
			},
		}

		testClient := &testComputeDiskAccessPrivateEndpointConnectionsClient{
			MockComputeDiskAccessPrivateEndpointConnectionsClient: mockClient,
			pager: mockPager,
		}

		wrapper := manual.NewComputeDiskAccessPrivateEndpointConnection(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support Search operation")
		}

		sdpItems, err := searchable.Search(ctx, wrapper.Scopes()[0], diskAccessName, true)
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
			if item.GetType() != azureshared.ComputeDiskAccessPrivateEndpointConnection.String() {
				t.Errorf("Expected type %s, got %s", azureshared.ComputeDiskAccessPrivateEndpointConnection.String(), item.GetType())
			}
		}
	})

	t.Run("Search_NilNameSkipped", func(t *testing.T) {
		validConn := createAzureComputeDiskAccessPrivateEndpointConnection("valid-pec", "")

		mockClient := mocks.NewMockComputeDiskAccessPrivateEndpointConnectionsClient(ctrl)
		mockPager := &mockComputeDiskAccessPrivateEndpointConnectionsPager{
			pages: []armcompute.DiskAccessesClientListPrivateEndpointConnectionsResponse{
				{
					PrivateEndpointConnectionListResult: armcompute.PrivateEndpointConnectionListResult{
						Value: []*armcompute.PrivateEndpointConnection{
							{Name: nil},
							validConn,
						},
					},
				},
			},
		}

		testClient := &testComputeDiskAccessPrivateEndpointConnectionsClient{
			MockComputeDiskAccessPrivateEndpointConnectionsClient: mockClient,
			pager: mockPager,
		}

		wrapper := manual.NewComputeDiskAccessPrivateEndpointConnection(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support Search operation")
		}

		sdpItems, err := searchable.Search(ctx, wrapper.Scopes()[0], diskAccessName, true)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if len(sdpItems) != 1 {
			t.Fatalf("Expected 1 item (nil name skipped), got: %d", len(sdpItems))
		}
		if sdpItems[0].UniqueAttributeValue() != shared.CompositeLookupKey(diskAccessName, "valid-pec") {
			t.Errorf("Expected unique value %s, got %s", shared.CompositeLookupKey(diskAccessName, "valid-pec"), sdpItems[0].UniqueAttributeValue())
		}
	})

	t.Run("Search_InvalidQueryParts", func(t *testing.T) {
		mockClient := mocks.NewMockComputeDiskAccessPrivateEndpointConnectionsClient(ctrl)
		testClient := &testComputeDiskAccessPrivateEndpointConnectionsClient{MockComputeDiskAccessPrivateEndpointConnectionsClient: mockClient}

		wrapper := manual.NewComputeDiskAccessPrivateEndpointConnection(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		_, qErr := wrapper.Search(ctx, wrapper.Scopes()[0])
		if qErr == nil {
			t.Error("Expected error when providing no query parts, but got nil")
		}
	})

	t.Run("ErrorHandling_Get", func(t *testing.T) {
		expectedErr := errors.New("private endpoint connection not found")

		mockClient := mocks.NewMockComputeDiskAccessPrivateEndpointConnectionsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, diskAccessName, "nonexistent-pec").Return(
			armcompute.DiskAccessesClientGetAPrivateEndpointConnectionResponse{}, expectedErr)

		testClient := &testComputeDiskAccessPrivateEndpointConnectionsClient{MockComputeDiskAccessPrivateEndpointConnectionsClient: mockClient}
		wrapper := manual.NewComputeDiskAccessPrivateEndpointConnection(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(diskAccessName, "nonexistent-pec")
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr == nil {
			t.Error("Expected error when getting non-existent private endpoint connection, but got nil")
		}
	})

	t.Run("PotentialLinks", func(t *testing.T) {
		wrapper := manual.NewComputeDiskAccessPrivateEndpointConnection(nil, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		links := wrapper.PotentialLinks()
		if !links[azureshared.ComputeDiskAccess] {
			t.Error("Expected ComputeDiskAccess in PotentialLinks")
		}
		if !links[azureshared.NetworkPrivateEndpoint] {
			t.Error("Expected NetworkPrivateEndpoint in PotentialLinks")
		}
	})
}

func createAzureComputeDiskAccessPrivateEndpointConnection(connectionName, privateEndpointID string) *armcompute.PrivateEndpointConnection {
	state := armcompute.PrivateEndpointConnectionProvisioningStateSucceeded
	status := armcompute.PrivateEndpointServiceConnectionStatusApproved
	conn := &armcompute.PrivateEndpointConnection{
		ID:   new("/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.Compute/diskAccesses/test-disk-access/privateEndpointConnections/" + connectionName),
		Name: new(connectionName),
		Type: new("Microsoft.Compute/diskAccesses/privateEndpointConnections"),
		Properties: &armcompute.PrivateEndpointConnectionProperties{
			ProvisioningState: &state,
			PrivateLinkServiceConnectionState: &armcompute.PrivateLinkServiceConnectionState{
				Status: &status,
			},
		},
	}
	if privateEndpointID != "" {
		conn.Properties.PrivateEndpoint = &armcompute.PrivateEndpoint{
			ID: new(privateEndpointID),
		}
	}
	return conn
}
