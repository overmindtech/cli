package manual_test

import (
	"context"
	"errors"
	"sync"
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
	"github.com/overmindtech/cli/sources/stdlib"
)

func TestNetworkVirtualNetworkGatewayConnection(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"

	t.Run("Get", func(t *testing.T) {
		connectionName := "test-vpn-connection"
		resource := createVirtualNetworkGatewayConnection(connectionName, subscriptionID, resourceGroup)

		mockClient := mocks.NewMockVirtualNetworkGatewayConnectionsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, connectionName, nil).Return(
			armnetwork.VirtualNetworkGatewayConnectionsClientGetResponse{
				VirtualNetworkGatewayConnection: *resource,
			}, nil)

		wrapper := manual.NewNetworkVirtualNetworkGatewayConnection(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], connectionName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.NetworkVirtualNetworkGatewayConnection.String() {
			t.Errorf("Expected type %s, got %s", azureshared.NetworkVirtualNetworkGatewayConnection.String(), sdpItem.GetType())
		}

		if sdpItem.GetUniqueAttribute() != "name" {
			t.Errorf("Expected unique attribute 'name', got %s", sdpItem.GetUniqueAttribute())
		}

		if sdpItem.UniqueAttributeValue() != connectionName {
			t.Errorf("Expected unique attribute value %s, got %s", connectionName, sdpItem.UniqueAttributeValue())
		}

		t.Run("StaticTests", func(t *testing.T) {
		queryTests := shared.QueryTests{
			{
				ExpectedType:   azureshared.NetworkVirtualNetworkGateway.String(),
				ExpectedMethod: sdp.QueryMethod_GET,
				ExpectedQuery:  "gateway1",
				ExpectedScope:  subscriptionID + "." + resourceGroup,
			},
			{
				ExpectedType:   azureshared.NetworkVirtualNetworkGateway.String(),
				ExpectedMethod: sdp.QueryMethod_GET,
				ExpectedQuery:  "gateway2",
				ExpectedScope:  subscriptionID + "." + resourceGroup,
			},
			{
				ExpectedType:   azureshared.NetworkLocalNetworkGateway.String(),
				ExpectedMethod: sdp.QueryMethod_GET,
				ExpectedQuery:  "local-gw",
				ExpectedScope:  subscriptionID + "." + resourceGroup,
			},
			{
				ExpectedType:   azureshared.NetworkExpressRouteCircuitPeering.String(),
				ExpectedMethod: sdp.QueryMethod_GET,
				ExpectedQuery:  "circuit1|peering1",
				ExpectedScope:  subscriptionID + "." + resourceGroup,
			},
			{
				ExpectedType:   azureshared.NetworkVirtualNetworkGatewayNatRule.String(),
				ExpectedMethod: sdp.QueryMethod_GET,
				ExpectedQuery:  "gateway1|egress-rule1",
				ExpectedScope:  subscriptionID + "." + resourceGroup,
			},
			{
				ExpectedType:   azureshared.NetworkVirtualNetworkGatewayNatRule.String(),
				ExpectedMethod: sdp.QueryMethod_GET,
				ExpectedQuery:  "gateway1|ingress-rule1",
				ExpectedScope:  subscriptionID + "." + resourceGroup,
			},
			{
				ExpectedType:   stdlib.NetworkIP.String(),
				ExpectedMethod: sdp.QueryMethod_GET,
				ExpectedQuery:  "10.0.0.1",
				ExpectedScope:  "global",
			},
			{
				ExpectedType:   azureshared.NetworkVirtualNetworkGatewayIPConfiguration.String(),
				ExpectedMethod: sdp.QueryMethod_GET,
				ExpectedQuery:  "gateway1|default",
				ExpectedScope:  subscriptionID + "." + resourceGroup,
			},
			{
				ExpectedType:   stdlib.NetworkIP.String(),
				ExpectedMethod: sdp.QueryMethod_GET,
				ExpectedQuery:  "10.0.0.2",
				ExpectedScope:  "global",
			},
			{
				ExpectedType:   azureshared.NetworkVirtualNetworkGatewayIPConfiguration.String(),
				ExpectedMethod: sdp.QueryMethod_GET,
				ExpectedQuery:  "gateway2|default",
				ExpectedScope:  subscriptionID + "." + resourceGroup,
			},
			{
				ExpectedType:   stdlib.NetworkIP.String(),
				ExpectedMethod: sdp.QueryMethod_GET,
				ExpectedQuery:  "192.168.1.1",
				ExpectedScope:  "global",
			},
			{
				ExpectedType:   stdlib.NetworkIP.String(),
				ExpectedMethod: sdp.QueryMethod_GET,
				ExpectedQuery:  "10.1.1.1",
				ExpectedScope:  "global",
			},
		}

			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("List", func(t *testing.T) {
		resource1 := createVirtualNetworkGatewayConnectionMinimal("vpn-conn-1", subscriptionID, resourceGroup)
		resource2 := createVirtualNetworkGatewayConnectionMinimal("vpn-conn-2", subscriptionID, resourceGroup)

		mockClient := mocks.NewMockVirtualNetworkGatewayConnectionsClient(ctrl)
		mockPager := newMockVirtualNetworkGatewayConnectionsPager(ctrl, []*armnetwork.VirtualNetworkGatewayConnection{resource1, resource2})

		mockClient.EXPECT().NewListPager(resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewNetworkVirtualNetworkGatewayConnection(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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
		}
	})

	t.Run("ListStream", func(t *testing.T) {
		resource1 := createVirtualNetworkGatewayConnectionMinimal("vpn-conn-1", subscriptionID, resourceGroup)
		resource2 := createVirtualNetworkGatewayConnectionMinimal("vpn-conn-2", subscriptionID, resourceGroup)

		mockClient := mocks.NewMockVirtualNetworkGatewayConnectionsClient(ctrl)
		mockPager := newMockVirtualNetworkGatewayConnectionsPager(ctrl, []*armnetwork.VirtualNetworkGatewayConnection{resource1, resource2})

		mockClient.EXPECT().NewListPager(resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewNetworkVirtualNetworkGatewayConnection(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		wg := &sync.WaitGroup{}
		wg.Add(2)

		var items []*sdp.Item
		mockItemHandler := func(item *sdp.Item) {
			items = append(items, item)
			wg.Done()
		}

		var errs []error
		mockErrorHandler := func(err error) {
			errs = append(errs, err)
		}

		stream := discovery.NewQueryResultStream(mockItemHandler, mockErrorHandler)

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
	})

	t.Run("ListWithNilName", func(t *testing.T) {
		resourceWithName := createVirtualNetworkGatewayConnectionMinimal("valid-conn", subscriptionID, resourceGroup)
		connTypeIPsec := armnetwork.VirtualNetworkGatewayConnectionTypeIPsec
		resourceWithNilName := &armnetwork.VirtualNetworkGatewayConnection{
			Name: nil,
			Properties: &armnetwork.VirtualNetworkGatewayConnectionPropertiesFormat{
				ConnectionType:         &connTypeIPsec,
				VirtualNetworkGateway1: &armnetwork.VirtualNetworkGateway{ID: new("/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/virtualNetworkGateways/gateway1")},
			},
		}

		mockClient := mocks.NewMockVirtualNetworkGatewayConnectionsClient(ctrl)
		mockPager := newMockVirtualNetworkGatewayConnectionsPager(ctrl, []*armnetwork.VirtualNetworkGatewayConnection{resourceWithNilName, resourceWithName})

		mockClient.EXPECT().NewListPager(resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewNetworkVirtualNetworkGatewayConnection(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		listable, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support List operation")
		}

		sdpItems, err := listable.List(ctx, wrapper.Scopes()[0], true)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if len(sdpItems) != 1 {
			t.Fatalf("Expected 1 item (nil name skipped), got: %d", len(sdpItems))
		}

		if sdpItems[0].UniqueAttributeValue() != "valid-conn" {
			t.Errorf("Expected valid-conn, got %s", sdpItems[0].UniqueAttributeValue())
		}
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		expectedErr := errors.New("resource not found")

		mockClient := mocks.NewMockVirtualNetworkGatewayConnectionsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, "nonexistent", nil).Return(
			armnetwork.VirtualNetworkGatewayConnectionsClientGetResponse{}, expectedErr)

		wrapper := manual.NewNetworkVirtualNetworkGatewayConnection(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "nonexistent", true)
		if qErr == nil {
			t.Error("Expected error when getting non-existent resource, but got nil")
		}
	})

	t.Run("GetWithEmptyName", func(t *testing.T) {
		mockClient := mocks.NewMockVirtualNetworkGatewayConnectionsClient(ctrl)

		wrapper := manual.NewNetworkVirtualNetworkGatewayConnection(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "", true)
		if qErr == nil {
			t.Error("Expected error when getting resource with empty name, but got nil")
		}
	})

	t.Run("HealthStatus", func(t *testing.T) {
		tests := []struct {
			name           string
			state          armnetwork.ProvisioningState
			expectedHealth sdp.Health
		}{
			{"Succeeded", armnetwork.ProvisioningStateSucceeded, sdp.Health_HEALTH_OK},
			{"Updating", armnetwork.ProvisioningStateUpdating, sdp.Health_HEALTH_PENDING},
			{"Deleting", armnetwork.ProvisioningStateDeleting, sdp.Health_HEALTH_PENDING},
			{"Failed", armnetwork.ProvisioningStateFailed, sdp.Health_HEALTH_ERROR},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				connTypeIPsec := armnetwork.VirtualNetworkGatewayConnectionTypeIPsec
				resource := &armnetwork.VirtualNetworkGatewayConnection{
					Name:     new("conn-" + tc.name),
					Location: new("eastus"),
					Type:     new("Microsoft.Network/connections"),
					ID:       new("/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/connections/conn-" + tc.name),
					Properties: &armnetwork.VirtualNetworkGatewayConnectionPropertiesFormat{
						ProvisioningState:      &tc.state,
						ConnectionType:         &connTypeIPsec,
						VirtualNetworkGateway1: &armnetwork.VirtualNetworkGateway{ID: new("/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/virtualNetworkGateways/gateway1")},
					},
				}

				mockClient := mocks.NewMockVirtualNetworkGatewayConnectionsClient(ctrl)
				mockClient.EXPECT().Get(ctx, resourceGroup, "conn-"+tc.name, nil).Return(
					armnetwork.VirtualNetworkGatewayConnectionsClientGetResponse{
						VirtualNetworkGatewayConnection: *resource,
					}, nil)

				wrapper := manual.NewNetworkVirtualNetworkGatewayConnection(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
				adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

				sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "conn-"+tc.name, true)
				if qErr != nil {
					t.Fatalf("Expected no error, got: %v", qErr)
				}

				if sdpItem.GetHealth() != tc.expectedHealth {
					t.Errorf("Expected health %v, got %v", tc.expectedHealth, sdpItem.GetHealth())
				}
			})
		}
	})
}

func createVirtualNetworkGatewayConnection(name, subscriptionID, resourceGroup string) *armnetwork.VirtualNetworkGatewayConnection {
	provisioningState := armnetwork.ProvisioningStateSucceeded
	connTypeVnet2Vnet := armnetwork.VirtualNetworkGatewayConnectionTypeVnet2Vnet
	return &armnetwork.VirtualNetworkGatewayConnection{
		Name:     new(name),
		Location: new("eastus"),
		Type:     new("Microsoft.Network/connections"),
		ID:       new("/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/connections/" + name),
		Tags: map[string]*string{
			"env": new("test"),
		},
		Properties: &armnetwork.VirtualNetworkGatewayConnectionPropertiesFormat{
			ProvisioningState: &provisioningState,
			ConnectionType:    &connTypeVnet2Vnet,
			VirtualNetworkGateway1: &armnetwork.VirtualNetworkGateway{
				ID: new("/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/virtualNetworkGateways/gateway1"),
			},
			VirtualNetworkGateway2: &armnetwork.VirtualNetworkGateway{
				ID: new("/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/virtualNetworkGateways/gateway2"),
			},
			LocalNetworkGateway2: &armnetwork.LocalNetworkGateway{
				ID: new("/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/localNetworkGateways/local-gw"),
			},
			Peer: &armnetwork.SubResource{
				ID: new("/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/expressRouteCircuits/circuit1/peerings/peering1"),
			},
			EgressNatRules: []*armnetwork.SubResource{
				{
					ID: new("/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/virtualNetworkGateways/gateway1/natRules/egress-rule1"),
				},
			},
			IngressNatRules: []*armnetwork.SubResource{
				{
					ID: new("/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/virtualNetworkGateways/gateway1/natRules/ingress-rule1"),
				},
			},
			GatewayCustomBgpIPAddresses: []*armnetwork.GatewayCustomBgpIPAddressIPConfiguration{
				{
					CustomBgpIPAddress: new("10.0.0.1"),
					IPConfigurationID:  new("/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/virtualNetworkGateways/gateway1/ipConfigurations/default"),
				},
				{
					CustomBgpIPAddress: new("10.0.0.2"),
					IPConfigurationID:  new("/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/virtualNetworkGateways/gateway2/ipConfigurations/default"),
				},
			},
			TunnelProperties: []*armnetwork.VirtualNetworkGatewayConnectionTunnelProperties{
				{
					TunnelIPAddress:   new("192.168.1.1"),
					BgpPeeringAddress: new("10.1.1.1"),
				},
			},
		},
	}
}

func createVirtualNetworkGatewayConnectionMinimal(name, subscriptionID, resourceGroup string) *armnetwork.VirtualNetworkGatewayConnection {
	provisioningState := armnetwork.ProvisioningStateSucceeded
	connTypeIPsec := armnetwork.VirtualNetworkGatewayConnectionTypeIPsec
	return &armnetwork.VirtualNetworkGatewayConnection{
		Name:     new(name),
		Location: new("eastus"),
		Type:     new("Microsoft.Network/connections"),
		ID:       new("/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/connections/" + name),
		Properties: &armnetwork.VirtualNetworkGatewayConnectionPropertiesFormat{
			ProvisioningState: &provisioningState,
			ConnectionType:    &connTypeIPsec,
			VirtualNetworkGateway1: &armnetwork.VirtualNetworkGateway{
				ID: new("/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/virtualNetworkGateways/gateway1"),
			},
		},
	}
}

type mockVirtualNetworkGatewayConnectionsPager struct {
	ctrl  *gomock.Controller
	items []*armnetwork.VirtualNetworkGatewayConnection
	index int
	more  bool
}

func newMockVirtualNetworkGatewayConnectionsPager(ctrl *gomock.Controller, items []*armnetwork.VirtualNetworkGatewayConnection) clients.VirtualNetworkGatewayConnectionsPager {
	return &mockVirtualNetworkGatewayConnectionsPager{
		ctrl:  ctrl,
		items: items,
		index: 0,
		more:  len(items) > 0,
	}
}

func (m *mockVirtualNetworkGatewayConnectionsPager) More() bool {
	return m.more
}

func (m *mockVirtualNetworkGatewayConnectionsPager) NextPage(ctx context.Context) (armnetwork.VirtualNetworkGatewayConnectionsClientListResponse, error) {
	if m.index >= len(m.items) {
		m.more = false
		return armnetwork.VirtualNetworkGatewayConnectionsClientListResponse{
			VirtualNetworkGatewayConnectionListResult: armnetwork.VirtualNetworkGatewayConnectionListResult{
				Value: []*armnetwork.VirtualNetworkGatewayConnection{},
			},
		}, nil
	}

	item := m.items[m.index]
	m.index++
	m.more = m.index < len(m.items)

	return armnetwork.VirtualNetworkGatewayConnectionsClientListResponse{
		VirtualNetworkGatewayConnectionListResult: armnetwork.VirtualNetworkGatewayConnectionListResult{
			Value: []*armnetwork.VirtualNetworkGatewayConnection{item},
		},
	}, nil
}
