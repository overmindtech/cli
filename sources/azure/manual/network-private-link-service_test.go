package manual_test

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v9"
	"go.uber.org/mock/gomock"

	"github.com/overmindtech/cli/go/discovery"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/manual"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/azure/shared/mocks"
	"github.com/overmindtech/cli/sources/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

func TestNetworkPrivateLinkService(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"

	t.Run("Get", func(t *testing.T) {
		plsName := "test-pls"
		pls := createAzurePrivateLinkService(plsName, subscriptionID, resourceGroup)

		mockClient := mocks.NewMockPrivateLinkServicesClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, plsName).Return(
			armnetwork.PrivateLinkServicesClientGetResponse{
				PrivateLinkService: *pls,
			}, nil)

		wrapper := manual.NewNetworkPrivateLinkService(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], plsName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.NetworkPrivateLinkService.String() {
			t.Errorf("Expected type %s, got %s", azureshared.NetworkPrivateLinkService, sdpItem.GetType())
		}

		if sdpItem.GetUniqueAttribute() != "name" {
			t.Errorf("Expected unique attribute 'name', got %s", sdpItem.GetUniqueAttribute())
		}

		if sdpItem.UniqueAttributeValue() != plsName {
			t.Errorf("Expected unique attribute value %s, got %s", plsName, sdpItem.UniqueAttributeValue())
		}

		if sdpItem.GetTags()["env"] != "test" {
			t.Errorf("Expected tag 'env=test', got: %v", sdpItem.GetTags()["env"])
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					ExpectedType:   azureshared.NetworkSubnet.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey("test-vnet", "test-subnet"),
					ExpectedScope:  fmt.Sprintf("%s.%s", subscriptionID, resourceGroup),
				},
				{
					ExpectedType:   azureshared.NetworkVirtualNetwork.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-vnet",
					ExpectedScope:  fmt.Sprintf("%s.%s", subscriptionID, resourceGroup),
				},
				{
					ExpectedType:   stdlib.NetworkIP.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "10.0.0.100",
					ExpectedScope:  "global",
				},
				{
					ExpectedType:   azureshared.NetworkLoadBalancerFrontendIPConfiguration.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey("test-lb", "test-frontend-ip"),
					ExpectedScope:  fmt.Sprintf("%s.%s", subscriptionID, resourceGroup),
				},
				{
					ExpectedType:   azureshared.NetworkLoadBalancer.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-lb",
					ExpectedScope:  fmt.Sprintf("%s.%s", subscriptionID, resourceGroup),
				},
				{
					ExpectedType:   azureshared.NetworkNetworkInterface.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-nic",
					ExpectedScope:  fmt.Sprintf("%s.%s", subscriptionID, resourceGroup),
				},
				{
					ExpectedType:   azureshared.NetworkPrivateEndpoint.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-pe",
					ExpectedScope:  fmt.Sprintf("%s.%s", subscriptionID, resourceGroup),
				},
				{
					ExpectedType:   stdlib.NetworkDNS.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  "pls.example.com",
					ExpectedScope:  "global",
				},
				{
					ExpectedType:   stdlib.NetworkIP.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "10.0.0.200",
					ExpectedScope:  "global",
				},
				{
					ExpectedType:   stdlib.NetworkDNS.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  "test-pls.abc123.westus2.azure.privatelinkservice",
					ExpectedScope:  "global",
				},
				{
					ExpectedType:   azureshared.ExtendedLocationCustomLocation.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-custom-location",
					ExpectedScope:  fmt.Sprintf("%s.%s", subscriptionID, resourceGroup),
				},
			}

			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("Get_EmptyName", func(t *testing.T) {
		mockClient := mocks.NewMockPrivateLinkServicesClient(ctrl)

		wrapper := manual.NewNetworkPrivateLinkService(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "", true)
		if qErr == nil {
			t.Error("Expected error when getting private link service with empty name, but got nil")
		}
	})

	t.Run("List", func(t *testing.T) {
		pls1 := createAzurePrivateLinkService("test-pls-1", subscriptionID, resourceGroup)
		pls2 := createAzurePrivateLinkService("test-pls-2", subscriptionID, resourceGroup)

		mockClient := mocks.NewMockPrivateLinkServicesClient(ctrl)
		mockPager := NewMockPrivateLinkServicesPager(ctrl)

		gomock.InOrder(
			mockPager.EXPECT().More().Return(true),
			mockPager.EXPECT().NextPage(ctx).Return(
				armnetwork.PrivateLinkServicesClientListResponse{
					PrivateLinkServiceListResult: armnetwork.PrivateLinkServiceListResult{
						Value: []*armnetwork.PrivateLinkService{pls1, pls2},
					},
				}, nil),
			mockPager.EXPECT().More().Return(false),
		)

		mockClient.EXPECT().List(resourceGroup).Return(mockPager)

		wrapper := manual.NewNetworkPrivateLinkService(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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
			if item.GetType() != azureshared.NetworkPrivateLinkService.String() {
				t.Fatalf("Expected type %s, got: %s", azureshared.NetworkPrivateLinkService, item.GetType())
			}
		}
	})

	t.Run("ListStream", func(t *testing.T) {
		pls1 := createAzurePrivateLinkService("test-pls-1", subscriptionID, resourceGroup)
		pls2 := createAzurePrivateLinkService("test-pls-2", subscriptionID, resourceGroup)

		mockClient := mocks.NewMockPrivateLinkServicesClient(ctrl)
		mockPager := NewMockPrivateLinkServicesPager(ctrl)

		gomock.InOrder(
			mockPager.EXPECT().More().Return(true),
			mockPager.EXPECT().NextPage(ctx).Return(
				armnetwork.PrivateLinkServicesClientListResponse{
					PrivateLinkServiceListResult: armnetwork.PrivateLinkServiceListResult{
						Value: []*armnetwork.PrivateLinkService{pls1, pls2},
					},
				}, nil),
			mockPager.EXPECT().More().Return(false),
		)

		mockClient.EXPECT().List(resourceGroup).Return(mockPager)

		wrapper := manual.NewNetworkPrivateLinkService(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		wg := &sync.WaitGroup{}
		wg.Add(2)

		var items []*sdp.Item
		var errs []error

		mockItemHandler := func(item *sdp.Item) {
			items = append(items, item)
			wg.Done()
		}
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

	t.Run("List_WithNilName", func(t *testing.T) {
		pls1 := createAzurePrivateLinkService("test-pls-1", subscriptionID, resourceGroup)
		pls2 := &armnetwork.PrivateLinkService{
			Name:     nil,
			Location: new("eastus"),
			Tags:     map[string]*string{"env": new("test")},
			Properties: &armnetwork.PrivateLinkServiceProperties{
				ProvisioningState: to.Ptr(armnetwork.ProvisioningStateSucceeded),
			},
		}

		mockClient := mocks.NewMockPrivateLinkServicesClient(ctrl)
		mockPager := NewMockPrivateLinkServicesPager(ctrl)

		gomock.InOrder(
			mockPager.EXPECT().More().Return(true),
			mockPager.EXPECT().NextPage(ctx).Return(
				armnetwork.PrivateLinkServicesClientListResponse{
					PrivateLinkServiceListResult: armnetwork.PrivateLinkServiceListResult{
						Value: []*armnetwork.PrivateLinkService{pls1, pls2},
					},
				}, nil),
			mockPager.EXPECT().More().Return(false),
		)

		mockClient.EXPECT().List(resourceGroup).Return(mockPager)

		wrapper := manual.NewNetworkPrivateLinkService(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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
		if sdpItems[0].UniqueAttributeValue() != "test-pls-1" {
			t.Errorf("Expected item name 'test-pls-1', got: %s", sdpItems[0].UniqueAttributeValue())
		}
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		expectedErr := errors.New("private link service not found")

		mockClient := mocks.NewMockPrivateLinkServicesClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, "nonexistent-pls").Return(
			armnetwork.PrivateLinkServicesClientGetResponse{}, expectedErr)

		wrapper := manual.NewNetworkPrivateLinkService(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "nonexistent-pls", true)
		if qErr == nil {
			t.Fatal("Expected error when getting nonexistent private link service, got nil")
		}
	})

	t.Run("PotentialLinks", func(t *testing.T) {
		mockClient := mocks.NewMockPrivateLinkServicesClient(ctrl)
		wrapper := manual.NewNetworkPrivateLinkService(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		w := wrapper.(sources.Wrapper)
		potentialLinks := w.PotentialLinks()
		if len(potentialLinks) == 0 {
			t.Error("Expected PotentialLinks to return at least one link type")
		}
		if !potentialLinks[azureshared.NetworkSubnet] {
			t.Error("Expected PotentialLinks to include NetworkSubnet")
		}
		if !potentialLinks[azureshared.NetworkVirtualNetwork] {
			t.Error("Expected PotentialLinks to include NetworkVirtualNetwork")
		}
		if !potentialLinks[azureshared.NetworkLoadBalancer] {
			t.Error("Expected PotentialLinks to include NetworkLoadBalancer")
		}
		if !potentialLinks[azureshared.NetworkLoadBalancerFrontendIPConfiguration] {
			t.Error("Expected PotentialLinks to include NetworkLoadBalancerFrontendIPConfiguration")
		}
		if !potentialLinks[azureshared.NetworkNetworkInterface] {
			t.Error("Expected PotentialLinks to include NetworkNetworkInterface")
		}
		if !potentialLinks[azureshared.NetworkPrivateEndpoint] {
			t.Error("Expected PotentialLinks to include NetworkPrivateEndpoint")
		}
		if !potentialLinks[stdlib.NetworkIP] {
			t.Error("Expected PotentialLinks to include stdlib.NetworkIP")
		}
		if !potentialLinks[stdlib.NetworkDNS] {
			t.Error("Expected PotentialLinks to include stdlib.NetworkDNS")
		}
	})
}

// MockPrivateLinkServicesPager is a mock for PrivateLinkServicesPager
type MockPrivateLinkServicesPager struct {
	ctrl     *gomock.Controller
	recorder *MockPrivateLinkServicesPagerMockRecorder
}

type MockPrivateLinkServicesPagerMockRecorder struct {
	mock *MockPrivateLinkServicesPager
}

func NewMockPrivateLinkServicesPager(ctrl *gomock.Controller) *MockPrivateLinkServicesPager {
	mock := &MockPrivateLinkServicesPager{ctrl: ctrl}
	mock.recorder = &MockPrivateLinkServicesPagerMockRecorder{mock}
	return mock
}

func (m *MockPrivateLinkServicesPager) EXPECT() *MockPrivateLinkServicesPagerMockRecorder {
	return m.recorder
}

func (m *MockPrivateLinkServicesPager) More() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "More")
	ret0, _ := ret[0].(bool)
	return ret0
}

func (mr *MockPrivateLinkServicesPagerMockRecorder) More() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "More", reflect.TypeFor[func() bool]())
}

func (m *MockPrivateLinkServicesPager) NextPage(ctx context.Context) (armnetwork.PrivateLinkServicesClientListResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NextPage", ctx)
	ret0, _ := ret[0].(armnetwork.PrivateLinkServicesClientListResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockPrivateLinkServicesPagerMockRecorder) NextPage(ctx any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NextPage", reflect.TypeFor[func(ctx context.Context) (armnetwork.PrivateLinkServicesClientListResponse, error)](), ctx)
}

func createAzurePrivateLinkService(plsName, subscriptionID, resourceGroup string) *armnetwork.PrivateLinkService {
	subnetID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/test-subnet", subscriptionID, resourceGroup)
	lbFrontendIPID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/loadBalancers/test-lb/frontendIPConfigurations/test-frontend-ip", subscriptionID, resourceGroup)
	nicID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/networkInterfaces/test-nic", subscriptionID, resourceGroup)
	peID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/privateEndpoints/test-pe", subscriptionID, resourceGroup)
	customLocationID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.ExtendedLocation/customLocations/test-custom-location", subscriptionID, resourceGroup)

	return &armnetwork.PrivateLinkService{
		Name:     new(plsName),
		Location: new("eastus"),
		ExtendedLocation: &armnetwork.ExtendedLocation{
			Name: new(customLocationID),
		},
		Tags: map[string]*string{
			"env":     new("test"),
			"project": new("testing"),
		},
		Properties: &armnetwork.PrivateLinkServiceProperties{
			ProvisioningState: to.Ptr(armnetwork.ProvisioningStateSucceeded),
			IPConfigurations: []*armnetwork.PrivateLinkServiceIPConfiguration{
				{
					Properties: &armnetwork.PrivateLinkServiceIPConfigurationProperties{
						Subnet: &armnetwork.Subnet{
							ID: new(subnetID),
						},
						PrivateIPAddress: new("10.0.0.100"),
					},
				},
			},
			LoadBalancerFrontendIPConfigurations: []*armnetwork.FrontendIPConfiguration{
				{
					ID: new(lbFrontendIPID),
				},
			},
			NetworkInterfaces: []*armnetwork.Interface{
				{
					ID: new(nicID),
				},
			},
			PrivateEndpointConnections: []*armnetwork.PrivateEndpointConnection{
				{
					Properties: &armnetwork.PrivateEndpointConnectionProperties{
						PrivateEndpoint: &armnetwork.PrivateEndpoint{
							ID: new(peID),
						},
					},
				},
			},
			Fqdns: []*string{
				new("pls.example.com"),
			},
			DestinationIPAddress: new("10.0.0.200"),
			Alias:                new("test-pls.abc123.westus2.azure.privatelinkservice"),
		},
	}
}
