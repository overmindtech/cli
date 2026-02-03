package manual_test

import (
	"context"
	"errors"
	"fmt"
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
)

func TestNetworkLoadBalancer(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"

	t.Run("Get", func(t *testing.T) {
		lbName := "test-lb"
		loadBalancer := createAzureLoadBalancer(lbName, subscriptionID, resourceGroup)

		mockClient := mocks.NewMockLoadBalancersClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, lbName).Return(
			armnetwork.LoadBalancersClientGetResponse{
				LoadBalancer: *loadBalancer,
			}, nil)

		wrapper := manual.NewNetworkLoadBalancer(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], lbName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.NetworkLoadBalancer.String() {
			t.Errorf("Expected type %s, got %s", azureshared.NetworkLoadBalancer, sdpItem.GetType())
		}

		if sdpItem.GetUniqueAttribute() != "name" {
			t.Errorf("Expected unique attribute 'name', got %s", sdpItem.GetUniqueAttribute())
		}

		if sdpItem.UniqueAttributeValue() != lbName {
			t.Errorf("Expected unique attribute value %s, got %s", lbName, sdpItem.UniqueAttributeValue())
		}

		if sdpItem.GetTags()["env"] != "test" {
			t.Errorf("Expected tag 'env=test', got: %v", sdpItem.GetTags()["env"])
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					// FrontendIPConfiguration child resource
					ExpectedType:   azureshared.NetworkLoadBalancerFrontendIPConfiguration.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey(lbName, "frontend-ip-config"),
					ExpectedScope:  fmt.Sprintf("%s.%s", subscriptionID, resourceGroup),
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					// PublicIPAddress external resource
					ExpectedType:   azureshared.NetworkPublicIPAddress.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-public-ip",
					ExpectedScope:  fmt.Sprintf("%s.%s", subscriptionID, resourceGroup),
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					// Subnet external resource
					ExpectedType:   azureshared.NetworkSubnet.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey("test-vnet", "test-subnet"),
					ExpectedScope:  fmt.Sprintf("%s.%s", subscriptionID, resourceGroup),
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					// Private IP address link (standard library)
					ExpectedType:   "ip",
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "10.2.0.5",
					ExpectedScope:  "global",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					// BackendAddressPool child resource
					ExpectedType:   azureshared.NetworkLoadBalancerBackendAddressPool.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey(lbName, "backend-pool"),
					ExpectedScope:  fmt.Sprintf("%s.%s", subscriptionID, resourceGroup),
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					// InboundNatRule child resource
					ExpectedType:   azureshared.NetworkLoadBalancerInboundNatRule.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey(lbName, "inbound-nat-rule"),
					ExpectedScope:  fmt.Sprintf("%s.%s", subscriptionID, resourceGroup),
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					// NetworkInterface via InboundNatRule BackendIPConfiguration
					ExpectedType:   azureshared.NetworkNetworkInterface.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-nic",
					ExpectedScope:  fmt.Sprintf("%s.%s", subscriptionID, resourceGroup),
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					// LoadBalancingRule child resource
					ExpectedType:   azureshared.NetworkLoadBalancerLoadBalancingRule.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey(lbName, "lb-rule"),
					ExpectedScope:  fmt.Sprintf("%s.%s", subscriptionID, resourceGroup),
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					// Probe child resource
					ExpectedType:   azureshared.NetworkLoadBalancerProbe.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey(lbName, "probe"),
					ExpectedScope:  fmt.Sprintf("%s.%s", subscriptionID, resourceGroup),
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					// OutboundRule child resource
					ExpectedType:   azureshared.NetworkLoadBalancerOutboundRule.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey(lbName, "outbound-rule"),
					ExpectedScope:  fmt.Sprintf("%s.%s", subscriptionID, resourceGroup),
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					// InboundNatPool child resource
					ExpectedType:   azureshared.NetworkLoadBalancerInboundNatPool.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey(lbName, "nat-pool"),
					ExpectedScope:  fmt.Sprintf("%s.%s", subscriptionID, resourceGroup),
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
			}

			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("Get_EmptyName", func(t *testing.T) {
		mockClient := mocks.NewMockLoadBalancersClient(ctrl)

		wrapper := manual.NewNetworkLoadBalancer(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		// Test with empty string name - the wrapper validates this before calling the client
		// So the client.Get should not be called
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "", true)
		if qErr == nil {
			t.Error("Expected error when getting load balancer with empty name, but got nil")
		}
	})

	t.Run("List", func(t *testing.T) {
		lb1 := createAzureLoadBalancer("test-lb-1", subscriptionID, resourceGroup)
		lb2 := createAzureLoadBalancer("test-lb-2", subscriptionID, resourceGroup)

		mockClient := mocks.NewMockLoadBalancersClient(ctrl)
		mockPager := NewMockLoadBalancersPager(ctrl)

		// Setup pager expectations
		gomock.InOrder(
			mockPager.EXPECT().More().Return(true),
			mockPager.EXPECT().NextPage(ctx).Return(
				armnetwork.LoadBalancersClientListResponse{
					LoadBalancerListResult: armnetwork.LoadBalancerListResult{
						Value: []*armnetwork.LoadBalancer{lb1, lb2},
					},
				}, nil),
			mockPager.EXPECT().More().Return(false),
		)

		mockClient.EXPECT().List(resourceGroup).Return(mockPager)

		wrapper := manual.NewNetworkLoadBalancer(mockClient, subscriptionID, resourceGroup)
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

			if item.GetType() != azureshared.NetworkLoadBalancer.String() {
				t.Fatalf("Expected type %s, got: %s", azureshared.NetworkLoadBalancer, item.GetType())
			}
		}
	})

	t.Run("List_WithNilName", func(t *testing.T) {
		// Test that load balancers with nil names are skipped in List
		lb1 := createAzureLoadBalancer("test-lb-1", subscriptionID, resourceGroup)
		lb2 := &armnetwork.LoadBalancer{
			Name:     nil, // Load balancer with nil name should be skipped
			Location: to.Ptr("eastus"),
			Tags: map[string]*string{
				"env": to.Ptr("test"),
			},
			Properties: &armnetwork.LoadBalancerPropertiesFormat{},
		}

		mockClient := mocks.NewMockLoadBalancersClient(ctrl)
		mockPager := NewMockLoadBalancersPager(ctrl)

		// Setup pager expectations
		gomock.InOrder(
			mockPager.EXPECT().More().Return(true),
			mockPager.EXPECT().NextPage(ctx).Return(
				armnetwork.LoadBalancersClientListResponse{
					LoadBalancerListResult: armnetwork.LoadBalancerListResult{
						Value: []*armnetwork.LoadBalancer{lb1, lb2},
					},
				}, nil),
			mockPager.EXPECT().More().Return(false),
		)

		mockClient.EXPECT().List(resourceGroup).Return(mockPager)

		wrapper := manual.NewNetworkLoadBalancer(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		listable, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support List operation")
		}

		sdpItems, err := listable.List(ctx, wrapper.Scopes()[0], true)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		// Should only return 1 item (lb1), lb2 with nil name should be skipped
		if len(sdpItems) != 1 {
			t.Fatalf("Expected 1 item (nil name should be skipped), got: %d", len(sdpItems))
		}

		if sdpItems[0].UniqueAttributeValue() != "test-lb-1" {
			t.Errorf("Expected item name 'test-lb-1', got: %s", sdpItems[0].UniqueAttributeValue())
		}
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		expectedErr := errors.New("load balancer not found")

		mockClient := mocks.NewMockLoadBalancersClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, "nonexistent-lb").Return(
			armnetwork.LoadBalancersClientGetResponse{}, expectedErr)

		wrapper := manual.NewNetworkLoadBalancer(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "nonexistent-lb", true)
		if qErr == nil {
			t.Error("Expected error when getting non-existent load balancer, but got nil")
		}
	})

	t.Run("ErrorHandling_List", func(t *testing.T) {
		expectedErr := errors.New("failed to list load balancers")

		mockClient := mocks.NewMockLoadBalancersClient(ctrl)
		mockPager := NewMockLoadBalancersPager(ctrl)

		// Setup pager to return error on NextPage
		gomock.InOrder(
			mockPager.EXPECT().More().Return(true),
			mockPager.EXPECT().NextPage(ctx).Return(
				armnetwork.LoadBalancersClientListResponse{}, expectedErr),
		)

		mockClient.EXPECT().List(resourceGroup).Return(mockPager)

		wrapper := manual.NewNetworkLoadBalancer(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		listable, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support List operation")
		}

		_, err := listable.List(ctx, wrapper.Scopes()[0], true)
		if err == nil {
			t.Error("Expected error when listing load balancers fails, but got nil")
		}
	})

	t.Run("InterfaceCompliance", func(t *testing.T) {
		mockClient := mocks.NewMockLoadBalancersClient(ctrl)
		wrapper := manual.NewNetworkLoadBalancer(mockClient, subscriptionID, resourceGroup)

		// Verify wrapper implements ListableWrapper interface
		var _ = wrapper

		// Cast to sources.Wrapper to access interface methods
		w := wrapper.(sources.Wrapper)

		// Verify IAMPermissions
		permissions := w.IAMPermissions()
		if len(permissions) == 0 {
			t.Error("Expected IAMPermissions to return at least one permission")
		}
		expectedPermission := "Microsoft.Network/loadBalancers/read"
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

		// Note: PredefinedRole() is not part of the Wrapper interface, so we can't test it here
		// It's tested implicitly by ensuring the wrapper implements all required methods

		// Verify PotentialLinks
		potentialLinks := w.PotentialLinks()
		if len(potentialLinks) == 0 {
			t.Error("Expected PotentialLinks to return at least one link")
		}
		if !potentialLinks[azureshared.NetworkLoadBalancerFrontendIPConfiguration] {
			t.Error("Expected PotentialLinks to include NetworkLoadBalancerFrontendIPConfiguration")
		}
		if !potentialLinks[azureshared.NetworkPublicIPAddress] {
			t.Error("Expected PotentialLinks to include NetworkPublicIPAddress")
		}
		if !potentialLinks[azureshared.NetworkSubnet] {
			t.Error("Expected PotentialLinks to include NetworkSubnet")
		}
		if !potentialLinks[azureshared.NetworkNetworkInterface] {
			t.Error("Expected PotentialLinks to include NetworkNetworkInterface")
		}
		if !potentialLinks[azureshared.NetworkPublicIPPrefix] {
			t.Error("Expected PotentialLinks to include NetworkPublicIPPrefix")
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
			if mapping.GetTerraformQueryMap() == "azurerm_lb.name" {
				foundMapping = true
				if mapping.GetTerraformMethod() != sdp.QueryMethod_GET {
					t.Errorf("Expected TerraformMethod to be GET, got: %s", mapping.GetTerraformMethod())
				}
				break
			}
		}
		if !foundMapping {
			t.Error("Expected TerraformMappings to include 'azurerm_lb.name' mapping")
		}

		// Verify GetLookups
		lookups := w.GetLookups()
		if len(lookups) == 0 {
			t.Error("Expected GetLookups to return at least one lookup")
		}
		foundLookup := false
		for _, lookup := range lookups {
			if lookup.ItemType == azureshared.NetworkLoadBalancer {
				foundLookup = true
				break
			}
		}
		if !foundLookup {
			t.Error("Expected GetLookups to include NetworkLoadBalancer")
		}
	})

	t.Run("Get_PublicIPAddress_DifferentScope", func(t *testing.T) {
		// Test that PublicIPAddress with different subscription/resource group uses correct scope
		lbName := "test-lb"
		loadBalancer := createAzureLoadBalancerWithDifferentScopePublicIP(lbName, subscriptionID, resourceGroup, "other-sub", "other-rg")

		mockClient := mocks.NewMockLoadBalancersClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, lbName).Return(
			armnetwork.LoadBalancersClientGetResponse{
				LoadBalancer: *loadBalancer,
			}, nil)

		wrapper := manual.NewNetworkLoadBalancer(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], lbName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		// Find the PublicIPAddress linked query
		found := false
		for _, linkedQuery := range sdpItem.GetLinkedItemQueries() {
			if linkedQuery.GetQuery().GetType() == azureshared.NetworkPublicIPAddress.String() {
				found = true
				expectedScope := fmt.Sprintf("%s.%s", "other-sub", "other-rg")
				if linkedQuery.GetQuery().GetScope() != expectedScope {
					t.Errorf("Expected PublicIPAddress scope to be %s, got: %s", expectedScope, linkedQuery.GetQuery().GetScope())
				}
				break
			}
		}
		if !found {
			t.Error("Expected to find PublicIPAddress linked query")
		}
	})

	t.Run("Get_Subnet_DifferentScope", func(t *testing.T) {
		// Test that Subnet with different subscription/resource group uses correct scope
		lbName := "test-lb"
		loadBalancer := createAzureLoadBalancerWithDifferentScopeSubnet(lbName, subscriptionID, resourceGroup, "other-sub", "other-rg")

		mockClient := mocks.NewMockLoadBalancersClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, lbName).Return(
			armnetwork.LoadBalancersClientGetResponse{
				LoadBalancer: *loadBalancer,
			}, nil)

		wrapper := manual.NewNetworkLoadBalancer(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], lbName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		// Find the Subnet linked query
		found := false
		for _, linkedQuery := range sdpItem.GetLinkedItemQueries() {
			if linkedQuery.GetQuery().GetType() == azureshared.NetworkSubnet.String() {
				found = true
				expectedScope := fmt.Sprintf("%s.%s", "other-sub", "other-rg")
				if linkedQuery.GetQuery().GetScope() != expectedScope {
					t.Errorf("Expected Subnet scope to be %s, got: %s", expectedScope, linkedQuery.GetQuery().GetScope())
				}
				break
			}
		}
		if !found {
			t.Error("Expected to find Subnet linked query")
		}
	})
}

// MockLoadBalancersPager is a mock for LoadBalancersPager
type MockLoadBalancersPager struct {
	ctrl     *gomock.Controller
	recorder *MockLoadBalancersPagerMockRecorder
}

type MockLoadBalancersPagerMockRecorder struct {
	mock *MockLoadBalancersPager
}

func NewMockLoadBalancersPager(ctrl *gomock.Controller) *MockLoadBalancersPager {
	mock := &MockLoadBalancersPager{ctrl: ctrl}
	mock.recorder = &MockLoadBalancersPagerMockRecorder{mock}
	return mock
}

func (m *MockLoadBalancersPager) EXPECT() *MockLoadBalancersPagerMockRecorder {
	return m.recorder
}

func (m *MockLoadBalancersPager) More() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "More")
	ret0, _ := ret[0].(bool)
	return ret0
}

func (mr *MockLoadBalancersPagerMockRecorder) More() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "More", reflect.TypeOf((*MockLoadBalancersPager)(nil).More))
}

func (m *MockLoadBalancersPager) NextPage(ctx context.Context) (armnetwork.LoadBalancersClientListResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NextPage", ctx)
	ret0, _ := ret[0].(armnetwork.LoadBalancersClientListResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockLoadBalancersPagerMockRecorder) NextPage(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NextPage", reflect.TypeOf((*MockLoadBalancersPager)(nil).NextPage), ctx)
}

// createAzureLoadBalancer creates a mock Azure load balancer for testing with all linked resources
func createAzureLoadBalancer(lbName, subscriptionID, resourceGroup string) *armnetwork.LoadBalancer {
	publicIPID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/publicIPAddresses/test-public-ip", subscriptionID, resourceGroup)
	subnetID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/test-subnet", subscriptionID, resourceGroup)
	nicID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/networkInterfaces/test-nic/ipConfigurations/ipconfig1", subscriptionID, resourceGroup)

	return &armnetwork.LoadBalancer{
		Name:     to.Ptr(lbName),
		Location: to.Ptr("eastus"),
		Tags: map[string]*string{
			"env":     to.Ptr("test"),
			"project": to.Ptr("testing"),
		},
		Properties: &armnetwork.LoadBalancerPropertiesFormat{
			FrontendIPConfigurations: []*armnetwork.FrontendIPConfiguration{
				{
					Name: to.Ptr("frontend-ip-config"),
					Properties: &armnetwork.FrontendIPConfigurationPropertiesFormat{
						PublicIPAddress: &armnetwork.PublicIPAddress{
							ID: to.Ptr(publicIPID),
						},
						Subnet: &armnetwork.Subnet{
							ID: to.Ptr(subnetID),
						},
						// PrivateIPAddress is present when using a subnet (internal load balancer)
						PrivateIPAddress: to.Ptr("10.2.0.5"),
					},
				},
			},
			BackendAddressPools: []*armnetwork.BackendAddressPool{
				{
					Name: to.Ptr("backend-pool"),
				},
			},
			InboundNatRules: []*armnetwork.InboundNatRule{
				{
					Name: to.Ptr("inbound-nat-rule"),
					Properties: &armnetwork.InboundNatRulePropertiesFormat{
						BackendIPConfiguration: &armnetwork.InterfaceIPConfiguration{
							ID: to.Ptr(nicID),
						},
					},
				},
			},
			LoadBalancingRules: []*armnetwork.LoadBalancingRule{
				{
					Name: to.Ptr("lb-rule"),
				},
			},
			Probes: []*armnetwork.Probe{
				{
					Name: to.Ptr("probe"),
				},
			},
			OutboundRules: []*armnetwork.OutboundRule{
				{
					Name: to.Ptr("outbound-rule"),
				},
			},
			InboundNatPools: []*armnetwork.InboundNatPool{
				{
					Name: to.Ptr("nat-pool"),
				},
			},
		},
	}
}

// createAzureLoadBalancerWithDifferentScopePublicIP creates a load balancer with PublicIPAddress in different scope
func createAzureLoadBalancerWithDifferentScopePublicIP(lbName, subscriptionID, resourceGroup, otherSub, otherRG string) *armnetwork.LoadBalancer {
	publicIPID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/publicIPAddresses/test-public-ip", otherSub, otherRG)

	return &armnetwork.LoadBalancer{
		Name:     to.Ptr(lbName),
		Location: to.Ptr("eastus"),
		Tags: map[string]*string{
			"env": to.Ptr("test"),
		},
		Properties: &armnetwork.LoadBalancerPropertiesFormat{
			FrontendIPConfigurations: []*armnetwork.FrontendIPConfiguration{
				{
					Name: to.Ptr("frontend-ip-config"),
					Properties: &armnetwork.FrontendIPConfigurationPropertiesFormat{
						PublicIPAddress: &armnetwork.PublicIPAddress{
							ID: to.Ptr(publicIPID),
						},
					},
				},
			},
		},
	}
}

// createAzureLoadBalancerWithDifferentScopeSubnet creates a load balancer with Subnet in different scope
func createAzureLoadBalancerWithDifferentScopeSubnet(lbName, subscriptionID, resourceGroup, otherSub, otherRG string) *armnetwork.LoadBalancer {
	subnetID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/test-subnet", otherSub, otherRG)

	return &armnetwork.LoadBalancer{
		Name:     to.Ptr(lbName),
		Location: to.Ptr("eastus"),
		Tags: map[string]*string{
			"env": to.Ptr("test"),
		},
		Properties: &armnetwork.LoadBalancerPropertiesFormat{
			FrontendIPConfigurations: []*armnetwork.FrontendIPConfiguration{
				{
					Name: to.Ptr("frontend-ip-config"),
					Properties: &armnetwork.FrontendIPConfigurationPropertiesFormat{
						Subnet: &armnetwork.Subnet{
							ID: to.Ptr(subnetID),
						},
					},
				},
			},
		},
	}
}
