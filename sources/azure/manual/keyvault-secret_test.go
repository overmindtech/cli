package manual_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/keyvault/armkeyvault/v2"
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
	"github.com/overmindtech/cli/sources/stdlib"
)

// mockSecretsPager is a simple mock implementation of SecretsPager
type mockSecretsPager struct {
	pages []armkeyvault.SecretsClientListResponse
	index int
}

func (m *mockSecretsPager) More() bool {
	return m.index < len(m.pages)
}

func (m *mockSecretsPager) NextPage(ctx context.Context) (armkeyvault.SecretsClientListResponse, error) {
	if m.index >= len(m.pages) {
		return armkeyvault.SecretsClientListResponse{}, errors.New("no more pages")
	}
	page := m.pages[m.index]
	m.index++
	return page, nil
}

// errorSecretsPager is a mock pager that always returns an error
type errorSecretsPager struct{}

func (e *errorSecretsPager) More() bool {
	return true // Always return true so NextPage will be called
}

func (e *errorSecretsPager) NextPage(ctx context.Context) (armkeyvault.SecretsClientListResponse, error) {
	return armkeyvault.SecretsClientListResponse{}, errors.New("pager error")
}

// testSecretsClient wraps the mock to implement the correct interface
type testSecretsClient struct {
	*mocks.MockSecretsClient
	pager clients.SecretsPager
}

func (t *testSecretsClient) NewListPager(resourceGroupName, vaultName string, options *armkeyvault.SecretsClientListOptions) clients.SecretsPager {
	// Call the mock to satisfy expectations
	t.MockSecretsClient.NewListPager(resourceGroupName, vaultName, options)
	return t.pager
}

func TestKeyVaultSecret(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"
	vaultName := "test-keyvault"
	secretName := "test-secret"

	t.Run("Get", func(t *testing.T) {
		secret := createAzureSecret(secretName, subscriptionID, resourceGroup, vaultName)

		mockClient := mocks.NewMockSecretsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, vaultName, secretName, nil).Return(
			armkeyvault.SecretsClientGetResponse{
				Secret: *secret,
			}, nil)

		wrapper := manual.NewKeyVaultSecret(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		// Get requires vaultName and secretName as query parts
		query := vaultName + shared.QuerySeparator + secretName
		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.KeyVaultSecret.String() {
			t.Errorf("Expected type %s, got %s", azureshared.KeyVaultSecret, sdpItem.GetType())
		}

		if sdpItem.GetUniqueAttribute() != "uniqueAttr" {
			t.Errorf("Expected unique attribute 'uniqueAttr', got %s", sdpItem.GetUniqueAttribute())
		}

		expectedUniqueAttrValue := shared.CompositeLookupKey(vaultName, secretName)
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
					// Key Vault (GET) - same resource group
					ExpectedType:   azureshared.KeyVaultVault.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  vaultName,
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,  // If Key Vault is deleted/modified → secret access and configuration are affected
						Out: false, // If secret is deleted → Key Vault remains
					},
				},
				{
					// stdlib.NetworkDNS from SecretURI hostname
					ExpectedType:   stdlib.NetworkDNS.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  vaultName + ".vault.azure.net",
					ExpectedScope:  "global",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					// stdlib.NetworkHTTP from SecretURI
					ExpectedType:   stdlib.NetworkHTTP.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  fmt.Sprintf("https://%s.vault.azure.net/secrets/%s", vaultName, secretName),
					ExpectedScope:  "global",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
			}

			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("Get_InvalidQueryParts", func(t *testing.T) {
		mockClient := mocks.NewMockSecretsClient(ctrl)

		wrapper := manual.NewKeyVaultSecret(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		// Test with insufficient query parts (only vault name)
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], vaultName, true)
		if qErr == nil {
			t.Error("Expected error when providing insufficient query parts, but got nil")
		}
	})

	t.Run("Get_EmptyVaultName", func(t *testing.T) {
		mockClient := mocks.NewMockSecretsClient(ctrl)

		wrapper := manual.NewKeyVaultSecret(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		// Test with empty vault name
		query := shared.QuerySeparator + secretName
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr == nil {
			t.Error("Expected error when vault name is empty, but got nil")
		}
	})

	t.Run("Get_EmptySecretName", func(t *testing.T) {
		mockClient := mocks.NewMockSecretsClient(ctrl)

		wrapper := manual.NewKeyVaultSecret(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		// Test with empty secret name
		query := vaultName + shared.QuerySeparator
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr == nil {
			t.Error("Expected error when secret name is empty, but got nil")
		}
	})

	t.Run("Get_NoName", func(t *testing.T) {
		secret := &armkeyvault.Secret{
			Name: nil, // No name field
		}

		mockClient := mocks.NewMockSecretsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, vaultName, secretName, nil).Return(
			armkeyvault.SecretsClientGetResponse{
				Secret: *secret,
			}, nil)

		wrapper := manual.NewKeyVaultSecret(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := vaultName + shared.QuerySeparator + secretName
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr == nil {
			t.Error("Expected error when secret has no name, but got nil")
		}
	})

	t.Run("Get_NoLinkedResources", func(t *testing.T) {
		secret := createAzureSecretMinimal(secretName)

		mockClient := mocks.NewMockSecretsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, vaultName, secretName, nil).Return(
			armkeyvault.SecretsClientGetResponse{
				Secret: *secret,
			}, nil)

		wrapper := manual.NewKeyVaultSecret(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := vaultName + shared.QuerySeparator + secretName
		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		// Should have no linked item queries when ID is nil or empty
		if len(sdpItem.GetLinkedItemQueries()) != 0 {
			t.Errorf("Expected no linked item queries, got %d", len(sdpItem.GetLinkedItemQueries()))
		}
	})

	t.Run("Search", func(t *testing.T) {
		secret1 := createAzureSecret("secret-1", subscriptionID, resourceGroup, vaultName)
		secret2 := createAzureSecret("secret-2", subscriptionID, resourceGroup, vaultName)

		mockPager := &mockSecretsPager{
			pages: []armkeyvault.SecretsClientListResponse{
				{
					SecretListResult: armkeyvault.SecretListResult{
						Value: []*armkeyvault.Secret{
							{
								ID:         secret1.ID,
								Name:       secret1.Name,
								Type:       secret1.Type,
								Properties: secret1.Properties,
								Tags:       secret1.Tags,
							},
							{
								ID:         secret2.ID,
								Name:       secret2.Name,
								Type:       secret2.Type,
								Properties: secret2.Properties,
								Tags:       secret2.Tags,
							},
						},
					},
				},
			},
		}

		mockClient := mocks.NewMockSecretsClient(ctrl)
		mockClient.EXPECT().NewListPager(resourceGroup, vaultName, nil).Return(mockPager)

		testClient := &testSecretsClient{
			MockSecretsClient: mockClient,
			pager:             mockPager,
		}

		wrapper := manual.NewKeyVaultSecret(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support Search operation")
		}

		sdpItems, err := searchable.Search(ctx, wrapper.Scopes()[0], vaultName, true)
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

			if item.GetType() != azureshared.KeyVaultSecret.String() {
				t.Errorf("Expected type %s, got %s", azureshared.KeyVaultSecret, item.GetType())
			}
		}
	})

	t.Run("Search_InvalidQueryParts", func(t *testing.T) {
		mockClient := mocks.NewMockSecretsClient(ctrl)
		testClient := &testSecretsClient{MockSecretsClient: mockClient}

		wrapper := manual.NewKeyVaultSecret(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		// Test Search directly with no query parts - should return error before calling List
		_, qErr := wrapper.Search(ctx, wrapper.Scopes()[0])
		if qErr == nil {
			t.Error("Expected error when providing no query parts, but got nil")
		}
	})

	t.Run("Search_EmptyVaultName", func(t *testing.T) {
		mockClient := mocks.NewMockSecretsClient(ctrl)
		testClient := &testSecretsClient{MockSecretsClient: mockClient}

		wrapper := manual.NewKeyVaultSecret(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		// Test Search directly with empty vault name
		_, qErr := wrapper.Search(ctx, "")
		if qErr == nil {
			t.Error("Expected error when vault name is empty, but got nil")
		}
	})

	t.Run("Search_SecretWithNilName", func(t *testing.T) {
		validSecret := createAzureSecret("valid-secret", subscriptionID, resourceGroup, vaultName)
		mockPager := &mockSecretsPager{
			pages: []armkeyvault.SecretsClientListResponse{
				{
					SecretListResult: armkeyvault.SecretListResult{
						Value: []*armkeyvault.Secret{
							{
								// Secret with nil name should be skipped
								Name: nil,
							},
							{
								ID:         validSecret.ID,
								Name:       validSecret.Name,
								Type:       validSecret.Type,
								Properties: validSecret.Properties,
								Tags:       validSecret.Tags,
							},
						},
					},
				},
			},
		}

		mockClient := mocks.NewMockSecretsClient(ctrl)
		mockClient.EXPECT().NewListPager(resourceGroup, vaultName, nil).Return(mockPager)

		testClient := &testSecretsClient{
			MockSecretsClient: mockClient,
			pager:             mockPager,
		}

		wrapper := manual.NewKeyVaultSecret(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support Search operation")
		}

		sdpItems, err := searchable.Search(ctx, wrapper.Scopes()[0], vaultName, true)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		// Should only return 1 item (the one with a valid name)
		if len(sdpItems) != 1 {
			t.Fatalf("Expected 1 item, got: %d", len(sdpItems))
		}

		expectedUniqueAttrValue := shared.CompositeLookupKey(vaultName, "valid-secret")
		if sdpItems[0].UniqueAttributeValue() != expectedUniqueAttrValue {
			t.Errorf("Expected unique attribute value %s, got %s", expectedUniqueAttrValue, sdpItems[0].UniqueAttributeValue())
		}
	})

	t.Run("ErrorHandling_Get", func(t *testing.T) {
		expectedErr := errors.New("secret not found")

		mockClient := mocks.NewMockSecretsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, vaultName, "nonexistent-secret", nil).Return(
			armkeyvault.SecretsClientGetResponse{}, expectedErr)

		wrapper := manual.NewKeyVaultSecret(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := vaultName + shared.QuerySeparator + "nonexistent-secret"
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr == nil {
			t.Error("Expected error when getting non-existent secret, but got nil")
		}
	})

	t.Run("ErrorHandling_Search", func(t *testing.T) {
		mockClient := mocks.NewMockSecretsClient(ctrl)
		// Create a pager that returns an error when NextPage is called
		errorPager := &errorSecretsPager{}

		mockClient.EXPECT().NewListPager(resourceGroup, vaultName, nil).Return(errorPager)

		testClient := &testSecretsClient{
			MockSecretsClient: mockClient,
			pager:             errorPager,
		}

		wrapper := manual.NewKeyVaultSecret(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support Search operation")
		}

		_, err := searchable.Search(ctx, wrapper.Scopes()[0], vaultName, true)
		// The Search implementation should return an error when pager.NextPage returns an error
		// Errors from NextPage are converted to QueryError by the implementation
		if err == nil {
			t.Error("Expected error from pager when NextPage returns an error, but got nil")
		}
	})

	t.Run("InterfaceCompliance", func(t *testing.T) {
		mockClient := mocks.NewMockSecretsClient(ctrl)
		testClient := &testSecretsClient{MockSecretsClient: mockClient}
		wrapper := manual.NewKeyVaultSecret(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		// Verify wrapper implements SearchableWrapper (it's returned as this type)
		if wrapper == nil {
			t.Error("Wrapper should not be nil")
		}

		// Verify adapter implements SearchableAdapter
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())
		_, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Error("Adapter should implement SearchableAdapter interface")
		}
	})

	t.Run("PotentialLinks", func(t *testing.T) {
		mockClient := mocks.NewMockSecretsClient(ctrl)
		testClient := &testSecretsClient{MockSecretsClient: mockClient}
		wrapper := manual.NewKeyVaultSecret(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		links := wrapper.PotentialLinks()
		if len(links) == 0 {
			t.Error("Expected potential links to be defined")
		}

		if !links[azureshared.KeyVaultVault] {
			t.Error("Expected KeyVaultVault to be in potential links")
		}

		if !links[stdlib.NetworkDNS] {
			t.Error("Expected stdlib.NetworkDNS to be in potential links")
		}

		if !links[stdlib.NetworkHTTP] {
			t.Error("Expected stdlib.NetworkHTTP to be in potential links")
		}
	})

	t.Run("TerraformMappings", func(t *testing.T) {
		mockClient := mocks.NewMockSecretsClient(ctrl)
		testClient := &testSecretsClient{MockSecretsClient: mockClient}
		wrapper := manual.NewKeyVaultSecret(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		mappings := wrapper.TerraformMappings()
		if len(mappings) == 0 {
			t.Fatal("Expected TerraformMappings to be defined")
		}

		// Verify we have the correct mapping for azurerm_key_vault_secret.id
		foundIDMapping := false
		for _, mapping := range mappings {
			if mapping.GetTerraformQueryMap() == "azurerm_key_vault_secret.id" {
				foundIDMapping = true
				if mapping.GetTerraformMethod() != sdp.QueryMethod_SEARCH {
					t.Errorf("Expected TerraformMethod to be SEARCH for id mapping, got %s", mapping.GetTerraformMethod())
				}
			}
		}

		if !foundIDMapping {
			t.Error("Expected TerraformMappings to include 'azurerm_key_vault_secret.id' mapping")
		}

		// Verify we only have one mapping
		if len(mappings) != 1 {
			t.Errorf("Expected 1 TerraformMapping, got %d", len(mappings))
		}
	})

	t.Run("IAMPermissions", func(t *testing.T) {
		mockClient := mocks.NewMockSecretsClient(ctrl)
		testClient := &testSecretsClient{MockSecretsClient: mockClient}
		wrapper := manual.NewKeyVaultSecret(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		permissions := wrapper.IAMPermissions()
		if len(permissions) == 0 {
			t.Error("Expected IAMPermissions to be defined")
		}

		expectedPermission := "Microsoft.KeyVault/vaults/secrets/read"
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
	})

	t.Run("PredefinedRole", func(t *testing.T) {
		mockClient := mocks.NewMockSecretsClient(ctrl)
		testClient := &testSecretsClient{MockSecretsClient: mockClient}
		wrapper := manual.NewKeyVaultSecret(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		// PredefinedRole is available on the wrapper, not the interface
		// Use type assertion to access the concrete type
		type predefinedRoleInterface interface {
			PredefinedRole() string
		}
		if roleInterface, ok := wrapper.(predefinedRoleInterface); ok {
			role := roleInterface.PredefinedRole()
			if role != "Reader" {
				t.Errorf("Expected PredefinedRole to be 'Reader', got %s", role)
			}
		} else {
			t.Error("Wrapper should implement PredefinedRole method")
		}
	})

	t.Run("CrossResourceGroupScopes", func(t *testing.T) {
		// Test that linked resources in different resource groups use correct scopes
		// Secret ID format: /subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.KeyVault/vaults/{vaultName}/secrets/{secretName}
		// The vault can be in a different resource group
		differentResourceGroup := "different-rg"
		secret := createAzureSecretCrossRG(secretName, subscriptionID, differentResourceGroup, vaultName)

		mockClient := mocks.NewMockSecretsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, vaultName, secretName, nil).Return(
			armkeyvault.SecretsClientGetResponse{
				Secret: *secret,
			}, nil)

		wrapper := manual.NewKeyVaultSecret(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := vaultName + shared.QuerySeparator + secretName
		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		// Verify that linked vault uses its own scope, not the secret's resource group scope
		linkedQueries := sdpItem.GetLinkedItemQueries()
		if len(linkedQueries) != 1 {
			t.Fatalf("Expected 1 linked item query, got %d", len(linkedQueries))
		}

		linkedQuery := linkedQueries[0]
		scope := linkedQuery.GetQuery().GetScope()
		expectedScope := fmt.Sprintf("%s.%s", subscriptionID, differentResourceGroup)
		if scope != expectedScope {
			t.Errorf("Expected linked vault scope to be %s, got %s", expectedScope, scope)
		}

		if linkedQuery.GetQuery().GetQuery() != vaultName {
			t.Errorf("Expected linked vault query to be %s, got %s", vaultName, linkedQuery.GetQuery().GetQuery())
		}
	})
}

// createAzureSecret creates a mock Azure Key Vault secret with linked vault
func createAzureSecret(secretName, subscriptionID, resourceGroup, vaultName string) *armkeyvault.Secret {
	return &armkeyvault.Secret{
		ID:   to.Ptr(fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.KeyVault/vaults/%s/secrets/%s", subscriptionID, resourceGroup, vaultName, secretName)),
		Name: to.Ptr(secretName),
		Type: to.Ptr("Microsoft.KeyVault/vaults/secrets"),
		Tags: map[string]*string{
			"env":     to.Ptr("test"),
			"project": to.Ptr("testing"),
		},
		Properties: &armkeyvault.SecretProperties{
			Value:     to.Ptr("secret-value"),
			SecretURI: to.Ptr(fmt.Sprintf("https://%s.vault.azure.net/secrets/%s", vaultName, secretName)),
		},
	}
}

// createAzureSecretMinimal creates a minimal mock Azure Key Vault secret without ID (no linked resources)
func createAzureSecretMinimal(secretName string) *armkeyvault.Secret {
	return &armkeyvault.Secret{
		Name: to.Ptr(secretName),
		Type: to.Ptr("Microsoft.KeyVault/vaults/secrets"),
		Tags: map[string]*string{
			"env": to.Ptr("test"),
		},
		Properties: &armkeyvault.SecretProperties{
			Value: to.Ptr("secret-value"),
		},
	}
}

// createAzureSecretCrossRG creates a mock Azure Key Vault secret with vault in a different resource group
func createAzureSecretCrossRG(secretName, subscriptionID, vaultResourceGroup, vaultName string) *armkeyvault.Secret {
	return &armkeyvault.Secret{
		ID:   to.Ptr(fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.KeyVault/vaults/%s/secrets/%s", subscriptionID, vaultResourceGroup, vaultName, secretName)),
		Name: to.Ptr(secretName),
		Type: to.Ptr("Microsoft.KeyVault/vaults/secrets"),
		Tags: map[string]*string{
			"env": to.Ptr("test"),
		},
		Properties: &armkeyvault.SecretProperties{
			Value: to.Ptr("secret-value"),
		},
	}
}
