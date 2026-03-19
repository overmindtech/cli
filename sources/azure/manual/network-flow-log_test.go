package manual_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v9"
	"go.uber.org/mock/gomock"

	"github.com/overmindtech/cli/go/discovery"
	sdp "github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	"github.com/overmindtech/cli/sources/azure/manual"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/azure/shared/mocks"
	"github.com/overmindtech/cli/sources/shared"
)

type mockFlowLogsPager struct {
	pages []armnetwork.FlowLogsClientListResponse
	index int
}

func (m *mockFlowLogsPager) More() bool {
	return m.index < len(m.pages)
}

func (m *mockFlowLogsPager) NextPage(ctx context.Context) (armnetwork.FlowLogsClientListResponse, error) {
	if m.index >= len(m.pages) {
		return armnetwork.FlowLogsClientListResponse{}, errors.New("no more pages")
	}
	page := m.pages[m.index]
	m.index++
	return page, nil
}

type errorFlowLogsPager struct{}

func (e *errorFlowLogsPager) More() bool {
	return true
}

func (e *errorFlowLogsPager) NextPage(ctx context.Context) (armnetwork.FlowLogsClientListResponse, error) {
	return armnetwork.FlowLogsClientListResponse{}, errors.New("pager error")
}

type testFlowLogsClient struct {
	*mocks.MockFlowLogsClient
	pager clients.FlowLogsPager
}

func (t *testFlowLogsClient) NewListPager(resourceGroupName, networkWatcherName string, options *armnetwork.FlowLogsClientListOptions) clients.FlowLogsPager {
	return t.pager
}

func TestNetworkFlowLog(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"
	networkWatcherName := "test-watcher"
	flowLogName := "test-flow-log"

	t.Run("Get", func(t *testing.T) {
		flowLog := createAzureFlowLog(flowLogName, networkWatcherName, subscriptionID, resourceGroup)

		mockClient := mocks.NewMockFlowLogsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, networkWatcherName, flowLogName, nil).Return(
			armnetwork.FlowLogsClientGetResponse{
				FlowLog: *flowLog,
			}, nil)

		testClient := &testFlowLogsClient{MockFlowLogsClient: mockClient}
		wrapper := manual.NewNetworkFlowLog(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(networkWatcherName, flowLogName)
		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.NetworkFlowLog.String() {
			t.Errorf("Expected type %s, got %s", azureshared.NetworkFlowLog, sdpItem.GetType())
		}

		if sdpItem.GetUniqueAttribute() != "uniqueAttr" {
			t.Errorf("Expected unique attribute 'uniqueAttr', got %s", sdpItem.GetUniqueAttribute())
		}

		if sdpItem.UniqueAttributeValue() != shared.CompositeLookupKey(networkWatcherName, flowLogName) {
			t.Errorf("Expected unique attribute value %s, got %s", shared.CompositeLookupKey(networkWatcherName, flowLogName), sdpItem.UniqueAttributeValue())
		}

		if sdpItem.GetScope() != subscriptionID+"."+resourceGroup {
			t.Errorf("Expected scope %s, got %s", subscriptionID+"."+resourceGroup, sdpItem.GetScope())
		}

		if err := sdpItem.Validate(); err != nil {
			t.Fatalf("Expected no validation error, got: %v", err)
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					ExpectedType:   azureshared.NetworkNetworkSecurityGroup.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-nsg",
					ExpectedScope:  subscriptionID + "." + resourceGroup,
				},
				{
					ExpectedType:   azureshared.StorageAccount.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "teststorageaccount",
					ExpectedScope:  subscriptionID + "." + resourceGroup,
				},
				{
					ExpectedType:   azureshared.OperationalInsightsWorkspace.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-workspace",
					ExpectedScope:  subscriptionID + ".test-workspace-rg",
				},
				{
					ExpectedType:   azureshared.NetworkNetworkWatcher.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  networkWatcherName,
					ExpectedScope:  subscriptionID + "." + resourceGroup,
				},
				{
					ExpectedType:   azureshared.ManagedIdentityUserAssignedIdentity.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-identity",
					ExpectedScope:  subscriptionID + "." + resourceGroup,
				},
			}
			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("Get_VNetTarget", func(t *testing.T) {
		flowLog := createAzureFlowLogWithVNetTarget(flowLogName, networkWatcherName, subscriptionID, resourceGroup)

		mockClient := mocks.NewMockFlowLogsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, networkWatcherName, flowLogName, nil).Return(
			armnetwork.FlowLogsClientGetResponse{
				FlowLog: *flowLog,
			}, nil)

		testClient := &testFlowLogsClient{MockFlowLogsClient: mockClient}
		wrapper := manual.NewNetworkFlowLog(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(networkWatcherName, flowLogName)
		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		found := false
		for _, link := range sdpItem.GetLinkedItemQueries() {
			if link.GetQuery().GetType() == azureshared.NetworkVirtualNetwork.String() {
				found = true
				if link.GetQuery().GetQuery() != "test-vnet" {
					t.Errorf("Expected VNet query 'test-vnet', got %s", link.GetQuery().GetQuery())
				}
			}
		}
		if !found {
			t.Error("Expected a linked item query for VirtualNetwork, but none found")
		}
	})

	t.Run("Get_SubnetTarget", func(t *testing.T) {
		flowLog := createAzureFlowLogWithSubnetTarget(flowLogName, networkWatcherName, subscriptionID, resourceGroup)

		mockClient := mocks.NewMockFlowLogsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, networkWatcherName, flowLogName, nil).Return(
			armnetwork.FlowLogsClientGetResponse{
				FlowLog: *flowLog,
			}, nil)

		testClient := &testFlowLogsClient{MockFlowLogsClient: mockClient}
		wrapper := manual.NewNetworkFlowLog(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(networkWatcherName, flowLogName)
		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		found := false
		for _, link := range sdpItem.GetLinkedItemQueries() {
			if link.GetQuery().GetType() == azureshared.NetworkSubnet.String() {
				found = true
				expectedQuery := shared.CompositeLookupKey("test-vnet", "test-subnet")
				if link.GetQuery().GetQuery() != expectedQuery {
					t.Errorf("Expected Subnet query %s, got %s", expectedQuery, link.GetQuery().GetQuery())
				}
			}
		}
		if !found {
			t.Error("Expected a linked item query for Subnet, but none found")
		}
	})

	t.Run("Get_EmptyFlowLogName", func(t *testing.T) {
		mockClient := mocks.NewMockFlowLogsClient(ctrl)
		testClient := &testFlowLogsClient{MockFlowLogsClient: mockClient}

		wrapper := manual.NewNetworkFlowLog(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(networkWatcherName, "")
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr == nil {
			t.Error("Expected error when flow log name is empty, but got nil")
		}
	})

	t.Run("Get_EmptyNetworkWatcherName", func(t *testing.T) {
		mockClient := mocks.NewMockFlowLogsClient(ctrl)
		testClient := &testFlowLogsClient{MockFlowLogsClient: mockClient}

		wrapper := manual.NewNetworkFlowLog(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey("", flowLogName)
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr == nil {
			t.Error("Expected error when network watcher name is empty, but got nil")
		}
	})

	t.Run("Get_InsufficientQueryParts", func(t *testing.T) {
		mockClient := mocks.NewMockFlowLogsClient(ctrl)
		testClient := &testFlowLogsClient{MockFlowLogsClient: mockClient}

		wrapper := manual.NewNetworkFlowLog(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], networkWatcherName, true)
		if qErr == nil {
			t.Error("Expected error when providing insufficient query parts, but got nil")
		}
	})

	t.Run("Search", func(t *testing.T) {
		flowLog1 := createAzureFlowLog("flow-log-1", networkWatcherName, subscriptionID, resourceGroup)
		flowLog2 := createAzureFlowLog("flow-log-2", networkWatcherName, subscriptionID, resourceGroup)

		mockClient := mocks.NewMockFlowLogsClient(ctrl)
		mockPager := &mockFlowLogsPager{
			pages: []armnetwork.FlowLogsClientListResponse{
				{
					FlowLogListResult: armnetwork.FlowLogListResult{
						Value: []*armnetwork.FlowLog{flowLog1, flowLog2},
					},
				},
			},
		}

		testClient := &testFlowLogsClient{
			MockFlowLogsClient: mockClient,
			pager:              mockPager,
		}

		wrapper := manual.NewNetworkFlowLog(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support Search operation")
		}

		sdpItems, err := searchable.Search(ctx, wrapper.Scopes()[0], networkWatcherName, true)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if len(sdpItems) != 2 {
			t.Fatalf("Expected 2 items, got: %d", len(sdpItems))
		}

		for _, item := range sdpItems {
			if err := item.Validate(); err != nil {
				t.Fatalf("Expected no validation error, got: %v", err)
			}
			if item.GetType() != azureshared.NetworkFlowLog.String() {
				t.Errorf("Expected type %s, got %s", azureshared.NetworkFlowLog, item.GetType())
			}
		}
	})

	t.Run("Search_InvalidQueryParts", func(t *testing.T) {
		mockClient := mocks.NewMockFlowLogsClient(ctrl)
		testClient := &testFlowLogsClient{MockFlowLogsClient: mockClient}

		wrapper := manual.NewNetworkFlowLog(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		_, qErr := wrapper.Search(ctx, wrapper.Scopes()[0])
		if qErr == nil {
			t.Error("Expected error when providing no query parts, but got nil")
		}
	})

	t.Run("Search_FlowLogWithNilName", func(t *testing.T) {
		validFlowLog := createAzureFlowLog("valid-flow-log", networkWatcherName, subscriptionID, resourceGroup)

		mockClient := mocks.NewMockFlowLogsClient(ctrl)
		mockPager := &mockFlowLogsPager{
			pages: []armnetwork.FlowLogsClientListResponse{
				{
					FlowLogListResult: armnetwork.FlowLogListResult{
						Value: []*armnetwork.FlowLog{
							{Name: nil, ID: new("/some/id")},
							validFlowLog,
						},
					},
				},
			},
		}

		testClient := &testFlowLogsClient{
			MockFlowLogsClient: mockClient,
			pager:              mockPager,
		}

		wrapper := manual.NewNetworkFlowLog(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		searchable := adapter.(discovery.SearchableAdapter)
		sdpItems, err := searchable.Search(ctx, wrapper.Scopes()[0], networkWatcherName, true)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if len(sdpItems) != 1 {
			t.Fatalf("Expected 1 item (nil name skipped), got: %d", len(sdpItems))
		}
		if sdpItems[0].UniqueAttributeValue() != shared.CompositeLookupKey(networkWatcherName, "valid-flow-log") {
			t.Errorf("Expected unique value %s, got %s", shared.CompositeLookupKey(networkWatcherName, "valid-flow-log"), sdpItems[0].UniqueAttributeValue())
		}
	})

	t.Run("ErrorHandling_Get", func(t *testing.T) {
		expectedErr := errors.New("flow log not found")

		mockClient := mocks.NewMockFlowLogsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, networkWatcherName, "nonexistent", nil).Return(
			armnetwork.FlowLogsClientGetResponse{}, expectedErr)

		testClient := &testFlowLogsClient{MockFlowLogsClient: mockClient}
		wrapper := manual.NewNetworkFlowLog(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(networkWatcherName, "nonexistent")
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr == nil {
			t.Error("Expected error when getting non-existent flow log, but got nil")
		}
	})

	t.Run("ErrorHandling_Search", func(t *testing.T) {
		mockClient := mocks.NewMockFlowLogsClient(ctrl)
		testClient := &testFlowLogsClient{
			MockFlowLogsClient: mockClient,
			pager:              &errorFlowLogsPager{},
		}

		wrapper := manual.NewNetworkFlowLog(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		searchable := adapter.(discovery.SearchableAdapter)
		_, err := searchable.Search(ctx, wrapper.Scopes()[0], networkWatcherName, true)
		if err == nil {
			t.Error("Expected error from pager when NextPage returns an error, but got nil")
		}
	})

	t.Run("HealthMapping", func(t *testing.T) {
		tests := []struct {
			name           string
			state          armnetwork.ProvisioningState
			expectedHealth sdp.Health
		}{
			{"Succeeded", armnetwork.ProvisioningStateSucceeded, sdp.Health_HEALTH_OK},
			{"Updating", armnetwork.ProvisioningStateUpdating, sdp.Health_HEALTH_PENDING},
			{"Failed", armnetwork.ProvisioningStateFailed, sdp.Health_HEALTH_ERROR},
			{"Unknown", armnetwork.ProvisioningState("SomeOtherState"), sdp.Health_HEALTH_UNKNOWN},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				flowLog := createAzureFlowLog(flowLogName, networkWatcherName, subscriptionID, resourceGroup)
				flowLog.Properties.ProvisioningState = &tc.state

				mockClient := mocks.NewMockFlowLogsClient(ctrl)
				mockClient.EXPECT().Get(ctx, resourceGroup, networkWatcherName, flowLogName, nil).Return(
					armnetwork.FlowLogsClientGetResponse{
						FlowLog: *flowLog,
					}, nil)

				testClient := &testFlowLogsClient{MockFlowLogsClient: mockClient}
				wrapper := manual.NewNetworkFlowLog(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
				adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

				query := shared.CompositeLookupKey(networkWatcherName, flowLogName)
				sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
				if qErr != nil {
					t.Fatalf("Expected no error, got: %v", qErr)
				}

				if sdpItem.GetHealth() != tc.expectedHealth {
					t.Errorf("Expected health %s, got %s", tc.expectedHealth, sdpItem.GetHealth())
				}
			})
		}
	})

	t.Run("Get_NoLinks", func(t *testing.T) {
		flowLog := createAzureFlowLogWithoutLinks(flowLogName, networkWatcherName, subscriptionID, resourceGroup)

		mockClient := mocks.NewMockFlowLogsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, networkWatcherName, flowLogName, nil).Return(
			armnetwork.FlowLogsClientGetResponse{
				FlowLog: *flowLog,
			}, nil)

		testClient := &testFlowLogsClient{MockFlowLogsClient: mockClient}
		wrapper := manual.NewNetworkFlowLog(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(networkWatcherName, flowLogName)
		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		// Should only have the parent NetworkWatcher link
		if len(sdpItem.GetLinkedItemQueries()) != 1 {
			t.Errorf("Expected 1 linked query (parent only), got %d", len(sdpItem.GetLinkedItemQueries()))
		}
		if sdpItem.GetLinkedItemQueries()[0].GetQuery().GetType() != azureshared.NetworkNetworkWatcher.String() {
			t.Errorf("Expected parent link to NetworkWatcher, got %s", sdpItem.GetLinkedItemQueries()[0].GetQuery().GetType())
		}
	})
}

func createAzureFlowLog(name, networkWatcherName, subscriptionID, resourceGroup string) *armnetwork.FlowLog {
	provisioningState := armnetwork.ProvisioningStateSucceeded
	enabled := true
	nsgID := "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/networkSecurityGroups/test-nsg"
	storageID := "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Storage/storageAccounts/teststorageaccount"
	workspaceResourceID := "/subscriptions/" + subscriptionID + "/resourceGroups/test-workspace-rg/providers/Microsoft.OperationalInsights/workspaces/test-workspace"
	identityID := "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.ManagedIdentity/userAssignedIdentities/test-identity"

	return &armnetwork.FlowLog{
		ID:       new("/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/networkWatchers/" + networkWatcherName + "/flowLogs/" + name),
		Name:     &name,
		Type:     new("Microsoft.Network/networkWatchers/flowLogs"),
		Location: new("eastus"),
		Tags: map[string]*string{
			"env": new("test"),
		},
		Identity: &armnetwork.ManagedServiceIdentity{
			UserAssignedIdentities: map[string]*armnetwork.Components1Jq1T4ISchemasManagedserviceidentityPropertiesUserassignedidentitiesAdditionalproperties{
				identityID: {},
			},
		},
		Properties: &armnetwork.FlowLogPropertiesFormat{
			TargetResourceID:  &nsgID,
			StorageID:         &storageID,
			Enabled:           &enabled,
			ProvisioningState: &provisioningState,
			FlowAnalyticsConfiguration: &armnetwork.TrafficAnalyticsProperties{
				NetworkWatcherFlowAnalyticsConfiguration: &armnetwork.TrafficAnalyticsConfigurationProperties{
					Enabled:             &enabled,
					WorkspaceResourceID: &workspaceResourceID,
				},
			},
			RetentionPolicy: &armnetwork.RetentionPolicyParameters{
				Enabled: &enabled,
				Days:    new(int32(90)),
			},
		},
	}
}

func createAzureFlowLogWithVNetTarget(name, networkWatcherName, subscriptionID, resourceGroup string) *armnetwork.FlowLog {
	provisioningState := armnetwork.ProvisioningStateSucceeded
	enabled := true
	vnetID := "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/virtualNetworks/test-vnet"
	storageID := "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Storage/storageAccounts/teststorageaccount"

	return &armnetwork.FlowLog{
		ID:       new("/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/networkWatchers/" + networkWatcherName + "/flowLogs/" + name),
		Name:     &name,
		Type:     new("Microsoft.Network/networkWatchers/flowLogs"),
		Location: new("eastus"),
		Properties: &armnetwork.FlowLogPropertiesFormat{
			TargetResourceID:  &vnetID,
			StorageID:         &storageID,
			Enabled:           &enabled,
			ProvisioningState: &provisioningState,
		},
	}
}

func createAzureFlowLogWithSubnetTarget(name, networkWatcherName, subscriptionID, resourceGroup string) *armnetwork.FlowLog {
	provisioningState := armnetwork.ProvisioningStateSucceeded
	enabled := true
	subnetID := "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/test-subnet"
	storageID := "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Storage/storageAccounts/teststorageaccount"

	return &armnetwork.FlowLog{
		ID:       new("/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/networkWatchers/" + networkWatcherName + "/flowLogs/" + name),
		Name:     &name,
		Type:     new("Microsoft.Network/networkWatchers/flowLogs"),
		Location: new("eastus"),
		Properties: &armnetwork.FlowLogPropertiesFormat{
			TargetResourceID:  &subnetID,
			StorageID:         &storageID,
			Enabled:           &enabled,
			ProvisioningState: &provisioningState,
		},
	}
}

func createAzureFlowLogWithoutLinks(name, networkWatcherName, subscriptionID, resourceGroup string) *armnetwork.FlowLog {
	return &armnetwork.FlowLog{
		ID:       new("/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/networkWatchers/" + networkWatcherName + "/flowLogs/" + name),
		Name:     &name,
		Type:     new("Microsoft.Network/networkWatchers/flowLogs"),
		Location: new("eastus"),
	}
}
