package manual_test

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v6"
	"go.uber.org/mock/gomock"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/manual"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/azure/shared/mocks"
	"github.com/overmindtech/cli/sources/shared"
)

func TestComputeVirtualMachineScaleSet(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"

	t.Run("Get", func(t *testing.T) {
		scaleSetName := "test-vmss"
		scaleSet := createAzureVirtualMachineScaleSet(scaleSetName, "Succeeded")

		mockClient := mocks.NewMockVirtualMachineScaleSetsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, scaleSetName, nil).Return(
			armcompute.VirtualMachineScaleSetsClientGetResponse{
				VirtualMachineScaleSet: *scaleSet,
			}, nil)

		wrapper := manual.NewComputeVirtualMachineScaleSet(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper)

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], scaleSetName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.ComputeVirtualMachineScaleSet.String() {
			t.Errorf("Expected type %s, got %s", azureshared.ComputeVirtualMachineScaleSet, sdpItem.GetType())
		}

		if sdpItem.GetUniqueAttribute() != "name" {
			t.Errorf("Expected unique attribute 'name', got %s", sdpItem.GetUniqueAttribute())
		}

		if sdpItem.UniqueAttributeValue() != scaleSetName {
			t.Errorf("Expected unique attribute value %s, got %s", scaleSetName, sdpItem.UniqueAttributeValue())
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
					// Extension link - uses composite lookup key
					ExpectedType:   azureshared.ComputeVirtualMachineExtension.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey(scaleSetName, "CustomScriptExtension"),
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  false,
						Out: true,
					},
				},
				{
					// VM instances - always linked via SEARCH
					ExpectedType:   azureshared.ComputeVirtualMachine.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  scaleSetName,
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  false,
						Out: true,
					},
				},
				{
					// Network Security Group
					ExpectedType:   azureshared.NetworkNetworkSecurityGroup.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-nsg",
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					// Virtual Network
					ExpectedType:   azureshared.NetworkVirtualNetwork.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-vnet",
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					// Subnet - uses composite lookup key
					ExpectedType:   azureshared.NetworkSubnet.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey("test-vnet", "default"),
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					// Public IP Prefix
					ExpectedType:   azureshared.NetworkPublicIPPrefix.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-pip-prefix",
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					// Load Balancer
					ExpectedType:   azureshared.NetworkLoadBalancer.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-lb",
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					// Load Balancer Backend Address Pool - uses composite lookup key
					ExpectedType:   azureshared.NetworkLoadBalancerBackendAddressPool.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey("test-lb", "test-backend-pool"),
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					// Load Balancer Inbound NAT Pool - uses composite lookup key
					ExpectedType:   azureshared.NetworkLoadBalancerInboundNatPool.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey("test-lb", "test-nat-pool"),
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					// Application Gateway
					ExpectedType:   azureshared.NetworkApplicationGateway.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-ag",
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					// Application Gateway Backend Address Pool - uses composite lookup key
					ExpectedType:   azureshared.NetworkApplicationGatewayBackendAddressPool.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey("test-ag", "test-ag-pool"),
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					// Application Security Group
					ExpectedType:   azureshared.NetworkApplicationSecurityGroup.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-asg",
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					// Load Balancer Health Probe - uses composite lookup key
					ExpectedType:   azureshared.NetworkLoadBalancerProbe.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey("test-lb", "test-probe"),
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					// Disk Encryption Set (OS Disk)
					ExpectedType:   azureshared.ComputeDiskEncryptionSet.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-disk-encryption-set",
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					// Disk Encryption Set (Data Disk)
					ExpectedType:   azureshared.ComputeDiskEncryptionSet.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-disk-encryption-set-data",
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					// Image (custom image)
					ExpectedType:   azureshared.ComputeImage.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-image",
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					// Proximity Placement Group
					ExpectedType:   azureshared.ComputeProximityPlacementGroup.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-ppg",
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					// Dedicated Host Group
					ExpectedType:   azureshared.ComputeDedicatedHostGroup.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-host-group",
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					// User Assigned Identity
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
					// DNS name (boot diagnostics storage URI)
					ExpectedType:   "dns",
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  "teststorageaccount.blob.core.windows.net",
					ExpectedScope:  "global",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					// Storage Account (boot diagnostics)
					ExpectedType:   azureshared.StorageAccount.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "teststorageaccount",
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					// Key Vault (OS profile secrets)
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
					// Key Vault (extension protected settings)
					ExpectedType:   azureshared.KeyVaultVault.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-keyvault-ext",
					ExpectedScope:  subscriptionID + "." + resourceGroup,
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
		mockClient := mocks.NewMockVirtualMachineScaleSetsClient(ctrl)

		wrapper := manual.NewComputeVirtualMachineScaleSet(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper)

		// Test with empty string name
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "", true)
		if qErr == nil {
			t.Error("Expected error when getting scale set with empty name, but got nil")
		}
	})

	t.Run("Get_NilName", func(t *testing.T) {
		scaleSet := createAzureVirtualMachineScaleSet("", "Succeeded")
		scaleSet.Name = nil // Explicitly set to nil

		mockClient := mocks.NewMockVirtualMachineScaleSetsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, "test-vmss", nil).Return(
			armcompute.VirtualMachineScaleSetsClientGetResponse{
				VirtualMachineScaleSet: *scaleSet,
			}, nil)

		wrapper := manual.NewComputeVirtualMachineScaleSet(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper)

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "test-vmss", true)
		if qErr == nil {
			t.Error("Expected error when scale set name is nil, but got nil")
		}
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

		mockClient := mocks.NewMockVirtualMachineScaleSetsClient(ctrl)

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				scaleSet := createAzureVirtualMachineScaleSet("test-vmss", tc.provisioningState)

				mockClient.EXPECT().Get(ctx, resourceGroup, "test-vmss", nil).Return(
					armcompute.VirtualMachineScaleSetsClientGetResponse{
						VirtualMachineScaleSet: *scaleSet,
					}, nil)

				wrapper := manual.NewComputeVirtualMachineScaleSet(mockClient, subscriptionID, resourceGroup)
				adapter := sources.WrapperToAdapter(wrapper)

				sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "test-vmss", true)
				if qErr != nil {
					t.Fatalf("Expected no error, got: %v", qErr)
				}

				if sdpItem.GetHealth() != tc.expectedHealth {
					t.Errorf("Expected health %s, got: %s", tc.expectedHealth, sdpItem.GetHealth())
				}
			})
		}
	})

	t.Run("List", func(t *testing.T) {
		scaleSet1 := createAzureVirtualMachineScaleSet("test-vmss-1", "Succeeded")
		scaleSet2 := createAzureVirtualMachineScaleSet("test-vmss-2", "Succeeded")

		mockClient := mocks.NewMockVirtualMachineScaleSetsClient(ctrl)
		mockPager := NewMockVirtualMachineScaleSetsPager(ctrl)

		// Setup pager expectations
		gomock.InOrder(
			mockPager.EXPECT().More().Return(true),
			mockPager.EXPECT().NextPage(ctx).Return(
				armcompute.VirtualMachineScaleSetsClientListResponse{
					VirtualMachineScaleSetListResult: armcompute.VirtualMachineScaleSetListResult{
						Value: []*armcompute.VirtualMachineScaleSet{scaleSet1, scaleSet2},
					},
				}, nil),
			mockPager.EXPECT().More().Return(false),
		)

		mockClient.EXPECT().NewListPager(resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewComputeVirtualMachineScaleSet(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper)

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
		scaleSet1 := createAzureVirtualMachineScaleSet("test-vmss-1", "Succeeded")
		scaleSet2 := createAzureVirtualMachineScaleSet("test-vmss-2", "Succeeded")

		mockClient := mocks.NewMockVirtualMachineScaleSetsClient(ctrl)
		mockPager := NewMockVirtualMachineScaleSetsPager(ctrl)

		// Setup pager expectations
		gomock.InOrder(
			mockPager.EXPECT().More().Return(true),
			mockPager.EXPECT().NextPage(ctx).Return(
				armcompute.VirtualMachineScaleSetsClientListResponse{
					VirtualMachineScaleSetListResult: armcompute.VirtualMachineScaleSetListResult{
						Value: []*armcompute.VirtualMachineScaleSet{scaleSet1, scaleSet2},
					},
				}, nil),
			mockPager.EXPECT().More().Return(false),
		)

		mockClient.EXPECT().NewListPager(resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewComputeVirtualMachineScaleSet(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper)

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
		expectedErr := errors.New("VMSS not found")

		mockClient := mocks.NewMockVirtualMachineScaleSetsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, "nonexistent-vmss", nil).Return(
			armcompute.VirtualMachineScaleSetsClientGetResponse{}, expectedErr)

		wrapper := manual.NewComputeVirtualMachineScaleSet(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper)

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "nonexistent-vmss", true)
		if qErr == nil {
			t.Error("Expected error when getting non-existent VMSS, but got nil")
		}
	})

	t.Run("ListErrorHandling", func(t *testing.T) {
		expectedErr := errors.New("list error")

		mockClient := mocks.NewMockVirtualMachineScaleSetsClient(ctrl)
		mockPager := NewMockVirtualMachineScaleSetsPager(ctrl)

		// Setup pager expectations
		gomock.InOrder(
			mockPager.EXPECT().More().Return(true),
			mockPager.EXPECT().NextPage(ctx).Return(
				armcompute.VirtualMachineScaleSetsClientListResponse{}, expectedErr),
		)

		mockClient.EXPECT().NewListPager(resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewComputeVirtualMachineScaleSet(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper)

		listable, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support List operation")
		}

		_, err := listable.List(ctx, wrapper.Scopes()[0], true)
		if err == nil {
			t.Error("Expected error when listing fails, but got nil")
		}
	})
}

// MockVirtualMachineScaleSetsPager is a mock implementation of VirtualMachineScaleSetsPager
type MockVirtualMachineScaleSetsPager struct {
	ctrl     *gomock.Controller
	recorder *MockVirtualMachineScaleSetsPagerMockRecorder
}

type MockVirtualMachineScaleSetsPagerMockRecorder struct {
	mock *MockVirtualMachineScaleSetsPager
}

func NewMockVirtualMachineScaleSetsPager(ctrl *gomock.Controller) *MockVirtualMachineScaleSetsPager {
	mock := &MockVirtualMachineScaleSetsPager{ctrl: ctrl}
	mock.recorder = &MockVirtualMachineScaleSetsPagerMockRecorder{mock}
	return mock
}

func (m *MockVirtualMachineScaleSetsPager) EXPECT() *MockVirtualMachineScaleSetsPagerMockRecorder {
	return m.recorder
}

func (m *MockVirtualMachineScaleSetsPager) More() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "More")
	ret0, _ := ret[0].(bool)
	return ret0
}

func (mr *MockVirtualMachineScaleSetsPagerMockRecorder) More() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "More", reflect.TypeOf((*MockVirtualMachineScaleSetsPager)(nil).More))
}

func (m *MockVirtualMachineScaleSetsPager) NextPage(ctx context.Context) (armcompute.VirtualMachineScaleSetsClientListResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NextPage", ctx)
	ret0, _ := ret[0].(armcompute.VirtualMachineScaleSetsClientListResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockVirtualMachineScaleSetsPagerMockRecorder) NextPage(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NextPage", reflect.TypeOf((*MockVirtualMachineScaleSetsPager)(nil).NextPage), ctx)
}

// createAzureVirtualMachineScaleSet creates a mock Azure Virtual Machine Scale Set for testing
func createAzureVirtualMachineScaleSet(scaleSetName, provisioningState string) *armcompute.VirtualMachineScaleSet {
	return &armcompute.VirtualMachineScaleSet{
		Name:     to.Ptr(scaleSetName),
		Location: to.Ptr("eastus"),
		Tags: map[string]*string{
			"env":     to.Ptr("test"),
			"project": to.Ptr("testing"),
		},
		Properties: &armcompute.VirtualMachineScaleSetProperties{
			ProvisioningState: to.Ptr(provisioningState),
			VirtualMachineProfile: &armcompute.VirtualMachineScaleSetVMProfile{
				ExtensionProfile: &armcompute.VirtualMachineScaleSetExtensionProfile{
					Extensions: []*armcompute.VirtualMachineScaleSetExtension{
						{
							Name: to.Ptr("CustomScriptExtension"),
							Properties: &armcompute.VirtualMachineScaleSetExtensionProperties{
								Type:               to.Ptr("CustomScriptExtension"),
								Publisher:          to.Ptr("Microsoft.Compute"),
								TypeHandlerVersion: to.Ptr("1.10"),
								ProtectedSettingsFromKeyVault: &armcompute.KeyVaultSecretReference{
									SourceVault: &armcompute.SubResource{
										ID: to.Ptr("/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.KeyVault/vaults/test-keyvault-ext"),
									},
								},
							},
						},
					},
				},
				NetworkProfile: &armcompute.VirtualMachineScaleSetNetworkProfile{
					HealthProbe: &armcompute.APIEntityReference{
						ID: to.Ptr("/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.Network/loadBalancers/test-lb/probes/test-probe"),
					},
					NetworkInterfaceConfigurations: []*armcompute.VirtualMachineScaleSetNetworkConfiguration{
						{
							Name: to.Ptr("nic-config"),
							Properties: &armcompute.VirtualMachineScaleSetNetworkConfigurationProperties{
								NetworkSecurityGroup: &armcompute.SubResource{
									ID: to.Ptr("/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.Network/networkSecurityGroups/test-nsg"),
								},
								IPConfigurations: []*armcompute.VirtualMachineScaleSetIPConfiguration{
									{
										Name: to.Ptr("ip-config"),
										Properties: &armcompute.VirtualMachineScaleSetIPConfigurationProperties{
											Subnet: &armcompute.APIEntityReference{
												ID: to.Ptr("/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/default"),
											},
											PublicIPAddressConfiguration: &armcompute.VirtualMachineScaleSetPublicIPAddressConfiguration{
												Name: to.Ptr("public-ip-config"),
												Properties: &armcompute.VirtualMachineScaleSetPublicIPAddressConfigurationProperties{
													PublicIPPrefix: &armcompute.SubResource{
														ID: to.Ptr("/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.Network/publicIPPrefixes/test-pip-prefix"),
													},
												},
											},
											LoadBalancerBackendAddressPools: []*armcompute.SubResource{
												{
													ID: to.Ptr("/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.Network/loadBalancers/test-lb/backendAddressPools/test-backend-pool"),
												},
											},
											LoadBalancerInboundNatPools: []*armcompute.SubResource{
												{
													ID: to.Ptr("/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.Network/loadBalancers/test-lb/inboundNatPools/test-nat-pool"),
												},
											},
											ApplicationGatewayBackendAddressPools: []*armcompute.SubResource{
												{
													ID: to.Ptr("/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.Network/applicationGateways/test-ag/backendAddressPools/test-ag-pool"),
												},
											},
											ApplicationSecurityGroups: []*armcompute.SubResource{
												{
													ID: to.Ptr("/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.Network/applicationSecurityGroups/test-asg"),
												},
											},
										},
									},
								},
							},
						},
					},
				},
				StorageProfile: &armcompute.VirtualMachineScaleSetStorageProfile{
					ImageReference: &armcompute.ImageReference{
						ID: to.Ptr("/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.Compute/images/test-image"),
					},
					OSDisk: &armcompute.VirtualMachineScaleSetOSDisk{
						Name: to.Ptr("os-disk"),
						ManagedDisk: &armcompute.VirtualMachineScaleSetManagedDiskParameters{
							DiskEncryptionSet: &armcompute.DiskEncryptionSetParameters{
								ID: to.Ptr("/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.Compute/diskEncryptionSets/test-disk-encryption-set"),
							},
						},
					},
					DataDisks: []*armcompute.VirtualMachineScaleSetDataDisk{
						{
							Name: to.Ptr("data-disk-1"),
							ManagedDisk: &armcompute.VirtualMachineScaleSetManagedDiskParameters{
								DiskEncryptionSet: &armcompute.DiskEncryptionSetParameters{
									ID: to.Ptr("/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.Compute/diskEncryptionSets/test-disk-encryption-set-data"),
								},
							},
						},
					},
				},
				OSProfile: &armcompute.VirtualMachineScaleSetOSProfile{
					Secrets: []*armcompute.VaultSecretGroup{
						{
							SourceVault: &armcompute.SubResource{
								ID: to.Ptr("/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.KeyVault/vaults/test-keyvault"),
							},
						},
					},
				},
				DiagnosticsProfile: &armcompute.DiagnosticsProfile{
					BootDiagnostics: &armcompute.BootDiagnostics{
						StorageURI: to.Ptr("https://teststorageaccount.blob.core.windows.net/"),
					},
				},
			},
			ProximityPlacementGroup: &armcompute.SubResource{
				ID: to.Ptr("/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.Compute/proximityPlacementGroups/test-ppg"),
			},
			HostGroup: &armcompute.SubResource{
				ID: to.Ptr("/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.Compute/hostGroups/test-host-group"),
			},
		},
		Identity: &armcompute.VirtualMachineScaleSetIdentity{
			Type: to.Ptr(armcompute.ResourceIdentityTypeUserAssigned),
			UserAssignedIdentities: map[string]*armcompute.UserAssignedIdentitiesValue{
				"/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.ManagedIdentity/userAssignedIdentities/test-identity": {},
			},
		},
	}
}
