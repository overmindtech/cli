package manual_test

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/privatedns/armprivatedns"
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

func TestNetworkPrivateDNSZone(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"

	t.Run("Get", func(t *testing.T) {
		zoneName := "private.example.com"
		zone := createAzurePrivateZone(zoneName)

		mockClient := mocks.NewMockPrivateDNSZonesClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, zoneName, nil).Return(
			armprivatedns.PrivateZonesClientGetResponse{
				PrivateZone: *zone,
			}, nil)

		wrapper := manual.NewNetworkPrivateDNSZone(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], zoneName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.NetworkPrivateDNSZone.String() {
			t.Errorf("Expected type %s, got %s", azureshared.NetworkPrivateDNSZone, sdpItem.GetType())
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
			scope := subscriptionID + "." + resourceGroup
			queryTests := shared.QueryTests{
				{
					ExpectedType:   stdlib.NetworkDNS.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  zoneName,
					ExpectedScope:  "global",
				},
				{
					ExpectedType:   azureshared.NetworkDNSVirtualNetworkLink.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  zoneName,
					ExpectedScope:  scope,
				},
				{
					ExpectedType:   azureshared.NetworkDNSRecordSet.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  zoneName,
					ExpectedScope:  scope,
				},
			}
			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("GetWithEmptyName", func(t *testing.T) {
		mockClient := mocks.NewMockPrivateDNSZonesClient(ctrl)

		wrapper := manual.NewNetworkPrivateDNSZone(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "", true)
		if qErr == nil {
			t.Error("Expected error when zone name is empty, but got nil")
		}
	})

	t.Run("Get_ZoneWithNilName", func(t *testing.T) {
		provisioningState := armprivatedns.ProvisioningStateSucceeded
		zoneWithNilName := &armprivatedns.PrivateZone{
			Name:     nil,
			Location: new("eastus"),
			Properties: &armprivatedns.PrivateZoneProperties{
				ProvisioningState: &provisioningState,
			},
		}

		mockClient := mocks.NewMockPrivateDNSZonesClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, "test-zone", nil).Return(
			armprivatedns.PrivateZonesClientGetResponse{
				PrivateZone: *zoneWithNilName,
			}, nil)

		wrapper := manual.NewNetworkPrivateDNSZone(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "test-zone", true)
		if qErr == nil {
			t.Error("Expected error when zone has nil name, but got nil")
		}
	})

	t.Run("List", func(t *testing.T) {
		zone1 := createAzurePrivateZone("private1.example.com")
		zone2 := createAzurePrivateZone("private2.example.com")

		mockClient := mocks.NewMockPrivateDNSZonesClient(ctrl)
		mockPager := newMockPrivateDNSZonesPager(ctrl, []*armprivatedns.PrivateZone{zone1, zone2})

		mockClient.EXPECT().NewListByResourceGroupPager(resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewNetworkPrivateDNSZone(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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
			if item.GetType() != azureshared.NetworkPrivateDNSZone.String() {
				t.Fatalf("Expected type %s, got: %s", azureshared.NetworkPrivateDNSZone, item.GetType())
			}
		}
	})

	t.Run("List_WithNilName", func(t *testing.T) {
		zone1 := createAzurePrivateZone("private1.example.com")
		provisioningState := armprivatedns.ProvisioningStateSucceeded
		zone2NilName := &armprivatedns.PrivateZone{
			Name:     nil,
			Location: new("eastus"),
			Tags:     map[string]*string{"env": new("test")},
			Properties: &armprivatedns.PrivateZoneProperties{
				ProvisioningState: &provisioningState,
			},
		}

		mockClient := mocks.NewMockPrivateDNSZonesClient(ctrl)
		mockPager := newMockPrivateDNSZonesPager(ctrl, []*armprivatedns.PrivateZone{zone1, zone2NilName})

		mockClient.EXPECT().NewListByResourceGroupPager(resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewNetworkPrivateDNSZone(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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
		if sdpItems[0].UniqueAttributeValue() != "private1.example.com" {
			t.Errorf("Expected item name 'private1.example.com', got: %s", sdpItems[0].UniqueAttributeValue())
		}
	})

	t.Run("ListStream", func(t *testing.T) {
		zone1 := createAzurePrivateZone("stream1.example.com")
		zone2 := createAzurePrivateZone("stream2.example.com")

		mockClient := mocks.NewMockPrivateDNSZonesClient(ctrl)
		mockPager := newMockPrivateDNSZonesPager(ctrl, []*armprivatedns.PrivateZone{zone1, zone2})

		mockClient.EXPECT().NewListByResourceGroupPager(resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewNetworkPrivateDNSZone(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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

	t.Run("ErrorHandling", func(t *testing.T) {
		expectedErr := errors.New("private zone not found")

		mockClient := mocks.NewMockPrivateDNSZonesClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, "nonexistent-zone", nil).Return(
			armprivatedns.PrivateZonesClientGetResponse{}, expectedErr)

		wrapper := manual.NewNetworkPrivateDNSZone(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "nonexistent-zone", true)
		if qErr == nil {
			t.Error("Expected error when getting non-existent zone, but got nil")
		}
	})

	t.Run("InterfaceCompliance", func(t *testing.T) {
		mockClient := mocks.NewMockPrivateDNSZonesClient(ctrl)
		wrapper := manual.NewNetworkPrivateDNSZone(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		w := wrapper.(sources.Wrapper)

		permissions := w.IAMPermissions()
		if len(permissions) == 0 {
			t.Error("Expected IAMPermissions to return at least one permission")
		}
		expectedPermission := "Microsoft.Network/privateDnsZones/read"
		if !slices.Contains(permissions, expectedPermission) {
			t.Errorf("Expected IAMPermissions to include %s", expectedPermission)
		}

		potentialLinks := w.PotentialLinks()
		if !potentialLinks[azureshared.NetworkDNSRecordSet] {
			t.Error("Expected PotentialLinks to include NetworkDNSRecordSet")
		}
		if !potentialLinks[azureshared.NetworkDNSVirtualNetworkLink] {
			t.Error("Expected PotentialLinks to include NetworkDNSVirtualNetworkLink")
		}
		if !potentialLinks[stdlib.NetworkDNS] {
			t.Error("Expected PotentialLinks to include stdlib.NetworkDNS")
		}

		mappings := w.TerraformMappings()
		foundMapping := false
		for _, mapping := range mappings {
			if mapping.GetTerraformQueryMap() == "azurerm_private_dns_zone.name" {
				foundMapping = true
				if mapping.GetTerraformMethod() != sdp.QueryMethod_GET {
					t.Errorf("Expected TerraformMethod GET, got: %s", mapping.GetTerraformMethod())
				}
				break
			}
		}
		if !foundMapping {
			t.Error("Expected TerraformMappings to include 'azurerm_private_dns_zone.name'")
		}

		lookups := w.GetLookups()
		foundLookup := false
		for _, lookup := range lookups {
			if lookup.ItemType == azureshared.NetworkPrivateDNSZone {
				foundLookup = true
				break
			}
		}
		if !foundLookup {
			t.Error("Expected GetLookups to include NetworkPrivateDNSZone")
		}
	})
}

type mockPrivateDNSZonesPager struct {
	ctrl  *gomock.Controller
	items []*armprivatedns.PrivateZone
	index int
	more  bool
}

func newMockPrivateDNSZonesPager(ctrl *gomock.Controller, items []*armprivatedns.PrivateZone) clients.PrivateDNSZonesPager {
	return &mockPrivateDNSZonesPager{
		ctrl:  ctrl,
		items: items,
		index: 0,
		more:  len(items) > 0,
	}
}

func (m *mockPrivateDNSZonesPager) More() bool {
	return m.more
}

func (m *mockPrivateDNSZonesPager) NextPage(ctx context.Context) (armprivatedns.PrivateZonesClientListByResourceGroupResponse, error) {
	if m.index >= len(m.items) {
		m.more = false
		return armprivatedns.PrivateZonesClientListByResourceGroupResponse{
			PrivateZoneListResult: armprivatedns.PrivateZoneListResult{
				Value: []*armprivatedns.PrivateZone{},
			},
		}, nil
	}
	item := m.items[m.index]
	m.index++
	m.more = m.index < len(m.items)
	return armprivatedns.PrivateZonesClientListByResourceGroupResponse{
		PrivateZoneListResult: armprivatedns.PrivateZoneListResult{
			Value: []*armprivatedns.PrivateZone{item},
		},
	}, nil
}

func createAzurePrivateZone(zoneName string) *armprivatedns.PrivateZone {
	state := armprivatedns.ProvisioningStateSucceeded
	return &armprivatedns.PrivateZone{
		ID:       new("/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.Network/privateDnsZones/" + zoneName),
		Name:     new(zoneName),
		Type:     new("Microsoft.Network/privateDnsZones"),
		Location: new("global"),
		Tags: map[string]*string{
			"env":     new("test"),
			"project": new("testing"),
		},
		Properties: &armprivatedns.PrivateZoneProperties{
			ProvisioningState:     &state,
			MaxNumberOfRecordSets: new(int64(5000)),
			NumberOfRecordSets:    new(int64(0)),
		},
	}
}

// Ensure mockPrivateDNSZonesPager satisfies the pager interface at compile time.
var _ clients.PrivateDNSZonesPager = (*mockPrivateDNSZonesPager)(nil)
