package manual_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"
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

func TestComputeCapacityReservationGroup(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"
	scope := subscriptionID + "." + resourceGroup

	t.Run("Get", func(t *testing.T) {
		groupName := "test-crg"
		crg := createAzureCapacityReservationGroup(groupName)

		mockClient := mocks.NewMockCapacityReservationGroupsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, groupName, gomock.Eq(capacityReservationGroupGetOptions())).Return(
			armcompute.CapacityReservationGroupsClientGetResponse{
				CapacityReservationGroup: *crg,
			}, nil)

		wrapper := manual.NewComputeCapacityReservationGroup(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, scope, groupName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.ComputeCapacityReservationGroup.String() {
			t.Errorf("Expected type %s, got %s", azureshared.ComputeCapacityReservationGroup.String(), sdpItem.GetType())
		}

		if sdpItem.GetUniqueAttribute() != "name" {
			t.Errorf("Expected unique attribute 'name', got %s", sdpItem.GetUniqueAttribute())
		}

		if sdpItem.UniqueAttributeValue() != groupName {
			t.Errorf("Expected unique attribute value %s, got %s", groupName, sdpItem.UniqueAttributeValue())
		}

		if sdpItem.GetTags()["env"] != "test" {
			t.Errorf("Expected tag 'env=test', got: %v", sdpItem.GetTags()["env"])
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{}
			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("GetWithLinkedResources", func(t *testing.T) {
		groupName := "test-crg-with-links"
		crg := createAzureCapacityReservationGroupWithLinks(groupName, subscriptionID, resourceGroup, []string{"res-1", "res-2"}, []string{"vm-1", "vm-2"})

		mockClient := mocks.NewMockCapacityReservationGroupsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, groupName, gomock.Eq(capacityReservationGroupGetOptions())).Return(
			armcompute.CapacityReservationGroupsClientGetResponse{
				CapacityReservationGroup: *crg,
			}, nil)

		wrapper := manual.NewComputeCapacityReservationGroup(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, scope, groupName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					ExpectedType:   azureshared.ComputeCapacityReservation.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey(groupName, "res-1"),
					ExpectedScope:  scope,
				},
				{
					ExpectedType:   azureshared.ComputeCapacityReservation.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey(groupName, "res-2"),
					ExpectedScope:  scope,
				},
				{
					ExpectedType:   azureshared.ComputeVirtualMachine.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "vm-1",
					ExpectedScope:  scope,
				},
				{
					ExpectedType:   azureshared.ComputeVirtualMachine.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "vm-2",
					ExpectedScope:  scope,
				},
			}
			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("Get_InvalidQueryParts", func(t *testing.T) {
		mockClient := mocks.NewMockCapacityReservationGroupsClient(ctrl)

		wrapper := manual.NewComputeCapacityReservationGroup(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		_, qErr := wrapper.Get(ctx, scope)
		if qErr == nil {
			t.Error("Expected error when getting with no query parts, but got nil")
		}
	})

	t.Run("Get_EmptyName", func(t *testing.T) {
		mockClient := mocks.NewMockCapacityReservationGroupsClient(ctrl)

		wrapper := manual.NewComputeCapacityReservationGroup(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, scope, "", true)
		if qErr == nil {
			t.Error("Expected error when getting with empty name, but got nil")
		}
	})

	t.Run("Get_ClientError", func(t *testing.T) {
		expectedErr := errors.New("capacity reservation group not found")
		mockClient := mocks.NewMockCapacityReservationGroupsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, "nonexistent", gomock.Eq(capacityReservationGroupGetOptions())).Return(
			armcompute.CapacityReservationGroupsClientGetResponse{}, expectedErr)

		wrapper := manual.NewComputeCapacityReservationGroup(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, scope, "nonexistent", true)
		if qErr == nil {
			t.Error("Expected error when client returns error, but got nil")
		}
	})

	t.Run("List", func(t *testing.T) {
		crg1 := createAzureCapacityReservationGroup("test-crg-1")
		crg2 := createAzureCapacityReservationGroup("test-crg-2")

		mockClient := mocks.NewMockCapacityReservationGroupsClient(ctrl)
		mockPager := newMockCapacityReservationGroupsPager(ctrl, []*armcompute.CapacityReservationGroup{crg1, crg2})
		mockClient.EXPECT().NewListByResourceGroupPager(resourceGroup, gomock.Eq(capacityReservationGroupListOptions())).Return(mockPager)

		wrapper := manual.NewComputeCapacityReservationGroup(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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

	t.Run("ListStream", func(t *testing.T) {
		crg1 := createAzureCapacityReservationGroup("test-crg-1")
		crg2 := createAzureCapacityReservationGroup("test-crg-2")

		mockClient := mocks.NewMockCapacityReservationGroupsClient(ctrl)
		mockPager := newMockCapacityReservationGroupsPager(ctrl, []*armcompute.CapacityReservationGroup{crg1, crg2})
		mockClient.EXPECT().NewListByResourceGroupPager(resourceGroup, gomock.Eq(capacityReservationGroupListOptions())).Return(mockPager)

		wrapper := manual.NewComputeCapacityReservationGroup(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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

		listStreamable.ListStream(ctx, scope, true, stream)
		wg.Wait()

		if len(errs) != 0 {
			t.Fatalf("Expected no errors, got: %v", errs)
		}

		if len(items) != 2 {
			t.Fatalf("Expected 2 items, got: %d", len(items))
		}
	})

	t.Run("ListWithNilName", func(t *testing.T) {
		crg1 := createAzureCapacityReservationGroup("test-crg-1")
		crgNilName := &armcompute.CapacityReservationGroup{
			Name:     nil,
			Location: new("eastus"),
			Tags: map[string]*string{
				"env": new("test"),
			},
			Properties: &armcompute.CapacityReservationGroupProperties{},
		}

		mockClient := mocks.NewMockCapacityReservationGroupsClient(ctrl)
		mockPager := newMockCapacityReservationGroupsPager(ctrl, []*armcompute.CapacityReservationGroup{crg1, crgNilName})
		mockClient.EXPECT().NewListByResourceGroupPager(resourceGroup, gomock.Eq(capacityReservationGroupListOptions())).Return(mockPager)

		wrapper := manual.NewComputeCapacityReservationGroup(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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

	t.Run("ListWithPagerError", func(t *testing.T) {
		mockClient := mocks.NewMockCapacityReservationGroupsClient(ctrl)
		errorPager := newErrorCapacityReservationGroupsPager(ctrl)
		mockClient.EXPECT().NewListByResourceGroupPager(resourceGroup, gomock.Eq(capacityReservationGroupListOptions())).Return(errorPager)

		wrapper := manual.NewComputeCapacityReservationGroup(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		listable, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support List operation")
		}

		_, err := listable.List(ctx, scope, true)
		if err == nil {
			t.Error("Expected error when pager returns error, but got nil")
		}
	})

	t.Run("ListStreamWithPagerError", func(t *testing.T) {
		mockClient := mocks.NewMockCapacityReservationGroupsClient(ctrl)
		errorPager := newErrorCapacityReservationGroupsPager(ctrl)
		mockClient.EXPECT().NewListByResourceGroupPager(resourceGroup, gomock.Eq(capacityReservationGroupListOptions())).Return(errorPager)

		wrapper := manual.NewComputeCapacityReservationGroup(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		var errs []error
		mockErrorHandler := func(err error) {
			errs = append(errs, err)
		}

		stream := discovery.NewQueryResultStream(func(item *sdp.Item) {}, mockErrorHandler)

		listStreamable, ok := adapter.(discovery.ListStreamableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support ListStream operation")
		}

		listStreamable.ListStream(ctx, scope, true, stream)

		if len(errs) == 0 {
			t.Error("Expected error when pager returns error, but got none")
		}
	})
}

func capacityReservationGroupGetOptions() *armcompute.CapacityReservationGroupsClientGetOptions {
	return nil
}

func capacityReservationGroupListOptions() *armcompute.CapacityReservationGroupsClientListByResourceGroupOptions {
	expand := armcompute.ExpandTypesForGetCapacityReservationGroupsVirtualMachinesRef
	return &armcompute.CapacityReservationGroupsClientListByResourceGroupOptions{
		Expand: &expand,
	}
}

// createAzureCapacityReservationGroup creates a mock Azure Capacity Reservation Group for testing.
func createAzureCapacityReservationGroup(groupName string) *armcompute.CapacityReservationGroup {
	return &armcompute.CapacityReservationGroup{
		Name:     new(groupName),
		Location: new("eastus"),
		Tags: map[string]*string{
			"env":     new("test"),
			"project": new("testing"),
		},
		Properties: &armcompute.CapacityReservationGroupProperties{},
	}
}

// createAzureCapacityReservationGroupWithLinks creates a mock group with capacity reservation and VM links.
func createAzureCapacityReservationGroupWithLinks(groupName, subscriptionID, resourceGroup string, reservationNames, vmNames []string) *armcompute.CapacityReservationGroup {
	reservations := make([]*armcompute.SubResourceReadOnly, 0, len(reservationNames))
	for _, name := range reservationNames {
		reservations = append(reservations, &armcompute.SubResourceReadOnly{
			ID: new("/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Compute/capacityReservationGroups/" + groupName + "/capacityReservations/" + name),
		})
	}
	vms := make([]*armcompute.SubResourceReadOnly, 0, len(vmNames))
	for _, name := range vmNames {
		vms = append(vms, &armcompute.SubResourceReadOnly{
			ID: new("/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Compute/virtualMachines/" + name),
		})
	}
	return &armcompute.CapacityReservationGroup{
		Name:     new(groupName),
		Location: new("eastus"),
		Tags: map[string]*string{
			"env": new("test"),
		},
		Properties: &armcompute.CapacityReservationGroupProperties{
			CapacityReservations:      reservations,
			VirtualMachinesAssociated: vms,
		},
	}
}

// mockCapacityReservationGroupsPager is a mock pager for CapacityReservationGroupsClientListByResourceGroupResponse.
type mockCapacityReservationGroupsPager struct {
	ctrl  *gomock.Controller
	items []*armcompute.CapacityReservationGroup
	index int
	more  bool
}

func newMockCapacityReservationGroupsPager(ctrl *gomock.Controller, items []*armcompute.CapacityReservationGroup) clients.CapacityReservationGroupsPager {
	return &mockCapacityReservationGroupsPager{
		ctrl:  ctrl,
		items: items,
		index: 0,
		more:  len(items) > 0,
	}
}

func (m *mockCapacityReservationGroupsPager) More() bool {
	return m.more
}

func (m *mockCapacityReservationGroupsPager) NextPage(ctx context.Context) (armcompute.CapacityReservationGroupsClientListByResourceGroupResponse, error) {
	if m.index >= len(m.items) {
		m.more = false
		return armcompute.CapacityReservationGroupsClientListByResourceGroupResponse{
			CapacityReservationGroupListResult: armcompute.CapacityReservationGroupListResult{
				Value: []*armcompute.CapacityReservationGroup{},
			},
		}, nil
	}

	item := m.items[m.index]
	m.index++
	m.more = m.index < len(m.items)

	return armcompute.CapacityReservationGroupsClientListByResourceGroupResponse{
		CapacityReservationGroupListResult: armcompute.CapacityReservationGroupListResult{
			Value: []*armcompute.CapacityReservationGroup{item},
		},
	}, nil
}

// errorCapacityReservationGroupsPager is a mock pager that always returns an error.
type errorCapacityReservationGroupsPager struct {
	ctrl *gomock.Controller
}

func newErrorCapacityReservationGroupsPager(ctrl *gomock.Controller) clients.CapacityReservationGroupsPager {
	return &errorCapacityReservationGroupsPager{ctrl: ctrl}
}

func (e *errorCapacityReservationGroupsPager) More() bool {
	return true
}

func (e *errorCapacityReservationGroupsPager) NextPage(ctx context.Context) (armcompute.CapacityReservationGroupsClientListByResourceGroupResponse, error) {
	return armcompute.CapacityReservationGroupsClientListByResourceGroupResponse{}, errors.New("pager error")
}
