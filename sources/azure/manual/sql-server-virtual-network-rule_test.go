package manual_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/sql/armsql/v2"
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

type mockSqlServerVirtualNetworkRulePager struct {
	pages []armsql.VirtualNetworkRulesClientListByServerResponse
	index int
}

func (m *mockSqlServerVirtualNetworkRulePager) More() bool {
	return m.index < len(m.pages)
}

func (m *mockSqlServerVirtualNetworkRulePager) NextPage(ctx context.Context) (armsql.VirtualNetworkRulesClientListByServerResponse, error) {
	if m.index >= len(m.pages) {
		return armsql.VirtualNetworkRulesClientListByServerResponse{}, errors.New("no more pages")
	}
	page := m.pages[m.index]
	m.index++
	return page, nil
}

type errorSqlServerVirtualNetworkRulePager struct{}

func (e *errorSqlServerVirtualNetworkRulePager) More() bool {
	return true
}

func (e *errorSqlServerVirtualNetworkRulePager) NextPage(ctx context.Context) (armsql.VirtualNetworkRulesClientListByServerResponse, error) {
	return armsql.VirtualNetworkRulesClientListByServerResponse{}, errors.New("pager error")
}

type testSqlServerVirtualNetworkRuleClient struct {
	*mocks.MockSqlServerVirtualNetworkRuleClient
	pager clients.SqlServerVirtualNetworkRulePager
}

func (t *testSqlServerVirtualNetworkRuleClient) ListByServer(ctx context.Context, resourceGroupName, serverName string) clients.SqlServerVirtualNetworkRulePager {
	return t.pager
}

func TestSqlServerVirtualNetworkRule(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"
	serverName := "test-server"
	ruleName := "test-vnet-rule"

	t.Run("Get", func(t *testing.T) {
		rule := createAzureSqlServerVirtualNetworkRule(serverName, ruleName, "")

		mockClient := mocks.NewMockSqlServerVirtualNetworkRuleClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, serverName, ruleName).Return(
			armsql.VirtualNetworkRulesClientGetResponse{
				VirtualNetworkRule: *rule,
			}, nil)

		wrapper := manual.NewSqlServerVirtualNetworkRule(&testSqlServerVirtualNetworkRuleClient{MockSqlServerVirtualNetworkRuleClient: mockClient}, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(serverName, ruleName)
		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.SQLServerVirtualNetworkRule.String() {
			t.Errorf("Expected type %s, got %s", azureshared.SQLServerVirtualNetworkRule, sdpItem.GetType())
		}

		if sdpItem.GetUniqueAttribute() != "uniqueAttr" {
			t.Errorf("Expected unique attribute 'uniqueAttr', got %s", sdpItem.GetUniqueAttribute())
		}

		expectedUniqueAttrValue := shared.CompositeLookupKey(serverName, ruleName)
		if sdpItem.UniqueAttributeValue() != expectedUniqueAttrValue {
			t.Errorf("Expected unique attribute value %s, got %s", expectedUniqueAttrValue, sdpItem.UniqueAttributeValue())
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
					ExpectedType:   azureshared.SQLServer.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  serverName,
					ExpectedScope:  subscriptionID + "." + resourceGroup,
				},
			}
			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("Get_WithSubnetLink", func(t *testing.T) {
		subnetID := "/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/test-subnet"
		rule := createAzureSqlServerVirtualNetworkRule(serverName, ruleName, subnetID)

		mockClient := mocks.NewMockSqlServerVirtualNetworkRuleClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, serverName, ruleName).Return(
			armsql.VirtualNetworkRulesClientGetResponse{
				VirtualNetworkRule: *rule,
			}, nil)

		wrapper := manual.NewSqlServerVirtualNetworkRule(&testSqlServerVirtualNetworkRuleClient{MockSqlServerVirtualNetworkRuleClient: mockClient}, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(serverName, ruleName)
		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					ExpectedType:   azureshared.SQLServer.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  serverName,
					ExpectedScope:  subscriptionID + "." + resourceGroup,
				},
				{
					ExpectedType:   azureshared.NetworkVirtualNetwork.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-vnet",
					ExpectedScope:  subscriptionID + "." + resourceGroup,
				},
				{
					ExpectedType:   azureshared.NetworkSubnet.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey("test-vnet", "test-subnet"),
					ExpectedScope:  subscriptionID + "." + resourceGroup,
				},
			}
			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("GetWithInsufficientQueryParts", func(t *testing.T) {
		mockClient := mocks.NewMockSqlServerVirtualNetworkRuleClient(ctrl)
		wrapper := manual.NewSqlServerVirtualNetworkRule(&testSqlServerVirtualNetworkRuleClient{MockSqlServerVirtualNetworkRuleClient: mockClient}, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], serverName, true)
		if qErr == nil {
			t.Error("Expected error when providing only serverName (1 query part), but got nil")
		}
	})

	t.Run("GetWithEmptyName", func(t *testing.T) {
		mockClient := mocks.NewMockSqlServerVirtualNetworkRuleClient(ctrl)
		wrapper := manual.NewSqlServerVirtualNetworkRule(&testSqlServerVirtualNetworkRuleClient{MockSqlServerVirtualNetworkRuleClient: mockClient}, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(serverName, "")
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr == nil {
			t.Error("Expected error when virtual network rule name is empty, but got nil")
		}
	})

	t.Run("Search", func(t *testing.T) {
		rule1 := createAzureSqlServerVirtualNetworkRule(serverName, "rule1", "")
		rule2 := createAzureSqlServerVirtualNetworkRule(serverName, "rule2", "")

		mockClient := mocks.NewMockSqlServerVirtualNetworkRuleClient(ctrl)
		pager := &mockSqlServerVirtualNetworkRulePager{
			pages: []armsql.VirtualNetworkRulesClientListByServerResponse{
				{
					VirtualNetworkRuleListResult: armsql.VirtualNetworkRuleListResult{
						Value: []*armsql.VirtualNetworkRule{rule1, rule2},
					},
				},
			},
		}

		testClient := &testSqlServerVirtualNetworkRuleClient{
			MockSqlServerVirtualNetworkRuleClient: mockClient,
			pager:                                 pager,
		}
		wrapper := manual.NewSqlServerVirtualNetworkRule(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support Search operation")
		}

		items, qErr := searchable.Search(ctx, wrapper.Scopes()[0], serverName, true)
		if qErr != nil {
			t.Fatalf("Expected no error from Search, got: %v", qErr)
		}
		if len(items) != 2 {
			t.Errorf("Expected 2 items from Search, got %d", len(items))
		}
	})

	t.Run("SearchStream", func(t *testing.T) {
		rule1 := createAzureSqlServerVirtualNetworkRule(serverName, "rule1", "")

		mockClient := mocks.NewMockSqlServerVirtualNetworkRuleClient(ctrl)
		pager := &mockSqlServerVirtualNetworkRulePager{
			pages: []armsql.VirtualNetworkRulesClientListByServerResponse{
				{
					VirtualNetworkRuleListResult: armsql.VirtualNetworkRuleListResult{
						Value: []*armsql.VirtualNetworkRule{rule1},
					},
				},
			},
		}

		testClient := &testSqlServerVirtualNetworkRuleClient{
			MockSqlServerVirtualNetworkRuleClient: mockClient,
			pager:                                 pager,
		}
		wrapper := manual.NewSqlServerVirtualNetworkRule(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		searchStreamable, ok := adapter.(discovery.SearchStreamableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support SearchStream operation")
		}

		stream := discovery.NewRecordingQueryResultStream()
		searchStreamable.SearchStream(ctx, wrapper.Scopes()[0], serverName, true, stream)
		items := stream.GetItems()
		errs := stream.GetErrors()
		if len(errs) > 0 {
			t.Fatalf("Expected no errors from SearchStream, got: %v", errs)
		}
		if len(items) != 1 {
			t.Errorf("Expected 1 item from SearchStream, got %d", len(items))
		}
	})

	t.Run("SearchWithInsufficientQueryParts", func(t *testing.T) {
		mockClient := mocks.NewMockSqlServerVirtualNetworkRuleClient(ctrl)
		wrapper := manual.NewSqlServerVirtualNetworkRule(&testSqlServerVirtualNetworkRuleClient{MockSqlServerVirtualNetworkRuleClient: mockClient}, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		_, qErr := wrapper.Search(ctx, wrapper.Scopes()[0])
		if qErr == nil {
			t.Error("Expected error when providing no query parts, but got nil")
		}
	})

	t.Run("ErrorHandling_Get", func(t *testing.T) {
		expectedErr := errors.New("virtual network rule not found")

		mockClient := mocks.NewMockSqlServerVirtualNetworkRuleClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, serverName, "nonexistent-rule").Return(
			armsql.VirtualNetworkRulesClientGetResponse{}, expectedErr)

		wrapper := manual.NewSqlServerVirtualNetworkRule(&testSqlServerVirtualNetworkRuleClient{MockSqlServerVirtualNetworkRuleClient: mockClient}, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(serverName, "nonexistent-rule")
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr == nil {
			t.Error("Expected error when getting non-existent virtual network rule, but got nil")
		}
	})

	t.Run("ErrorHandling_Search", func(t *testing.T) {
		mockClient := mocks.NewMockSqlServerVirtualNetworkRuleClient(ctrl)
		errorPager := &errorSqlServerVirtualNetworkRulePager{}
		testClient := &testSqlServerVirtualNetworkRuleClient{
			MockSqlServerVirtualNetworkRuleClient: mockClient,
			pager:                                 errorPager,
		}

		wrapper := manual.NewSqlServerVirtualNetworkRule(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		_, qErr := wrapper.Search(ctx, wrapper.Scopes()[0], serverName)
		if qErr == nil {
			t.Error("Expected error from Search when pager returns error, but got nil")
		}
	})

	t.Run("InterfaceCompliance", func(t *testing.T) {
		mockClient := mocks.NewMockSqlServerVirtualNetworkRuleClient(ctrl)
		wrapper := manual.NewSqlServerVirtualNetworkRule(&testSqlServerVirtualNetworkRuleClient{MockSqlServerVirtualNetworkRuleClient: mockClient}, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		w := wrapper.(sources.Wrapper)

		permissions := w.IAMPermissions()
		if len(permissions) == 0 {
			t.Error("Expected IAMPermissions to return at least one permission")
		}
		expectedPermission := "Microsoft.Sql/servers/virtualNetworkRules/read"
		found := slices.Contains(permissions, expectedPermission)
		if !found {
			t.Errorf("Expected IAMPermissions to include %s", expectedPermission)
		}

		potentialLinks := w.PotentialLinks()
		if !potentialLinks[azureshared.SQLServer] {
			t.Error("Expected PotentialLinks to include SQLServer")
		}
		if !potentialLinks[azureshared.NetworkSubnet] {
			t.Error("Expected PotentialLinks to include NetworkSubnet")
		}
		if !potentialLinks[azureshared.NetworkVirtualNetwork] {
			t.Error("Expected PotentialLinks to include NetworkVirtualNetwork")
		}

		mappings := w.TerraformMappings()
		if len(mappings) == 0 {
			t.Error("Expected TerraformMappings to return at least one mapping")
		}
		foundMapping := false
		for _, mapping := range mappings {
			if mapping.GetTerraformQueryMap() == "azurerm_mssql_virtual_network_rule.id" {
				foundMapping = true
				break
			}
		}
		if !foundMapping {
			t.Error("Expected TerraformMappings to include 'azurerm_mssql_virtual_network_rule.id' mapping")
		}
	})
}

func createAzureSqlServerVirtualNetworkRule(serverName, ruleName, subnetID string) *armsql.VirtualNetworkRule {
	ruleID := "/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.Sql/servers/" + serverName + "/virtualNetworkRules/" + ruleName
	rule := &armsql.VirtualNetworkRule{
		Name:       &ruleName,
		ID:         &ruleID,
		Properties: &armsql.VirtualNetworkRuleProperties{},
	}
	if subnetID != "" {
		rule.Properties.VirtualNetworkSubnetID = &subnetID
	}
	return rule
}
