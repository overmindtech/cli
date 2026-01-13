package manual_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/cosmos/armcosmos"
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

// mockDocumentDBDatabaseAccountsPager is a simple mock implementation of DocumentDBDatabaseAccountsPager
type mockDocumentDBDatabaseAccountsPager struct {
	pages []armcosmos.DatabaseAccountsClientListByResourceGroupResponse
	index int
}

func (m *mockDocumentDBDatabaseAccountsPager) More() bool {
	return m.index < len(m.pages)
}

func (m *mockDocumentDBDatabaseAccountsPager) NextPage(ctx context.Context) (armcosmos.DatabaseAccountsClientListByResourceGroupResponse, error) {
	if m.index >= len(m.pages) {
		return armcosmos.DatabaseAccountsClientListByResourceGroupResponse{}, errors.New("no more pages")
	}
	page := m.pages[m.index]
	m.index++
	return page, nil
}

// errorDocumentDBDatabaseAccountsPager is a mock pager that always returns an error
type errorDocumentDBDatabaseAccountsPager struct{}

func (e *errorDocumentDBDatabaseAccountsPager) More() bool {
	return true // Always return true so NextPage will be called
}

func (e *errorDocumentDBDatabaseAccountsPager) NextPage(ctx context.Context) (armcosmos.DatabaseAccountsClientListByResourceGroupResponse, error) {
	return armcosmos.DatabaseAccountsClientListByResourceGroupResponse{}, errors.New("pager error")
}

// testDocumentDBDatabaseAccountsClient wraps the mock to implement the correct interface
type testDocumentDBDatabaseAccountsClient struct {
	*mocks.MockDocumentDBDatabaseAccountsClient
	pager clients.DocumentDBDatabaseAccountsPager
}

func (t *testDocumentDBDatabaseAccountsClient) ListByResourceGroup(resourceGroupName string) clients.DocumentDBDatabaseAccountsPager {
	// Call the mock to satisfy expectations
	t.MockDocumentDBDatabaseAccountsClient.ListByResourceGroup(resourceGroupName)
	return t.pager
}

func TestDocumentDBDatabaseAccounts(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"
	accountName := "test-cosmos-account"

	t.Run("Get", func(t *testing.T) {
		account := createAzureCosmosDBAccount(accountName, "Succeeded", subscriptionID, resourceGroup)

		mockClient := mocks.NewMockDocumentDBDatabaseAccountsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, accountName).Return(
			armcosmos.DatabaseAccountsClientGetResponse{
				DatabaseAccountGetResults: *account,
			}, nil)

		wrapper := manual.NewDocumentDBDatabaseAccounts(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], accountName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.DocumentDBDatabaseAccounts.String() {
			t.Errorf("Expected type %s, got %s", azureshared.DocumentDBDatabaseAccounts, sdpItem.GetType())
		}

		if sdpItem.GetUniqueAttribute() != "name" {
			t.Errorf("Expected unique attribute 'name', got %s", sdpItem.GetUniqueAttribute())
		}

		if sdpItem.UniqueAttributeValue() != accountName {
			t.Errorf("Expected unique attribute value %s, got %s", accountName, sdpItem.UniqueAttributeValue())
		}

		if sdpItem.GetTags()["env"] != "test" {
			t.Errorf("Expected tag 'env=test', got: %v", sdpItem.GetTags()["env"])
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					// Private Endpoint Connection (SEARCH)
					ExpectedType:   azureshared.DocumentDBPrivateEndpointConnection.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  accountName,
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					// Private Endpoint (GET) - same resource group
					ExpectedType:   azureshared.NetworkPrivateEndpoint.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-private-endpoint",
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					// Private Endpoint (GET) - different resource group
					ExpectedType:   azureshared.NetworkPrivateEndpoint.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-private-endpoint-diff-rg",
					ExpectedScope:  subscriptionID + ".different-rg",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					// Subnet (GET) - same resource group
					ExpectedType:   azureshared.NetworkSubnet.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey("test-vnet", "test-subnet"),
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					// Subnet (GET) - different resource group
					ExpectedType:   azureshared.NetworkSubnet.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey("test-vnet-diff-rg", "test-subnet-diff-rg"),
					ExpectedScope:  subscriptionID + ".different-rg",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					// Key Vault (GET)
					ExpectedType:   azureshared.KeyVaultVault.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-keyvault",
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					// User-Assigned Managed Identity (SEARCH) - same resource group
					ExpectedType:   azureshared.ManagedIdentityUserAssignedIdentity.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  resourceGroup,
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					// User-Assigned Managed Identity (SEARCH) - different resource group
					ExpectedType:   azureshared.ManagedIdentityUserAssignedIdentity.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  "identity-rg",
					ExpectedScope:  subscriptionID + ".identity-rg",
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
		mockClient := mocks.NewMockDocumentDBDatabaseAccountsClient(ctrl)

		wrapper := manual.NewDocumentDBDatabaseAccounts(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		// Test with empty name
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "", true)
		if qErr == nil {
			t.Error("Expected error when getting database account with empty name, but got nil")
		}
	})

	t.Run("Get_NoName", func(t *testing.T) {
		account := &armcosmos.DatabaseAccountGetResults{
			Name: nil, // No name field
			Properties: &armcosmos.DatabaseAccountGetProperties{
				ProvisioningState: to.Ptr("Succeeded"),
			},
		}

		mockClient := mocks.NewMockDocumentDBDatabaseAccountsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, accountName).Return(
			armcosmos.DatabaseAccountsClientGetResponse{
				DatabaseAccountGetResults: *account,
			}, nil)

		wrapper := manual.NewDocumentDBDatabaseAccounts(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], accountName, true)
		if qErr == nil {
			t.Error("Expected error when database account has no name, but got nil")
		}
	})

	t.Run("Get_NoLinkedResources", func(t *testing.T) {
		account := createAzureCosmosDBAccountMinimal(accountName, "Succeeded")

		mockClient := mocks.NewMockDocumentDBDatabaseAccountsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, accountName).Return(
			armcosmos.DatabaseAccountsClientGetResponse{
				DatabaseAccountGetResults: *account,
			}, nil)

		wrapper := manual.NewDocumentDBDatabaseAccounts(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], accountName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		// Should have no linked item queries
		if len(sdpItem.GetLinkedItemQueries()) != 0 {
			t.Errorf("Expected no linked item queries, got %d", len(sdpItem.GetLinkedItemQueries()))
		}
	})

	t.Run("List", func(t *testing.T) {
		account1 := createAzureCosmosDBAccount("test-cosmos-account-1", "Succeeded", subscriptionID, resourceGroup)
		account2 := createAzureCosmosDBAccount("test-cosmos-account-2", "Succeeded", subscriptionID, resourceGroup)

		mockPager := &mockDocumentDBDatabaseAccountsPager{
			pages: []armcosmos.DatabaseAccountsClientListByResourceGroupResponse{
				{
					DatabaseAccountsListResult: armcosmos.DatabaseAccountsListResult{
						Value: []*armcosmos.DatabaseAccountGetResults{account1, account2},
					},
				},
			},
		}

		mockClient := mocks.NewMockDocumentDBDatabaseAccountsClient(ctrl)
		mockClient.EXPECT().ListByResourceGroup(resourceGroup).Return(mockPager)

		testClient := &testDocumentDBDatabaseAccountsClient{
			MockDocumentDBDatabaseAccountsClient: mockClient,
			pager:                                mockPager,
		}

		wrapper := manual.NewDocumentDBDatabaseAccounts(testClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		listable, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support List operation")
		}

		sdpItems, err := listable.List(ctx, wrapper.Scopes()[0], true)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if len(sdpItems) != 2 {
			t.Fatalf("Expected 2 items, got: %d", len(sdpItems))
		}

		for _, item := range sdpItems {
			if item.Validate() != nil {
				t.Fatalf("Expected no validation error, got: %v", item.Validate())
			}

			if item.GetTags()["env"] != "test" {
				t.Fatalf("Expected tag 'env=test', got: %s", item.GetTags()["env"])
			}
		}
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		expectedErr := errors.New("database account not found")

		mockClient := mocks.NewMockDocumentDBDatabaseAccountsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, "nonexistent-account").Return(
			armcosmos.DatabaseAccountsClientGetResponse{}, expectedErr)

		wrapper := manual.NewDocumentDBDatabaseAccounts(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "nonexistent-account", true)
		if qErr == nil {
			t.Error("Expected error when getting non-existent database account, but got nil")
		}
	})

	t.Run("ListErrorHandling", func(t *testing.T) {
		mockClient := mocks.NewMockDocumentDBDatabaseAccountsClient(ctrl)
		errorPager := &errorDocumentDBDatabaseAccountsPager{}

		testClient := &testDocumentDBDatabaseAccountsClient{
			MockDocumentDBDatabaseAccountsClient: mockClient,
			pager:                                errorPager,
		}

		mockClient.EXPECT().ListByResourceGroup(resourceGroup).Return(errorPager)

		wrapper := manual.NewDocumentDBDatabaseAccounts(testClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		listable, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support List operation")
		}

		_, err := listable.List(ctx, wrapper.Scopes()[0], true)
		if err == nil {
			t.Error("Expected error when pager fails, but got nil")
		}
	})

	t.Run("CrossResourceGroupScopes", func(t *testing.T) {
		// Test that linked resources in different resource groups use correct scopes
		account := createAzureCosmosDBAccountCrossRG(accountName, "Succeeded", subscriptionID, resourceGroup)

		mockClient := mocks.NewMockDocumentDBDatabaseAccountsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, accountName).Return(
			armcosmos.DatabaseAccountsClientGetResponse{
				DatabaseAccountGetResults: *account,
			}, nil)

		wrapper := manual.NewDocumentDBDatabaseAccounts(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], accountName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		// Verify that linked resources use their own scopes, not the database account's scope
		foundDifferentScope := false
		for _, linkedQuery := range sdpItem.GetLinkedItemQueries() {
			scope := linkedQuery.GetQuery().GetScope()
			if scope != subscriptionID+"."+resourceGroup {
				foundDifferentScope = true
				// Verify the scope format is correct
				if scope != subscriptionID+".different-rg" && scope != subscriptionID+".identity-rg" {
					t.Errorf("Unexpected scope format: %s", scope)
				}
			}
		}

		if !foundDifferentScope {
			t.Error("Expected to find linked resources with different scopes, but all use the same scope")
		}
	})
}

// createAzureCosmosDBAccount creates a mock Azure Cosmos DB account with all linked resources
func createAzureCosmosDBAccount(accountName, provisioningState, subscriptionID, resourceGroup string) *armcosmos.DatabaseAccountGetResults {
	return &armcosmos.DatabaseAccountGetResults{
		Name:     to.Ptr(accountName),
		Location: to.Ptr("eastus"),
		Tags: map[string]*string{
			"env":     to.Ptr("test"),
			"project": to.Ptr("testing"),
		},
		Properties: &armcosmos.DatabaseAccountGetProperties{
			ProvisioningState: to.Ptr(provisioningState),
			// Private Endpoint Connections
			PrivateEndpointConnections: []*armcosmos.PrivateEndpointConnection{
				{
					Properties: &armcosmos.PrivateEndpointConnectionProperties{
						PrivateEndpoint: &armcosmos.PrivateEndpointProperty{
							ID: to.Ptr("/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/privateEndpoints/test-private-endpoint"),
						},
					},
				},
				{
					Properties: &armcosmos.PrivateEndpointConnectionProperties{
						PrivateEndpoint: &armcosmos.PrivateEndpointProperty{
							ID: to.Ptr("/subscriptions/" + subscriptionID + "/resourceGroups/different-rg/providers/Microsoft.Network/privateEndpoints/test-private-endpoint-diff-rg"),
						},
					},
				},
			},
			// Virtual Network Rules
			VirtualNetworkRules: []*armcosmos.VirtualNetworkRule{
				{
					ID: to.Ptr("/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/test-subnet"),
				},
				{
					ID: to.Ptr("/subscriptions/" + subscriptionID + "/resourceGroups/different-rg/providers/Microsoft.Network/virtualNetworks/test-vnet-diff-rg/subnets/test-subnet-diff-rg"),
				},
			},
			// Key Vault Key URI
			KeyVaultKeyURI: to.Ptr("https://test-keyvault.vault.azure.net/keys/test-key/version"),
		},
		Identity: &armcosmos.ManagedServiceIdentity{
			Type: to.Ptr(armcosmos.ResourceIdentityTypeUserAssigned),
			UserAssignedIdentities: map[string]*armcosmos.Components1Jq1T4ISchemasManagedserviceidentityPropertiesUserassignedidentitiesAdditionalproperties{
				"/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.ManagedIdentity/userAssignedIdentities/test-identity": {},
				"/subscriptions/" + subscriptionID + "/resourceGroups/identity-rg/providers/Microsoft.ManagedIdentity/userAssignedIdentities/test-identity-diff-rg":   {},
			},
		},
	}
}

// createAzureCosmosDBAccountMinimal creates a minimal mock Azure Cosmos DB account without linked resources
func createAzureCosmosDBAccountMinimal(accountName, provisioningState string) *armcosmos.DatabaseAccountGetResults {
	return &armcosmos.DatabaseAccountGetResults{
		Name:     to.Ptr(accountName),
		Location: to.Ptr("eastus"),
		Tags: map[string]*string{
			"env": to.Ptr("test"),
		},
		Properties: &armcosmos.DatabaseAccountGetProperties{
			ProvisioningState: to.Ptr(provisioningState),
		},
	}
}

// createAzureCosmosDBAccountCrossRG creates a mock Azure Cosmos DB account with linked resources in different resource groups
func createAzureCosmosDBAccountCrossRG(accountName, provisioningState, subscriptionID, resourceGroup string) *armcosmos.DatabaseAccountGetResults {
	return &armcosmos.DatabaseAccountGetResults{
		Name:     to.Ptr(accountName),
		Location: to.Ptr("eastus"),
		Tags: map[string]*string{
			"env": to.Ptr("test"),
		},
		Properties: &armcosmos.DatabaseAccountGetProperties{
			ProvisioningState: to.Ptr(provisioningState),
			// Private Endpoint in different resource group
			PrivateEndpointConnections: []*armcosmos.PrivateEndpointConnection{
				{
					Properties: &armcosmos.PrivateEndpointConnectionProperties{
						PrivateEndpoint: &armcosmos.PrivateEndpointProperty{
							ID: to.Ptr("/subscriptions/" + subscriptionID + "/resourceGroups/different-rg/providers/Microsoft.Network/privateEndpoints/test-pe-diff-rg"),
						},
					},
				},
			},
			// Subnet in different resource group
			VirtualNetworkRules: []*armcosmos.VirtualNetworkRule{
				{
					ID: to.Ptr("/subscriptions/" + subscriptionID + "/resourceGroups/different-rg/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/test-subnet"),
				},
			},
		},
		Identity: &armcosmos.ManagedServiceIdentity{
			Type: to.Ptr(armcosmos.ResourceIdentityTypeUserAssigned),
			UserAssignedIdentities: map[string]*armcosmos.Components1Jq1T4ISchemasManagedserviceidentityPropertiesUserassignedidentitiesAdditionalproperties{
				"/subscriptions/" + subscriptionID + "/resourceGroups/identity-rg/providers/Microsoft.ManagedIdentity/userAssignedIdentities/test-identity": {},
			},
		},
	}
}
