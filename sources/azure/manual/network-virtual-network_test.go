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

func TestNetworkVirtualNetwork(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"

	t.Run("Get", func(t *testing.T) {
		vnetName := "test-vnet"
		vnet := createAzureVirtualNetwork(vnetName)

		mockClient := mocks.NewMockVirtualNetworksClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, vnetName, nil).Return(
			armnetwork.VirtualNetworksClientGetResponse{
				VirtualNetwork: *vnet,
			}, nil)

		wrapper := manual.NewNetworkVirtualNetwork(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], vnetName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.NetworkVirtualNetwork.String() {
			t.Errorf("Expected type %s, got %s", azureshared.NetworkVirtualNetwork, sdpItem.GetType())
		}

		if sdpItem.GetUniqueAttribute() != "name" {
			t.Errorf("Expected unique attribute 'name', got %s", sdpItem.GetUniqueAttribute())
		}

		if sdpItem.UniqueAttributeValue() != vnetName {
			t.Errorf("Expected unique attribute value %s, got %s", vnetName, sdpItem.UniqueAttributeValue())
		}

		if sdpItem.GetTags()["env"] != "test" {
			t.Errorf("Expected tag 'env=test', got: %v", sdpItem.GetTags()["env"])
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					// NetworkSubnet link
					ExpectedType:   azureshared.NetworkSubnet.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  vnetName,
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  false,
						Out: true,
					},
				},
				{
					// NetworkVirtualNetworkPeering link
					ExpectedType:   azureshared.NetworkVirtualNetworkPeering.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  vnetName,
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

	t.Run("Get_WithDefaultPublicNatGatewayAndDhcpOptions", func(t *testing.T) {
		vnetName := "test-vnet-with-links"
		vnet := createAzureVirtualNetworkWithDefaultNatGatewayAndDhcpOptions(vnetName, subscriptionID, resourceGroup)

		mockClient := mocks.NewMockVirtualNetworksClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, vnetName, nil).Return(
			armnetwork.VirtualNetworksClientGetResponse{
				VirtualNetwork: *vnet,
			}, nil)

		wrapper := manual.NewNetworkVirtualNetwork(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], vnetName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					ExpectedType:   azureshared.NetworkSubnet.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  vnetName,
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  false,
						Out: true,
					},
				},
				{
					ExpectedType:   azureshared.NetworkVirtualNetworkPeering.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  vnetName,
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  false,
						Out: true,
					},
				},
				{
					ExpectedType:   azureshared.NetworkNatGateway.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-nat-gateway",
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
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
		mockClient := mocks.NewMockVirtualNetworksClient(ctrl)

		wrapper := manual.NewNetworkVirtualNetwork(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		// Test with empty string name - Get will still be called with empty string
		// and Azure will return an error
		mockClient.EXPECT().Get(ctx, resourceGroup, "", nil).Return(
			armnetwork.VirtualNetworksClientGetResponse{}, errors.New("virtual network not found"))

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "", true)
		if qErr == nil {
			t.Error("Expected error when getting virtual network with empty name, but got nil")
		}
	})

	t.Run("List", func(t *testing.T) {
		vnet1 := createAzureVirtualNetwork("test-vnet-1")
		vnet2 := createAzureVirtualNetwork("test-vnet-2")

		mockClient := mocks.NewMockVirtualNetworksClient(ctrl)
		mockPager := NewMockVirtualNetworksPager(ctrl)

		// Setup pager expectations
		gomock.InOrder(
			mockPager.EXPECT().More().Return(true),
			mockPager.EXPECT().NextPage(ctx).Return(
				armnetwork.VirtualNetworksClientListResponse{
					VirtualNetworkListResult: armnetwork.VirtualNetworkListResult{
						Value: []*armnetwork.VirtualNetwork{vnet1, vnet2},
					},
				}, nil),
			mockPager.EXPECT().More().Return(false),
		)

		mockClient.EXPECT().NewListPager(resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewNetworkVirtualNetwork(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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

			if item.GetType() != azureshared.NetworkVirtualNetwork.String() {
				t.Fatalf("Expected type %s, got: %s", azureshared.NetworkVirtualNetwork, item.GetType())
			}
		}
	})

	t.Run("List_WithNilName", func(t *testing.T) {
		// Create vnet with nil name to test error handling
		vnet1 := createAzureVirtualNetwork("test-vnet-1")
		vnet2 := &armnetwork.VirtualNetwork{
			Name:     nil, // VNet with nil name should cause an error in azureVirtualNetworkToSDPItem
			Location: to.Ptr("eastus"),
			Tags: map[string]*string{
				"env": to.Ptr("test"),
			},
			Properties: &armnetwork.VirtualNetworkPropertiesFormat{
				AddressSpace: &armnetwork.AddressSpace{
					AddressPrefixes: []*string{to.Ptr("10.0.0.0/16")},
				},
			},
		}

		mockClient := mocks.NewMockVirtualNetworksClient(ctrl)
		mockPager := NewMockVirtualNetworksPager(ctrl)

		// Setup pager expectations
		gomock.InOrder(
			mockPager.EXPECT().More().Return(true),
			mockPager.EXPECT().NextPage(ctx).Return(
				armnetwork.VirtualNetworksClientListResponse{
					VirtualNetworkListResult: armnetwork.VirtualNetworkListResult{
						Value: []*armnetwork.VirtualNetwork{vnet1, vnet2},
					},
				}, nil),
		)
		// Note: More() won't be called again after NextPage returns the items with nil name
		// because azureVirtualNetworkToSDPItem will return an error

		mockClient.EXPECT().NewListPager(resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewNetworkVirtualNetwork(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		listable, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support List operation")
		}

		_, err := listable.List(ctx, wrapper.Scopes()[0], true)
		// Should return an error because vnet2 has nil name
		if err == nil {
			t.Fatalf("Expected error when listing virtual networks with nil name, but got nil")
		}
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		expectedErr := errors.New("virtual network not found")

		mockClient := mocks.NewMockVirtualNetworksClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, "nonexistent-vnet", nil).Return(
			armnetwork.VirtualNetworksClientGetResponse{}, expectedErr)

		wrapper := manual.NewNetworkVirtualNetwork(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "nonexistent-vnet", true)
		if qErr == nil {
			t.Error("Expected error when getting non-existent virtual network, but got nil")
		}
	})

	t.Run("ErrorHandling_List", func(t *testing.T) {
		expectedErr := errors.New("failed to list virtual networks")

		mockClient := mocks.NewMockVirtualNetworksClient(ctrl)
		mockPager := NewMockVirtualNetworksPager(ctrl)

		// Setup pager to return error on NextPage
		gomock.InOrder(
			mockPager.EXPECT().More().Return(true),
			mockPager.EXPECT().NextPage(ctx).Return(
				armnetwork.VirtualNetworksClientListResponse{}, expectedErr),
		)

		mockClient.EXPECT().NewListPager(resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewNetworkVirtualNetwork(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		listable, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support List operation")
		}

		_, err := listable.List(ctx, wrapper.Scopes()[0], true)
		if err == nil {
			t.Error("Expected error when listing virtual networks fails, but got nil")
		}
	})

	t.Run("InterfaceCompliance", func(t *testing.T) {
		mockClient := mocks.NewMockVirtualNetworksClient(ctrl)
		wrapper := manual.NewNetworkVirtualNetwork(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		// Verify wrapper implements ListableWrapper interface
		var _ = wrapper

		// Cast to sources.Wrapper to access interface methods
		w := wrapper.(sources.Wrapper)

		// Verify IAMPermissions
		permissions := w.IAMPermissions()
		if len(permissions) == 0 {
			t.Error("Expected IAMPermissions to return at least one permission")
		}
		expectedPermission := "Microsoft.Network/virtualNetworks/read"
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
		if !potentialLinks[azureshared.NetworkSubnet] {
			t.Error("Expected PotentialLinks to include NetworkSubnet")
		}
		if !potentialLinks[azureshared.NetworkVirtualNetworkPeering] {
			t.Error("Expected PotentialLinks to include NetworkVirtualNetworkPeering")
		}
		if !potentialLinks[stdlib.NetworkIP] {
			t.Error("Expected PotentialLinks to include stdlib.NetworkIP")
		}
		if !potentialLinks[stdlib.NetworkDNS] {
			t.Error("Expected PotentialLinks to include stdlib.NetworkDNS")
		}

		// Verify TerraformMappings
		mappings := w.TerraformMappings()
		if len(mappings) == 0 {
			t.Error("Expected TerraformMappings to return at least one mapping")
		}
		foundMapping := false
		for _, mapping := range mappings {
			if mapping.GetTerraformQueryMap() == "azurerm_virtual_network.name" {
				foundMapping = true
				if mapping.GetTerraformMethod() != sdp.QueryMethod_GET {
					t.Errorf("Expected TerraformMethod to be GET for name mapping, got %s", mapping.GetTerraformMethod())
				}
			}
		}
		if !foundMapping {
			t.Error("Expected TerraformMappings to include 'azurerm_virtual_network.name' mapping")
		}
	})
}

// MockVirtualNetworksPager is a simple mock for VirtualNetworksPager
type MockVirtualNetworksPager struct {
	ctrl     *gomock.Controller
	recorder *MockVirtualNetworksPagerMockRecorder
}

type MockVirtualNetworksPagerMockRecorder struct {
	mock *MockVirtualNetworksPager
}

func NewMockVirtualNetworksPager(ctrl *gomock.Controller) *MockVirtualNetworksPager {
	mock := &MockVirtualNetworksPager{ctrl: ctrl}
	mock.recorder = &MockVirtualNetworksPagerMockRecorder{mock}
	return mock
}

func (m *MockVirtualNetworksPager) EXPECT() *MockVirtualNetworksPagerMockRecorder {
	return m.recorder
}

func (m *MockVirtualNetworksPager) More() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "More")
	ret0, _ := ret[0].(bool)
	return ret0
}

func (mr *MockVirtualNetworksPagerMockRecorder) More() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "More", reflect.TypeOf((*MockVirtualNetworksPager)(nil).More))
}

func (m *MockVirtualNetworksPager) NextPage(ctx context.Context) (armnetwork.VirtualNetworksClientListResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NextPage", ctx)
	ret0, _ := ret[0].(armnetwork.VirtualNetworksClientListResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockVirtualNetworksPagerMockRecorder) NextPage(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NextPage", reflect.TypeOf((*MockVirtualNetworksPager)(nil).NextPage), ctx)
}

// createAzureVirtualNetwork creates a mock Azure virtual network for testing
func createAzureVirtualNetwork(vnetName string) *armnetwork.VirtualNetwork {
	return &armnetwork.VirtualNetwork{
		Name:     to.Ptr(vnetName),
		Location: to.Ptr("eastus"),
		Tags: map[string]*string{
			"env":     to.Ptr("test"),
			"project": to.Ptr("testing"),
		},
		Properties: &armnetwork.VirtualNetworkPropertiesFormat{
			AddressSpace: &armnetwork.AddressSpace{
				AddressPrefixes: []*string{to.Ptr("10.0.0.0/16")},
			},
			Subnets: []*armnetwork.Subnet{
				{
					Name: to.Ptr("default"),
					Properties: &armnetwork.SubnetPropertiesFormat{
						AddressPrefix: to.Ptr("10.0.0.0/24"),
					},
				},
			},
		},
	}
}

// createAzureVirtualNetworkWithDefaultNatGatewayAndDhcpOptions creates a VNet with
// DefaultPublicNatGateway and DhcpOptions.DNSServers (IP and hostname) for testing linked queries.
func createAzureVirtualNetworkWithDefaultNatGatewayAndDhcpOptions(vnetName, subscriptionID, resourceGroup string) *armnetwork.VirtualNetwork {
	natGatewayID := "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/natGateways/test-nat-gateway"
	return &armnetwork.VirtualNetwork{
		Name:     to.Ptr(vnetName),
		Location: to.Ptr("eastus"),
		Tags: map[string]*string{
			"env": to.Ptr("test"),
		},
		Properties: &armnetwork.VirtualNetworkPropertiesFormat{
			AddressSpace: &armnetwork.AddressSpace{
				AddressPrefixes: []*string{to.Ptr("10.0.0.0/16")},
			},
			DefaultPublicNatGateway: &armnetwork.SubResource{
				ID: to.Ptr(natGatewayID),
			},
			DhcpOptions: &armnetwork.DhcpOptions{
				DNSServers: []*string{
					to.Ptr("10.0.0.1"),     // IP address → stdlib.NetworkIP
					to.Ptr("dns.internal"), // hostname → stdlib.NetworkDNS
				},
			},
			Subnets: []*armnetwork.Subnet{
				{
					Name: to.Ptr("default"),
					Properties: &armnetwork.SubnetPropertiesFormat{
						AddressPrefix: to.Ptr("10.0.0.0/24"),
					},
				},
			},
		},
	}
}
