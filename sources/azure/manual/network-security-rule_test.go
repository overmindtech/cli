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

type mockSecurityRulesPager struct {
	pages []armnetwork.SecurityRulesClientListResponse
	index int
}

func (m *mockSecurityRulesPager) More() bool {
	return m.index < len(m.pages)
}

func (m *mockSecurityRulesPager) NextPage(ctx context.Context) (armnetwork.SecurityRulesClientListResponse, error) {
	if m.index >= len(m.pages) {
		return armnetwork.SecurityRulesClientListResponse{}, errors.New("no more pages")
	}
	page := m.pages[m.index]
	m.index++
	return page, nil
}

type errorSecurityRulesPager struct{}

func (e *errorSecurityRulesPager) More() bool {
	return true
}

func (e *errorSecurityRulesPager) NextPage(ctx context.Context) (armnetwork.SecurityRulesClientListResponse, error) {
	return armnetwork.SecurityRulesClientListResponse{}, errors.New("pager error")
}

type testSecurityRulesClient struct {
	*mocks.MockSecurityRulesClient
	pager clients.SecurityRulesPager
}

func (t *testSecurityRulesClient) NewListPager(resourceGroupName, networkSecurityGroupName string, options *armnetwork.SecurityRulesClientListOptions) clients.SecurityRulesPager {
	return t.pager
}

func TestNetworkSecurityRule(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"
	nsgName := "test-nsg"
	ruleName := "test-rule"

	t.Run("Get", func(t *testing.T) {
		rule := createAzureSecurityRule(ruleName, nsgName)

		mockClient := mocks.NewMockSecurityRulesClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, nsgName, ruleName, nil).Return(
			armnetwork.SecurityRulesClientGetResponse{
				SecurityRule: *rule,
			}, nil)

		testClient := &testSecurityRulesClient{MockSecurityRulesClient: mockClient}
		wrapper := manual.NewNetworkSecurityRule(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(nsgName, ruleName)
		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.NetworkSecurityRule.String() {
			t.Errorf("Expected type %s, got %s", azureshared.NetworkSecurityRule, sdpItem.GetType())
		}

		if sdpItem.GetUniqueAttribute() != "uniqueAttr" {
			t.Errorf("Expected unique attribute 'uniqueAttr', got %s", sdpItem.GetUniqueAttribute())
		}

		if sdpItem.UniqueAttributeValue() != shared.CompositeLookupKey(nsgName, ruleName) {
			t.Errorf("Expected unique attribute value %s, got %s", shared.CompositeLookupKey(nsgName, ruleName), sdpItem.UniqueAttributeValue())
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
					ExpectedQuery:  nsgName,
					ExpectedScope:  subscriptionID + "." + resourceGroup,
				},
			}
			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("Get_EmptyRuleName", func(t *testing.T) {
		mockClient := mocks.NewMockSecurityRulesClient(ctrl)
		testClient := &testSecurityRulesClient{MockSecurityRulesClient: mockClient}

		wrapper := manual.NewNetworkSecurityRule(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(nsgName, "")
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr == nil {
			t.Error("Expected error when rule name is empty, but got nil")
		}
	})

	t.Run("Get_InsufficientQueryParts", func(t *testing.T) {
		mockClient := mocks.NewMockSecurityRulesClient(ctrl)
		testClient := &testSecurityRulesClient{MockSecurityRulesClient: mockClient}

		wrapper := manual.NewNetworkSecurityRule(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], nsgName, true)
		if qErr == nil {
			t.Error("Expected error when providing insufficient query parts, but got nil")
		}
	})

	t.Run("Search", func(t *testing.T) {
		rule1 := createAzureSecurityRule("rule-1", nsgName)
		rule2 := createAzureSecurityRule("rule-2", nsgName)

		mockClient := mocks.NewMockSecurityRulesClient(ctrl)
		mockPager := &mockSecurityRulesPager{
			pages: []armnetwork.SecurityRulesClientListResponse{
				{
					SecurityRuleListResult: armnetwork.SecurityRuleListResult{
						Value: []*armnetwork.SecurityRule{rule1, rule2},
					},
				},
			},
		}

		testClient := &testSecurityRulesClient{
			MockSecurityRulesClient: mockClient,
			pager:                   mockPager,
		}

		wrapper := manual.NewNetworkSecurityRule(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support Search operation")
		}

		sdpItems, err := searchable.Search(ctx, wrapper.Scopes()[0], nsgName, true)
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
			if item.GetType() != azureshared.NetworkSecurityRule.String() {
				t.Errorf("Expected type %s, got %s", azureshared.NetworkSecurityRule, item.GetType())
			}
		}
	})

	t.Run("Search_InvalidQueryParts", func(t *testing.T) {
		mockClient := mocks.NewMockSecurityRulesClient(ctrl)
		testClient := &testSecurityRulesClient{MockSecurityRulesClient: mockClient}

		wrapper := manual.NewNetworkSecurityRule(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		_, qErr := wrapper.Search(ctx, wrapper.Scopes()[0])
		if qErr == nil {
			t.Error("Expected error when providing no query parts, but got nil")
		}
	})

	t.Run("Search_RuleWithNilName", func(t *testing.T) {
		validRule := createAzureSecurityRule("valid-rule", nsgName)

		mockClient := mocks.NewMockSecurityRulesClient(ctrl)
		mockPager := &mockSecurityRulesPager{
			pages: []armnetwork.SecurityRulesClientListResponse{
				{
					SecurityRuleListResult: armnetwork.SecurityRuleListResult{
						Value: []*armnetwork.SecurityRule{
							{Name: nil, ID: new("/some/id")},
							validRule,
						},
					},
				},
			},
		}

		testClient := &testSecurityRulesClient{
			MockSecurityRulesClient: mockClient,
			pager:                   mockPager,
		}

		wrapper := manual.NewNetworkSecurityRule(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		searchable := adapter.(discovery.SearchableAdapter)
		sdpItems, err := searchable.Search(ctx, wrapper.Scopes()[0], nsgName, true)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if len(sdpItems) != 1 {
			t.Fatalf("Expected 1 item (nil name skipped), got: %d", len(sdpItems))
		}
		if sdpItems[0].UniqueAttributeValue() != shared.CompositeLookupKey(nsgName, "valid-rule") {
			t.Errorf("Expected unique value %s, got %s", shared.CompositeLookupKey(nsgName, "valid-rule"), sdpItems[0].UniqueAttributeValue())
		}
	})

	t.Run("ErrorHandling_Get", func(t *testing.T) {
		expectedErr := errors.New("security rule not found")

		mockClient := mocks.NewMockSecurityRulesClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, nsgName, "nonexistent-rule", nil).Return(
			armnetwork.SecurityRulesClientGetResponse{}, expectedErr)

		testClient := &testSecurityRulesClient{MockSecurityRulesClient: mockClient}
		wrapper := manual.NewNetworkSecurityRule(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(nsgName, "nonexistent-rule")
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr == nil {
			t.Error("Expected error when getting non-existent rule, but got nil")
		}
	})

	t.Run("ErrorHandling_Search", func(t *testing.T) {
		mockClient := mocks.NewMockSecurityRulesClient(ctrl)
		testClient := &testSecurityRulesClient{
			MockSecurityRulesClient: mockClient,
			pager:                   &errorSecurityRulesPager{},
		}

		wrapper := manual.NewNetworkSecurityRule(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		searchable := adapter.(discovery.SearchableAdapter)
		_, err := searchable.Search(ctx, wrapper.Scopes()[0], nsgName, true)
		if err == nil {
			t.Error("Expected error from pager when NextPage returns an error, but got nil")
		}
	})
}

func createAzureSecurityRule(ruleName, nsgName string) *armnetwork.SecurityRule {
	idStr := "/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.Network/networkSecurityGroups/" + nsgName + "/securityRules/" + ruleName
	typeStr := "Microsoft.Network/networkSecurityGroups/securityRules"
	access := armnetwork.SecurityRuleAccessAllow
	direction := armnetwork.SecurityRuleDirectionInbound
	protocol := armnetwork.SecurityRuleProtocolAsterisk
	priority := int32(100)
	return &armnetwork.SecurityRule{
		ID:   &idStr,
		Name: &ruleName,
		Type: &typeStr,
		Properties: &armnetwork.SecurityRulePropertiesFormat{
			Access:    &access,
			Direction: &direction,
			Protocol:  &protocol,
			Priority:  &priority,
		},
	}
}
