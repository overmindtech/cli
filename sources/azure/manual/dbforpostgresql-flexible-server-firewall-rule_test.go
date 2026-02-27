package manual_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/postgresql/armpostgresqlflexibleservers/v5"
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

type mockPostgreSQLFlexibleServerFirewallRulePager struct {
	pages []armpostgresqlflexibleservers.FirewallRulesClientListByServerResponse
	index int
}

func (m *mockPostgreSQLFlexibleServerFirewallRulePager) More() bool {
	return m.index < len(m.pages)
}

func (m *mockPostgreSQLFlexibleServerFirewallRulePager) NextPage(ctx context.Context) (armpostgresqlflexibleservers.FirewallRulesClientListByServerResponse, error) {
	if m.index >= len(m.pages) {
		return armpostgresqlflexibleservers.FirewallRulesClientListByServerResponse{}, errors.New("no more pages")
	}
	page := m.pages[m.index]
	m.index++
	return page, nil
}

type errorPostgreSQLFlexibleServerFirewallRulePager struct{}

func (e *errorPostgreSQLFlexibleServerFirewallRulePager) More() bool {
	return true
}

func (e *errorPostgreSQLFlexibleServerFirewallRulePager) NextPage(ctx context.Context) (armpostgresqlflexibleservers.FirewallRulesClientListByServerResponse, error) {
	return armpostgresqlflexibleservers.FirewallRulesClientListByServerResponse{}, errors.New("pager error")
}

type testPostgreSQLFlexibleServerFirewallRuleClient struct {
	*mocks.MockPostgreSQLFlexibleServerFirewallRuleClient
	pager clients.PostgreSQLFlexibleServerFirewallRulePager
}

func (t *testPostgreSQLFlexibleServerFirewallRuleClient) ListByServer(ctx context.Context, resourceGroupName, serverName string) clients.PostgreSQLFlexibleServerFirewallRulePager {
	return t.pager
}

func TestDBforPostgreSQLFlexibleServerFirewallRule(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"
	serverName := "test-server"
	firewallRuleName := "test-rule"

	t.Run("Get", func(t *testing.T) {
		rule := createAzurePostgreSQLFlexibleServerFirewallRule(serverName, firewallRuleName)

		mockClient := mocks.NewMockPostgreSQLFlexibleServerFirewallRuleClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, serverName, firewallRuleName).Return(
			armpostgresqlflexibleservers.FirewallRulesClientGetResponse{
				FirewallRule: *rule,
			}, nil)

		wrapper := manual.NewDBforPostgreSQLFlexibleServerFirewallRule(&testPostgreSQLFlexibleServerFirewallRuleClient{MockPostgreSQLFlexibleServerFirewallRuleClient: mockClient}, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(serverName, firewallRuleName)
		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.DBforPostgreSQLFlexibleServerFirewallRule.String() {
			t.Errorf("Expected type %s, got %s", azureshared.DBforPostgreSQLFlexibleServerFirewallRule, sdpItem.GetType())
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
					ExpectedType:   azureshared.DBforPostgreSQLFlexibleServer.String(),
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
		mockClient := mocks.NewMockPostgreSQLFlexibleServerFirewallRuleClient(ctrl)
		wrapper := manual.NewDBforPostgreSQLFlexibleServerFirewallRule(&testPostgreSQLFlexibleServerFirewallRuleClient{MockPostgreSQLFlexibleServerFirewallRuleClient: mockClient}, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], serverName, true)
		if qErr == nil {
			t.Error("Expected error when providing only serverName (1 query part), but got nil")
		}
	})

	t.Run("GetWithEmptyName", func(t *testing.T) {
		mockClient := mocks.NewMockPostgreSQLFlexibleServerFirewallRuleClient(ctrl)
		wrapper := manual.NewDBforPostgreSQLFlexibleServerFirewallRule(&testPostgreSQLFlexibleServerFirewallRuleClient{MockPostgreSQLFlexibleServerFirewallRuleClient: mockClient}, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(serverName, "")
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr == nil {
			t.Error("Expected error when firewall rule name is empty, but got nil")
		}
	})

	t.Run("Search", func(t *testing.T) {
		rule1 := createAzurePostgreSQLFlexibleServerFirewallRule(serverName, "rule1")
		rule2 := createAzurePostgreSQLFlexibleServerFirewallRule(serverName, "rule2")

		mockClient := mocks.NewMockPostgreSQLFlexibleServerFirewallRuleClient(ctrl)
		pager := &mockPostgreSQLFlexibleServerFirewallRulePager{
			pages: []armpostgresqlflexibleservers.FirewallRulesClientListByServerResponse{
				{
					FirewallRuleList: armpostgresqlflexibleservers.FirewallRuleList{
						Value: []*armpostgresqlflexibleservers.FirewallRule{rule1, rule2},
					},
				},
			},
		}

		testClient := &testPostgreSQLFlexibleServerFirewallRuleClient{
			MockPostgreSQLFlexibleServerFirewallRuleClient: mockClient,
			pager: pager,
		}
		wrapper := manual.NewDBforPostgreSQLFlexibleServerFirewallRule(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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
		rule1 := createAzurePostgreSQLFlexibleServerFirewallRule(serverName, "rule1")

		mockClient := mocks.NewMockPostgreSQLFlexibleServerFirewallRuleClient(ctrl)
		pager := &mockPostgreSQLFlexibleServerFirewallRulePager{
			pages: []armpostgresqlflexibleservers.FirewallRulesClientListByServerResponse{
				{
					FirewallRuleList: armpostgresqlflexibleservers.FirewallRuleList{
						Value: []*armpostgresqlflexibleservers.FirewallRule{rule1},
					},
				},
			},
		}

		testClient := &testPostgreSQLFlexibleServerFirewallRuleClient{
			MockPostgreSQLFlexibleServerFirewallRuleClient: mockClient,
			pager: pager,
		}
		wrapper := manual.NewDBforPostgreSQLFlexibleServerFirewallRule(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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
		mockClient := mocks.NewMockPostgreSQLFlexibleServerFirewallRuleClient(ctrl)
		wrapper := manual.NewDBforPostgreSQLFlexibleServerFirewallRule(&testPostgreSQLFlexibleServerFirewallRuleClient{MockPostgreSQLFlexibleServerFirewallRuleClient: mockClient}, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		_, qErr := wrapper.Search(ctx, wrapper.Scopes()[0])
		if qErr == nil {
			t.Error("Expected error when providing no query parts, but got nil")
		}
	})

	t.Run("ErrorHandling_Get", func(t *testing.T) {
		expectedErr := errors.New("firewall rule not found")

		mockClient := mocks.NewMockPostgreSQLFlexibleServerFirewallRuleClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, serverName, "nonexistent-rule").Return(
			armpostgresqlflexibleservers.FirewallRulesClientGetResponse{}, expectedErr)

		wrapper := manual.NewDBforPostgreSQLFlexibleServerFirewallRule(&testPostgreSQLFlexibleServerFirewallRuleClient{MockPostgreSQLFlexibleServerFirewallRuleClient: mockClient}, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(serverName, "nonexistent-rule")
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr == nil {
			t.Error("Expected error when getting non-existent firewall rule, but got nil")
		}
	})

	t.Run("ErrorHandling_Search", func(t *testing.T) {
		mockClient := mocks.NewMockPostgreSQLFlexibleServerFirewallRuleClient(ctrl)
		errorPager := &errorPostgreSQLFlexibleServerFirewallRulePager{}
		testClient := &testPostgreSQLFlexibleServerFirewallRuleClient{
			MockPostgreSQLFlexibleServerFirewallRuleClient: mockClient,
			pager: errorPager,
		}

		wrapper := manual.NewDBforPostgreSQLFlexibleServerFirewallRule(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		_, qErr := wrapper.Search(ctx, wrapper.Scopes()[0], serverName)
		if qErr == nil {
			t.Error("Expected error from Search when pager returns error, but got nil")
		}
	})

	t.Run("InterfaceCompliance", func(t *testing.T) {
		mockClient := mocks.NewMockPostgreSQLFlexibleServerFirewallRuleClient(ctrl)
		wrapper := manual.NewDBforPostgreSQLFlexibleServerFirewallRule(&testPostgreSQLFlexibleServerFirewallRuleClient{MockPostgreSQLFlexibleServerFirewallRuleClient: mockClient}, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		w := wrapper.(sources.Wrapper)

		permissions := w.IAMPermissions()
		if len(permissions) == 0 {
			t.Error("Expected IAMPermissions to return at least one permission")
		}
		expectedPermission := "Microsoft.DBforPostgreSQL/flexibleServers/firewallRules/read"
		found := slices.Contains(permissions, expectedPermission)
		if !found {
			t.Errorf("Expected IAMPermissions to include %s", expectedPermission)
		}

		potentialLinks := w.PotentialLinks()
		if !potentialLinks[azureshared.DBforPostgreSQLFlexibleServer] {
			t.Error("Expected PotentialLinks to include DBforPostgreSQLFlexibleServer")
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
			if mapping.GetTerraformQueryMap() == "azurerm_postgresql_flexible_server_firewall_rule.id" {
				foundMapping = true
				break
			}
		}
		if !foundMapping {
			t.Error("Expected TerraformMappings to include 'azurerm_postgresql_flexible_server_firewall_rule.id' mapping")
		}
	})
}

func createAzurePostgreSQLFlexibleServerFirewallRule(serverName, firewallRuleName string) *armpostgresqlflexibleservers.FirewallRule {
	ruleID := "/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.DBforPostgreSQL/flexibleServers/" + serverName + "/firewallRules/" + firewallRuleName
	return &armpostgresqlflexibleservers.FirewallRule{
		Name: stringPtr(firewallRuleName),
		ID:   stringPtr(ruleID),
		Properties: &armpostgresqlflexibleservers.FirewallRuleProperties{
			StartIPAddress: stringPtr("0.0.0.0"),
			EndIPAddress:   stringPtr("255.255.255.255"),
		},
	}
}

func stringPtr(s string) *string {
	return &s
}
