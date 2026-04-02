package manual_test

import (
	"context"
	"errors"
	"slices"
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

func TestNetworkIPGroup(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"

	t.Run("Get", func(t *testing.T) {
		ipGroupName := "test-ip-group"
		ipGroup := createAzureIPGroup(ipGroupName)

		mockClient := mocks.NewMockIPGroupsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, ipGroupName, nil).Return(
			armnetwork.IPGroupsClientGetResponse{
				IPGroup: *ipGroup,
			}, nil)

		wrapper := manual.NewNetworkIPGroup(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], ipGroupName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.NetworkIPGroup.String() {
			t.Errorf("Expected type %s, got %s", azureshared.NetworkIPGroup, sdpItem.GetType())
		}

		if sdpItem.GetUniqueAttribute() != "name" {
			t.Errorf("Expected unique attribute 'name', got %s", sdpItem.GetUniqueAttribute())
		}

		if sdpItem.UniqueAttributeValue() != ipGroupName {
			t.Errorf("Expected unique attribute value %s, got %s", ipGroupName, sdpItem.UniqueAttributeValue())
		}

		if sdpItem.GetTags()["env"] != "test" {
			t.Errorf("Expected tag 'env=test', got: %v", sdpItem.GetTags()["env"])
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					ExpectedType:   stdlib.NetworkIP.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "10.0.0.0/24",
					ExpectedScope:  "global",
				},
				{
					ExpectedType:   stdlib.NetworkIP.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "192.168.1.1",
					ExpectedScope:  "global",
				},
				{
					ExpectedType:   azureshared.NetworkFirewall.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-firewall",
					ExpectedScope:  subscriptionID + "." + resourceGroup,
				},
				{
					ExpectedType:   azureshared.NetworkFirewallPolicy.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-firewall-policy",
					ExpectedScope:  subscriptionID + "." + resourceGroup,
				},
			}
			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("GetWithEmptyName", func(t *testing.T) {
		mockClient := mocks.NewMockIPGroupsClient(ctrl)

		wrapper := manual.NewNetworkIPGroup(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "", true)
		if qErr == nil {
			t.Error("Expected error when IP group name is empty, but got nil")
		}
	})

	t.Run("Get_IPGroupWithNilName", func(t *testing.T) {
		provisioningState := armnetwork.ProvisioningStateSucceeded
		ipGroupWithNilName := &armnetwork.IPGroup{
			Name:     nil,
			Location: new("eastus"),
			Properties: &armnetwork.IPGroupPropertiesFormat{
				ProvisioningState: &provisioningState,
			},
		}

		mockClient := mocks.NewMockIPGroupsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, "test-ip-group", nil).Return(
			armnetwork.IPGroupsClientGetResponse{
				IPGroup: *ipGroupWithNilName,
			}, nil)

		wrapper := manual.NewNetworkIPGroup(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "test-ip-group", true)
		if qErr == nil {
			t.Error("Expected error when IP group has nil name, but got nil")
		}
	})

	t.Run("List", func(t *testing.T) {
		ipGroup1 := createAzureIPGroup("ip-group-1")
		ipGroup2 := createAzureIPGroup("ip-group-2")

		mockClient := mocks.NewMockIPGroupsClient(ctrl)
		mockPager := newMockIPGroupsPager(ctrl, []*armnetwork.IPGroup{ipGroup1, ipGroup2})

		mockClient.EXPECT().NewListByResourceGroupPager(resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewNetworkIPGroup(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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
			if item.GetType() != azureshared.NetworkIPGroup.String() {
				t.Fatalf("Expected type %s, got: %s", azureshared.NetworkIPGroup, item.GetType())
			}
		}
	})

	t.Run("List_WithNilName", func(t *testing.T) {
		ipGroup1 := createAzureIPGroup("ip-group-1")
		provisioningState := armnetwork.ProvisioningStateSucceeded
		ipGroup2NilName := &armnetwork.IPGroup{
			Name:     nil,
			Location: new("eastus"),
			Tags:     map[string]*string{"env": new("test")},
			Properties: &armnetwork.IPGroupPropertiesFormat{
				ProvisioningState: &provisioningState,
			},
		}

		mockClient := mocks.NewMockIPGroupsClient(ctrl)
		mockPager := newMockIPGroupsPager(ctrl, []*armnetwork.IPGroup{ipGroup1, ipGroup2NilName})

		mockClient.EXPECT().NewListByResourceGroupPager(resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewNetworkIPGroup(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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
		if sdpItems[0].UniqueAttributeValue() != "ip-group-1" {
			t.Errorf("Expected item name 'ip-group-1', got: %s", sdpItems[0].UniqueAttributeValue())
		}
	})

	t.Run("ListStream", func(t *testing.T) {
		ipGroup1 := createAzureIPGroup("stream-ip-group-1")
		ipGroup2 := createAzureIPGroup("stream-ip-group-2")

		mockClient := mocks.NewMockIPGroupsClient(ctrl)
		mockPager := newMockIPGroupsPager(ctrl, []*armnetwork.IPGroup{ipGroup1, ipGroup2})

		mockClient.EXPECT().NewListByResourceGroupPager(resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewNetworkIPGroup(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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
		expectedErr := errors.New("IP group not found")

		mockClient := mocks.NewMockIPGroupsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, "nonexistent-ip-group", nil).Return(
			armnetwork.IPGroupsClientGetResponse{}, expectedErr)

		wrapper := manual.NewNetworkIPGroup(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "nonexistent-ip-group", true)
		if qErr == nil {
			t.Error("Expected error when getting non-existent IP group, but got nil")
		}
	})

	t.Run("InterfaceCompliance", func(t *testing.T) {
		mockClient := mocks.NewMockIPGroupsClient(ctrl)
		wrapper := manual.NewNetworkIPGroup(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		w := wrapper.(sources.Wrapper)

		permissions := w.IAMPermissions()
		if len(permissions) == 0 {
			t.Error("Expected IAMPermissions to return at least one permission")
		}
		expectedPermission := "Microsoft.Network/ipGroups/read"
		if !slices.Contains(permissions, expectedPermission) {
			t.Errorf("Expected IAMPermissions to include %s", expectedPermission)
		}

		lookups := w.GetLookups()
		foundLookup := false
		for _, lookup := range lookups {
			if lookup.ItemType == azureshared.NetworkIPGroup {
				foundLookup = true
				break
			}
		}
		if !foundLookup {
			t.Error("Expected GetLookups to include NetworkIPGroup")
		}

		potentialLinks := w.PotentialLinks()
		if !potentialLinks[stdlib.NetworkIP] {
			t.Error("Expected PotentialLinks to include stdlib.NetworkIP")
		}
		if !potentialLinks[azureshared.NetworkFirewall] {
			t.Error("Expected PotentialLinks to include NetworkFirewall")
		}
		if !potentialLinks[azureshared.NetworkFirewallPolicy] {
			t.Error("Expected PotentialLinks to include NetworkFirewallPolicy")
		}
	})

	t.Run("HealthStatus", func(t *testing.T) {
		tests := []struct {
			name              string
			provisioningState armnetwork.ProvisioningState
			expectedHealth    sdp.Health
		}{
			{"Succeeded", armnetwork.ProvisioningStateSucceeded, sdp.Health_HEALTH_OK},
			{"Updating", armnetwork.ProvisioningStateUpdating, sdp.Health_HEALTH_PENDING},
			{"Deleting", armnetwork.ProvisioningStateDeleting, sdp.Health_HEALTH_PENDING},
			{"Failed", armnetwork.ProvisioningStateFailed, sdp.Health_HEALTH_ERROR},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				ipGroup := &armnetwork.IPGroup{
					ID:       new("/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.Network/ipGroups/test-ip-group"),
					Name:     new("test-ip-group"),
					Type:     new("Microsoft.Network/ipGroups"),
					Location: new("eastus"),
					Tags:     map[string]*string{},
					Properties: &armnetwork.IPGroupPropertiesFormat{
						ProvisioningState: &tc.provisioningState,
					},
				}

				mockClient := mocks.NewMockIPGroupsClient(ctrl)
				mockClient.EXPECT().Get(ctx, resourceGroup, "test-ip-group", nil).Return(
					armnetwork.IPGroupsClientGetResponse{
						IPGroup: *ipGroup,
					}, nil)

				wrapper := manual.NewNetworkIPGroup(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
				adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

				sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "test-ip-group", true)
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

type mockIPGroupsPager struct {
	ctrl  *gomock.Controller
	items []*armnetwork.IPGroup
	index int
	more  bool
}

func newMockIPGroupsPager(ctrl *gomock.Controller, items []*armnetwork.IPGroup) clients.IPGroupsPager {
	return &mockIPGroupsPager{
		ctrl:  ctrl,
		items: items,
		index: 0,
		more:  len(items) > 0,
	}
}

func (m *mockIPGroupsPager) More() bool {
	return m.more
}

func (m *mockIPGroupsPager) NextPage(ctx context.Context) (armnetwork.IPGroupsClientListByResourceGroupResponse, error) {
	if m.index >= len(m.items) {
		m.more = false
		return armnetwork.IPGroupsClientListByResourceGroupResponse{
			IPGroupListResult: armnetwork.IPGroupListResult{
				Value: []*armnetwork.IPGroup{},
			},
		}, nil
	}
	item := m.items[m.index]
	m.index++
	m.more = m.index < len(m.items)
	return armnetwork.IPGroupsClientListByResourceGroupResponse{
		IPGroupListResult: armnetwork.IPGroupListResult{
			Value: []*armnetwork.IPGroup{item},
		},
	}, nil
}

func createAzureIPGroup(name string) *armnetwork.IPGroup {
	provisioningState := armnetwork.ProvisioningStateSucceeded
	return &armnetwork.IPGroup{
		ID:       new("/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.Network/ipGroups/" + name),
		Name:     new(name),
		Type:     new("Microsoft.Network/ipGroups"),
		Location: new("eastus"),
		Tags: map[string]*string{
			"env":     new("test"),
			"project": new("testing"),
		},
		Properties: &armnetwork.IPGroupPropertiesFormat{
			ProvisioningState: &provisioningState,
			IPAddresses: []*string{
				new("10.0.0.0/24"),
				new("192.168.1.1"),
			},
			Firewalls: []*armnetwork.SubResource{
				{
					ID: new("/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.Network/azureFirewalls/test-firewall"),
				},
			},
			FirewallPolicies: []*armnetwork.SubResource{
				{
					ID: new("/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.Network/firewallPolicies/test-firewall-policy"),
				},
			},
		},
	}
}

var _ clients.IPGroupsPager = (*mockIPGroupsPager)(nil)
