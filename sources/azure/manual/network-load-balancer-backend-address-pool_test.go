package manual_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v9"
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
)

type mockBackendAddressPoolPager struct {
	pages []armnetwork.LoadBalancerBackendAddressPoolsClientListResponse
	index int
}

func (m *mockBackendAddressPoolPager) More() bool {
	return m.index < len(m.pages)
}

func (m *mockBackendAddressPoolPager) NextPage(ctx context.Context) (armnetwork.LoadBalancerBackendAddressPoolsClientListResponse, error) {
	if m.index >= len(m.pages) {
		return armnetwork.LoadBalancerBackendAddressPoolsClientListResponse{}, errors.New("no more pages")
	}
	page := m.pages[m.index]
	m.index++
	return page, nil
}

type errorBackendAddressPoolPager struct{}

func (e *errorBackendAddressPoolPager) More() bool {
	return true
}

func (e *errorBackendAddressPoolPager) NextPage(ctx context.Context) (armnetwork.LoadBalancerBackendAddressPoolsClientListResponse, error) {
	return armnetwork.LoadBalancerBackendAddressPoolsClientListResponse{}, errors.New("pager error")
}

type testBackendAddressPoolClient struct {
	*mocks.MockLoadBalancerBackendAddressPoolsClient
	pager clients.LoadBalancerBackendAddressPoolsPager
}

func (t *testBackendAddressPoolClient) NewListPager(resourceGroupName, loadBalancerName string) clients.LoadBalancerBackendAddressPoolsPager {
	return t.pager
}

func TestNetworkLoadBalancerBackendAddressPool(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"
	loadBalancerName := "test-lb"
	backendPoolName := "test-backend-pool"

	t.Run("Get", func(t *testing.T) {
		backendPool := createAzureBackendAddressPool(backendPoolName, loadBalancerName, subscriptionID, resourceGroup)

		mockClient := mocks.NewMockLoadBalancerBackendAddressPoolsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, loadBalancerName, backendPoolName).Return(
			armnetwork.LoadBalancerBackendAddressPoolsClientGetResponse{
				BackendAddressPool: *backendPool,
			}, nil)

		testClient := &testBackendAddressPoolClient{MockLoadBalancerBackendAddressPoolsClient: mockClient}
		wrapper := manual.NewNetworkLoadBalancerBackendAddressPool(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(loadBalancerName, backendPoolName)
		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.NetworkLoadBalancerBackendAddressPool.String() {
			t.Errorf("Expected type %s, got %s", azureshared.NetworkLoadBalancerBackendAddressPool, sdpItem.GetType())
		}

		if sdpItem.GetUniqueAttribute() != "uniqueAttr" {
			t.Errorf("Expected unique attribute 'uniqueAttr', got %s", sdpItem.GetUniqueAttribute())
		}

		expectedUniqueValue := shared.CompositeLookupKey(loadBalancerName, backendPoolName)
		if sdpItem.UniqueAttributeValue() != expectedUniqueValue {
			t.Errorf("Expected unique attribute value %s, got %s", expectedUniqueValue, sdpItem.UniqueAttributeValue())
		}

		if sdpItem.GetScope() != subscriptionID+"."+resourceGroup {
			t.Errorf("Expected scope %s, got %s", subscriptionID+"."+resourceGroup, sdpItem.GetScope())
		}

		if err := sdpItem.Validate(); err != nil {
			t.Fatalf("Expected no validation error, got: %v", err)
		}

		if sdpItem.GetHealth() != sdp.Health_HEALTH_OK {
			t.Errorf("Expected health OK, got %s", sdpItem.GetHealth())
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					ExpectedType:   azureshared.NetworkLoadBalancer.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  loadBalancerName,
					ExpectedScope:  fmt.Sprintf("%s.%s", subscriptionID, resourceGroup),
				},
				{
					ExpectedType:   azureshared.NetworkVirtualNetwork.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-vnet",
					ExpectedScope:  fmt.Sprintf("%s.%s", subscriptionID, resourceGroup),
				},
				{
					ExpectedType:   azureshared.NetworkLoadBalancerInboundNatRule.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey(loadBalancerName, "nat-rule-1"),
					ExpectedScope:  fmt.Sprintf("%s.%s", subscriptionID, resourceGroup),
				},
				{
					ExpectedType:   azureshared.NetworkLoadBalancerLoadBalancingRule.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey(loadBalancerName, "lb-rule-1"),
					ExpectedScope:  fmt.Sprintf("%s.%s", subscriptionID, resourceGroup),
				},
				{
					ExpectedType:   azureshared.NetworkLoadBalancerOutboundRule.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey(loadBalancerName, "outbound-rule-1"),
					ExpectedScope:  fmt.Sprintf("%s.%s", subscriptionID, resourceGroup),
				},
				{
					ExpectedType:   azureshared.NetworkNetworkInterfaceIPConfiguration.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey("test-nic", "test-ip-config"),
					ExpectedScope:  fmt.Sprintf("%s.%s", subscriptionID, resourceGroup),
				},
				{
					ExpectedType:   azureshared.NetworkVirtualNetwork.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "addr-vnet",
					ExpectedScope:  fmt.Sprintf("%s.%s", subscriptionID, resourceGroup),
				},
				{
					ExpectedType:   azureshared.NetworkSubnet.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey("addr-vnet", "addr-subnet"),
					ExpectedScope:  fmt.Sprintf("%s.%s", subscriptionID, resourceGroup),
				},
				{
					ExpectedType:   azureshared.NetworkLoadBalancerFrontendIPConfiguration.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey("regional-lb", "frontend-1"),
					ExpectedScope:  fmt.Sprintf("%s.%s", subscriptionID, resourceGroup),
				},
				{
					ExpectedType:   "ip",
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "10.0.0.10",
					ExpectedScope:  "global",
				},
			}
			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("Get_WithInsufficientQueryParts", func(t *testing.T) {
		mockClient := mocks.NewMockLoadBalancerBackendAddressPoolsClient(ctrl)
		testClient := &testBackendAddressPoolClient{MockLoadBalancerBackendAddressPoolsClient: mockClient}

		wrapper := manual.NewNetworkLoadBalancerBackendAddressPool(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], loadBalancerName, true)
		if qErr == nil {
			t.Error("Expected error when providing insufficient query parts, but got nil")
		}
	})

	t.Run("Get_WithEmptyLoadBalancerName", func(t *testing.T) {
		mockClient := mocks.NewMockLoadBalancerBackendAddressPoolsClient(ctrl)
		testClient := &testBackendAddressPoolClient{MockLoadBalancerBackendAddressPoolsClient: mockClient}

		wrapper := manual.NewNetworkLoadBalancerBackendAddressPool(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey("", backendPoolName)
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr == nil {
			t.Error("Expected error when loadBalancerName is empty, but got nil")
		}
	})

	t.Run("Get_WithEmptyBackendPoolName", func(t *testing.T) {
		mockClient := mocks.NewMockLoadBalancerBackendAddressPoolsClient(ctrl)
		testClient := &testBackendAddressPoolClient{MockLoadBalancerBackendAddressPoolsClient: mockClient}

		wrapper := manual.NewNetworkLoadBalancerBackendAddressPool(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(loadBalancerName, "")
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr == nil {
			t.Error("Expected error when backendAddressPoolName is empty, but got nil")
		}
	})

	t.Run("Search", func(t *testing.T) {
		pool1 := createAzureBackendAddressPoolMinimal("pool-1", loadBalancerName, subscriptionID, resourceGroup)
		pool2 := createAzureBackendAddressPoolMinimal("pool-2", loadBalancerName, subscriptionID, resourceGroup)

		mockClient := mocks.NewMockLoadBalancerBackendAddressPoolsClient(ctrl)
		mockPager := &mockBackendAddressPoolPager{
			pages: []armnetwork.LoadBalancerBackendAddressPoolsClientListResponse{
				{
					LoadBalancerBackendAddressPoolListResult: armnetwork.LoadBalancerBackendAddressPoolListResult{
						Value: []*armnetwork.BackendAddressPool{pool1, pool2},
					},
				},
			},
		}

		testClient := &testBackendAddressPoolClient{
			MockLoadBalancerBackendAddressPoolsClient: mockClient,
			pager: mockPager,
		}

		wrapper := manual.NewNetworkLoadBalancerBackendAddressPool(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support Search operation")
		}

		sdpItems, err := searchable.Search(ctx, wrapper.Scopes()[0], loadBalancerName, true)
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
			if item.GetType() != azureshared.NetworkLoadBalancerBackendAddressPool.String() {
				t.Errorf("Expected type %s, got %s", azureshared.NetworkLoadBalancerBackendAddressPool, item.GetType())
			}
		}
	})

	t.Run("Search_WithNilName", func(t *testing.T) {
		validPool := createAzureBackendAddressPoolMinimal("valid-pool", loadBalancerName, subscriptionID, resourceGroup)

		mockClient := mocks.NewMockLoadBalancerBackendAddressPoolsClient(ctrl)
		mockPager := &mockBackendAddressPoolPager{
			pages: []armnetwork.LoadBalancerBackendAddressPoolsClientListResponse{
				{
					LoadBalancerBackendAddressPoolListResult: armnetwork.LoadBalancerBackendAddressPoolListResult{
						Value: []*armnetwork.BackendAddressPool{
							{Name: nil, ID: new("/some/id")},
							validPool,
						},
					},
				},
			},
		}

		testClient := &testBackendAddressPoolClient{
			MockLoadBalancerBackendAddressPoolsClient: mockClient,
			pager: mockPager,
		}

		wrapper := manual.NewNetworkLoadBalancerBackendAddressPool(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		searchable := adapter.(discovery.SearchableAdapter)
		sdpItems, err := searchable.Search(ctx, wrapper.Scopes()[0], loadBalancerName, true)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if len(sdpItems) != 1 {
			t.Fatalf("Expected 1 item (nil name skipped), got: %d", len(sdpItems))
		}
		expectedValue := shared.CompositeLookupKey(loadBalancerName, "valid-pool")
		if sdpItems[0].UniqueAttributeValue() != expectedValue {
			t.Errorf("Expected unique value %s, got %s", expectedValue, sdpItems[0].UniqueAttributeValue())
		}
	})

	t.Run("Search_InvalidQueryParts", func(t *testing.T) {
		mockClient := mocks.NewMockLoadBalancerBackendAddressPoolsClient(ctrl)
		testClient := &testBackendAddressPoolClient{MockLoadBalancerBackendAddressPoolsClient: mockClient}

		wrapper := manual.NewNetworkLoadBalancerBackendAddressPool(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		_, qErr := wrapper.Search(ctx, wrapper.Scopes()[0])
		if qErr == nil {
			t.Error("Expected error when providing no query parts, but got nil")
		}
	})

	t.Run("Search_WithEmptyLoadBalancerName", func(t *testing.T) {
		mockClient := mocks.NewMockLoadBalancerBackendAddressPoolsClient(ctrl)
		testClient := &testBackendAddressPoolClient{MockLoadBalancerBackendAddressPoolsClient: mockClient}

		wrapper := manual.NewNetworkLoadBalancerBackendAddressPool(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		_, qErr := wrapper.Search(ctx, wrapper.Scopes()[0], "")
		if qErr == nil {
			t.Error("Expected error when loadBalancerName is empty, but got nil")
		}
	})

	t.Run("ErrorHandling_Get", func(t *testing.T) {
		expectedErr := errors.New("backend pool not found")

		mockClient := mocks.NewMockLoadBalancerBackendAddressPoolsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, loadBalancerName, "nonexistent-pool").Return(
			armnetwork.LoadBalancerBackendAddressPoolsClientGetResponse{}, expectedErr)

		testClient := &testBackendAddressPoolClient{MockLoadBalancerBackendAddressPoolsClient: mockClient}
		wrapper := manual.NewNetworkLoadBalancerBackendAddressPool(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(loadBalancerName, "nonexistent-pool")
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr == nil {
			t.Error("Expected error when getting non-existent backend pool, but got nil")
		}
	})

	t.Run("ErrorHandling_Search", func(t *testing.T) {
		mockClient := mocks.NewMockLoadBalancerBackendAddressPoolsClient(ctrl)
		testClient := &testBackendAddressPoolClient{
			MockLoadBalancerBackendAddressPoolsClient: mockClient,
			pager: &errorBackendAddressPoolPager{},
		}

		wrapper := manual.NewNetworkLoadBalancerBackendAddressPool(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		searchable := adapter.(discovery.SearchableAdapter)
		_, err := searchable.Search(ctx, wrapper.Scopes()[0], loadBalancerName, true)
		if err == nil {
			t.Error("Expected error from pager when NextPage returns an error, but got nil")
		}
	})

	t.Run("Get_CrossResourceGroupLinks", func(t *testing.T) {
		backendPool := createAzureBackendAddressPoolCrossRG(backendPoolName, loadBalancerName, "other-sub", "other-rg")

		mockClient := mocks.NewMockLoadBalancerBackendAddressPoolsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, loadBalancerName, backendPoolName).Return(
			armnetwork.LoadBalancerBackendAddressPoolsClientGetResponse{
				BackendAddressPool: *backendPool,
			}, nil)

		testClient := &testBackendAddressPoolClient{MockLoadBalancerBackendAddressPoolsClient: mockClient}
		wrapper := manual.NewNetworkLoadBalancerBackendAddressPool(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(loadBalancerName, backendPoolName)
		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		found := false
		for _, linkedQuery := range sdpItem.GetLinkedItemQueries() {
			if linkedQuery.GetQuery().GetType() == azureshared.NetworkVirtualNetwork.String() {
				found = true
				expectedScope := "other-sub.other-rg"
				if linkedQuery.GetQuery().GetScope() != expectedScope {
					t.Errorf("Expected VirtualNetwork scope to be %s, got: %s", expectedScope, linkedQuery.GetQuery().GetScope())
				}
				break
			}
		}
		if !found {
			t.Error("Expected to find VirtualNetwork linked query")
		}
	})

	t.Run("Get_NoProperties", func(t *testing.T) {
		backendPool := &armnetwork.BackendAddressPool{
			ID:   new(fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/loadBalancers/%s/backendAddressPools/%s", subscriptionID, resourceGroup, loadBalancerName, backendPoolName)),
			Name: new(backendPoolName),
			Type: new("Microsoft.Network/loadBalancers/backendAddressPools"),
		}

		mockClient := mocks.NewMockLoadBalancerBackendAddressPoolsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, loadBalancerName, backendPoolName).Return(
			armnetwork.LoadBalancerBackendAddressPoolsClientGetResponse{
				BackendAddressPool: *backendPool,
			}, nil)

		testClient := &testBackendAddressPoolClient{MockLoadBalancerBackendAddressPoolsClient: mockClient}
		wrapper := manual.NewNetworkLoadBalancerBackendAddressPool(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(loadBalancerName, backendPoolName)
		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		// Should only have the parent load balancer link
		linkedQueries := sdpItem.GetLinkedItemQueries()
		if len(linkedQueries) != 1 {
			t.Errorf("Expected 1 linked query (parent LB only), got %d", len(linkedQueries))
		}
		if linkedQueries[0].GetQuery().GetType() != azureshared.NetworkLoadBalancer.String() {
			t.Errorf("Expected parent LB link, got type %s", linkedQueries[0].GetQuery().GetType())
		}
	})
}

func createAzureBackendAddressPool(name, lbName, subscriptionID, resourceGroup string) *armnetwork.BackendAddressPool {
	provisioningState := armnetwork.ProvisioningStateSucceeded
	vnetID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/virtualNetworks/test-vnet", subscriptionID, resourceGroup)
	natRuleID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/loadBalancers/%s/inboundNatRules/nat-rule-1", subscriptionID, resourceGroup, lbName)
	lbRuleID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/loadBalancers/%s/loadBalancingRules/lb-rule-1", subscriptionID, resourceGroup, lbName)
	outboundRuleID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/loadBalancers/%s/outboundRules/outbound-rule-1", subscriptionID, resourceGroup, lbName)
	nicIPConfigID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/networkInterfaces/test-nic/ipConfigurations/test-ip-config", subscriptionID, resourceGroup)
	addrVnetID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/virtualNetworks/addr-vnet", subscriptionID, resourceGroup)
	addrSubnetID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/virtualNetworks/addr-vnet/subnets/addr-subnet", subscriptionID, resourceGroup)
	frontendIPConfigID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/loadBalancers/regional-lb/frontendIPConfigurations/frontend-1", subscriptionID, resourceGroup)

	return &armnetwork.BackendAddressPool{
		ID:   new(fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/loadBalancers/%s/backendAddressPools/%s", subscriptionID, resourceGroup, lbName, name)),
		Name: new(name),
		Type: new("Microsoft.Network/loadBalancers/backendAddressPools"),
		Properties: &armnetwork.BackendAddressPoolPropertiesFormat{
			ProvisioningState: &provisioningState,
			VirtualNetwork: &armnetwork.SubResource{
				ID: new(vnetID),
			},
			InboundNatRules: []*armnetwork.SubResource{
				{ID: new(natRuleID)},
			},
			LoadBalancingRules: []*armnetwork.SubResource{
				{ID: new(lbRuleID)},
			},
			OutboundRules: []*armnetwork.SubResource{
				{ID: new(outboundRuleID)},
			},
			BackendIPConfigurations: []*armnetwork.InterfaceIPConfiguration{
				{ID: new(nicIPConfigID)},
			},
			LoadBalancerBackendAddresses: []*armnetwork.LoadBalancerBackendAddress{
				{
					Name: new("backend-addr-1"),
					Properties: &armnetwork.LoadBalancerBackendAddressPropertiesFormat{
						IPAddress: new("10.0.0.10"),
						VirtualNetwork: &armnetwork.SubResource{
							ID: new(addrVnetID),
						},
						Subnet: &armnetwork.SubResource{
							ID: new(addrSubnetID),
						},
						LoadBalancerFrontendIPConfiguration: &armnetwork.SubResource{
							ID: new(frontendIPConfigID),
						},
					},
				},
			},
		},
	}
}

func createAzureBackendAddressPoolMinimal(name, lbName, subscriptionID, resourceGroup string) *armnetwork.BackendAddressPool {
	provisioningState := armnetwork.ProvisioningStateSucceeded
	return &armnetwork.BackendAddressPool{
		ID:   new(fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/loadBalancers/%s/backendAddressPools/%s", subscriptionID, resourceGroup, lbName, name)),
		Name: new(name),
		Type: new("Microsoft.Network/loadBalancers/backendAddressPools"),
		Properties: &armnetwork.BackendAddressPoolPropertiesFormat{
			ProvisioningState: &provisioningState,
		},
	}
}

func createAzureBackendAddressPoolCrossRG(name, lbName, otherSub, otherRG string) *armnetwork.BackendAddressPool {
	provisioningState := armnetwork.ProvisioningStateSucceeded
	vnetID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/virtualNetworks/cross-rg-vnet", otherSub, otherRG)

	return &armnetwork.BackendAddressPool{
		ID:   new(fmt.Sprintf("/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.Network/loadBalancers/%s/backendAddressPools/%s", lbName, name)),
		Name: new(name),
		Type: new("Microsoft.Network/loadBalancers/backendAddressPools"),
		Properties: &armnetwork.BackendAddressPoolPropertiesFormat{
			ProvisioningState: &provisioningState,
			VirtualNetwork: &armnetwork.SubResource{
				ID: new(vnetID),
			},
		},
	}
}
