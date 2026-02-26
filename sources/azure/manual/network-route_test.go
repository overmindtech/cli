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
	"github.com/overmindtech/cli/sources/stdlib"
)

type mockRoutesPager struct {
	pages []armnetwork.RoutesClientListResponse
	index int
}

func (m *mockRoutesPager) More() bool {
	return m.index < len(m.pages)
}

func (m *mockRoutesPager) NextPage(ctx context.Context) (armnetwork.RoutesClientListResponse, error) {
	if m.index >= len(m.pages) {
		return armnetwork.RoutesClientListResponse{}, errors.New("no more pages")
	}
	page := m.pages[m.index]
	m.index++
	return page, nil
}

type errorRoutesPager struct{}

func (e *errorRoutesPager) More() bool {
	return true
}

func (e *errorRoutesPager) NextPage(ctx context.Context) (armnetwork.RoutesClientListResponse, error) {
	return armnetwork.RoutesClientListResponse{}, errors.New("pager error")
}

type testRoutesClient struct {
	*mocks.MockRoutesClient
	pager clients.RoutesPager
}

func (t *testRoutesClient) NewListPager(resourceGroupName, routeTableName string, options *armnetwork.RoutesClientListOptions) clients.RoutesPager {
	return t.pager
}

func TestNetworkRoute(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"
	routeTableName := "test-route-table"
	routeName := "test-route"

	t.Run("Get", func(t *testing.T) {
		route := createAzureRoute(routeName, routeTableName)

		mockClient := mocks.NewMockRoutesClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, routeTableName, routeName, nil).Return(
			armnetwork.RoutesClientGetResponse{
				Route: *route,
			}, nil)

		testClient := &testRoutesClient{MockRoutesClient: mockClient}
		wrapper := manual.NewNetworkRoute(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(routeTableName, routeName)
		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.NetworkRoute.String() {
			t.Errorf("Expected type %s, got %s", azureshared.NetworkRoute, sdpItem.GetType())
		}

		if sdpItem.GetUniqueAttribute() != "uniqueAttr" {
			t.Errorf("Expected unique attribute 'uniqueAttr', got %s", sdpItem.GetUniqueAttribute())
		}

		if sdpItem.UniqueAttributeValue() != shared.CompositeLookupKey(routeTableName, routeName) {
			t.Errorf("Expected unique attribute value %s, got %s", shared.CompositeLookupKey(routeTableName, routeName), sdpItem.UniqueAttributeValue())
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
					ExpectedType:   azureshared.NetworkRouteTable.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  routeTableName,
					ExpectedScope:  subscriptionID + "." + resourceGroup,
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

	t.Run("Get_EmptyRouteName", func(t *testing.T) {
		mockClient := mocks.NewMockRoutesClient(ctrl)
		testClient := &testRoutesClient{MockRoutesClient: mockClient}

		wrapper := manual.NewNetworkRoute(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(routeTableName, "")
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr == nil {
			t.Error("Expected error when route name is empty, but got nil")
		}
	})

	t.Run("Get_InvalidQueryParts", func(t *testing.T) {
		mockClient := mocks.NewMockRoutesClient(ctrl)
		testClient := &testRoutesClient{MockRoutesClient: mockClient}

		wrapper := manual.NewNetworkRoute(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], routeTableName, true)
		if qErr == nil {
			t.Error("Expected error when providing insufficient query parts, but got nil")
		}
	})

	t.Run("Search", func(t *testing.T) {
		route1 := createAzureRoute("route-1", routeTableName)
		route2 := createAzureRoute("route-2", routeTableName)

		mockClient := mocks.NewMockRoutesClient(ctrl)
		mockPager := &mockRoutesPager{
			pages: []armnetwork.RoutesClientListResponse{
				{
					RouteListResult: armnetwork.RouteListResult{
						Value: []*armnetwork.Route{route1, route2},
					},
				},
			},
		}

		testClient := &testRoutesClient{
			MockRoutesClient: mockClient,
			pager:            mockPager,
		}

		wrapper := manual.NewNetworkRoute(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support Search operation")
		}

		sdpItems, err := searchable.Search(ctx, wrapper.Scopes()[0], routeTableName, true)
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
			if item.GetType() != azureshared.NetworkRoute.String() {
				t.Errorf("Expected type %s, got %s", azureshared.NetworkRoute, item.GetType())
			}
		}
	})

	t.Run("Search_InvalidQueryParts", func(t *testing.T) {
		mockClient := mocks.NewMockRoutesClient(ctrl)
		testClient := &testRoutesClient{MockRoutesClient: mockClient}

		wrapper := manual.NewNetworkRoute(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		_, qErr := wrapper.Search(ctx, wrapper.Scopes()[0])
		if qErr == nil {
			t.Error("Expected error when providing no query parts, but got nil")
		}
	})

	t.Run("Search_RouteWithNilName", func(t *testing.T) {
		validRoute := createAzureRoute("valid-route", routeTableName)

		mockClient := mocks.NewMockRoutesClient(ctrl)
		mockPager := &mockRoutesPager{
			pages: []armnetwork.RoutesClientListResponse{
				{
					RouteListResult: armnetwork.RouteListResult{
						Value: []*armnetwork.Route{
							{Name: nil, ID: strPtr("/some/id")},
							validRoute,
						},
					},
				},
			},
		}

		testClient := &testRoutesClient{
			MockRoutesClient: mockClient,
			pager:            mockPager,
		}

		wrapper := manual.NewNetworkRoute(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		searchable := adapter.(discovery.SearchableAdapter)
		sdpItems, err := searchable.Search(ctx, wrapper.Scopes()[0], routeTableName, true)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if len(sdpItems) != 1 {
			t.Fatalf("Expected 1 item (nil name skipped), got: %d", len(sdpItems))
		}
		if sdpItems[0].UniqueAttributeValue() != shared.CompositeLookupKey(routeTableName, "valid-route") {
			t.Errorf("Expected unique value %s, got %s", shared.CompositeLookupKey(routeTableName, "valid-route"), sdpItems[0].UniqueAttributeValue())
		}
	})

	t.Run("ErrorHandling_Get", func(t *testing.T) {
		expectedErr := errors.New("route not found")

		mockClient := mocks.NewMockRoutesClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, routeTableName, "nonexistent-route", nil).Return(
			armnetwork.RoutesClientGetResponse{}, expectedErr)

		testClient := &testRoutesClient{MockRoutesClient: mockClient}
		wrapper := manual.NewNetworkRoute(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(routeTableName, "nonexistent-route")
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr == nil {
			t.Error("Expected error when getting non-existent route, but got nil")
		}
	})

	t.Run("ErrorHandling_Search", func(t *testing.T) {
		mockClient := mocks.NewMockRoutesClient(ctrl)
		testClient := &testRoutesClient{
			MockRoutesClient: mockClient,
			pager:            &errorRoutesPager{},
		}

		wrapper := manual.NewNetworkRoute(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		searchable := adapter.(discovery.SearchableAdapter)
		_, err := searchable.Search(ctx, wrapper.Scopes()[0], routeTableName, true)
		if err == nil {
			t.Error("Expected error from pager when NextPage returns an error, but got nil")
		}
	})
}

func createAzureRoute(routeName, routeTableName string) *armnetwork.Route {
	idStr := "/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.Network/routeTables/" + routeTableName + "/routes/" + routeName
	typeStr := "Microsoft.Network/routeTables/routes"
	provisioningState := armnetwork.ProvisioningStateSucceeded
	nextHopIP := "10.0.0.1"
	nextHopType := armnetwork.RouteNextHopTypeVnetLocal
	return &armnetwork.Route{
		ID:   &idStr,
		Name: &routeName,
		Type: &typeStr,
		Properties: &armnetwork.RoutePropertiesFormat{
			ProvisioningState: &provisioningState,
			NextHopIPAddress:  &nextHopIP,
			AddressPrefix:     strPtr("10.0.0.0/24"),
			NextHopType:       &nextHopType,
		},
	}
}
