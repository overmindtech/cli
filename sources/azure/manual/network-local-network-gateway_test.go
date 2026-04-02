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

func TestNetworkLocalNetworkGateway(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"
	scope := subscriptionID + "." + resourceGroup

	t.Run("Get", func(t *testing.T) {
		gatewayName := "test-local-gateway"
		gw := createAzureLocalNetworkGateway(gatewayName)

		mockClient := mocks.NewMockLocalNetworkGatewaysClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, gatewayName, nil).Return(
			armnetwork.LocalNetworkGatewaysClientGetResponse{
				LocalNetworkGateway: *gw,
			}, nil)

		wrapper := manual.NewNetworkLocalNetworkGateway(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, scope, gatewayName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.NetworkLocalNetworkGateway.String() {
			t.Errorf("Expected type %s, got %s", azureshared.NetworkLocalNetworkGateway.String(), sdpItem.GetType())
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
					ExpectedType:   stdlib.NetworkIP.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "203.0.113.1",
					ExpectedScope:  "global",
				},
			}
			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("Get_WithFqdn", func(t *testing.T) {
		gatewayName := "test-local-gateway-fqdn"
		gw := createAzureLocalNetworkGatewayWithFqdn(gatewayName)

		mockClient := mocks.NewMockLocalNetworkGatewaysClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, gatewayName, nil).Return(
			armnetwork.LocalNetworkGatewaysClientGetResponse{
				LocalNetworkGateway: *gw,
			}, nil)

		wrapper := manual.NewNetworkLocalNetworkGateway(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, scope, gatewayName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					ExpectedType:   stdlib.NetworkDNS.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  "vpn.example.com",
					ExpectedScope:  "global",
				},
			}
			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("Get_WithBgpSettings", func(t *testing.T) {
		gatewayName := "test-local-gateway-bgp"
		gw := createAzureLocalNetworkGatewayWithBgp(gatewayName)

		mockClient := mocks.NewMockLocalNetworkGatewaysClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, gatewayName, nil).Return(
			armnetwork.LocalNetworkGatewaysClientGetResponse{
				LocalNetworkGateway: *gw,
			}, nil)

		wrapper := manual.NewNetworkLocalNetworkGateway(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, scope, gatewayName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					ExpectedType:   stdlib.NetworkIP.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "203.0.113.1",
					ExpectedScope:  "global",
				},
				{
					ExpectedType:   stdlib.NetworkIP.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "10.0.0.1",
					ExpectedScope:  "global",
				},
			}
			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("GetWithEmptyName", func(t *testing.T) {
		mockClient := mocks.NewMockLocalNetworkGatewaysClient(ctrl)

		wrapper := manual.NewNetworkLocalNetworkGateway(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, scope, "", true)
		if qErr == nil {
			t.Error("Expected error when getting gateway with empty name, but got nil")
		}
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		gatewayName := "nonexistent-gateway"
		expectedErr := errors.New("local network gateway not found")

		mockClient := mocks.NewMockLocalNetworkGatewaysClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, gatewayName, nil).Return(
			armnetwork.LocalNetworkGatewaysClientGetResponse{}, expectedErr)

		wrapper := manual.NewNetworkLocalNetworkGateway(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, scope, gatewayName, true)
		if qErr == nil {
			t.Fatal("Expected error when gateway not found, got nil")
		}
	})

	t.Run("List", func(t *testing.T) {
		gw1 := createAzureLocalNetworkGateway("local-gateway-1")
		gw2 := createAzureLocalNetworkGateway("local-gateway-2")

		mockClient := mocks.NewMockLocalNetworkGatewaysClient(ctrl)
		mockPager := newMockLocalNetworkGatewaysPager(ctrl, []*armnetwork.LocalNetworkGateway{gw1, gw2})

		mockClient.EXPECT().NewListPager(resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewNetworkLocalNetworkGateway(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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
			if item.GetType() != azureshared.NetworkLocalNetworkGateway.String() {
				t.Errorf("Item %d: expected type %s, got %s", i, azureshared.NetworkLocalNetworkGateway.String(), item.GetType())
			}
			if item.Validate() != nil {
				t.Errorf("Item %d: validation error: %v", i, item.Validate())
			}
		}
	})

	t.Run("ListStream", func(t *testing.T) {
		gw1 := createAzureLocalNetworkGateway("local-gateway-1")
		gw2 := createAzureLocalNetworkGateway("local-gateway-2")

		mockClient := mocks.NewMockLocalNetworkGatewaysClient(ctrl)
		mockPager := newMockLocalNetworkGatewaysPager(ctrl, []*armnetwork.LocalNetworkGateway{gw1, gw2})

		mockClient.EXPECT().NewListPager(resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewNetworkLocalNetworkGateway(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		listStream, ok := adapter.(discovery.ListStreamableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support ListStream operation")
		}

		var received []*sdp.Item
		stream := &localNetworkGatewayCollectingStream{items: &received}
		listStream.ListStream(ctx, scope, true, stream)

		if len(received) != 2 {
			t.Fatalf("Expected 2 items from stream, got %d", len(received))
		}
	})

	t.Run("List_NilNameSkipped", func(t *testing.T) {
		gw1 := createAzureLocalNetworkGateway("local-gateway-1")
		gw2NilName := createAzureLocalNetworkGateway("local-gateway-2")
		gw2NilName.Name = nil

		mockClient := mocks.NewMockLocalNetworkGatewaysClient(ctrl)
		mockPager := newMockLocalNetworkGatewaysPager(ctrl, []*armnetwork.LocalNetworkGateway{gw1, gw2NilName})

		mockClient.EXPECT().NewListPager(resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewNetworkLocalNetworkGateway(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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
		if items[0].UniqueAttributeValue() != "local-gateway-1" {
			t.Errorf("Expected only local-gateway-1, got %s", items[0].UniqueAttributeValue())
		}
	})

	t.Run("GetLookups", func(t *testing.T) {
		wrapper := manual.NewNetworkLocalNetworkGateway(nil, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		lookups := wrapper.GetLookups()
		if len(lookups) == 0 {
			t.Error("Expected GetLookups to return at least one lookup")
		}
		found := false
		for _, l := range lookups {
			if l.ItemType.String() == azureshared.NetworkLocalNetworkGateway.String() {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected GetLookups to include NetworkLocalNetworkGateway")
		}
	})

	t.Run("PotentialLinks", func(t *testing.T) {
		wrapper := manual.NewNetworkLocalNetworkGateway(nil, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		potentialLinks := wrapper.PotentialLinks()
		for _, linkType := range []shared.ItemType{
			stdlib.NetworkIP,
			stdlib.NetworkDNS,
		} {
			if !potentialLinks[linkType] {
				t.Errorf("Expected PotentialLinks to include %s", linkType)
			}
		}
	})
}

type localNetworkGatewayCollectingStream struct {
	items *[]*sdp.Item
}

func (c *localNetworkGatewayCollectingStream) SendItem(item *sdp.Item) {
	*c.items = append(*c.items, item)
}

func (c *localNetworkGatewayCollectingStream) SendError(err error) {}

type mockLocalNetworkGatewaysPager struct {
	ctrl  *gomock.Controller
	items []*armnetwork.LocalNetworkGateway
	index int
	more  bool
}

func newMockLocalNetworkGatewaysPager(ctrl *gomock.Controller, items []*armnetwork.LocalNetworkGateway) *mockLocalNetworkGatewaysPager {
	return &mockLocalNetworkGatewaysPager{
		ctrl:  ctrl,
		items: items,
		index: 0,
		more:  len(items) > 0,
	}
}

func (m *mockLocalNetworkGatewaysPager) More() bool {
	return m.more
}

func (m *mockLocalNetworkGatewaysPager) NextPage(ctx context.Context) (armnetwork.LocalNetworkGatewaysClientListResponse, error) {
	if m.index >= len(m.items) {
		m.more = false
		return armnetwork.LocalNetworkGatewaysClientListResponse{
			LocalNetworkGatewayListResult: armnetwork.LocalNetworkGatewayListResult{
				Value: []*armnetwork.LocalNetworkGateway{},
			},
		}, nil
	}
	item := m.items[m.index]
	m.index++
	m.more = m.index < len(m.items)
	return armnetwork.LocalNetworkGatewaysClientListResponse{
		LocalNetworkGatewayListResult: armnetwork.LocalNetworkGatewayListResult{
			Value: []*armnetwork.LocalNetworkGateway{item},
		},
	}, nil
}

func createAzureLocalNetworkGateway(name string) *armnetwork.LocalNetworkGateway {
	provisioningState := armnetwork.ProvisioningStateSucceeded
	gatewayIP := "203.0.113.1"
	return &armnetwork.LocalNetworkGateway{
		ID:       new("/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.Network/localNetworkGateways/" + name),
		Name:     new(name),
		Type:     new("Microsoft.Network/localNetworkGateways"),
		Location: new("eastus"),
		Tags: map[string]*string{
			"env":     new("test"),
			"project": new("testing"),
		},
		Properties: &armnetwork.LocalNetworkGatewayPropertiesFormat{
			ProvisioningState: &provisioningState,
			GatewayIPAddress:  &gatewayIP,
			LocalNetworkAddressSpace: &armnetwork.AddressSpace{
				AddressPrefixes: []*string{
					new("10.1.0.0/16"),
					new("10.2.0.0/16"),
				},
			},
		},
	}
}

func createAzureLocalNetworkGatewayWithFqdn(name string) *armnetwork.LocalNetworkGateway {
	provisioningState := armnetwork.ProvisioningStateSucceeded
	fqdn := "vpn.example.com"
	return &armnetwork.LocalNetworkGateway{
		ID:       new("/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.Network/localNetworkGateways/" + name),
		Name:     new(name),
		Type:     new("Microsoft.Network/localNetworkGateways"),
		Location: new("eastus"),
		Tags: map[string]*string{
			"env": new("test"),
		},
		Properties: &armnetwork.LocalNetworkGatewayPropertiesFormat{
			ProvisioningState: &provisioningState,
			Fqdn:              &fqdn,
			LocalNetworkAddressSpace: &armnetwork.AddressSpace{
				AddressPrefixes: []*string{
					new("10.1.0.0/16"),
				},
			},
		},
	}
}

func createAzureLocalNetworkGatewayWithBgp(name string) *armnetwork.LocalNetworkGateway {
	provisioningState := armnetwork.ProvisioningStateSucceeded
	gatewayIP := "203.0.113.1"
	bgpPeeringAddress := "10.0.0.1"
	asn := int64(65001)
	return &armnetwork.LocalNetworkGateway{
		ID:       new("/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.Network/localNetworkGateways/" + name),
		Name:     new(name),
		Type:     new("Microsoft.Network/localNetworkGateways"),
		Location: new("eastus"),
		Tags: map[string]*string{
			"env": new("test"),
		},
		Properties: &armnetwork.LocalNetworkGatewayPropertiesFormat{
			ProvisioningState: &provisioningState,
			GatewayIPAddress:  &gatewayIP,
			BgpSettings: &armnetwork.BgpSettings{
				Asn:               &asn,
				BgpPeeringAddress: &bgpPeeringAddress,
			},
			LocalNetworkAddressSpace: &armnetwork.AddressSpace{
				AddressPrefixes: []*string{
					new("10.1.0.0/16"),
				},
			},
		},
	}
}
