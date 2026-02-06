package manual_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v8"
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

func TestNetworkNetworkInterface(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"

	t.Run("Get", func(t *testing.T) {
		nicName := "test-nic"
		nic := createAzureNetworkInterface(nicName, "test-vm", "test-nsg")

		mockClient := mocks.NewMockNetworkInterfacesClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, nicName).Return(
			armnetwork.InterfacesClientGetResponse{
				Interface: *nic,
			}, nil)

		wrapper := manual.NewNetworkNetworkInterface(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], nicName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.NetworkNetworkInterface.String() {
			t.Errorf("Expected type %s, got %s", azureshared.NetworkNetworkInterface, sdpItem.GetType())
		}

		if sdpItem.GetUniqueAttribute() != "name" {
			t.Errorf("Expected unique attribute 'name', got %s", sdpItem.GetUniqueAttribute())
		}

		if sdpItem.UniqueAttributeValue() != nicName {
			t.Errorf("Expected unique attribute value %s, got %s", nicName, sdpItem.UniqueAttributeValue())
		}

		if sdpItem.GetTags()["env"] != "test" {
			t.Errorf("Expected tag 'env=test', got: %v", sdpItem.GetTags()["env"])
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					// NetworkNetworkInterfaceIPConfiguration link
					ExpectedType:   azureshared.NetworkNetworkInterfaceIPConfiguration.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  nicName,
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  false,
						Out: true,
					},
				},
				{
					// ComputeVirtualMachine link
					ExpectedType:   azureshared.ComputeVirtualMachine.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-vm",
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  false,
						Out: true,
					},
				},
				{
					// NetworkNetworkSecurityGroup link
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
					// NetworkNetworkInterfaceTapConfiguration link (child resource)
					ExpectedType:   azureshared.NetworkNetworkInterfaceTapConfiguration.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  nicName,
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  false,
						Out: true,
					},
				},
			}

			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})

		t.Run("DNSServers_IP_and_hostname", func(t *testing.T) {
			nicWithDNS := createAzureNetworkInterfaceWithDNSServers(nicName, "test-vm", "test-nsg", []string{"10.0.0.1", "dns.internal"})
			mockClient := mocks.NewMockNetworkInterfacesClient(ctrl)
			mockClient.EXPECT().Get(ctx, resourceGroup, nicName).Return(
				armnetwork.InterfacesClientGetResponse{
					Interface: *nicWithDNS,
				}, nil)

			wrapper := manual.NewNetworkNetworkInterface(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
			adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

			sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], nicName, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			// Same base links as main Get test, plus DNS server links (IP → NetworkIP, hostname → NetworkDNS)
			queryTests := shared.QueryTests{
				{
					ExpectedType:   azureshared.NetworkNetworkInterfaceIPConfiguration.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  nicName,
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  false,
						Out: true,
					},
				},
				{
					ExpectedType:   azureshared.ComputeVirtualMachine.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-vm",
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  false,
						Out: true,
					},
				},
				{
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
					ExpectedType:   azureshared.NetworkNetworkInterfaceTapConfiguration.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  nicName,
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  false,
						Out: true,
					},
				},
				{
					ExpectedType:   stdlib.NetworkIP.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "10.0.0.1",
					ExpectedScope:  "global",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					ExpectedType:   stdlib.NetworkDNS.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  "dns.internal",
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
		mockClient := mocks.NewMockNetworkInterfacesClient(ctrl)

		wrapper := manual.NewNetworkNetworkInterface(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		// Test with empty string name - Get will still be called with empty string
		// and Azure will return an error
		mockClient.EXPECT().Get(ctx, resourceGroup, "").Return(
			armnetwork.InterfacesClientGetResponse{}, errors.New("network interface not found"))

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "", true)
		if qErr == nil {
			t.Error("Expected error when getting network interface with empty name, but got nil")
		}
	})

	t.Run("List", func(t *testing.T) {
		nic1 := createAzureNetworkInterface("test-nic-1", "test-vm-1", "test-nsg-1")
		nic2 := createAzureNetworkInterface("test-nic-2", "test-vm-2", "test-nsg-2")

		mockClient := mocks.NewMockNetworkInterfacesClient(ctrl)
		mockPager := NewMockNetworkInterfacesPager(ctrl)

		// Setup pager expectations
		gomock.InOrder(
			mockPager.EXPECT().More().Return(true),
			mockPager.EXPECT().NextPage(ctx).Return(
				armnetwork.InterfacesClientListResponse{
					InterfaceListResult: armnetwork.InterfaceListResult{
						Value: []*armnetwork.Interface{nic1, nic2},
					},
				}, nil),
			mockPager.EXPECT().More().Return(false),
		)

		mockClient.EXPECT().List(ctx, resourceGroup).Return(mockPager)

		wrapper := manual.NewNetworkNetworkInterface(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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

			if item.GetType() != azureshared.NetworkNetworkInterface.String() {
				t.Fatalf("Expected type %s, got: %s", azureshared.NetworkNetworkInterface, item.GetType())
			}
		}
	})

	t.Run("List_WithNilName", func(t *testing.T) {
		// Create NIC with nil name to test error handling
		nic1 := createAzureNetworkInterface("test-nic-1", "test-vm-1", "test-nsg-1")
		nic2 := &armnetwork.Interface{
			Name:     nil, // NIC with nil name should cause an error in azureNetworkInterfaceToSDPItem
			Location: to.Ptr("eastus"),
			Tags: map[string]*string{
				"env": to.Ptr("test"),
			},
			Properties: &armnetwork.InterfacePropertiesFormat{
				IPConfigurations: []*armnetwork.InterfaceIPConfiguration{
					{
						Name: to.Ptr("ipconfig1"),
						Properties: &armnetwork.InterfaceIPConfigurationPropertiesFormat{
							PrivateIPAllocationMethod: to.Ptr(armnetwork.IPAllocationMethodDynamic),
						},
					},
				},
			},
		}

		mockClient := mocks.NewMockNetworkInterfacesClient(ctrl)
		mockPager := NewMockNetworkInterfacesPager(ctrl)

		// Setup pager expectations
		gomock.InOrder(
			mockPager.EXPECT().More().Return(true),
			mockPager.EXPECT().NextPage(ctx).Return(
				armnetwork.InterfacesClientListResponse{
					InterfaceListResult: armnetwork.InterfaceListResult{
						Value: []*armnetwork.Interface{nic1, nic2},
					},
				}, nil),
		)
		// Note: More() won't be called again after NextPage returns the items with nil name
		// because azureNetworkInterfaceToSDPItem will return an error

		mockClient.EXPECT().List(ctx, resourceGroup).Return(mockPager)

		wrapper := manual.NewNetworkNetworkInterface(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		listable, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support List operation")
		}

		_, err := listable.List(ctx, wrapper.Scopes()[0], true)
		// Should return an error because nic2 has nil name
		if err == nil {
			t.Fatalf("Expected error when listing network interfaces with nil name, but got nil")
		}
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		expectedErr := errors.New("network interface not found")

		mockClient := mocks.NewMockNetworkInterfacesClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, "nonexistent-nic").Return(
			armnetwork.InterfacesClientGetResponse{}, expectedErr)

		wrapper := manual.NewNetworkNetworkInterface(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "nonexistent-nic", true)
		if qErr == nil {
			t.Error("Expected error when getting non-existent network interface, but got nil")
		}
	})

	t.Run("ErrorHandling_List", func(t *testing.T) {
		expectedErr := errors.New("failed to list network interfaces")

		mockClient := mocks.NewMockNetworkInterfacesClient(ctrl)
		mockPager := NewMockNetworkInterfacesPager(ctrl)

		// Setup pager to return error on NextPage
		gomock.InOrder(
			mockPager.EXPECT().More().Return(true),
			mockPager.EXPECT().NextPage(ctx).Return(
				armnetwork.InterfacesClientListResponse{}, expectedErr),
		)

		mockClient.EXPECT().List(ctx, resourceGroup).Return(mockPager)

		wrapper := manual.NewNetworkNetworkInterface(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		listable, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support List operation")
		}

		_, err := listable.List(ctx, wrapper.Scopes()[0], true)
		if err == nil {
			t.Error("Expected error when listing network interfaces fails, but got nil")
		}
	})

	t.Run("InterfaceCompliance", func(t *testing.T) {
		mockClient := mocks.NewMockNetworkInterfacesClient(ctrl)
		wrapper := manual.NewNetworkNetworkInterface(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		// Verify wrapper implements ListableWrapper interface
		var _ = wrapper

		// Cast to sources.Wrapper to access interface methods
		w := wrapper.(sources.Wrapper)

		// Verify IAMPermissions
		permissions := w.IAMPermissions()
		if len(permissions) == 0 {
			t.Error("Expected IAMPermissions to return at least one permission")
		}
		expectedPermission := "Microsoft.Network/networkInterfaces/read"
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

		// Verify PotentialLinks
		potentialLinks := w.PotentialLinks()
		if len(potentialLinks) == 0 {
			t.Error("Expected PotentialLinks to return at least one link")
		}
		if !potentialLinks[azureshared.NetworkVirtualNetwork] {
			t.Error("Expected PotentialLinks to include NetworkVirtualNetwork")
		}

		// Verify TerraformMappings
		mappings := w.TerraformMappings()
		if len(mappings) == 0 {
			t.Error("Expected TerraformMappings to return at least one mapping")
		}
		foundMapping := false
		for _, mapping := range mappings {
			if mapping.GetTerraformQueryMap() == "azurerm_network_interface.name" {
				foundMapping = true
				break
			}
		}
		if !foundMapping {
			t.Error("Expected TerraformMappings to include 'azurerm_network_interface.name' mapping")
		}

	})
}

// MockNetworkInterfacesPager is a simple mock for NetworkInterfacesPager
type MockNetworkInterfacesPager struct {
	ctrl     *gomock.Controller
	recorder *MockNetworkInterfacesPagerMockRecorder
}

type MockNetworkInterfacesPagerMockRecorder struct {
	mock *MockNetworkInterfacesPager
}

func NewMockNetworkInterfacesPager(ctrl *gomock.Controller) *MockNetworkInterfacesPager {
	mock := &MockNetworkInterfacesPager{ctrl: ctrl}
	mock.recorder = &MockNetworkInterfacesPagerMockRecorder{mock}
	return mock
}

func (m *MockNetworkInterfacesPager) EXPECT() *MockNetworkInterfacesPagerMockRecorder {
	return m.recorder
}

func (m *MockNetworkInterfacesPager) More() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "More")
	ret0, _ := ret[0].(bool)
	return ret0
}

func (mr *MockNetworkInterfacesPagerMockRecorder) More() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "More", reflect.TypeOf((*MockNetworkInterfacesPager)(nil).More))
}

func (m *MockNetworkInterfacesPager) NextPage(ctx context.Context) (armnetwork.InterfacesClientListResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NextPage", ctx)
	ret0, _ := ret[0].(armnetwork.InterfacesClientListResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockNetworkInterfacesPagerMockRecorder) NextPage(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NextPage", reflect.TypeOf((*MockNetworkInterfacesPager)(nil).NextPage), ctx)
}

// createAzureNetworkInterface creates a mock Azure network interface for testing
func createAzureNetworkInterface(nicName, vmName, nsgName string) *armnetwork.Interface {
	vmID := "/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.Compute/virtualMachines/" + vmName
	nsgID := "/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.Network/networkSecurityGroups/" + nsgName

	return &armnetwork.Interface{
		Name:     to.Ptr(nicName),
		Location: to.Ptr("eastus"),
		Tags: map[string]*string{
			"env":     to.Ptr("test"),
			"project": to.Ptr("testing"),
		},
		Properties: &armnetwork.InterfacePropertiesFormat{
			VirtualMachine: &armnetwork.SubResource{
				ID: to.Ptr(vmID),
			},
			NetworkSecurityGroup: &armnetwork.SecurityGroup{
				ID: to.Ptr(nsgID),
			},
			IPConfigurations: []*armnetwork.InterfaceIPConfiguration{
				{
					Name: to.Ptr("ipconfig1"),
					Properties: &armnetwork.InterfaceIPConfigurationPropertiesFormat{
						PrivateIPAllocationMethod: to.Ptr(armnetwork.IPAllocationMethodDynamic),
					},
				},
			},
		},
	}
}

// createAzureNetworkInterfaceWithDNSServers creates a mock Azure network interface with DNSSettings for testing DNS server links (IP vs hostname).
func createAzureNetworkInterfaceWithDNSServers(nicName, vmName, nsgName string, dnsServers []string) *armnetwork.Interface {
	nic := createAzureNetworkInterface(nicName, vmName, nsgName)
	ptrs := make([]*string, len(dnsServers))
	for i := range dnsServers {
		ptrs[i] = to.Ptr(dnsServers[i])
	}
	nic.Properties.DNSSettings = &armnetwork.InterfaceDNSSettings{
		DNSServers: ptrs,
	}
	return nic
}
