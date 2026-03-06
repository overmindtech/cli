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
)

func TestNetworkNatGateway(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"
	scope := subscriptionID + "." + resourceGroup

	t.Run("Get", func(t *testing.T) {
		natGatewayName := "test-nat-gateway"
		ng := createAzureNatGateway(natGatewayName)

		mockClient := mocks.NewMockNatGatewaysClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, natGatewayName, nil).Return(
			armnetwork.NatGatewaysClientGetResponse{
				NatGateway: *ng,
			}, nil)

		wrapper := manual.NewNetworkNatGateway(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, scope, natGatewayName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.NetworkNatGateway.String() {
			t.Errorf("Expected type %s, got %s", azureshared.NetworkNatGateway.String(), sdpItem.GetType())
		}

		if sdpItem.GetUniqueAttribute() != "name" {
			t.Errorf("Expected unique attribute 'name', got %s", sdpItem.GetUniqueAttribute())
		}

		if sdpItem.UniqueAttributeValue() != natGatewayName {
			t.Errorf("Expected unique attribute value %s, got %s", natGatewayName, sdpItem.UniqueAttributeValue())
		}

		if sdpItem.GetTags()["env"] != "test" {
			t.Errorf("Expected tag 'env=test', got: %v", sdpItem.GetTags()["env"])
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{}
			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("Get_WithLinkedResources", func(t *testing.T) {
		natGatewayName := "test-nat-gateway-with-links"
		ng := createAzureNatGatewayWithLinks(natGatewayName, subscriptionID, resourceGroup)

		mockClient := mocks.NewMockNatGatewaysClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, natGatewayName, nil).Return(
			armnetwork.NatGatewaysClientGetResponse{
				NatGateway: *ng,
			}, nil)

		wrapper := manual.NewNetworkNatGateway(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, scope, natGatewayName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					ExpectedType:   azureshared.NetworkPublicIPAddress.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-public-ip",
					ExpectedScope:  scope,
				},
				{
					ExpectedType:   azureshared.NetworkPublicIPPrefix.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-public-ip-prefix",
					ExpectedScope:  scope,
				},
				{
					ExpectedType:   azureshared.NetworkSubnet.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey("test-vnet", "test-subnet"),
					ExpectedScope:  scope,
				},
				{
					ExpectedType:   azureshared.NetworkVirtualNetwork.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "source-vnet",
					ExpectedScope:  scope,
				},
			}
			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("GetWithEmptyName", func(t *testing.T) {
		mockClient := mocks.NewMockNatGatewaysClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, "", nil).Return(
			armnetwork.NatGatewaysClientGetResponse{}, errors.New("nat gateway not found"))

		wrapper := manual.NewNetworkNatGateway(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, scope, "", true)
		if qErr == nil {
			t.Error("Expected error when getting nat gateway with empty name, but got nil")
		}
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		natGatewayName := "nonexistent-nat-gateway"
		expectedErr := errors.New("nat gateway not found")

		mockClient := mocks.NewMockNatGatewaysClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, natGatewayName, nil).Return(
			armnetwork.NatGatewaysClientGetResponse{}, expectedErr)

		wrapper := manual.NewNetworkNatGateway(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, scope, natGatewayName, true)
		if qErr == nil {
			t.Fatal("Expected error when nat gateway not found, got nil")
		}
	})

	t.Run("List", func(t *testing.T) {
		ng1 := createAzureNatGateway("nat-gateway-1")
		ng2 := createAzureNatGateway("nat-gateway-2")

		mockClient := mocks.NewMockNatGatewaysClient(ctrl)
		mockPager := newMockNatGatewaysPager(ctrl, []*armnetwork.NatGateway{ng1, ng2})

		mockClient.EXPECT().NewListPager(resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewNetworkNatGateway(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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
			if item.GetType() != azureshared.NetworkNatGateway.String() {
				t.Errorf("Item %d: expected type %s, got %s", i, azureshared.NetworkNatGateway.String(), item.GetType())
			}
			if item.Validate() != nil {
				t.Errorf("Item %d: validation error: %v", i, item.Validate())
			}
		}
	})

	t.Run("ListStream", func(t *testing.T) {
		ng1 := createAzureNatGateway("nat-gateway-1")
		ng2 := createAzureNatGateway("nat-gateway-2")

		mockClient := mocks.NewMockNatGatewaysClient(ctrl)
		mockPager := newMockNatGatewaysPager(ctrl, []*armnetwork.NatGateway{ng1, ng2})

		mockClient.EXPECT().NewListPager(resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewNetworkNatGateway(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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
		ng1 := createAzureNatGateway("nat-gateway-1")
		ng2NilName := createAzureNatGateway("nat-gateway-2")
		ng2NilName.Name = nil

		mockClient := mocks.NewMockNatGatewaysClient(ctrl)
		mockPager := newMockNatGatewaysPager(ctrl, []*armnetwork.NatGateway{ng1, ng2NilName})

		mockClient.EXPECT().NewListPager(resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewNetworkNatGateway(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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
		if items[0].UniqueAttributeValue() != "nat-gateway-1" {
			t.Errorf("Expected only nat-gateway-1, got %s", items[0].UniqueAttributeValue())
		}
	})

	t.Run("GetLookups", func(t *testing.T) {
		wrapper := manual.NewNetworkNatGateway(nil, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		lookups := wrapper.GetLookups()
		if len(lookups) == 0 {
			t.Error("Expected GetLookups to return at least one lookup")
		}
		found := false
		for _, l := range lookups {
			if l.ItemType.String() == azureshared.NetworkNatGateway.String() {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected GetLookups to include NetworkNatGateway")
		}
	})

	t.Run("PotentialLinks", func(t *testing.T) {
		wrapper := manual.NewNetworkNatGateway(nil, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		potentialLinks := wrapper.PotentialLinks()
		for _, linkType := range []shared.ItemType{
			azureshared.NetworkPublicIPAddress,
			azureshared.NetworkPublicIPPrefix,
			azureshared.NetworkSubnet,
			azureshared.NetworkVirtualNetwork,
		} {
			if !potentialLinks[linkType] {
				t.Errorf("Expected PotentialLinks to include %s", linkType)
			}
		}
	})
}

type mockNatGatewaysPager struct {
	ctrl  *gomock.Controller
	items []*armnetwork.NatGateway
	index int
	more  bool
}

func newMockNatGatewaysPager(ctrl *gomock.Controller, items []*armnetwork.NatGateway) *mockNatGatewaysPager {
	return &mockNatGatewaysPager{
		ctrl:  ctrl,
		items: items,
		index: 0,
		more:  len(items) > 0,
	}
}

func (m *mockNatGatewaysPager) More() bool {
	return m.more
}

func (m *mockNatGatewaysPager) NextPage(ctx context.Context) (armnetwork.NatGatewaysClientListResponse, error) {
	if m.index >= len(m.items) {
		m.more = false
		return armnetwork.NatGatewaysClientListResponse{
			NatGatewayListResult: armnetwork.NatGatewayListResult{
				Value: []*armnetwork.NatGateway{},
			},
		}, nil
	}
	item := m.items[m.index]
	m.index++
	m.more = m.index < len(m.items)
	return armnetwork.NatGatewaysClientListResponse{
		NatGatewayListResult: armnetwork.NatGatewayListResult{
			Value: []*armnetwork.NatGateway{item},
		},
	}, nil
}

func createAzureNatGateway(name string) *armnetwork.NatGateway {
	provisioningState := armnetwork.ProvisioningStateSucceeded
	return &armnetwork.NatGateway{
		ID:       new("/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.Network/natGateways/" + name),
		Name:     new(name),
		Type:     new("Microsoft.Network/natGateways"),
		Location: new("eastus"),
		Tags: map[string]*string{
			"env":     new("test"),
			"project": new("testing"),
		},
		Properties: &armnetwork.NatGatewayPropertiesFormat{
			ProvisioningState: &provisioningState,
		},
	}
}

func createAzureNatGatewayWithLinks(name, subscriptionID, resourceGroup string) *armnetwork.NatGateway {
	ng := createAzureNatGateway(name)
	baseID := "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network"
	publicIPID := baseID + "/publicIPAddresses/test-public-ip"
	publicIPPrefixID := baseID + "/publicIPPrefixes/test-public-ip-prefix"
	subnetID := baseID + "/virtualNetworks/test-vnet/subnets/test-subnet"
	sourceVnetID := baseID + "/virtualNetworks/source-vnet"

	ng.Properties.PublicIPAddresses = []*armnetwork.SubResource{
		{ID: new(publicIPID)},
	}
	ng.Properties.PublicIPPrefixes = []*armnetwork.SubResource{
		{ID: new(publicIPPrefixID)},
	}
	ng.Properties.Subnets = []*armnetwork.SubResource{
		{ID: new(subnetID)},
	}
	ng.Properties.SourceVirtualNetwork = &armnetwork.SubResource{
		ID: new(sourceVnetID),
	}
	return ng
}
