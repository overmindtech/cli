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

type mockFrontendIPConfigPager struct {
	pages []armnetwork.LoadBalancerFrontendIPConfigurationsClientListResponse
	index int
}

func (m *mockFrontendIPConfigPager) More() bool {
	return m.index < len(m.pages)
}

func (m *mockFrontendIPConfigPager) NextPage(ctx context.Context) (armnetwork.LoadBalancerFrontendIPConfigurationsClientListResponse, error) {
	if m.index >= len(m.pages) {
		return armnetwork.LoadBalancerFrontendIPConfigurationsClientListResponse{}, errors.New("no more pages")
	}
	page := m.pages[m.index]
	m.index++
	return page, nil
}

type errorFrontendIPConfigPager struct{}

func (e *errorFrontendIPConfigPager) More() bool {
	return true
}

func (e *errorFrontendIPConfigPager) NextPage(ctx context.Context) (armnetwork.LoadBalancerFrontendIPConfigurationsClientListResponse, error) {
	return armnetwork.LoadBalancerFrontendIPConfigurationsClientListResponse{}, errors.New("pager error")
}

type testFrontendIPConfigClient struct {
	*mocks.MockLoadBalancerFrontendIPConfigurationsClient
	pager clients.LoadBalancerFrontendIPConfigurationsPager
}

func (t *testFrontendIPConfigClient) NewListPager(resourceGroupName, loadBalancerName string) clients.LoadBalancerFrontendIPConfigurationsPager {
	return t.pager
}

func TestNetworkLoadBalancerFrontendIPConfiguration(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"
	loadBalancerName := "test-lb"
	frontendIPConfigName := "test-frontend-ip"

	t.Run("Get", func(t *testing.T) {
		frontendIPConfig := createAzureFrontendIPConfiguration(frontendIPConfigName, loadBalancerName, subscriptionID, resourceGroup)

		mockClient := mocks.NewMockLoadBalancerFrontendIPConfigurationsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, loadBalancerName, frontendIPConfigName).Return(
			armnetwork.LoadBalancerFrontendIPConfigurationsClientGetResponse{
				FrontendIPConfiguration: *frontendIPConfig,
			}, nil)

		testClient := &testFrontendIPConfigClient{MockLoadBalancerFrontendIPConfigurationsClient: mockClient}
		wrapper := manual.NewNetworkLoadBalancerFrontendIPConfiguration(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(loadBalancerName, frontendIPConfigName)
		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.NetworkLoadBalancerFrontendIPConfiguration.String() {
			t.Errorf("Expected type %s, got %s", azureshared.NetworkLoadBalancerFrontendIPConfiguration, sdpItem.GetType())
		}

		if sdpItem.GetUniqueAttribute() != "uniqueAttr" {
			t.Errorf("Expected unique attribute 'uniqueAttr', got %s", sdpItem.GetUniqueAttribute())
		}

		expectedUniqueValue := shared.CompositeLookupKey(loadBalancerName, frontendIPConfigName)
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
					ExpectedType:   azureshared.NetworkPublicIPAddress.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-public-ip",
					ExpectedScope:  fmt.Sprintf("%s.%s", subscriptionID, resourceGroup),
				},
				{
					ExpectedType:   azureshared.NetworkSubnet.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey("test-vnet", "test-subnet"),
					ExpectedScope:  fmt.Sprintf("%s.%s", subscriptionID, resourceGroup),
				},
				{
					ExpectedType:   azureshared.NetworkPublicIPPrefix.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-ip-prefix",
					ExpectedScope:  fmt.Sprintf("%s.%s", subscriptionID, resourceGroup),
				},
				{
					ExpectedType:   azureshared.NetworkLoadBalancerFrontendIPConfiguration.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey("gateway-lb", "gateway-frontend"),
					ExpectedScope:  fmt.Sprintf("%s.%s", subscriptionID, resourceGroup),
				},
				{
					ExpectedType:   azureshared.NetworkLoadBalancerInboundNatRule.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey(loadBalancerName, "nat-rule-1"),
					ExpectedScope:  fmt.Sprintf("%s.%s", subscriptionID, resourceGroup),
				},
				{
					ExpectedType:   azureshared.NetworkLoadBalancerInboundNatPool.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey(loadBalancerName, "nat-pool-1"),
					ExpectedScope:  fmt.Sprintf("%s.%s", subscriptionID, resourceGroup),
				},
				{
					ExpectedType:   azureshared.NetworkLoadBalancerOutboundRule.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey(loadBalancerName, "outbound-rule-1"),
					ExpectedScope:  fmt.Sprintf("%s.%s", subscriptionID, resourceGroup),
				},
				{
					ExpectedType:   azureshared.NetworkLoadBalancerLoadBalancingRule.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey(loadBalancerName, "lb-rule-1"),
					ExpectedScope:  fmt.Sprintf("%s.%s", subscriptionID, resourceGroup),
				},
				{
					ExpectedType:   "ip",
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "10.0.0.5",
					ExpectedScope:  "global",
				},
			}
			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("Get_WithInsufficientQueryParts", func(t *testing.T) {
		mockClient := mocks.NewMockLoadBalancerFrontendIPConfigurationsClient(ctrl)
		testClient := &testFrontendIPConfigClient{MockLoadBalancerFrontendIPConfigurationsClient: mockClient}

		wrapper := manual.NewNetworkLoadBalancerFrontendIPConfiguration(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], loadBalancerName, true)
		if qErr == nil {
			t.Error("Expected error when providing insufficient query parts, but got nil")
		}
	})

	t.Run("Get_WithEmptyLoadBalancerName", func(t *testing.T) {
		mockClient := mocks.NewMockLoadBalancerFrontendIPConfigurationsClient(ctrl)
		testClient := &testFrontendIPConfigClient{MockLoadBalancerFrontendIPConfigurationsClient: mockClient}

		wrapper := manual.NewNetworkLoadBalancerFrontendIPConfiguration(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey("", frontendIPConfigName)
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr == nil {
			t.Error("Expected error when loadBalancerName is empty, but got nil")
		}
	})

	t.Run("Get_WithEmptyFrontendIPConfigName", func(t *testing.T) {
		mockClient := mocks.NewMockLoadBalancerFrontendIPConfigurationsClient(ctrl)
		testClient := &testFrontendIPConfigClient{MockLoadBalancerFrontendIPConfigurationsClient: mockClient}

		wrapper := manual.NewNetworkLoadBalancerFrontendIPConfiguration(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(loadBalancerName, "")
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr == nil {
			t.Error("Expected error when frontendIPConfigurationName is empty, but got nil")
		}
	})

	t.Run("Search", func(t *testing.T) {
		frontendIP1 := createAzureFrontendIPConfigurationMinimal("frontend-1", loadBalancerName, subscriptionID, resourceGroup)
		frontendIP2 := createAzureFrontendIPConfigurationMinimal("frontend-2", loadBalancerName, subscriptionID, resourceGroup)

		mockClient := mocks.NewMockLoadBalancerFrontendIPConfigurationsClient(ctrl)
		mockPager := &mockFrontendIPConfigPager{
			pages: []armnetwork.LoadBalancerFrontendIPConfigurationsClientListResponse{
				{
					LoadBalancerFrontendIPConfigurationListResult: armnetwork.LoadBalancerFrontendIPConfigurationListResult{
						Value: []*armnetwork.FrontendIPConfiguration{frontendIP1, frontendIP2},
					},
				},
			},
		}

		testClient := &testFrontendIPConfigClient{
			MockLoadBalancerFrontendIPConfigurationsClient: mockClient,
			pager: mockPager,
		}

		wrapper := manual.NewNetworkLoadBalancerFrontendIPConfiguration(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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
			if item.GetType() != azureshared.NetworkLoadBalancerFrontendIPConfiguration.String() {
				t.Errorf("Expected type %s, got %s", azureshared.NetworkLoadBalancerFrontendIPConfiguration, item.GetType())
			}
		}
	})

	t.Run("Search_WithNilName", func(t *testing.T) {
		validFrontendIP := createAzureFrontendIPConfigurationMinimal("valid-frontend", loadBalancerName, subscriptionID, resourceGroup)

		mockClient := mocks.NewMockLoadBalancerFrontendIPConfigurationsClient(ctrl)
		mockPager := &mockFrontendIPConfigPager{
			pages: []armnetwork.LoadBalancerFrontendIPConfigurationsClientListResponse{
				{
					LoadBalancerFrontendIPConfigurationListResult: armnetwork.LoadBalancerFrontendIPConfigurationListResult{
						Value: []*armnetwork.FrontendIPConfiguration{
							{Name: nil, ID: new("/some/id")},
							validFrontendIP,
						},
					},
				},
			},
		}

		testClient := &testFrontendIPConfigClient{
			MockLoadBalancerFrontendIPConfigurationsClient: mockClient,
			pager: mockPager,
		}

		wrapper := manual.NewNetworkLoadBalancerFrontendIPConfiguration(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		searchable := adapter.(discovery.SearchableAdapter)
		sdpItems, err := searchable.Search(ctx, wrapper.Scopes()[0], loadBalancerName, true)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if len(sdpItems) != 1 {
			t.Fatalf("Expected 1 item (nil name skipped), got: %d", len(sdpItems))
		}
		expectedValue := shared.CompositeLookupKey(loadBalancerName, "valid-frontend")
		if sdpItems[0].UniqueAttributeValue() != expectedValue {
			t.Errorf("Expected unique value %s, got %s", expectedValue, sdpItems[0].UniqueAttributeValue())
		}
	})

	t.Run("Search_InvalidQueryParts", func(t *testing.T) {
		mockClient := mocks.NewMockLoadBalancerFrontendIPConfigurationsClient(ctrl)
		testClient := &testFrontendIPConfigClient{MockLoadBalancerFrontendIPConfigurationsClient: mockClient}

		wrapper := manual.NewNetworkLoadBalancerFrontendIPConfiguration(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		_, qErr := wrapper.Search(ctx, wrapper.Scopes()[0])
		if qErr == nil {
			t.Error("Expected error when providing no query parts, but got nil")
		}
	})

	t.Run("ErrorHandling_Get", func(t *testing.T) {
		expectedErr := errors.New("frontend IP config not found")

		mockClient := mocks.NewMockLoadBalancerFrontendIPConfigurationsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, loadBalancerName, "nonexistent-frontend").Return(
			armnetwork.LoadBalancerFrontendIPConfigurationsClientGetResponse{}, expectedErr)

		testClient := &testFrontendIPConfigClient{MockLoadBalancerFrontendIPConfigurationsClient: mockClient}
		wrapper := manual.NewNetworkLoadBalancerFrontendIPConfiguration(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(loadBalancerName, "nonexistent-frontend")
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr == nil {
			t.Error("Expected error when getting non-existent frontend IP config, but got nil")
		}
	})

	t.Run("ErrorHandling_Search", func(t *testing.T) {
		mockClient := mocks.NewMockLoadBalancerFrontendIPConfigurationsClient(ctrl)
		testClient := &testFrontendIPConfigClient{
			MockLoadBalancerFrontendIPConfigurationsClient: mockClient,
			pager: &errorFrontendIPConfigPager{},
		}

		wrapper := manual.NewNetworkLoadBalancerFrontendIPConfiguration(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		searchable := adapter.(discovery.SearchableAdapter)
		_, err := searchable.Search(ctx, wrapper.Scopes()[0], loadBalancerName, true)
		if err == nil {
			t.Error("Expected error from pager when NextPage returns an error, but got nil")
		}
	})

	t.Run("Get_CrossResourceGroupLinks", func(t *testing.T) {
		frontendIPConfig := createAzureFrontendIPConfigCrossRG(frontendIPConfigName, loadBalancerName, "other-sub", "other-rg")

		mockClient := mocks.NewMockLoadBalancerFrontendIPConfigurationsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, loadBalancerName, frontendIPConfigName).Return(
			armnetwork.LoadBalancerFrontendIPConfigurationsClientGetResponse{
				FrontendIPConfiguration: *frontendIPConfig,
			}, nil)

		testClient := &testFrontendIPConfigClient{MockLoadBalancerFrontendIPConfigurationsClient: mockClient}
		wrapper := manual.NewNetworkLoadBalancerFrontendIPConfiguration(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(loadBalancerName, frontendIPConfigName)
		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		found := false
		for _, linkedQuery := range sdpItem.GetLinkedItemQueries() {
			if linkedQuery.GetQuery().GetType() == azureshared.NetworkPublicIPAddress.String() {
				found = true
				expectedScope := "other-sub.other-rg"
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

	t.Run("Get_NoProperties", func(t *testing.T) {
		frontendIPConfig := &armnetwork.FrontendIPConfiguration{
			ID:   new(fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/loadBalancers/%s/frontendIPConfigurations/%s", subscriptionID, resourceGroup, loadBalancerName, frontendIPConfigName)),
			Name: new(frontendIPConfigName),
			Type: new("Microsoft.Network/loadBalancers/frontendIPConfigurations"),
		}

		mockClient := mocks.NewMockLoadBalancerFrontendIPConfigurationsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, loadBalancerName, frontendIPConfigName).Return(
			armnetwork.LoadBalancerFrontendIPConfigurationsClientGetResponse{
				FrontendIPConfiguration: *frontendIPConfig,
			}, nil)

		testClient := &testFrontendIPConfigClient{MockLoadBalancerFrontendIPConfigurationsClient: mockClient}
		wrapper := manual.NewNetworkLoadBalancerFrontendIPConfiguration(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(loadBalancerName, frontendIPConfigName)
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

func createAzureFrontendIPConfiguration(name, lbName, subscriptionID, resourceGroup string) *armnetwork.FrontendIPConfiguration {
	provisioningState := armnetwork.ProvisioningStateSucceeded
	publicIPID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/publicIPAddresses/test-public-ip", subscriptionID, resourceGroup)
	subnetID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/test-subnet", subscriptionID, resourceGroup)
	publicIPPrefixID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/publicIPPrefixes/test-ip-prefix", subscriptionID, resourceGroup)
	gatewayLBFrontendID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/loadBalancers/gateway-lb/frontendIPConfigurations/gateway-frontend", subscriptionID, resourceGroup)
	natRuleID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/loadBalancers/%s/inboundNatRules/nat-rule-1", subscriptionID, resourceGroup, lbName)
	natPoolID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/loadBalancers/%s/inboundNatPools/nat-pool-1", subscriptionID, resourceGroup, lbName)
	outboundRuleID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/loadBalancers/%s/outboundRules/outbound-rule-1", subscriptionID, resourceGroup, lbName)
	lbRuleID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/loadBalancers/%s/loadBalancingRules/lb-rule-1", subscriptionID, resourceGroup, lbName)

	return &armnetwork.FrontendIPConfiguration{
		ID:   new(fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/loadBalancers/%s/frontendIPConfigurations/%s", subscriptionID, resourceGroup, lbName, name)),
		Name: new(name),
		Type: new("Microsoft.Network/loadBalancers/frontendIPConfigurations"),
		Properties: &armnetwork.FrontendIPConfigurationPropertiesFormat{
			ProvisioningState: &provisioningState,
			PublicIPAddress: &armnetwork.PublicIPAddress{
				ID: new(publicIPID),
			},
			Subnet: &armnetwork.Subnet{
				ID: new(subnetID),
			},
			PublicIPPrefix: &armnetwork.SubResource{
				ID: new(publicIPPrefixID),
			},
			GatewayLoadBalancer: &armnetwork.SubResource{
				ID: new(gatewayLBFrontendID),
			},
			PrivateIPAddress: new("10.0.0.5"),
			InboundNatRules: []*armnetwork.SubResource{
				{ID: new(natRuleID)},
			},
			InboundNatPools: []*armnetwork.SubResource{
				{ID: new(natPoolID)},
			},
			OutboundRules: []*armnetwork.SubResource{
				{ID: new(outboundRuleID)},
			},
			LoadBalancingRules: []*armnetwork.SubResource{
				{ID: new(lbRuleID)},
			},
		},
	}
}

func createAzureFrontendIPConfigurationMinimal(name, lbName, subscriptionID, resourceGroup string) *armnetwork.FrontendIPConfiguration {
	provisioningState := armnetwork.ProvisioningStateSucceeded
	return &armnetwork.FrontendIPConfiguration{
		ID:   new(fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/loadBalancers/%s/frontendIPConfigurations/%s", subscriptionID, resourceGroup, lbName, name)),
		Name: new(name),
		Type: new("Microsoft.Network/loadBalancers/frontendIPConfigurations"),
		Properties: &armnetwork.FrontendIPConfigurationPropertiesFormat{
			ProvisioningState: &provisioningState,
		},
	}
}

func createAzureFrontendIPConfigCrossRG(name, lbName, otherSub, otherRG string) *armnetwork.FrontendIPConfiguration {
	provisioningState := armnetwork.ProvisioningStateSucceeded
	publicIPID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/publicIPAddresses/cross-rg-ip", otherSub, otherRG)

	return &armnetwork.FrontendIPConfiguration{
		ID:   new(fmt.Sprintf("/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.Network/loadBalancers/%s/frontendIPConfigurations/%s", lbName, name)),
		Name: new(name),
		Type: new("Microsoft.Network/loadBalancers/frontendIPConfigurations"),
		Properties: &armnetwork.FrontendIPConfigurationPropertiesFormat{
			ProvisioningState: &provisioningState,
			PublicIPAddress: &armnetwork.PublicIPAddress{
				ID: new(publicIPID),
			},
		},
	}
}
