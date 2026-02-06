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
)

func TestNetworkPublicIPAddress(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"

	t.Run("Get", func(t *testing.T) {
		publicIPName := "test-public-ip"
		// Create public IP with network interface (not load balancer, as they're mutually exclusive)
		publicIP := createAzurePublicIPAddress(publicIPName, "test-nic", "test-prefix", "test-nat-gateway", "test-ddos-plan", "")

		mockClient := mocks.NewMockPublicIPAddressesClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, publicIPName).Return(
			armnetwork.PublicIPAddressesClientGetResponse{
				PublicIPAddress: *publicIP,
			}, nil)

		wrapper := manual.NewNetworkPublicIPAddress(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], publicIPName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.NetworkPublicIPAddress.String() {
			t.Errorf("Expected type %s, got %s", azureshared.NetworkPublicIPAddress, sdpItem.GetType())
		}

		if sdpItem.GetUniqueAttribute() != "name" {
			t.Errorf("Expected unique attribute 'name', got %s", sdpItem.GetUniqueAttribute())
		}

		if sdpItem.UniqueAttributeValue() != publicIPName {
			t.Errorf("Expected unique attribute value %s, got %s", publicIPName, sdpItem.UniqueAttributeValue())
		}

		if sdpItem.GetTags()["env"] != "test" {
			t.Errorf("Expected tag 'env=test', got: %v", sdpItem.GetTags()["env"])
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					// IP address link (standard library)
					ExpectedType:   "ip",
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "203.0.113.1",
					ExpectedScope:  "global",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					// NetworkNetworkInterface link (via IPConfiguration)
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
					// NetworkPublicIPPrefix link
					ExpectedType:   azureshared.NetworkPublicIPPrefix.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-prefix",
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					// NetworkNatGateway link
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
					// NetworkDdosProtectionPlan link
					ExpectedType:   azureshared.NetworkDdosProtectionPlan.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-ddos-plan",
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

	t.Run("Get_WithLoadBalancer", func(t *testing.T) {
		publicIPName := "test-public-ip-lb"
		// Create public IP with load balancer (not network interface, as they're mutually exclusive)
		publicIP := createAzurePublicIPAddress(publicIPName, "", "", "", "", "test-load-balancer")

		mockClient := mocks.NewMockPublicIPAddressesClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, publicIPName).Return(
			armnetwork.PublicIPAddressesClientGetResponse{
				PublicIPAddress: *publicIP,
			}, nil)

		wrapper := manual.NewNetworkPublicIPAddress(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], publicIPName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		// Verify LoadBalancer link exists
		foundLoadBalancer := false
		for _, linkedQuery := range sdpItem.GetLinkedItemQueries() {
			if linkedQuery.GetQuery().GetType() == azureshared.NetworkLoadBalancer.String() &&
				linkedQuery.GetQuery().GetQuery() == "test-load-balancer" {
				foundLoadBalancer = true
				break
			}
		}
		if !foundLoadBalancer {
			t.Error("Expected to find LoadBalancer linked item query")
		}
	})

	t.Run("Get_WithLinkedPublicIP", func(t *testing.T) {
		publicIPName := "test-public-ip"
		linkedIPName := "linked-public-ip"
		publicIP := createAzurePublicIPAddressWithLinkedIP(publicIPName, linkedIPName, subscriptionID, resourceGroup)

		mockClient := mocks.NewMockPublicIPAddressesClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, publicIPName).Return(
			armnetwork.PublicIPAddressesClientGetResponse{
				PublicIPAddress: *publicIP,
			}, nil)

		wrapper := manual.NewNetworkPublicIPAddress(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], publicIPName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		// Verify linked public IP address query
		foundLinkedIP := false
		for _, linkedQuery := range sdpItem.GetLinkedItemQueries() {
			if linkedQuery.GetQuery().GetType() == azureshared.NetworkPublicIPAddress.String() &&
				linkedQuery.GetQuery().GetQuery() == linkedIPName {
				foundLinkedIP = true
				if linkedQuery.GetBlastPropagation().GetIn() != true || linkedQuery.GetBlastPropagation().GetOut() != true {
					t.Error("Expected linked public IP to have In: true, Out: true")
				}
				break
			}
		}
		if !foundLinkedIP {
			t.Error("Expected to find linked public IP address query")
		}
	})

	t.Run("Get_WithServicePublicIP", func(t *testing.T) {
		publicIPName := "test-public-ip"
		serviceIPName := "service-public-ip"
		publicIP := createAzurePublicIPAddressWithServiceIP(publicIPName, serviceIPName, subscriptionID, resourceGroup)

		mockClient := mocks.NewMockPublicIPAddressesClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, publicIPName).Return(
			armnetwork.PublicIPAddressesClientGetResponse{
				PublicIPAddress: *publicIP,
			}, nil)

		wrapper := manual.NewNetworkPublicIPAddress(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], publicIPName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		// Verify service public IP address query
		foundServiceIP := false
		for _, linkedQuery := range sdpItem.GetLinkedItemQueries() {
			if linkedQuery.GetQuery().GetType() == azureshared.NetworkPublicIPAddress.String() &&
				linkedQuery.GetQuery().GetQuery() == serviceIPName {
				foundServiceIP = true
				if linkedQuery.GetBlastPropagation().GetIn() != true || linkedQuery.GetBlastPropagation().GetOut() != false {
					t.Error("Expected service public IP to have In: true, Out: false")
				}
				break
			}
		}
		if !foundServiceIP {
			t.Error("Expected to find service public IP address query")
		}
	})

	t.Run("Get_InvalidQueryParts", func(t *testing.T) {
		mockClient := mocks.NewMockPublicIPAddressesClient(ctrl)

		wrapper := manual.NewNetworkPublicIPAddress(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		// Test with empty name - Get will still be called with empty string
		// and Azure will return an error
		mockClient.EXPECT().Get(ctx, resourceGroup, "").Return(
			armnetwork.PublicIPAddressesClientGetResponse{}, errors.New("public IP address not found"))

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "", true)
		if qErr == nil {
			t.Error("Expected error when getting public IP address with empty name, but got nil")
		}
	})

	t.Run("List", func(t *testing.T) {
		publicIP1 := createAzurePublicIPAddress("test-public-ip-1", "", "", "", "", "")
		publicIP2 := createAzurePublicIPAddress("test-public-ip-2", "", "", "", "", "")

		mockClient := mocks.NewMockPublicIPAddressesClient(ctrl)
		mockPager := NewMockPublicIPAddressesPager(ctrl)

		// Setup pager expectations
		gomock.InOrder(
			mockPager.EXPECT().More().Return(true),
			mockPager.EXPECT().NextPage(ctx).Return(
				armnetwork.PublicIPAddressesClientListResponse{
					PublicIPAddressListResult: armnetwork.PublicIPAddressListResult{
						Value: []*armnetwork.PublicIPAddress{publicIP1, publicIP2},
					},
				}, nil),
			mockPager.EXPECT().More().Return(false),
		)

		mockClient.EXPECT().List(ctx, resourceGroup).Return(mockPager)

		wrapper := manual.NewNetworkPublicIPAddress(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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

			if item.GetType() != azureshared.NetworkPublicIPAddress.String() {
				t.Fatalf("Expected type %s, got: %s", azureshared.NetworkPublicIPAddress, item.GetType())
			}
		}
	})

	t.Run("List_WithNilName", func(t *testing.T) {
		// Create public IP with nil name to test error handling
		// Note: The List method skips items with nil names (continues), so it doesn't return an error
		publicIP1 := createAzurePublicIPAddress("test-public-ip-1", "", "", "", "", "")
		publicIP2 := &armnetwork.PublicIPAddress{
			Name:     nil, // Public IP with nil name will be skipped
			Location: to.Ptr("eastus"),
			Tags: map[string]*string{
				"env": to.Ptr("test"),
			},
			Properties: &armnetwork.PublicIPAddressPropertiesFormat{},
		}

		mockClient := mocks.NewMockPublicIPAddressesClient(ctrl)
		mockPager := NewMockPublicIPAddressesPager(ctrl)

		// Setup pager expectations
		// More() is called: once before NextPage, once after processing the page
		gomock.InOrder(
			mockPager.EXPECT().More().Return(true),
			mockPager.EXPECT().NextPage(ctx).Return(
				armnetwork.PublicIPAddressesClientListResponse{
					PublicIPAddressListResult: armnetwork.PublicIPAddressListResult{
						Value: []*armnetwork.PublicIPAddress{publicIP1, publicIP2},
					},
				}, nil),
			mockPager.EXPECT().More().Return(false), // No more pages after processing
		)

		mockClient.EXPECT().List(ctx, resourceGroup).Return(mockPager)

		wrapper := manual.NewNetworkPublicIPAddress(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		listable, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support List operation")
		}

		sdpItems, err := listable.List(ctx, wrapper.Scopes()[0], true)
		// Should not return an error - items with nil names are skipped
		if err != nil {
			t.Fatalf("Expected no error when listing public IP addresses with nil name (they are skipped), but got: %v", err)
		}
		// Should only return 1 item (the one with a valid name)
		if len(sdpItems) != 1 {
			t.Fatalf("Expected 1 item (nil name item should be skipped), got %d", len(sdpItems))
		}
		if sdpItems[0].UniqueAttributeValue() != "test-public-ip-1" {
			t.Fatalf("Expected item with name 'test-public-ip-1', got %s", sdpItems[0].UniqueAttributeValue())
		}
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		expectedErr := errors.New("public IP address not found")

		mockClient := mocks.NewMockPublicIPAddressesClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, "nonexistent-ip").Return(
			armnetwork.PublicIPAddressesClientGetResponse{}, expectedErr)

		wrapper := manual.NewNetworkPublicIPAddress(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "nonexistent-ip", true)
		if qErr == nil {
			t.Error("Expected error when getting non-existent public IP address, but got nil")
		}
	})

	t.Run("ErrorHandling_List", func(t *testing.T) {
		expectedErr := errors.New("failed to list public IP addresses")

		mockClient := mocks.NewMockPublicIPAddressesClient(ctrl)
		mockPager := NewMockPublicIPAddressesPager(ctrl)

		// Setup pager to return error on NextPage
		gomock.InOrder(
			mockPager.EXPECT().More().Return(true),
			mockPager.EXPECT().NextPage(ctx).Return(
				armnetwork.PublicIPAddressesClientListResponse{}, expectedErr),
		)

		mockClient.EXPECT().List(ctx, resourceGroup).Return(mockPager)

		wrapper := manual.NewNetworkPublicIPAddress(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		listable, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support List operation")
		}

		_, err := listable.List(ctx, wrapper.Scopes()[0], true)
		if err == nil {
			t.Error("Expected error when listing public IP addresses fails, but got nil")
		}
	})

	t.Run("InterfaceCompliance", func(t *testing.T) {
		mockClient := mocks.NewMockPublicIPAddressesClient(ctrl)
		wrapper := manual.NewNetworkPublicIPAddress(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		// Verify wrapper implements ListableWrapper interface
		var _ = wrapper

		// Cast to sources.Wrapper to access interface methods
		w := wrapper.(sources.Wrapper)

		// Verify IAMPermissions
		permissions := w.IAMPermissions()
		if len(permissions) == 0 {
			t.Error("Expected IAMPermissions to return at least one permission")
		}
		expectedPermission := "Microsoft.Network/publicIPAddresses/read"
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
		if !potentialLinks[azureshared.NetworkNetworkInterface] {
			t.Error("Expected PotentialLinks to include NetworkNetworkInterface")
		}
		if !potentialLinks[azureshared.NetworkPublicIPAddress] {
			t.Error("Expected PotentialLinks to include NetworkPublicIPAddress")
		}

		// Verify TerraformMappings
		mappings := w.TerraformMappings()
		if len(mappings) == 0 {
			t.Error("Expected TerraformMappings to return at least one mapping")
		}
		foundMapping := false
		for _, mapping := range mappings {
			if mapping.GetTerraformQueryMap() == "azurerm_public_ip.name" {
				foundMapping = true
				break
			}
		}
		if !foundMapping {
			t.Error("Expected TerraformMappings to include 'azurerm_public_ip.name' mapping")
		}
	})
}

// MockPublicIPAddressesPager is a simple mock for PublicIPAddressesPager
type MockPublicIPAddressesPager struct {
	ctrl     *gomock.Controller
	recorder *MockPublicIPAddressesPagerMockRecorder
}

type MockPublicIPAddressesPagerMockRecorder struct {
	mock *MockPublicIPAddressesPager
}

func NewMockPublicIPAddressesPager(ctrl *gomock.Controller) *MockPublicIPAddressesPager {
	mock := &MockPublicIPAddressesPager{ctrl: ctrl}
	mock.recorder = &MockPublicIPAddressesPagerMockRecorder{mock}
	return mock
}

func (m *MockPublicIPAddressesPager) EXPECT() *MockPublicIPAddressesPagerMockRecorder {
	return m.recorder
}

func (m *MockPublicIPAddressesPager) More() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "More")
	ret0, _ := ret[0].(bool)
	return ret0
}

func (mr *MockPublicIPAddressesPagerMockRecorder) More() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "More", reflect.TypeOf((*MockPublicIPAddressesPager)(nil).More))
}

func (m *MockPublicIPAddressesPager) NextPage(ctx context.Context) (armnetwork.PublicIPAddressesClientListResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NextPage", ctx)
	ret0, _ := ret[0].(armnetwork.PublicIPAddressesClientListResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockPublicIPAddressesPagerMockRecorder) NextPage(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NextPage", reflect.TypeOf((*MockPublicIPAddressesPager)(nil).NextPage), ctx)
}

// createAzurePublicIPAddress creates a mock Azure public IP address for testing
func createAzurePublicIPAddress(name, nicName, prefixName, natGatewayName, ddosPlanName, loadBalancerName string) *armnetwork.PublicIPAddress {
	publicIP := &armnetwork.PublicIPAddress{
		Name:     to.Ptr(name),
		Location: to.Ptr("eastus"),
		Tags: map[string]*string{
			"env":     to.Ptr("test"),
			"project": to.Ptr("testing"),
		},
		Properties: &armnetwork.PublicIPAddressPropertiesFormat{
			PublicIPAddressVersion:   to.Ptr(armnetwork.IPVersionIPv4),
			PublicIPAllocationMethod: to.Ptr(armnetwork.IPAllocationMethodStatic),
			IPAddress:                to.Ptr("203.0.113.1"), // Add IP address for testing
		},
	}

	// Add IPConfiguration if nicName is provided
	if nicName != "" {
		ipConfigID := "/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.Network/networkInterfaces/" + nicName + "/ipConfigurations/ipconfig1"
		publicIP.Properties.IPConfiguration = &armnetwork.IPConfiguration{
			ID: to.Ptr(ipConfigID),
		}
	}

	// Add PublicIPPrefix if prefixName is provided
	if prefixName != "" {
		prefixID := "/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.Network/publicIPPrefixes/" + prefixName
		publicIP.Properties.PublicIPPrefix = &armnetwork.SubResource{
			ID: to.Ptr(prefixID),
		}
	}

	// Add NatGateway if natGatewayName is provided
	if natGatewayName != "" {
		natGatewayID := "/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.Network/natGateways/" + natGatewayName
		publicIP.Properties.NatGateway = &armnetwork.NatGateway{
			ID: to.Ptr(natGatewayID),
		}
	}

	// Add DDoS Protection Plan if ddosPlanName is provided
	if ddosPlanName != "" {
		ddosPlanID := "/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.Network/ddosProtectionPlans/" + ddosPlanName
		publicIP.Properties.DdosSettings = &armnetwork.DdosSettings{
			DdosProtectionPlan: &armnetwork.SubResource{
				ID: to.Ptr(ddosPlanID),
			},
		}
	}

	// Add LoadBalancer IPConfiguration if loadBalancerName is provided
	if loadBalancerName != "" {
		lbIPConfigID := "/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.Network/loadBalancers/" + loadBalancerName + "/frontendIPConfigurations/frontendIPConfig1"
		publicIP.Properties.IPConfiguration = &armnetwork.IPConfiguration{
			ID: to.Ptr(lbIPConfigID),
		}
	}

	return publicIP
}

// createAzurePublicIPAddressWithLinkedIP creates a mock Azure public IP address with a linked public IP
func createAzurePublicIPAddressWithLinkedIP(name, linkedIPName, subscriptionID, resourceGroup string) *armnetwork.PublicIPAddress {
	linkedIPID := "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/publicIPAddresses/" + linkedIPName

	return &armnetwork.PublicIPAddress{
		Name:     to.Ptr(name),
		Location: to.Ptr("eastus"),
		Tags: map[string]*string{
			"env": to.Ptr("test"),
		},
		Properties: &armnetwork.PublicIPAddressPropertiesFormat{
			PublicIPAddressVersion:   to.Ptr(armnetwork.IPVersionIPv4),
			PublicIPAllocationMethod: to.Ptr(armnetwork.IPAllocationMethodStatic),
			LinkedPublicIPAddress: &armnetwork.PublicIPAddress{
				ID: to.Ptr(linkedIPID),
			},
		},
	}
}

// createAzurePublicIPAddressWithServiceIP creates a mock Azure public IP address with a service public IP
func createAzurePublicIPAddressWithServiceIP(name, serviceIPName, subscriptionID, resourceGroup string) *armnetwork.PublicIPAddress {
	serviceIPID := "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/publicIPAddresses/" + serviceIPName

	return &armnetwork.PublicIPAddress{
		Name:     to.Ptr(name),
		Location: to.Ptr("eastus"),
		Tags: map[string]*string{
			"env": to.Ptr("test"),
		},
		Properties: &armnetwork.PublicIPAddressPropertiesFormat{
			PublicIPAddressVersion:   to.Ptr(armnetwork.IPVersionIPv4),
			PublicIPAllocationMethod: to.Ptr(armnetwork.IPAllocationMethodStatic),
			ServicePublicIPAddress: &armnetwork.PublicIPAddress{
				ID: to.Ptr(serviceIPID),
			},
		},
	}
}
