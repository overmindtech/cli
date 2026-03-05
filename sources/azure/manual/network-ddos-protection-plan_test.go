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
)

func TestNetworkDdosProtectionPlan(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"

	t.Run("Get", func(t *testing.T) {
		planName := "test-ddos-plan"
		plan := createAzureDdosProtectionPlan(planName)

		mockClient := mocks.NewMockDdosProtectionPlansClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, planName, nil).Return(
			armnetwork.DdosProtectionPlansClientGetResponse{
				DdosProtectionPlan: *plan,
			}, nil)

		wrapper := manual.NewNetworkDdosProtectionPlan(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], planName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.NetworkDdosProtectionPlan.String() {
			t.Errorf("Expected type %s, got %s", azureshared.NetworkDdosProtectionPlan.String(), sdpItem.GetType())
		}

		if sdpItem.GetUniqueAttribute() != "name" {
			t.Errorf("Expected unique attribute 'name', got %s", sdpItem.GetUniqueAttribute())
		}

		if sdpItem.UniqueAttributeValue() != planName {
			t.Errorf("Expected unique attribute value %s, got %s", planName, sdpItem.UniqueAttributeValue())
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
		planName := "test-ddos-plan-with-links"
		plan := createAzureDdosProtectionPlanWithLinks(planName, subscriptionID, resourceGroup)

		mockClient := mocks.NewMockDdosProtectionPlansClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, planName, nil).Return(
			armnetwork.DdosProtectionPlansClientGetResponse{
				DdosProtectionPlan: *plan,
			}, nil)

		wrapper := manual.NewNetworkDdosProtectionPlan(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], planName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		t.Run("StaticTests", func(t *testing.T) {
			scope := subscriptionID + "." + resourceGroup
			queryTests := shared.QueryTests{
				{
					ExpectedType:   azureshared.NetworkVirtualNetwork.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-vnet",
					ExpectedScope:  scope,
				},
				{
					ExpectedType:   azureshared.NetworkPublicIPAddress.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-public-ip",
					ExpectedScope:  scope,
				},
			}
			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("GetWithEmptyName", func(t *testing.T) {
		mockClient := mocks.NewMockDdosProtectionPlansClient(ctrl)

		wrapper := manual.NewNetworkDdosProtectionPlan(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "", true)
		if qErr == nil {
			t.Error("Expected error when DDoS protection plan name is empty, but got nil")
		}
	})

	t.Run("Get_PlanWithNilName", func(t *testing.T) {
		provisioningState := armnetwork.ProvisioningStateSucceeded
		planWithNilName := &armnetwork.DdosProtectionPlan{
			Name:     nil,
			Location: new("eastus"),
			Properties: &armnetwork.DdosProtectionPlanPropertiesFormat{
				ProvisioningState: &provisioningState,
			},
		}

		mockClient := mocks.NewMockDdosProtectionPlansClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, "test-plan", nil).Return(
			armnetwork.DdosProtectionPlansClientGetResponse{
				DdosProtectionPlan: *planWithNilName,
			}, nil)

		wrapper := manual.NewNetworkDdosProtectionPlan(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "test-plan", true)
		if qErr == nil {
			t.Error("Expected error when DDoS protection plan has nil name, but got nil")
		}
	})

	t.Run("List", func(t *testing.T) {
		plan1 := createAzureDdosProtectionPlan("plan-1")
		plan2 := createAzureDdosProtectionPlan("plan-2")

		mockClient := mocks.NewMockDdosProtectionPlansClient(ctrl)
		mockPager := newMockDdosProtectionPlansPager(ctrl, []*armnetwork.DdosProtectionPlan{plan1, plan2})

		mockClient.EXPECT().NewListByResourceGroupPager(resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewNetworkDdosProtectionPlan(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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
			if item.GetType() != azureshared.NetworkDdosProtectionPlan.String() {
				t.Fatalf("Expected type %s, got: %s", azureshared.NetworkDdosProtectionPlan.String(), item.GetType())
			}
		}
	})

	t.Run("List_WithNilName", func(t *testing.T) {
		plan1 := createAzureDdosProtectionPlan("plan-1")
		provisioningState := armnetwork.ProvisioningStateSucceeded
		plan2NilName := &armnetwork.DdosProtectionPlan{
			Name:     nil,
			Location: new("eastus"),
			Tags:     map[string]*string{"env": new("test")},
			Properties: &armnetwork.DdosProtectionPlanPropertiesFormat{
				ProvisioningState: &provisioningState,
			},
		}

		mockClient := mocks.NewMockDdosProtectionPlansClient(ctrl)
		mockPager := newMockDdosProtectionPlansPager(ctrl, []*armnetwork.DdosProtectionPlan{plan1, plan2NilName})

		mockClient.EXPECT().NewListByResourceGroupPager(resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewNetworkDdosProtectionPlan(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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
		if sdpItems[0].UniqueAttributeValue() != "plan-1" {
			t.Errorf("Expected item name 'plan-1', got: %s", sdpItems[0].UniqueAttributeValue())
		}
	})

	t.Run("ListStream", func(t *testing.T) {
		plan1 := createAzureDdosProtectionPlan("stream-plan-1")
		plan2 := createAzureDdosProtectionPlan("stream-plan-2")

		mockClient := mocks.NewMockDdosProtectionPlansClient(ctrl)
		mockPager := newMockDdosProtectionPlansPager(ctrl, []*armnetwork.DdosProtectionPlan{plan1, plan2})

		mockClient.EXPECT().NewListByResourceGroupPager(resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewNetworkDdosProtectionPlan(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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
		expectedErr := errors.New("DDoS protection plan not found")

		mockClient := mocks.NewMockDdosProtectionPlansClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, "nonexistent-plan", nil).Return(
			armnetwork.DdosProtectionPlansClientGetResponse{}, expectedErr)

		wrapper := manual.NewNetworkDdosProtectionPlan(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "nonexistent-plan", true)
		if qErr == nil {
			t.Error("Expected error when getting non-existent DDoS protection plan, but got nil")
		}
	})

	t.Run("InterfaceCompliance", func(t *testing.T) {
		mockClient := mocks.NewMockDdosProtectionPlansClient(ctrl)
		wrapper := manual.NewNetworkDdosProtectionPlan(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		w := wrapper.(sources.Wrapper)

		permissions := w.IAMPermissions()
		if len(permissions) == 0 {
			t.Error("Expected IAMPermissions to return at least one permission")
		}
		expectedPermission := "Microsoft.Network/ddosProtectionPlans/read"
		if !slices.Contains(permissions, expectedPermission) {
			t.Errorf("Expected IAMPermissions to include %s", expectedPermission)
		}

		mappings := w.TerraformMappings()
		foundMapping := false
		for _, mapping := range mappings {
			if mapping.GetTerraformQueryMap() == "azurerm_network_ddos_protection_plan.name" {
				foundMapping = true
				if mapping.GetTerraformMethod() != sdp.QueryMethod_GET {
					t.Errorf("Expected TerraformMethod GET, got: %s", mapping.GetTerraformMethod())
				}
				break
			}
		}
		if !foundMapping {
			t.Error("Expected TerraformMappings to include 'azurerm_network_ddos_protection_plan.name'")
		}

		lookups := w.GetLookups()
		foundLookup := false
		for _, lookup := range lookups {
			if lookup.ItemType == azureshared.NetworkDdosProtectionPlan {
				foundLookup = true
				break
			}
		}
		if !foundLookup {
			t.Error("Expected GetLookups to include NetworkDdosProtectionPlan")
		}

		potentialLinks := w.PotentialLinks()
		for _, linkType := range []shared.ItemType{azureshared.NetworkVirtualNetwork, azureshared.NetworkPublicIPAddress} {
			if !potentialLinks[linkType] {
				t.Errorf("Expected PotentialLinks to include %s", linkType)
			}
		}
	})
}

type mockDdosProtectionPlansPager struct {
	ctrl  *gomock.Controller
	items []*armnetwork.DdosProtectionPlan
	index int
	more  bool
}

func newMockDdosProtectionPlansPager(ctrl *gomock.Controller, items []*armnetwork.DdosProtectionPlan) clients.DdosProtectionPlansPager {
	return &mockDdosProtectionPlansPager{
		ctrl:  ctrl,
		items: items,
		index: 0,
		more:  len(items) > 0,
	}
}

func (m *mockDdosProtectionPlansPager) More() bool {
	return m.more
}

func (m *mockDdosProtectionPlansPager) NextPage(ctx context.Context) (armnetwork.DdosProtectionPlansClientListByResourceGroupResponse, error) {
	if m.index >= len(m.items) {
		m.more = false
		return armnetwork.DdosProtectionPlansClientListByResourceGroupResponse{
			DdosProtectionPlanListResult: armnetwork.DdosProtectionPlanListResult{
				Value: []*armnetwork.DdosProtectionPlan{},
			},
		}, nil
	}
	item := m.items[m.index]
	m.index++
	m.more = m.index < len(m.items)
	return armnetwork.DdosProtectionPlansClientListByResourceGroupResponse{
		DdosProtectionPlanListResult: armnetwork.DdosProtectionPlanListResult{
			Value: []*armnetwork.DdosProtectionPlan{item},
		},
	}, nil
}

func createAzureDdosProtectionPlan(name string) *armnetwork.DdosProtectionPlan {
	provisioningState := armnetwork.ProvisioningStateSucceeded
	return &armnetwork.DdosProtectionPlan{
		ID:       new("/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.Network/ddosProtectionPlans/" + name),
		Name:     new(name),
		Type:     new("Microsoft.Network/ddosProtectionPlans"),
		Location: new("eastus"),
		Tags: map[string]*string{
			"env":     new("test"),
			"project": new("testing"),
		},
		Properties: &armnetwork.DdosProtectionPlanPropertiesFormat{
			ProvisioningState: &provisioningState,
		},
	}
}

func createAzureDdosProtectionPlanWithLinks(name, subscriptionID, resourceGroup string) *armnetwork.DdosProtectionPlan {
	plan := createAzureDdosProtectionPlan(name)
	plan.Properties.VirtualNetworks = []*armnetwork.SubResource{
		{ID: new("/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/virtualNetworks/test-vnet")},
	}
	plan.Properties.PublicIPAddresses = []*armnetwork.SubResource{
		{ID: new("/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/publicIPAddresses/test-public-ip")},
	}
	return plan
}

var _ clients.DdosProtectionPlansPager = (*mockDdosProtectionPlansPager)(nil)
