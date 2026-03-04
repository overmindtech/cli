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

func TestNetworkApplicationSecurityGroup(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"

	t.Run("Get", func(t *testing.T) {
		asgName := "test-asg"
		asg := createAzureApplicationSecurityGroup(asgName)

		mockClient := mocks.NewMockApplicationSecurityGroupsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, asgName, nil).Return(
			armnetwork.ApplicationSecurityGroupsClientGetResponse{
				ApplicationSecurityGroup: *asg,
			}, nil)

		wrapper := manual.NewNetworkApplicationSecurityGroup(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], asgName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.NetworkApplicationSecurityGroup.String() {
			t.Errorf("Expected type %s, got %s", azureshared.NetworkApplicationSecurityGroup, sdpItem.GetType())
		}

		if sdpItem.GetUniqueAttribute() != "name" {
			t.Errorf("Expected unique attribute 'name', got %s", sdpItem.GetUniqueAttribute())
		}

		if sdpItem.UniqueAttributeValue() != asgName {
			t.Errorf("Expected unique attribute value %s, got %s", asgName, sdpItem.UniqueAttributeValue())
		}

		if sdpItem.GetTags()["env"] != "test" {
			t.Errorf("Expected tag 'env=test', got: %v", sdpItem.GetTags()["env"])
		}

		t.Run("StaticTests", func(t *testing.T) {
			// Application Security Group has no linked item queries
			queryTests := shared.QueryTests{}
			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("GetWithEmptyName", func(t *testing.T) {
		mockClient := mocks.NewMockApplicationSecurityGroupsClient(ctrl)

		wrapper := manual.NewNetworkApplicationSecurityGroup(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "", true)
		if qErr == nil {
			t.Error("Expected error when application security group name is empty, but got nil")
		}
	})

	t.Run("Get_ASGWithNilName", func(t *testing.T) {
		provisioningState := armnetwork.ProvisioningStateSucceeded
		asgWithNilName := &armnetwork.ApplicationSecurityGroup{
			Name:     nil,
			Location: new("eastus"),
			Properties: &armnetwork.ApplicationSecurityGroupPropertiesFormat{
				ProvisioningState: &provisioningState,
			},
		}

		mockClient := mocks.NewMockApplicationSecurityGroupsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, "test-asg", nil).Return(
			armnetwork.ApplicationSecurityGroupsClientGetResponse{
				ApplicationSecurityGroup: *asgWithNilName,
			}, nil)

		wrapper := manual.NewNetworkApplicationSecurityGroup(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "test-asg", true)
		if qErr == nil {
			t.Error("Expected error when application security group has nil name, but got nil")
		}
	})

	t.Run("List", func(t *testing.T) {
		asg1 := createAzureApplicationSecurityGroup("asg-1")
		asg2 := createAzureApplicationSecurityGroup("asg-2")

		mockClient := mocks.NewMockApplicationSecurityGroupsClient(ctrl)
		mockPager := newMockApplicationSecurityGroupsPager(ctrl, []*armnetwork.ApplicationSecurityGroup{asg1, asg2})

		mockClient.EXPECT().NewListPager(resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewNetworkApplicationSecurityGroup(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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
			if item.GetType() != azureshared.NetworkApplicationSecurityGroup.String() {
				t.Fatalf("Expected type %s, got: %s", azureshared.NetworkApplicationSecurityGroup, item.GetType())
			}
		}
	})

	t.Run("List_WithNilName", func(t *testing.T) {
		asg1 := createAzureApplicationSecurityGroup("asg-1")
		provisioningState := armnetwork.ProvisioningStateSucceeded
		asg2NilName := &armnetwork.ApplicationSecurityGroup{
			Name:     nil,
			Location: new("eastus"),
			Tags:     map[string]*string{"env": new("test")},
			Properties: &armnetwork.ApplicationSecurityGroupPropertiesFormat{
				ProvisioningState: &provisioningState,
			},
		}

		mockClient := mocks.NewMockApplicationSecurityGroupsClient(ctrl)
		mockPager := newMockApplicationSecurityGroupsPager(ctrl, []*armnetwork.ApplicationSecurityGroup{asg1, asg2NilName})

		mockClient.EXPECT().NewListPager(resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewNetworkApplicationSecurityGroup(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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
		if sdpItems[0].UniqueAttributeValue() != "asg-1" {
			t.Errorf("Expected item name 'asg-1', got: %s", sdpItems[0].UniqueAttributeValue())
		}
	})

	t.Run("ListStream", func(t *testing.T) {
		asg1 := createAzureApplicationSecurityGroup("stream-asg-1")
		asg2 := createAzureApplicationSecurityGroup("stream-asg-2")

		mockClient := mocks.NewMockApplicationSecurityGroupsClient(ctrl)
		mockPager := newMockApplicationSecurityGroupsPager(ctrl, []*armnetwork.ApplicationSecurityGroup{asg1, asg2})

		mockClient.EXPECT().NewListPager(resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewNetworkApplicationSecurityGroup(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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
		expectedErr := errors.New("application security group not found")

		mockClient := mocks.NewMockApplicationSecurityGroupsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, "nonexistent-asg", nil).Return(
			armnetwork.ApplicationSecurityGroupsClientGetResponse{}, expectedErr)

		wrapper := manual.NewNetworkApplicationSecurityGroup(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "nonexistent-asg", true)
		if qErr == nil {
			t.Error("Expected error when getting non-existent application security group, but got nil")
		}
	})

	t.Run("InterfaceCompliance", func(t *testing.T) {
		mockClient := mocks.NewMockApplicationSecurityGroupsClient(ctrl)
		wrapper := manual.NewNetworkApplicationSecurityGroup(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		w := wrapper.(sources.Wrapper)

		permissions := w.IAMPermissions()
		if len(permissions) == 0 {
			t.Error("Expected IAMPermissions to return at least one permission")
		}
		expectedPermission := "Microsoft.Network/applicationSecurityGroups/read"
		if !slices.Contains(permissions, expectedPermission) {
			t.Errorf("Expected IAMPermissions to include %s", expectedPermission)
		}

		mappings := w.TerraformMappings()
		foundMapping := false
		for _, mapping := range mappings {
			if mapping.GetTerraformQueryMap() == "azurerm_application_security_group.name" {
				foundMapping = true
				if mapping.GetTerraformMethod() != sdp.QueryMethod_GET {
					t.Errorf("Expected TerraformMethod GET, got: %s", mapping.GetTerraformMethod())
				}
				break
			}
		}
		if !foundMapping {
			t.Error("Expected TerraformMappings to include 'azurerm_application_security_group.name'")
		}

		lookups := w.GetLookups()
		foundLookup := false
		for _, lookup := range lookups {
			if lookup.ItemType == azureshared.NetworkApplicationSecurityGroup {
				foundLookup = true
				break
			}
		}
		if !foundLookup {
			t.Error("Expected GetLookups to include NetworkApplicationSecurityGroup")
		}
	})
}

type mockApplicationSecurityGroupsPager struct {
	ctrl  *gomock.Controller
	items []*armnetwork.ApplicationSecurityGroup
	index int
	more  bool
}

func newMockApplicationSecurityGroupsPager(ctrl *gomock.Controller, items []*armnetwork.ApplicationSecurityGroup) clients.ApplicationSecurityGroupsPager {
	return &mockApplicationSecurityGroupsPager{
		ctrl:  ctrl,
		items: items,
		index: 0,
		more:  len(items) > 0,
	}
}

func (m *mockApplicationSecurityGroupsPager) More() bool {
	return m.more
}

func (m *mockApplicationSecurityGroupsPager) NextPage(ctx context.Context) (armnetwork.ApplicationSecurityGroupsClientListResponse, error) {
	if m.index >= len(m.items) {
		m.more = false
		return armnetwork.ApplicationSecurityGroupsClientListResponse{
			ApplicationSecurityGroupListResult: armnetwork.ApplicationSecurityGroupListResult{
				Value: []*armnetwork.ApplicationSecurityGroup{},
			},
		}, nil
	}
	item := m.items[m.index]
	m.index++
	m.more = m.index < len(m.items)
	return armnetwork.ApplicationSecurityGroupsClientListResponse{
		ApplicationSecurityGroupListResult: armnetwork.ApplicationSecurityGroupListResult{
			Value: []*armnetwork.ApplicationSecurityGroup{item},
		},
	}, nil
}

func createAzureApplicationSecurityGroup(name string) *armnetwork.ApplicationSecurityGroup {
	provisioningState := armnetwork.ProvisioningStateSucceeded
	return &armnetwork.ApplicationSecurityGroup{
		ID:       new("/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.Network/applicationSecurityGroups/" + name),
		Name:     new(name),
		Type:     new("Microsoft.Network/applicationSecurityGroups"),
		Location: new("eastus"),
		Tags: map[string]*string{
			"env":     new("test"),
			"project": new("testing"),
		},
		Properties: &armnetwork.ApplicationSecurityGroupPropertiesFormat{
			ProvisioningState: &provisioningState,
			ResourceGUID:      new("00000000-0000-0000-0000-000000000001"),
		},
	}
}

// Ensure mockApplicationSecurityGroupsPager satisfies the pager interface at compile time.
var _ clients.ApplicationSecurityGroupsPager = (*mockApplicationSecurityGroupsPager)(nil)
