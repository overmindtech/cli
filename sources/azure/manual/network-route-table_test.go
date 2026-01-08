package manual_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"
	"go.uber.org/mock/gomock"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/manual"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/azure/shared/mocks"
	"github.com/overmindtech/cli/sources/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

func TestNetworkRouteTable(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"

	t.Run("Get", func(t *testing.T) {
		routeTableName := "test-route-table"
		routeTable := createAzureRouteTable(routeTableName)

		mockClient := mocks.NewMockRouteTablesClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, routeTableName, nil).Return(
			armnetwork.RouteTablesClientGetResponse{
				RouteTable: *routeTable,
			}, nil)

		wrapper := manual.NewNetworkRouteTable(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper)

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], routeTableName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.NetworkRouteTable.String() {
			t.Errorf("Expected type %s, got %s", azureshared.NetworkRouteTable, sdpItem.GetType())
		}

		if sdpItem.GetUniqueAttribute() != "name" {
			t.Errorf("Expected unique attribute 'name', got %s", sdpItem.GetUniqueAttribute())
		}

		if sdpItem.UniqueAttributeValue() != routeTableName {
			t.Errorf("Expected unique attribute value %s, got %s", routeTableName, sdpItem.UniqueAttributeValue())
		}

		if sdpItem.GetTags()["env"] != "test" {
			t.Errorf("Expected tag 'env=test', got: %v", sdpItem.GetTags()["env"])
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					// Route link (child resource)
					ExpectedType:   azureshared.NetworkRoute.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey(routeTableName, "test-route"),
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					// Route with NextHopIPAddress link (IP address to stdlib)
					ExpectedType:   stdlib.NetworkIP.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "10.0.0.1",
					ExpectedScope:  "global",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					// Subnet link (external resource)
					ExpectedType:   azureshared.NetworkSubnet.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey("test-vnet", "test-subnet"),
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
			}

			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("Get_InvalidQueryParts", func(t *testing.T) {
		mockClient := mocks.NewMockRouteTablesClient(ctrl)

		wrapper := manual.NewNetworkRouteTable(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper)

		// Test with empty string name - validation happens before client.Get is called
		// so no mock expectation is needed
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "", true)
		if qErr == nil {
			t.Error("Expected error when getting route table with empty name, but got nil")
		}
	})

	t.Run("Get_WithNilName", func(t *testing.T) {
		routeTable := &armnetwork.RouteTable{
			Name:     nil, // Route table with nil name should cause an error
			Location: to.Ptr("eastus"),
			Tags: map[string]*string{
				"env": to.Ptr("test"),
			},
		}

		mockClient := mocks.NewMockRouteTablesClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, "test-route-table", nil).Return(
			armnetwork.RouteTablesClientGetResponse{
				RouteTable: *routeTable,
			}, nil)

		wrapper := manual.NewNetworkRouteTable(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper)

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "test-route-table", true)
		if qErr == nil {
			t.Error("Expected error when route table has nil name, but got nil")
		}
	})

	t.Run("List", func(t *testing.T) {
		routeTable1 := createAzureRouteTable("test-route-table-1")
		routeTable2 := createAzureRouteTable("test-route-table-2")

		mockClient := mocks.NewMockRouteTablesClient(ctrl)
		mockPager := NewMockRouteTablesPager(ctrl)

		// Setup pager expectations
		gomock.InOrder(
			mockPager.EXPECT().More().Return(true),
			mockPager.EXPECT().NextPage(ctx).Return(
				armnetwork.RouteTablesClientListResponse{
					RouteTableListResult: armnetwork.RouteTableListResult{
						Value: []*armnetwork.RouteTable{routeTable1, routeTable2},
					},
				}, nil),
			mockPager.EXPECT().More().Return(false),
		)

		mockClient.EXPECT().List(resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewNetworkRouteTable(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper)

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

			if item.GetTags()["env"] != "test" {
				t.Fatalf("Expected tag 'env=test', got: %s", item.GetTags()["env"])
			}

			if item.GetType() != azureshared.NetworkRouteTable.String() {
				t.Fatalf("Expected type %s, got: %s", azureshared.NetworkRouteTable, item.GetType())
			}
		}
	})

	t.Run("List_WithNilName", func(t *testing.T) {
		// Create route table with nil name to test error handling
		routeTable1 := createAzureRouteTable("test-route-table-1")
		routeTable2 := &armnetwork.RouteTable{
			Name:     nil, // Route table with nil name should be skipped
			Location: to.Ptr("eastus"),
			Tags: map[string]*string{
				"env": to.Ptr("test"),
			},
		}

		mockClient := mocks.NewMockRouteTablesClient(ctrl)
		mockPager := NewMockRouteTablesPager(ctrl)

		// Setup pager expectations
		gomock.InOrder(
			mockPager.EXPECT().More().Return(true),
			mockPager.EXPECT().NextPage(ctx).Return(
				armnetwork.RouteTablesClientListResponse{
					RouteTableListResult: armnetwork.RouteTableListResult{
						Value: []*armnetwork.RouteTable{routeTable1, routeTable2},
					},
				}, nil),
			mockPager.EXPECT().More().Return(false),
		)

		mockClient.EXPECT().List(resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewNetworkRouteTable(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper)

		listable, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support List operation")
		}

		sdpItems, err := listable.List(ctx, wrapper.Scopes()[0], true)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		// Should only return 1 item (routeTable1), routeTable2 should be skipped
		if len(sdpItems) != 1 {
			t.Fatalf("Expected 1 item (nil name should be skipped), got: %d", len(sdpItems))
		}
	})

	t.Run("List_ErrorHandling", func(t *testing.T) {
		expectedErr := errors.New("failed to list route tables")

		mockClient := mocks.NewMockRouteTablesClient(ctrl)
		mockPager := NewMockRouteTablesPager(ctrl)

		// Setup pager to return error on NextPage
		gomock.InOrder(
			mockPager.EXPECT().More().Return(true),
			mockPager.EXPECT().NextPage(ctx).Return(
				armnetwork.RouteTablesClientListResponse{}, expectedErr),
		)

		mockClient.EXPECT().List(resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewNetworkRouteTable(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper)

		listable, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support List operation")
		}

		_, err := listable.List(ctx, wrapper.Scopes()[0], true)
		if err == nil {
			t.Error("Expected error when listing route tables fails, but got nil")
		}
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		expectedErr := errors.New("route table not found")

		mockClient := mocks.NewMockRouteTablesClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, "nonexistent-route-table", nil).Return(
			armnetwork.RouteTablesClientGetResponse{}, expectedErr)

		wrapper := manual.NewNetworkRouteTable(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper)

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "nonexistent-route-table", true)
		if qErr == nil {
			t.Error("Expected error when getting non-existent route table, but got nil")
		}
	})

	t.Run("CrossResourceGroupLinks", func(t *testing.T) {
		// Test route table with subnet in different resource group
		routeTableName := "test-route-table"
		otherResourceGroup := "other-rg"
		otherSubscriptionID := "other-subscription"

		routeTable := &armnetwork.RouteTable{
			Name:     to.Ptr(routeTableName),
			Location: to.Ptr("eastus"),
			Tags: map[string]*string{
				"env": to.Ptr("test"),
			},
			Properties: &armnetwork.RouteTablePropertiesFormat{
				Subnets: []*armnetwork.Subnet{
					{
						ID: to.Ptr("/subscriptions/" + otherSubscriptionID + "/resourceGroups/" + otherResourceGroup + "/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/test-subnet"),
					},
				},
			},
		}

		mockClient := mocks.NewMockRouteTablesClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, routeTableName, nil).Return(
			armnetwork.RouteTablesClientGetResponse{
				RouteTable: *routeTable,
			}, nil)

		wrapper := manual.NewNetworkRouteTable(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper)

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], routeTableName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		// Check that subnet link uses the correct scope
		foundSubnetLink := false
		for _, link := range sdpItem.GetLinkedItemQueries() {
			if link.GetQuery().GetType() == azureshared.NetworkSubnet.String() {
				foundSubnetLink = true
				expectedScope := otherSubscriptionID + "." + otherResourceGroup
				if link.GetQuery().GetScope() != expectedScope {
					t.Errorf("Expected subnet scope %s, got %s", expectedScope, link.GetQuery().GetScope())
				}
			}
		}
		if !foundSubnetLink {
			t.Error("Expected to find subnet link")
		}
	})

	t.Run("RouteWithNextHopIPAddress", func(t *testing.T) {
		// Test route table with route that has NextHopIPAddress
		routeTableName := "test-route-table"
		routeTable := &armnetwork.RouteTable{
			Name:     to.Ptr(routeTableName),
			Location: to.Ptr("eastus"),
			Tags: map[string]*string{
				"env": to.Ptr("test"),
			},
			Properties: &armnetwork.RouteTablePropertiesFormat{
				Routes: []*armnetwork.Route{
					{
						Name: to.Ptr("test-route"),
						Properties: &armnetwork.RoutePropertiesFormat{
							AddressPrefix:    to.Ptr("10.0.0.0/16"),
							NextHopType:      to.Ptr(armnetwork.RouteNextHopTypeVirtualAppliance),
							NextHopIPAddress: to.Ptr("10.0.0.1"),
						},
					},
				},
			},
		}

		mockClient := mocks.NewMockRouteTablesClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, routeTableName, nil).Return(
			armnetwork.RouteTablesClientGetResponse{
				RouteTable: *routeTable,
			}, nil)

		wrapper := manual.NewNetworkRouteTable(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper)

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], routeTableName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		// Check that IP address link exists
		foundIPLink := false
		for _, link := range sdpItem.GetLinkedItemQueries() {
			if link.GetQuery().GetType() == stdlib.NetworkIP.String() {
				foundIPLink = true
				if link.GetQuery().GetQuery() != "10.0.0.1" {
					t.Errorf("Expected IP address '10.0.0.1', got %s", link.GetQuery().GetQuery())
				}
				if link.GetQuery().GetScope() != "global" {
					t.Errorf("Expected IP scope 'global', got %s", link.GetQuery().GetScope())
				}
			}
		}
		if !foundIPLink {
			t.Error("Expected to find IP address link")
		}
	})

	t.Run("RouteWithoutNextHopIPAddress", func(t *testing.T) {
		// Test route table with route that doesn't have NextHopIPAddress
		routeTableName := "test-route-table"
		routeTable := &armnetwork.RouteTable{
			Name:     to.Ptr(routeTableName),
			Location: to.Ptr("eastus"),
			Tags: map[string]*string{
				"env": to.Ptr("test"),
			},
			Properties: &armnetwork.RouteTablePropertiesFormat{
				Routes: []*armnetwork.Route{
					{
						Name: to.Ptr("test-route"),
						Properties: &armnetwork.RoutePropertiesFormat{
							AddressPrefix: to.Ptr("10.0.0.0/16"),
							NextHopType:   to.Ptr(armnetwork.RouteNextHopTypeInternet),
							// No NextHopIPAddress
						},
					},
				},
			},
		}

		mockClient := mocks.NewMockRouteTablesClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, routeTableName, nil).Return(
			armnetwork.RouteTablesClientGetResponse{
				RouteTable: *routeTable,
			}, nil)

		wrapper := manual.NewNetworkRouteTable(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper)

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], routeTableName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		// Check that no IP address link exists
		for _, link := range sdpItem.GetLinkedItemQueries() {
			if link.GetQuery().GetType() == stdlib.NetworkIP.String() {
				t.Error("Expected no IP address link when NextHopIPAddress is not set")
			}
		}
	})

	t.Run("InterfaceCompliance", func(t *testing.T) {
		mockClient := mocks.NewMockRouteTablesClient(ctrl)
		wrapper := manual.NewNetworkRouteTable(mockClient, subscriptionID, resourceGroup)

		// Verify wrapper implements ListableWrapper interface
		var _ sources.ListableWrapper = wrapper

		// Verify IAMPermissions
		permissions := wrapper.IAMPermissions()
		if len(permissions) == 0 {
			t.Error("Expected IAMPermissions to return at least one permission")
		}
		expectedPermission := "Microsoft.Network/routeTables/read"
		found := false
		for _, perm := range permissions {
			if perm == expectedPermission {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected IAMPermissions to include %s", expectedPermission)
		}

		// Verify PotentialLinks
		potentialLinks := wrapper.PotentialLinks()
		if len(potentialLinks) == 0 {
			t.Error("Expected PotentialLinks to return at least one link")
		}
		expectedLinks := []shared.ItemType{
			azureshared.NetworkRoute,
			azureshared.NetworkSubnet,
			azureshared.NetworkVirtualNetworkGateway,
			stdlib.NetworkIP,
		}
		for _, expectedLink := range expectedLinks {
			if !potentialLinks[expectedLink] {
				t.Errorf("Expected PotentialLinks to include %s", expectedLink)
			}
		}

		// Verify TerraformMappings
		mappings := wrapper.TerraformMappings()
		if len(mappings) == 0 {
			t.Error("Expected TerraformMappings to return at least one mapping")
		}
		foundMapping := false
		for _, mapping := range mappings {
			if mapping.GetTerraformQueryMap() == "azurerm_route_table.name" {
				foundMapping = true
				break
			}
		}
		if !foundMapping {
			t.Error("Expected TerraformMappings to include 'azurerm_route_table.name' mapping")
		}

		// Verify PredefinedRole
		// PredefinedRole is available on the wrapper, not the adapter
		// Use type assertion with interface{} to access the method
		if roleInterface, ok := interface{}(wrapper).(interface{ PredefinedRole() string }); ok {
			role := roleInterface.PredefinedRole()
			if role != "Reader" {
				t.Errorf("Expected PredefinedRole to be 'Reader', got %s", role)
			}
		} else {
			t.Error("Wrapper does not implement PredefinedRole method")
		}
	})
}

// MockRouteTablesPager is a simple mock for RouteTablesPager
type MockRouteTablesPager struct {
	ctrl     *gomock.Controller
	recorder *MockRouteTablesPagerMockRecorder
}

type MockRouteTablesPagerMockRecorder struct {
	mock *MockRouteTablesPager
}

func NewMockRouteTablesPager(ctrl *gomock.Controller) *MockRouteTablesPager {
	mock := &MockRouteTablesPager{ctrl: ctrl}
	mock.recorder = &MockRouteTablesPagerMockRecorder{mock}
	return mock
}

func (m *MockRouteTablesPager) EXPECT() *MockRouteTablesPagerMockRecorder {
	return m.recorder
}

func (m *MockRouteTablesPager) More() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "More")
	ret0, _ := ret[0].(bool)
	return ret0
}

func (mr *MockRouteTablesPagerMockRecorder) More() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "More", reflect.TypeOf((*MockRouteTablesPager)(nil).More))
}

func (m *MockRouteTablesPager) NextPage(ctx context.Context) (armnetwork.RouteTablesClientListResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NextPage", ctx)
	ret0, _ := ret[0].(armnetwork.RouteTablesClientListResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockRouteTablesPagerMockRecorder) NextPage(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NextPage", reflect.TypeOf((*MockRouteTablesPager)(nil).NextPage), ctx)
}

// createAzureRouteTable creates a mock Azure route table for testing
func createAzureRouteTable(routeTableName string) *armnetwork.RouteTable {
	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"

	return &armnetwork.RouteTable{
		Name:     to.Ptr(routeTableName),
		Location: to.Ptr("eastus"),
		Tags: map[string]*string{
			"env":     to.Ptr("test"),
			"project": to.Ptr("testing"),
		},
		Properties: &armnetwork.RouteTablePropertiesFormat{
			// Routes (child resources)
			Routes: []*armnetwork.Route{
				{
					Name: to.Ptr("test-route"),
					Properties: &armnetwork.RoutePropertiesFormat{
						AddressPrefix:    to.Ptr("10.0.0.0/16"),
						NextHopType:      to.Ptr(armnetwork.RouteNextHopTypeVirtualAppliance),
						NextHopIPAddress: to.Ptr("10.0.0.1"),
					},
				},
			},
			// Subnets (external resources)
			Subnets: []*armnetwork.Subnet{
				{
					ID: to.Ptr("/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/test-subnet"),
				},
			},
		},
	}
}
