package manual_test

import (
	"context"
	"errors"
	"fmt"
	"reflect"
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

func TestNetworkPrivateEndpoint(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"

	t.Run("Get", func(t *testing.T) {
		peName := "test-pe"
		pe := createAzurePrivateEndpoint(peName, subscriptionID, resourceGroup)

		mockClient := mocks.NewMockPrivateEndpointsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, peName).Return(
			armnetwork.PrivateEndpointsClientGetResponse{
				PrivateEndpoint: *pe,
			}, nil)

		wrapper := manual.NewNetworkPrivateEndpoint(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], peName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.NetworkPrivateEndpoint.String() {
			t.Errorf("Expected type %s, got %s", azureshared.NetworkPrivateEndpoint, sdpItem.GetType())
		}

		if sdpItem.GetUniqueAttribute() != "name" {
			t.Errorf("Expected unique attribute 'name', got %s", sdpItem.GetUniqueAttribute())
		}

		if sdpItem.UniqueAttributeValue() != peName {
			t.Errorf("Expected unique attribute value %s, got %s", peName, sdpItem.UniqueAttributeValue())
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
				}, {
					ExpectedType:   azureshared.NetworkVirtualNetwork.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-vnet",
					ExpectedScope:  fmt.Sprintf("%s.%s", subscriptionID, resourceGroup),
				}, {
					ExpectedType:   azureshared.NetworkNetworkInterface.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-nic",
					ExpectedScope:  fmt.Sprintf("%s.%s", subscriptionID, resourceGroup),
				}, {
					ExpectedType:   azureshared.NetworkApplicationSecurityGroup.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-asg",
					ExpectedScope:  fmt.Sprintf("%s.%s", subscriptionID, resourceGroup),
				}, {
					ExpectedType:   stdlib.NetworkIP.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "10.0.0.10",
					ExpectedScope:  "global",
				}, {
					ExpectedType:   stdlib.NetworkDNS.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  "myendpoint.example.com",
					ExpectedScope:  "global",
				}, {
					ExpectedType:   stdlib.NetworkIP.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "10.0.0.5",
					ExpectedScope:  "global",
				},
			}

			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("Get_EmptyName", func(t *testing.T) {
		mockClient := mocks.NewMockPrivateEndpointsClient(ctrl)

		wrapper := manual.NewNetworkPrivateEndpoint(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "", true)
		if qErr == nil {
			t.Error("Expected error when getting private endpoint with empty name, but got nil")
		}
	})

	t.Run("List", func(t *testing.T) {
		pe1 := createAzurePrivateEndpoint("test-pe-1", subscriptionID, resourceGroup)
		pe2 := createAzurePrivateEndpoint("test-pe-2", subscriptionID, resourceGroup)

		mockClient := mocks.NewMockPrivateEndpointsClient(ctrl)
		mockPager := NewMockPrivateEndpointsPager(ctrl)

		gomock.InOrder(
			mockPager.EXPECT().More().Return(true),
			mockPager.EXPECT().NextPage(ctx).Return(
				armnetwork.PrivateEndpointsClientListResponse{
					PrivateEndpointListResult: armnetwork.PrivateEndpointListResult{
						Value: []*armnetwork.PrivateEndpoint{pe1, pe2},
					},
				}, nil),
			mockPager.EXPECT().More().Return(false),
		)

		mockClient.EXPECT().List(resourceGroup).Return(mockPager)

		wrapper := manual.NewNetworkPrivateEndpoint(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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
			if item.GetType() != azureshared.NetworkPrivateEndpoint.String() {
				t.Fatalf("Expected type %s, got: %s", azureshared.NetworkPrivateEndpoint, item.GetType())
			}
		}
	})

	t.Run("List_WithNilName", func(t *testing.T) {
		pe1 := createAzurePrivateEndpoint("test-pe-1", subscriptionID, resourceGroup)
		pe2 := &armnetwork.PrivateEndpoint{
			Name:     nil,
			Location: new("eastus"),
			Tags:     map[string]*string{"env": new("test")},
			Properties: &armnetwork.PrivateEndpointProperties{
				ProvisioningState: to.Ptr(armnetwork.ProvisioningStateSucceeded),
			},
		}

		mockClient := mocks.NewMockPrivateEndpointsClient(ctrl)
		mockPager := NewMockPrivateEndpointsPager(ctrl)

		gomock.InOrder(
			mockPager.EXPECT().More().Return(true),
			mockPager.EXPECT().NextPage(ctx).Return(
				armnetwork.PrivateEndpointsClientListResponse{
					PrivateEndpointListResult: armnetwork.PrivateEndpointListResult{
						Value: []*armnetwork.PrivateEndpoint{pe1, pe2},
					},
				}, nil),
			mockPager.EXPECT().More().Return(false),
		)

		mockClient.EXPECT().List(resourceGroup).Return(mockPager)

		wrapper := manual.NewNetworkPrivateEndpoint(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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
		if sdpItems[0].UniqueAttributeValue() != "test-pe-1" {
			t.Errorf("Expected item name 'test-pe-1', got: %s", sdpItems[0].UniqueAttributeValue())
		}
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		expectedErr := errors.New("private endpoint not found")

		mockClient := mocks.NewMockPrivateEndpointsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, "nonexistent-pe").Return(
			armnetwork.PrivateEndpointsClientGetResponse{}, expectedErr)

		wrapper := manual.NewNetworkPrivateEndpoint(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "nonexistent-pe", true)
		if qErr == nil {
			t.Fatal("Expected error when getting nonexistent private endpoint, got nil")
		}
	})

	t.Run("PotentialLinks", func(t *testing.T) {
		mockClient := mocks.NewMockPrivateEndpointsClient(ctrl)
		wrapper := manual.NewNetworkPrivateEndpoint(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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
	})
}

// MockPrivateEndpointsPager is a mock for PrivateEndpointsPager
type MockPrivateEndpointsPager struct {
	ctrl     *gomock.Controller
	recorder *MockPrivateEndpointsPagerMockRecorder
}

type MockPrivateEndpointsPagerMockRecorder struct {
	mock *MockPrivateEndpointsPager
}

func NewMockPrivateEndpointsPager(ctrl *gomock.Controller) *MockPrivateEndpointsPager {
	mock := &MockPrivateEndpointsPager{ctrl: ctrl}
	mock.recorder = &MockPrivateEndpointsPagerMockRecorder{mock}
	return mock
}

func (m *MockPrivateEndpointsPager) EXPECT() *MockPrivateEndpointsPagerMockRecorder {
	return m.recorder
}

func (m *MockPrivateEndpointsPager) More() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "More")
	ret0, _ := ret[0].(bool)
	return ret0
}

func (mr *MockPrivateEndpointsPagerMockRecorder) More() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "More", reflect.TypeFor[func() bool]())
}

func (m *MockPrivateEndpointsPager) NextPage(ctx context.Context) (armnetwork.PrivateEndpointsClientListResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NextPage", ctx)
	ret0, _ := ret[0].(armnetwork.PrivateEndpointsClientListResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockPrivateEndpointsPagerMockRecorder) NextPage(ctx any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NextPage", reflect.TypeFor[func(ctx context.Context) (armnetwork.PrivateEndpointsClientListResponse, error)](), ctx)
}

func createAzurePrivateEndpoint(peName, subscriptionID, resourceGroup string) *armnetwork.PrivateEndpoint {
	subnetID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/test-subnet", subscriptionID, resourceGroup)
	nicID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/networkInterfaces/test-nic", subscriptionID, resourceGroup)
	asgID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/applicationSecurityGroups/test-asg", subscriptionID, resourceGroup)

	return &armnetwork.PrivateEndpoint{
		Name:     new(peName),
		Location: new("eastus"),
		Tags: map[string]*string{
			"env":     new("test"),
			"project": new("testing"),
		},
		Properties: &armnetwork.PrivateEndpointProperties{
			ProvisioningState: to.Ptr(armnetwork.ProvisioningStateSucceeded),
			Subnet: &armnetwork.Subnet{
				ID: new(subnetID),
			},
			NetworkInterfaces: []*armnetwork.Interface{
				{ID: new(nicID)},
			},
			ApplicationSecurityGroups: []*armnetwork.ApplicationSecurityGroup{
				{ID: new(asgID)},
			},
			IPConfigurations: []*armnetwork.PrivateEndpointIPConfiguration{
				{
					Properties: &armnetwork.PrivateEndpointIPConfigurationProperties{
						PrivateIPAddress: new("10.0.0.10"),
					},
				},
			},
			CustomDNSConfigs: []*armnetwork.CustomDNSConfigPropertiesFormat{
				{
					Fqdn:        new("myendpoint.example.com"),
					IPAddresses: []*string{new("10.0.0.5")},
				},
			},
		},
	}
}
