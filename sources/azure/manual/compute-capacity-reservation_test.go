package manual

import (
	"context"
	"errors"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"
	"go.uber.org/mock/gomock"

	"github.com/overmindtech/cli/go/discovery"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/azure/shared/mocks"
	"github.com/overmindtech/cli/sources/shared"
)

func createAzureCapacityReservation(reservationName, groupName string) *armcompute.CapacityReservation {
	return &armcompute.CapacityReservation{
		ID:       new("/subscriptions/sub/resourceGroups/rg/providers/Microsoft.Compute/capacityReservationGroups/" + groupName + "/capacityReservations/" + reservationName),
		Name:     new(reservationName),
		Type:     new("Microsoft.Compute/capacityReservationGroups/capacityReservations"),
		Location: new("eastus"),
		Tags:     map[string]*string{"env": new("test")},
		SKU: &armcompute.SKU{
			Name:     new("Standard_D2s_v3"),
			Capacity: new(int64(1)),
		},
		Properties: &armcompute.CapacityReservationProperties{
			ProvisioningState: new("Succeeded"),
		},
	}
}

func createAzureCapacityReservationWithVMs(reservationName, groupName, subscriptionID, resourceGroup string, vmNames ...string) *armcompute.CapacityReservation {
	vms := make([]*armcompute.SubResourceReadOnly, 0, len(vmNames))
	for _, vmName := range vmNames {
		vms = append(vms, &armcompute.SubResourceReadOnly{
			ID: new("/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Compute/virtualMachines/" + vmName),
		})
	}
	return &armcompute.CapacityReservation{
		ID:       new("/subscriptions/sub/resourceGroups/rg/providers/Microsoft.Compute/capacityReservationGroups/" + groupName + "/capacityReservations/" + reservationName),
		Name:     new(reservationName),
		Type:     new("Microsoft.Compute/capacityReservationGroups/capacityReservations"),
		Location: new("eastus"),
		Tags:     map[string]*string{"env": new("test")},
		SKU: &armcompute.SKU{
			Name:     new("Standard_D2s_v3"),
			Capacity: new(int64(1)),
		},
		Properties: &armcompute.CapacityReservationProperties{
			ProvisioningState:         new("Succeeded"),
			VirtualMachinesAssociated: vms,
		},
	}
}

type mockCapacityReservationsPager struct {
	items []*armcompute.CapacityReservation
	index int
}

func (m *mockCapacityReservationsPager) More() bool {
	return m.index < len(m.items)
}

func (m *mockCapacityReservationsPager) NextPage(ctx context.Context) (armcompute.CapacityReservationsClientListByCapacityReservationGroupResponse, error) {
	if m.index >= len(m.items) {
		return armcompute.CapacityReservationsClientListByCapacityReservationGroupResponse{
			CapacityReservationListResult: armcompute.CapacityReservationListResult{
				Value: []*armcompute.CapacityReservation{},
			},
		}, nil
	}
	item := m.items[m.index]
	m.index++
	return armcompute.CapacityReservationsClientListByCapacityReservationGroupResponse{
		CapacityReservationListResult: armcompute.CapacityReservationListResult{
			Value: []*armcompute.CapacityReservation{item},
		},
	}, nil
}

type errorCapacityReservationsPager struct{}

func (e *errorCapacityReservationsPager) More() bool {
	return true
}

func (e *errorCapacityReservationsPager) NextPage(ctx context.Context) (armcompute.CapacityReservationsClientListByCapacityReservationGroupResponse, error) {
	return armcompute.CapacityReservationsClientListByCapacityReservationGroupResponse{}, errors.New("pager error")
}

type testCapacityReservationsClient struct {
	*mocks.MockCapacityReservationsClient
	pager clients.CapacityReservationsPager
}

func (t *testCapacityReservationsClient) NewListByCapacityReservationGroupPager(resourceGroupName string, capacityReservationGroupName string, options *armcompute.CapacityReservationsClientListByCapacityReservationGroupOptions) clients.CapacityReservationsPager {
	if t.pager != nil {
		return t.pager
	}
	return t.MockCapacityReservationsClient.NewListByCapacityReservationGroupPager(resourceGroupName, capacityReservationGroupName, options)
}

func TestComputeCapacityReservation(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"
	scope := subscriptionID + "." + resourceGroup
	groupName := "test-crg"
	reservationName := "test-reservation"

	t.Run("Get", func(t *testing.T) {
		res := createAzureCapacityReservation(reservationName, groupName)

		mockClient := mocks.NewMockCapacityReservationsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, groupName, reservationName, gomock.Eq(capacityReservationGetOptions())).Return(
			armcompute.CapacityReservationsClientGetResponse{
				CapacityReservation: *res,
			}, nil)

		wrapper := NewComputeCapacityReservation(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(groupName, reservationName)
		sdpItem, qErr := adapter.Get(ctx, scope, query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.ComputeCapacityReservation.String() {
			t.Errorf("Expected type %s, got %s", azureshared.ComputeCapacityReservation.String(), sdpItem.GetType())
		}

		if sdpItem.GetUniqueAttribute() != "uniqueAttr" {
			t.Errorf("Expected unique attribute 'uniqueAttr', got %s", sdpItem.GetUniqueAttribute())
		}

		expectedUnique := shared.CompositeLookupKey(groupName, reservationName)
		if sdpItem.UniqueAttributeValue() != expectedUnique {
			t.Errorf("Expected unique attribute value %s, got %s", expectedUnique, sdpItem.UniqueAttributeValue())
		}

		if sdpItem.GetTags()["env"] != "test" {
			t.Errorf("Expected tag env=test, got: %v", sdpItem.GetTags()["env"])
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{ExpectedType: azureshared.ComputeCapacityReservationGroup.String(), ExpectedMethod: sdp.QueryMethod_GET, ExpectedQuery: groupName, ExpectedScope: scope},
			}
			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("Get_WithVMLinks", func(t *testing.T) {
		res := createAzureCapacityReservationWithVMs(reservationName, groupName, subscriptionID, resourceGroup, "vm-1", "vm-2")

		mockClient := mocks.NewMockCapacityReservationsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, groupName, reservationName, gomock.Eq(capacityReservationGetOptions())).Return(
			armcompute.CapacityReservationsClientGetResponse{
				CapacityReservation: *res,
			}, nil)

		wrapper := NewComputeCapacityReservation(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(groupName, reservationName)
		sdpItem, qErr := adapter.Get(ctx, scope, query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		queryTests := shared.QueryTests{
			{ExpectedType: azureshared.ComputeCapacityReservationGroup.String(), ExpectedMethod: sdp.QueryMethod_GET, ExpectedQuery: groupName, ExpectedScope: scope},
			{ExpectedType: azureshared.ComputeVirtualMachine.String(), ExpectedMethod: sdp.QueryMethod_GET, ExpectedQuery: "vm-1", ExpectedScope: scope},
			{ExpectedType: azureshared.ComputeVirtualMachine.String(), ExpectedMethod: sdp.QueryMethod_GET, ExpectedQuery: "vm-2", ExpectedScope: scope},
		}
		shared.RunStaticTests(t, adapter, sdpItem, queryTests)
	})

	t.Run("Get_InvalidQueryParts", func(t *testing.T) {
		mockClient := mocks.NewMockCapacityReservationsClient(ctrl)
		wrapper := NewComputeCapacityReservation(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, scope, groupName, true)
		if qErr == nil {
			t.Error("Expected error when Get with wrong number of query parts, but got nil")
		}
	})

	t.Run("Get_EmptyGroupName", func(t *testing.T) {
		mockClient := mocks.NewMockCapacityReservationsClient(ctrl)
		wrapper := NewComputeCapacityReservation(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey("", reservationName)
		_, qErr := adapter.Get(ctx, scope, query, true)
		if qErr == nil {
			t.Error("Expected error when capacity reservation group name is empty, but got nil")
		}
	})

	t.Run("Get_EmptyReservationName", func(t *testing.T) {
		mockClient := mocks.NewMockCapacityReservationsClient(ctrl)
		wrapper := NewComputeCapacityReservation(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(groupName, "")
		_, qErr := adapter.Get(ctx, scope, query, true)
		if qErr == nil {
			t.Error("Expected error when capacity reservation name is empty, but got nil")
		}
	})

	t.Run("Get_ClientError", func(t *testing.T) {
		expectedErr := errors.New("reservation not found")
		mockClient := mocks.NewMockCapacityReservationsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, groupName, "nonexistent", gomock.Eq(capacityReservationGetOptions())).Return(
			armcompute.CapacityReservationsClientGetResponse{}, expectedErr)

		wrapper := NewComputeCapacityReservation(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(groupName, "nonexistent")
		_, qErr := adapter.Get(ctx, scope, query, true)
		if qErr == nil {
			t.Error("Expected error when client returns error, but got nil")
		}
	})

	t.Run("Search", func(t *testing.T) {
		res1 := createAzureCapacityReservation("res-1", groupName)
		res2 := createAzureCapacityReservation("res-2", groupName)

		mockClient := mocks.NewMockCapacityReservationsClient(ctrl)
		pager := &mockCapacityReservationsPager{
			items: []*armcompute.CapacityReservation{res1, res2},
		}
		testClient := &testCapacityReservationsClient{
			MockCapacityReservationsClient: mockClient,
			pager:                          pager,
		}

		wrapper := NewComputeCapacityReservation(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support Search operation")
		}

		sdpItems, err := searchable.Search(ctx, scope, groupName, true)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if len(sdpItems) != 2 {
			t.Fatalf("Expected 2 items, got: %d", len(sdpItems))
		}

		for _, item := range sdpItems {
			if err := item.Validate(); err != nil {
				t.Errorf("Expected valid item, got: %v", err)
			}
		}
	})

	t.Run("Search_InvalidQueryParts", func(t *testing.T) {
		mockClient := mocks.NewMockCapacityReservationsClient(ctrl)
		wrapper := NewComputeCapacityReservation(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		_, qErr := wrapper.Search(ctx, scope, groupName, reservationName)
		if qErr == nil {
			t.Error("Expected error when Search with wrong number of query parts, but got nil")
		}
	})

	t.Run("Search_EmptyGroupName", func(t *testing.T) {
		mockClient := mocks.NewMockCapacityReservationsClient(ctrl)
		wrapper := NewComputeCapacityReservation(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		_, qErr := wrapper.Search(ctx, scope, "")
		if qErr == nil {
			t.Error("Expected error when capacity reservation group name is empty, but got nil")
		}
	})

	t.Run("Search_PagerError", func(t *testing.T) {
		mockClient := mocks.NewMockCapacityReservationsClient(ctrl)
		errorPager := &errorCapacityReservationsPager{}
		testClient := &testCapacityReservationsClient{
			MockCapacityReservationsClient: mockClient,
			pager:                          errorPager,
		}

		wrapper := NewComputeCapacityReservation(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support Search operation")
		}

		_, err := searchable.Search(ctx, scope, groupName, true)
		if err == nil {
			t.Error("Expected error when pager returns error, but got nil")
		}
	})

	t.Run("PotentialLinks", func(t *testing.T) {
		mockClient := mocks.NewMockCapacityReservationsClient(ctrl)
		wrapper := NewComputeCapacityReservation(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		links := wrapper.PotentialLinks()
		expected := map[shared.ItemType]bool{
			azureshared.ComputeCapacityReservationGroup: true,
			azureshared.ComputeVirtualMachine:           true,
		}
		for itemType, want := range expected {
			if got := links[itemType]; got != want {
				t.Errorf("PotentialLinks()[%v] = %v, want %v", itemType, got, want)
			}
		}
	})

	t.Run("ImplementsSearchableAdapter", func(t *testing.T) {
		mockClient := mocks.NewMockCapacityReservationsClient(ctrl)
		wrapper := NewComputeCapacityReservation(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Error("Adapter should implement SearchableAdapter interface")
		}
	})
}
