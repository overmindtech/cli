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

func createAzureDedicatedHost(hostName, hostGroupName string) *armcompute.DedicatedHost {
	return &armcompute.DedicatedHost{
		ID:       new("/subscriptions/sub/resourceGroups/rg/providers/Microsoft.Compute/hostGroups/" + hostGroupName + "/hosts/" + hostName),
		Name:     new(hostName),
		Type:     new("Microsoft.Compute/hostGroups/hosts"),
		Location: new("eastus"),
		Tags:     map[string]*string{"env": new("test")},
		SKU: &armcompute.SKU{
			Name: new("DSv3-Type1"),
		},
		Properties: &armcompute.DedicatedHostProperties{
			PlatformFaultDomain: new(int32(0)),
			ProvisioningState:   new("Succeeded"),
		},
	}
}

func createAzureDedicatedHostWithVMs(hostName, hostGroupName, subscriptionID, resourceGroup string, vmNames ...string) *armcompute.DedicatedHost {
	vms := make([]*armcompute.SubResourceReadOnly, 0, len(vmNames))
	for _, vmName := range vmNames {
		vms = append(vms, &armcompute.SubResourceReadOnly{
			ID: new("/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Compute/virtualMachines/" + vmName),
		})
	}
	return &armcompute.DedicatedHost{
		ID:       new("/subscriptions/sub/resourceGroups/rg/providers/Microsoft.Compute/hostGroups/" + hostGroupName + "/hosts/" + hostName),
		Name:     new(hostName),
		Type:     new("Microsoft.Compute/hostGroups/hosts"),
		Location: new("eastus"),
		Tags:     map[string]*string{"env": new("test")},
		SKU: &armcompute.SKU{
			Name: new("DSv3-Type1"),
		},
		Properties: &armcompute.DedicatedHostProperties{
			PlatformFaultDomain: new(int32(0)),
			ProvisioningState:   new("Succeeded"),
			VirtualMachines:     vms,
		},
	}
}

type mockDedicatedHostsPager struct {
	items []*armcompute.DedicatedHost
	index int
}

func (m *mockDedicatedHostsPager) More() bool {
	return m.index < len(m.items)
}

func (m *mockDedicatedHostsPager) NextPage(ctx context.Context) (armcompute.DedicatedHostsClientListByHostGroupResponse, error) {
	if m.index >= len(m.items) {
		return armcompute.DedicatedHostsClientListByHostGroupResponse{
			DedicatedHostListResult: armcompute.DedicatedHostListResult{
				Value: []*armcompute.DedicatedHost{},
			},
		}, nil
	}
	item := m.items[m.index]
	m.index++
	return armcompute.DedicatedHostsClientListByHostGroupResponse{
		DedicatedHostListResult: armcompute.DedicatedHostListResult{
			Value: []*armcompute.DedicatedHost{item},
		},
	}, nil
}

type errorDedicatedHostsPager struct{}

func (e *errorDedicatedHostsPager) More() bool {
	return true
}

func (e *errorDedicatedHostsPager) NextPage(ctx context.Context) (armcompute.DedicatedHostsClientListByHostGroupResponse, error) {
	return armcompute.DedicatedHostsClientListByHostGroupResponse{}, errors.New("pager error")
}

type testDedicatedHostsClient struct {
	*mocks.MockDedicatedHostsClient
	pager clients.DedicatedHostsPager
}

func (t *testDedicatedHostsClient) NewListByHostGroupPager(resourceGroupName string, hostGroupName string, options *armcompute.DedicatedHostsClientListByHostGroupOptions) clients.DedicatedHostsPager {
	if t.pager != nil {
		return t.pager
	}
	return t.MockDedicatedHostsClient.NewListByHostGroupPager(resourceGroupName, hostGroupName, options)
}

func TestComputeDedicatedHost(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"
	scope := subscriptionID + "." + resourceGroup
	hostGroupName := "test-host-group"
	hostName := "test-host"

	t.Run("Get", func(t *testing.T) {
		host := createAzureDedicatedHost(hostName, hostGroupName)

		mockClient := mocks.NewMockDedicatedHostsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, hostGroupName, hostName, nil).Return(
			armcompute.DedicatedHostsClientGetResponse{
				DedicatedHost: *host,
			}, nil)

		wrapper := NewComputeDedicatedHost(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(hostGroupName, hostName)
		sdpItem, qErr := adapter.Get(ctx, scope, query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.ComputeDedicatedHost.String() {
			t.Errorf("Expected type %s, got %s", azureshared.ComputeDedicatedHost.String(), sdpItem.GetType())
		}

		if sdpItem.GetUniqueAttribute() != "uniqueAttr" {
			t.Errorf("Expected unique attribute 'uniqueAttr', got %s", sdpItem.GetUniqueAttribute())
		}

		expectedUnique := shared.CompositeLookupKey(hostGroupName, hostName)
		if sdpItem.UniqueAttributeValue() != expectedUnique {
			t.Errorf("Expected unique attribute value %s, got %s", expectedUnique, sdpItem.UniqueAttributeValue())
		}

		if sdpItem.GetTags()["env"] != "test" {
			t.Errorf("Expected tag env=test, got: %v", sdpItem.GetTags()["env"])
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{ExpectedType: azureshared.ComputeDedicatedHostGroup.String(), ExpectedMethod: sdp.QueryMethod_GET, ExpectedQuery: hostGroupName, ExpectedScope: scope},
			}
			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("Get_WithVMLinks", func(t *testing.T) {
		host := createAzureDedicatedHostWithVMs(hostName, hostGroupName, subscriptionID, resourceGroup, "vm-1", "vm-2")

		mockClient := mocks.NewMockDedicatedHostsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, hostGroupName, hostName, nil).Return(
			armcompute.DedicatedHostsClientGetResponse{
				DedicatedHost: *host,
			}, nil)

		wrapper := NewComputeDedicatedHost(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(hostGroupName, hostName)
		sdpItem, qErr := adapter.Get(ctx, scope, query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		queryTests := shared.QueryTests{
			{ExpectedType: azureshared.ComputeDedicatedHostGroup.String(), ExpectedMethod: sdp.QueryMethod_GET, ExpectedQuery: hostGroupName, ExpectedScope: scope},
			{ExpectedType: azureshared.ComputeVirtualMachine.String(), ExpectedMethod: sdp.QueryMethod_GET, ExpectedQuery: "vm-1", ExpectedScope: scope},
			{ExpectedType: azureshared.ComputeVirtualMachine.String(), ExpectedMethod: sdp.QueryMethod_GET, ExpectedQuery: "vm-2", ExpectedScope: scope},
		}
		shared.RunStaticTests(t, adapter, sdpItem, queryTests)
	})

	t.Run("Get_InvalidQueryParts", func(t *testing.T) {
		mockClient := mocks.NewMockDedicatedHostsClient(ctrl)
		wrapper := NewComputeDedicatedHost(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, scope, hostGroupName, true)
		if qErr == nil {
			t.Error("Expected error when Get with wrong number of query parts, but got nil")
		}
	})

	t.Run("Get_EmptyHostGroupName", func(t *testing.T) {
		mockClient := mocks.NewMockDedicatedHostsClient(ctrl)
		wrapper := NewComputeDedicatedHost(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey("", hostName)
		_, qErr := adapter.Get(ctx, scope, query, true)
		if qErr == nil {
			t.Error("Expected error when host group name is empty, but got nil")
		}
	})

	t.Run("Get_EmptyHostName", func(t *testing.T) {
		mockClient := mocks.NewMockDedicatedHostsClient(ctrl)
		wrapper := NewComputeDedicatedHost(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(hostGroupName, "")
		_, qErr := adapter.Get(ctx, scope, query, true)
		if qErr == nil {
			t.Error("Expected error when host name is empty, but got nil")
		}
	})

	t.Run("Get_ClientError", func(t *testing.T) {
		expectedErr := errors.New("host not found")
		mockClient := mocks.NewMockDedicatedHostsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, hostGroupName, "nonexistent", nil).Return(
			armcompute.DedicatedHostsClientGetResponse{}, expectedErr)

		wrapper := NewComputeDedicatedHost(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(hostGroupName, "nonexistent")
		_, qErr := adapter.Get(ctx, scope, query, true)
		if qErr == nil {
			t.Error("Expected error when client returns error, but got nil")
		}
	})

	t.Run("Search", func(t *testing.T) {
		host1 := createAzureDedicatedHost("host-1", hostGroupName)
		host2 := createAzureDedicatedHost("host-2", hostGroupName)

		mockClient := mocks.NewMockDedicatedHostsClient(ctrl)
		pager := &mockDedicatedHostsPager{
			items: []*armcompute.DedicatedHost{host1, host2},
		}
		testClient := &testDedicatedHostsClient{
			MockDedicatedHostsClient: mockClient,
			pager:                     pager,
		}

		wrapper := NewComputeDedicatedHost(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support Search operation")
		}

		sdpItems, err := searchable.Search(ctx, scope, hostGroupName, true)
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
		mockClient := mocks.NewMockDedicatedHostsClient(ctrl)
		wrapper := NewComputeDedicatedHost(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		_, qErr := wrapper.Search(ctx, scope, hostGroupName, hostName)
		if qErr == nil {
			t.Error("Expected error when Search with wrong number of query parts, but got nil")
		}
	})

	t.Run("Search_EmptyHostGroupName", func(t *testing.T) {
		mockClient := mocks.NewMockDedicatedHostsClient(ctrl)
		wrapper := NewComputeDedicatedHost(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		_, qErr := wrapper.Search(ctx, scope, "")
		if qErr == nil {
			t.Error("Expected error when host group name is empty, but got nil")
		}
	})

	t.Run("Search_PagerError", func(t *testing.T) {
		mockClient := mocks.NewMockDedicatedHostsClient(ctrl)
		errorPager := &errorDedicatedHostsPager{}
		testClient := &testDedicatedHostsClient{
			MockDedicatedHostsClient: mockClient,
			pager:                    errorPager,
		}

		wrapper := NewComputeDedicatedHost(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support Search operation")
		}

		_, err := searchable.Search(ctx, scope, hostGroupName, true)
		if err == nil {
			t.Error("Expected error when pager returns error, but got nil")
		}
	})

	t.Run("PotentialLinks", func(t *testing.T) {
		mockClient := mocks.NewMockDedicatedHostsClient(ctrl)
		wrapper := NewComputeDedicatedHost(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		links := wrapper.PotentialLinks()
		expected := map[shared.ItemType]bool{
			azureshared.ComputeDedicatedHostGroup: true,
			azureshared.ComputeVirtualMachine:     true,
		}
		for itemType, want := range expected {
			if got := links[itemType]; got != want {
				t.Errorf("PotentialLinks()[%v] = %v, want %v", itemType, got, want)
			}
		}
	})

	t.Run("ImplementsSearchableAdapter", func(t *testing.T) {
		mockClient := mocks.NewMockDedicatedHostsClient(ctrl)
		wrapper := NewComputeDedicatedHost(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Error("Adapter should implement SearchableAdapter interface")
		}
	})
}
