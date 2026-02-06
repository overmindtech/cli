package manual_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"
	"go.uber.org/mock/gomock"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/manual"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/azure/shared/mocks"
	"github.com/overmindtech/cli/sources/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

func TestComputeVirtualMachineExtension(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"
	vmName := "test-vm"
	extensionName := "test-extension"

	t.Run("Get", func(t *testing.T) {
		extension := createAzureVirtualMachineExtension(extensionName, vmName)

		mockClient := mocks.NewMockVirtualMachineExtensionsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, vmName, extensionName, nil).Return(
			armcompute.VirtualMachineExtensionsClientGetResponse{
				VirtualMachineExtension: *extension,
			}, nil)

		wrapper := manual.NewComputeVirtualMachineExtension(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		scope := subscriptionID + "." + resourceGroup
		query := shared.CompositeLookupKey(vmName, extensionName)
		sdpItem, qErr := adapter.Get(ctx, scope, query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.ComputeVirtualMachineExtension.String() {
			t.Errorf("Expected type %s, got %s", azureshared.ComputeVirtualMachineExtension, sdpItem.GetType())
		}

		expectedUniqueAttr := shared.CompositeLookupKey(vmName, extensionName)
		if sdpItem.UniqueAttributeValue() != expectedUniqueAttr {
			t.Errorf("Expected unique attribute value %s, got %s", expectedUniqueAttr, sdpItem.UniqueAttributeValue())
		}

		if sdpItem.GetTags()["env"] != "test" {
			t.Errorf("Expected tag 'env=test', got: %v", sdpItem.GetTags()["env"])
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					// Virtual Machine (parent resource)
					ExpectedType:   azureshared.ComputeVirtualMachine.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  vmName,
					ExpectedScope:  scope,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
			}

			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("Get_WithKeyVault", func(t *testing.T) {
		extension := createAzureVirtualMachineExtensionWithKeyVault(extensionName, vmName)

		mockClient := mocks.NewMockVirtualMachineExtensionsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, vmName, extensionName, nil).Return(
			armcompute.VirtualMachineExtensionsClientGetResponse{
				VirtualMachineExtension: *extension,
			}, nil)

		wrapper := manual.NewComputeVirtualMachineExtension(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		scope := subscriptionID + "." + resourceGroup
		query := shared.CompositeLookupKey(vmName, extensionName)
		sdpItem, qErr := adapter.Get(ctx, scope, query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		linkedQueries := sdpItem.GetLinkedItemQueries()
		if len(linkedQueries) == 0 {
			t.Fatal("Expected linked queries, but got none")
		}

		hasKeyVaultLink := false
		hasVMLink := false

		for _, liq := range linkedQueries {
			switch liq.GetQuery().GetType() {
			case azureshared.KeyVaultVault.String():
				hasKeyVaultLink = true
				if liq.GetQuery().GetQuery() != "test-keyvault" {
					t.Errorf("Expected Key Vault name 'test-keyvault', got %s", liq.GetQuery().GetQuery())
				}
				if liq.GetQuery().GetMethod() != sdp.QueryMethod_GET {
					t.Errorf("Expected method GET, got %s", liq.GetQuery().GetMethod())
				}
				if liq.GetQuery().GetScope() != scope {
					t.Errorf("Expected scope %s, got %s", scope, liq.GetQuery().GetScope())
				}
				if liq.GetBlastPropagation().GetIn() != true {
					t.Error("Expected blast propagation In=true for Key Vault")
				}
				if liq.GetBlastPropagation().GetOut() != false {
					t.Error("Expected blast propagation Out=false for Key Vault")
				}
			case azureshared.ComputeVirtualMachine.String():
				hasVMLink = true
			}
		}

		if !hasKeyVaultLink {
			t.Error("Expected Key Vault link, but didn't find one")
		}

		if !hasVMLink {
			t.Error("Expected VM link, but didn't find one")
		}
	})

	t.Run("Get_WithKeyVault_DifferentResourceGroup", func(t *testing.T) {
		extension := createAzureVirtualMachineExtension(extensionName, vmName)
		extension.Properties.ProtectedSettingsFromKeyVault = &armcompute.KeyVaultSecretReference{
			SourceVault: &armcompute.SubResource{
				ID: to.Ptr("/subscriptions/test-subscription/resourceGroups/different-rg/providers/Microsoft.KeyVault/vaults/test-keyvault"),
			},
			SecretURL: to.Ptr("https://test-keyvault.vault.azure.net/secrets/test-secret/version"),
		}

		mockClient := mocks.NewMockVirtualMachineExtensionsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, vmName, extensionName, nil).Return(
			armcompute.VirtualMachineExtensionsClientGetResponse{
				VirtualMachineExtension: *extension,
			}, nil)

		wrapper := manual.NewComputeVirtualMachineExtension(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		scope := subscriptionID + "." + resourceGroup
		query := shared.CompositeLookupKey(vmName, extensionName)
		sdpItem, qErr := adapter.Get(ctx, scope, query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		linkedQueries := sdpItem.GetLinkedItemQueries()
		hasKeyVaultLink := false
		hasDNSLink := false
		expectedScope := subscriptionID + ".different-rg"
		expectedDNSName := "test-keyvault.vault.azure.net"

		for _, liq := range linkedQueries {
			if liq.GetQuery().GetType() == azureshared.KeyVaultVault.String() {
				hasKeyVaultLink = true
				if liq.GetQuery().GetScope() != expectedScope {
					t.Errorf("Expected scope %s for Key Vault in different resource group, got %s", expectedScope, liq.GetQuery().GetScope())
				}
			}
			if liq.GetQuery().GetType() == stdlib.NetworkDNS.String() {
				if liq.GetQuery().GetQuery() == expectedDNSName {
					hasDNSLink = true
					if liq.GetQuery().GetScope() != "global" {
						t.Errorf("Expected scope 'global' for DNS link, got %s", liq.GetQuery().GetScope())
					}
					if liq.GetQuery().GetMethod() != sdp.QueryMethod_SEARCH {
						t.Errorf("Expected method SEARCH for DNS link, got %v", liq.GetQuery().GetMethod())
					}
					if liq.GetBlastPropagation().GetIn() != true || liq.GetBlastPropagation().GetOut() != true {
						t.Errorf("Expected blast propagation In: true, Out: true for DNS link, got In: %v, Out: %v", liq.GetBlastPropagation().GetIn(), liq.GetBlastPropagation().GetOut())
					}
				}
			}
		}

		if !hasKeyVaultLink {
			t.Error("Expected Key Vault link, but didn't find one")
		}
		if !hasDNSLink {
			t.Error("Expected DNS link from SecretURL, but didn't find one")
		}
	})

	t.Run("Get_WithSettingsURL", func(t *testing.T) {
		extension := createAzureVirtualMachineExtensionWithSettingsURL(extensionName, vmName)

		mockClient := mocks.NewMockVirtualMachineExtensionsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, vmName, extensionName, nil).Return(
			armcompute.VirtualMachineExtensionsClientGetResponse{
				VirtualMachineExtension: *extension,
			}, nil)

		wrapper := manual.NewComputeVirtualMachineExtension(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		scope := subscriptionID + "." + resourceGroup
		query := shared.CompositeLookupKey(vmName, extensionName)
		sdpItem, qErr := adapter.Get(ctx, scope, query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		linkedQueries := sdpItem.GetLinkedItemQueries()
		hasHTTPLink := false
		hasDNSLink := false

		for _, liq := range linkedQueries {
			if liq.GetQuery().GetType() == stdlib.NetworkHTTP.String() {
				hasHTTPLink = true
				if liq.GetQuery().GetQuery() != "https://example.com/scripts/script.sh" {
					t.Errorf("Expected HTTP link query 'https://example.com/scripts/script.sh', got %s", liq.GetQuery().GetQuery())
				}
				if liq.GetQuery().GetMethod() != sdp.QueryMethod_SEARCH {
					t.Errorf("Expected HTTP method SEARCH, got %s", liq.GetQuery().GetMethod())
				}
				if liq.GetQuery().GetScope() != "global" {
					t.Errorf("Expected HTTP scope 'global', got %s", liq.GetQuery().GetScope())
				}
			}
			if liq.GetQuery().GetType() == stdlib.NetworkDNS.String() {
				hasDNSLink = true
				if liq.GetQuery().GetQuery() != "example.com" {
					t.Errorf("Expected DNS link query 'example.com', got %s", liq.GetQuery().GetQuery())
				}
				if liq.GetQuery().GetMethod() != sdp.QueryMethod_SEARCH {
					t.Errorf("Expected DNS method SEARCH, got %s", liq.GetQuery().GetMethod())
				}
				if liq.GetQuery().GetScope() != "global" {
					t.Errorf("Expected DNS scope 'global', got %s", liq.GetQuery().GetScope())
				}
			}
		}

		if !hasHTTPLink {
			t.Error("Expected HTTP link from settings URL, but didn't find one")
		}

		if !hasDNSLink {
			t.Error("Expected DNS link from settings URL, but didn't find one")
		}
	})

	t.Run("Get_WithSettingsIP", func(t *testing.T) {
		extension := createAzureVirtualMachineExtensionWithSettingsIP(extensionName, vmName)

		mockClient := mocks.NewMockVirtualMachineExtensionsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, vmName, extensionName, nil).Return(
			armcompute.VirtualMachineExtensionsClientGetResponse{
				VirtualMachineExtension: *extension,
			}, nil)

		wrapper := manual.NewComputeVirtualMachineExtension(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		scope := subscriptionID + "." + resourceGroup
		query := shared.CompositeLookupKey(vmName, extensionName)
		sdpItem, qErr := adapter.Get(ctx, scope, query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		linkedQueries := sdpItem.GetLinkedItemQueries()
		hasIPLink := false

		for _, liq := range linkedQueries {
			if liq.GetQuery().GetType() == stdlib.NetworkIP.String() {
				hasIPLink = true
				if liq.GetQuery().GetQuery() != "10.0.0.1" {
					t.Errorf("Expected IP link query '10.0.0.1', got %s", liq.GetQuery().GetQuery())
				}
				if liq.GetQuery().GetMethod() != sdp.QueryMethod_GET {
					t.Errorf("Expected IP method GET, got %s", liq.GetQuery().GetMethod())
				}
				if liq.GetQuery().GetScope() != "global" {
					t.Errorf("Expected IP scope 'global', got %s", liq.GetQuery().GetScope())
				}
			}
		}

		if !hasIPLink {
			t.Error("Expected IP link from settings, but didn't find one")
		}
	})

	t.Run("Get_WithProtectedSettings", func(t *testing.T) {
		extension := createAzureVirtualMachineExtensionWithProtectedSettings(extensionName, vmName)

		mockClient := mocks.NewMockVirtualMachineExtensionsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, vmName, extensionName, nil).Return(
			armcompute.VirtualMachineExtensionsClientGetResponse{
				VirtualMachineExtension: *extension,
			}, nil)

		wrapper := manual.NewComputeVirtualMachineExtension(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		scope := subscriptionID + "." + resourceGroup
		query := shared.CompositeLookupKey(vmName, extensionName)
		sdpItem, qErr := adapter.Get(ctx, scope, query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		linkedQueries := sdpItem.GetLinkedItemQueries()
		hasHTTPLink := false
		hasDNSLink := false

		for _, liq := range linkedQueries {
			if liq.GetQuery().GetType() == stdlib.NetworkHTTP.String() {
				hasHTTPLink = true
				if liq.GetQuery().GetQuery() != "https://api.example.com/v1" {
					t.Errorf("Expected HTTP link query 'https://api.example.com/v1', got %s", liq.GetQuery().GetQuery())
				}
			}
			if liq.GetQuery().GetType() == stdlib.NetworkDNS.String() {
				hasDNSLink = true
				if liq.GetQuery().GetQuery() != "api.example.com" {
					t.Errorf("Expected DNS link query 'api.example.com', got %s", liq.GetQuery().GetQuery())
				}
			}
		}

		if !hasHTTPLink {
			t.Error("Expected HTTP link from protected settings, but didn't find one")
		}

		if !hasDNSLink {
			t.Error("Expected DNS link from protected settings, but didn't find one")
		}
	})

	t.Run("Get_WithAllLinks", func(t *testing.T) {
		extension := createAzureVirtualMachineExtensionWithAllLinks(extensionName, vmName)

		mockClient := mocks.NewMockVirtualMachineExtensionsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, vmName, extensionName, nil).Return(
			armcompute.VirtualMachineExtensionsClientGetResponse{
				VirtualMachineExtension: *extension,
			}, nil)

		wrapper := manual.NewComputeVirtualMachineExtension(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		scope := subscriptionID + "." + resourceGroup
		query := shared.CompositeLookupKey(vmName, extensionName)
		sdpItem, qErr := adapter.Get(ctx, scope, query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		linkedQueries := sdpItem.GetLinkedItemQueries()
		if len(linkedQueries) == 0 {
			t.Fatal("Expected linked queries, but got none")
		}

		// Should have multiple links: VM, Key Vault, HTTP, DNS, IP
		if len(linkedQueries) < 5 {
			t.Errorf("Expected at least 5 linked queries, got %d", len(linkedQueries))
		}

		linkTypes := make(map[string]int)
		for _, liq := range linkedQueries {
			linkTypes[liq.GetQuery().GetType()]++
		}

		if linkTypes[azureshared.ComputeVirtualMachine.String()] != 1 {
			t.Errorf("Expected 1 VM link, got %d", linkTypes[azureshared.ComputeVirtualMachine.String()])
		}

		if linkTypes[azureshared.KeyVaultVault.String()] != 1 {
			t.Errorf("Expected 1 Key Vault link, got %d", linkTypes[azureshared.KeyVaultVault.String()])
		}
	})

	t.Run("Get_ErrorHandling", func(t *testing.T) {
		t.Run("InvalidQueryParts", func(t *testing.T) {
			mockClient := mocks.NewMockVirtualMachineExtensionsClient(ctrl)

			wrapper := manual.NewComputeVirtualMachineExtension(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
			adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

			scope := subscriptionID + "." + resourceGroup

			// Test with too few query parts
			_, qErr := adapter.Get(ctx, scope, "only-vm-name", true)
			if qErr == nil {
				t.Error("Expected error for invalid query parts, got nil")
			}

			// Test with too many query parts
			_, qErr = adapter.Get(ctx, scope, shared.CompositeLookupKey(vmName, extensionName, "extra"), true)
			if qErr == nil {
				t.Error("Expected error for too many query parts, got nil")
			}
		})

		t.Run("EmptyVirtualMachineName", func(t *testing.T) {
			mockClient := mocks.NewMockVirtualMachineExtensionsClient(ctrl)
			wrapper := manual.NewComputeVirtualMachineExtension(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
			adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

			scope := subscriptionID + "." + resourceGroup
			_, qErr := adapter.Get(ctx, scope, shared.CompositeLookupKey("", extensionName), true)
			if qErr == nil {
				t.Error("Expected error for empty virtual machine name, got nil")
			}
		})

		t.Run("EmptyExtensionName", func(t *testing.T) {
			mockClient := mocks.NewMockVirtualMachineExtensionsClient(ctrl)
			wrapper := manual.NewComputeVirtualMachineExtension(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
			adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

			scope := subscriptionID + "." + resourceGroup
			_, qErr := adapter.Get(ctx, scope, shared.CompositeLookupKey(vmName, ""), true)
			if qErr == nil {
				t.Error("Expected error for empty extension name, got nil")
			}
		})

		t.Run("ClientError", func(t *testing.T) {
			mockClient := mocks.NewMockVirtualMachineExtensionsClient(ctrl)
			mockClient.EXPECT().Get(ctx, resourceGroup, vmName, extensionName, nil).Return(
				armcompute.VirtualMachineExtensionsClientGetResponse{},
				errors.New("client error"))

			wrapper := manual.NewComputeVirtualMachineExtension(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
			adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

			scope := subscriptionID + "." + resourceGroup
			query := shared.CompositeLookupKey(vmName, extensionName)
			_, qErr := adapter.Get(ctx, scope, query, true)
			if qErr == nil {
				t.Error("Expected error from client, got nil")
			}
		})
	})

	t.Run("Search", func(t *testing.T) {
		extension1 := createAzureVirtualMachineExtension("extension-1", vmName)
		extension2 := createAzureVirtualMachineExtension("extension-2", vmName)

		mockClient := mocks.NewMockVirtualMachineExtensionsClient(ctrl)
		mockClient.EXPECT().List(ctx, resourceGroup, vmName, nil).Return(
			armcompute.VirtualMachineExtensionsClientListResponse{
				VirtualMachineExtensionsListResult: armcompute.VirtualMachineExtensionsListResult{
					Value: []*armcompute.VirtualMachineExtension{
						extension1,
						extension2,
					},
				},
			}, nil)

		wrapper := manual.NewComputeVirtualMachineExtension(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support Search operation")
		}

		scope := subscriptionID + "." + resourceGroup
		items, qErr := searchable.Search(ctx, scope, vmName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if len(items) != 2 {
			t.Errorf("Expected 2 items, got %d", len(items))
		}

		for _, item := range items {
			if item.GetType() != azureshared.ComputeVirtualMachineExtension.String() {
				t.Errorf("Expected type %s, got %s", azureshared.ComputeVirtualMachineExtension, item.GetType())
			}
		}
	})

	t.Run("Search_ErrorHandling", func(t *testing.T) {
		t.Run("InvalidQueryParts", func(t *testing.T) {
			mockClient := mocks.NewMockVirtualMachineExtensionsClient(ctrl)

			wrapper := manual.NewComputeVirtualMachineExtension(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
			adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

			searchable, ok := adapter.(discovery.SearchableAdapter)
			if !ok {
				t.Fatalf("Adapter does not support Search operation")
			}

			scope := subscriptionID + "." + resourceGroup

			// Test with too many query parts - Search takes a single query string,
			// so we test this at the wrapper level by calling Search directly
			_, qErr := wrapper.Search(ctx, scope, vmName, "extra")
			if qErr == nil {
				t.Error("Expected error for too many query parts, got nil")
			}

			// Test with empty VM name
			_, err := searchable.Search(ctx, scope, "", true)
			if err == nil {
				t.Error("Expected error for empty VM name, got nil")
			}
		})

		t.Run("ClientError", func(t *testing.T) {
			mockClient := mocks.NewMockVirtualMachineExtensionsClient(ctrl)
			mockClient.EXPECT().List(ctx, resourceGroup, vmName, nil).Return(
				armcompute.VirtualMachineExtensionsClientListResponse{},
				errors.New("client error"))

			wrapper := manual.NewComputeVirtualMachineExtension(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
			adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

			searchable, ok := adapter.(discovery.SearchableAdapter)
			if !ok {
				t.Fatalf("Adapter does not support Search operation")
			}

			scope := subscriptionID + "." + resourceGroup
			_, err := searchable.Search(ctx, scope, vmName, true)
			if err == nil {
				t.Error("Expected error from client, got nil")
			}
		})

		t.Run("ExtensionWithoutName", func(t *testing.T) {
			extension := createAzureVirtualMachineExtension(extensionName, vmName)
			extension.Name = nil // Extension without name should be skipped

			mockClient := mocks.NewMockVirtualMachineExtensionsClient(ctrl)
			mockClient.EXPECT().List(ctx, resourceGroup, vmName, nil).Return(
				armcompute.VirtualMachineExtensionsClientListResponse{
					VirtualMachineExtensionsListResult: armcompute.VirtualMachineExtensionsListResult{
						Value: []*armcompute.VirtualMachineExtension{
							extension,
						},
					},
				}, nil)

			wrapper := manual.NewComputeVirtualMachineExtension(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
			adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

			searchable, ok := adapter.(discovery.SearchableAdapter)
			if !ok {
				t.Fatalf("Adapter does not support Search operation")
			}

			scope := subscriptionID + "." + resourceGroup
			items, qErr := searchable.Search(ctx, scope, vmName, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			// Extension without name should be skipped
			if len(items) != 0 {
				t.Errorf("Expected 0 items (extension without name should be skipped), got %d", len(items))
			}
		})
	})

	t.Run("PotentialLinks", func(t *testing.T) {
		mockClient := mocks.NewMockVirtualMachineExtensionsClient(ctrl)
		wrapper := manual.NewComputeVirtualMachineExtension(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		links := wrapper.PotentialLinks()

		expectedLinks := map[shared.ItemType]bool{
			azureshared.ComputeVirtualMachine: true,
			azureshared.KeyVaultVault:         true,
			stdlib.NetworkHTTP:                true,
			stdlib.NetworkDNS:                 true,
			stdlib.NetworkIP:                  true,
		}

		for expectedType, expectedValue := range expectedLinks {
			if links[expectedType] != expectedValue {
				t.Errorf("Expected PotentialLinks[%s] = %v, got %v", expectedType, expectedValue, links[expectedType])
			}
		}
	})

	t.Run("GetLookups", func(t *testing.T) {
		mockClient := mocks.NewMockVirtualMachineExtensionsClient(ctrl)
		wrapper := manual.NewComputeVirtualMachineExtension(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		lookups := wrapper.GetLookups()
		if len(lookups) != 2 {
			t.Errorf("Expected 2 lookups, got %d", len(lookups))
		}

		// Verify the first lookup is for the virtual machine name
		if lookups[0].ItemType.String() != azureshared.ComputeVirtualMachine.String() {
			t.Errorf("Expected first lookup item type %s, got %s", azureshared.ComputeVirtualMachine, lookups[0].ItemType)
		}

		// Verify the second lookup is for the extension name
		if lookups[1].ItemType.String() != azureshared.ComputeVirtualMachineExtension.String() {
			t.Errorf("Expected second lookup item type %s, got %s", azureshared.ComputeVirtualMachineExtension, lookups[1].ItemType)
		}
	})

	t.Run("SearchLookups", func(t *testing.T) {
		mockClient := mocks.NewMockVirtualMachineExtensionsClient(ctrl)
		wrapper := manual.NewComputeVirtualMachineExtension(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		searchLookups := wrapper.SearchLookups()
		if len(searchLookups) != 1 {
			t.Errorf("Expected 1 search lookup, got %d", len(searchLookups))
		}

		if len(searchLookups[0]) != 1 {
			t.Errorf("Expected 1 lookup in search lookups, got %d", len(searchLookups[0]))
		}

		// Verify the lookup is for the virtual machine name
		if searchLookups[0][0].ItemType.String() != azureshared.ComputeVirtualMachine.String() {
			t.Errorf("Expected search lookup item type %s, got %s", azureshared.ComputeVirtualMachine, searchLookups[0][0].ItemType)
		}
	})

	t.Run("TerraformMappings", func(t *testing.T) {
		mockClient := mocks.NewMockVirtualMachineExtensionsClient(ctrl)
		wrapper := manual.NewComputeVirtualMachineExtension(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		mappings := wrapper.TerraformMappings()
		if len(mappings) != 1 {
			t.Errorf("Expected 1 Terraform mapping, got %d", len(mappings))
		}

		if mappings[0].GetTerraformMethod() != sdp.QueryMethod_SEARCH {
			t.Errorf("Expected Terraform method SEARCH, got %s", mappings[0].GetTerraformMethod())
		}

		if mappings[0].GetTerraformQueryMap() != "azurerm_virtual_machine_extension.id" {
			t.Errorf("Expected Terraform query map 'azurerm_virtual_machine_extension.id', got %s", mappings[0].GetTerraformQueryMap())
		}
	})

	t.Run("IAMPermissions", func(t *testing.T) {
		mockClient := mocks.NewMockVirtualMachineExtensionsClient(ctrl)
		wrapper := manual.NewComputeVirtualMachineExtension(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		permissions := wrapper.IAMPermissions()
		if len(permissions) != 1 {
			t.Errorf("Expected 1 IAM permission, got %d", len(permissions))
		}

		expectedPermission := "Microsoft.Compute/virtualMachines/extensions/read"
		if permissions[0] != expectedPermission {
			t.Errorf("Expected IAM permission '%s', got '%s'", expectedPermission, permissions[0])
		}
	})

	t.Run("PredefinedRole", func(t *testing.T) {
		mockClient := mocks.NewMockVirtualMachineExtensionsClient(ctrl)
		wrapper := manual.NewComputeVirtualMachineExtension(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		// PredefinedRole is available on the wrapper, not the adapter
		role := wrapper.(interface{ PredefinedRole() string }).PredefinedRole()
		expectedRole := "Reader"
		if role != expectedRole {
			t.Errorf("Expected predefined role '%s', got '%s'", expectedRole, role)
		}
	})
}

func createAzureVirtualMachineExtension(extensionName, vmName string) *armcompute.VirtualMachineExtension {
	return &armcompute.VirtualMachineExtension{
		Name:     to.Ptr(extensionName),
		Location: to.Ptr("eastus"),
		Type:     to.Ptr("Microsoft.Compute/virtualMachines/extensions"),
		Tags: map[string]*string{
			"env":     to.Ptr("test"),
			"project": to.Ptr("testing"),
		},
		Properties: &armcompute.VirtualMachineExtensionProperties{
			Publisher:          to.Ptr("Microsoft.Compute"),
			Type:               to.Ptr("CustomScriptExtension"),
			TypeHandlerVersion: to.Ptr("1.10"),
			ProvisioningState:  to.Ptr("Succeeded"),
		},
	}
}

func createAzureVirtualMachineExtensionWithKeyVault(extensionName, vmName string) *armcompute.VirtualMachineExtension {
	extension := createAzureVirtualMachineExtension(extensionName, vmName)
	extension.Properties.ProtectedSettingsFromKeyVault = &armcompute.KeyVaultSecretReference{
		SourceVault: &armcompute.SubResource{
			ID: to.Ptr("/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.KeyVault/vaults/test-keyvault"),
		},
	}
	return extension
}

func createAzureVirtualMachineExtensionWithSettingsURL(extensionName, vmName string) *armcompute.VirtualMachineExtension {
	extension := createAzureVirtualMachineExtension(extensionName, vmName)
	extension.Properties.Settings = map[string]interface{}{
		"fileUris": []interface{}{
			"https://example.com/scripts/script.sh",
		},
		"commandToExecute": "bash script.sh",
	}
	return extension
}

func createAzureVirtualMachineExtensionWithSettingsIP(extensionName, vmName string) *armcompute.VirtualMachineExtension {
	extension := createAzureVirtualMachineExtension(extensionName, vmName)
	extension.Properties.Settings = map[string]interface{}{
		"serverIP": "10.0.0.1",
		"port":     8080,
	}
	return extension
}

func createAzureVirtualMachineExtensionWithProtectedSettings(extensionName, vmName string) *armcompute.VirtualMachineExtension {
	extension := createAzureVirtualMachineExtension(extensionName, vmName)
	extension.Properties.ProtectedSettings = map[string]interface{}{
		"storageAccountName": "mystorageaccount",
		"storageAccountKey":  "secret-key",
		"endpoint":           "https://api.example.com/v1",
	}
	return extension
}

func createAzureVirtualMachineExtensionWithAllLinks(extensionName, vmName string) *armcompute.VirtualMachineExtension {
	extension := createAzureVirtualMachineExtension(extensionName, vmName)
	extension.Properties.ProtectedSettingsFromKeyVault = &armcompute.KeyVaultSecretReference{
		SourceVault: &armcompute.SubResource{
			ID: to.Ptr("/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.KeyVault/vaults/test-keyvault"),
		},
	}
	extension.Properties.Settings = map[string]interface{}{
		"fileUris": []interface{}{
			"https://example.com/scripts/script.sh",
		},
		"serverIP": "10.0.0.1",
	}
	extension.Properties.ProtectedSettings = map[string]interface{}{
		"endpoint": "https://api.example.com/v1",
	}
	return extension
}
