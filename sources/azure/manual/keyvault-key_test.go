package manual_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
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
	"github.com/overmindtech/cli/sources/stdlib"
)


type mockKeysPager struct {
	pages []armkeyvault.KeysClientListResponse
	index int
}

func (m *mockKeysPager) More() bool {
	return m.index < len(m.pages)
}

func (m *mockKeysPager) NextPage(ctx context.Context) (armkeyvault.KeysClientListResponse, error) {
	if m.index >= len(m.pages) {
		return armkeyvault.KeysClientListResponse{}, errors.New("no more pages")
	}
	page := m.pages[m.index]
	m.index++
	return page, nil
}

type errorKeysPager struct{}

func (e *errorKeysPager) More() bool { return true }

func (e *errorKeysPager) NextPage(ctx context.Context) (armkeyvault.KeysClientListResponse, error) {
	return armkeyvault.KeysClientListResponse{}, errors.New("pager error")
}

type testKeysClient struct {
	*mocks.MockKeysClient
	pager clients.KeysPager
}

func (t *testKeysClient) NewListPager(resourceGroupName, vaultName string, options *armkeyvault.KeysClientListOptions) clients.KeysPager {
	t.MockKeysClient.NewListPager(resourceGroupName, vaultName, options)
	return t.pager
}

func TestKeyVaultKey(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"
	vaultName := "test-keyvault"
	keyName := "test-key"

	t.Run("Get", func(t *testing.T) {
		key := createAzureKey(keyName, subscriptionID, resourceGroup, vaultName)

		mockClient := mocks.NewMockKeysClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, vaultName, keyName, nil).Return(
			armkeyvault.KeysClientGetResponse{
				Key: *key,
			}, nil)

		wrapper := manual.NewKeyVaultKey(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := vaultName + shared.QuerySeparator + keyName
		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.KeyVaultKey.String() {
			t.Errorf("Expected type %s, got %s", azureshared.KeyVaultKey, sdpItem.GetType())
		}

		if sdpItem.GetUniqueAttribute() != "uniqueAttr" {
			t.Errorf("Expected unique attribute 'uniqueAttr', got %s", sdpItem.GetUniqueAttribute())
		}

		expectedUniqueAttrValue := shared.CompositeLookupKey(vaultName, keyName)
		if sdpItem.UniqueAttributeValue() != expectedUniqueAttrValue {
			t.Errorf("Expected unique attribute value %s, got %s", expectedUniqueAttrValue, sdpItem.UniqueAttributeValue())
		}

		if sdpItem.GetScope() != subscriptionID+"."+resourceGroup {
			t.Errorf("Expected scope %s, got %s", subscriptionID+"."+resourceGroup, sdpItem.GetScope())
		}

		if err := sdpItem.Validate(); err != nil {
			t.Fatalf("Expected no validation error, got: %v", err)
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					ExpectedType:   azureshared.KeyVaultVault.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  vaultName,
					ExpectedScope:  subscriptionID + "." + resourceGroup,
				}, {
					ExpectedType:   stdlib.NetworkDNS.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  vaultName + ".vault.azure.net",
					ExpectedScope:  "global",
				}, {
					ExpectedType:   stdlib.NetworkHTTP.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  fmt.Sprintf("https://%s.vault.azure.net/keys/%s", vaultName, keyName),
					ExpectedScope:  "global",
				}}

			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("Get_InvalidQueryParts", func(t *testing.T) {
		mockClient := mocks.NewMockKeysClient(ctrl)

		wrapper := manual.NewKeyVaultKey(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], vaultName, true)
		if qErr == nil {
			t.Error("Expected error when providing insufficient query parts, but got nil")
		}
	})

	t.Run("Get_EmptyVaultName", func(t *testing.T) {
		mockClient := mocks.NewMockKeysClient(ctrl)

		wrapper := manual.NewKeyVaultKey(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.QuerySeparator + keyName
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr == nil {
			t.Error("Expected error when vault name is empty, but got nil")
		}
	})

	t.Run("Get_EmptyKeyName", func(t *testing.T) {
		mockClient := mocks.NewMockKeysClient(ctrl)

		wrapper := manual.NewKeyVaultKey(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := vaultName + shared.QuerySeparator
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr == nil {
			t.Error("Expected error when key name is empty, but got nil")
		}
	})

	t.Run("Get_NoName", func(t *testing.T) {
		key := &armkeyvault.Key{
			Name: nil,
		}

		mockClient := mocks.NewMockKeysClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, vaultName, keyName, nil).Return(
			armkeyvault.KeysClientGetResponse{
				Key: *key,
			}, nil)

		wrapper := manual.NewKeyVaultKey(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := vaultName + shared.QuerySeparator + keyName
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr == nil {
			t.Error("Expected error when key has no name, but got nil")
		}
	})

	t.Run("Get_NoLinkedResources", func(t *testing.T) {
		key := createAzureKeyMinimal(keyName)

		mockClient := mocks.NewMockKeysClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, vaultName, keyName, nil).Return(
			armkeyvault.KeysClientGetResponse{
				Key: *key,
			}, nil)

		wrapper := manual.NewKeyVaultKey(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := vaultName + shared.QuerySeparator + keyName
		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if len(sdpItem.GetLinkedItemQueries()) != 0 {
			t.Errorf("Expected no linked item queries, got %d", len(sdpItem.GetLinkedItemQueries()))
		}
	})

	t.Run("Search", func(t *testing.T) {
		key1 := createAzureKey("key-1", subscriptionID, resourceGroup, vaultName)
		key2 := createAzureKey("key-2", subscriptionID, resourceGroup, vaultName)

		mockPager := &mockKeysPager{
			pages: []armkeyvault.KeysClientListResponse{
				{
					KeyListResult: armkeyvault.KeyListResult{
						Value: []*armkeyvault.Key{
							{ID: key1.ID, Name: key1.Name, Type: key1.Type, Properties: key1.Properties, Tags: key1.Tags},
							{ID: key2.ID, Name: key2.Name, Type: key2.Type, Properties: key2.Properties, Tags: key2.Tags},
						},
					},
				},
			},
		}

		mockClient := mocks.NewMockKeysClient(ctrl)
		mockClient.EXPECT().NewListPager(resourceGroup, vaultName, nil).Return(mockPager)

		testClient := &testKeysClient{
			MockKeysClient: mockClient,
			pager:          mockPager,
		}

		wrapper := manual.NewKeyVaultKey(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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
			if item.GetType() != azureshared.KeyVaultKey.String() {
				t.Errorf("Expected type %s, got %s", azureshared.KeyVaultKey, item.GetType())
			}
		}
	})

	t.Run("Search_InvalidQueryParts", func(t *testing.T) {
		mockClient := mocks.NewMockKeysClient(ctrl)
		testClient := &testKeysClient{MockKeysClient: mockClient}

		wrapper := manual.NewKeyVaultKey(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		_, qErr := wrapper.Search(ctx, wrapper.Scopes()[0])
		if qErr == nil {
			t.Error("Expected error when providing no query parts, but got nil")
		}
	})

	t.Run("Search_EmptyVaultName", func(t *testing.T) {
		mockClient := mocks.NewMockKeysClient(ctrl)
		testClient := &testKeysClient{MockKeysClient: mockClient}

		wrapper := manual.NewKeyVaultKey(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		_, qErr := wrapper.Search(ctx, wrapper.Scopes()[0], "")
		if qErr == nil {
			t.Error("Expected error when vault name is empty, but got nil")
		}
	})

	t.Run("Search_KeyWithNilName", func(t *testing.T) {
		validKey := createAzureKey("valid-key", subscriptionID, resourceGroup, vaultName)
		mockPager := &mockKeysPager{
			pages: []armkeyvault.KeysClientListResponse{
				{
					KeyListResult: armkeyvault.KeyListResult{
						Value: []*armkeyvault.Key{
							{Name: nil},
							{ID: validKey.ID, Name: validKey.Name, Type: validKey.Type, Properties: validKey.Properties, Tags: validKey.Tags},
						},
					},
				},
			},
		}

		mockClient := mocks.NewMockKeysClient(ctrl)
		mockClient.EXPECT().NewListPager(resourceGroup, vaultName, nil).Return(mockPager)

		testClient := &testKeysClient{
			MockKeysClient: mockClient,
			pager:          mockPager,
		}

		wrapper := manual.NewKeyVaultKey(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support Search operation")
		}

		sdpItems, err := searchable.Search(ctx, wrapper.Scopes()[0], vaultName, true)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if len(sdpItems) != 1 {
			t.Fatalf("Expected 1 item, got: %d", len(sdpItems))
		}

		expectedUniqueAttrValue := shared.CompositeLookupKey(vaultName, "valid-key")
		if sdpItems[0].UniqueAttributeValue() != expectedUniqueAttrValue {
			t.Errorf("Expected unique attribute value %s, got %s", expectedUniqueAttrValue, sdpItems[0].UniqueAttributeValue())
		}
	})

	t.Run("ErrorHandling_Get", func(t *testing.T) {
		expectedErr := errors.New("key not found")

		mockClient := mocks.NewMockKeysClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, vaultName, "nonexistent-key", nil).Return(
			armkeyvault.KeysClientGetResponse{}, expectedErr)

		wrapper := manual.NewKeyVaultKey(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := vaultName + shared.QuerySeparator + "nonexistent-key"
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr == nil {
			t.Error("Expected error when getting non-existent key, but got nil")
		}
	})

	t.Run("ErrorHandling_Search", func(t *testing.T) {
		mockClient := mocks.NewMockKeysClient(ctrl)
		errorPager := &errorKeysPager{}

		mockClient.EXPECT().NewListPager(resourceGroup, vaultName, nil).Return(errorPager)

		testClient := &testKeysClient{
			MockKeysClient: mockClient,
			pager:          errorPager,
		}

		wrapper := manual.NewKeyVaultKey(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support Search operation")
		}

		_, err := searchable.Search(ctx, wrapper.Scopes()[0], vaultName, true)
		if err == nil {
			t.Error("Expected error from pager when NextPage returns an error, but got nil")
		}
	})

	t.Run("InterfaceCompliance", func(t *testing.T) {
		mockClient := mocks.NewMockKeysClient(ctrl)
		testClient := &testKeysClient{MockKeysClient: mockClient}
		wrapper := manual.NewKeyVaultKey(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		if wrapper == nil {
			t.Error("Wrapper should not be nil")
		}

		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())
		_, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Error("Adapter should implement SearchableAdapter interface")
		}
	})

	t.Run("PotentialLinks", func(t *testing.T) {
		mockClient := mocks.NewMockKeysClient(ctrl)
		testClient := &testKeysClient{MockKeysClient: mockClient}
		wrapper := manual.NewKeyVaultKey(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

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
		mockClient := mocks.NewMockKeysClient(ctrl)
		testClient := &testKeysClient{MockKeysClient: mockClient}
		wrapper := manual.NewKeyVaultKey(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		mappings := wrapper.TerraformMappings()
		if len(mappings) == 0 {
			t.Fatal("Expected TerraformMappings to be defined")
		}

		foundIDMapping := false
		for _, mapping := range mappings {
			if mapping.GetTerraformQueryMap() == "azurerm_key_vault_key.id" {
				foundIDMapping = true
				if mapping.GetTerraformMethod() != sdp.QueryMethod_SEARCH {
					t.Errorf("Expected TerraformMethod to be SEARCH for id mapping, got %s", mapping.GetTerraformMethod())
				}
			}
		}
		if !foundIDMapping {
			t.Error("Expected TerraformMappings to include 'azurerm_key_vault_key.id' mapping")
		}
		if len(mappings) != 1 {
			t.Errorf("Expected 1 TerraformMapping, got %d", len(mappings))
		}
	})

	t.Run("IAMPermissions", func(t *testing.T) {
		mockClient := mocks.NewMockKeysClient(ctrl)
		testClient := &testKeysClient{MockKeysClient: mockClient}
		wrapper := manual.NewKeyVaultKey(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		permissions := wrapper.IAMPermissions()
		if len(permissions) == 0 {
			t.Error("Expected IAMPermissions to be defined")
		}
		expectedPermission := "Microsoft.KeyVault/vaults/keys/read"
		if !slices.Contains(permissions, expectedPermission) {
			t.Errorf("Expected IAMPermissions to include %s", expectedPermission)
		}
	})

	t.Run("PredefinedRole", func(t *testing.T) {
		mockClient := mocks.NewMockKeysClient(ctrl)
		testClient := &testKeysClient{MockKeysClient: mockClient}
		wrapper := manual.NewKeyVaultKey(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

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
}

func createAzureKey(keyName, subscriptionID, resourceGroup, vaultName string) *armkeyvault.Key {
	return &armkeyvault.Key{
		ID:   new(fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.KeyVault/vaults/%s/keys/%s", subscriptionID, resourceGroup, vaultName, keyName)),
		Name: new(keyName),
		Type: new("Microsoft.KeyVault/vaults/keys"),
		Tags: map[string]*string{
			"env":     new("test"),
			"project": new("testing"),
		},
		Properties: &armkeyvault.KeyProperties{
			KeyURI: new(fmt.Sprintf("https://%s.vault.azure.net/keys/%s", vaultName, keyName)),
		},
	}
}

func createAzureKeyMinimal(keyName string) *armkeyvault.Key {
	return &armkeyvault.Key{
		Name: new(keyName),
		Type: new("Microsoft.KeyVault/vaults/keys"),
		Tags: map[string]*string{
			"env": new("test"),
		},
		Properties: &armkeyvault.KeyProperties{},
	}
}
