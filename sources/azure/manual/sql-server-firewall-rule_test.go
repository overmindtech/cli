package manual_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
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
	"github.com/overmindtech/cli/sources/stdlib"
)

type mockSqlServerFirewallRulePager struct {
	pages []armsql.FirewallRulesClientListByServerResponse
	index int
}

func (m *mockSqlServerFirewallRulePager) More() bool {
	return m.index < len(m.pages)
}

func (m *mockSqlServerFirewallRulePager) NextPage(ctx context.Context) (armsql.FirewallRulesClientListByServerResponse, error) {
	if m.index >= len(m.pages) {
		return armsql.FirewallRulesClientListByServerResponse{}, errors.New("no more pages")
	}
	page := m.pages[m.index]
	m.index++
	return page, nil
}

type errorSqlServerFirewallRulePager struct{}

func (e *errorSqlServerFirewallRulePager) More() bool {
	return true
}

func (e *errorSqlServerFirewallRulePager) NextPage(ctx context.Context) (armsql.FirewallRulesClientListByServerResponse, error) {
	return armsql.FirewallRulesClientListByServerResponse{}, errors.New("pager error")
}

type testSqlServerFirewallRuleClient struct {
	*mocks.MockSqlServerFirewallRuleClient
	pager clients.SqlServerFirewallRulePager
}

func (t *testSqlServerFirewallRuleClient) ListByServer(ctx context.Context, resourceGroupName, serverName string) clients.SqlServerFirewallRulePager {
	return t.pager
}

func TestSqlServerFirewallRule(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"
	serverName := "test-server"
	firewallRuleName := "test-rule"

	t.Run("Get", func(t *testing.T) {
		rule := createAzureSqlServerFirewallRule(serverName, firewallRuleName)

		mockClient := mocks.NewMockSqlServerFirewallRuleClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, serverName, firewallRuleName).Return(
			armsql.FirewallRulesClientGetResponse{
				FirewallRule: *rule,
			}, nil)

		wrapper := manual.NewSqlServerFirewallRule(&testSqlServerFirewallRuleClient{MockSqlServerFirewallRuleClient: mockClient}, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(serverName, firewallRuleName)
		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.SQLServerFirewallRule.String() {
			t.Errorf("Expected type %s, got %s", azureshared.SQLServerFirewallRule, sdpItem.GetType())
		}

		if sdpItem.GetUniqueAttribute() != "uniqueAttr" {
			t.Errorf("Expected unique attribute 'uniqueAttr', got %s", sdpItem.GetUniqueAttribute())
		}

		expectedUniqueAttrValue := shared.CompositeLookupKey(serverName, firewallRuleName)
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
				{
					ExpectedType:   stdlib.NetworkIP.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "0.0.0.0",
					ExpectedScope:  "global",
				},
				{
					ExpectedType:   stdlib.NetworkIP.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "255.255.255.255",
					ExpectedScope:  "global",
				},
			}
			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("GetWithInsufficientQueryParts", func(t *testing.T) {
		mockClient := mocks.NewMockSqlServerFirewallRuleClient(ctrl)
		wrapper := manual.NewSqlServerFirewallRule(&testSqlServerFirewallRuleClient{MockSqlServerFirewallRuleClient: mockClient}, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], serverName, true)
		if qErr == nil {
			t.Error("Expected error when providing only serverName (1 query part), but got nil")
		}
	})

	t.Run("GetWithEmptyName", func(t *testing.T) {
		mockClient := mocks.NewMockSqlServerFirewallRuleClient(ctrl)
		wrapper := manual.NewSqlServerFirewallRule(&testSqlServerFirewallRuleClient{MockSqlServerFirewallRuleClient: mockClient}, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(serverName, "")
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr == nil {
			t.Error("Expected error when firewall rule name is empty, but got nil")
		}
	})

	t.Run("Search", func(t *testing.T) {
		rule1 := createAzureSqlServerFirewallRule(serverName, "rule1")
		rule2 := createAzureSqlServerFirewallRule(serverName, "rule2")

		mockClient := mocks.NewMockSqlServerFirewallRuleClient(ctrl)
		pager := &mockSqlServerFirewallRulePager{
			pages: []armsql.FirewallRulesClientListByServerResponse{
				{
					FirewallRuleListResult: armsql.FirewallRuleListResult{
						Value: []*armsql.FirewallRule{rule1, rule2},
					},
				},
			},
		}

		testClient := &testSqlServerFirewallRuleClient{
			MockSqlServerFirewallRuleClient: mockClient,
			pager:                          pager,
		}
		wrapper := manual.NewSqlServerFirewallRule(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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
		rule1 := createAzureSqlServerFirewallRule(serverName, "rule1")

		mockClient := mocks.NewMockSqlServerFirewallRuleClient(ctrl)
		pager := &mockSqlServerFirewallRulePager{
			pages: []armsql.FirewallRulesClientListByServerResponse{
				{
					FirewallRuleListResult: armsql.FirewallRuleListResult{
						Value: []*armsql.FirewallRule{rule1},
					},
				},
			},
		}

		testClient := &testSqlServerFirewallRuleClient{
			MockSqlServerFirewallRuleClient: mockClient,
			pager:                          pager,
		}
		wrapper := manual.NewSqlServerFirewallRule(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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
		mockClient := mocks.NewMockSqlServerFirewallRuleClient(ctrl)
		wrapper := manual.NewSqlServerFirewallRule(&testSqlServerFirewallRuleClient{MockSqlServerFirewallRuleClient: mockClient}, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		_, qErr := wrapper.Search(ctx, wrapper.Scopes()[0])
		if qErr == nil {
			t.Error("Expected error when providing no query parts, but got nil")
		}
	})

	t.Run("ErrorHandling_Get", func(t *testing.T) {
		expectedErr := errors.New("firewall rule not found")

		mockClient := mocks.NewMockSqlServerFirewallRuleClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, serverName, "nonexistent-rule").Return(
			armsql.FirewallRulesClientGetResponse{}, expectedErr)

		wrapper := manual.NewSqlServerFirewallRule(&testSqlServerFirewallRuleClient{MockSqlServerFirewallRuleClient: mockClient}, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(serverName, "nonexistent-rule")
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr == nil {
			t.Error("Expected error when getting non-existent firewall rule, but got nil")
		}
	})

	t.Run("ErrorHandling_Search", func(t *testing.T) {
		mockClient := mocks.NewMockSqlServerFirewallRuleClient(ctrl)
		errorPager := &errorSqlServerFirewallRulePager{}
		testClient := &testSqlServerFirewallRuleClient{
			MockSqlServerFirewallRuleClient: mockClient,
			pager:                          errorPager,
		}

		wrapper := manual.NewSqlServerFirewallRule(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		_, qErr := wrapper.Search(ctx, wrapper.Scopes()[0], serverName)
		if qErr == nil {
			t.Error("Expected error from Search when pager returns error, but got nil")
		}
	})

	t.Run("InterfaceCompliance", func(t *testing.T) {
		mockClient := mocks.NewMockSqlServerFirewallRuleClient(ctrl)
		wrapper := manual.NewSqlServerFirewallRule(&testSqlServerFirewallRuleClient{MockSqlServerFirewallRuleClient: mockClient}, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		w := wrapper.(sources.Wrapper)

		permissions := w.IAMPermissions()
		if len(permissions) == 0 {
			t.Error("Expected IAMPermissions to return at least one permission")
		}
		expectedPermission := "Microsoft.Sql/servers/firewallRules/read"
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

		potentialLinks := w.PotentialLinks()
		if !potentialLinks[azureshared.SQLServer] {
			t.Error("Expected PotentialLinks to include SQLServer")
		}
		if !potentialLinks[stdlib.NetworkIP] {
			t.Error("Expected PotentialLinks to include stdlib.NetworkIP")
		}

		mappings := w.TerraformMappings()
		if len(mappings) == 0 {
			t.Error("Expected TerraformMappings to return at least one mapping")
		}
		foundMapping := false
		for _, mapping := range mappings {
			if mapping.GetTerraformQueryMap() == "azurerm_mssql_firewall_rule.id" {
				foundMapping = true
				break
			}
		}
		if !foundMapping {
			t.Error("Expected TerraformMappings to include 'azurerm_mssql_firewall_rule.id' mapping")
		}
	})
}

func createAzureSqlServerFirewallRule(serverName, firewallRuleName string) *armsql.FirewallRule {
	ruleID := "/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.Sql/servers/" + serverName + "/firewallRules/" + firewallRuleName
	return &armsql.FirewallRule{
		Name: to.Ptr(firewallRuleName),
		ID:   to.Ptr(ruleID),
		Properties: &armsql.ServerFirewallRuleProperties{
			StartIPAddress: to.Ptr("0.0.0.0"),
			EndIPAddress:   to.Ptr("255.255.255.255"),
		},
	}
}
