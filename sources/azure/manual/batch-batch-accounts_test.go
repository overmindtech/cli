package manual_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/batch/armbatch/v3"
	"go.uber.org/mock/gomock"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/manual"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/azure/shared/mocks"
	"github.com/overmindtech/cli/sources/shared"
)

// mockBatchAccountsPager is a simple mock implementation of BatchAccountsPager
type mockBatchAccountsPager struct {
	ctrl     *gomock.Controller
	more     bool
	response armbatch.AccountClientListByResourceGroupResponse
	err      error
}

func (m *mockBatchAccountsPager) More() bool {
	return m.more
}

func (m *mockBatchAccountsPager) NextPage(ctx context.Context) (armbatch.AccountClientListByResourceGroupResponse, error) {
	m.more = false // After NextPage, More() should return false
	return m.response, m.err
}

func TestBatchAccount(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"

	t.Run("Get", func(t *testing.T) {
		accountName := "test-batch-account"
		account := createAzureBatchAccount(accountName, "Succeeded", subscriptionID, resourceGroup)

		mockClient := mocks.NewMockBatchAccountsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, accountName).Return(
			armbatch.AccountClientGetResponse{
				Account: *account,
			}, nil)

		wrapper := manual.NewBatchAccount(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], accountName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.BatchBatchAccount.String() {
			t.Errorf("Expected type %s, got %s", azureshared.BatchBatchAccount, sdpItem.GetType())
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
					// Storage Account link
					ExpectedType:   azureshared.StorageAccount.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-storage-account",
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					// Key Vault link
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
					// Private Endpoint link
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
					// User Assigned Managed Identity link
					ExpectedType:   azureshared.ManagedIdentityUserAssignedIdentity.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-identity",
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					// Node Identity Reference link
					ExpectedType:   azureshared.ManagedIdentityUserAssignedIdentity.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-node-identity",
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					// Applications (child resource)
					ExpectedType:   azureshared.BatchBatchApplication.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  accountName,
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					// Pools (child resource)
					ExpectedType:   azureshared.BatchBatchPool.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  accountName,
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					// Certificates (child resource)
					ExpectedType:   azureshared.BatchBatchCertificate.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  accountName,
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					// Private Endpoint Connections (child resource)
					ExpectedType:   azureshared.BatchBatchPrivateEndpointConnection.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  accountName,
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					// Private Link Resources (child resource)
					ExpectedType:   azureshared.BatchBatchPrivateLinkResource.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  accountName,
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					// Detectors (child resource)
					ExpectedType:   azureshared.BatchBatchDetector.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  accountName,
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
			}

			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("Get_EmptyAccountName", func(t *testing.T) {
		mockClient := mocks.NewMockBatchAccountsClient(ctrl)

		wrapper := manual.NewBatchAccount(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "", true)
		if qErr == nil {
			t.Error("Expected error when account name is empty, but got nil")
		}
	})

	t.Run("Get_InvalidQueryParts", func(t *testing.T) {
		mockClient := mocks.NewMockBatchAccountsClient(ctrl)

		wrapper := manual.NewBatchAccount(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		// Test with no query parts
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "", true)
		if qErr == nil {
			t.Error("Expected error when no query parts provided, but got nil")
		}
	})

	t.Run("Get_ClientError", func(t *testing.T) {
		accountName := "test-batch-account"
		expectedErr := errors.New("batch account not found")

		mockClient := mocks.NewMockBatchAccountsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, accountName).Return(
			armbatch.AccountClientGetResponse{}, expectedErr)

		wrapper := manual.NewBatchAccount(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], accountName, true)
		if qErr == nil {
			t.Error("Expected error when client returns error, but got nil")
		}
	})

	t.Run("List", func(t *testing.T) {
		account1 := createAzureBatchAccount("test-batch-account-1", "Succeeded", subscriptionID, resourceGroup)
		account2 := createAzureBatchAccount("test-batch-account-2", "Succeeded", subscriptionID, resourceGroup)

		mockClient := mocks.NewMockBatchAccountsClient(ctrl)
		mockPager := &mockBatchAccountsPager{
			ctrl: ctrl,
			more: true,
			response: armbatch.AccountClientListByResourceGroupResponse{
				AccountListResult: armbatch.AccountListResult{
					Value: []*armbatch.Account{account1, account2},
				},
			},
		}

		mockClient.EXPECT().ListByResourceGroup(ctx, resourceGroup).Return(mockPager)

		wrapper := manual.NewBatchAccount(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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

	t.Run("List_WithNilName", func(t *testing.T) {
		account1 := createAzureBatchAccount("test-batch-account-1", "Succeeded", subscriptionID, resourceGroup)
		account2NilName := createAzureBatchAccount("test-batch-account-2", "Succeeded", subscriptionID, resourceGroup)
		account2NilName.Name = nil // Set name to nil to test filtering

		mockClient := mocks.NewMockBatchAccountsClient(ctrl)
		mockPager := &mockBatchAccountsPager{
			ctrl: ctrl,
			more: true,
			response: armbatch.AccountClientListByResourceGroupResponse{
				AccountListResult: armbatch.AccountListResult{
					Value: []*armbatch.Account{account1, account2NilName},
				},
			},
		}

		mockClient.EXPECT().ListByResourceGroup(ctx, resourceGroup).Return(mockPager)

		wrapper := manual.NewBatchAccount(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		listable, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support List operation")
		}

		sdpItems, err := listable.List(ctx, wrapper.Scopes()[0], true)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		// Should only return 1 item since account2 has nil name
		if len(sdpItems) != 1 {
			t.Fatalf("Expected 1 item (filtered out nil name), got: %d", len(sdpItems))
		}
	})

	t.Run("List_PagerError", func(t *testing.T) {
		expectedErr := errors.New("pager error")

		mockClient := mocks.NewMockBatchAccountsClient(ctrl)
		mockPager := &mockBatchAccountsPager{
			ctrl: ctrl,
			more: true,
			err:  expectedErr,
		}

		mockClient.EXPECT().ListByResourceGroup(ctx, resourceGroup).Return(mockPager)

		wrapper := manual.NewBatchAccount(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		listable, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support List operation")
		}

		_, err := listable.List(ctx, wrapper.Scopes()[0], true)
		if err == nil {
			t.Error("Expected error when pager returns error, but got nil")
		}
	})

	t.Run("GetLookups", func(t *testing.T) {
		mockClient := mocks.NewMockBatchAccountsClient(ctrl)
		wrapper := manual.NewBatchAccount(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		lookups := wrapper.GetLookups()
		if len(lookups) != 1 {
			t.Fatalf("Expected 1 lookup, got: %d", len(lookups))
		}

		if lookups[0].ItemType != azureshared.BatchBatchAccount {
			t.Errorf("Expected lookup item type %s, got %s", azureshared.BatchBatchAccount, lookups[0].ItemType)
		}
	})

	t.Run("PotentialLinks", func(t *testing.T) {
		mockClient := mocks.NewMockBatchAccountsClient(ctrl)
		wrapper := manual.NewBatchAccount(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		potentialLinks := wrapper.PotentialLinks()
		expectedLinks := []shared.ItemType{
			azureshared.StorageAccount,
			azureshared.KeyVaultVault,
			azureshared.NetworkPrivateEndpoint,
			azureshared.ManagedIdentityUserAssignedIdentity,
			azureshared.BatchBatchApplication,
			azureshared.BatchBatchPool,
			azureshared.BatchBatchCertificate,
			azureshared.BatchBatchPrivateEndpointConnection,
			azureshared.BatchBatchPrivateLinkResource,
			azureshared.BatchBatchDetector,
		}

		for _, expectedLink := range expectedLinks {
			if !potentialLinks[expectedLink] {
				t.Errorf("Expected potential link %s to be true, got false", expectedLink)
			}
		}
	})

	t.Run("TerraformMappings", func(t *testing.T) {
		mockClient := mocks.NewMockBatchAccountsClient(ctrl)
		wrapper := manual.NewBatchAccount(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		mappings := wrapper.TerraformMappings()
		if len(mappings) != 1 {
			t.Fatalf("Expected 1 terraform mapping, got: %d", len(mappings))
		}

		if mappings[0].GetTerraformMethod() != sdp.QueryMethod_GET {
			t.Errorf("Expected terraform method GET, got: %s", mappings[0].GetTerraformMethod())
		}

		if mappings[0].GetTerraformQueryMap() != "azurerm_batch_account.name" {
			t.Errorf("Expected terraform query map 'azurerm_batch_account.name', got: %s", mappings[0].GetTerraformQueryMap())
		}
	})

	t.Run("IAMPermissions", func(t *testing.T) {
		mockClient := mocks.NewMockBatchAccountsClient(ctrl)
		wrapper := manual.NewBatchAccount(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		permissions := wrapper.IAMPermissions()
		expectedPermissions := []string{
			"Microsoft.Batch/batchAccounts/read",
		}

		if len(permissions) != len(expectedPermissions) {
			t.Fatalf("Expected %d permissions, got: %d", len(expectedPermissions), len(permissions))
		}

		for i, expected := range expectedPermissions {
			if permissions[i] != expected {
				t.Errorf("Expected permission %s, got: %s", expected, permissions[i])
			}
		}
	})

	t.Run("PredefinedRole", func(t *testing.T) {
		mockClient := mocks.NewMockBatchAccountsClient(ctrl)
		wrapper := manual.NewBatchAccount(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		// PredefinedRole is available on the wrapper, not the adapter
		role := wrapper.(interface{ PredefinedRole() string }).PredefinedRole()
		expectedRole := "Azure Batch Account Reader"

		if role != expectedRole {
			t.Errorf("Expected role %s, got: %s", expectedRole, role)
		}
	})

	t.Run("CrossResourceGroupScope", func(t *testing.T) {
		// Test that resources in different resource groups use the correct scope
		otherSubscriptionID := "other-subscription"
		otherResourceGroup := "other-rg"

		accountName := "test-batch-account"
		account := createAzureBatchAccountWithCrossRGResources(
			accountName, "Succeeded",
			subscriptionID, resourceGroup,
			otherSubscriptionID, otherResourceGroup,
		)

		mockClient := mocks.NewMockBatchAccountsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, accountName).Return(
			armbatch.AccountClientGetResponse{
				Account: *account,
			}, nil)

		wrapper := manual.NewBatchAccount(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], accountName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		// Find the storage account link (which is in a different resource group)
		foundCrossRGStorage := false
		for _, linkedQuery := range sdpItem.GetLinkedItemQueries() {
			if linkedQuery.GetQuery().GetType() == azureshared.StorageAccount.String() {
				expectedScope := otherSubscriptionID + "." + otherResourceGroup
				if linkedQuery.GetQuery().GetScope() != expectedScope {
					t.Errorf("Expected storage account scope %s, got: %s", expectedScope, linkedQuery.GetQuery().GetScope())
				}
				foundCrossRGStorage = true
			}
		}

		if !foundCrossRGStorage {
			t.Error("Expected to find storage account link with cross-resource-group scope")
		}
	})
}

// createAzureBatchAccount creates a mock Azure Batch Account for testing
func createAzureBatchAccount(accountName, provisioningState, subscriptionID, resourceGroup string) *armbatch.Account {
	storageAccountID := "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Storage/storageAccounts/test-storage-account"
	keyVaultID := "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.KeyVault/vaults/test-keyvault"
	privateEndpointID := "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/privateEndpoints/test-private-endpoint"
	identityID := "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.ManagedIdentity/userAssignedIdentities/test-identity"
	nodeIdentityID := "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.ManagedIdentity/userAssignedIdentities/test-node-identity"

	return &armbatch.Account{
		Name:     to.Ptr(accountName),
		Location: to.Ptr("eastus"),
		Tags: map[string]*string{
			"env":     to.Ptr("test"),
			"project": to.Ptr("testing"),
		},
		Properties: &armbatch.AccountProperties{
			ProvisioningState: (*armbatch.ProvisioningState)(to.Ptr(provisioningState)),
			AutoStorage: &armbatch.AutoStorageProperties{
				StorageAccountID: to.Ptr(storageAccountID),
				LastKeySync:      to.Ptr(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
				NodeIdentityReference: &armbatch.ComputeNodeIdentityReference{
					ResourceID: to.Ptr(nodeIdentityID),
				},
			},
			KeyVaultReference: &armbatch.KeyVaultReference{
				ID:  to.Ptr(keyVaultID),
				URL: to.Ptr("https://test-keyvault.vault.azure.net/"),
			},
			PrivateEndpointConnections: []*armbatch.PrivateEndpointConnection{
				{
					Properties: &armbatch.PrivateEndpointConnectionProperties{
						PrivateEndpoint: &armbatch.PrivateEndpoint{
							ID: to.Ptr(privateEndpointID),
						},
					},
				},
			},
		},
		Identity: &armbatch.AccountIdentity{
			Type: (*armbatch.ResourceIdentityType)(to.Ptr(armbatch.ResourceIdentityTypeUserAssigned)),
			UserAssignedIdentities: map[string]*armbatch.UserAssignedIdentities{
				identityID: {},
			},
		},
	}
}

// createAzureBatchAccountWithCrossRGResources creates a mock Azure Batch Account with resources in different resource groups
func createAzureBatchAccountWithCrossRGResources(
	accountName, provisioningState,
	subscriptionID, resourceGroup,
	otherSubscriptionID, otherResourceGroup string,
) *armbatch.Account {
	// Storage account is in a different resource group
	storageAccountID := "/subscriptions/" + otherSubscriptionID + "/resourceGroups/" + otherResourceGroup + "/providers/Microsoft.Storage/storageAccounts/test-storage-account"

	return &armbatch.Account{
		Name:     to.Ptr(accountName),
		Location: to.Ptr("eastus"),
		Tags: map[string]*string{
			"env": to.Ptr("test"),
		},
		Properties: &armbatch.AccountProperties{
			ProvisioningState: (*armbatch.ProvisioningState)(to.Ptr(provisioningState)),
			AutoStorage: &armbatch.AutoStorageProperties{
				StorageAccountID: to.Ptr(storageAccountID),
				LastKeySync:      to.Ptr(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
			},
		},
	}
}
