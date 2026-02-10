package manual_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"
	"go.uber.org/mock/gomock"

	"github.com/overmindtech/workspace/discovery"
	"github.com/overmindtech/workspace/sdp-go"
	"github.com/overmindtech/workspace/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	"github.com/overmindtech/cli/sources/azure/manual"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/azure/shared/mocks"
	"github.com/overmindtech/cli/sources/shared"
)

func TestComputeDiskAccess(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"
	scope := subscriptionID + "." + resourceGroup

	t.Run("Get", func(t *testing.T) {
		diskAccessName := "test-disk-access"
		diskAccess := createAzureDiskAccess(diskAccessName)

		mockClient := mocks.NewMockDiskAccessesClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, diskAccessName, nil).Return(
			armcompute.DiskAccessesClientGetResponse{
				DiskAccess: *diskAccess,
			}, nil)

		wrapper := manual.NewComputeDiskAccess(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, scope, diskAccessName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.ComputeDiskAccess.String() {
			t.Errorf("Expected type %s, got %s", azureshared.ComputeDiskAccess.String(), sdpItem.GetType())
		}

		if sdpItem.GetUniqueAttribute() != "name" {
			t.Errorf("Expected unique attribute 'name', got %s", sdpItem.GetUniqueAttribute())
		}

		if sdpItem.UniqueAttributeValue() != diskAccessName {
			t.Errorf("Expected unique attribute value %s, got %s", diskAccessName, sdpItem.UniqueAttributeValue())
		}

		if sdpItem.GetTags()["env"] != "test" {
			t.Errorf("Expected tag 'env=test', got: %v", sdpItem.GetTags()["env"])
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					// Child resource: Private Endpoint Connections
					ExpectedType:   azureshared.ComputeDiskAccessPrivateEndpointConnection.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  diskAccessName,
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

	t.Run("GetWithPrivateEndpointConnections", func(t *testing.T) {
		diskAccessName := "test-disk-access-with-pe"
		diskAccess := createAzureDiskAccessWithPrivateEndpointConnections(diskAccessName, subscriptionID, resourceGroup)

		mockClient := mocks.NewMockDiskAccessesClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, diskAccessName, nil).Return(
			armcompute.DiskAccessesClientGetResponse{
				DiskAccess: *diskAccess,
			}, nil)

		wrapper := manual.NewComputeDiskAccess(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, scope, diskAccessName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					ExpectedType:   azureshared.ComputeDiskAccessPrivateEndpointConnection.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  diskAccessName,
					ExpectedScope:  scope,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					// Network Private Endpoint (same resource group)
					ExpectedType:   azureshared.NetworkPrivateEndpoint.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-private-endpoint",
					ExpectedScope:  scope,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					// Network Private Endpoint (different resource group)
					ExpectedType:   azureshared.NetworkPrivateEndpoint.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-private-endpoint-other-rg",
					ExpectedScope:  subscriptionID + ".other-rg",
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
		mockClient := mocks.NewMockDiskAccessesClient(ctrl)

		wrapper := manual.NewComputeDiskAccess(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		_, qErr := wrapper.Get(ctx, scope)
		if qErr == nil {
			t.Error("Expected error when getting with no query parts, but got nil")
		}
	})

	t.Run("Get_EmptyName", func(t *testing.T) {
		mockClient := mocks.NewMockDiskAccessesClient(ctrl)

		wrapper := manual.NewComputeDiskAccess(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, scope, "", true)
		if qErr == nil {
			t.Error("Expected error when getting with empty name, but got nil")
		}
	})

	t.Run("Get_ClientError", func(t *testing.T) {
		expectedErr := errors.New("disk access not found")
		mockClient := mocks.NewMockDiskAccessesClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, "nonexistent", nil).Return(
			armcompute.DiskAccessesClientGetResponse{}, expectedErr)

		wrapper := manual.NewComputeDiskAccess(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, scope, "nonexistent", true)
		if qErr == nil {
			t.Error("Expected error when client returns error, but got nil")
		}
	})

	t.Run("List", func(t *testing.T) {
		diskAccess1 := createAzureDiskAccess("test-disk-access-1")
		diskAccess2 := createAzureDiskAccess("test-disk-access-2")

		mockClient := mocks.NewMockDiskAccessesClient(ctrl)
		mockPager := newMockDiskAccessesPager(ctrl, []*armcompute.DiskAccess{diskAccess1, diskAccess2})
		mockClient.EXPECT().NewListByResourceGroupPager(resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewComputeDiskAccess(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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
		diskAccess1 := createAzureDiskAccess("test-disk-access-1")
		diskAccess2 := createAzureDiskAccess("test-disk-access-2")

		mockClient := mocks.NewMockDiskAccessesClient(ctrl)
		mockPager := newMockDiskAccessesPager(ctrl, []*armcompute.DiskAccess{diskAccess1, diskAccess2})
		mockClient.EXPECT().NewListByResourceGroupPager(resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewComputeDiskAccess(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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
		diskAccess1 := createAzureDiskAccess("test-disk-access-1")
		diskAccessNilName := &armcompute.DiskAccess{
			Name:     nil,
			Location: to.Ptr("eastus"),
			Tags: map[string]*string{
				"env": to.Ptr("test"),
			},
		}

		mockClient := mocks.NewMockDiskAccessesClient(ctrl)
		mockPager := newMockDiskAccessesPager(ctrl, []*armcompute.DiskAccess{diskAccess1, diskAccessNilName})
		mockClient.EXPECT().NewListByResourceGroupPager(resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewComputeDiskAccess(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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
		mockClient := mocks.NewMockDiskAccessesClient(ctrl)
		errorPager := newErrorDiskAccessesPager(ctrl)
		mockClient.EXPECT().NewListByResourceGroupPager(resourceGroup, nil).Return(errorPager)

		wrapper := manual.NewComputeDiskAccess(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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
		mockClient := mocks.NewMockDiskAccessesClient(ctrl)
		errorPager := newErrorDiskAccessesPager(ctrl)
		mockClient.EXPECT().NewListByResourceGroupPager(resourceGroup, nil).Return(errorPager)

		wrapper := manual.NewComputeDiskAccess(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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

// createAzureDiskAccess creates a mock Azure Disk Access for testing.
func createAzureDiskAccess(diskAccessName string) *armcompute.DiskAccess {
	return &armcompute.DiskAccess{
		Name:     to.Ptr(diskAccessName),
		Location: to.Ptr("eastus"),
		Tags: map[string]*string{
			"env":     to.Ptr("test"),
			"project": to.Ptr("testing"),
		},
		Properties: &armcompute.DiskAccessProperties{
			ProvisioningState: to.Ptr("Succeeded"),
		},
	}
}

// createAzureDiskAccessWithPrivateEndpointConnections creates a mock Azure Disk Access with private endpoint connections.
func createAzureDiskAccessWithPrivateEndpointConnections(diskAccessName, subscriptionID, resourceGroup string) *armcompute.DiskAccess {
	return &armcompute.DiskAccess{
		Name:     to.Ptr(diskAccessName),
		Location: to.Ptr("eastus"),
		Tags: map[string]*string{
			"env": to.Ptr("test"),
		},
		Properties: &armcompute.DiskAccessProperties{
			ProvisioningState: to.Ptr("Succeeded"),
			PrivateEndpointConnections: []*armcompute.PrivateEndpointConnection{
				{
					Name: to.Ptr("pe-connection-1"),
					Properties: &armcompute.PrivateEndpointConnectionProperties{
						PrivateEndpoint: &armcompute.PrivateEndpoint{
							ID: to.Ptr("/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/privateEndpoints/test-private-endpoint"),
						},
					},
				},
				{
					Name: to.Ptr("pe-connection-2"),
					Properties: &armcompute.PrivateEndpointConnectionProperties{
						PrivateEndpoint: &armcompute.PrivateEndpoint{
							ID: to.Ptr("/subscriptions/" + subscriptionID + "/resourceGroups/other-rg/providers/Microsoft.Network/privateEndpoints/test-private-endpoint-other-rg"),
						},
					},
				},
			},
		},
	}
}

// mockDiskAccessesPager is a mock pager for DiskAccessesClientListByResourceGroupResponse.
type mockDiskAccessesPager struct {
	ctrl  *gomock.Controller
	items []*armcompute.DiskAccess
	index int
	more  bool
}

func newMockDiskAccessesPager(ctrl *gomock.Controller, items []*armcompute.DiskAccess) clients.DiskAccessesPager {
	return &mockDiskAccessesPager{
		ctrl:  ctrl,
		items: items,
		index: 0,
		more:  len(items) > 0,
	}
}

func (m *mockDiskAccessesPager) More() bool {
	return m.more
}

func (m *mockDiskAccessesPager) NextPage(ctx context.Context) (armcompute.DiskAccessesClientListByResourceGroupResponse, error) {
	if m.index >= len(m.items) {
		m.more = false
		return armcompute.DiskAccessesClientListByResourceGroupResponse{
			DiskAccessList: armcompute.DiskAccessList{
				Value: []*armcompute.DiskAccess{},
			},
		}, nil
	}

	item := m.items[m.index]
	m.index++
	m.more = m.index < len(m.items)

	return armcompute.DiskAccessesClientListByResourceGroupResponse{
		DiskAccessList: armcompute.DiskAccessList{
			Value: []*armcompute.DiskAccess{item},
		},
	}, nil
}

// errorDiskAccessesPager is a mock pager that always returns an error.
type errorDiskAccessesPager struct {
	ctrl *gomock.Controller
}

func newErrorDiskAccessesPager(ctrl *gomock.Controller) clients.DiskAccessesPager {
	return &errorDiskAccessesPager{ctrl: ctrl}
}

func (e *errorDiskAccessesPager) More() bool {
	return true
}

func (e *errorDiskAccessesPager) NextPage(ctx context.Context) (armcompute.DiskAccessesClientListByResourceGroupResponse, error) {
	return armcompute.DiskAccessesClientListByResourceGroupResponse{}, errors.New("pager error")
}
