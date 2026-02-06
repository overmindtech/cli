package manual_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"
	"go.uber.org/mock/gomock"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	"github.com/overmindtech/cli/sources/azure/manual"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/azure/shared/mocks"
	"github.com/overmindtech/cli/sources/shared"
)

func TestComputeProximityPlacementGroup(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"
	scope := subscriptionID + "." + resourceGroup

	t.Run("Get", func(t *testing.T) {
		ppgName := "test-ppg"
		ppg := createAzureProximityPlacementGroup(ppgName, subscriptionID, resourceGroup)

		mockClient := mocks.NewMockProximityPlacementGroupsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, ppgName, nil).Return(
			armcompute.ProximityPlacementGroupsClientGetResponse{
				ProximityPlacementGroup: *ppg,
			}, nil)

		wrapper := manual.NewComputeProximityPlacementGroup(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, scope, ppgName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.ComputeProximityPlacementGroup.String() {
			t.Errorf("Expected type %s, got %s", azureshared.ComputeProximityPlacementGroup.String(), sdpItem.GetType())
		}

		if sdpItem.GetUniqueAttribute() != "name" {
			t.Errorf("Expected unique attribute 'name', got %s", sdpItem.GetUniqueAttribute())
		}

		if sdpItem.UniqueAttributeValue() != ppgName {
			t.Errorf("Expected unique attribute value %s, got %s", ppgName, sdpItem.UniqueAttributeValue())
		}

		if sdpItem.GetTags()["env"] != "test" {
			t.Errorf("Expected tag 'env=test', got: %v", sdpItem.GetTags()["env"])
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					ExpectedType:   azureshared.ComputeVirtualMachine.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-vm",
					ExpectedScope:  scope,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					ExpectedType:   azureshared.ComputeAvailabilitySet.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-avset",
					ExpectedScope:  scope,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					ExpectedType:   azureshared.ComputeVirtualMachineScaleSet.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-vmss",
					ExpectedScope:  scope,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
			}

			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("GetWithCrossResourceGroupLinks", func(t *testing.T) {
		ppgName := "test-ppg-cross-rg"
		ppg := createAzureProximityPlacementGroupWithCrossResourceGroupLinks(ppgName, subscriptionID)

		mockClient := mocks.NewMockProximityPlacementGroupsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, ppgName, nil).Return(
			armcompute.ProximityPlacementGroupsClientGetResponse{
				ProximityPlacementGroup: *ppg,
			}, nil)

		wrapper := manual.NewComputeProximityPlacementGroup(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, scope, ppgName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		expectedVMScope := subscriptionID + ".vm-rg"
		expectedAVSetScope := subscriptionID + ".avset-rg"
		expectedVMSSScope := subscriptionID + ".vmss-rg"

		for _, link := range sdpItem.GetLinkedItemQueries() {
			q := link.GetQuery()
			switch q.GetType() {
			case azureshared.ComputeVirtualMachine.String():
				if q.GetScope() != expectedVMScope {
					t.Errorf("Expected VM scope %s, got %s", expectedVMScope, q.GetScope())
				}
			case azureshared.ComputeAvailabilitySet.String():
				if q.GetScope() != expectedAVSetScope {
					t.Errorf("Expected Availability Set scope %s, got %s", expectedAVSetScope, q.GetScope())
				}
			case azureshared.ComputeVirtualMachineScaleSet.String():
				if q.GetScope() != expectedVMSSScope {
					t.Errorf("Expected VMSS scope %s, got %s", expectedVMSSScope, q.GetScope())
				}
			}
		}
	})

	t.Run("GetWithoutLinks", func(t *testing.T) {
		ppgName := "test-ppg-no-links"
		ppg := createAzureProximityPlacementGroupWithoutLinks(ppgName)

		mockClient := mocks.NewMockProximityPlacementGroupsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, ppgName, nil).Return(
			armcompute.ProximityPlacementGroupsClientGetResponse{
				ProximityPlacementGroup: *ppg,
			}, nil)

		wrapper := manual.NewComputeProximityPlacementGroup(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, scope, ppgName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if len(sdpItem.GetLinkedItemQueries()) != 0 {
			t.Errorf("Expected no linked queries, got %d", len(sdpItem.GetLinkedItemQueries()))
		}
	})

	t.Run("List", func(t *testing.T) {
		ppg1 := createAzureProximityPlacementGroup("test-ppg-1", subscriptionID, resourceGroup)
		ppg2 := createAzureProximityPlacementGroup("test-ppg-2", subscriptionID, resourceGroup)

		mockClient := mocks.NewMockProximityPlacementGroupsClient(ctrl)
		mockPager := newMockProximityPlacementGroupsPager(ctrl, []*armcompute.ProximityPlacementGroup{ppg1, ppg2})

		mockClient.EXPECT().ListByResourceGroup(ctx, resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewComputeProximityPlacementGroup(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		listable, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support List operation")
		}

		sdpItems, err := listable.List(ctx, scope, true)
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
		}
	})

	// ListStream is not implemented for the proximity placement group adapter
	// (wrapper does not implement ListStreamableWrapper), so no ListStream test.

	t.Run("ListWithNilName", func(t *testing.T) {
		ppg1 := createAzureProximityPlacementGroup("test-ppg-1", subscriptionID, resourceGroup)
		ppgNilName := &armcompute.ProximityPlacementGroup{
			Name:     nil,
			Location: to.Ptr("eastus"),
			Tags: map[string]*string{
				"env": to.Ptr("test"),
			},
		}

		mockClient := mocks.NewMockProximityPlacementGroupsClient(ctrl)
		mockPager := newMockProximityPlacementGroupsPager(ctrl, []*armcompute.ProximityPlacementGroup{ppg1, ppgNilName})

		mockClient.EXPECT().ListByResourceGroup(ctx, resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewComputeProximityPlacementGroup(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		listable, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support List operation")
		}

		sdpItems, err := listable.List(ctx, scope, true)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if len(sdpItems) != 1 {
			t.Fatalf("Expected 1 item (nil name skipped), got: %d", len(sdpItems))
		}
	})

	t.Run("GetError", func(t *testing.T) {
		expectedErr := errors.New("proximity placement group not found")

		mockClient := mocks.NewMockProximityPlacementGroupsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, "nonexistent-ppg", nil).Return(
			armcompute.ProximityPlacementGroupsClientGetResponse{}, expectedErr)

		wrapper := manual.NewComputeProximityPlacementGroup(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, scope, "nonexistent-ppg", true)
		if qErr == nil {
			t.Error("Expected error when getting non-existent proximity placement group, but got nil")
		}
	})

	t.Run("GetWithEmptyName", func(t *testing.T) {
		mockClient := mocks.NewMockProximityPlacementGroupsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, "", nil).Return(
			armcompute.ProximityPlacementGroupsClientGetResponse{}, errors.New("proximity placement group name is required"))

		wrapper := manual.NewComputeProximityPlacementGroup(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, scope, "", true)
		if qErr == nil {
			t.Error("Expected error when getting proximity placement group with empty name, but got nil")
		}
	})

}

func createAzureProximityPlacementGroup(ppgName, subscriptionID, resourceGroup string) *armcompute.ProximityPlacementGroup {
	baseID := "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Compute"
	return &armcompute.ProximityPlacementGroup{
		Name:     to.Ptr(ppgName),
		Location: to.Ptr("eastus"),
		Tags: map[string]*string{
			"env":     to.Ptr("test"),
			"project": to.Ptr("testing"),
		},
		Properties: &armcompute.ProximityPlacementGroupProperties{
			ProximityPlacementGroupType: to.Ptr(armcompute.ProximityPlacementGroupTypeStandard),
			VirtualMachines: []*armcompute.SubResourceWithColocationStatus{
				{ID: to.Ptr(baseID + "/virtualMachines/test-vm")},
			},
			AvailabilitySets: []*armcompute.SubResourceWithColocationStatus{
				{ID: to.Ptr(baseID + "/availabilitySets/test-avset")},
			},
			VirtualMachineScaleSets: []*armcompute.SubResourceWithColocationStatus{
				{ID: to.Ptr(baseID + "/virtualMachineScaleSets/test-vmss")},
			},
		},
		Zones: []*string{to.Ptr("1")},
	}
}

func createAzureProximityPlacementGroupWithCrossResourceGroupLinks(ppgName, subscriptionID string) *armcompute.ProximityPlacementGroup {
	return &armcompute.ProximityPlacementGroup{
		Name:     to.Ptr(ppgName),
		Location: to.Ptr("eastus"),
		Tags: map[string]*string{
			"env": to.Ptr("test"),
		},
		Properties: &armcompute.ProximityPlacementGroupProperties{
			ProximityPlacementGroupType: to.Ptr(armcompute.ProximityPlacementGroupTypeStandard),
			VirtualMachines: []*armcompute.SubResourceWithColocationStatus{
				{ID: to.Ptr("/subscriptions/" + subscriptionID + "/resourceGroups/vm-rg/providers/Microsoft.Compute/virtualMachines/test-vm")},
			},
			AvailabilitySets: []*armcompute.SubResourceWithColocationStatus{
				{ID: to.Ptr("/subscriptions/" + subscriptionID + "/resourceGroups/avset-rg/providers/Microsoft.Compute/availabilitySets/test-avset")},
			},
			VirtualMachineScaleSets: []*armcompute.SubResourceWithColocationStatus{
				{ID: to.Ptr("/subscriptions/" + subscriptionID + "/resourceGroups/vmss-rg/providers/Microsoft.Compute/virtualMachineScaleSets/test-vmss")},
			},
		},
	}
}

func createAzureProximityPlacementGroupWithoutLinks(ppgName string) *armcompute.ProximityPlacementGroup {
	return &armcompute.ProximityPlacementGroup{
		Name:     to.Ptr(ppgName),
		Location: to.Ptr("eastus"),
		Tags: map[string]*string{
			"env": to.Ptr("test"),
		},
		Properties: &armcompute.ProximityPlacementGroupProperties{
			ProximityPlacementGroupType: to.Ptr(armcompute.ProximityPlacementGroupTypeStandard),
		},
	}
}

type mockProximityPlacementGroupsPager struct {
	ctrl  *gomock.Controller
	items []*armcompute.ProximityPlacementGroup
	index int
	more  bool
}

func newMockProximityPlacementGroupsPager(ctrl *gomock.Controller, items []*armcompute.ProximityPlacementGroup) clients.ProximityPlacementGroupsPager {
	return &mockProximityPlacementGroupsPager{
		ctrl:  ctrl,
		items: items,
		index: 0,
		more:  len(items) > 0,
	}
}

func (m *mockProximityPlacementGroupsPager) More() bool {
	return m.more
}

func (m *mockProximityPlacementGroupsPager) NextPage(ctx context.Context) (armcompute.ProximityPlacementGroupsClientListByResourceGroupResponse, error) {
	if m.index >= len(m.items) {
		m.more = false
		return armcompute.ProximityPlacementGroupsClientListByResourceGroupResponse{
			ProximityPlacementGroupListResult: armcompute.ProximityPlacementGroupListResult{
				Value: []*armcompute.ProximityPlacementGroup{},
			},
		}, nil
	}

	item := m.items[m.index]
	m.index++
	m.more = m.index < len(m.items)

	return armcompute.ProximityPlacementGroupsClientListByResourceGroupResponse{
		ProximityPlacementGroupListResult: armcompute.ProximityPlacementGroupListResult{
			Value: []*armcompute.ProximityPlacementGroup{item},
		},
	}, nil
}
