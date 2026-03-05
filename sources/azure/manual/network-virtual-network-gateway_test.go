package manual_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v9"
	"go.uber.org/mock/gomock"

	"github.com/overmindtech/cli/go/discovery"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/manual"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/azure/shared/mocks"
	"github.com/overmindtech/cli/sources/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

func TestNetworkVirtualNetworkGateway(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"
	scope := subscriptionID + "." + resourceGroup

	t.Run("Get", func(t *testing.T) {
		gatewayName := "test-gateway"
		gw := createAzureVirtualNetworkGateway(gatewayName)

		mockClient := mocks.NewMockVirtualNetworkGatewaysClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, gatewayName, nil).Return(
			armnetwork.VirtualNetworkGatewaysClientGetResponse{
				VirtualNetworkGateway: *gw,
			}, nil)

		wrapper := manual.NewNetworkVirtualNetworkGateway(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, scope, gatewayName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.NetworkVirtualNetworkGateway.String() {
			t.Errorf("Expected type %s, got %s", azureshared.NetworkVirtualNetworkGateway.String(), sdpItem.GetType())
		}

		if sdpItem.GetUniqueAttribute() != "name" {
			t.Errorf("Expected unique attribute 'name', got %s", sdpItem.GetUniqueAttribute())
		}

		if sdpItem.UniqueAttributeValue() != gatewayName {
			t.Errorf("Expected unique attribute value %s, got %s", gatewayName, sdpItem.UniqueAttributeValue())
		}

		if sdpItem.GetTags()["env"] != "test" {
			t.Errorf("Expected tag 'env=test', got: %v", sdpItem.GetTags()["env"])
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					ExpectedType:   azureshared.NetworkVirtualNetworkGatewayConnection.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  gatewayName,
					ExpectedScope:  scope,
				},
			}
			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("Get_WithLinkedResources", func(t *testing.T) {
		gatewayName := "test-gateway-with-links"
		gw := createAzureVirtualNetworkGatewayWithLinks(gatewayName, subscriptionID, resourceGroup)

		mockClient := mocks.NewMockVirtualNetworkGatewaysClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, gatewayName, nil).Return(
			armnetwork.VirtualNetworkGatewaysClientGetResponse{
				VirtualNetworkGateway: *gw,
			}, nil)

		wrapper := manual.NewNetworkVirtualNetworkGateway(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, scope, gatewayName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					ExpectedType:   azureshared.NetworkSubnet.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey("test-vnet", "GatewaySubnet"),
					ExpectedScope:  scope,
				},
				{
					ExpectedType:   azureshared.NetworkPublicIPAddress.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-gateway-pip",
					ExpectedScope:  scope,
				},
				{
					ExpectedType:   stdlib.NetworkIP.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "10.0.1.4",
					ExpectedScope:  "global",
				},
				{
					ExpectedType:   stdlib.NetworkIP.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "10.0.0.5",
					ExpectedScope:  "global",
				},
				{
					ExpectedType:   azureshared.NetworkVirtualNetworkGatewayConnection.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  gatewayName,
					ExpectedScope:  scope,
				},
			}
			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("GetWithEmptyName", func(t *testing.T) {
		mockClient := mocks.NewMockVirtualNetworkGatewaysClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, "", nil).Return(
			armnetwork.VirtualNetworkGatewaysClientGetResponse{}, errors.New("virtual network gateway not found"))

		wrapper := manual.NewNetworkVirtualNetworkGateway(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, scope, "", true)
		if qErr == nil {
			t.Error("Expected error when getting gateway with empty name, but got nil")
		}
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		gatewayName := "nonexistent-gateway"
		expectedErr := errors.New("virtual network gateway not found")

		mockClient := mocks.NewMockVirtualNetworkGatewaysClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, gatewayName, nil).Return(
			armnetwork.VirtualNetworkGatewaysClientGetResponse{}, expectedErr)

		wrapper := manual.NewNetworkVirtualNetworkGateway(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, scope, gatewayName, true)
		if qErr == nil {
			t.Fatal("Expected error when gateway not found, got nil")
		}
	})

	t.Run("List", func(t *testing.T) {
		gw1 := createAzureVirtualNetworkGateway("gateway-1")
		gw2 := createAzureVirtualNetworkGateway("gateway-2")

		mockClient := mocks.NewMockVirtualNetworkGatewaysClient(ctrl)
		mockPager := newMockVirtualNetworkGatewaysPager(ctrl, []*armnetwork.VirtualNetworkGateway{gw1, gw2})

		mockClient.EXPECT().NewListPager(resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewNetworkVirtualNetworkGateway(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		listable, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support List operation")
		}

		items, err := listable.List(ctx, scope, true)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if len(items) != 2 {
			t.Fatalf("Expected 2 items, got %d", len(items))
		}

		for i, item := range items {
			if item.GetType() != azureshared.NetworkVirtualNetworkGateway.String() {
				t.Errorf("Item %d: expected type %s, got %s", i, azureshared.NetworkVirtualNetworkGateway.String(), item.GetType())
			}
			if item.Validate() != nil {
				t.Errorf("Item %d: validation error: %v", i, item.Validate())
			}
		}
	})

	t.Run("ListStream", func(t *testing.T) {
		gw1 := createAzureVirtualNetworkGateway("gateway-1")
		gw2 := createAzureVirtualNetworkGateway("gateway-2")

		mockClient := mocks.NewMockVirtualNetworkGatewaysClient(ctrl)
		mockPager := newMockVirtualNetworkGatewaysPager(ctrl, []*armnetwork.VirtualNetworkGateway{gw1, gw2})

		mockClient.EXPECT().NewListPager(resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewNetworkVirtualNetworkGateway(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		listStream, ok := adapter.(discovery.ListStreamableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support ListStream operation")
		}

		var received []*sdp.Item
		stream := &collectingStream{items: &received}
		listStream.ListStream(ctx, scope, true, stream)

		if len(received) != 2 {
			t.Fatalf("Expected 2 items from stream, got %d", len(received))
		}
	})

	t.Run("List_NilNameSkipped", func(t *testing.T) {
		gw1 := createAzureVirtualNetworkGateway("gateway-1")
		gw2NilName := createAzureVirtualNetworkGateway("gateway-2")
		gw2NilName.Name = nil

		mockClient := mocks.NewMockVirtualNetworkGatewaysClient(ctrl)
		mockPager := newMockVirtualNetworkGatewaysPager(ctrl, []*armnetwork.VirtualNetworkGateway{gw1, gw2NilName})

		mockClient.EXPECT().NewListPager(resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewNetworkVirtualNetworkGateway(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		listable, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support List operation")
		}

		items, err := listable.List(ctx, scope, true)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if len(items) != 1 {
			t.Fatalf("Expected 1 item (nil name skipped), got %d", len(items))
		}
		if items[0].UniqueAttributeValue() != "gateway-1" {
			t.Errorf("Expected only gateway-1, got %s", items[0].UniqueAttributeValue())
		}
	})

	t.Run("GetLookups", func(t *testing.T) {
		wrapper := manual.NewNetworkVirtualNetworkGateway(nil, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		lookups := wrapper.GetLookups()
		if len(lookups) == 0 {
			t.Error("Expected GetLookups to return at least one lookup")
		}
		found := false
		for _, l := range lookups {
			if l.ItemType.String() == azureshared.NetworkVirtualNetworkGateway.String() {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected GetLookups to include NetworkVirtualNetworkGateway")
		}
	})

	t.Run("PotentialLinks", func(t *testing.T) {
		wrapper := manual.NewNetworkVirtualNetworkGateway(nil, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		potentialLinks := wrapper.PotentialLinks()
		for _, linkType := range []shared.ItemType{
			azureshared.NetworkSubnet,
			azureshared.NetworkPublicIPAddress,
			azureshared.NetworkLocalNetworkGateway,
			azureshared.NetworkVirtualNetworkGatewayConnection,
			azureshared.ExtendedLocationCustomLocation,
			azureshared.ManagedIdentityUserAssignedIdentity,
			azureshared.NetworkVirtualNetwork,
			stdlib.NetworkIP,
			stdlib.NetworkDNS,
		} {
			if !potentialLinks[linkType] {
				t.Errorf("Expected PotentialLinks to include %s", linkType)
			}
		}
	})
}

type collectingStream struct {
	items *[]*sdp.Item
}

func (c *collectingStream) SendItem(item *sdp.Item) {
	*c.items = append(*c.items, item)
}

func (c *collectingStream) SendError(err error) {}

type mockVirtualNetworkGatewaysPager struct {
	ctrl  *gomock.Controller
	items []*armnetwork.VirtualNetworkGateway
	index int
	more  bool
}

func newMockVirtualNetworkGatewaysPager(ctrl *gomock.Controller, items []*armnetwork.VirtualNetworkGateway) *mockVirtualNetworkGatewaysPager {
	return &mockVirtualNetworkGatewaysPager{
		ctrl:  ctrl,
		items: items,
		index: 0,
		more:  len(items) > 0,
	}
}

func (m *mockVirtualNetworkGatewaysPager) More() bool {
	return m.more
}

func (m *mockVirtualNetworkGatewaysPager) NextPage(ctx context.Context) (armnetwork.VirtualNetworkGatewaysClientListResponse, error) {
	if m.index >= len(m.items) {
		m.more = false
		return armnetwork.VirtualNetworkGatewaysClientListResponse{
			VirtualNetworkGatewayListResult: armnetwork.VirtualNetworkGatewayListResult{
				Value: []*armnetwork.VirtualNetworkGateway{},
			},
		}, nil
	}
	item := m.items[m.index]
	m.index++
	m.more = m.index < len(m.items)
	return armnetwork.VirtualNetworkGatewaysClientListResponse{
		VirtualNetworkGatewayListResult: armnetwork.VirtualNetworkGatewayListResult{
			Value: []*armnetwork.VirtualNetworkGateway{item},
		},
	}, nil
}

func createAzureVirtualNetworkGateway(name string) *armnetwork.VirtualNetworkGateway {
	provisioningState := armnetwork.ProvisioningStateSucceeded
	gatewayType := armnetwork.VirtualNetworkGatewayTypeVPN
	vpnType := armnetwork.VPNTypeRouteBased
	return &armnetwork.VirtualNetworkGateway{
		ID:       new("/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.Network/virtualNetworkGateways/" + name),
		Name:     new(name),
		Type:     new("Microsoft.Network/virtualNetworkGateways"),
		Location: new("eastus"),
		Tags: map[string]*string{
			"env":     new("test"),
			"project": new("testing"),
		},
		Properties: &armnetwork.VirtualNetworkGatewayPropertiesFormat{
			ProvisioningState: &provisioningState,
			GatewayType:       &gatewayType,
			VPNType:           &vpnType,
		},
	}
}

func createAzureVirtualNetworkGatewayWithLinks(name, subscriptionID, resourceGroup string) *armnetwork.VirtualNetworkGateway {
	gw := createAzureVirtualNetworkGateway(name)
	subnetID := "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/GatewaySubnet"
	publicIPID := "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/publicIPAddresses/test-gateway-pip"
	privateIP := "10.0.1.4"
	inboundDNS := "10.0.0.5"
	gw.Properties.IPConfigurations = []*armnetwork.VirtualNetworkGatewayIPConfiguration{
		{
			ID:   new("/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/virtualNetworkGateways/" + name + "/ipConfigurations/default"),
			Name: new("default"),
			Properties: &armnetwork.VirtualNetworkGatewayIPConfigurationPropertiesFormat{
				Subnet: &armnetwork.SubResource{
					ID: new(subnetID),
				},
				PublicIPAddress: &armnetwork.SubResource{
					ID: new(publicIPID),
				},
				PrivateIPAddress: &privateIP,
			},
		},
	}
	gw.Properties.InboundDNSForwardingEndpoint = &inboundDNS
	return gw
}
