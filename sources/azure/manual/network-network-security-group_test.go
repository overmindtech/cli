package manual_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"
	"go.uber.org/mock/gomock"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/manual"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/azure/shared/mocks"
	"github.com/overmindtech/cli/sources/shared"
)

func TestNetworkNetworkSecurityGroup(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"

	t.Run("Get", func(t *testing.T) {
		nsgName := "test-nsg"
		nsg := createAzureNetworkSecurityGroup(nsgName)

		mockClient := mocks.NewMockNetworkSecurityGroupsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, nsgName, nil).Return(
			armnetwork.SecurityGroupsClientGetResponse{
				SecurityGroup: *nsg,
			}, nil)

		wrapper := manual.NewNetworkNetworkSecurityGroup(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], nsgName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.NetworkNetworkSecurityGroup.String() {
			t.Errorf("Expected type %s, got %s", azureshared.NetworkNetworkSecurityGroup, sdpItem.GetType())
		}

		if sdpItem.GetUniqueAttribute() != "name" {
			t.Errorf("Expected unique attribute 'name', got %s", sdpItem.GetUniqueAttribute())
		}

		if sdpItem.UniqueAttributeValue() != nsgName {
			t.Errorf("Expected unique attribute value %s, got %s", nsgName, sdpItem.UniqueAttributeValue())
		}

		if sdpItem.GetTags()["env"] != "test" {
			t.Errorf("Expected tag 'env=test', got: %v", sdpItem.GetTags()["env"])
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					// SecurityRule link
					ExpectedType:   azureshared.NetworkSecurityRule.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey(nsgName, "test-security-rule"),
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					// DefaultSecurityRule link
					ExpectedType:   azureshared.NetworkDefaultSecurityRule.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey(nsgName, "AllowVnetInBound"),
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					// Subnet link
					ExpectedType:   azureshared.NetworkSubnet.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey("test-vnet", "test-subnet"),
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					// NetworkInterface link
					ExpectedType:   azureshared.NetworkNetworkInterface.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-nic",
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  false,
						Out: true,
					},
				},
				{
					// ApplicationSecurityGroup link (from SecurityRule Source)
					ExpectedType:   azureshared.NetworkApplicationSecurityGroup.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-asg-source",
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					// ApplicationSecurityGroup link (from SecurityRule Destination)
					ExpectedType:   azureshared.NetworkApplicationSecurityGroup.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-asg-dest",
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					// ApplicationSecurityGroup link (from DefaultSecurityRule Source)
					ExpectedType:   azureshared.NetworkApplicationSecurityGroup.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-asg-default-source",
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

	t.Run("Get_InvalidQueryParts", func(t *testing.T) {
		mockClient := mocks.NewMockNetworkSecurityGroupsClient(ctrl)

		wrapper := manual.NewNetworkNetworkSecurityGroup(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		// Test with empty string name - Get will still be called with empty string
		// and Azure will return an error
		mockClient.EXPECT().Get(ctx, resourceGroup, "", nil).Return(
			armnetwork.SecurityGroupsClientGetResponse{}, errors.New("network security group not found"))

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "", true)
		if qErr == nil {
			t.Error("Expected error when getting network security group with empty name, but got nil")
		}
	})

	t.Run("Get_WithNilName", func(t *testing.T) {
		nsg := &armnetwork.SecurityGroup{
			Name:     nil, // NSG with nil name should cause an error
			Location: to.Ptr("eastus"),
			Tags: map[string]*string{
				"env": to.Ptr("test"),
			},
		}

		mockClient := mocks.NewMockNetworkSecurityGroupsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, "test-nsg", nil).Return(
			armnetwork.SecurityGroupsClientGetResponse{
				SecurityGroup: *nsg,
			}, nil)

		wrapper := manual.NewNetworkNetworkSecurityGroup(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "test-nsg", true)
		if qErr == nil {
			t.Error("Expected error when network security group has nil name, but got nil")
		}
	})

	t.Run("List", func(t *testing.T) {
		nsg1 := createAzureNetworkSecurityGroup("test-nsg-1")
		nsg2 := createAzureNetworkSecurityGroup("test-nsg-2")

		mockClient := mocks.NewMockNetworkSecurityGroupsClient(ctrl)
		mockPager := NewMockNetworkSecurityGroupsPager(ctrl)

		// Setup pager expectations
		gomock.InOrder(
			mockPager.EXPECT().More().Return(true),
			mockPager.EXPECT().NextPage(ctx).Return(
				armnetwork.SecurityGroupsClientListResponse{
					SecurityGroupListResult: armnetwork.SecurityGroupListResult{
						Value: []*armnetwork.SecurityGroup{nsg1, nsg2},
					},
				}, nil),
			mockPager.EXPECT().More().Return(false),
		)

		mockClient.EXPECT().List(ctx, resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewNetworkNetworkSecurityGroup(mockClient, subscriptionID, resourceGroup)
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

			if item.GetType() != azureshared.NetworkNetworkSecurityGroup.String() {
				t.Fatalf("Expected type %s, got: %s", azureshared.NetworkNetworkSecurityGroup, item.GetType())
			}
		}
	})

	t.Run("List_WithNilName", func(t *testing.T) {
		// Create NSG with nil name to test error handling
		nsg1 := createAzureNetworkSecurityGroup("test-nsg-1")
		nsg2 := &armnetwork.SecurityGroup{
			Name:     nil, // NSG with nil name should be skipped
			Location: to.Ptr("eastus"),
			Tags: map[string]*string{
				"env": to.Ptr("test"),
			},
		}

		mockClient := mocks.NewMockNetworkSecurityGroupsClient(ctrl)
		mockPager := NewMockNetworkSecurityGroupsPager(ctrl)

		// Setup pager expectations
		gomock.InOrder(
			mockPager.EXPECT().More().Return(true),
			mockPager.EXPECT().NextPage(ctx).Return(
				armnetwork.SecurityGroupsClientListResponse{
					SecurityGroupListResult: armnetwork.SecurityGroupListResult{
						Value: []*armnetwork.SecurityGroup{nsg1, nsg2},
					},
				}, nil),
			mockPager.EXPECT().More().Return(false),
		)

		mockClient.EXPECT().List(ctx, resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewNetworkNetworkSecurityGroup(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		listable, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support List operation")
		}

		sdpItems, err := listable.List(ctx, wrapper.Scopes()[0], true)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		// Should only return 1 item (nsg1), nsg2 should be skipped
		if len(sdpItems) != 1 {
			t.Fatalf("Expected 1 item (nil name should be skipped), got: %d", len(sdpItems))
		}
	})

	t.Run("List_ErrorHandling", func(t *testing.T) {
		expectedErr := errors.New("failed to list network security groups")

		mockClient := mocks.NewMockNetworkSecurityGroupsClient(ctrl)
		mockPager := NewMockNetworkSecurityGroupsPager(ctrl)

		// Setup pager to return error on NextPage
		gomock.InOrder(
			mockPager.EXPECT().More().Return(true),
			mockPager.EXPECT().NextPage(ctx).Return(
				armnetwork.SecurityGroupsClientListResponse{}, expectedErr),
		)

		mockClient.EXPECT().List(ctx, resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewNetworkNetworkSecurityGroup(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		listable, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support List operation")
		}

		_, err := listable.List(ctx, wrapper.Scopes()[0], true)
		if err == nil {
			t.Error("Expected error when listing network security groups fails, but got nil")
		}
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		expectedErr := errors.New("network security group not found")

		mockClient := mocks.NewMockNetworkSecurityGroupsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, "nonexistent-nsg", nil).Return(
			armnetwork.SecurityGroupsClientGetResponse{}, expectedErr)

		wrapper := manual.NewNetworkNetworkSecurityGroup(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "nonexistent-nsg", true)
		if qErr == nil {
			t.Error("Expected error when getting non-existent network security group, but got nil")
		}
	})

	t.Run("CrossResourceGroupLinks", func(t *testing.T) {
		// Test NSG with subnet and NIC in different resource groups
		nsgName := "test-nsg"
		otherResourceGroup := "other-rg"
		otherSubscriptionID := "other-subscription"

		nsg := &armnetwork.SecurityGroup{
			Name:     to.Ptr(nsgName),
			Location: to.Ptr("eastus"),
			Tags: map[string]*string{
				"env": to.Ptr("test"),
			},
			Properties: &armnetwork.SecurityGroupPropertiesFormat{
				Subnets: []*armnetwork.Subnet{
					{
						ID: to.Ptr("/subscriptions/" + otherSubscriptionID + "/resourceGroups/" + otherResourceGroup + "/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/test-subnet"),
					},
				},
				NetworkInterfaces: []*armnetwork.Interface{
					{
						ID: to.Ptr("/subscriptions/" + otherSubscriptionID + "/resourceGroups/" + otherResourceGroup + "/providers/Microsoft.Network/networkInterfaces/test-nic"),
					},
				},
			},
		}

		mockClient := mocks.NewMockNetworkSecurityGroupsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, nsgName, nil).Return(
			armnetwork.SecurityGroupsClientGetResponse{
				SecurityGroup: *nsg,
			}, nil)

		wrapper := manual.NewNetworkNetworkSecurityGroup(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], nsgName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		// Check that subnet link uses the correct scope
		foundSubnetLink := false
		for _, link := range sdpItem.GetLinkedItemQueries() {
			if link.GetQuery().GetType() == azureshared.NetworkSubnet.String() {
				foundSubnetLink = true
				expectedScope := otherSubscriptionID + "." + otherResourceGroup
				if link.GetQuery().GetScope() != expectedScope {
					t.Errorf("Expected subnet scope %s, got %s", expectedScope, link.GetQuery().GetScope())
				}
			}
		}
		if !foundSubnetLink {
			t.Error("Expected to find subnet link")
		}

		// Check that NIC link uses the correct scope
		foundNICLink := false
		for _, link := range sdpItem.GetLinkedItemQueries() {
			if link.GetQuery().GetType() == azureshared.NetworkNetworkInterface.String() {
				foundNICLink = true
				expectedScope := otherSubscriptionID + "." + otherResourceGroup
				if link.GetQuery().GetScope() != expectedScope {
					t.Errorf("Expected NIC scope %s, got %s", expectedScope, link.GetQuery().GetScope())
				}
			}
		}
		if !foundNICLink {
			t.Error("Expected to find NIC link")
		}
	})

	t.Run("InterfaceCompliance", func(t *testing.T) {
		mockClient := mocks.NewMockNetworkSecurityGroupsClient(ctrl)
		wrapper := manual.NewNetworkNetworkSecurityGroup(mockClient, subscriptionID, resourceGroup)

		// Verify wrapper implements ListableWrapper interface
		var _ = wrapper

		// Cast to sources.Wrapper to access interface methods
		w := wrapper.(sources.Wrapper)

		// Verify IAMPermissions
		permissions := w.IAMPermissions()
		if len(permissions) == 0 {
			t.Error("Expected IAMPermissions to return at least one permission")
		}
		expectedPermission := "Microsoft.Network/networkSecurityGroups/read"
		found := false
		for _, perm := range permissions {
			if perm == expectedPermission {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected IAMPermissions to include %s", expectedPermission)
		}

		// Verify PotentialLinks
		potentialLinks := w.PotentialLinks()
		if len(potentialLinks) == 0 {
			t.Error("Expected PotentialLinks to return at least one link")
		}
		expectedLinks := []shared.ItemType{
			azureshared.NetworkSecurityRule,
			azureshared.NetworkDefaultSecurityRule,
			azureshared.NetworkSubnet,
			azureshared.NetworkNetworkInterface,
			azureshared.NetworkApplicationSecurityGroup,
			azureshared.NetworkIPGroup,
		}
		for _, expectedLink := range expectedLinks {
			if !potentialLinks[expectedLink] {
				t.Errorf("Expected PotentialLinks to include %s", expectedLink)
			}
		}

		// Verify TerraformMappings
		mappings := w.TerraformMappings()
		if len(mappings) == 0 {
			t.Error("Expected TerraformMappings to return at least one mapping")
		}
		foundMapping := false
		for _, mapping := range mappings {
			if mapping.GetTerraformQueryMap() == "azurerm_network_security_group.name" {
				foundMapping = true
				break
			}
		}
		if !foundMapping {
			t.Error("Expected TerraformMappings to include 'azurerm_network_security_group.name' mapping")
		}

		// Verify PredefinedRole
		// PredefinedRole is available on the wrapper, not the adapter
		role := wrapper.(interface{ PredefinedRole() string }).PredefinedRole()
		if role != "Reader" {
			t.Errorf("Expected PredefinedRole to be 'Reader', got %s", role)
		}
	})
}

// MockNetworkSecurityGroupsPager is a simple mock for NetworkSecurityGroupsPager
type MockNetworkSecurityGroupsPager struct {
	ctrl     *gomock.Controller
	recorder *MockNetworkSecurityGroupsPagerMockRecorder
}

type MockNetworkSecurityGroupsPagerMockRecorder struct {
	mock *MockNetworkSecurityGroupsPager
}

func NewMockNetworkSecurityGroupsPager(ctrl *gomock.Controller) *MockNetworkSecurityGroupsPager {
	mock := &MockNetworkSecurityGroupsPager{ctrl: ctrl}
	mock.recorder = &MockNetworkSecurityGroupsPagerMockRecorder{mock}
	return mock
}

func (m *MockNetworkSecurityGroupsPager) EXPECT() *MockNetworkSecurityGroupsPagerMockRecorder {
	return m.recorder
}

func (m *MockNetworkSecurityGroupsPager) More() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "More")
	ret0, _ := ret[0].(bool)
	return ret0
}

func (mr *MockNetworkSecurityGroupsPagerMockRecorder) More() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "More", reflect.TypeOf((*MockNetworkSecurityGroupsPager)(nil).More))
}

func (m *MockNetworkSecurityGroupsPager) NextPage(ctx context.Context) (armnetwork.SecurityGroupsClientListResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NextPage", ctx)
	ret0, _ := ret[0].(armnetwork.SecurityGroupsClientListResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockNetworkSecurityGroupsPagerMockRecorder) NextPage(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NextPage", reflect.TypeOf((*MockNetworkSecurityGroupsPager)(nil).NextPage), ctx)
}

// createAzureNetworkSecurityGroup creates a mock Azure network security group for testing
func createAzureNetworkSecurityGroup(nsgName string) *armnetwork.SecurityGroup {
	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"

	return &armnetwork.SecurityGroup{
		Name:     to.Ptr(nsgName),
		Location: to.Ptr("eastus"),
		Tags: map[string]*string{
			"env":     to.Ptr("test"),
			"project": to.Ptr("testing"),
		},
		Properties: &armnetwork.SecurityGroupPropertiesFormat{
			// SecurityRules (child resources)
			SecurityRules: []*armnetwork.SecurityRule{
				{
					Name: to.Ptr("test-security-rule"),
					Properties: &armnetwork.SecurityRulePropertiesFormat{
						Priority:  to.Ptr(int32(1000)),
						Direction: to.Ptr(armnetwork.SecurityRuleDirectionInbound),
						Access:    to.Ptr(armnetwork.SecurityRuleAccessAllow),
						SourceApplicationSecurityGroups: []*armnetwork.ApplicationSecurityGroup{
							{
								ID: to.Ptr("/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/applicationSecurityGroups/test-asg-source"),
							},
						},
						DestinationApplicationSecurityGroups: []*armnetwork.ApplicationSecurityGroup{
							{
								ID: to.Ptr("/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/applicationSecurityGroups/test-asg-dest"),
							},
						},
					},
				},
			},
			// DefaultSecurityRules (child resources)
			DefaultSecurityRules: []*armnetwork.SecurityRule{
				{
					Name: to.Ptr("AllowVnetInBound"),
					Properties: &armnetwork.SecurityRulePropertiesFormat{
						Priority:  to.Ptr(int32(65000)),
						Direction: to.Ptr(armnetwork.SecurityRuleDirectionInbound),
						Access:    to.Ptr(armnetwork.SecurityRuleAccessAllow),
						SourceApplicationSecurityGroups: []*armnetwork.ApplicationSecurityGroup{
							{
								ID: to.Ptr("/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/applicationSecurityGroups/test-asg-default-source"),
							},
						},
					},
				},
			},
			// Subnets (external resources)
			Subnets: []*armnetwork.Subnet{
				{
					ID: to.Ptr("/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/test-subnet"),
				},
			},
			// NetworkInterfaces (external resources)
			NetworkInterfaces: []*armnetwork.Interface{
				{
					ID: to.Ptr("/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/networkInterfaces/test-nic"),
				},
			},
		},
	}
}
