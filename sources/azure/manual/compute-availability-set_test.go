package manual_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v6"
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

func TestComputeAvailabilitySet(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"

	t.Run("Get", func(t *testing.T) {
		availabilitySetName := "test-avset"
		avSet := createAzureAvailabilitySet(availabilitySetName)

		mockClient := mocks.NewMockAvailabilitySetsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, availabilitySetName, nil).Return(
			armcompute.AvailabilitySetsClientGetResponse{
				AvailabilitySet: *avSet,
			}, nil)

		wrapper := manual.NewComputeAvailabilitySet(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], availabilitySetName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.ComputeAvailabilitySet.String() {
			t.Errorf("Expected type %s, got %s", azureshared.ComputeAvailabilitySet, sdpItem.GetType())
		}

		if sdpItem.GetUniqueAttribute() != "name" {
			t.Errorf("Expected unique attribute 'name', got %s", sdpItem.GetUniqueAttribute())
		}

		if sdpItem.UniqueAttributeValue() != availabilitySetName {
			t.Errorf("Expected unique attribute value %s, got %s", availabilitySetName, sdpItem.UniqueAttributeValue())
		}

		if sdpItem.GetTags()["env"] != "test" {
			t.Errorf("Expected tag 'env=test', got: %v", sdpItem.GetTags()["env"])
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					// Properties.ProximityPlacementGroup.ID
					ExpectedType:   azureshared.ComputeProximityPlacementGroup.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-ppg",
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					// Properties.VirtualMachines[0].ID
					ExpectedType:   azureshared.ComputeVirtualMachine.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-vm-1",
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					// Properties.VirtualMachines[1].ID
					ExpectedType:   azureshared.ComputeVirtualMachine.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-vm-2",
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
			}

			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("GetWithCrossResourceGroupLinks", func(t *testing.T) {
		availabilitySetName := "test-avset-cross-rg"
		avSet := createAzureAvailabilitySetWithCrossResourceGroupLinks(availabilitySetName, subscriptionID)

		mockClient := mocks.NewMockAvailabilitySetsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, availabilitySetName, nil).Return(
			armcompute.AvailabilitySetsClientGetResponse{
				AvailabilitySet: *avSet,
			}, nil)

		wrapper := manual.NewComputeAvailabilitySet(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], availabilitySetName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		// Verify that links use the correct scope from different resource groups
		foundPPGLink := false
		foundVMLink := false
		for _, link := range sdpItem.GetLinkedItemQueries() {
			if link.GetQuery().GetType() == azureshared.ComputeProximityPlacementGroup.String() {
				foundPPGLink = true
				expectedScope := subscriptionID + ".other-rg"
				if link.GetQuery().GetScope() != expectedScope {
					t.Errorf("Expected PPG scope %s, got %s", expectedScope, link.GetQuery().GetScope())
				}
			}
			if link.GetQuery().GetType() == azureshared.ComputeVirtualMachine.String() {
				foundVMLink = true
				expectedScope := subscriptionID + ".vm-rg"
				if link.GetQuery().GetScope() != expectedScope {
					t.Errorf("Expected VM scope %s, got %s", expectedScope, link.GetQuery().GetScope())
				}
			}
		}

		if !foundPPGLink {
			t.Error("Expected to find Proximity Placement Group link")
		}
		if !foundVMLink {
			t.Error("Expected to find Virtual Machine link")
		}
	})

	t.Run("GetWithoutLinks", func(t *testing.T) {
		availabilitySetName := "test-avset-no-links"
		avSet := createAzureAvailabilitySetWithoutLinks(availabilitySetName)

		mockClient := mocks.NewMockAvailabilitySetsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, availabilitySetName, nil).Return(
			armcompute.AvailabilitySetsClientGetResponse{
				AvailabilitySet: *avSet,
			}, nil)

		wrapper := manual.NewComputeAvailabilitySet(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], availabilitySetName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if len(sdpItem.GetLinkedItemQueries()) != 0 {
			t.Errorf("Expected no linked queries, got %d", len(sdpItem.GetLinkedItemQueries()))
		}
	})

	t.Run("List", func(t *testing.T) {
		avSet1 := createAzureAvailabilitySet("test-avset-1")
		avSet2 := createAzureAvailabilitySet("test-avset-2")

		mockClient := mocks.NewMockAvailabilitySetsClient(ctrl)
		mockPager := newMockAvailabilitySetsPager(ctrl, []*armcompute.AvailabilitySet{avSet1, avSet2})

		mockClient.EXPECT().NewListPager(resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewComputeAvailabilitySet(mockClient, subscriptionID, resourceGroup)
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
		}
	})

	t.Run("ListStream", func(t *testing.T) {
		avSet1 := createAzureAvailabilitySet("test-avset-1")
		avSet2 := createAzureAvailabilitySet("test-avset-2")

		mockClient := mocks.NewMockAvailabilitySetsClient(ctrl)
		mockPager := newMockAvailabilitySetsPager(ctrl, []*armcompute.AvailabilitySet{avSet1, avSet2})

		mockClient.EXPECT().NewListPager(resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewComputeAvailabilitySet(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		wg := &sync.WaitGroup{}
		wg.Add(2) // we added two items

		var items []*sdp.Item
		mockItemHandler := func(item *sdp.Item) {
			items = append(items, item)
			wg.Done() // signal that we processed an item
		}

		var errs []error
		mockErrorHandler := func(err error) {
			errs = append(errs, err)
		}

		stream := discovery.NewQueryResultStream(mockItemHandler, mockErrorHandler)

		// Check if adapter supports list streaming
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

		// Verify adapter doesn't support SearchStream
		_, ok = adapter.(discovery.SearchStreamableAdapter)
		if ok {
			t.Fatalf("Adapter should not support SearchStream operation")
		}
	})

	t.Run("ListWithNilName", func(t *testing.T) {
		avSet1 := createAzureAvailabilitySet("test-avset-1")
		avSetNilName := &armcompute.AvailabilitySet{
			Name:     nil, // nil name should be skipped
			Location: to.Ptr("eastus"),
			Tags: map[string]*string{
				"env": to.Ptr("test"),
			},
		}

		mockClient := mocks.NewMockAvailabilitySetsClient(ctrl)
		mockPager := newMockAvailabilitySetsPager(ctrl, []*armcompute.AvailabilitySet{avSet1, avSetNilName})

		mockClient.EXPECT().NewListPager(resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewComputeAvailabilitySet(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		listable, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support List operation")
		}

		sdpItems, err := listable.List(ctx, wrapper.Scopes()[0], true)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		// Should only return 1 item (the one with a name)
		if len(sdpItems) != 1 {
			t.Fatalf("Expected 1 item (nil name skipped), got: %d", len(sdpItems))
		}
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		expectedErr := errors.New("availability set not found")

		mockClient := mocks.NewMockAvailabilitySetsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, "nonexistent-avset", nil).Return(
			armcompute.AvailabilitySetsClientGetResponse{}, expectedErr)

		wrapper := manual.NewComputeAvailabilitySet(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "nonexistent-avset", true)
		if qErr == nil {
			t.Error("Expected error when getting non-existent availability set, but got nil")
		}
	})

	t.Run("GetWithEmptyName", func(t *testing.T) {
		mockClient := mocks.NewMockAvailabilitySetsClient(ctrl)

		wrapper := manual.NewComputeAvailabilitySet(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "", true)
		if qErr == nil {
			t.Error("Expected error when getting availability set with empty name, but got nil")
		}
	})

	t.Run("GetWithInsufficientQueryParts", func(t *testing.T) {
		mockClient := mocks.NewMockAvailabilitySetsClient(ctrl)

		wrapper := manual.NewComputeAvailabilitySet(mockClient, subscriptionID, resourceGroup)
		// Test the wrapper's Get method directly with insufficient query parts
		_, qErr := wrapper.Get(ctx)
		if qErr == nil {
			t.Error("Expected error when getting availability set with insufficient query parts, but got nil")
		}
	})
}

// createAzureAvailabilitySet creates a mock Azure Availability Set for testing
func createAzureAvailabilitySet(avSetName string) *armcompute.AvailabilitySet {
	return &armcompute.AvailabilitySet{
		Name:     to.Ptr(avSetName),
		Location: to.Ptr("eastus"),
		Tags: map[string]*string{
			"env":     to.Ptr("test"),
			"project": to.Ptr("testing"),
		},
		Properties: &armcompute.AvailabilitySetProperties{
			PlatformFaultDomainCount:  to.Ptr(int32(2)),
			PlatformUpdateDomainCount: to.Ptr(int32(5)),
			ProximityPlacementGroup: &armcompute.SubResource{
				ID: to.Ptr("/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.Compute/proximityPlacementGroups/test-ppg"),
			},
			VirtualMachines: []*armcompute.SubResource{
				{
					ID: to.Ptr("/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.Compute/virtualMachines/test-vm-1"),
				},
				{
					ID: to.Ptr("/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.Compute/virtualMachines/test-vm-2"),
				},
			},
		},
	}
}

// createAzureAvailabilitySetWithCrossResourceGroupLinks creates a mock Availability Set
// with links to resources in different resource groups
func createAzureAvailabilitySetWithCrossResourceGroupLinks(avSetName, subscriptionID string) *armcompute.AvailabilitySet {
	return &armcompute.AvailabilitySet{
		Name:     to.Ptr(avSetName),
		Location: to.Ptr("eastus"),
		Tags: map[string]*string{
			"env": to.Ptr("test"),
		},
		Properties: &armcompute.AvailabilitySetProperties{
			PlatformFaultDomainCount:  to.Ptr(int32(2)),
			PlatformUpdateDomainCount: to.Ptr(int32(5)),
			ProximityPlacementGroup: &armcompute.SubResource{
				ID: to.Ptr("/subscriptions/" + subscriptionID + "/resourceGroups/other-rg/providers/Microsoft.Compute/proximityPlacementGroups/test-ppg"),
			},
			VirtualMachines: []*armcompute.SubResource{
				{
					ID: to.Ptr("/subscriptions/" + subscriptionID + "/resourceGroups/vm-rg/providers/Microsoft.Compute/virtualMachines/test-vm"),
				},
			},
		},
	}
}

// createAzureAvailabilitySetWithoutLinks creates a mock Availability Set without any linked resources
func createAzureAvailabilitySetWithoutLinks(avSetName string) *armcompute.AvailabilitySet {
	return &armcompute.AvailabilitySet{
		Name:     to.Ptr(avSetName),
		Location: to.Ptr("eastus"),
		Tags: map[string]*string{
			"env": to.Ptr("test"),
		},
		Properties: &armcompute.AvailabilitySetProperties{
			PlatformFaultDomainCount:  to.Ptr(int32(2)),
			PlatformUpdateDomainCount: to.Ptr(int32(5)),
			// No ProximityPlacementGroup
			// No VirtualMachines
		},
	}
}

// mockAvailabilitySetsPager is a simple mock implementation of the Pager interface for testing
type mockAvailabilitySetsPager struct {
	ctrl  *gomock.Controller
	items []*armcompute.AvailabilitySet
	index int
	more  bool
}

func newMockAvailabilitySetsPager(ctrl *gomock.Controller, items []*armcompute.AvailabilitySet) clients.AvailabilitySetsPager {
	return &mockAvailabilitySetsPager{
		ctrl:  ctrl,
		items: items,
		index: 0,
		more:  len(items) > 0,
	}
}

func (m *mockAvailabilitySetsPager) More() bool {
	return m.more
}

func (m *mockAvailabilitySetsPager) NextPage(ctx context.Context) (armcompute.AvailabilitySetsClientListResponse, error) {
	if m.index >= len(m.items) {
		m.more = false
		return armcompute.AvailabilitySetsClientListResponse{
			AvailabilitySetListResult: armcompute.AvailabilitySetListResult{
				Value: []*armcompute.AvailabilitySet{},
			},
		}, nil
	}

	item := m.items[m.index]
	m.index++
	m.more = m.index < len(m.items)

	return armcompute.AvailabilitySetsClientListResponse{
		AvailabilitySetListResult: armcompute.AvailabilitySetListResult{
			Value: []*armcompute.AvailabilitySet{item},
		},
	}, nil
}
