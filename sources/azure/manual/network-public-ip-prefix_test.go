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
	"github.com/overmindtech/cli/sources/stdlib"
)

func TestNetworkPublicIPPrefix(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"

	t.Run("Get", func(t *testing.T) {
		prefixName := "test-prefix"
		prefix := createAzurePublicIPPrefix(prefixName)

		mockClient := mocks.NewMockPublicIPPrefixesClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, prefixName, nil).Return(
			armnetwork.PublicIPPrefixesClientGetResponse{
				PublicIPPrefix: *prefix,
			}, nil)

		wrapper := manual.NewNetworkPublicIPPrefix(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], prefixName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.NetworkPublicIPPrefix.String() {
			t.Errorf("Expected type %s, got %s", azureshared.NetworkPublicIPPrefix.String(), sdpItem.GetType())
		}

		if sdpItem.GetUniqueAttribute() != "name" {
			t.Errorf("Expected unique attribute 'name', got %s", sdpItem.GetUniqueAttribute())
		}

		if sdpItem.UniqueAttributeValue() != prefixName {
			t.Errorf("Expected unique attribute value %s, got %s", prefixName, sdpItem.UniqueAttributeValue())
		}

		if sdpItem.GetTags()["env"] != "test" {
			t.Errorf("Expected tag 'env=test', got: %v", sdpItem.GetTags()["env"])
		}

		t.Run("StaticTests", func(t *testing.T) {
			// Public IP prefix with no linked resources in base createAzurePublicIPPrefix
			queryTests := shared.QueryTests{}
			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("Get_WithLinkedResources", func(t *testing.T) {
		prefixName := "test-prefix-with-links"
		prefix := createAzurePublicIPPrefixWithLinks(prefixName, subscriptionID, resourceGroup)

		mockClient := mocks.NewMockPublicIPPrefixesClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, prefixName, nil).Return(
			armnetwork.PublicIPPrefixesClientGetResponse{
				PublicIPPrefix: *prefix,
			}, nil)

		wrapper := manual.NewNetworkPublicIPPrefix(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], prefixName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		t.Run("StaticTests", func(t *testing.T) {
			scope := subscriptionID + "." + resourceGroup
			queryTests := shared.QueryTests{
				{
					ExpectedType:   azureshared.ExtendedLocationCustomLocation.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-custom-location",
					ExpectedScope:  scope,
				},
				{
					ExpectedType:   stdlib.NetworkIP.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "20.10.0.0/28",
					ExpectedScope:  "global",
				},
				{
					ExpectedType:   azureshared.NetworkCustomIPPrefix.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-custom-prefix",
					ExpectedScope:  scope,
				},
				{
					ExpectedType:   azureshared.NetworkNatGateway.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-nat-gateway",
					ExpectedScope:  scope,
				},
				{
					ExpectedType:   azureshared.NetworkLoadBalancer.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-load-balancer",
					ExpectedScope:  scope,
				},
				{
					ExpectedType:   azureshared.NetworkLoadBalancerFrontendIPConfiguration.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey("test-load-balancer", "frontend"),
					ExpectedScope:  scope,
				},
				{
					ExpectedType:   azureshared.NetworkPublicIPAddress.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "referenced-public-ip",
					ExpectedScope:  scope,
				},
			}
			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("GetWithEmptyName", func(t *testing.T) {
		mockClient := mocks.NewMockPublicIPPrefixesClient(ctrl)

		wrapper := manual.NewNetworkPublicIPPrefix(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "", true)
		if qErr == nil {
			t.Error("Expected error when public IP prefix name is empty, but got nil")
		}
	})

	t.Run("Get_PrefixWithNilName", func(t *testing.T) {
		provisioningState := armnetwork.ProvisioningStateSucceeded
		prefixWithNilName := &armnetwork.PublicIPPrefix{
			Name:     nil,
			Location: new("eastus"),
			Properties: &armnetwork.PublicIPPrefixPropertiesFormat{
				ProvisioningState: &provisioningState,
			},
		}

		mockClient := mocks.NewMockPublicIPPrefixesClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, "test-prefix", nil).Return(
			armnetwork.PublicIPPrefixesClientGetResponse{
				PublicIPPrefix: *prefixWithNilName,
			}, nil)

		wrapper := manual.NewNetworkPublicIPPrefix(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "test-prefix", true)
		if qErr == nil {
			t.Error("Expected error when public IP prefix has nil name, but got nil")
		}
	})

	t.Run("List", func(t *testing.T) {
		prefix1 := createAzurePublicIPPrefix("prefix-1")
		prefix2 := createAzurePublicIPPrefix("prefix-2")

		mockClient := mocks.NewMockPublicIPPrefixesClient(ctrl)
		mockPager := newMockPublicIPPrefixesPager(ctrl, []*armnetwork.PublicIPPrefix{prefix1, prefix2})

		mockClient.EXPECT().NewListPager(resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewNetworkPublicIPPrefix(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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
			if item.GetType() != azureshared.NetworkPublicIPPrefix.String() {
				t.Fatalf("Expected type %s, got: %s", azureshared.NetworkPublicIPPrefix.String(), item.GetType())
			}
		}
	})

	t.Run("List_WithNilName", func(t *testing.T) {
		prefix1 := createAzurePublicIPPrefix("prefix-1")
		provisioningState := armnetwork.ProvisioningStateSucceeded
		prefix2NilName := &armnetwork.PublicIPPrefix{
			Name:     nil,
			Location: new("eastus"),
			Tags:     map[string]*string{"env": new("test")},
			Properties: &armnetwork.PublicIPPrefixPropertiesFormat{
				ProvisioningState: &provisioningState,
			},
		}

		mockClient := mocks.NewMockPublicIPPrefixesClient(ctrl)
		mockPager := newMockPublicIPPrefixesPager(ctrl, []*armnetwork.PublicIPPrefix{prefix1, prefix2NilName})

		mockClient.EXPECT().NewListPager(resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewNetworkPublicIPPrefix(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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
		if sdpItems[0].UniqueAttributeValue() != "prefix-1" {
			t.Errorf("Expected item name 'prefix-1', got: %s", sdpItems[0].UniqueAttributeValue())
		}
	})

	t.Run("ListStream", func(t *testing.T) {
		prefix1 := createAzurePublicIPPrefix("stream-prefix-1")
		prefix2 := createAzurePublicIPPrefix("stream-prefix-2")

		mockClient := mocks.NewMockPublicIPPrefixesClient(ctrl)
		mockPager := newMockPublicIPPrefixesPager(ctrl, []*armnetwork.PublicIPPrefix{prefix1, prefix2})

		mockClient.EXPECT().NewListPager(resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewNetworkPublicIPPrefix(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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
		expectedErr := errors.New("public IP prefix not found")

		mockClient := mocks.NewMockPublicIPPrefixesClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, "nonexistent-prefix", nil).Return(
			armnetwork.PublicIPPrefixesClientGetResponse{}, expectedErr)

		wrapper := manual.NewNetworkPublicIPPrefix(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "nonexistent-prefix", true)
		if qErr == nil {
			t.Error("Expected error when getting non-existent public IP prefix, but got nil")
		}
	})

	t.Run("InterfaceCompliance", func(t *testing.T) {
		mockClient := mocks.NewMockPublicIPPrefixesClient(ctrl)
		wrapper := manual.NewNetworkPublicIPPrefix(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		w := wrapper.(sources.Wrapper)

		permissions := w.IAMPermissions()
		if len(permissions) == 0 {
			t.Error("Expected IAMPermissions to return at least one permission")
		}
		expectedPermission := "Microsoft.Network/publicIPPrefixes/read"
		if !slices.Contains(permissions, expectedPermission) {
			t.Errorf("Expected IAMPermissions to include %s", expectedPermission)
		}

		mappings := w.TerraformMappings()
		foundMapping := false
		for _, mapping := range mappings {
			if mapping.GetTerraformQueryMap() == "azurerm_public_ip_prefix.name" {
				foundMapping = true
				if mapping.GetTerraformMethod() != sdp.QueryMethod_GET {
					t.Errorf("Expected TerraformMethod GET, got: %s", mapping.GetTerraformMethod())
				}
				break
			}
		}
		if !foundMapping {
			t.Error("Expected TerraformMappings to include 'azurerm_public_ip_prefix.name'")
		}

		lookups := w.GetLookups()
		foundLookup := false
		for _, lookup := range lookups {
			if lookup.ItemType == azureshared.NetworkPublicIPPrefix {
				foundLookup = true
				break
			}
		}
		if !foundLookup {
			t.Error("Expected GetLookups to include NetworkPublicIPPrefix")
		}

		potentialLinks := w.PotentialLinks()
		for _, linkType := range []shared.ItemType{azureshared.ExtendedLocationCustomLocation, azureshared.NetworkCustomIPPrefix, azureshared.NetworkNatGateway, azureshared.NetworkLoadBalancer, azureshared.NetworkLoadBalancerFrontendIPConfiguration, azureshared.NetworkPublicIPAddress, stdlib.NetworkIP} {
			if !potentialLinks[linkType] {
				t.Errorf("Expected PotentialLinks to include %s", linkType)
			}
		}
	})
}

type mockPublicIPPrefixesPager struct {
	ctrl  *gomock.Controller
	items []*armnetwork.PublicIPPrefix
	index int
	more  bool
}

func newMockPublicIPPrefixesPager(ctrl *gomock.Controller, items []*armnetwork.PublicIPPrefix) clients.PublicIPPrefixesPager {
	return &mockPublicIPPrefixesPager{
		ctrl:  ctrl,
		items: items,
		index: 0,
		more:  len(items) > 0,
	}
}

func (m *mockPublicIPPrefixesPager) More() bool {
	return m.more
}

func (m *mockPublicIPPrefixesPager) NextPage(ctx context.Context) (armnetwork.PublicIPPrefixesClientListResponse, error) {
	if m.index >= len(m.items) {
		m.more = false
		return armnetwork.PublicIPPrefixesClientListResponse{
			PublicIPPrefixListResult: armnetwork.PublicIPPrefixListResult{
				Value: []*armnetwork.PublicIPPrefix{},
			},
		}, nil
	}
	item := m.items[m.index]
	m.index++
	m.more = m.index < len(m.items)
	return armnetwork.PublicIPPrefixesClientListResponse{
		PublicIPPrefixListResult: armnetwork.PublicIPPrefixListResult{
			Value: []*armnetwork.PublicIPPrefix{item},
		},
	}, nil
}

func createAzurePublicIPPrefix(name string) *armnetwork.PublicIPPrefix {
	provisioningState := armnetwork.ProvisioningStateSucceeded
	prefixLength := int32(28)
	return &armnetwork.PublicIPPrefix{
		ID:       new("/subscriptions/test-sub/resourceGroups/test-rg/providers/Microsoft.Network/publicIPPrefixes/" + name),
		Name:     new(name),
		Type:     new("Microsoft.Network/publicIPPrefixes"),
		Location: new("eastus"),
		Tags: map[string]*string{
			"env":     new("test"),
			"project": new("testing"),
		},
		Properties: &armnetwork.PublicIPPrefixPropertiesFormat{
			ProvisioningState: &provisioningState,
			PrefixLength:      &prefixLength,
		},
	}
}

func createAzurePublicIPPrefixWithLinks(name, subscriptionID, resourceGroup string) *armnetwork.PublicIPPrefix {
	prefix := createAzurePublicIPPrefix(name)
	prefix.Properties.IPPrefix = new("20.10.0.0/28")
	customLocationID := "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.ExtendedLocation/customLocations/test-custom-location"
	customPrefixID := "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/customIPPrefixes/test-custom-prefix"
	natGatewayID := "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/natGateways/test-nat-gateway"
	lbFeConfigID := "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/loadBalancers/test-load-balancer/frontendIPConfigurations/frontend"
	publicIPID := "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/publicIPAddresses/referenced-public-ip"

	prefix.ExtendedLocation = &armnetwork.ExtendedLocation{
		Name: new(customLocationID),
	}
	prefix.Properties.CustomIPPrefix = &armnetwork.SubResource{
		ID: new(customPrefixID),
	}
	prefix.Properties.NatGateway = &armnetwork.NatGateway{
		ID: new(natGatewayID),
	}
	prefix.Properties.LoadBalancerFrontendIPConfiguration = &armnetwork.SubResource{
		ID: new(lbFeConfigID),
	}
	prefix.Properties.PublicIPAddresses = []*armnetwork.ReferencedPublicIPAddress{
		{ID: new(publicIPID)},
	}
	return prefix
}

var _ clients.PublicIPPrefixesPager = (*mockPublicIPPrefixesPager)(nil)
