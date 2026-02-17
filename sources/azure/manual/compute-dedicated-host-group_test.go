package manual_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
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

func TestComputeDedicatedHostGroup(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"
	scope := subscriptionID + "." + resourceGroup

	t.Run("Get", func(t *testing.T) {
		hostGroupName := "test-host-group"
		dedicatedHostGroup := createAzureDedicatedHostGroup(hostGroupName)

		mockClient := mocks.NewMockDedicatedHostGroupsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, hostGroupName, nil).Return(
			armcompute.DedicatedHostGroupsClientGetResponse{
				DedicatedHostGroup: *dedicatedHostGroup,
			}, nil)

		wrapper := manual.NewComputeDedicatedHostGroup(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, scope, hostGroupName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.ComputeDedicatedHostGroup.String() {
			t.Errorf("Expected type %s, got %s", azureshared.ComputeDedicatedHostGroup.String(), sdpItem.GetType())
		}

		if sdpItem.GetUniqueAttribute() != "name" {
			t.Errorf("Expected unique attribute 'name', got %s", sdpItem.GetUniqueAttribute())
		}

		if sdpItem.UniqueAttributeValue() != hostGroupName {
			t.Errorf("Expected unique attribute value %s, got %s", hostGroupName, sdpItem.UniqueAttributeValue())
		}

		if sdpItem.GetTags()["env"] != "test" {
			t.Errorf("Expected tag 'env=test', got: %v", sdpItem.GetTags()["env"])
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{}
			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("GetWithHosts", func(t *testing.T) {
		hostGroupName := "test-host-group-with-hosts"
		dedicatedHostGroup := createAzureDedicatedHostGroupWithHosts(hostGroupName, subscriptionID, resourceGroup, "host-1", "host-2")

		mockClient := mocks.NewMockDedicatedHostGroupsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, hostGroupName, nil).Return(
			armcompute.DedicatedHostGroupsClientGetResponse{
				DedicatedHostGroup: *dedicatedHostGroup,
			}, nil)

		wrapper := manual.NewComputeDedicatedHostGroup(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, scope, hostGroupName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					ExpectedType:   azureshared.ComputeDedicatedHost.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey(hostGroupName, "host-1"),
					ExpectedScope:  scope,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					ExpectedType:   azureshared.ComputeDedicatedHost.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey(hostGroupName, "host-2"),
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

	t.Run("Get_InvalidQueryParts", func(t *testing.T) {
		mockClient := mocks.NewMockDedicatedHostGroupsClient(ctrl)

		wrapper := manual.NewComputeDedicatedHostGroup(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		_, qErr := wrapper.Get(ctx, scope)
		if qErr == nil {
			t.Error("Expected error when getting with no query parts, but got nil")
		}
	})

	t.Run("Get_EmptyName", func(t *testing.T) {
		mockClient := mocks.NewMockDedicatedHostGroupsClient(ctrl)

		wrapper := manual.NewComputeDedicatedHostGroup(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, scope, "", true)
		if qErr == nil {
			t.Error("Expected error when getting with empty name, but got nil")
		}
	})

	t.Run("Get_ClientError", func(t *testing.T) {
		expectedErr := errors.New("dedicated host group not found")
		mockClient := mocks.NewMockDedicatedHostGroupsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, "nonexistent", nil).Return(
			armcompute.DedicatedHostGroupsClientGetResponse{}, expectedErr)

		wrapper := manual.NewComputeDedicatedHostGroup(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, scope, "nonexistent", true)
		if qErr == nil {
			t.Error("Expected error when client returns error, but got nil")
		}
	})

	t.Run("List", func(t *testing.T) {
		hostGroup1 := createAzureDedicatedHostGroup("test-host-group-1")
		hostGroup2 := createAzureDedicatedHostGroup("test-host-group-2")

		mockClient := mocks.NewMockDedicatedHostGroupsClient(ctrl)
		mockPager := newMockDedicatedHostGroupsPager(ctrl, []*armcompute.DedicatedHostGroup{hostGroup1, hostGroup2})
		mockClient.EXPECT().NewListByResourceGroupPager(resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewComputeDedicatedHostGroup(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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
		hostGroup1 := createAzureDedicatedHostGroup("test-host-group-1")
		hostGroup2 := createAzureDedicatedHostGroup("test-host-group-2")

		mockClient := mocks.NewMockDedicatedHostGroupsClient(ctrl)
		mockPager := newMockDedicatedHostGroupsPager(ctrl, []*armcompute.DedicatedHostGroup{hostGroup1, hostGroup2})
		mockClient.EXPECT().NewListByResourceGroupPager(resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewComputeDedicatedHostGroup(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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
		hostGroup1 := createAzureDedicatedHostGroup("test-host-group-1")
		hostGroupNilName := &armcompute.DedicatedHostGroup{
			Name:     nil,
			Location: to.Ptr("eastus"),
			Tags: map[string]*string{
				"env": to.Ptr("test"),
			},
			Properties: &armcompute.DedicatedHostGroupProperties{
				PlatformFaultDomainCount: to.Ptr(int32(2)),
			},
		}

		mockClient := mocks.NewMockDedicatedHostGroupsClient(ctrl)
		mockPager := newMockDedicatedHostGroupsPager(ctrl, []*armcompute.DedicatedHostGroup{hostGroup1, hostGroupNilName})
		mockClient.EXPECT().NewListByResourceGroupPager(resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewComputeDedicatedHostGroup(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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
		mockClient := mocks.NewMockDedicatedHostGroupsClient(ctrl)
		errorPager := newErrorDedicatedHostGroupsPager(ctrl)
		mockClient.EXPECT().NewListByResourceGroupPager(resourceGroup, nil).Return(errorPager)

		wrapper := manual.NewComputeDedicatedHostGroup(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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
		mockClient := mocks.NewMockDedicatedHostGroupsClient(ctrl)
		errorPager := newErrorDedicatedHostGroupsPager(ctrl)
		mockClient.EXPECT().NewListByResourceGroupPager(resourceGroup, nil).Return(errorPager)

		wrapper := manual.NewComputeDedicatedHostGroup(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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

// createAzureDedicatedHostGroup creates a mock Azure Dedicated Host Group for testing.
func createAzureDedicatedHostGroup(hostGroupName string) *armcompute.DedicatedHostGroup {
	return &armcompute.DedicatedHostGroup{
		Name:     to.Ptr(hostGroupName),
		Location: to.Ptr("eastus"),
		Tags: map[string]*string{
			"env":     to.Ptr("test"),
			"project": to.Ptr("testing"),
		},
		Properties: &armcompute.DedicatedHostGroupProperties{
			PlatformFaultDomainCount:   to.Ptr(int32(2)),
			SupportAutomaticPlacement:   to.Ptr(false),
			AdditionalCapabilities:     nil,
			Hosts:                      nil,
			InstanceView:               nil,
		},
	}
}

// createAzureDedicatedHostGroupWithHosts creates a mock Azure Dedicated Host Group with host references.
func createAzureDedicatedHostGroupWithHosts(hostGroupName, subscriptionID, resourceGroup string, hostNames ...string) *armcompute.DedicatedHostGroup {
	hosts := make([]*armcompute.SubResourceReadOnly, 0, len(hostNames))
	for _, name := range hostNames {
		hosts = append(hosts, &armcompute.SubResourceReadOnly{
			ID: to.Ptr("/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Compute/hostGroups/" + hostGroupName + "/hosts/" + name),
		})
	}
	return &armcompute.DedicatedHostGroup{
		Name:     to.Ptr(hostGroupName),
		Location: to.Ptr("eastus"),
		Tags: map[string]*string{
			"env": to.Ptr("test"),
		},
		Properties: &armcompute.DedicatedHostGroupProperties{
			PlatformFaultDomainCount: to.Ptr(int32(2)),
			Hosts:                    hosts,
		},
	}
}

// mockDedicatedHostGroupsPager is a mock pager for DedicatedHostGroupsClientListByResourceGroupResponse.
type mockDedicatedHostGroupsPager struct {
	ctrl  *gomock.Controller
	items []*armcompute.DedicatedHostGroup
	index int
	more  bool
}

func newMockDedicatedHostGroupsPager(ctrl *gomock.Controller, items []*armcompute.DedicatedHostGroup) clients.DedicatedHostGroupsPager {
	return &mockDedicatedHostGroupsPager{
		ctrl:  ctrl,
		items: items,
		index: 0,
		more:  len(items) > 0,
	}
}

func (m *mockDedicatedHostGroupsPager) More() bool {
	return m.more
}

func (m *mockDedicatedHostGroupsPager) NextPage(ctx context.Context) (armcompute.DedicatedHostGroupsClientListByResourceGroupResponse, error) {
	if m.index >= len(m.items) {
		m.more = false
		return armcompute.DedicatedHostGroupsClientListByResourceGroupResponse{
			DedicatedHostGroupListResult: armcompute.DedicatedHostGroupListResult{
				Value: []*armcompute.DedicatedHostGroup{},
			},
		}, nil
	}

	item := m.items[m.index]
	m.index++
	m.more = m.index < len(m.items)

	return armcompute.DedicatedHostGroupsClientListByResourceGroupResponse{
		DedicatedHostGroupListResult: armcompute.DedicatedHostGroupListResult{
			Value: []*armcompute.DedicatedHostGroup{item},
		},
	}, nil
}

// errorDedicatedHostGroupsPager is a mock pager that always returns an error.
type errorDedicatedHostGroupsPager struct {
	ctrl *gomock.Controller
}

func newErrorDedicatedHostGroupsPager(ctrl *gomock.Controller) clients.DedicatedHostGroupsPager {
	return &errorDedicatedHostGroupsPager{ctrl: ctrl}
}

func (e *errorDedicatedHostGroupsPager) More() bool {
	return true
}

func (e *errorDedicatedHostGroupsPager) NextPage(ctx context.Context) (armcompute.DedicatedHostGroupsClientListByResourceGroupResponse, error) {
	return armcompute.DedicatedHostGroupsClientListByResourceGroupResponse{}, errors.New("pager error")
}
