package manual_test

import (
	"context"
	"errors"
	"reflect"
	"slices"
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

// MockInterfaceIPConfigurationsPager is a simple mock for InterfaceIPConfigurationsPager
type MockInterfaceIPConfigurationsPager struct {
	ctrl     *gomock.Controller
	recorder *MockInterfaceIPConfigurationsPagerMockRecorder
}

type MockInterfaceIPConfigurationsPagerMockRecorder struct {
	mock *MockInterfaceIPConfigurationsPager
}

func NewMockInterfaceIPConfigurationsPager(ctrl *gomock.Controller) *MockInterfaceIPConfigurationsPager {
	mock := &MockInterfaceIPConfigurationsPager{ctrl: ctrl}
	mock.recorder = &MockInterfaceIPConfigurationsPagerMockRecorder{mock}
	return mock
}

func (m *MockInterfaceIPConfigurationsPager) EXPECT() *MockInterfaceIPConfigurationsPagerMockRecorder {
	return m.recorder
}

func (m *MockInterfaceIPConfigurationsPager) More() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "More")
	ret0, _ := ret[0].(bool)
	return ret0
}

func (mr *MockInterfaceIPConfigurationsPagerMockRecorder) More() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "More", reflect.TypeFor[func() bool]())
}

func (m *MockInterfaceIPConfigurationsPager) NextPage(ctx context.Context) (armnetwork.InterfaceIPConfigurationsClientListResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NextPage", ctx)
	ret0, _ := ret[0].(armnetwork.InterfaceIPConfigurationsClientListResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockInterfaceIPConfigurationsPagerMockRecorder) NextPage(ctx any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NextPage", reflect.TypeFor[func(ctx context.Context) (armnetwork.InterfaceIPConfigurationsClientListResponse, error)](), ctx)
}

// testInterfaceIPConfigurationsClient wraps the mock to implement the correct interface
type testInterfaceIPConfigurationsClient struct {
	*mocks.MockInterfaceIPConfigurationsClient
	pager clients.InterfaceIPConfigurationsPager
}

func (t *testInterfaceIPConfigurationsClient) List(ctx context.Context, resourceGroupName, networkInterfaceName string) clients.InterfaceIPConfigurationsPager {
	return t.pager
}

func TestNetworkNetworkInterfaceIPConfiguration(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"
	networkInterfaceName := "test-nic"
	ipConfigName := "ipconfig1"

	t.Run("Get", func(t *testing.T) {
		ipConfig := createAzureIPConfiguration(subscriptionID, resourceGroup, networkInterfaceName, ipConfigName)

		mockClient := mocks.NewMockInterfaceIPConfigurationsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, networkInterfaceName, ipConfigName).Return(
			armnetwork.InterfaceIPConfigurationsClientGetResponse{
				InterfaceIPConfiguration: *ipConfig,
			}, nil)

		wrapper := manual.NewNetworkNetworkInterfaceIPConfiguration(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(networkInterfaceName, ipConfigName)
		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.NetworkNetworkInterfaceIPConfiguration.String() {
			t.Errorf("Expected type %s, got %s", azureshared.NetworkNetworkInterfaceIPConfiguration, sdpItem.GetType())
		}

		if sdpItem.GetUniqueAttribute() != "uniqueAttr" {
			t.Errorf("Expected unique attribute 'uniqueAttr', got %s", sdpItem.GetUniqueAttribute())
		}

		expectedUniqueValue := shared.CompositeLookupKey(networkInterfaceName, ipConfigName)
		if sdpItem.UniqueAttributeValue() != expectedUniqueValue {
			t.Errorf("Expected unique attribute value %s, got %s", expectedUniqueValue, sdpItem.UniqueAttributeValue())
		}

		if sdpItem.GetScope() != subscriptionID+"."+resourceGroup {
			t.Errorf("Expected scope %s, got %s", subscriptionID+"."+resourceGroup, sdpItem.GetScope())
		}

		if sdpItem.Validate() != nil {
			t.Fatalf("Expected no validation error, got: %v", sdpItem.Validate())
		}

		// Verify health status is OK for Succeeded provisioning state
		if sdpItem.GetHealth() != sdp.Health_HEALTH_OK {
			t.Errorf("Expected health OK, got %v", sdpItem.GetHealth())
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					// Parent NetworkInterface link
					ExpectedType:   azureshared.NetworkNetworkInterface.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  networkInterfaceName,
					ExpectedScope:  subscriptionID + "." + resourceGroup,
				},
				{
					// Subnet link
					ExpectedType:   azureshared.NetworkSubnet.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey("test-vnet", "test-subnet"),
					ExpectedScope:  subscriptionID + "." + resourceGroup,
				},
				{
					// Public IP address link
					ExpectedType:   azureshared.NetworkPublicIPAddress.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-pip",
					ExpectedScope:  subscriptionID + "." + resourceGroup,
				},
				{
					// Private IP address link (stdlib)
					ExpectedType:   stdlib.NetworkIP.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "10.0.0.4",
					ExpectedScope:  "global",
				},
			}

			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("GetWithInsufficientQueryParts", func(t *testing.T) {
		mockClient := mocks.NewMockInterfaceIPConfigurationsClient(ctrl)

		wrapper := manual.NewNetworkNetworkInterfaceIPConfiguration(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		// Test with only network interface name (missing ipConfigName)
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], networkInterfaceName, true)
		if qErr == nil {
			t.Error("Expected error when providing insufficient query parts, but got nil")
		}
	})

	t.Run("GetWithEmptyNetworkInterfaceName", func(t *testing.T) {
		mockClient := mocks.NewMockInterfaceIPConfigurationsClient(ctrl)

		wrapper := manual.NewNetworkNetworkInterfaceIPConfiguration(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		// Test directly on wrapper to get the QueryError
		_, qErr := wrapper.Get(ctx, wrapper.Scopes()[0], "", ipConfigName)
		if qErr == nil {
			t.Fatal("Expected error when providing empty network interface name, but got nil")
		}
		if qErr.GetErrorString() != "networkInterfaceName cannot be empty" {
			t.Errorf("Expected specific error message, got: %s", qErr.GetErrorString())
		}
	})

	t.Run("GetWithEmptyIPConfigName", func(t *testing.T) {
		mockClient := mocks.NewMockInterfaceIPConfigurationsClient(ctrl)

		wrapper := manual.NewNetworkNetworkInterfaceIPConfiguration(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		// Test directly on wrapper to get the QueryError
		_, qErr := wrapper.Get(ctx, wrapper.Scopes()[0], networkInterfaceName, "")
		if qErr == nil {
			t.Fatal("Expected error when providing empty IP config name, but got nil")
		}
		if qErr.GetErrorString() != "ipConfigurationName cannot be empty" {
			t.Errorf("Expected specific error message, got: %s", qErr.GetErrorString())
		}
	})

	t.Run("Search", func(t *testing.T) {
		ipConfig1 := createAzureIPConfiguration(subscriptionID, resourceGroup, networkInterfaceName, "ipconfig1")
		ipConfig2 := createAzureIPConfiguration(subscriptionID, resourceGroup, networkInterfaceName, "ipconfig2")

		mockClient := mocks.NewMockInterfaceIPConfigurationsClient(ctrl)
		mockPager := NewMockInterfaceIPConfigurationsPager(ctrl)

		gomock.InOrder(
			mockPager.EXPECT().More().Return(true),
			mockPager.EXPECT().NextPage(ctx).Return(
				armnetwork.InterfaceIPConfigurationsClientListResponse{
					InterfaceIPConfigurationListResult: armnetwork.InterfaceIPConfigurationListResult{
						Value: []*armnetwork.InterfaceIPConfiguration{ipConfig1, ipConfig2},
					},
				}, nil),
			mockPager.EXPECT().More().Return(false),
		)

		testClient := &testInterfaceIPConfigurationsClient{
			MockInterfaceIPConfigurationsClient: mockClient,
			pager:                               mockPager,
		}

		wrapper := manual.NewNetworkNetworkInterfaceIPConfiguration(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support Search operation")
		}

		sdpItems, err := searchable.Search(ctx, wrapper.Scopes()[0], networkInterfaceName, true)
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

			if item.GetType() != azureshared.NetworkNetworkInterfaceIPConfiguration.String() {
				t.Errorf("Expected type %s, got %s", azureshared.NetworkNetworkInterfaceIPConfiguration, item.GetType())
			}
		}
	})

	t.Run("SearchWithEmptyNetworkInterfaceName", func(t *testing.T) {
		mockClient := mocks.NewMockInterfaceIPConfigurationsClient(ctrl)
		testClient := &testInterfaceIPConfigurationsClient{MockInterfaceIPConfigurationsClient: mockClient}

		wrapper := manual.NewNetworkNetworkInterfaceIPConfiguration(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		// Test Search directly with empty network interface name
		_, qErr := wrapper.Search(ctx, wrapper.Scopes()[0], "")
		if qErr == nil {
			t.Error("Expected error when providing empty network interface name, but got nil")
		}
	})

	t.Run("SearchWithNoQueryParts", func(t *testing.T) {
		mockClient := mocks.NewMockInterfaceIPConfigurationsClient(ctrl)
		testClient := &testInterfaceIPConfigurationsClient{MockInterfaceIPConfigurationsClient: mockClient}

		wrapper := manual.NewNetworkNetworkInterfaceIPConfiguration(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		// Test Search directly with no query parts
		_, qErr := wrapper.Search(ctx, wrapper.Scopes()[0])
		if qErr == nil {
			t.Error("Expected error when providing no query parts, but got nil")
		}
	})

	t.Run("Search_IPConfigWithNilName", func(t *testing.T) {
		mockClient := mocks.NewMockInterfaceIPConfigurationsClient(ctrl)
		mockPager := NewMockInterfaceIPConfigurationsPager(ctrl)

		ipConfigValid := createAzureIPConfiguration(subscriptionID, resourceGroup, networkInterfaceName, "ipconfig-valid")
		ipConfigNilName := &armnetwork.InterfaceIPConfiguration{
			Name: nil,
		}

		gomock.InOrder(
			mockPager.EXPECT().More().Return(true),
			mockPager.EXPECT().NextPage(ctx).Return(
				armnetwork.InterfaceIPConfigurationsClientListResponse{
					InterfaceIPConfigurationListResult: armnetwork.InterfaceIPConfigurationListResult{
						Value: []*armnetwork.InterfaceIPConfiguration{ipConfigNilName, ipConfigValid},
					},
				}, nil),
			mockPager.EXPECT().More().Return(false),
		)

		testClient := &testInterfaceIPConfigurationsClient{
			MockInterfaceIPConfigurationsClient: mockClient,
			pager:                               mockPager,
		}

		wrapper := manual.NewNetworkNetworkInterfaceIPConfiguration(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support Search operation")
		}

		sdpItems, err := searchable.Search(ctx, wrapper.Scopes()[0], networkInterfaceName, true)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		// Should only return 1 item (the one with a valid name)
		if len(sdpItems) != 1 {
			t.Fatalf("Expected 1 item, got: %d", len(sdpItems))
		}

		expectedUniqueValue := shared.CompositeLookupKey(networkInterfaceName, "ipconfig-valid")
		if sdpItems[0].UniqueAttributeValue() != expectedUniqueValue {
			t.Errorf("Expected unique value %s, got %s", expectedUniqueValue, sdpItems[0].UniqueAttributeValue())
		}
	})

	t.Run("ErrorHandling_Get", func(t *testing.T) {
		expectedErr := errors.New("IP configuration not found")

		mockClient := mocks.NewMockInterfaceIPConfigurationsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, networkInterfaceName, "nonexistent-ipconfig").Return(
			armnetwork.InterfaceIPConfigurationsClientGetResponse{}, expectedErr)

		wrapper := manual.NewNetworkNetworkInterfaceIPConfiguration(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(networkInterfaceName, "nonexistent-ipconfig")
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr == nil {
			t.Error("Expected error when getting non-existent IP configuration, but got nil")
		}
	})

	t.Run("ErrorHandling_Search", func(t *testing.T) {
		expectedErr := errors.New("failed to list IP configurations")

		mockClient := mocks.NewMockInterfaceIPConfigurationsClient(ctrl)
		mockPager := NewMockInterfaceIPConfigurationsPager(ctrl)

		gomock.InOrder(
			mockPager.EXPECT().More().Return(true),
			mockPager.EXPECT().NextPage(ctx).Return(
				armnetwork.InterfaceIPConfigurationsClientListResponse{}, expectedErr),
		)

		testClient := &testInterfaceIPConfigurationsClient{
			MockInterfaceIPConfigurationsClient: mockClient,
			pager:                               mockPager,
		}

		wrapper := manual.NewNetworkNetworkInterfaceIPConfiguration(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support Search operation")
		}

		_, err := searchable.Search(ctx, wrapper.Scopes()[0], networkInterfaceName, true)
		if err == nil {
			t.Error("Expected error when listing IP configurations fails, but got nil")
		}
	})

	t.Run("InterfaceCompliance", func(t *testing.T) {
		mockClient := mocks.NewMockInterfaceIPConfigurationsClient(ctrl)
		wrapper := manual.NewNetworkNetworkInterfaceIPConfiguration(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		// Cast to sources.Wrapper to access interface methods
		w := wrapper.(sources.Wrapper)

		// Verify IAMPermissions
		permissions := w.IAMPermissions()
		if len(permissions) == 0 {
			t.Error("Expected IAMPermissions to return at least one permission")
		}
		expectedPermission := "Microsoft.Network/networkInterfaces/ipConfigurations/read"
		if !slices.Contains(permissions, expectedPermission) {
			t.Errorf("Expected IAMPermissions to include %s, got %v", expectedPermission, permissions)
		}

		// Verify PotentialLinks
		potentialLinks := w.PotentialLinks()
		if len(potentialLinks) == 0 {
			t.Error("Expected PotentialLinks to return at least one link")
		}
		if !potentialLinks[azureshared.NetworkNetworkInterface] {
			t.Error("Expected PotentialLinks to include NetworkNetworkInterface")
		}
		if !potentialLinks[azureshared.NetworkSubnet] {
			t.Error("Expected PotentialLinks to include NetworkSubnet")
		}
		if !potentialLinks[stdlib.NetworkIP] {
			t.Error("Expected PotentialLinks to include NetworkIP")
		}

		// Verify SearchLookups
		searchLookups := wrapper.SearchLookups()
		if len(searchLookups) == 0 {
			t.Error("Expected SearchLookups to return at least one lookup")
		}

		// Verify GetLookups
		getLookups := wrapper.GetLookups()
		if len(getLookups) != 2 {
			t.Errorf("Expected GetLookups to return 2 lookups (parent + child), got %d", len(getLookups))
		}
	})

	t.Run("HealthStatus_Pending", func(t *testing.T) {
		ipConfig := createAzureIPConfigurationWithProvisioningState(subscriptionID, resourceGroup, networkInterfaceName, ipConfigName, armnetwork.ProvisioningStateUpdating)

		mockClient := mocks.NewMockInterfaceIPConfigurationsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, networkInterfaceName, ipConfigName).Return(
			armnetwork.InterfaceIPConfigurationsClientGetResponse{
				InterfaceIPConfiguration: *ipConfig,
			}, nil)

		wrapper := manual.NewNetworkNetworkInterfaceIPConfiguration(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(networkInterfaceName, ipConfigName)
		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetHealth() != sdp.Health_HEALTH_PENDING {
			t.Errorf("Expected health PENDING, got %v", sdpItem.GetHealth())
		}
	})

	t.Run("HealthStatus_Error", func(t *testing.T) {
		ipConfig := createAzureIPConfigurationWithProvisioningState(subscriptionID, resourceGroup, networkInterfaceName, ipConfigName, armnetwork.ProvisioningStateFailed)

		mockClient := mocks.NewMockInterfaceIPConfigurationsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, networkInterfaceName, ipConfigName).Return(
			armnetwork.InterfaceIPConfigurationsClientGetResponse{
				InterfaceIPConfiguration: *ipConfig,
			}, nil)

		wrapper := manual.NewNetworkNetworkInterfaceIPConfiguration(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(networkInterfaceName, ipConfigName)
		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetHealth() != sdp.Health_HEALTH_ERROR {
			t.Errorf("Expected health ERROR, got %v", sdpItem.GetHealth())
		}
	})

	t.Run("GetWithApplicationSecurityGroups", func(t *testing.T) {
		ipConfig := createAzureIPConfigurationWithASG(subscriptionID, resourceGroup, networkInterfaceName, ipConfigName, "test-asg")

		mockClient := mocks.NewMockInterfaceIPConfigurationsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, networkInterfaceName, ipConfigName).Return(
			armnetwork.InterfaceIPConfigurationsClientGetResponse{
				InterfaceIPConfiguration: *ipConfig,
			}, nil)

		wrapper := manual.NewNetworkNetworkInterfaceIPConfiguration(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(networkInterfaceName, ipConfigName)
		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		// Verify ASG link exists among the linked queries
		foundASG := false
		for _, lq := range sdpItem.GetLinkedItemQueries() {
			if lq.GetQuery().GetType() == azureshared.NetworkApplicationSecurityGroup.String() &&
				lq.GetQuery().GetMethod() == sdp.QueryMethod_GET &&
				lq.GetQuery().GetQuery() == "test-asg" &&
				lq.GetQuery().GetScope() == subscriptionID+"."+resourceGroup {
				foundASG = true
				break
			}
		}
		if !foundASG {
			t.Error("Expected to find ASG link in linked item queries")
		}
	})

	t.Run("GetWithFQDNs", func(t *testing.T) {
		ipConfig := createAzureIPConfigurationWithFQDNs(subscriptionID, resourceGroup, networkInterfaceName, ipConfigName, []string{"test.privatelink.blob.core.windows.net", "example.internal"})

		mockClient := mocks.NewMockInterfaceIPConfigurationsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, networkInterfaceName, ipConfigName).Return(
			armnetwork.InterfaceIPConfigurationsClientGetResponse{
				InterfaceIPConfiguration: *ipConfig,
			}, nil)

		wrapper := manual.NewNetworkNetworkInterfaceIPConfiguration(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(networkInterfaceName, ipConfigName)
		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		// Verify DNS links exist among the linked queries
		expectedFQDNs := []string{"test.privatelink.blob.core.windows.net", "example.internal"}
		for _, fqdn := range expectedFQDNs {
			found := false
			for _, lq := range sdpItem.GetLinkedItemQueries() {
				if lq.GetQuery().GetType() == stdlib.NetworkDNS.String() &&
					lq.GetQuery().GetMethod() == sdp.QueryMethod_SEARCH &&
					lq.GetQuery().GetQuery() == fqdn &&
					lq.GetQuery().GetScope() == "global" {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected to find DNS link for FQDN %s in linked item queries", fqdn)
			}
		}
	})
}

// createAzureIPConfiguration creates a mock Azure IP configuration for testing
func createAzureIPConfiguration(subscriptionID, resourceGroup, nicName, ipConfigName string) *armnetwork.InterfaceIPConfiguration {
	subnetID := "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/test-subnet"
	pipID := "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/publicIPAddresses/test-pip"
	provisioningState := armnetwork.ProvisioningStateSucceeded

	return &armnetwork.InterfaceIPConfiguration{
		ID:   new("/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/networkInterfaces/" + nicName + "/ipConfigurations/" + ipConfigName),
		Name: new(ipConfigName),
		Type: new("Microsoft.Network/networkInterfaces/ipConfigurations"),
		Properties: &armnetwork.InterfaceIPConfigurationPropertiesFormat{
			ProvisioningState:         &provisioningState,
			PrivateIPAddress:          new("10.0.0.4"),
			PrivateIPAllocationMethod: new(armnetwork.IPAllocationMethodDynamic),
			Primary:                   new(true),
			Subnet: &armnetwork.Subnet{
				ID: new(subnetID),
			},
			PublicIPAddress: &armnetwork.PublicIPAddress{
				ID: new(pipID),
			},
		},
	}
}

// createAzureIPConfigurationWithProvisioningState creates a mock IP config with a specific provisioning state
func createAzureIPConfigurationWithProvisioningState(subscriptionID, resourceGroup, nicName, ipConfigName string, state armnetwork.ProvisioningState) *armnetwork.InterfaceIPConfiguration {
	ipConfig := createAzureIPConfiguration(subscriptionID, resourceGroup, nicName, ipConfigName)
	ipConfig.Properties.ProvisioningState = &state
	return ipConfig
}

// createAzureIPConfigurationWithASG creates a mock IP config with application security groups
func createAzureIPConfigurationWithASG(subscriptionID, resourceGroup, nicName, ipConfigName, asgName string) *armnetwork.InterfaceIPConfiguration {
	ipConfig := createAzureIPConfiguration(subscriptionID, resourceGroup, nicName, ipConfigName)
	asgID := "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/applicationSecurityGroups/" + asgName
	ipConfig.Properties.ApplicationSecurityGroups = []*armnetwork.ApplicationSecurityGroup{
		{
			ID: new(asgID),
		},
	}
	return ipConfig
}

// createAzureIPConfigurationWithFQDNs creates a mock IP config with PrivateLinkConnectionProperties FQDNs
func createAzureIPConfigurationWithFQDNs(subscriptionID, resourceGroup, nicName, ipConfigName string, fqdns []string) *armnetwork.InterfaceIPConfiguration {
	ipConfig := createAzureIPConfiguration(subscriptionID, resourceGroup, nicName, ipConfigName)
	fqdnPtrs := make([]*string, len(fqdns))
	for i := range fqdns {
		fqdnPtrs[i] = new(fqdns[i])
	}
	ipConfig.Properties.PrivateLinkConnectionProperties = &armnetwork.InterfaceIPConfigurationPrivateLinkConnectionProperties{
		Fqdns: fqdnPtrs,
	}
	return ipConfig
}
