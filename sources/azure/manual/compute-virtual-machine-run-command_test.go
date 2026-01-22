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
	"github.com/overmindtech/cli/sources/azure/clients"
	"github.com/overmindtech/cli/sources/azure/manual"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/azure/shared/mocks"
	"github.com/overmindtech/cli/sources/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

// mockVirtualMachineRunCommandsPager is a simple mock implementation of VirtualMachineRunCommandsPager
type mockVirtualMachineRunCommandsPager struct {
	pages []armcompute.VirtualMachineRunCommandsClientListByVirtualMachineResponse
	index int
}

func (m *mockVirtualMachineRunCommandsPager) More() bool {
	return m.index < len(m.pages)
}

func (m *mockVirtualMachineRunCommandsPager) NextPage(ctx context.Context) (armcompute.VirtualMachineRunCommandsClientListByVirtualMachineResponse, error) {
	if m.index >= len(m.pages) {
		return armcompute.VirtualMachineRunCommandsClientListByVirtualMachineResponse{}, errors.New("no more pages")
	}
	page := m.pages[m.index]
	m.index++
	return page, nil
}

// errorVirtualMachineRunCommandsPager is a mock pager that always returns an error
type errorVirtualMachineRunCommandsPager struct{}

func (e *errorVirtualMachineRunCommandsPager) More() bool {
	return true // Always return true so NextPage will be called
}

func (e *errorVirtualMachineRunCommandsPager) NextPage(ctx context.Context) (armcompute.VirtualMachineRunCommandsClientListByVirtualMachineResponse, error) {
	return armcompute.VirtualMachineRunCommandsClientListByVirtualMachineResponse{}, errors.New("pager error")
}

// testVirtualMachineRunCommandsClient wraps the mock to implement the correct interface
type testVirtualMachineRunCommandsClient struct {
	*mocks.MockVirtualMachineRunCommandsClient
	pager clients.VirtualMachineRunCommandsPager
}

func (t *testVirtualMachineRunCommandsClient) NewListByVirtualMachinePager(resourceGroupName, virtualMachineName string, options *armcompute.VirtualMachineRunCommandsClientListByVirtualMachineOptions) clients.VirtualMachineRunCommandsPager {
	return t.pager
}

func createAzureVirtualMachineRunCommand(runCommandName, vmName string) *armcompute.VirtualMachineRunCommand {
	return &armcompute.VirtualMachineRunCommand{
		Name:     to.Ptr(runCommandName),
		Location: to.Ptr("eastus"),
		Tags: map[string]*string{
			"env":     to.Ptr("test"),
			"project": to.Ptr("testing"),
		},
		Properties: &armcompute.VirtualMachineRunCommandProperties{
			ProvisioningState: to.Ptr("Succeeded"),
		},
	}
}

func createAzureVirtualMachineRunCommandWithBlobURIs(runCommandName, vmName string) *armcompute.VirtualMachineRunCommand {
	runCommand := createAzureVirtualMachineRunCommand(runCommandName, vmName)
	runCommand.Properties.OutputBlobURI = to.Ptr("https://mystorageaccount.blob.core.windows.net/outputcontainer/output.log")
	runCommand.Properties.ErrorBlobURI = to.Ptr("https://mystorageaccount.blob.core.windows.net/errorcontainer/error.log")
	return runCommand
}

func createAzureVirtualMachineRunCommandWithHTTPScriptURI(runCommandName, vmName string) *armcompute.VirtualMachineRunCommand {
	runCommand := createAzureVirtualMachineRunCommand(runCommandName, vmName)
	runCommand.Properties.Source = &armcompute.VirtualMachineRunCommandScriptSource{
		ScriptURI: to.Ptr("https://example.com/scripts/script.sh"),
	}
	return runCommand
}

func createAzureVirtualMachineRunCommandWithAllLinks(runCommandName, vmName string) *armcompute.VirtualMachineRunCommand {
	runCommand := createAzureVirtualMachineRunCommand(runCommandName, vmName)
	runCommand.Properties.OutputBlobURI = to.Ptr("https://mystorageaccount.blob.core.windows.net/outputcontainer/output.log")
	runCommand.Properties.ErrorBlobURI = to.Ptr("https://mystorageaccount.blob.core.windows.net/errorcontainer/error.log")
	runCommand.Properties.Source = &armcompute.VirtualMachineRunCommandScriptSource{
		ScriptURI: to.Ptr("https://mystorageaccount.blob.core.windows.net/scripts/script.sh"),
	}
	return runCommand
}

func TestComputeVirtualMachineRunCommand(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"
	vmName := "test-vm"
	runCommandName := "test-run-command"

	t.Run("Get", func(t *testing.T) {
		runCommand := createAzureVirtualMachineRunCommand(runCommandName, vmName)

		mockClient := mocks.NewMockVirtualMachineRunCommandsClient(ctrl)
		mockClient.EXPECT().GetByVirtualMachine(ctx, resourceGroup, vmName, runCommandName, nil).Return(
			armcompute.VirtualMachineRunCommandsClientGetByVirtualMachineResponse{
				VirtualMachineRunCommand: *runCommand,
			}, nil)

		wrapper := manual.NewComputeVirtualMachineRunCommand(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		scope := subscriptionID + "." + resourceGroup
		query := shared.CompositeLookupKey(vmName, runCommandName)
		sdpItem, qErr := adapter.Get(ctx, scope, query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.ComputeVirtualMachineRunCommand.String() {
			t.Errorf("Expected type %s, got %s", azureshared.ComputeVirtualMachineRunCommand, sdpItem.GetType())
		}

		expectedUniqueAttr := shared.CompositeLookupKey(vmName, runCommandName)
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

	t.Run("Get_WithBlobURIs", func(t *testing.T) {
		runCommand := createAzureVirtualMachineRunCommandWithBlobURIs(runCommandName, vmName)

		mockClient := mocks.NewMockVirtualMachineRunCommandsClient(ctrl)
		mockClient.EXPECT().GetByVirtualMachine(ctx, resourceGroup, vmName, runCommandName, nil).Return(
			armcompute.VirtualMachineRunCommandsClientGetByVirtualMachineResponse{
				VirtualMachineRunCommand: *runCommand,
			}, nil)

		wrapper := manual.NewComputeVirtualMachineRunCommand(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		scope := subscriptionID + "." + resourceGroup
		query := shared.CompositeLookupKey(vmName, runCommandName)
		sdpItem, qErr := adapter.Get(ctx, scope, query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		// Verify linked queries
		linkedQueries := sdpItem.GetLinkedItemQueries()
		if len(linkedQueries) == 0 {
			t.Fatal("Expected linked queries, but got none")
		}

		// Check for Storage Account links (from outputBlobUri and errorBlobUri)
		storageAccountLinks := 0
		blobContainerLinks := 0
		dnsLinks := 0
		httpLinks := 0
		vmLinks := 0

		for _, liq := range linkedQueries {
			switch liq.GetQuery().GetType() {
			case azureshared.StorageAccount.String():
				storageAccountLinks++
				if liq.GetQuery().GetQuery() != "mystorageaccount" {
					t.Errorf("Expected storage account name 'mystorageaccount', got %s", liq.GetQuery().GetQuery())
				}
				if liq.GetQuery().GetMethod() != sdp.QueryMethod_GET {
					t.Errorf("Expected method GET, got %s", liq.GetQuery().GetMethod())
				}
				if liq.GetBlastPropagation().GetIn() != true {
					t.Error("Expected blast propagation In=true for Storage Account")
				}
				if liq.GetBlastPropagation().GetOut() != false {
					t.Error("Expected blast propagation Out=false for Storage Account")
				}
			case azureshared.StorageBlobContainer.String():
				blobContainerLinks++
				expectedQuery := shared.CompositeLookupKey("mystorageaccount", "outputcontainer")
				if liq.GetQuery().GetQuery() != expectedQuery && liq.GetQuery().GetQuery() != shared.CompositeLookupKey("mystorageaccount", "errorcontainer") {
					t.Errorf("Expected blob container query to contain 'mystorageaccount' and container name, got %s", liq.GetQuery().GetQuery())
				}
				if liq.GetQuery().GetMethod() != sdp.QueryMethod_GET {
					t.Errorf("Expected method GET, got %s", liq.GetQuery().GetMethod())
				}
			case stdlib.NetworkDNS.String():
				dnsLinks++
				if liq.GetQuery().GetScope() != "global" {
					t.Errorf("Expected DNS scope 'global', got %s", liq.GetQuery().GetScope())
				}
			case stdlib.NetworkHTTP.String():
				httpLinks++
				if liq.GetQuery().GetScope() != "global" {
					t.Errorf("Expected HTTP scope 'global', got %s", liq.GetQuery().GetScope())
				}
			case azureshared.ComputeVirtualMachine.String():
				vmLinks++
			}
		}

		// We should have at least 2 Storage Account links (from outputBlobUri and errorBlobUri)
		if storageAccountLinks < 2 {
			t.Errorf("Expected at least 2 Storage Account links, got %d", storageAccountLinks)
		}

		// We should have at least 2 Blob Container links
		if blobContainerLinks < 2 {
			t.Errorf("Expected at least 2 Blob Container links, got %d", blobContainerLinks)
		}

		// DNS and HTTP links should NOT be present for blob URIs
		// The StorageBlobContainer would have those links instead
		if dnsLinks > 0 {
			t.Errorf("Expected no DNS links for blob URIs (StorageBlobContainer has them), got %d", dnsLinks)
		}
		if httpLinks > 0 {
			t.Errorf("Expected no HTTP links for blob URIs (StorageBlobContainer has them), got %d", httpLinks)
		}

		// We should have 1 VM link
		if vmLinks != 1 {
			t.Errorf("Expected 1 VM link, got %d", vmLinks)
		}
	})

	t.Run("Get_WithHTTPScriptURI", func(t *testing.T) {
		runCommand := createAzureVirtualMachineRunCommandWithHTTPScriptURI(runCommandName, vmName)

		mockClient := mocks.NewMockVirtualMachineRunCommandsClient(ctrl)
		mockClient.EXPECT().GetByVirtualMachine(ctx, resourceGroup, vmName, runCommandName, nil).Return(
			armcompute.VirtualMachineRunCommandsClientGetByVirtualMachineResponse{
				VirtualMachineRunCommand: *runCommand,
			}, nil)

		wrapper := manual.NewComputeVirtualMachineRunCommand(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		scope := subscriptionID + "." + resourceGroup
		query := shared.CompositeLookupKey(vmName, runCommandName)
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
			}
			if liq.GetQuery().GetType() == stdlib.NetworkDNS.String() {
				hasDNSLink = true
				if liq.GetQuery().GetQuery() != "example.com" {
					t.Errorf("Expected DNS link query 'example.com', got %s", liq.GetQuery().GetQuery())
				}
			}
		}

		if !hasHTTPLink {
			t.Error("Expected HTTP link from script URI, but didn't find one")
		}

		if !hasDNSLink {
			t.Error("Expected DNS link from script URI, but didn't find one")
		}
	})

	t.Run("Get_WithAllLinks", func(t *testing.T) {
		runCommand := createAzureVirtualMachineRunCommandWithAllLinks(runCommandName, vmName)

		mockClient := mocks.NewMockVirtualMachineRunCommandsClient(ctrl)
		mockClient.EXPECT().GetByVirtualMachine(ctx, resourceGroup, vmName, runCommandName, nil).Return(
			armcompute.VirtualMachineRunCommandsClientGetByVirtualMachineResponse{
				VirtualMachineRunCommand: *runCommand,
			}, nil)

		wrapper := manual.NewComputeVirtualMachineRunCommand(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		scope := subscriptionID + "." + resourceGroup
		query := shared.CompositeLookupKey(vmName, runCommandName)
		sdpItem, qErr := adapter.Get(ctx, scope, query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		linkedQueries := sdpItem.GetLinkedItemQueries()
		if len(linkedQueries) == 0 {
			t.Fatal("Expected linked queries, but got none")
		}

		// Should have multiple links: VM, Storage Accounts, Blob Containers, DNS, HTTP
		// The exact count depends on how many unique resources are linked
		if len(linkedQueries) < 5 {
			t.Errorf("Expected at least 5 linked queries, got %d", len(linkedQueries))
		}
	})

	t.Run("Get_ErrorHandling", func(t *testing.T) {
		t.Run("EmptyScope", func(t *testing.T) {
			mockClient := mocks.NewMockVirtualMachineRunCommandsClient(ctrl)
			wrapper := manual.NewComputeVirtualMachineRunCommand(mockClient, subscriptionID, resourceGroup)
			adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

			_, qErr := adapter.Get(ctx, "", shared.CompositeLookupKey(vmName, runCommandName), true)
			if qErr == nil {
				t.Error("Expected error for empty scope, got nil")
			}
		})

		t.Run("WrongQueryPartsCount", func(t *testing.T) {
			mockClient := mocks.NewMockVirtualMachineRunCommandsClient(ctrl)
			wrapper := manual.NewComputeVirtualMachineRunCommand(mockClient, subscriptionID, resourceGroup)
			adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

			scope := subscriptionID + "." + resourceGroup
			_, qErr := adapter.Get(ctx, scope, vmName, true)
			if qErr == nil {
				t.Error("Expected error for wrong query parts count, got nil")
			}
		})

		t.Run("EmptyVirtualMachineName", func(t *testing.T) {
			mockClient := mocks.NewMockVirtualMachineRunCommandsClient(ctrl)
			wrapper := manual.NewComputeVirtualMachineRunCommand(mockClient, subscriptionID, resourceGroup)
			adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

			scope := subscriptionID + "." + resourceGroup
			_, qErr := adapter.Get(ctx, scope, shared.CompositeLookupKey("", runCommandName), true)
			if qErr == nil {
				t.Error("Expected error for empty virtual machine name, got nil")
			}
		})

		t.Run("EmptyRunCommandName", func(t *testing.T) {
			mockClient := mocks.NewMockVirtualMachineRunCommandsClient(ctrl)
			wrapper := manual.NewComputeVirtualMachineRunCommand(mockClient, subscriptionID, resourceGroup)
			adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

			scope := subscriptionID + "." + resourceGroup
			_, qErr := adapter.Get(ctx, scope, shared.CompositeLookupKey(vmName, ""), true)
			if qErr == nil {
				t.Error("Expected error for empty run command name, got nil")
			}
		})

		t.Run("ClientError", func(t *testing.T) {
			mockClient := mocks.NewMockVirtualMachineRunCommandsClient(ctrl)
			mockClient.EXPECT().GetByVirtualMachine(ctx, resourceGroup, vmName, runCommandName, nil).Return(
				armcompute.VirtualMachineRunCommandsClientGetByVirtualMachineResponse{},
				errors.New("client error"))

			wrapper := manual.NewComputeVirtualMachineRunCommand(mockClient, subscriptionID, resourceGroup)
			adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

			scope := subscriptionID + "." + resourceGroup
			_, qErr := adapter.Get(ctx, scope, shared.CompositeLookupKey(vmName, runCommandName), true)
			if qErr == nil {
				t.Error("Expected error from client, got nil")
			}
		})
	})

	t.Run("Search", func(t *testing.T) {
		runCommand1 := createAzureVirtualMachineRunCommand("run-command-1", vmName)
		runCommand2 := createAzureVirtualMachineRunCommand("run-command-2", vmName)

		mockClient := mocks.NewMockVirtualMachineRunCommandsClient(ctrl)
		mockPager := &mockVirtualMachineRunCommandsPager{
			pages: []armcompute.VirtualMachineRunCommandsClientListByVirtualMachineResponse{
				{
					VirtualMachineRunCommandsListResult: armcompute.VirtualMachineRunCommandsListResult{
						Value: []*armcompute.VirtualMachineRunCommand{runCommand1, runCommand2},
					},
				},
			},
			index: 0,
		}

		testClient := &testVirtualMachineRunCommandsClient{
			MockVirtualMachineRunCommandsClient: mockClient,
			pager:                               mockPager,
		}

		wrapper := manual.NewComputeVirtualMachineRunCommand(testClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support Search operation")
		}

		scope := subscriptionID + "." + resourceGroup
		sdpItems, err := searchable.Search(ctx, scope, vmName, true)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if len(sdpItems) != 2 {
			t.Fatalf("Expected 2 items, got: %d", len(sdpItems))
		}

		// Verify items have correct types
		for _, item := range sdpItems {
			if item.GetType() != azureshared.ComputeVirtualMachineRunCommand.String() {
				t.Errorf("Expected type %s, got %s", azureshared.ComputeVirtualMachineRunCommand, item.GetType())
			}
		}
	})

	t.Run("Search_ErrorHandling", func(t *testing.T) {
		t.Run("WrongQueryPartsCount", func(t *testing.T) {
			mockClient := mocks.NewMockVirtualMachineRunCommandsClient(ctrl)
			wrapper := manual.NewComputeVirtualMachineRunCommand(mockClient, subscriptionID, resourceGroup)
			adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

			searchable, ok := adapter.(discovery.SearchableAdapter)
			if !ok {
				t.Fatalf("Adapter does not support Search operation")
			}

			scope := subscriptionID + "." + resourceGroup
			_, err := searchable.Search(ctx, scope, shared.CompositeLookupKey(vmName, runCommandName), true)
			if err == nil {
				t.Error("Expected error for wrong query parts count, got nil")
			}
		})

		t.Run("EmptyVirtualMachineName", func(t *testing.T) {
			mockClient := mocks.NewMockVirtualMachineRunCommandsClient(ctrl)
			wrapper := manual.NewComputeVirtualMachineRunCommand(mockClient, subscriptionID, resourceGroup)
			adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

			searchable, ok := adapter.(discovery.SearchableAdapter)
			if !ok {
				t.Fatalf("Adapter does not support Search operation")
			}

			scope := subscriptionID + "." + resourceGroup
			_, err := searchable.Search(ctx, scope, "", true)
			if err == nil {
				t.Error("Expected error for empty virtual machine name, got nil")
			}
		})

		t.Run("PagerError", func(t *testing.T) {
			mockClient := mocks.NewMockVirtualMachineRunCommandsClient(ctrl)
			errorPager := &errorVirtualMachineRunCommandsPager{}

			testClient := &testVirtualMachineRunCommandsClient{
				MockVirtualMachineRunCommandsClient: mockClient,
				pager:                               errorPager,
			}

			wrapper := manual.NewComputeVirtualMachineRunCommand(testClient, subscriptionID, resourceGroup)
			adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

			searchable, ok := adapter.(discovery.SearchableAdapter)
			if !ok {
				t.Fatalf("Adapter does not support Search operation")
			}

			scope := subscriptionID + "." + resourceGroup
			_, err := searchable.Search(ctx, scope, vmName, true)
			if err == nil {
				t.Error("Expected error from pager when NextPage returns an error, but got nil")
			}
		})

		t.Run("SkipItemsWithoutName", func(t *testing.T) {
			runCommandWithName := createAzureVirtualMachineRunCommand("run-command-1", vmName)
			runCommandWithoutName := &armcompute.VirtualMachineRunCommand{
				Location: to.Ptr("eastus"),
				Properties: &armcompute.VirtualMachineRunCommandProperties{
					ProvisioningState: to.Ptr("Succeeded"),
				},
			}

			mockClient := mocks.NewMockVirtualMachineRunCommandsClient(ctrl)
			mockPager := &mockVirtualMachineRunCommandsPager{
				pages: []armcompute.VirtualMachineRunCommandsClientListByVirtualMachineResponse{
					{
						VirtualMachineRunCommandsListResult: armcompute.VirtualMachineRunCommandsListResult{
							Value: []*armcompute.VirtualMachineRunCommand{runCommandWithName, runCommandWithoutName},
						},
					},
				},
				index: 0,
			}

			testClient := &testVirtualMachineRunCommandsClient{
				MockVirtualMachineRunCommandsClient: mockClient,
				pager:                               mockPager,
			}

			wrapper := manual.NewComputeVirtualMachineRunCommand(testClient, subscriptionID, resourceGroup)
			adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

			searchable, ok := adapter.(discovery.SearchableAdapter)
			if !ok {
				t.Fatalf("Adapter does not support Search operation")
			}

			scope := subscriptionID + "." + resourceGroup
			sdpItems, err := searchable.Search(ctx, scope, vmName, true)
			if err != nil {
				t.Fatalf("Expected no error, got: %v", err)
			}

			// Should only return 1 item (the one with a name)
			if len(sdpItems) != 1 {
				t.Fatalf("Expected 1 item (skipping item without name), got: %d", len(sdpItems))
			}
		})
	})

	t.Run("PotentialLinks", func(t *testing.T) {
		mockClient := mocks.NewMockVirtualMachineRunCommandsClient(ctrl)
		wrapper := manual.NewComputeVirtualMachineRunCommand(mockClient, subscriptionID, resourceGroup)

		potentialLinks := wrapper.PotentialLinks()
		expectedLinks := map[shared.ItemType]bool{
			azureshared.ComputeVirtualMachine:               true,
			azureshared.StorageAccount:                      true,
			azureshared.StorageBlobContainer:                true,
			azureshared.ManagedIdentityUserAssignedIdentity: true,
			stdlib.NetworkHTTP:                              true,
			stdlib.NetworkDNS:                               true,
		}

		for expectedType, expectedValue := range expectedLinks {
			if potentialLinks[expectedType] != expectedValue {
				t.Errorf("Expected PotentialLinks[%s] = %v, got %v", expectedType, expectedValue, potentialLinks[expectedType])
			}
		}

		// Verify all expected links are present
		for expectedType := range expectedLinks {
			if _, exists := potentialLinks[expectedType]; !exists {
				t.Errorf("Expected PotentialLinks to include %s, but it's missing", expectedType)
			}
		}
	})

	t.Run("IAMPermissions", func(t *testing.T) {
		mockClient := mocks.NewMockVirtualMachineRunCommandsClient(ctrl)
		wrapper := manual.NewComputeVirtualMachineRunCommand(mockClient, subscriptionID, resourceGroup)

		permissions := wrapper.IAMPermissions()
		expectedPermission := "Microsoft.Compute/virtualMachines/runCommands/read"

		if len(permissions) != 1 {
			t.Fatalf("Expected 1 permission, got: %d", len(permissions))
		}

		if permissions[0] != expectedPermission {
			t.Errorf("Expected permission '%s', got '%s'", expectedPermission, permissions[0])
		}
	})

	t.Run("TerraformMappings", func(t *testing.T) {
		mockClient := mocks.NewMockVirtualMachineRunCommandsClient(ctrl)
		wrapper := manual.NewComputeVirtualMachineRunCommand(mockClient, subscriptionID, resourceGroup)

		mappings := wrapper.TerraformMappings()
		if len(mappings) != 1 {
			t.Fatalf("Expected 1 Terraform mapping, got: %d", len(mappings))
		}

		if mappings[0].GetTerraformMethod() != sdp.QueryMethod_SEARCH {
			t.Errorf("Expected Terraform method SEARCH, got %s", mappings[0].GetTerraformMethod())
		}

		if mappings[0].GetTerraformQueryMap() != "azurerm_virtual_machine_run_command.id" {
			t.Errorf("Expected Terraform query map 'azurerm_virtual_machine_run_command.id', got '%s'", mappings[0].GetTerraformQueryMap())
		}
	})

	t.Run("GetLookups", func(t *testing.T) {
		mockClient := mocks.NewMockVirtualMachineRunCommandsClient(ctrl)
		wrapper := manual.NewComputeVirtualMachineRunCommand(mockClient, subscriptionID, resourceGroup)

		lookups := wrapper.GetLookups()
		if len(lookups) != 2 {
			t.Fatalf("Expected 2 lookups, got: %d", len(lookups))
		}
	})

	t.Run("SearchLookups", func(t *testing.T) {
		mockClient := mocks.NewMockVirtualMachineRunCommandsClient(ctrl)
		wrapper := manual.NewComputeVirtualMachineRunCommand(mockClient, subscriptionID, resourceGroup)

		searchLookups := wrapper.SearchLookups()
		if len(searchLookups) != 1 {
			t.Fatalf("Expected 1 search lookup set, got: %d", len(searchLookups))
		}

		if len(searchLookups[0]) != 1 {
			t.Fatalf("Expected 1 lookup in search lookup set, got: %d", len(searchLookups[0]))
		}
	})
}
