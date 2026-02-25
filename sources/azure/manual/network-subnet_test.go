package manual_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
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

type mockSubnetsPager struct {
	pages []armnetwork.SubnetsClientListResponse
	index int
}

func (m *mockSubnetsPager) More() bool {
	return m.index < len(m.pages)
}

func (m *mockSubnetsPager) NextPage(ctx context.Context) (armnetwork.SubnetsClientListResponse, error) {
	if m.index >= len(m.pages) {
		return armnetwork.SubnetsClientListResponse{}, errors.New("no more pages")
	}
	page := m.pages[m.index]
	m.index++
	return page, nil
}

type errorSubnetsPager struct{}

func (e *errorSubnetsPager) More() bool {
	return true
}

func (e *errorSubnetsPager) NextPage(ctx context.Context) (armnetwork.SubnetsClientListResponse, error) {
	return armnetwork.SubnetsClientListResponse{}, errors.New("pager error")
}

type testSubnetsClient struct {
	*mocks.MockSubnetsClient
	pager clients.SubnetsPager
}

func (t *testSubnetsClient) NewListPager(resourceGroupName, virtualNetworkName string, options *armnetwork.SubnetsClientListOptions) clients.SubnetsPager {
	return t.pager
}

func TestNetworkSubnet(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"
	virtualNetworkName := "test-vnet"
	subnetName := "test-subnet"

	t.Run("Get", func(t *testing.T) {
		subnet := createAzureSubnet(subnetName, virtualNetworkName)

		mockClient := mocks.NewMockSubnetsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, virtualNetworkName, subnetName, nil).Return(
			armnetwork.SubnetsClientGetResponse{
				Subnet: *subnet,
			}, nil)

		testClient := &testSubnetsClient{MockSubnetsClient: mockClient}
		wrapper := manual.NewNetworkSubnet(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(virtualNetworkName, subnetName)
		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.NetworkSubnet.String() {
			t.Errorf("Expected type %s, got %s", azureshared.NetworkSubnet, sdpItem.GetType())
		}

		if sdpItem.GetUniqueAttribute() != "uniqueAttr" {
			t.Errorf("Expected unique attribute 'uniqueAttr', got %s", sdpItem.GetUniqueAttribute())
		}

		if sdpItem.UniqueAttributeValue() != shared.CompositeLookupKey(virtualNetworkName, subnetName) {
			t.Errorf("Expected unique attribute value %s, got %s", shared.CompositeLookupKey(virtualNetworkName, subnetName), sdpItem.UniqueAttributeValue())
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

	t.Run("Get_InvalidQueryParts", func(t *testing.T) {
		mockClient := mocks.NewMockSubnetsClient(ctrl)
		testClient := &testSubnetsClient{MockSubnetsClient: mockClient}

		wrapper := manual.NewNetworkSubnet(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], virtualNetworkName, true)
		if qErr == nil {
			t.Error("Expected error when providing insufficient query parts, but got nil")
		}
	})

	t.Run("Search", func(t *testing.T) {
		subnet1 := createAzureSubnet("subnet-1", virtualNetworkName)
		subnet2 := createAzureSubnet("subnet-2", virtualNetworkName)

		mockClient := mocks.NewMockSubnetsClient(ctrl)
		mockPager := &mockSubnetsPager{
			pages: []armnetwork.SubnetsClientListResponse{
				{
					SubnetListResult: armnetwork.SubnetListResult{
						Value: []*armnetwork.Subnet{subnet1, subnet2},
					},
				},
			},
		}

		testClient := &testSubnetsClient{
			MockSubnetsClient: mockClient,
			pager:             mockPager,
		}

		wrapper := manual.NewNetworkSubnet(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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
			if item.GetType() != azureshared.NetworkSubnet.String() {
				t.Errorf("Expected type %s, got %s", azureshared.NetworkSubnet, item.GetType())
			}
		}
	})

	t.Run("Search_InvalidQueryParts", func(t *testing.T) {
		mockClient := mocks.NewMockSubnetsClient(ctrl)
		testClient := &testSubnetsClient{MockSubnetsClient: mockClient}

		wrapper := manual.NewNetworkSubnet(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		_, qErr := wrapper.Search(ctx, wrapper.Scopes()[0])
		if qErr == nil {
			t.Error("Expected error when providing no query parts, but got nil")
		}
	})

	t.Run("Search_SubnetWithNilName", func(t *testing.T) {
		validSubnet := createAzureSubnet("valid-subnet", virtualNetworkName)

		mockClient := mocks.NewMockSubnetsClient(ctrl)
		mockPager := &mockSubnetsPager{
			pages: []armnetwork.SubnetsClientListResponse{
				{
					SubnetListResult: armnetwork.SubnetListResult{
						Value: []*armnetwork.Subnet{
							{Name: nil, ID: to.Ptr("/some/id")},
							validSubnet,
						},
					},
				},
			},
		}

		testClient := &testSubnetsClient{
			MockSubnetsClient: mockClient,
			pager:             mockPager,
		}

		wrapper := manual.NewNetworkSubnet(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		searchable := adapter.(discovery.SearchableAdapter)
		sdpItems, err := searchable.Search(ctx, wrapper.Scopes()[0], virtualNetworkName, true)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if len(sdpItems) != 1 {
			t.Fatalf("Expected 1 item (nil name skipped), got: %d", len(sdpItems))
		}
		if sdpItems[0].UniqueAttributeValue() != shared.CompositeLookupKey(virtualNetworkName, "valid-subnet") {
			t.Errorf("Expected unique value %s, got %s", shared.CompositeLookupKey(virtualNetworkName, "valid-subnet"), sdpItems[0].UniqueAttributeValue())
		}
	})

	t.Run("ErrorHandling_Get", func(t *testing.T) {
		expectedErr := errors.New("subnet not found")

		mockClient := mocks.NewMockSubnetsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, virtualNetworkName, "nonexistent-subnet", nil).Return(
			armnetwork.SubnetsClientGetResponse{}, expectedErr)

		testClient := &testSubnetsClient{MockSubnetsClient: mockClient}
		wrapper := manual.NewNetworkSubnet(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(virtualNetworkName, "nonexistent-subnet")
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr == nil {
			t.Error("Expected error when getting non-existent subnet, but got nil")
		}
	})

	t.Run("ErrorHandling_Search", func(t *testing.T) {
		mockClient := mocks.NewMockSubnetsClient(ctrl)
		testClient := &testSubnetsClient{
			MockSubnetsClient: mockClient,
			pager:             &errorSubnetsPager{},
		}

		wrapper := manual.NewNetworkSubnet(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		searchable := adapter.(discovery.SearchableAdapter)
		_, err := searchable.Search(ctx, wrapper.Scopes()[0], virtualNetworkName, true)
		if err == nil {
			t.Error("Expected error from pager when NextPage returns an error, but got nil")
		}
	})
}

func createAzureSubnet(subnetName, vnetName string) *armnetwork.Subnet {
	return &armnetwork.Subnet{
		ID:   to.Ptr("/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.Network/virtualNetworks/" + vnetName + "/subnets/" + subnetName),
		Name: to.Ptr(subnetName),
		Type: to.Ptr("Microsoft.Network/virtualNetworks/subnets"),
		Properties: &armnetwork.SubnetPropertiesFormat{
			AddressPrefix: to.Ptr("10.0.0.0/24"),
		},
	}
}
