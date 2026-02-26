package manual_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v9"
	"go.uber.org/mock/gomock"

	"github.com/overmindtech/cli/go/discovery"
	sdp "github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	"github.com/overmindtech/cli/sources/azure/manual"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/azure/shared/mocks"
	"github.com/overmindtech/cli/sources/shared"
)

type mockVirtualNetworkPeeringsPager struct {
	pages []armnetwork.VirtualNetworkPeeringsClientListResponse
	index int
}

func (m *mockVirtualNetworkPeeringsPager) More() bool {
	return m.index < len(m.pages)
}

func (m *mockVirtualNetworkPeeringsPager) NextPage(ctx context.Context) (armnetwork.VirtualNetworkPeeringsClientListResponse, error) {
	if m.index >= len(m.pages) {
		return armnetwork.VirtualNetworkPeeringsClientListResponse{}, errors.New("no more pages")
	}
	page := m.pages[m.index]
	m.index++
	return page, nil
}

type errorVirtualNetworkPeeringsPager struct{}

func (e *errorVirtualNetworkPeeringsPager) More() bool {
	return true
}

func (e *errorVirtualNetworkPeeringsPager) NextPage(ctx context.Context) (armnetwork.VirtualNetworkPeeringsClientListResponse, error) {
	return armnetwork.VirtualNetworkPeeringsClientListResponse{}, errors.New("pager error")
}

type testVirtualNetworkPeeringsClient struct {
	*mocks.MockVirtualNetworkPeeringsClient
	pager clients.VirtualNetworkPeeringsPager
}

func (t *testVirtualNetworkPeeringsClient) NewListPager(resourceGroupName, virtualNetworkName string, options *armnetwork.VirtualNetworkPeeringsClientListOptions) clients.VirtualNetworkPeeringsPager {
	return t.pager
}

func TestNetworkVirtualNetworkPeering(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"
	virtualNetworkName := "test-vnet"
	peeringName := "test-peering"

	t.Run("Get", func(t *testing.T) {
		peering := createAzureVirtualNetworkPeering(peeringName, virtualNetworkName)

		mockClient := mocks.NewMockVirtualNetworkPeeringsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, virtualNetworkName, peeringName, nil).Return(
			armnetwork.VirtualNetworkPeeringsClientGetResponse{
				VirtualNetworkPeering: *peering,
			}, nil)

		testClient := &testVirtualNetworkPeeringsClient{MockVirtualNetworkPeeringsClient: mockClient}
		wrapper := manual.NewNetworkVirtualNetworkPeering(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(virtualNetworkName, peeringName)
		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.NetworkVirtualNetworkPeering.String() {
			t.Errorf("Expected type %s, got %s", azureshared.NetworkVirtualNetworkPeering, sdpItem.GetType())
		}

		if sdpItem.GetUniqueAttribute() != "uniqueAttr" {
			t.Errorf("Expected unique attribute 'uniqueAttr', got %s", sdpItem.GetUniqueAttribute())
		}

		if sdpItem.UniqueAttributeValue() != shared.CompositeLookupKey(virtualNetworkName, peeringName) {
			t.Errorf("Expected unique attribute value %s, got %s", shared.CompositeLookupKey(virtualNetworkName, peeringName), sdpItem.UniqueAttributeValue())
		}

		if sdpItem.GetScope() != subscriptionID+"."+resourceGroup {
			t.Errorf("Expected scope %s, got %s", subscriptionID+"."+resourceGroup, sdpItem.GetScope())
		}

		if err := sdpItem.Validate(); err != nil {
			t.Fatalf("Expected no validation error, got: %v", err)
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					ExpectedType:   azureshared.NetworkVirtualNetwork.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  virtualNetworkName,
					ExpectedScope:  subscriptionID + "." + resourceGroup,
				},
			}
			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("Get_EmptyPeeringName", func(t *testing.T) {
		mockClient := mocks.NewMockVirtualNetworkPeeringsClient(ctrl)
		testClient := &testVirtualNetworkPeeringsClient{MockVirtualNetworkPeeringsClient: mockClient}

		wrapper := manual.NewNetworkVirtualNetworkPeering(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(virtualNetworkName, "")
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr == nil {
			t.Error("Expected error when peering name is empty, but got nil")
		}
	})

	t.Run("Get_InvalidQueryParts", func(t *testing.T) {
		mockClient := mocks.NewMockVirtualNetworkPeeringsClient(ctrl)
		testClient := &testVirtualNetworkPeeringsClient{MockVirtualNetworkPeeringsClient: mockClient}

		wrapper := manual.NewNetworkVirtualNetworkPeering(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], virtualNetworkName, true)
		if qErr == nil {
			t.Error("Expected error when providing insufficient query parts, but got nil")
		}
	})

	t.Run("Search", func(t *testing.T) {
		peering1 := createAzureVirtualNetworkPeering("peering-1", virtualNetworkName)
		peering2 := createAzureVirtualNetworkPeering("peering-2", virtualNetworkName)

		mockClient := mocks.NewMockVirtualNetworkPeeringsClient(ctrl)
		mockPager := &mockVirtualNetworkPeeringsPager{
			pages: []armnetwork.VirtualNetworkPeeringsClientListResponse{
				{
					VirtualNetworkPeeringListResult: armnetwork.VirtualNetworkPeeringListResult{
						Value: []*armnetwork.VirtualNetworkPeering{peering1, peering2},
					},
				},
			},
		}

		testClient := &testVirtualNetworkPeeringsClient{
			MockVirtualNetworkPeeringsClient: mockClient,
			pager:                             mockPager,
		}

		wrapper := manual.NewNetworkVirtualNetworkPeering(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support Search operation")
		}

		sdpItems, err := searchable.Search(ctx, wrapper.Scopes()[0], virtualNetworkName, true)
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
			if item.GetType() != azureshared.NetworkVirtualNetworkPeering.String() {
				t.Errorf("Expected type %s, got %s", azureshared.NetworkVirtualNetworkPeering, item.GetType())
			}
		}
	})

	t.Run("Search_InvalidQueryParts", func(t *testing.T) {
		mockClient := mocks.NewMockVirtualNetworkPeeringsClient(ctrl)
		testClient := &testVirtualNetworkPeeringsClient{MockVirtualNetworkPeeringsClient: mockClient}

		wrapper := manual.NewNetworkVirtualNetworkPeering(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		_, qErr := wrapper.Search(ctx, wrapper.Scopes()[0])
		if qErr == nil {
			t.Error("Expected error when providing no query parts, but got nil")
		}
	})

	t.Run("Search_PeeringWithNilName", func(t *testing.T) {
		validPeering := createAzureVirtualNetworkPeering("valid-peering", virtualNetworkName)

		mockClient := mocks.NewMockVirtualNetworkPeeringsClient(ctrl)
		mockPager := &mockVirtualNetworkPeeringsPager{
			pages: []armnetwork.VirtualNetworkPeeringsClientListResponse{
				{
					VirtualNetworkPeeringListResult: armnetwork.VirtualNetworkPeeringListResult{
						Value: []*armnetwork.VirtualNetworkPeering{
							{Name: nil, ID: strPtr("/some/id")},
							validPeering,
						},
					},
				},
			},
		}

		testClient := &testVirtualNetworkPeeringsClient{
			MockVirtualNetworkPeeringsClient: mockClient,
			pager:                             mockPager,
		}

		wrapper := manual.NewNetworkVirtualNetworkPeering(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		searchable := adapter.(discovery.SearchableAdapter)
		sdpItems, err := searchable.Search(ctx, wrapper.Scopes()[0], virtualNetworkName, true)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if len(sdpItems) != 1 {
			t.Fatalf("Expected 1 item (nil name skipped), got: %d", len(sdpItems))
		}
		if sdpItems[0].UniqueAttributeValue() != shared.CompositeLookupKey(virtualNetworkName, "valid-peering") {
			t.Errorf("Expected unique value %s, got %s", shared.CompositeLookupKey(virtualNetworkName, "valid-peering"), sdpItems[0].UniqueAttributeValue())
		}
	})

	t.Run("ErrorHandling_Get", func(t *testing.T) {
		expectedErr := errors.New("peering not found")

		mockClient := mocks.NewMockVirtualNetworkPeeringsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, virtualNetworkName, "nonexistent-peering", nil).Return(
			armnetwork.VirtualNetworkPeeringsClientGetResponse{}, expectedErr)

		testClient := &testVirtualNetworkPeeringsClient{MockVirtualNetworkPeeringsClient: mockClient}
		wrapper := manual.NewNetworkVirtualNetworkPeering(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(virtualNetworkName, "nonexistent-peering")
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr == nil {
			t.Error("Expected error when getting non-existent peering, but got nil")
		}
	})

	t.Run("ErrorHandling_Search", func(t *testing.T) {
		mockClient := mocks.NewMockVirtualNetworkPeeringsClient(ctrl)
		testClient := &testVirtualNetworkPeeringsClient{
			MockVirtualNetworkPeeringsClient: mockClient,
			pager:                             &errorVirtualNetworkPeeringsPager{},
		}

		wrapper := manual.NewNetworkVirtualNetworkPeering(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		searchable := adapter.(discovery.SearchableAdapter)
		_, err := searchable.Search(ctx, wrapper.Scopes()[0], virtualNetworkName, true)
		if err == nil {
			t.Error("Expected error from pager when NextPage returns an error, but got nil")
		}
	})
}

func createAzureVirtualNetworkPeering(peeringName, vnetName string) *armnetwork.VirtualNetworkPeering {
	idStr := "/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.Network/virtualNetworks/" + vnetName + "/virtualNetworkPeerings/" + peeringName
	typeStr := "Microsoft.Network/virtualNetworks/virtualNetworkPeerings"
	provisioningState := armnetwork.ProvisioningStateSucceeded
	return &armnetwork.VirtualNetworkPeering{
		ID:   &idStr,
		Name: &peeringName,
		Type: &typeStr,
		Properties: &armnetwork.VirtualNetworkPeeringPropertiesFormat{
			ProvisioningState: &provisioningState,
		},
	}
}

func strPtr(s string) *string {
	return &s
}
