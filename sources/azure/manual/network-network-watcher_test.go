package manual_test

import (
	"context"
	"errors"
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

func TestNetworkNetworkWatcher(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"

	t.Run("Get", func(t *testing.T) {
		resourceName := "test-network-watcher"
		resource := createNetworkWatcher(resourceName)

		mockClient := mocks.NewMockNetworkWatchersClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, resourceName, nil).Return(
			armnetwork.WatchersClientGetResponse{
				Watcher: *resource,
			}, nil)

		wrapper := manual.NewNetworkNetworkWatcher(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], resourceName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.NetworkNetworkWatcher.String() {
			t.Errorf("Expected type %s, got %s", azureshared.NetworkNetworkWatcher, sdpItem.GetType())
		}

		if sdpItem.GetUniqueAttribute() != "name" {
			t.Errorf("Expected unique attribute 'name', got %s", sdpItem.GetUniqueAttribute())
		}

		if sdpItem.UniqueAttributeValue() != resourceName {
			t.Errorf("Expected unique attribute value %s, got %s", resourceName, sdpItem.UniqueAttributeValue())
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					ExpectedType:   azureshared.NetworkFlowLog.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  resourceName,
					ExpectedScope:  subscriptionID + "." + resourceGroup,
				},
			}

			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("Get_ProvisioningStateSucceeded", func(t *testing.T) {
		resourceName := "test-network-watcher-succeeded"
		resource := createNetworkWatcherWithProvisioningState(resourceName, armnetwork.ProvisioningStateSucceeded)

		mockClient := mocks.NewMockNetworkWatchersClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, resourceName, nil).Return(
			armnetwork.WatchersClientGetResponse{
				Watcher: *resource,
			}, nil)

		wrapper := manual.NewNetworkNetworkWatcher(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], resourceName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetHealth() != sdp.Health_HEALTH_OK {
			t.Errorf("Expected health HEALTH_OK, got %s", sdpItem.GetHealth())
		}
	})

	t.Run("Get_ProvisioningStateFailed", func(t *testing.T) {
		resourceName := "test-network-watcher-failed"
		resource := createNetworkWatcherWithProvisioningState(resourceName, armnetwork.ProvisioningStateFailed)

		mockClient := mocks.NewMockNetworkWatchersClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, resourceName, nil).Return(
			armnetwork.WatchersClientGetResponse{
				Watcher: *resource,
			}, nil)

		wrapper := manual.NewNetworkNetworkWatcher(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], resourceName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetHealth() != sdp.Health_HEALTH_ERROR {
			t.Errorf("Expected health HEALTH_ERROR, got %s", sdpItem.GetHealth())
		}
	})

	t.Run("List", func(t *testing.T) {
		resource1 := createNetworkWatcher("test-network-watcher-1")
		resource2 := createNetworkWatcher("test-network-watcher-2")

		mockClient := mocks.NewMockNetworkWatchersClient(ctrl)
		mockPager := newMockNetworkWatchersPager(ctrl, []*armnetwork.Watcher{resource1, resource2})

		mockClient.EXPECT().NewListPager(resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewNetworkNetworkWatcher(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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
		}
	})

	t.Run("List_SkipNilName", func(t *testing.T) {
		resource1 := createNetworkWatcher("test-network-watcher-1")
		resource2 := &armnetwork.Watcher{
			Name: nil, // nil name should be skipped
		}

		mockClient := mocks.NewMockNetworkWatchersClient(ctrl)
		mockPager := newMockNetworkWatchersPager(ctrl, []*armnetwork.Watcher{resource1, resource2})

		mockClient.EXPECT().NewListPager(resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewNetworkNetworkWatcher(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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
			t.Fatalf("Expected 1 item (skipping nil name), got: %d", len(sdpItems))
		}
	})

	t.Run("ListStream", func(t *testing.T) {
		resource1 := createNetworkWatcher("test-network-watcher-1")
		resource2 := createNetworkWatcher("test-network-watcher-2")

		mockClient := mocks.NewMockNetworkWatchersClient(ctrl)
		mockPager := newMockNetworkWatchersPager(ctrl, []*armnetwork.Watcher{resource1, resource2})

		mockClient.EXPECT().NewListPager(resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewNetworkNetworkWatcher(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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
		expectedErr := errors.New("resource not found")

		mockClient := mocks.NewMockNetworkWatchersClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, "nonexistent", nil).Return(
			armnetwork.WatchersClientGetResponse{}, expectedErr)

		wrapper := manual.NewNetworkNetworkWatcher(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "nonexistent", true)
		if qErr == nil {
			t.Error("Expected error when getting non-existent resource, but got nil")
		}
	})

	t.Run("GetWithEmptyName", func(t *testing.T) {
		mockClient := mocks.NewMockNetworkWatchersClient(ctrl)

		wrapper := manual.NewNetworkNetworkWatcher(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "", true)
		if qErr == nil {
			t.Error("Expected error when getting resource with empty name, but got nil")
		}
	})
}

func createNetworkWatcher(name string) *armnetwork.Watcher {
	provisioningState := armnetwork.ProvisioningStateSucceeded
	return &armnetwork.Watcher{
		ID:       new(string),
		Name:     &name,
		Type:     new(string),
		Location: new(string),
		Tags: map[string]*string{
			"env": new(string),
		},
		Properties: &armnetwork.WatcherPropertiesFormat{
			ProvisioningState: &provisioningState,
		},
	}
}

func createNetworkWatcherWithProvisioningState(name string, state armnetwork.ProvisioningState) *armnetwork.Watcher {
	return &armnetwork.Watcher{
		ID:       new(string),
		Name:     &name,
		Type:     new(string),
		Location: new(string),
		Tags: map[string]*string{
			"env": new(string),
		},
		Properties: &armnetwork.WatcherPropertiesFormat{
			ProvisioningState: &state,
		},
	}
}

type mockNetworkWatchersPager struct {
	ctrl  *gomock.Controller
	items []*armnetwork.Watcher
	index int
	more  bool
}

func newMockNetworkWatchersPager(ctrl *gomock.Controller, items []*armnetwork.Watcher) clients.NetworkWatchersPager {
	return &mockNetworkWatchersPager{
		ctrl:  ctrl,
		items: items,
		index: 0,
		more:  len(items) > 0,
	}
}

func (m *mockNetworkWatchersPager) More() bool {
	return m.more
}

func (m *mockNetworkWatchersPager) NextPage(ctx context.Context) (armnetwork.WatchersClientListResponse, error) {
	if m.index >= len(m.items) {
		m.more = false
		return armnetwork.WatchersClientListResponse{
			WatcherListResult: armnetwork.WatcherListResult{
				Value: []*armnetwork.Watcher{},
			},
		}, nil
	}

	item := m.items[m.index]
	m.index++
	m.more = m.index < len(m.items)

	return armnetwork.WatchersClientListResponse{
		WatcherListResult: armnetwork.WatcherListResult{
			Value: []*armnetwork.Watcher{item},
		},
	}, nil
}
