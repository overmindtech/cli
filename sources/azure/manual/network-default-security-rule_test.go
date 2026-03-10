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

type mockDefaultSecurityRulesPager struct {
	pages []armnetwork.DefaultSecurityRulesClientListResponse
	index int
}

func (m *mockDefaultSecurityRulesPager) More() bool {
	return m.index < len(m.pages)
}

func (m *mockDefaultSecurityRulesPager) NextPage(ctx context.Context) (armnetwork.DefaultSecurityRulesClientListResponse, error) {
	if m.index >= len(m.pages) {
		return armnetwork.DefaultSecurityRulesClientListResponse{}, errors.New("no more pages")
	}
	page := m.pages[m.index]
	m.index++
	return page, nil
}

type errorDefaultSecurityRulesPager struct{}

func (e *errorDefaultSecurityRulesPager) More() bool {
	return true
}

func (e *errorDefaultSecurityRulesPager) NextPage(ctx context.Context) (armnetwork.DefaultSecurityRulesClientListResponse, error) {
	return armnetwork.DefaultSecurityRulesClientListResponse{}, errors.New("pager error")
}

type testDefaultSecurityRulesClient struct {
	*mocks.MockDefaultSecurityRulesClient
	pager clients.DefaultSecurityRulesPager
}

func (t *testDefaultSecurityRulesClient) NewListPager(resourceGroupName, networkSecurityGroupName string, options *armnetwork.DefaultSecurityRulesClientListOptions) clients.DefaultSecurityRulesPager {
	return t.pager
}

func TestNetworkDefaultSecurityRule(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"
	nsgName := "test-nsg"
	ruleName := "AllowVnetInBound"

	t.Run("Get", func(t *testing.T) {
		rule := createAzureDefaultSecurityRule(ruleName, nsgName)

		mockClient := mocks.NewMockDefaultSecurityRulesClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, nsgName, ruleName, nil).Return(
			armnetwork.DefaultSecurityRulesClientGetResponse{
				SecurityRule: rule,
			}, nil)

		testClient := &testDefaultSecurityRulesClient{MockDefaultSecurityRulesClient: mockClient}
		wrapper := manual.NewNetworkDefaultSecurityRule(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(nsgName, ruleName)
		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.NetworkDefaultSecurityRule.String() {
			t.Errorf("Expected type %s, got %s", azureshared.NetworkDefaultSecurityRule, sdpItem.GetType())
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
		mockClient := mocks.NewMockDefaultSecurityRulesClient(ctrl)
		testClient := &testDefaultSecurityRulesClient{MockDefaultSecurityRulesClient: mockClient}

		wrapper := manual.NewNetworkDefaultSecurityRule(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(nsgName, "")
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr == nil {
			t.Error("Expected error when rule name is empty, but got nil")
		}
	})

	t.Run("Get_InsufficientQueryParts", func(t *testing.T) {
		mockClient := mocks.NewMockDefaultSecurityRulesClient(ctrl)
		testClient := &testDefaultSecurityRulesClient{MockDefaultSecurityRulesClient: mockClient}

		wrapper := manual.NewNetworkDefaultSecurityRule(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], nsgName, true)
		if qErr == nil {
			t.Error("Expected error when providing insufficient query parts, but got nil")
		}
	})

	t.Run("Search", func(t *testing.T) {
		rule1 := createAzureDefaultSecurityRule("AllowVnetInBound", nsgName)
		rule2 := createAzureDefaultSecurityRule("AllowAzureLoadBalancerInBound", nsgName)

		mockClient := mocks.NewMockDefaultSecurityRulesClient(ctrl)
		mockPager := &mockDefaultSecurityRulesPager{
			pages: []armnetwork.DefaultSecurityRulesClientListResponse{
				{
					SecurityRuleListResult: armnetwork.SecurityRuleListResult{
						Value: []*armnetwork.SecurityRule{&rule1, &rule2},
					},
				},
			},
		}

		testClient := &testDefaultSecurityRulesClient{
			MockDefaultSecurityRulesClient: mockClient,
			pager:                          mockPager,
		}

		wrapper := manual.NewNetworkDefaultSecurityRule(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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
			if item.GetType() != azureshared.NetworkDefaultSecurityRule.String() {
				t.Errorf("Expected type %s, got %s", azureshared.NetworkDefaultSecurityRule, item.GetType())
			}
		}
	})

	t.Run("Search_InvalidQueryParts", func(t *testing.T) {
		mockClient := mocks.NewMockDefaultSecurityRulesClient(ctrl)
		testClient := &testDefaultSecurityRulesClient{MockDefaultSecurityRulesClient: mockClient}

		wrapper := manual.NewNetworkDefaultSecurityRule(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		_, qErr := wrapper.Search(ctx, wrapper.Scopes()[0])
		if qErr == nil {
			t.Error("Expected error when providing no query parts, but got nil")
		}
	})

	t.Run("Search_RuleWithNilName", func(t *testing.T) {
		validRule := createAzureDefaultSecurityRule("AllowVnetInBound", nsgName)

		mockClient := mocks.NewMockDefaultSecurityRulesClient(ctrl)
		mockPager := &mockDefaultSecurityRulesPager{
			pages: []armnetwork.DefaultSecurityRulesClientListResponse{
				{
					SecurityRuleListResult: armnetwork.SecurityRuleListResult{
						Value: []*armnetwork.SecurityRule{
							{Name: nil, ID: new(string)},
							&validRule,
						},
					},
				},
			},
		}

		testClient := &testDefaultSecurityRulesClient{
			MockDefaultSecurityRulesClient: mockClient,
			pager:                          mockPager,
		}

		wrapper := manual.NewNetworkDefaultSecurityRule(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		searchable := adapter.(discovery.SearchableAdapter)
		sdpItems, err := searchable.Search(ctx, wrapper.Scopes()[0], nsgName, true)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if len(sdpItems) != 1 {
			t.Fatalf("Expected 1 item (nil name skipped), got: %d", len(sdpItems))
		}
		if sdpItems[0].UniqueAttributeValue() != shared.CompositeLookupKey(nsgName, "AllowVnetInBound") {
			t.Errorf("Expected unique value %s, got %s", shared.CompositeLookupKey(nsgName, "AllowVnetInBound"), sdpItems[0].UniqueAttributeValue())
		}
	})

	t.Run("ErrorHandling_Get", func(t *testing.T) {
		expectedErr := errors.New("default security rule not found")

		mockClient := mocks.NewMockDefaultSecurityRulesClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, nsgName, "nonexistent-rule", nil).Return(
			armnetwork.DefaultSecurityRulesClientGetResponse{}, expectedErr)

		testClient := &testDefaultSecurityRulesClient{MockDefaultSecurityRulesClient: mockClient}
		wrapper := manual.NewNetworkDefaultSecurityRule(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(nsgName, "nonexistent-rule")
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr == nil {
			t.Error("Expected error when getting non-existent rule, but got nil")
		}
	})

	t.Run("ErrorHandling_Search", func(t *testing.T) {
		mockClient := mocks.NewMockDefaultSecurityRulesClient(ctrl)
		testClient := &testDefaultSecurityRulesClient{
			MockDefaultSecurityRulesClient: mockClient,
			pager:                          &errorDefaultSecurityRulesPager{},
		}

		wrapper := manual.NewNetworkDefaultSecurityRule(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		searchable := adapter.(discovery.SearchableAdapter)
		_, err := searchable.Search(ctx, wrapper.Scopes()[0], nsgName, true)
		if err == nil {
			t.Error("Expected error from pager when NextPage returns an error, but got nil")
		}
	})
}

func createAzureDefaultSecurityRule(ruleName, nsgName string) armnetwork.SecurityRule {
	idStr := "/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.Network/networkSecurityGroups/" + nsgName + "/defaultSecurityRules/" + ruleName
	typeStr := "Microsoft.Network/networkSecurityGroups/defaultSecurityRules"
	access := armnetwork.SecurityRuleAccessAllow
	direction := armnetwork.SecurityRuleDirectionInbound
	protocol := armnetwork.SecurityRuleProtocolAsterisk
	priority := int32(65000)
	return armnetwork.SecurityRule{
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
