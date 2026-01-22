package manual_test

import (
	"context"
	"errors"
	"sync"
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
)

func TestComputeVirtualMachine(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"

	t.Run("Get", func(t *testing.T) {
		vmName := "test-vm"
		vm := createAzureVirtualMachine(vmName, "Succeeded")

		mockClient := mocks.NewMockVirtualMachinesClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, vmName, nil).Return(
			armcompute.VirtualMachinesClientGetResponse{
				VirtualMachine: *vm,
			}, nil)

		wrapper := manual.NewComputeVirtualMachine(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], vmName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.ComputeVirtualMachine.String() {
			t.Errorf("Expected type %s, got %s", azureshared.ComputeVirtualMachine, sdpItem.GetType())
		}

		if sdpItem.GetUniqueAttribute() != "name" {
			t.Errorf("Expected unique attribute 'name', got %s", sdpItem.GetUniqueAttribute())
		}

		if sdpItem.UniqueAttributeValue() != vmName {
			t.Errorf("Expected unique attribute value %s, got %s", vmName, sdpItem.UniqueAttributeValue())
		}

		if sdpItem.GetTags()["env"] != "test" {
			t.Errorf("Expected tag 'env=test', got: %v", sdpItem.GetTags()["env"])
		}

		if sdpItem.GetHealth() != sdp.Health_HEALTH_OK {
			t.Errorf("Expected health OK, got: %s", sdpItem.GetHealth())
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					// osDisk.managedDisk.id
					ExpectedType:   azureshared.ComputeDisk.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "os-disk",
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					// dataDisks[0].managedDisk.id
					ExpectedType:   azureshared.ComputeDisk.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "data-disk-1",
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					// networkInterfaces[0].id
					ExpectedType:   azureshared.NetworkNetworkInterface.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-nic",
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					// availabilitySet.id
					ExpectedType:   azureshared.ComputeAvailabilitySet.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-avset",
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					// Resources[0] (VM Extension) - uses composite lookup key
					ExpectedType:   azureshared.ComputeVirtualMachineExtension.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey(vmName, "CustomScriptExtension"),
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  false,
						Out: true,
					},
				},
				{
					// Run commands - always linked via SEARCH
					ExpectedType:   azureshared.ComputeVirtualMachineRunCommand.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  vmName,
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  false,
						Out: true,
					},
				},
			}

			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("HealthCheck", func(t *testing.T) {
		type testCase struct {
			name              string
			provisioningState string
			expectedHealth    sdp.Health
		}

		testCases := []testCase{
			{
				name:              "Succeeded",
				provisioningState: "Succeeded",
				expectedHealth:    sdp.Health_HEALTH_OK,
			},
			{
				name:              "Creating",
				provisioningState: "Creating",
				expectedHealth:    sdp.Health_HEALTH_PENDING,
			},
			{
				name:              "Updating",
				provisioningState: "Updating",
				expectedHealth:    sdp.Health_HEALTH_PENDING,
			},
			{
				name:              "Migrating",
				provisioningState: "Migrating",
				expectedHealth:    sdp.Health_HEALTH_PENDING,
			},
			{
				name:              "Failed",
				provisioningState: "Failed",
				expectedHealth:    sdp.Health_HEALTH_ERROR,
			},
			{
				name:              "Deleting",
				provisioningState: "Deleting",
				expectedHealth:    sdp.Health_HEALTH_ERROR,
			},
			{
				name:              "Unknown",
				provisioningState: "Unknown",
				expectedHealth:    sdp.Health_HEALTH_UNKNOWN,
			},
		}

		mockClient := mocks.NewMockVirtualMachinesClient(ctrl)

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				vm := createAzureVirtualMachine("test-vm", tc.provisioningState)

				mockClient.EXPECT().Get(ctx, resourceGroup, "test-vm", nil).Return(
					armcompute.VirtualMachinesClientGetResponse{
						VirtualMachine: *vm,
					}, nil)

				wrapper := manual.NewComputeVirtualMachine(mockClient, subscriptionID, resourceGroup)
				adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

				sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "test-vm", true)
				if qErr != nil {
					t.Fatalf("Expected no error, got: %v", qErr)
				}

				if sdpItem.GetHealth() != tc.expectedHealth {
					t.Fatalf("Expected health %s, got: %s", tc.expectedHealth, sdpItem.GetHealth())
				}
			})
		}
	})

	t.Run("List", func(t *testing.T) {
		vm1 := createAzureVirtualMachine("test-vm-1", "Succeeded")
		vm2 := createAzureVirtualMachine("test-vm-2", "Succeeded")

		mockClient := mocks.NewMockVirtualMachinesClient(ctrl)
		mockPager := mocks.NewMockVirtualMachinesPager(ctrl)

		// Setup pager expectations
		gomock.InOrder(
			mockPager.EXPECT().More().Return(true),
			mockPager.EXPECT().NextPage(ctx).Return(
				armcompute.VirtualMachinesClientListResponse{
					VirtualMachineListResult: armcompute.VirtualMachineListResult{
						Value: []*armcompute.VirtualMachine{vm1, vm2},
					},
				}, nil),
			mockPager.EXPECT().More().Return(false),
		)

		mockClient.EXPECT().NewListPager(resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewComputeVirtualMachine(mockClient, subscriptionID, resourceGroup)
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

	t.Run("ListStream", func(t *testing.T) {
		vm1 := createAzureVirtualMachine("test-vm-1", "Succeeded")
		vm2 := createAzureVirtualMachine("test-vm-2", "Succeeded")

		mockClient := mocks.NewMockVirtualMachinesClient(ctrl)
		mockPager := mocks.NewMockVirtualMachinesPager(ctrl)

		// Setup pager expectations
		gomock.InOrder(
			mockPager.EXPECT().More().Return(true),
			mockPager.EXPECT().NextPage(ctx).Return(
				armcompute.VirtualMachinesClientListResponse{
					VirtualMachineListResult: armcompute.VirtualMachineListResult{
						Value: []*armcompute.VirtualMachine{vm1, vm2},
					},
				}, nil),
			mockPager.EXPECT().More().Return(false),
		)

		mockClient.EXPECT().NewListPager(resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewComputeVirtualMachine(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		wg := &sync.WaitGroup{}
		wg.Add(2) // we added two items

		var items []*sdp.Item
		mockItemHandler := func(item *sdp.Item) {
			items = append(items, item)
			wg.Done() // signal that we processed an item
		}

		var errs []error
		mockErrorHandler := func(err error) {
			errs = append(errs, err)
		}

		stream := discovery.NewQueryResultStream(mockItemHandler, mockErrorHandler)

		// Check if adapter supports list streaming
		listStreamable, ok := adapter.(discovery.ListStreamableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support ListStream operation")
		}

		listStreamable.ListStream(ctx, wrapper.Scopes()[0], true, stream)
		wg.Wait()

		if len(errs) != 0 {
			t.Fatalf("Expected no errors, got: %v", errs)
		}

		if len(items) != 2 {
			t.Fatalf("Expected 2 items, got: %d", len(items))
		}

		// Verify adapter doesn't support SearchStream
		_, ok = adapter.(discovery.SearchStreamableAdapter)
		if ok {
			t.Fatalf("Adapter should not support SearchStream operation")
		}
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		expectedErr := errors.New("VM not found")

		mockClient := mocks.NewMockVirtualMachinesClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, "nonexistent-vm", nil).Return(
			armcompute.VirtualMachinesClientGetResponse{}, expectedErr)

		wrapper := manual.NewComputeVirtualMachine(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "nonexistent-vm", true)
		if qErr == nil {
			t.Error("Expected error when getting non-existent VM, but got nil")
		}
	})
}

// createAzureVirtualMachine creates a mock Azure VM for testing
func createAzureVirtualMachine(vmName, provisioningState string) *armcompute.VirtualMachine {
	return &armcompute.VirtualMachine{
		Name:     to.Ptr(vmName),
		Location: to.Ptr("eastus"),
		Tags: map[string]*string{
			"env":     to.Ptr("test"),
			"project": to.Ptr("testing"),
		},
		Properties: &armcompute.VirtualMachineProperties{
			ProvisioningState: to.Ptr(provisioningState),
			StorageProfile: &armcompute.StorageProfile{
				OSDisk: &armcompute.OSDisk{
					Name: to.Ptr("os-disk"),
					ManagedDisk: &armcompute.ManagedDiskParameters{
						ID: to.Ptr("/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.Compute/disks/os-disk"),
					},
				},
				DataDisks: []*armcompute.DataDisk{
					{
						Name: to.Ptr("data-disk-1"),
						ManagedDisk: &armcompute.ManagedDiskParameters{
							ID: to.Ptr("/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.Compute/disks/data-disk-1"),
						},
					},
				},
			},
			NetworkProfile: &armcompute.NetworkProfile{
				NetworkInterfaces: []*armcompute.NetworkInterfaceReference{
					{
						ID: to.Ptr("/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.Network/networkInterfaces/test-nic"),
					},
				},
			},
			AvailabilitySet: &armcompute.SubResource{
				ID: to.Ptr("/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.Compute/availabilitySets/test-avset"),
			},
		},
		// Add VM extensions to Resources
		Resources: []*armcompute.VirtualMachineExtension{
			{
				Name: to.Ptr("CustomScriptExtension"),
				Type: to.Ptr("Microsoft.Compute/virtualMachines/extensions"),
			},
		},
	}
}
