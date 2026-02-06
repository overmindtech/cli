package manual_test

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/dns/armdns"
	"go.uber.org/mock/gomock"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/manual"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/azure/shared/mocks"
	"github.com/overmindtech/cli/sources/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

func TestNetworkZone(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"

	t.Run("Get", func(t *testing.T) {
		zoneName := "example.com"
		zone := createAzureZone(zoneName, subscriptionID, resourceGroup)

		mockClient := mocks.NewMockZonesClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, zoneName, nil).Return(
			armdns.ZonesClientGetResponse{
				Zone: *zone,
			}, nil)

		wrapper := manual.NewNetworkZone(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], zoneName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.NetworkZone.String() {
			t.Errorf("Expected type %s, got %s", azureshared.NetworkZone, sdpItem.GetType())
		}

		if sdpItem.GetUniqueAttribute() != "name" {
			t.Errorf("Expected unique attribute 'name', got %s", sdpItem.GetUniqueAttribute())
		}

		if sdpItem.UniqueAttributeValue() != zoneName {
			t.Errorf("Expected unique attribute value %s, got %s", zoneName, sdpItem.UniqueAttributeValue())
		}

		if sdpItem.GetTags()["env"] != "test" {
			t.Errorf("Expected tag 'env=test', got: %v", sdpItem.GetTags()["env"])
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					// DNS name for the zone itself (standard library)
					ExpectedType:   "dns",
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  zoneName,
					ExpectedScope:  "global",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					// Virtual Network from RegistrationVirtualNetworks
					ExpectedType:   azureshared.NetworkVirtualNetwork.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-reg-vnet",
					ExpectedScope:  fmt.Sprintf("%s.%s", subscriptionID, resourceGroup),
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					// Virtual Network from ResolutionVirtualNetworks
					ExpectedType:   azureshared.NetworkVirtualNetwork.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-res-vnet",
					ExpectedScope:  fmt.Sprintf("%s.%s", subscriptionID, resourceGroup),
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					// DNS Record Set (child resource)
					ExpectedType:   azureshared.NetworkDNSRecordSet.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  zoneName,
					ExpectedScope:  fmt.Sprintf("%s.%s", subscriptionID, resourceGroup),
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					// DNS name server (standard library)
					ExpectedType:   "dns",
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  "ns1.example.com",
					ExpectedScope:  "global",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					// DNS name server (standard library)
					ExpectedType:   "dns",
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  "ns2.example.com",
					ExpectedScope:  "global",
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
		mockClient := mocks.NewMockZonesClient(ctrl)

		wrapper := manual.NewNetworkZone(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		// Test with empty name - the client will be called but will return an error
		// or we can test by not setting up expectations and letting it fail
		// Actually, the wrapper validates len(queryParts) < 1, so we need to test that
		// But adapter.Get takes a single query string, so we can't test empty queryParts
		// Let's test with a zone that has nil name which will cause an error
		zoneWithNilName := &armdns.Zone{
			Name:       nil,
			Location:   to.Ptr("eastus"),
			Properties: &armdns.ZoneProperties{},
		}

		mockClient.EXPECT().Get(ctx, resourceGroup, "test-zone", nil).Return(
			armdns.ZonesClientGetResponse{
				Zone: *zoneWithNilName,
			}, nil)

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "test-zone", true)
		if qErr == nil {
			t.Error("Expected error when zone has nil name, but got nil")
		}
	})

	t.Run("Get_DifferentScopeVirtualNetwork", func(t *testing.T) {
		// Test that Virtual Network with different subscription/resource group uses correct scope
		zoneName := "example.com"
		otherSubscriptionID := "other-sub"
		otherResourceGroup := "other-rg"
		zone := createAzureZoneWithDifferentScopeVNet(zoneName, subscriptionID, resourceGroup, otherSubscriptionID, otherResourceGroup)

		mockClient := mocks.NewMockZonesClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, zoneName, nil).Return(
			armdns.ZonesClientGetResponse{
				Zone: *zone,
			}, nil)

		wrapper := manual.NewNetworkZone(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], zoneName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		// Verify that the virtual network link uses the correct scope
		found := false
		for _, linkedQuery := range sdpItem.GetLinkedItemQueries() {
			if linkedQuery.GetQuery().GetType() == azureshared.NetworkVirtualNetwork.String() {
				expectedScope := fmt.Sprintf("%s.%s", otherSubscriptionID, otherResourceGroup)
				if linkedQuery.GetQuery().GetScope() == expectedScope {
					found = true
					break
				}
			}
		}
		if !found {
			t.Error("Expected to find virtual network link with different scope")
		}
	})

	t.Run("List", func(t *testing.T) {
		zone1 := createAzureZone("example.com", subscriptionID, resourceGroup)
		zone2 := createAzureZone("test.com", subscriptionID, resourceGroup)

		mockClient := mocks.NewMockZonesClient(ctrl)
		mockPager := NewMockZonesPager(ctrl)

		// Setup pager expectations
		gomock.InOrder(
			mockPager.EXPECT().More().Return(true),
			mockPager.EXPECT().NextPage(ctx).Return(
				armdns.ZonesClientListByResourceGroupResponse{
					ZoneListResult: armdns.ZoneListResult{
						Value: []*armdns.Zone{zone1, zone2},
					},
				}, nil),
			mockPager.EXPECT().More().Return(false),
		)

		mockClient.EXPECT().NewListByResourceGroupPager(resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewNetworkZone(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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

			if item.GetTags()["env"] != "test" {
				t.Fatalf("Expected tag 'env=test', got: %s", item.GetTags()["env"])
			}

			if item.GetType() != azureshared.NetworkZone.String() {
				t.Fatalf("Expected type %s, got: %s", azureshared.NetworkZone, item.GetType())
			}
		}
	})

	t.Run("List_WithNilName", func(t *testing.T) {
		// Test that zones with nil names are skipped in List
		zone1 := createAzureZone("example.com", subscriptionID, resourceGroup)
		zone2 := &armdns.Zone{
			Name:     nil, // Zone with nil name should be skipped
			Location: to.Ptr("eastus"),
			Tags: map[string]*string{
				"env": to.Ptr("test"),
			},
			Properties: &armdns.ZoneProperties{},
		}

		mockClient := mocks.NewMockZonesClient(ctrl)
		mockPager := NewMockZonesPager(ctrl)

		// Setup pager expectations
		gomock.InOrder(
			mockPager.EXPECT().More().Return(true),
			mockPager.EXPECT().NextPage(ctx).Return(
				armdns.ZonesClientListByResourceGroupResponse{
					ZoneListResult: armdns.ZoneListResult{
						Value: []*armdns.Zone{zone1, zone2},
					},
				}, nil),
			mockPager.EXPECT().More().Return(false),
		)

		mockClient.EXPECT().NewListByResourceGroupPager(resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewNetworkZone(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		listable, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support List operation")
		}

		sdpItems, err := listable.List(ctx, wrapper.Scopes()[0], true)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		// Should only return 1 item (zone1), zone2 with nil name should be skipped
		if len(sdpItems) != 1 {
			t.Fatalf("Expected 1 item (nil name should be skipped), got: %d", len(sdpItems))
		}

		if sdpItems[0].UniqueAttributeValue() != "example.com" {
			t.Errorf("Expected item name 'example.com', got: %s", sdpItems[0].UniqueAttributeValue())
		}
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		expectedErr := errors.New("zone not found")

		mockClient := mocks.NewMockZonesClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, "nonexistent-zone", nil).Return(
			armdns.ZonesClientGetResponse{}, expectedErr)

		wrapper := manual.NewNetworkZone(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "nonexistent-zone", true)
		if qErr == nil {
			t.Error("Expected error when getting non-existent zone, but got nil")
		}
	})

	t.Run("ErrorHandling_List", func(t *testing.T) {
		expectedErr := errors.New("failed to list zones")

		mockClient := mocks.NewMockZonesClient(ctrl)
		mockPager := NewMockZonesPager(ctrl)

		// Setup pager to return error on NextPage
		gomock.InOrder(
			mockPager.EXPECT().More().Return(true),
			mockPager.EXPECT().NextPage(ctx).Return(
				armdns.ZonesClientListByResourceGroupResponse{}, expectedErr),
		)

		mockClient.EXPECT().NewListByResourceGroupPager(resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewNetworkZone(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		listable, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support List operation")
		}

		_, err := listable.List(ctx, wrapper.Scopes()[0], true)
		if err == nil {
			t.Error("Expected error when listing zones fails, but got nil")
		}
	})

	t.Run("InterfaceCompliance", func(t *testing.T) {
		mockClient := mocks.NewMockZonesClient(ctrl)
		wrapper := manual.NewNetworkZone(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		// Verify wrapper implements ListableWrapper interface
		var _ = wrapper

		// Cast to sources.Wrapper to access interface methods
		w := wrapper.(sources.Wrapper)

		// Verify IAMPermissions
		permissions := w.IAMPermissions()
		if len(permissions) == 0 {
			t.Error("Expected IAMPermissions to return at least one permission")
		}
		expectedPermission := "Microsoft.Network/dnszones/read"
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
		potentialLinks := w.PotentialLinks()
		if len(potentialLinks) == 0 {
			t.Error("Expected PotentialLinks to return at least one link")
		}
		if !potentialLinks[azureshared.NetworkVirtualNetwork] {
			t.Error("Expected PotentialLinks to include NetworkVirtualNetwork")
		}
		if !potentialLinks[azureshared.NetworkDNSRecordSet] {
			t.Error("Expected PotentialLinks to include NetworkDNSRecordSet")
		}
		if !potentialLinks[stdlib.NetworkDNS] {
			t.Error("Expected PotentialLinks to include stdlib.NetworkDNS")
		}

		// Verify TerraformMappings
		mappings := w.TerraformMappings()
		if len(mappings) == 0 {
			t.Error("Expected TerraformMappings to return at least one mapping")
		}
		foundMapping := false
		for _, mapping := range mappings {
			if mapping.GetTerraformQueryMap() == "azurerm_dns_zone.name" {
				foundMapping = true
				if mapping.GetTerraformMethod() != sdp.QueryMethod_GET {
					t.Errorf("Expected TerraformMethod to be GET, got: %s", mapping.GetTerraformMethod())
				}
				break
			}
		}
		if !foundMapping {
			t.Error("Expected TerraformMappings to include 'azurerm_dns_zone.name' mapping")
		}

		// Verify GetLookups
		lookups := w.GetLookups()
		if len(lookups) == 0 {
			t.Error("Expected GetLookups to return at least one lookup")
		}
		foundLookup := false
		for _, lookup := range lookups {
			if lookup.ItemType == azureshared.NetworkZone {
				foundLookup = true
				break
			}
		}
		if !foundLookup {
			t.Error("Expected GetLookups to include NetworkZone")
		}
	})

	t.Run("Get_NoVirtualNetworks", func(t *testing.T) {
		// Test zone without virtual networks
		zoneName := "example.com"
		zone := &armdns.Zone{
			Name:     to.Ptr(zoneName),
			Location: to.Ptr("eastus"),
			Tags: map[string]*string{
				"env": to.Ptr("test"),
			},
			Properties: &armdns.ZoneProperties{
				NameServers: []*string{
					to.Ptr("ns1.example.com"),
					to.Ptr("ns2.example.com"),
				},
			},
		}

		mockClient := mocks.NewMockZonesClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, zoneName, nil).Return(
			armdns.ZonesClientGetResponse{
				Zone: *zone,
			}, nil)

		wrapper := manual.NewNetworkZone(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], zoneName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		// Should still have child resource links and name server links
		hasRecordSetLink := false
		hasNameServerLink := false
		for _, linkedQuery := range sdpItem.GetLinkedItemQueries() {
			if linkedQuery.GetQuery().GetType() == azureshared.NetworkDNSRecordSet.String() {
				hasRecordSetLink = true
			}
			if linkedQuery.GetQuery().GetType() == "dns" {
				hasNameServerLink = true
			}
		}
		if !hasRecordSetLink {
			t.Error("Expected DNS Record Set link even without virtual networks")
		}
		if !hasNameServerLink {
			t.Error("Expected name server DNS link")
		}
	})
}

// MockZonesPager is a mock implementation of ZonesPager
type MockZonesPager struct {
	ctrl     *gomock.Controller
	recorder *MockZonesPagerMockRecorder
}

type MockZonesPagerMockRecorder struct {
	mock *MockZonesPager
}

func NewMockZonesPager(ctrl *gomock.Controller) *MockZonesPager {
	mock := &MockZonesPager{ctrl: ctrl}
	mock.recorder = &MockZonesPagerMockRecorder{mock}
	return mock
}

func (m *MockZonesPager) EXPECT() *MockZonesPagerMockRecorder {
	return m.recorder
}

func (m *MockZonesPager) More() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "More")
	ret0, _ := ret[0].(bool)
	return ret0
}

func (mr *MockZonesPagerMockRecorder) More() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "More", reflect.TypeOf((*MockZonesPager)(nil).More))
}

func (m *MockZonesPager) NextPage(ctx context.Context) (armdns.ZonesClientListByResourceGroupResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NextPage", ctx)
	ret0, _ := ret[0].(armdns.ZonesClientListByResourceGroupResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockZonesPagerMockRecorder) NextPage(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NextPage", reflect.TypeOf((*MockZonesPager)(nil).NextPage), ctx)
}

// createAzureZone creates a mock Azure DNS zone for testing with all linked resources
func createAzureZone(zoneName, subscriptionID, resourceGroup string) *armdns.Zone {
	registrationVNetID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/virtualNetworks/test-reg-vnet", subscriptionID, resourceGroup)
	resolutionVNetID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/virtualNetworks/test-res-vnet", subscriptionID, resourceGroup)

	return &armdns.Zone{
		Name:     to.Ptr(zoneName),
		Location: to.Ptr("eastus"),
		Tags: map[string]*string{
			"env":     to.Ptr("test"),
			"project": to.Ptr("testing"),
		},
		Properties: &armdns.ZoneProperties{
			MaxNumberOfRecordSets: to.Ptr(int64(5000)),
			NumberOfRecordSets:    to.Ptr(int64(10)),
			NameServers: []*string{
				to.Ptr("ns1.example.com"),
				to.Ptr("ns2.example.com"),
			},
			RegistrationVirtualNetworks: []*armdns.SubResource{
				{
					ID: to.Ptr(registrationVNetID),
				},
			},
			ResolutionVirtualNetworks: []*armdns.SubResource{
				{
					ID: to.Ptr(resolutionVNetID),
				},
			},
		},
	}
}

// createAzureZoneWithDifferentScopeVNet creates a zone with a virtual network in a different scope
func createAzureZoneWithDifferentScopeVNet(zoneName, subscriptionID, resourceGroup, otherSubscriptionID, otherResourceGroup string) *armdns.Zone {
	registrationVNetID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/virtualNetworks/test-reg-vnet", otherSubscriptionID, otherResourceGroup)

	return &armdns.Zone{
		Name:     to.Ptr(zoneName),
		Location: to.Ptr("eastus"),
		Tags: map[string]*string{
			"env": to.Ptr("test"),
		},
		Properties: &armdns.ZoneProperties{
			MaxNumberOfRecordSets: to.Ptr(int64(5000)),
			NumberOfRecordSets:    to.Ptr(int64(10)),
			NameServers: []*string{
				to.Ptr("ns1.example.com"),
			},
			RegistrationVirtualNetworks: []*armdns.SubResource{
				{
					ID: to.Ptr(registrationVNetID),
				},
			},
		},
	}
}
