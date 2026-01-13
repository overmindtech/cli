package manual_test

import (
	"context"
	"errors"
	"fmt"
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
	"github.com/overmindtech/cli/sources/stdlib"
)

func TestNetworkApplicationGateway(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"

	t.Run("Get", func(t *testing.T) {
		agName := "test-ag"
		applicationGateway := createAzureApplicationGateway(agName, subscriptionID, resourceGroup)

		mockClient := mocks.NewMockApplicationGatewaysClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, agName, nil).Return(
			armnetwork.ApplicationGatewaysClientGetResponse{
				ApplicationGateway: *applicationGateway,
			}, nil)

		wrapper := manual.NewNetworkApplicationGateway(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], agName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.NetworkApplicationGateway.String() {
			t.Errorf("Expected type %s, got %s", azureshared.NetworkApplicationGateway, sdpItem.GetType())
		}

		if sdpItem.GetUniqueAttribute() != "name" {
			t.Errorf("Expected unique attribute 'name', got %s", sdpItem.GetUniqueAttribute())
		}

		if sdpItem.UniqueAttributeValue() != agName {
			t.Errorf("Expected unique attribute value %s, got %s", agName, sdpItem.UniqueAttributeValue())
		}

		if sdpItem.GetTags()["env"] != "test" {
			t.Errorf("Expected tag 'env=test', got: %v", sdpItem.GetTags()["env"])
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					// GatewayIPConfiguration child resource
					ExpectedType:   azureshared.NetworkApplicationGatewayGatewayIPConfiguration.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey(agName, "gateway-ip-config"),
					ExpectedScope:  fmt.Sprintf("%s.%s", subscriptionID, resourceGroup),
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					// Subnet from GatewayIPConfiguration
					ExpectedType:   azureshared.NetworkSubnet.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey("test-vnet", "test-subnet"),
					ExpectedScope:  fmt.Sprintf("%s.%s", subscriptionID, resourceGroup),
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					// VirtualNetwork from GatewayIPConfiguration subnet
					ExpectedType:   azureshared.NetworkVirtualNetwork.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-vnet",
					ExpectedScope:  fmt.Sprintf("%s.%s", subscriptionID, resourceGroup),
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					// FrontendIPConfiguration child resource
					ExpectedType:   azureshared.NetworkApplicationGatewayFrontendIPConfiguration.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey(agName, "frontend-ip-config"),
					ExpectedScope:  fmt.Sprintf("%s.%s", subscriptionID, resourceGroup),
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					// PublicIPAddress external resource
					ExpectedType:   azureshared.NetworkPublicIPAddress.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-public-ip",
					ExpectedScope:  fmt.Sprintf("%s.%s", subscriptionID, resourceGroup),
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					// Private IP address link (standard library)
					ExpectedType:   stdlib.NetworkIP.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "10.2.0.5",
					ExpectedScope:  "global",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					// BackendAddressPool child resource
					ExpectedType:   azureshared.NetworkApplicationGatewayBackendAddressPool.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey(agName, "backend-pool"),
					ExpectedScope:  fmt.Sprintf("%s.%s", subscriptionID, resourceGroup),
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					// Backend IP address link (standard library)
					ExpectedType:   stdlib.NetworkIP.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "10.0.1.4",
					ExpectedScope:  "global",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					// HTTPListener child resource
					ExpectedType:   azureshared.NetworkApplicationGatewayHTTPListener.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey(agName, "http-listener"),
					ExpectedScope:  fmt.Sprintf("%s.%s", subscriptionID, resourceGroup),
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					// BackendHTTPSettings child resource
					ExpectedType:   azureshared.NetworkApplicationGatewayBackendHTTPSettings.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey(agName, "backend-http-settings"),
					ExpectedScope:  fmt.Sprintf("%s.%s", subscriptionID, resourceGroup),
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					// RequestRoutingRule child resource
					ExpectedType:   azureshared.NetworkApplicationGatewayRequestRoutingRule.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey(agName, "routing-rule"),
					ExpectedScope:  fmt.Sprintf("%s.%s", subscriptionID, resourceGroup),
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					// Probe child resource
					ExpectedType:   azureshared.NetworkApplicationGatewayProbe.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey(agName, "health-probe"),
					ExpectedScope:  fmt.Sprintf("%s.%s", subscriptionID, resourceGroup),
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					// SSLCertificate child resource
					ExpectedType:   azureshared.NetworkApplicationGatewaySSLCertificate.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey(agName, "ssl-cert"),
					ExpectedScope:  fmt.Sprintf("%s.%s", subscriptionID, resourceGroup),
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					// Key Vault Secret from SSLCertificate KeyVaultSecretID
					ExpectedType:   azureshared.KeyVaultSecret.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey("test-keyvault", "test-secret"),
					ExpectedScope:  fmt.Sprintf("%s.%s", subscriptionID, resourceGroup),
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					// DNS name from SSLCertificate KeyVaultSecretID
					ExpectedType:   stdlib.NetworkDNS.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  "test-keyvault.vault.azure.net",
					ExpectedScope:  "global",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					// URLPathMap child resource
					ExpectedType:   azureshared.NetworkApplicationGatewayURLPathMap.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey(agName, "url-path-map"),
					ExpectedScope:  fmt.Sprintf("%s.%s", subscriptionID, resourceGroup),
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					// AuthenticationCertificate child resource
					ExpectedType:   azureshared.NetworkApplicationGatewayAuthenticationCertificate.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey(agName, "auth-cert"),
					ExpectedScope:  fmt.Sprintf("%s.%s", subscriptionID, resourceGroup),
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					// TrustedRootCertificate child resource
					ExpectedType:   azureshared.NetworkApplicationGatewayTrustedRootCertificate.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey(agName, "trusted-root-cert"),
					ExpectedScope:  fmt.Sprintf("%s.%s", subscriptionID, resourceGroup),
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					// Key Vault Secret from TrustedRootCertificate KeyVaultSecretID
					ExpectedType:   azureshared.KeyVaultSecret.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey("test-trusted-keyvault", "test-trusted-secret"),
					ExpectedScope:  fmt.Sprintf("%s.%s", subscriptionID, resourceGroup),
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					// DNS name from TrustedRootCertificate KeyVaultSecretID
					ExpectedType:   stdlib.NetworkDNS.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  "test-trusted-keyvault.vault.azure.net",
					ExpectedScope:  "global",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					// RewriteRuleSet child resource
					ExpectedType:   azureshared.NetworkApplicationGatewayRewriteRuleSet.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey(agName, "rewrite-rule-set"),
					ExpectedScope:  fmt.Sprintf("%s.%s", subscriptionID, resourceGroup),
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					// RedirectConfiguration child resource
					ExpectedType:   azureshared.NetworkApplicationGatewayRedirectConfiguration.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey(agName, "redirect-config"),
					ExpectedScope:  fmt.Sprintf("%s.%s", subscriptionID, resourceGroup),
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					// WAF Policy external resource
					ExpectedType:   azureshared.NetworkApplicationGatewayWebApplicationFirewallPolicy.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-waf-policy",
					ExpectedScope:  fmt.Sprintf("%s.%s", subscriptionID, resourceGroup),
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					// User Assigned Managed Identity external resource
					ExpectedType:   azureshared.ManagedIdentityUserAssignedIdentity.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-identity",
					ExpectedScope:  fmt.Sprintf("%s.%s", subscriptionID, resourceGroup),
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
		mockClient := mocks.NewMockApplicationGatewaysClient(ctrl)

		wrapper := manual.NewNetworkApplicationGateway(mockClient, subscriptionID, resourceGroup)

		// Test with wrong number of query parts - need to call through the wrapper directly
		_, qErr := wrapper.Get(ctx, "part1", "part2")
		if qErr == nil {
			t.Error("Expected error when getting application gateway with wrong number of query parts, but got nil")
		}
	})

	t.Run("Get_EmptyName", func(t *testing.T) {
		mockClient := mocks.NewMockApplicationGatewaysClient(ctrl)

		wrapper := manual.NewNetworkApplicationGateway(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		// Test with empty string name - validation happens before client.Get is called
		// so no mock expectation is needed
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "", true)
		if qErr == nil {
			t.Error("Expected error when getting application gateway with empty name, but got nil")
		}
	})

	t.Run("Get_WithNilName", func(t *testing.T) {
		applicationGateway := &armnetwork.ApplicationGateway{
			Name:     nil, // Application Gateway with nil name should cause an error
			Location: to.Ptr("eastus"),
			Tags: map[string]*string{
				"env": to.Ptr("test"),
			},
		}

		mockClient := mocks.NewMockApplicationGatewaysClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, "test-ag", nil).Return(
			armnetwork.ApplicationGatewaysClientGetResponse{
				ApplicationGateway: *applicationGateway,
			}, nil)

		wrapper := manual.NewNetworkApplicationGateway(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "test-ag", true)
		if qErr == nil {
			t.Error("Expected error when application gateway has nil name, but got nil")
		}
	})

	t.Run("Get_ErrorHandling", func(t *testing.T) {
		mockClient := mocks.NewMockApplicationGatewaysClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, "test-ag", nil).Return(
			armnetwork.ApplicationGatewaysClientGetResponse{}, errors.New("not found"))

		wrapper := manual.NewNetworkApplicationGateway(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "test-ag", true)
		if qErr == nil {
			t.Error("Expected error when client returns error, but got nil")
		}
	})

	t.Run("List", func(t *testing.T) {
		ag1 := createAzureApplicationGateway("test-ag-1", subscriptionID, resourceGroup)
		ag2 := createAzureApplicationGateway("test-ag-2", subscriptionID, resourceGroup)

		mockClient := mocks.NewMockApplicationGatewaysClient(ctrl)
		mockPager := NewMockApplicationGatewaysPager(ctrl)

		// Setup pager expectations
		gomock.InOrder(
			mockPager.EXPECT().More().Return(true),
			mockPager.EXPECT().NextPage(ctx).Return(
				armnetwork.ApplicationGatewaysClientListResponse{
					ApplicationGatewayListResult: armnetwork.ApplicationGatewayListResult{
						Value: []*armnetwork.ApplicationGateway{ag1, ag2},
					},
				}, nil),
			mockPager.EXPECT().More().Return(false),
		)

		mockClient.EXPECT().List(resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewNetworkApplicationGateway(mockClient, subscriptionID, resourceGroup)
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

			if item.GetType() != azureshared.NetworkApplicationGateway.String() {
				t.Fatalf("Expected type %s, got: %s", azureshared.NetworkApplicationGateway, item.GetType())
			}
		}
	})

	t.Run("List_WithNilName", func(t *testing.T) {
		ag1 := createAzureApplicationGateway("test-ag-1", subscriptionID, resourceGroup)
		ag2 := &armnetwork.ApplicationGateway{
			Name:     nil, // Application Gateway with nil name should be skipped
			Location: to.Ptr("eastus"),
			Tags: map[string]*string{
				"env": to.Ptr("test"),
			},
		}

		mockClient := mocks.NewMockApplicationGatewaysClient(ctrl)
		mockPager := NewMockApplicationGatewaysPager(ctrl)

		// Setup pager expectations
		gomock.InOrder(
			mockPager.EXPECT().More().Return(true),
			mockPager.EXPECT().NextPage(ctx).Return(
				armnetwork.ApplicationGatewaysClientListResponse{
					ApplicationGatewayListResult: armnetwork.ApplicationGatewayListResult{
						Value: []*armnetwork.ApplicationGateway{ag1, ag2},
					},
				}, nil),
			mockPager.EXPECT().More().Return(false),
		)

		mockClient.EXPECT().List(resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewNetworkApplicationGateway(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		listable, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support List operation")
		}

		sdpItems, err := listable.List(ctx, wrapper.Scopes()[0], true)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		// Should only return 1 item (ag1), ag2 should be skipped
		if len(sdpItems) != 1 {
			t.Fatalf("Expected 1 item, got: %d", len(sdpItems))
		}
	})

	t.Run("List_ErrorHandling", func(t *testing.T) {
		mockClient := mocks.NewMockApplicationGatewaysClient(ctrl)
		mockPager := NewMockApplicationGatewaysPager(ctrl)

		gomock.InOrder(
			mockPager.EXPECT().More().Return(true),
			mockPager.EXPECT().NextPage(ctx).Return(
				armnetwork.ApplicationGatewaysClientListResponse{}, errors.New("list error")),
		)

		mockClient.EXPECT().List(resourceGroup, nil).Return(mockPager)

		wrapper := manual.NewNetworkApplicationGateway(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		listable, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support List operation")
		}

		_, err := listable.List(ctx, wrapper.Scopes()[0], true)
		if err == nil {
			t.Error("Expected error when pager returns error, but got nil")
		}
	})

	t.Run("CrossResourceGroupLinks", func(t *testing.T) {
		agName := "test-ag"
		applicationGateway := createAzureApplicationGatewayWithDifferentScopePublicIP(agName, subscriptionID, resourceGroup, "other-sub", "other-rg")

		mockClient := mocks.NewMockApplicationGatewaysClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, agName, nil).Return(
			armnetwork.ApplicationGatewaysClientGetResponse{
				ApplicationGateway: *applicationGateway,
			}, nil)

		wrapper := manual.NewNetworkApplicationGateway(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], agName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		// Find the PublicIPAddress linked query
		found := false
		for _, linkedQuery := range sdpItem.GetLinkedItemQueries() {
			if linkedQuery.GetQuery().GetType() == azureshared.NetworkPublicIPAddress.String() {
				found = true
				expectedScope := fmt.Sprintf("%s.%s", "other-sub", "other-rg")
				if linkedQuery.GetQuery().GetScope() != expectedScope {
					t.Errorf("Expected PublicIPAddress scope to be %s, got: %s", expectedScope, linkedQuery.GetQuery().GetScope())
				}
				break
			}
		}
		if !found {
			t.Error("Expected to find PublicIPAddress linked query")
		}
	})

	t.Run("InterfaceCompliance", func(t *testing.T) {
		mockClient := mocks.NewMockApplicationGatewaysClient(ctrl)
		wrapper := manual.NewNetworkApplicationGateway(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		// Verify adapter implements ListableAdapter interface
		_, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			t.Error("Adapter should implement ListableAdapter interface")
		}

		// Verify GetLookups
		lookups := wrapper.GetLookups()
		if len(lookups) == 0 {
			t.Error("Expected GetLookups to return at least one lookup")
		}

		// Verify PotentialLinks
		potentialLinks := wrapper.PotentialLinks()
		expectedLinks := []shared.ItemType{
			azureshared.NetworkApplicationGatewayGatewayIPConfiguration,
			azureshared.NetworkApplicationGatewayFrontendIPConfiguration,
			azureshared.NetworkApplicationGatewayBackendAddressPool,
			azureshared.NetworkApplicationGatewayHTTPListener,
			azureshared.NetworkApplicationGatewayBackendHTTPSettings,
			azureshared.NetworkApplicationGatewayRequestRoutingRule,
			azureshared.NetworkApplicationGatewayProbe,
			azureshared.NetworkApplicationGatewaySSLCertificate,
			azureshared.NetworkApplicationGatewayURLPathMap,
			azureshared.NetworkApplicationGatewayAuthenticationCertificate,
			azureshared.NetworkApplicationGatewayTrustedRootCertificate,
			azureshared.NetworkApplicationGatewayRewriteRuleSet,
			azureshared.NetworkApplicationGatewayRedirectConfiguration,
			azureshared.NetworkSubnet,
			azureshared.NetworkVirtualNetwork,
			azureshared.NetworkPublicIPAddress,
			azureshared.NetworkApplicationGatewayWebApplicationFirewallPolicy,
			azureshared.ManagedIdentityUserAssignedIdentity,
			azureshared.KeyVaultSecret,
			stdlib.NetworkIP,
			stdlib.NetworkDNS,
		}
		for _, expectedLink := range expectedLinks {
			if !potentialLinks[expectedLink] {
				t.Errorf("Expected PotentialLinks to include %s", expectedLink)
			}
		}

		// Verify TerraformMappings
		mappings := wrapper.TerraformMappings()
		if len(mappings) == 0 {
			t.Error("Expected TerraformMappings to return at least one mapping")
		}
		foundMapping := false
		for _, mapping := range mappings {
			if mapping.GetTerraformQueryMap() == "azurerm_application_gateway.name" {
				foundMapping = true
				break
			}
		}
		if !foundMapping {
			t.Error("Expected TerraformMappings to include 'azurerm_application_gateway.name' mapping")
		}

		// Verify PredefinedRole
		if roleInterface, ok := interface{}(wrapper).(interface{ PredefinedRole() string }); ok {
			role := roleInterface.PredefinedRole()
			if role != "Reader" {
				t.Errorf("Expected PredefinedRole to be 'Reader', got %s", role)
			}
		} else {
			t.Error("Wrapper does not implement PredefinedRole method")
		}
	})
}

// MockApplicationGatewaysPager is a simple mock for ApplicationGatewaysPager
type MockApplicationGatewaysPager struct {
	ctrl     *gomock.Controller
	recorder *MockApplicationGatewaysPagerMockRecorder
}

type MockApplicationGatewaysPagerMockRecorder struct {
	mock *MockApplicationGatewaysPager
}

func NewMockApplicationGatewaysPager(ctrl *gomock.Controller) *MockApplicationGatewaysPager {
	mock := &MockApplicationGatewaysPager{ctrl: ctrl}
	mock.recorder = &MockApplicationGatewaysPagerMockRecorder{mock}
	return mock
}

func (m *MockApplicationGatewaysPager) EXPECT() *MockApplicationGatewaysPagerMockRecorder {
	return m.recorder
}

func (m *MockApplicationGatewaysPager) More() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "More")
	ret0, _ := ret[0].(bool)
	return ret0
}

func (mr *MockApplicationGatewaysPagerMockRecorder) More() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "More", reflect.TypeOf((*MockApplicationGatewaysPager)(nil).More))
}

func (m *MockApplicationGatewaysPager) NextPage(ctx context.Context) (armnetwork.ApplicationGatewaysClientListResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NextPage", ctx)
	ret0, _ := ret[0].(armnetwork.ApplicationGatewaysClientListResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockApplicationGatewaysPagerMockRecorder) NextPage(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NextPage", reflect.TypeOf((*MockApplicationGatewaysPager)(nil).NextPage), ctx)
}

// createAzureApplicationGateway creates a mock Azure Application Gateway for testing
func createAzureApplicationGateway(agName, subscriptionID, resourceGroup string) *armnetwork.ApplicationGateway {
	return &armnetwork.ApplicationGateway{
		Name:     to.Ptr(agName),
		Location: to.Ptr("eastus"),
		Tags: map[string]*string{
			"env":     to.Ptr("test"),
			"project": to.Ptr("testing"),
		},
		Properties: &armnetwork.ApplicationGatewayPropertiesFormat{
			// GatewayIPConfigurations (Child Resource)
			GatewayIPConfigurations: []*armnetwork.ApplicationGatewayIPConfiguration{
				{
					Name: to.Ptr("gateway-ip-config"),
					Properties: &armnetwork.ApplicationGatewayIPConfigurationPropertiesFormat{
						Subnet: &armnetwork.SubResource{
							ID: to.Ptr("/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/test-subnet"),
						},
					},
				},
			},
			// FrontendIPConfigurations (Child Resource)
			FrontendIPConfigurations: []*armnetwork.ApplicationGatewayFrontendIPConfiguration{
				{
					Name: to.Ptr("frontend-ip-config"),
					Properties: &armnetwork.ApplicationGatewayFrontendIPConfigurationPropertiesFormat{
						PublicIPAddress: &armnetwork.SubResource{
							ID: to.Ptr("/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/publicIPAddresses/test-public-ip"),
						},
						PrivateIPAddress: to.Ptr("10.2.0.5"),
					},
				},
			},
			// BackendAddressPools (Child Resource)
			BackendAddressPools: []*armnetwork.ApplicationGatewayBackendAddressPool{
				{
					Name: to.Ptr("backend-pool"),
					Properties: &armnetwork.ApplicationGatewayBackendAddressPoolPropertiesFormat{
						BackendAddresses: []*armnetwork.ApplicationGatewayBackendAddress{
							{
								IPAddress: to.Ptr("10.0.1.4"),
							},
						},
					},
				},
			},
			// HTTPListeners (Child Resource)
			HTTPListeners: []*armnetwork.ApplicationGatewayHTTPListener{
				{
					Name: to.Ptr("http-listener"),
				},
			},
			// BackendHTTPSettingsCollection (Child Resource)
			BackendHTTPSettingsCollection: []*armnetwork.ApplicationGatewayBackendHTTPSettings{
				{
					Name: to.Ptr("backend-http-settings"),
				},
			},
			// RequestRoutingRules (Child Resource)
			RequestRoutingRules: []*armnetwork.ApplicationGatewayRequestRoutingRule{
				{
					Name: to.Ptr("routing-rule"),
				},
			},
			// Probes (Child Resource)
			Probes: []*armnetwork.ApplicationGatewayProbe{
				{
					Name: to.Ptr("health-probe"),
				},
			},
			// SSLCertificates (Child Resource)
			SSLCertificates: []*armnetwork.ApplicationGatewaySSLCertificate{
				{
					Name: to.Ptr("ssl-cert"),
					Properties: &armnetwork.ApplicationGatewaySSLCertificatePropertiesFormat{
						KeyVaultSecretID: to.Ptr("https://test-keyvault.vault.azure.net/secrets/test-secret/version"),
					},
				},
			},
			// URLPathMaps (Child Resource)
			URLPathMaps: []*armnetwork.ApplicationGatewayURLPathMap{
				{
					Name: to.Ptr("url-path-map"),
				},
			},
			// AuthenticationCertificates (Child Resource)
			AuthenticationCertificates: []*armnetwork.ApplicationGatewayAuthenticationCertificate{
				{
					Name: to.Ptr("auth-cert"),
				},
			},
			// TrustedRootCertificates (Child Resource)
			TrustedRootCertificates: []*armnetwork.ApplicationGatewayTrustedRootCertificate{
				{
					Name: to.Ptr("trusted-root-cert"),
					Properties: &armnetwork.ApplicationGatewayTrustedRootCertificatePropertiesFormat{
						KeyVaultSecretID: to.Ptr("https://test-trusted-keyvault.vault.azure.net/secrets/test-trusted-secret/version"),
					},
				},
			},
			// RewriteRuleSets (Child Resource)
			RewriteRuleSets: []*armnetwork.ApplicationGatewayRewriteRuleSet{
				{
					Name: to.Ptr("rewrite-rule-set"),
				},
			},
			// RedirectConfigurations (Child Resource)
			RedirectConfigurations: []*armnetwork.ApplicationGatewayRedirectConfiguration{
				{
					Name: to.Ptr("redirect-config"),
				},
			},
			// FirewallPolicy (External Resource)
			FirewallPolicy: &armnetwork.SubResource{
				ID: to.Ptr("/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/ApplicationGatewayWebApplicationFirewallPolicies/test-waf-policy"),
			},
		},
		Identity: &armnetwork.ManagedServiceIdentity{
			Type: to.Ptr(armnetwork.ResourceIdentityTypeUserAssigned),
			UserAssignedIdentities: map[string]*armnetwork.Components1Jq1T4ISchemasManagedserviceidentityPropertiesUserassignedidentitiesAdditionalproperties{
				"/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.ManagedIdentity/userAssignedIdentities/test-identity": {},
			},
		},
	}
}

// createAzureApplicationGatewayWithDifferentScopePublicIP creates an Application Gateway with PublicIPAddress in different scope
func createAzureApplicationGatewayWithDifferentScopePublicIP(agName, subscriptionID, resourceGroup, otherSubscriptionID, otherResourceGroup string) *armnetwork.ApplicationGateway {
	ag := createAzureApplicationGateway(agName, subscriptionID, resourceGroup)
	// Override FrontendIPConfiguration with PublicIPAddress in different scope
	ag.Properties.FrontendIPConfigurations = []*armnetwork.ApplicationGatewayFrontendIPConfiguration{
		{
			Name: to.Ptr("frontend-ip-config"),
			Properties: &armnetwork.ApplicationGatewayFrontendIPConfigurationPropertiesFormat{
				PublicIPAddress: &armnetwork.SubResource{
					ID: to.Ptr("/subscriptions/" + otherSubscriptionID + "/resourceGroups/" + otherResourceGroup + "/providers/Microsoft.Network/publicIPAddresses/test-public-ip"),
				},
			},
		},
	}
	return ag
}
