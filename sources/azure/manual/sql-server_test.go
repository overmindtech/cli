package manual_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/sql/armsql"
	"go.uber.org/mock/gomock"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	"github.com/overmindtech/cli/sources/azure/manual"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/azure/shared/mocks"
	"github.com/overmindtech/cli/sources/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

// mockSqlServersPager is a simple mock implementation of SqlServersPager
type mockSqlServersPager struct {
	pages []armsql.ServersClientListByResourceGroupResponse
	index int
}

func (m *mockSqlServersPager) More() bool {
	return m.index < len(m.pages)
}

func (m *mockSqlServersPager) NextPage(ctx context.Context) (armsql.ServersClientListByResourceGroupResponse, error) {
	if m.index >= len(m.pages) {
		return armsql.ServersClientListByResourceGroupResponse{}, errors.New("no more pages")
	}
	page := m.pages[m.index]
	m.index++
	return page, nil
}

// errorSqlServersPager is a mock pager that always returns an error
type errorSqlServersPager struct{}

func (e *errorSqlServersPager) More() bool {
	return true // Always return true so NextPage will be called
}

func (e *errorSqlServersPager) NextPage(ctx context.Context) (armsql.ServersClientListByResourceGroupResponse, error) {
	return armsql.ServersClientListByResourceGroupResponse{}, errors.New("pager error")
}

// testSqlServersClient wraps the mock to implement the correct interface
type testSqlServersClient struct {
	*mocks.MockSqlServersClient
	pager clients.SqlServersPager
}

func (t *testSqlServersClient) ListByResourceGroup(ctx context.Context, resourceGroupName string, options *armsql.ServersClientListByResourceGroupOptions) clients.SqlServersPager {
	return t.pager
}

func TestSqlServer(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"
	serverName := "test-server"

	t.Run("Get", func(t *testing.T) {
		server := createAzureSqlServer(serverName, "", "")

		mockClient := mocks.NewMockSqlServersClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, serverName, nil).Return(
			armsql.ServersClientGetResponse{
				Server: *server,
			}, nil)

		testClient := &testSqlServersClient{MockSqlServersClient: mockClient}
		wrapper := manual.NewSqlServer(testClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper)

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], serverName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.SQLServer.String() {
			t.Errorf("Expected type %s, got %s", azureshared.SQLServer, sdpItem.GetType())
		}

		if sdpItem.GetUniqueAttribute() != "name" {
			t.Errorf("Expected unique attribute 'name', got %s", sdpItem.GetUniqueAttribute())
		}

		if sdpItem.UniqueAttributeValue() != serverName {
			t.Errorf("Expected unique attribute value %s, got %s", serverName, sdpItem.UniqueAttributeValue())
		}

		if sdpItem.GetScope() != subscriptionID+"."+resourceGroup {
			t.Errorf("Expected scope %s, got %s", subscriptionID+"."+resourceGroup, sdpItem.GetScope())
		}

		// Validate the item
		if err := sdpItem.Validate(); err != nil {
			t.Fatalf("Expected no validation error, got: %v", err)
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				// Child resources
				{
					ExpectedType:   azureshared.SQLDatabase.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  serverName,
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					ExpectedType:   azureshared.SQLElasticPool.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  serverName,
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					ExpectedType:   azureshared.SQLServerFirewallRule.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  serverName,
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					ExpectedType:   azureshared.SQLServerVirtualNetworkRule.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  serverName,
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					ExpectedType:   azureshared.SQLServerKey.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  serverName,
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					ExpectedType:   azureshared.SQLServerFailoverGroup.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  serverName,
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					ExpectedType:   azureshared.SQLServerAdministrator.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  serverName,
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					ExpectedType:   azureshared.SQLServerSyncGroup.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  serverName,
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					ExpectedType:   azureshared.SQLServerSyncAgent.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  serverName,
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					ExpectedType:   azureshared.SQLServerPrivateEndpointConnection.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  serverName,
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					ExpectedType:   azureshared.SQLServerAuditingSetting.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  serverName,
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					ExpectedType:   azureshared.SQLServerSecurityAlertPolicy.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  serverName,
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					ExpectedType:   azureshared.SQLServerVulnerabilityAssessment.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  serverName,
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					ExpectedType:   azureshared.SQLServerEncryptionProtector.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  serverName,
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					ExpectedType:   azureshared.SQLServerBlobAuditingPolicy.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  serverName,
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					ExpectedType:   azureshared.SQLServerAutomaticTuning.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  serverName,
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					ExpectedType:   azureshared.SQLServerAdvancedThreatProtectionSetting.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  serverName,
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					ExpectedType:   azureshared.SQLServerDnsAlias.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  serverName,
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					ExpectedType:   azureshared.SQLServerUsage.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  serverName,
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					ExpectedType:   azureshared.SQLServerOperation.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  serverName,
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					ExpectedType:   azureshared.SQLServerAdvisor.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  serverName,
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					ExpectedType:   azureshared.SQLServerBackupLongTermRetentionPolicy.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  serverName,
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					ExpectedType:   azureshared.SQLServerDevOpsAuditSetting.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  serverName,
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					ExpectedType:   azureshared.SQLServerTrustGroup.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  serverName,
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					ExpectedType:   azureshared.SQLServerOutboundFirewallRule.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  serverName,
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					ExpectedType:   azureshared.SQLServerPrivateLinkResource.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  serverName,
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					// DNS name link (from FullyQualifiedDomainName)
					ExpectedType:   stdlib.NetworkDNS.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  serverName + ".database.windows.net",
					ExpectedScope:  "global",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
			}

			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("Get_WithManagedIdentity", func(t *testing.T) {
		identityID := "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.ManagedIdentity/userAssignedIdentities/test-identity"
		server := createAzureSqlServer(serverName, identityID, "test-server.database.windows.net")

		mockClient := mocks.NewMockSqlServersClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, serverName, nil).Return(
			armsql.ServersClientGetResponse{
				Server: *server,
			}, nil)

		testClient := &testSqlServersClient{MockSqlServersClient: mockClient}
		wrapper := manual.NewSqlServer(testClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper)

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], serverName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		// Verify Managed Identity link exists
		foundIdentityLink := false
		foundDNSLink := false
		for _, link := range sdpItem.GetLinkedItemQueries() {
			if link.GetQuery().GetType() == azureshared.ManagedIdentityUserAssignedIdentity.String() {
				foundIdentityLink = true
				if link.GetQuery().GetQuery() != "test-identity" {
					t.Errorf("Expected identity name 'test-identity', got %s", link.GetQuery().GetQuery())
				}
				if link.GetQuery().GetScope() != subscriptionID+"."+resourceGroup {
					t.Errorf("Expected identity scope %s, got %s", subscriptionID+"."+resourceGroup, link.GetQuery().GetScope())
				}
			}
			if link.GetQuery().GetType() == stdlib.NetworkDNS.String() {
				foundDNSLink = true
				if link.GetQuery().GetQuery() != "test-server.database.windows.net" {
					t.Errorf("Expected DNS name 'test-server.database.windows.net', got %s", link.GetQuery().GetQuery())
				}
			}
		}

		if !foundIdentityLink {
			t.Error("Expected to find Managed Identity link")
		}
		if !foundDNSLink {
			t.Error("Expected to find DNS link")
		}
	})

	t.Run("Get_WithMultipleUserAssignedIdentities", func(t *testing.T) {
		primaryIdentityID := "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.ManagedIdentity/userAssignedIdentities/primary-identity"
		secondaryIdentityID := "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.ManagedIdentity/userAssignedIdentities/secondary-identity"
		tertiaryIdentityID := "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.ManagedIdentity/userAssignedIdentities/tertiary-identity"

		server := createAzureSqlServerWithUserAssignedIdentities(
			serverName,
			primaryIdentityID,
			"test-server.database.windows.net",
			map[string]*armsql.UserIdentity{
				primaryIdentityID:   {}, // Primary identity is also in the map (should be deduplicated)
				secondaryIdentityID: {},
				tertiaryIdentityID:  {},
			},
		)

		mockClient := mocks.NewMockSqlServersClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, serverName, nil).Return(
			armsql.ServersClientGetResponse{
				Server: *server,
			}, nil)

		testClient := &testSqlServersClient{MockSqlServersClient: mockClient}
		wrapper := manual.NewSqlServer(testClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper)

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], serverName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		// Verify all three managed identity links exist (primary, secondary, tertiary)
		// Primary should be included from PrimaryUserAssignedIdentityID
		// Secondary and tertiary should be included from Identity.UserAssignedIdentities
		identityLinks := make(map[string]bool)
		for _, link := range sdpItem.GetLinkedItemQueries() {
			if link.GetQuery().GetType() == azureshared.ManagedIdentityUserAssignedIdentity.String() {
				identityLinks[link.GetQuery().GetQuery()] = true
				if link.GetQuery().GetScope() != subscriptionID+"."+resourceGroup {
					t.Errorf("Expected identity scope %s, got %s", subscriptionID+"."+resourceGroup, link.GetQuery().GetScope())
				}
			}
		}

		expectedIdentities := []string{"primary-identity", "secondary-identity", "tertiary-identity"}
		if len(identityLinks) != len(expectedIdentities) {
			t.Errorf("Expected %d identity links, got %d: %v", len(expectedIdentities), len(identityLinks), identityLinks)
		}

		for _, expectedIdentity := range expectedIdentities {
			if !identityLinks[expectedIdentity] {
				t.Errorf("Expected to find identity link for '%s'", expectedIdentity)
			}
		}
	})

	t.Run("Get_WithPrivateEndpointConnections", func(t *testing.T) {
		privateEndpointID1 := "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/privateEndpoints/test-private-endpoint-1"
		privateEndpointID2 := "/subscriptions/" + subscriptionID + "/resourceGroups/different-rg/providers/Microsoft.Network/privateEndpoints/test-private-endpoint-2"

		server := createAzureSqlServerWithPrivateEndpointConnections(
			serverName,
			"",
			"test-server.database.windows.net",
			[]*armsql.ServerPrivateEndpointConnection{
				{
					Properties: &armsql.PrivateEndpointConnectionProperties{
						PrivateEndpoint: &armsql.PrivateEndpointProperty{
							ID: to.Ptr(privateEndpointID1),
						},
					},
				},
				{
					Properties: &armsql.PrivateEndpointConnectionProperties{
						PrivateEndpoint: &armsql.PrivateEndpointProperty{
							ID: to.Ptr(privateEndpointID2),
						},
					},
				},
			},
		)

		mockClient := mocks.NewMockSqlServersClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, serverName, nil).Return(
			armsql.ServersClientGetResponse{
				Server: *server,
			}, nil)

		testClient := &testSqlServersClient{MockSqlServersClient: mockClient}
		wrapper := manual.NewSqlServer(testClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper)

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], serverName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		// Verify PrivateEndpointConnection child resource link exists
		foundPrivateEndpointConnectionLink := false
		// Verify NetworkPrivateEndpoint links exist
		privateEndpointLinks := make(map[string]string) // name -> scope

		for _, link := range sdpItem.GetLinkedItemQueries() {
			if link.GetQuery().GetType() == azureshared.SQLServerPrivateEndpointConnection.String() {
				foundPrivateEndpointConnectionLink = true
				if link.GetQuery().GetQuery() != serverName {
					t.Errorf("Expected PrivateEndpointConnection query '%s', got %s", serverName, link.GetQuery().GetQuery())
				}
			}
			if link.GetQuery().GetType() == azureshared.NetworkPrivateEndpoint.String() {
				privateEndpointLinks[link.GetQuery().GetQuery()] = link.GetQuery().GetScope()
				if link.GetQuery().GetMethod() != sdp.QueryMethod_GET {
					t.Errorf("Expected NetworkPrivateEndpoint method GET, got %v", link.GetQuery().GetMethod())
				}
				if link.GetBlastPropagation().GetIn() != true || link.GetBlastPropagation().GetOut() != true {
					t.Errorf("Expected NetworkPrivateEndpoint BlastPropagation In=true, Out=true, got In=%v, Out=%v",
						link.GetBlastPropagation().GetIn(), link.GetBlastPropagation().GetOut())
				}
			}
		}

		if !foundPrivateEndpointConnectionLink {
			t.Error("Expected to find PrivateEndpointConnection child resource link")
		}

		// Verify both private endpoints are linked
		expectedPrivateEndpoints := map[string]string{
			"test-private-endpoint-1": subscriptionID + "." + resourceGroup,
			"test-private-endpoint-2": subscriptionID + ".different-rg",
		}

		if len(privateEndpointLinks) != len(expectedPrivateEndpoints) {
			t.Errorf("Expected %d NetworkPrivateEndpoint links, got %d: %v", len(expectedPrivateEndpoints), len(privateEndpointLinks), privateEndpointLinks)
		}

		for expectedName, expectedScope := range expectedPrivateEndpoints {
			if actualScope, found := privateEndpointLinks[expectedName]; !found {
				t.Errorf("Expected to find NetworkPrivateEndpoint link for '%s'", expectedName)
			} else if actualScope != expectedScope {
				t.Errorf("Expected NetworkPrivateEndpoint '%s' scope '%s', got '%s'", expectedName, expectedScope, actualScope)
			}
		}
	})

	t.Run("Get_WithKeyVault", func(t *testing.T) {
		keyID := "https://test-keyvault.vault.azure.net/keys/test-key/version"
		server := createAzureSqlServerWithKeyId(serverName, "", "test-server.database.windows.net", keyID)

		mockClient := mocks.NewMockSqlServersClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, serverName, nil).Return(
			armsql.ServersClientGetResponse{
				Server: *server,
			}, nil)

		testClient := &testSqlServersClient{MockSqlServersClient: mockClient}
		wrapper := manual.NewSqlServer(testClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper)

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], serverName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		// Verify KeyVault link exists
		foundKeyVaultLink := false
		for _, link := range sdpItem.GetLinkedItemQueries() {
			if link.GetQuery().GetType() == azureshared.KeyVaultVault.String() {
				foundKeyVaultLink = true
				if link.GetQuery().GetQuery() != "test-keyvault" {
					t.Errorf("Expected KeyVault name 'test-keyvault', got %s", link.GetQuery().GetQuery())
				}
				if link.GetQuery().GetScope() != subscriptionID+"."+resourceGroup {
					t.Errorf("Expected KeyVault scope %s, got %s", subscriptionID+"."+resourceGroup, link.GetQuery().GetScope())
				}
				if link.GetQuery().GetMethod() != sdp.QueryMethod_GET {
					t.Errorf("Expected KeyVault method GET, got %v", link.GetQuery().GetMethod())
				}
				if link.GetBlastPropagation().GetIn() != true || link.GetBlastPropagation().GetOut() != false {
					t.Errorf("Expected KeyVault BlastPropagation In=true, Out=false, got In=%v, Out=%v",
						link.GetBlastPropagation().GetIn(), link.GetBlastPropagation().GetOut())
				}
			}
		}

		if !foundKeyVaultLink {
			t.Error("Expected to find KeyVault link")
		}
	})

	t.Run("Get_WithCrossResourceGroupManagedIdentity", func(t *testing.T) {
		otherResourceGroup := "other-rg"
		identityID := "/subscriptions/" + subscriptionID + "/resourceGroups/" + otherResourceGroup + "/providers/Microsoft.ManagedIdentity/userAssignedIdentities/test-identity"
		server := createAzureSqlServer(serverName, identityID, "")

		mockClient := mocks.NewMockSqlServersClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, serverName, nil).Return(
			armsql.ServersClientGetResponse{
				Server: *server,
			}, nil)

		testClient := &testSqlServersClient{MockSqlServersClient: mockClient}
		wrapper := manual.NewSqlServer(testClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper)

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], serverName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		// Verify that managed identity link uses the correct scope from different resource group
		foundIdentityLink := false
		for _, link := range sdpItem.GetLinkedItemQueries() {
			if link.GetQuery().GetType() == azureshared.ManagedIdentityUserAssignedIdentity.String() {
				foundIdentityLink = true
				expectedScope := subscriptionID + "." + otherResourceGroup
				if link.GetQuery().GetScope() != expectedScope {
					t.Errorf("Expected identity scope %s, got %s", expectedScope, link.GetQuery().GetScope())
				}
				if link.GetQuery().GetQuery() != "test-identity" {
					t.Errorf("Expected identity name 'test-identity', got %s", link.GetQuery().GetQuery())
				}
				break
			}
		}

		if !foundIdentityLink {
			t.Error("Expected to find Managed Identity link")
		}
	})

	t.Run("Get_WithFQDNOnly", func(t *testing.T) {
		server := createAzureSqlServer(serverName, "", "test-server.database.windows.net")

		mockClient := mocks.NewMockSqlServersClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, serverName, nil).Return(
			armsql.ServersClientGetResponse{
				Server: *server,
			}, nil)

		testClient := &testSqlServersClient{MockSqlServersClient: mockClient}
		wrapper := manual.NewSqlServer(testClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper)

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], serverName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		// Verify DNS link exists
		foundDNSLink := false
		for _, link := range sdpItem.GetLinkedItemQueries() {
			if link.GetQuery().GetType() == stdlib.NetworkDNS.String() {
				foundDNSLink = true
				if link.GetQuery().GetQuery() != "test-server.database.windows.net" {
					t.Errorf("Expected DNS name 'test-server.database.windows.net', got %s", link.GetQuery().GetQuery())
				}
				if link.GetQuery().GetScope() != "global" {
					t.Errorf("Expected DNS scope 'global', got %s", link.GetQuery().GetScope())
				}
				break
			}
		}

		if !foundDNSLink {
			t.Error("Expected to find DNS link")
		}
	})

	t.Run("Get_InvalidQueryParts", func(t *testing.T) {
		mockClient := mocks.NewMockSqlServersClient(ctrl)
		testClient := &testSqlServersClient{MockSqlServersClient: mockClient}

		wrapper := manual.NewSqlServer(testClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper)

		// Test with insufficient query parts (no server name)
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "", true)
		if qErr == nil {
			t.Error("Expected error when providing empty server name, but got nil")
		}
	})

	t.Run("List", func(t *testing.T) {
		server1 := createAzureSqlServer("server-1", "", "")
		server2 := createAzureSqlServer("server-2", "", "")

		mockClient := mocks.NewMockSqlServersClient(ctrl)
		mockPager := &mockSqlServersPager{
			pages: []armsql.ServersClientListByResourceGroupResponse{
				{
					ServerListResult: armsql.ServerListResult{
						Value: []*armsql.Server{server1, server2},
					},
				},
			},
		}

		testClient := &testSqlServersClient{
			MockSqlServersClient: mockClient,
			pager:                mockPager,
		}

		wrapper := manual.NewSqlServer(testClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper)

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
			if err := item.Validate(); err != nil {
				t.Fatalf("Expected no validation error, got: %v", err)
			}

			if item.GetType() != azureshared.SQLServer.String() {
				t.Errorf("Expected type %s, got %s", azureshared.SQLServer, item.GetType())
			}

			if item.GetTags()["env"] != "test" {
				t.Fatalf("Expected tag 'env=test', got: %s", item.GetTags()["env"])
			}
		}
	})

	t.Run("List_WithNilName", func(t *testing.T) {
		server1 := createAzureSqlServer("server-1", "", "")
		server2 := &armsql.Server{
			Name:     nil, // Server with nil name should be skipped
			Location: to.Ptr("eastus"),
			Tags: map[string]*string{
				"env": to.Ptr("test"),
			},
			Properties: &armsql.ServerProperties{
				Version: to.Ptr("12.0"),
			},
		}

		mockClient := mocks.NewMockSqlServersClient(ctrl)
		mockPager := &mockSqlServersPager{
			pages: []armsql.ServersClientListByResourceGroupResponse{
				{
					ServerListResult: armsql.ServerListResult{
						Value: []*armsql.Server{server1, server2},
					},
				},
			},
		}

		testClient := &testSqlServersClient{
			MockSqlServersClient: mockClient,
			pager:                mockPager,
		}

		wrapper := manual.NewSqlServer(testClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper)

		listable, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support List operation")
		}

		sdpItems, err := listable.List(ctx, wrapper.Scopes()[0], true)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		// Should only return 1 item (server with nil name is skipped)
		if len(sdpItems) != 1 {
			t.Fatalf("Expected 1 item (nil name filtered out), got: %d", len(sdpItems))
		}

		if sdpItems[0].UniqueAttributeValue() != "server-1" {
			t.Fatalf("Expected server name 'server-1', got: %s", sdpItems[0].UniqueAttributeValue())
		}
	})

	t.Run("ListStream", func(t *testing.T) {
		server1 := createAzureSqlServer("server-1", "", "")
		server2 := createAzureSqlServer("server-2", "", "")

		mockClient := mocks.NewMockSqlServersClient(ctrl)
		mockPager := &mockSqlServersPager{
			pages: []armsql.ServersClientListByResourceGroupResponse{
				{
					ServerListResult: armsql.ServerListResult{
						Value: []*armsql.Server{server1, server2},
					},
				},
			},
		}

		testClient := &testSqlServersClient{
			MockSqlServersClient: mockClient,
			pager:                mockPager,
		}

		wrapper := manual.NewSqlServer(testClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper)

		wg := &sync.WaitGroup{}
		wg.Add(2) // we added two items

		var items []*sdp.Item
		mockItemHandler := func(item *sdp.Item) {
			items = append(items, item)
			wg.Done() // signal that we processed an item
		}

		var errs []error
		mockErrorHandler := func(err error) {
			errs = append(errs, err)
		}

		stream := discovery.NewQueryResultStream(mockItemHandler, mockErrorHandler)

		// Check if adapter supports list streaming
		listStreamable, ok := adapter.(discovery.ListStreamableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support ListStream operation")
		}

		listStreamable.ListStream(ctx, wrapper.Scopes()[0], true, stream)
		wg.Wait()

		if len(errs) != 0 {
			t.Fatalf("Expected no errors, got: %v", errs)
		}

		if len(items) != 2 {
			t.Fatalf("Expected 2 items, got: %d", len(items))
		}

		// Verify adapter doesn't support SearchStream
		_, ok = adapter.(discovery.SearchStreamableAdapter)
		if ok {
			t.Fatalf("Adapter should not support SearchStream operation")
		}
	})

	t.Run("ErrorHandling_Get", func(t *testing.T) {
		expectedErr := errors.New("server not found")

		mockClient := mocks.NewMockSqlServersClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, "nonexistent-server", nil).Return(
			armsql.ServersClientGetResponse{}, expectedErr)

		testClient := &testSqlServersClient{MockSqlServersClient: mockClient}
		wrapper := manual.NewSqlServer(testClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper)

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "nonexistent-server", true)
		if qErr == nil {
			t.Error("Expected error when getting non-existent server, but got nil")
		}
	})

	t.Run("ErrorHandling_List", func(t *testing.T) {
		mockClient := mocks.NewMockSqlServersClient(ctrl)
		// Create a pager that returns an error when NextPage is called
		errorPager := &errorSqlServersPager{}

		testClient := &testSqlServersClient{
			MockSqlServersClient: mockClient,
			pager:                errorPager,
		}

		wrapper := manual.NewSqlServer(testClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper)

		listable, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support List operation")
		}

		_, err := listable.List(ctx, wrapper.Scopes()[0], true)
		// The List implementation should return an error when pager.NextPage returns an error
		if err == nil {
			t.Error("Expected error from pager when NextPage returns an error, but got nil")
		}
	})

	t.Run("ErrorHandling_ListStream", func(t *testing.T) {
		mockClient := mocks.NewMockSqlServersClient(ctrl)
		// Create a pager that returns an error when NextPage is called
		errorPager := &errorSqlServersPager{}

		testClient := &testSqlServersClient{
			MockSqlServersClient: mockClient,
			pager:                errorPager,
		}

		wrapper := manual.NewSqlServer(testClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper)

		var errs []error
		mockErrorHandler := func(err error) {
			errs = append(errs, err)
		}

		stream := discovery.NewQueryResultStream(func(item *sdp.Item) {}, mockErrorHandler)

		listStreamable, ok := adapter.(discovery.ListStreamableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support ListStream operation")
		}

		listStreamable.ListStream(ctx, wrapper.Scopes()[0], true, stream)

		// Should have received an error
		if len(errs) == 0 {
			t.Error("Expected error from pager when NextPage returns an error, but got none")
		}
	})

	t.Run("InterfaceCompliance", func(t *testing.T) {
		mockClient := mocks.NewMockSqlServersClient(ctrl)
		testClient := &testSqlServersClient{MockSqlServersClient: mockClient}
		wrapper := manual.NewSqlServer(testClient, subscriptionID, resourceGroup)

		// Cast to sources.Wrapper to access interface methods
		w := wrapper.(sources.Wrapper)

		// Verify IAMPermissions
		permissions := w.IAMPermissions()
		if len(permissions) == 0 {
			t.Error("Expected IAMPermissions to return at least one permission")
		}
		expectedPermission := "Microsoft.Sql/servers/read"
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

		// Verify PredefinedRole
		// PredefinedRole is available on the wrapper, not the adapter
		if roleInterface, ok := interface{}(wrapper).(interface{ PredefinedRole() string }); ok {
			role := roleInterface.PredefinedRole()
			if role != "Reader" {
				t.Errorf("Expected PredefinedRole to be 'Reader', got %s", role)
			}
		} else {
			t.Error("Wrapper does not implement PredefinedRole method")
		}

		// Verify PotentialLinks
		potentialLinks := w.PotentialLinks()
		if len(potentialLinks) == 0 {
			t.Error("Expected PotentialLinks to return at least one link")
		}
		expectedLinks := []shared.ItemType{
			azureshared.SQLDatabase,
			azureshared.SQLElasticPool,
			azureshared.ManagedIdentityUserAssignedIdentity,
			azureshared.NetworkPrivateEndpoint,
			azureshared.KeyVaultVault,
			stdlib.NetworkDNS,
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
			if mapping.GetTerraformQueryMap() == "azurerm_mssql_server.name" {
				foundMapping = true
				break
			}
		}
		if !foundMapping {
			t.Error("Expected TerraformMappings to include 'azurerm_mssql_server.name' mapping")
		}
	})
}

// createAzureSqlServer creates a mock Azure SQL Server for testing
func createAzureSqlServer(serverName, primaryUserAssignedIdentityID, fullyQualifiedDomainName string) *armsql.Server {
	serverID := "/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.Sql/servers/" + serverName

	server := &armsql.Server{
		Name:     to.Ptr(serverName),
		Location: to.Ptr("eastus"),
		Tags: map[string]*string{
			"env":     to.Ptr("test"),
			"project": to.Ptr("testing"),
		},
		ID: to.Ptr(serverID),
		Properties: &armsql.ServerProperties{
			Version:                  to.Ptr("12.0"),
			AdministratorLogin:       to.Ptr("admin"),
			FullyQualifiedDomainName: to.Ptr(fullyQualifiedDomainName),
		},
	}

	if primaryUserAssignedIdentityID != "" {
		server.Properties.PrimaryUserAssignedIdentityID = to.Ptr(primaryUserAssignedIdentityID)
	}

	if fullyQualifiedDomainName == "" && serverName != "" {
		// Set a default FQDN if not provided but server name is set
		server.Properties.FullyQualifiedDomainName = to.Ptr(serverName + ".database.windows.net")
	}

	return server
}

// createAzureSqlServerWithUserAssignedIdentities creates a mock Azure SQL Server with UserAssignedIdentities
func createAzureSqlServerWithUserAssignedIdentities(serverName, primaryUserAssignedIdentityID, fullyQualifiedDomainName string, userAssignedIdentities map[string]*armsql.UserIdentity) *armsql.Server {
	server := createAzureSqlServer(serverName, primaryUserAssignedIdentityID, fullyQualifiedDomainName)
	if userAssignedIdentities != nil {
		server.Identity = &armsql.ResourceIdentity{
			Type:                   to.Ptr(armsql.IdentityTypeUserAssigned),
			UserAssignedIdentities: userAssignedIdentities,
		}
	}
	return server
}

// createAzureSqlServerWithPrivateEndpointConnections creates a mock Azure SQL Server with PrivateEndpointConnections
func createAzureSqlServerWithPrivateEndpointConnections(serverName, primaryUserAssignedIdentityID, fullyQualifiedDomainName string, privateEndpointConnections []*armsql.ServerPrivateEndpointConnection) *armsql.Server {
	server := createAzureSqlServer(serverName, primaryUserAssignedIdentityID, fullyQualifiedDomainName)
	if privateEndpointConnections != nil {
		server.Properties.PrivateEndpointConnections = privateEndpointConnections
	}
	return server
}

// createAzureSqlServerWithKeyId creates a mock Azure SQL Server with KeyId encryption property
func createAzureSqlServerWithKeyId(serverName, primaryUserAssignedIdentityID, fullyQualifiedDomainName, keyID string) *armsql.Server {
	server := createAzureSqlServer(serverName, primaryUserAssignedIdentityID, fullyQualifiedDomainName)
	if keyID != "" {
		server.Properties.KeyID = to.Ptr(keyID)
	}
	return server
}
