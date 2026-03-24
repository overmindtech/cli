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

type mockLoadBalancerProbePager struct {
	pages []armnetwork.LoadBalancerProbesClientListResponse
	index int
}

func (m *mockLoadBalancerProbePager) More() bool {
	return m.index < len(m.pages)
}

func (m *mockLoadBalancerProbePager) NextPage(ctx context.Context) (armnetwork.LoadBalancerProbesClientListResponse, error) {
	if m.index >= len(m.pages) {
		return armnetwork.LoadBalancerProbesClientListResponse{}, errors.New("no more pages")
	}
	page := m.pages[m.index]
	m.index++
	return page, nil
}

type errorLoadBalancerProbePager struct{}

func (e *errorLoadBalancerProbePager) More() bool {
	return true
}

func (e *errorLoadBalancerProbePager) NextPage(ctx context.Context) (armnetwork.LoadBalancerProbesClientListResponse, error) {
	return armnetwork.LoadBalancerProbesClientListResponse{}, errors.New("pager error")
}

type testLoadBalancerProbeClient struct {
	*mocks.MockLoadBalancerProbesClient
	pager clients.LoadBalancerProbesPager
}

func (t *testLoadBalancerProbeClient) NewListPager(resourceGroupName, loadBalancerName string) clients.LoadBalancerProbesPager {
	return t.pager
}

func TestNetworkLoadBalancerProbe(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"
	loadBalancerName := "test-lb"
	probeName := "test-probe"

	t.Run("Get", func(t *testing.T) {
		probe := createAzureLoadBalancerProbe(probeName, loadBalancerName, subscriptionID, resourceGroup)

		mockClient := mocks.NewMockLoadBalancerProbesClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, loadBalancerName, probeName).Return(
			armnetwork.LoadBalancerProbesClientGetResponse{
				Probe: *probe,
			}, nil)

		testClient := &testLoadBalancerProbeClient{MockLoadBalancerProbesClient: mockClient}
		wrapper := manual.NewNetworkLoadBalancerProbe(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(loadBalancerName, probeName)
		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.NetworkLoadBalancerProbe.String() {
			t.Errorf("Expected type %s, got %s", azureshared.NetworkLoadBalancerProbe, sdpItem.GetType())
		}

		if sdpItem.GetUniqueAttribute() != "uniqueAttr" {
			t.Errorf("Expected unique attribute 'uniqueAttr', got %s", sdpItem.GetUniqueAttribute())
		}

		expectedUniqueValue := shared.CompositeLookupKey(loadBalancerName, probeName)
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
					ExpectedType:   azureshared.NetworkLoadBalancerLoadBalancingRule.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey(loadBalancerName, "lb-rule-1"),
					ExpectedScope:  fmt.Sprintf("%s.%s", subscriptionID, resourceGroup),
				},
			}
			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("Get_WithInsufficientQueryParts", func(t *testing.T) {
		mockClient := mocks.NewMockLoadBalancerProbesClient(ctrl)
		testClient := &testLoadBalancerProbeClient{MockLoadBalancerProbesClient: mockClient}

		wrapper := manual.NewNetworkLoadBalancerProbe(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], loadBalancerName, true)
		if qErr == nil {
			t.Error("Expected error when providing insufficient query parts, but got nil")
		}
	})

	t.Run("Get_WithEmptyLoadBalancerName", func(t *testing.T) {
		mockClient := mocks.NewMockLoadBalancerProbesClient(ctrl)
		testClient := &testLoadBalancerProbeClient{MockLoadBalancerProbesClient: mockClient}

		wrapper := manual.NewNetworkLoadBalancerProbe(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey("", probeName)
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr == nil {
			t.Error("Expected error when loadBalancerName is empty, but got nil")
		}
	})

	t.Run("Get_WithEmptyProbeName", func(t *testing.T) {
		mockClient := mocks.NewMockLoadBalancerProbesClient(ctrl)
		testClient := &testLoadBalancerProbeClient{MockLoadBalancerProbesClient: mockClient}

		wrapper := manual.NewNetworkLoadBalancerProbe(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(loadBalancerName, "")
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr == nil {
			t.Error("Expected error when probeName is empty, but got nil")
		}
	})

	t.Run("Search", func(t *testing.T) {
		probe1 := createAzureLoadBalancerProbeMinimal("probe-1", loadBalancerName, subscriptionID, resourceGroup)
		probe2 := createAzureLoadBalancerProbeMinimal("probe-2", loadBalancerName, subscriptionID, resourceGroup)

		mockClient := mocks.NewMockLoadBalancerProbesClient(ctrl)
		mockPager := &mockLoadBalancerProbePager{
			pages: []armnetwork.LoadBalancerProbesClientListResponse{
				{
					LoadBalancerProbeListResult: armnetwork.LoadBalancerProbeListResult{
						Value: []*armnetwork.Probe{probe1, probe2},
					},
				},
			},
		}

		testClient := &testLoadBalancerProbeClient{
			MockLoadBalancerProbesClient: mockClient,
			pager:                        mockPager,
		}

		wrapper := manual.NewNetworkLoadBalancerProbe(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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
			if item.GetType() != azureshared.NetworkLoadBalancerProbe.String() {
				t.Errorf("Expected type %s, got %s", azureshared.NetworkLoadBalancerProbe, item.GetType())
			}
		}
	})

	t.Run("Search_WithNilName", func(t *testing.T) {
		validProbe := createAzureLoadBalancerProbeMinimal("valid-probe", loadBalancerName, subscriptionID, resourceGroup)

		mockClient := mocks.NewMockLoadBalancerProbesClient(ctrl)
		mockPager := &mockLoadBalancerProbePager{
			pages: []armnetwork.LoadBalancerProbesClientListResponse{
				{
					LoadBalancerProbeListResult: armnetwork.LoadBalancerProbeListResult{
						Value: []*armnetwork.Probe{
							{Name: nil, ID: new("/some/id")},
							validProbe,
						},
					},
				},
			},
		}

		testClient := &testLoadBalancerProbeClient{
			MockLoadBalancerProbesClient: mockClient,
			pager:                        mockPager,
		}

		wrapper := manual.NewNetworkLoadBalancerProbe(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		searchable := adapter.(discovery.SearchableAdapter)
		sdpItems, err := searchable.Search(ctx, wrapper.Scopes()[0], loadBalancerName, true)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if len(sdpItems) != 1 {
			t.Fatalf("Expected 1 item (nil name skipped), got: %d", len(sdpItems))
		}
		expectedValue := shared.CompositeLookupKey(loadBalancerName, "valid-probe")
		if sdpItems[0].UniqueAttributeValue() != expectedValue {
			t.Errorf("Expected unique value %s, got %s", expectedValue, sdpItems[0].UniqueAttributeValue())
		}
	})

	t.Run("Search_InvalidQueryParts", func(t *testing.T) {
		mockClient := mocks.NewMockLoadBalancerProbesClient(ctrl)
		testClient := &testLoadBalancerProbeClient{MockLoadBalancerProbesClient: mockClient}

		wrapper := manual.NewNetworkLoadBalancerProbe(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		_, qErr := wrapper.Search(ctx, wrapper.Scopes()[0])
		if qErr == nil {
			t.Error("Expected error when providing no query parts, but got nil")
		}
	})

	t.Run("Search_WithEmptyLoadBalancerName", func(t *testing.T) {
		mockClient := mocks.NewMockLoadBalancerProbesClient(ctrl)
		testClient := &testLoadBalancerProbeClient{MockLoadBalancerProbesClient: mockClient}

		wrapper := manual.NewNetworkLoadBalancerProbe(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		_, qErr := wrapper.Search(ctx, wrapper.Scopes()[0], "")
		if qErr == nil {
			t.Error("Expected error when loadBalancerName is empty, but got nil")
		}
	})

	t.Run("ErrorHandling_Get", func(t *testing.T) {
		expectedErr := errors.New("probe not found")

		mockClient := mocks.NewMockLoadBalancerProbesClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, loadBalancerName, "nonexistent-probe").Return(
			armnetwork.LoadBalancerProbesClientGetResponse{}, expectedErr)

		testClient := &testLoadBalancerProbeClient{MockLoadBalancerProbesClient: mockClient}
		wrapper := manual.NewNetworkLoadBalancerProbe(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(loadBalancerName, "nonexistent-probe")
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr == nil {
			t.Error("Expected error when getting non-existent probe, but got nil")
		}
	})

	t.Run("ErrorHandling_Search", func(t *testing.T) {
		mockClient := mocks.NewMockLoadBalancerProbesClient(ctrl)
		testClient := &testLoadBalancerProbeClient{
			MockLoadBalancerProbesClient: mockClient,
			pager:                        &errorLoadBalancerProbePager{},
		}

		wrapper := manual.NewNetworkLoadBalancerProbe(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		searchable := adapter.(discovery.SearchableAdapter)
		_, err := searchable.Search(ctx, wrapper.Scopes()[0], loadBalancerName, true)
		if err == nil {
			t.Error("Expected error from pager when NextPage returns an error, but got nil")
		}
	})

	t.Run("Get_NoProperties", func(t *testing.T) {
		probe := &armnetwork.Probe{
			ID:   new(fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/loadBalancers/%s/probes/%s", subscriptionID, resourceGroup, loadBalancerName, probeName)),
			Name: new(probeName),
			Type: new("Microsoft.Network/loadBalancers/probes"),
		}

		mockClient := mocks.NewMockLoadBalancerProbesClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, loadBalancerName, probeName).Return(
			armnetwork.LoadBalancerProbesClientGetResponse{
				Probe: *probe,
			}, nil)

		testClient := &testLoadBalancerProbeClient{MockLoadBalancerProbesClient: mockClient}
		wrapper := manual.NewNetworkLoadBalancerProbe(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(loadBalancerName, probeName)
		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		linkedQueries := sdpItem.GetLinkedItemQueries()
		if len(linkedQueries) != 1 {
			t.Errorf("Expected 1 linked query (parent LB only), got %d", len(linkedQueries))
		}
		if linkedQueries[0].GetQuery().GetType() != azureshared.NetworkLoadBalancer.String() {
			t.Errorf("Expected parent LB link, got type %s", linkedQueries[0].GetQuery().GetType())
		}
	})
}

func createAzureLoadBalancerProbe(name, lbName, subscriptionID, resourceGroup string) *armnetwork.Probe {
	provisioningState := armnetwork.ProvisioningStateSucceeded
	port := int32(80)
	protocol := armnetwork.ProbeProtocolHTTP
	intervalInSeconds := int32(15)
	numberOfProbes := int32(2)
	lbRuleID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/loadBalancers/%s/loadBalancingRules/lb-rule-1", subscriptionID, resourceGroup, lbName)

	return &armnetwork.Probe{
		ID:   new(fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/loadBalancers/%s/probes/%s", subscriptionID, resourceGroup, lbName, name)),
		Name: new(name),
		Type: new("Microsoft.Network/loadBalancers/probes"),
		Properties: &armnetwork.ProbePropertiesFormat{
			ProvisioningState: &provisioningState,
			Port:              &port,
			Protocol:          &protocol,
			IntervalInSeconds: &intervalInSeconds,
			NumberOfProbes:    &numberOfProbes,
			RequestPath:       new("/health"),
			LoadBalancingRules: []*armnetwork.SubResource{
				{ID: new(lbRuleID)},
			},
		},
	}
}

func createAzureLoadBalancerProbeMinimal(name, lbName, subscriptionID, resourceGroup string) *armnetwork.Probe {
	provisioningState := armnetwork.ProvisioningStateSucceeded
	port := int32(80)
	protocol := armnetwork.ProbeProtocolTCP
	return &armnetwork.Probe{
		ID:   new(fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/loadBalancers/%s/probes/%s", subscriptionID, resourceGroup, lbName, name)),
		Name: new(name),
		Type: new("Microsoft.Network/loadBalancers/probes"),
		Properties: &armnetwork.ProbePropertiesFormat{
			ProvisioningState: &provisioningState,
			Port:              &port,
			Protocol:          &protocol,
		},
	}
}
