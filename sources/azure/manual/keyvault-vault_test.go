package manual_test

import (
	"context"
	"errors"
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

// mockVaultsPager is a simple mock implementation of VaultsPager
type mockVaultsPager struct {
	pages []armkeyvault.VaultsClientListByResourceGroupResponse
	index int
}

func (m *mockVaultsPager) More() bool {
	return m.index < len(m.pages)
}

func (m *mockVaultsPager) NextPage(ctx context.Context) (armkeyvault.VaultsClientListByResourceGroupResponse, error) {
	if m.index >= len(m.pages) {
		return armkeyvault.VaultsClientListByResourceGroupResponse{}, errors.New("no more pages")
	}
	page := m.pages[m.index]
	m.index++
	return page, nil
}

// errorVaultsPager is a mock pager that always returns an error
type errorVaultsPager struct{}

func (e *errorVaultsPager) More() bool {
	return true // Always return true so NextPage will be called
}

func (e *errorVaultsPager) NextPage(ctx context.Context) (armkeyvault.VaultsClientListByResourceGroupResponse, error) {
	return armkeyvault.VaultsClientListByResourceGroupResponse{}, errors.New("pager error")
}

// testVaultsClient wraps the mock to implement the correct interface
type testVaultsClient struct {
	*mocks.MockVaultsClient
	pager clients.VaultsPager
}

func (t *testVaultsClient) NewListByResourceGroupPager(resourceGroupName string, options *armkeyvault.VaultsClientListByResourceGroupOptions) clients.VaultsPager {
	// Call the mock to satisfy expectations
	t.MockVaultsClient.NewListByResourceGroupPager(resourceGroupName, options)
	return t.pager
}

func TestKeyVaultVault(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"
	vaultName := "test-keyvault"

	t.Run("Get", func(t *testing.T) {
		vault := createAzureKeyVault(vaultName, subscriptionID, resourceGroup)

		mockClient := mocks.NewMockVaultsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, vaultName, nil).Return(
			armkeyvault.VaultsClientGetResponse{
				Vault: *vault,
			}, nil)

		wrapper := manual.NewKeyVaultVault(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], vaultName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.KeyVaultVault.String() {
			t.Errorf("Expected type %s, got %s", azureshared.KeyVaultVault, sdpItem.GetType())
		}

		if sdpItem.GetUniqueAttribute() != "name" {
			t.Errorf("Expected unique attribute 'name', got %s", sdpItem.GetUniqueAttribute())
		}

		if sdpItem.UniqueAttributeValue() != vaultName {
			t.Errorf("Expected unique attribute value %s, got %s", vaultName, sdpItem.UniqueAttributeValue())
		}

		if sdpItem.GetTags()["env"] != "test" {
			t.Errorf("Expected tag 'env=test', got: %v", sdpItem.GetTags()["env"])
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
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
					// Managed HSM (GET) - different resource group
					ExpectedType:   azureshared.KeyVaultManagedHSM.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-managed-hsm",
					ExpectedScope:  subscriptionID + ".hsm-rg",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					// stdlib.NetworkIP (GET) - from NetworkACLs IPRules
					ExpectedType:   stdlib.NetworkIP.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "192.168.1.100",
					ExpectedScope:  "global",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					// stdlib.NetworkIP (GET) - from NetworkACLs IPRules (CIDR range)
					ExpectedType:   stdlib.NetworkIP.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "10.0.0.0/24",
					ExpectedScope:  "global",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					// stdlib.NetworkHTTP (SEARCH) - from VaultURI
					ExpectedType:   stdlib.NetworkHTTP.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  "https://test-keyvault.vault.azure.net/",
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
		mockClient := mocks.NewMockVaultsClient(ctrl)

		wrapper := manual.NewKeyVaultVault(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		// Test with empty name
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "", true)
		if qErr == nil {
			t.Error("Expected error when getting vault with empty name, but got nil")
		}
	})

	t.Run("Get_NoName", func(t *testing.T) {
		vault := &armkeyvault.Vault{
			Name: nil, // No name field
			Properties: &armkeyvault.VaultProperties{
				TenantID: to.Ptr("test-tenant-id"),
			},
		}

		mockClient := mocks.NewMockVaultsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, vaultName, nil).Return(
			armkeyvault.VaultsClientGetResponse{
				Vault: *vault,
			}, nil)

		wrapper := manual.NewKeyVaultVault(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], vaultName, true)
		if qErr == nil {
			t.Error("Expected error when vault has no name, but got nil")
		}
	})

	t.Run("Get_NoLinkedResources", func(t *testing.T) {
		vault := createAzureKeyVaultMinimal(vaultName)

		mockClient := mocks.NewMockVaultsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, vaultName, nil).Return(
			armkeyvault.VaultsClientGetResponse{
				Vault: *vault,
			}, nil)

		wrapper := manual.NewKeyVaultVault(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], vaultName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		// Should have no linked item queries
		if len(sdpItem.GetLinkedItemQueries()) != 0 {
			t.Errorf("Expected no linked item queries, got %d", len(sdpItem.GetLinkedItemQueries()))
		}
	})

	t.Run("List", func(t *testing.T) {
		vault1 := createAzureKeyVault("test-keyvault-1", subscriptionID, resourceGroup)
		vault2 := createAzureKeyVault("test-keyvault-2", subscriptionID, resourceGroup)

		mockPager := &mockVaultsPager{
			pages: []armkeyvault.VaultsClientListByResourceGroupResponse{
				{
					VaultListResult: armkeyvault.VaultListResult{
						Value: []*armkeyvault.Vault{vault1, vault2},
					},
				},
			},
		}

		mockClient := mocks.NewMockVaultsClient(ctrl)
		mockClient.EXPECT().NewListByResourceGroupPager(resourceGroup, nil).Return(mockPager)

		testClient := &testVaultsClient{
			MockVaultsClient: mockClient,
			pager:            mockPager,
		}

		wrapper := manual.NewKeyVaultVault(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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

		// Verify first item
		if sdpItems[0].UniqueAttributeValue() != "test-keyvault-1" {
			t.Errorf("Expected first item name 'test-keyvault-1', got %s", sdpItems[0].UniqueAttributeValue())
		}

		// Verify second item
		if sdpItems[1].UniqueAttributeValue() != "test-keyvault-2" {
			t.Errorf("Expected second item name 'test-keyvault-2', got %s", sdpItems[1].UniqueAttributeValue())
		}
	})

	t.Run("List_Error", func(t *testing.T) {
		errorPager := &errorVaultsPager{}

		mockClient := mocks.NewMockVaultsClient(ctrl)
		mockClient.EXPECT().NewListByResourceGroupPager(resourceGroup, nil).Return(errorPager)

		testClient := &testVaultsClient{
			MockVaultsClient: mockClient,
			pager:            errorPager,
		}

		wrapper := manual.NewKeyVaultVault(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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

	t.Run("Get_Error", func(t *testing.T) {
		mockClient := mocks.NewMockVaultsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, vaultName, nil).Return(
			armkeyvault.VaultsClientGetResponse{},
			errors.New("client error"))

		wrapper := manual.NewKeyVaultVault(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], vaultName, true)
		if qErr == nil {
			t.Error("Expected error when client returns error, but got nil")
		}
	})

	t.Run("CrossResourceGroupScopes", func(t *testing.T) {
		// Test that linked resources in different resource groups use correct scopes
		vault := createAzureKeyVaultCrossRG(vaultName, subscriptionID, resourceGroup)

		mockClient := mocks.NewMockVaultsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, vaultName, nil).Return(
			armkeyvault.VaultsClientGetResponse{
				Vault: *vault,
			}, nil)

		wrapper := manual.NewKeyVaultVault(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], vaultName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		// Verify that linked resources use their own scopes, not the vault's scope
		foundDifferentScope := false
		for _, linkedQuery := range sdpItem.GetLinkedItemQueries() {
			scope := linkedQuery.GetQuery().GetScope()
			if scope != subscriptionID+"."+resourceGroup {
				foundDifferentScope = true
				// Verify the scope format is correct
				if scope != subscriptionID+".different-rg" && scope != subscriptionID+".hsm-rg" {
					t.Errorf("Unexpected scope format: %s", scope)
				}
			}
		}

		if !foundDifferentScope {
			t.Error("Expected to find at least one linked item query with a different scope, but all used default scope")
		}
	})

	t.Run("PotentialLinks", func(t *testing.T) {
		mockClient := mocks.NewMockVaultsClient(ctrl)
		wrapper := manual.NewKeyVaultVault(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		links := wrapper.PotentialLinks()
		if len(links) == 0 {
			t.Error("Expected potential links to be defined")
		}

		expectedLinks := map[shared.ItemType]bool{
			azureshared.NetworkPrivateEndpoint: true,
			azureshared.NetworkSubnet:          true,
			azureshared.KeyVaultManagedHSM:     true,
			stdlib.NetworkIP:                   true,
			stdlib.NetworkHTTP:                 true,
		}
		for expectedType, expectedValue := range expectedLinks {
			if links[expectedType] != expectedValue {
				t.Errorf("Expected PotentialLinks[%s] = %v, got %v", expectedType.String(), expectedValue, links[expectedType])
			}
		}
	})
}

// createAzureKeyVault creates a mock Azure Key Vault with linked resources
func createAzureKeyVault(vaultName, subscriptionID, resourceGroup string) *armkeyvault.Vault {
	return &armkeyvault.Vault{
		Name:     to.Ptr(vaultName),
		Location: to.Ptr("eastus"),
		Tags: map[string]*string{
			"env":     to.Ptr("test"),
			"project": to.Ptr("testing"),
		},
		Properties: &armkeyvault.VaultProperties{
			TenantID: to.Ptr("test-tenant-id"),
			// Private Endpoint Connections
			PrivateEndpointConnections: []*armkeyvault.PrivateEndpointConnectionItem{
				{
					Properties: &armkeyvault.PrivateEndpointConnectionProperties{
						PrivateEndpoint: &armkeyvault.PrivateEndpoint{
							ID: to.Ptr("/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/privateEndpoints/test-private-endpoint"),
						},
					},
				},
				{
					Properties: &armkeyvault.PrivateEndpointConnectionProperties{
						PrivateEndpoint: &armkeyvault.PrivateEndpoint{
							ID: to.Ptr("/subscriptions/" + subscriptionID + "/resourceGroups/different-rg/providers/Microsoft.Network/privateEndpoints/test-private-endpoint-diff-rg"),
						},
					},
				},
			},
			// Network ACLs with Virtual Network Rules and IP Rules
			NetworkACLs: &armkeyvault.NetworkRuleSet{
				VirtualNetworkRules: []*armkeyvault.VirtualNetworkRule{
					{
						ID: to.Ptr("/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/test-subnet"),
					},
					{
						ID: to.Ptr("/subscriptions/" + subscriptionID + "/resourceGroups/different-rg/providers/Microsoft.Network/virtualNetworks/test-vnet-diff-rg/subnets/test-subnet-diff-rg"),
					},
				},
				IPRules: []*armkeyvault.IPRule{
					{Value: to.Ptr("192.168.1.100")},
					{Value: to.Ptr("10.0.0.0/24")},
				},
			},
			// Vault URI for keys and secrets operations
			VaultURI: to.Ptr("https://" + vaultName + ".vault.azure.net/"),
			// Managed HSM Pool Resource ID
			HsmPoolResourceID: to.Ptr("/subscriptions/" + subscriptionID + "/resourceGroups/hsm-rg/providers/Microsoft.KeyVault/managedHSMs/test-managed-hsm"),
		},
	}
}

// createAzureKeyVaultMinimal creates a minimal mock Azure Key Vault without linked resources
func createAzureKeyVaultMinimal(vaultName string) *armkeyvault.Vault {
	return &armkeyvault.Vault{
		Name:     to.Ptr(vaultName),
		Location: to.Ptr("eastus"),
		Tags: map[string]*string{
			"env": to.Ptr("test"),
		},
		Properties: &armkeyvault.VaultProperties{
			TenantID: to.Ptr("test-tenant-id"),
		},
	}
}

// createAzureKeyVaultCrossRG creates a mock Azure Key Vault with linked resources in different resource groups
func createAzureKeyVaultCrossRG(vaultName, subscriptionID, resourceGroup string) *armkeyvault.Vault {
	return &armkeyvault.Vault{
		Name:     to.Ptr(vaultName),
		Location: to.Ptr("eastus"),
		Tags: map[string]*string{
			"env": to.Ptr("test"),
		},
		Properties: &armkeyvault.VaultProperties{
			TenantID: to.Ptr("test-tenant-id"),
			// Private Endpoint in different resource group
			PrivateEndpointConnections: []*armkeyvault.PrivateEndpointConnectionItem{
				{
					Properties: &armkeyvault.PrivateEndpointConnectionProperties{
						PrivateEndpoint: &armkeyvault.PrivateEndpoint{
							ID: to.Ptr("/subscriptions/" + subscriptionID + "/resourceGroups/different-rg/providers/Microsoft.Network/privateEndpoints/test-pe-diff-rg"),
						},
					},
				},
			},
			// Subnet in different resource group
			NetworkACLs: &armkeyvault.NetworkRuleSet{
				VirtualNetworkRules: []*armkeyvault.VirtualNetworkRule{
					{
						ID: to.Ptr("/subscriptions/" + subscriptionID + "/resourceGroups/different-rg/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/test-subnet"),
					},
				},
			},
			// Managed HSM in different resource group
			HsmPoolResourceID: to.Ptr("/subscriptions/" + subscriptionID + "/resourceGroups/hsm-rg/providers/Microsoft.KeyVault/managedHSMs/test-managed-hsm"),
		},
	}
}
