package manual_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/keyvault/armkeyvault/v2"
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

type mockKeyVaultManagedHSMPrivateEndpointConnectionsPager struct {
	pages []armkeyvault.MHSMPrivateEndpointConnectionsClientListByResourceResponse
	index int
}

func (m *mockKeyVaultManagedHSMPrivateEndpointConnectionsPager) More() bool {
	return m.index < len(m.pages)
}

func (m *mockKeyVaultManagedHSMPrivateEndpointConnectionsPager) NextPage(ctx context.Context) (armkeyvault.MHSMPrivateEndpointConnectionsClientListByResourceResponse, error) {
	if m.index >= len(m.pages) {
		return armkeyvault.MHSMPrivateEndpointConnectionsClientListByResourceResponse{}, errors.New("no more pages")
	}
	page := m.pages[m.index]
	m.index++
	return page, nil
}

type testKeyVaultManagedHSMPrivateEndpointConnectionsClient struct {
	*mocks.MockKeyVaultManagedHSMPrivateEndpointConnectionsClient
	pager clients.KeyVaultManagedHSMPrivateEndpointConnectionsPager
}

func (t *testKeyVaultManagedHSMPrivateEndpointConnectionsClient) ListByResource(ctx context.Context, resourceGroupName, hsmName string) clients.KeyVaultManagedHSMPrivateEndpointConnectionsPager {
	return t.pager
}

func TestKeyVaultManagedHSMPrivateEndpointConnection(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"
	hsmName := "test-hsm"
	connectionName := "test-pec"

	t.Run("Get", func(t *testing.T) {
		conn := createAzureMHSMPrivateEndpointConnection(connectionName, "")

		mockClient := mocks.NewMockKeyVaultManagedHSMPrivateEndpointConnectionsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, hsmName, connectionName).Return(
			armkeyvault.MHSMPrivateEndpointConnectionsClientGetResponse{
				MHSMPrivateEndpointConnection: *conn,
			}, nil)

		testClient := &testKeyVaultManagedHSMPrivateEndpointConnectionsClient{MockKeyVaultManagedHSMPrivateEndpointConnectionsClient: mockClient}
		wrapper := manual.NewKeyVaultManagedHSMPrivateEndpointConnection(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(hsmName, connectionName)
		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.KeyVaultManagedHSMPrivateEndpointConnection.String() {
			t.Errorf("Expected type %s, got %s", azureshared.KeyVaultManagedHSMPrivateEndpointConnection, sdpItem.GetType())
		}

		if sdpItem.GetUniqueAttribute() != "uniqueAttr" {
			t.Errorf("Expected unique attribute 'uniqueAttr', got %s", sdpItem.GetUniqueAttribute())
		}

		if sdpItem.UniqueAttributeValue() != shared.CompositeLookupKey(hsmName, connectionName) {
			t.Errorf("Expected unique attribute value %s, got %s", shared.CompositeLookupKey(hsmName, connectionName), sdpItem.UniqueAttributeValue())
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

			foundKeyVaultManagedHSM := false
			for _, lq := range linkedQueries {
				if lq.GetQuery().GetType() == azureshared.KeyVaultManagedHSM.String() {
					foundKeyVaultManagedHSM = true
					if lq.GetQuery().GetMethod() != sdp.QueryMethod_GET {
						t.Errorf("Expected KeyVaultManagedHSM link method GET, got %v", lq.GetQuery().GetMethod())
					}
					if lq.GetQuery().GetQuery() != hsmName {
						t.Errorf("Expected KeyVaultManagedHSM query %s, got %s", hsmName, lq.GetQuery().GetQuery())
					}
				}
			}
			if !foundKeyVaultManagedHSM {
				t.Error("Expected linked query to KeyVaultManagedHSM")
			}
		})
	})

	t.Run("Get_WithPrivateEndpointLink", func(t *testing.T) {
		peID := "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/privateEndpoints/test-pe"
		conn := createAzureMHSMPrivateEndpointConnection(connectionName, peID)

		mockClient := mocks.NewMockKeyVaultManagedHSMPrivateEndpointConnectionsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, hsmName, connectionName).Return(
			armkeyvault.MHSMPrivateEndpointConnectionsClientGetResponse{
				MHSMPrivateEndpointConnection: *conn,
			}, nil)

		testClient := &testKeyVaultManagedHSMPrivateEndpointConnectionsClient{MockKeyVaultManagedHSMPrivateEndpointConnectionsClient: mockClient}
		wrapper := manual.NewKeyVaultManagedHSMPrivateEndpointConnection(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(hsmName, connectionName)
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
		mockClient := mocks.NewMockKeyVaultManagedHSMPrivateEndpointConnectionsClient(ctrl)
		testClient := &testKeyVaultManagedHSMPrivateEndpointConnectionsClient{MockKeyVaultManagedHSMPrivateEndpointConnectionsClient: mockClient}

		wrapper := manual.NewKeyVaultManagedHSMPrivateEndpointConnection(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], hsmName, true)
		if qErr == nil {
			t.Error("Expected error when providing insufficient query parts, but got nil")
		}
	})

	t.Run("Search", func(t *testing.T) {
		conn1 := createAzureMHSMPrivateEndpointConnection("pec-1", "")
		conn2 := createAzureMHSMPrivateEndpointConnection("pec-2", "")

		mockClient := mocks.NewMockKeyVaultManagedHSMPrivateEndpointConnectionsClient(ctrl)
		mockPager := &mockKeyVaultManagedHSMPrivateEndpointConnectionsPager{
			pages: []armkeyvault.MHSMPrivateEndpointConnectionsClientListByResourceResponse{
				{
					MHSMPrivateEndpointConnectionsListResult: armkeyvault.MHSMPrivateEndpointConnectionsListResult{
						Value: []*armkeyvault.MHSMPrivateEndpointConnection{conn1, conn2},
					},
				},
			},
		}

		testClient := &testKeyVaultManagedHSMPrivateEndpointConnectionsClient{
			MockKeyVaultManagedHSMPrivateEndpointConnectionsClient: mockClient,
			pager: mockPager,
		}

		wrapper := manual.NewKeyVaultManagedHSMPrivateEndpointConnection(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support Search operation")
		}

		sdpItems, err := searchable.Search(ctx, wrapper.Scopes()[0], hsmName, true)
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
			if item.GetType() != azureshared.KeyVaultManagedHSMPrivateEndpointConnection.String() {
				t.Errorf("Expected type %s, got %s", azureshared.KeyVaultManagedHSMPrivateEndpointConnection, item.GetType())
			}
		}
	})

	t.Run("Search_NilNameSkipped", func(t *testing.T) {
		validConn := createAzureMHSMPrivateEndpointConnection("valid-pec", "")

		mockClient := mocks.NewMockKeyVaultManagedHSMPrivateEndpointConnectionsClient(ctrl)
		mockPager := &mockKeyVaultManagedHSMPrivateEndpointConnectionsPager{
			pages: []armkeyvault.MHSMPrivateEndpointConnectionsClientListByResourceResponse{
				{
					MHSMPrivateEndpointConnectionsListResult: armkeyvault.MHSMPrivateEndpointConnectionsListResult{
						Value: []*armkeyvault.MHSMPrivateEndpointConnection{
							{Name: nil},
							validConn,
						},
					},
				},
			},
		}

		testClient := &testKeyVaultManagedHSMPrivateEndpointConnectionsClient{
			MockKeyVaultManagedHSMPrivateEndpointConnectionsClient: mockClient,
			pager: mockPager,
		}

		wrapper := manual.NewKeyVaultManagedHSMPrivateEndpointConnection(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support Search operation")
		}

		sdpItems, err := searchable.Search(ctx, wrapper.Scopes()[0], hsmName, true)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if len(sdpItems) != 1 {
			t.Fatalf("Expected 1 item (nil name skipped), got: %d", len(sdpItems))
		}
		if sdpItems[0].UniqueAttributeValue() != shared.CompositeLookupKey(hsmName, "valid-pec") {
			t.Errorf("Expected unique value %s, got %s", shared.CompositeLookupKey(hsmName, "valid-pec"), sdpItems[0].UniqueAttributeValue())
		}
	})

	t.Run("Search_InvalidQueryParts", func(t *testing.T) {
		mockClient := mocks.NewMockKeyVaultManagedHSMPrivateEndpointConnectionsClient(ctrl)
		testClient := &testKeyVaultManagedHSMPrivateEndpointConnectionsClient{MockKeyVaultManagedHSMPrivateEndpointConnectionsClient: mockClient}

		wrapper := manual.NewKeyVaultManagedHSMPrivateEndpointConnection(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		_, qErr := wrapper.Search(ctx, wrapper.Scopes()[0])
		if qErr == nil {
			t.Error("Expected error when providing no query parts, but got nil")
		}
	})

	t.Run("ErrorHandling_Get", func(t *testing.T) {
		expectedErr := errors.New("private endpoint connection not found")

		mockClient := mocks.NewMockKeyVaultManagedHSMPrivateEndpointConnectionsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, hsmName, "nonexistent-pec").Return(
			armkeyvault.MHSMPrivateEndpointConnectionsClientGetResponse{}, expectedErr)

		testClient := &testKeyVaultManagedHSMPrivateEndpointConnectionsClient{MockKeyVaultManagedHSMPrivateEndpointConnectionsClient: mockClient}
		wrapper := manual.NewKeyVaultManagedHSMPrivateEndpointConnection(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(hsmName, "nonexistent-pec")
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr == nil {
			t.Error("Expected error when getting non-existent private endpoint connection, but got nil")
		}
	})

	t.Run("Get_WithUserAssignedIdentityLink", func(t *testing.T) {
		identityID := "/subscriptions/" + subscriptionID + "/resourceGroups/identity-rg/providers/Microsoft.ManagedIdentity/userAssignedIdentities/test-identity"
		conn := createAzureMHSMPrivateEndpointConnectionWithIdentity(connectionName, "", identityID)

		mockClient := mocks.NewMockKeyVaultManagedHSMPrivateEndpointConnectionsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, hsmName, connectionName).Return(
			armkeyvault.MHSMPrivateEndpointConnectionsClientGetResponse{
				MHSMPrivateEndpointConnection: *conn,
			}, nil)

		testClient := &testKeyVaultManagedHSMPrivateEndpointConnectionsClient{MockKeyVaultManagedHSMPrivateEndpointConnectionsClient: mockClient}
		wrapper := manual.NewKeyVaultManagedHSMPrivateEndpointConnection(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(hsmName, connectionName)
		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		foundIdentity := false
		for _, lq := range sdpItem.GetLinkedItemQueries() {
			if lq.GetQuery().GetType() == azureshared.ManagedIdentityUserAssignedIdentity.String() {
				foundIdentity = true
				if lq.GetQuery().GetQuery() != "test-identity" {
					t.Errorf("Expected ManagedIdentityUserAssignedIdentity query 'test-identity', got %s", lq.GetQuery().GetQuery())
				}
				if lq.GetQuery().GetScope() != subscriptionID+".identity-rg" {
					t.Errorf("Expected scope %s.identity-rg for identity in different RG, got %s", subscriptionID, lq.GetQuery().GetScope())
				}
			}
		}
		if !foundIdentity {
			t.Error("Expected linked query to ManagedIdentityUserAssignedIdentity when Identity.UserAssignedIdentities is set")
		}
	})

	t.Run("PotentialLinks", func(t *testing.T) {
		wrapper := manual.NewKeyVaultManagedHSMPrivateEndpointConnection(nil, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		links := wrapper.PotentialLinks()
		if !links[azureshared.KeyVaultManagedHSM] {
			t.Error("Expected KeyVaultManagedHSM in PotentialLinks")
		}
		if !links[azureshared.NetworkPrivateEndpoint] {
			t.Error("Expected NetworkPrivateEndpoint in PotentialLinks")
		}
		if !links[azureshared.ManagedIdentityUserAssignedIdentity] {
			t.Error("Expected ManagedIdentityUserAssignedIdentity in PotentialLinks")
		}
	})
}

func createAzureMHSMPrivateEndpointConnection(connectionName, privateEndpointID string) *armkeyvault.MHSMPrivateEndpointConnection {
	return createAzureMHSMPrivateEndpointConnectionWithIdentity(connectionName, privateEndpointID, "")
}

func createAzureMHSMPrivateEndpointConnectionWithIdentity(connectionName, privateEndpointID, identityResourceID string) *armkeyvault.MHSMPrivateEndpointConnection {
	state := armkeyvault.PrivateEndpointConnectionProvisioningStateSucceeded
	conn := &armkeyvault.MHSMPrivateEndpointConnection{
		ID:   new("/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.KeyVault/managedHSMs/test-hsm/privateEndpointConnections/" + connectionName),
		Name: new(connectionName),
		Type: new("Microsoft.KeyVault/managedHSMs/privateEndpointConnections"),
		Properties: &armkeyvault.MHSMPrivateEndpointConnectionProperties{
			ProvisioningState: &state,
		},
	}
	if privateEndpointID != "" {
		conn.Properties.PrivateEndpoint = &armkeyvault.MHSMPrivateEndpoint{
			ID: new(privateEndpointID),
		}
	}
	if identityResourceID != "" {
		conn.Identity = &armkeyvault.ManagedServiceIdentity{
			Type: new(armkeyvault.ManagedServiceIdentityTypeUserAssigned),
			UserAssignedIdentities: map[string]*armkeyvault.UserAssignedIdentity{
				identityResourceID: {},
			},
		}
	}
	return conn
}
