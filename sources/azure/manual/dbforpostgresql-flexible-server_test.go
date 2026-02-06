package manual_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/postgresql/armpostgresqlflexibleservers/v5"
	"go.uber.org/mock/gomock"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	"github.com/overmindtech/cli/sources/azure/manual"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/azure/shared/mocks"
	"github.com/overmindtech/cli/sources/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

// mockPostgreSQLFlexibleServersPager is a simple mock implementation of PostgreSQLFlexibleServersPager
type mockPostgreSQLFlexibleServersPager struct {
	pages []armpostgresqlflexibleservers.ServersClientListByResourceGroupResponse
	index int
}

func (m *mockPostgreSQLFlexibleServersPager) More() bool {
	return m.index < len(m.pages)
}

func (m *mockPostgreSQLFlexibleServersPager) NextPage(ctx context.Context) (armpostgresqlflexibleservers.ServersClientListByResourceGroupResponse, error) {
	if m.index >= len(m.pages) {
		return armpostgresqlflexibleservers.ServersClientListByResourceGroupResponse{}, errors.New("no more pages")
	}
	page := m.pages[m.index]
	m.index++
	return page, nil
}

// errorPostgreSQLFlexibleServersPager is a mock pager that always returns an error
type errorPostgreSQLFlexibleServersPager struct{}

func (e *errorPostgreSQLFlexibleServersPager) More() bool {
	return true // Always return true so NextPage will be called
}

func (e *errorPostgreSQLFlexibleServersPager) NextPage(ctx context.Context) (armpostgresqlflexibleservers.ServersClientListByResourceGroupResponse, error) {
	return armpostgresqlflexibleservers.ServersClientListByResourceGroupResponse{}, errors.New("pager error")
}

// testPostgreSQLFlexibleServersClient wraps the mock to implement the correct interface
type testPostgreSQLFlexibleServersClient struct {
	*mocks.MockPostgreSQLFlexibleServersClient
	pager clients.PostgreSQLFlexibleServersPager
}

func (t *testPostgreSQLFlexibleServersClient) ListByResourceGroup(ctx context.Context, resourceGroupName string, options *armpostgresqlflexibleservers.ServersClientListByResourceGroupOptions) clients.PostgreSQLFlexibleServersPager {
	return t.pager
}

func TestDBforPostgreSQLFlexibleServer(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"
	serverName := "test-server"

	t.Run("Get", func(t *testing.T) {
		server := createAzurePostgreSQLFlexibleServer(serverName, "", "")

		mockClient := mocks.NewMockPostgreSQLFlexibleServersClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, serverName, nil).Return(
			armpostgresqlflexibleservers.ServersClientGetResponse{
				Server: *server,
			}, nil)

		testClient := &testPostgreSQLFlexibleServersClient{MockPostgreSQLFlexibleServersClient: mockClient}
		wrapper := manual.NewDBforPostgreSQLFlexibleServer(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], serverName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.DBforPostgreSQLFlexibleServer.String() {
			t.Errorf("Expected type %s, got %s", azureshared.DBforPostgreSQLFlexibleServer, sdpItem.GetType())
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
					ExpectedType:   azureshared.DBforPostgreSQLDatabase.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  serverName,
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					ExpectedType:   azureshared.DBforPostgreSQLFlexibleServerFirewallRule.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  serverName,
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					ExpectedType:   azureshared.DBforPostgreSQLFlexibleServerConfiguration.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  serverName,
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					ExpectedType:   azureshared.DBforPostgreSQLFlexibleServerAdministrator.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  serverName,
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					ExpectedType:   azureshared.DBforPostgreSQLFlexibleServerPrivateEndpointConnection.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  serverName,
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					ExpectedType:   azureshared.DBforPostgreSQLFlexibleServerPrivateLinkResource.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  serverName,
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					ExpectedType:   azureshared.DBforPostgreSQLFlexibleServerReplica.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  serverName,
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					ExpectedType:   azureshared.DBforPostgreSQLFlexibleServerMigration.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  serverName,
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					ExpectedType:   azureshared.DBforPostgreSQLFlexibleServerBackup.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  serverName,
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					ExpectedType:   azureshared.DBforPostgreSQLFlexibleServerVirtualEndpoint.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  serverName,
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
			}

			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("Get_WithSubnet", func(t *testing.T) {
		subnetID := "/subscriptions/sub-id/resourceGroups/vnet-rg/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/test-subnet"
		server := createAzurePostgreSQLFlexibleServer(serverName, subnetID, "")

		mockClient := mocks.NewMockPostgreSQLFlexibleServersClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, serverName, nil).Return(
			armpostgresqlflexibleservers.ServersClientGetResponse{
				Server: *server,
			}, nil)

		testClient := &testPostgreSQLFlexibleServersClient{MockPostgreSQLFlexibleServersClient: mockClient}
		wrapper := manual.NewDBforPostgreSQLFlexibleServer(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], serverName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		// Verify subnet and virtual network links are present
		foundSubnetLink := false
		foundVNetLink := false
		for _, linkedQuery := range sdpItem.GetLinkedItemQueries() {
			if linkedQuery.GetQuery().GetType() == azureshared.NetworkSubnet.String() {
				foundSubnetLink = true
				if linkedQuery.GetQuery().GetMethod() != sdp.QueryMethod_GET {
					t.Errorf("Expected subnet link to use GET method, got %v", linkedQuery.GetQuery().GetMethod())
				}
				if linkedQuery.GetQuery().GetScope() != "sub-id.vnet-rg" {
					t.Errorf("Expected subnet link scope to be 'sub-id.vnet-rg', got %s", linkedQuery.GetQuery().GetScope())
				}
				if !linkedQuery.GetBlastPropagation().GetIn() {
					t.Error("Expected subnet link to have In=true")
				}
				if linkedQuery.GetBlastPropagation().GetOut() {
					t.Error("Expected subnet link to have Out=false")
				}
			}
			if linkedQuery.GetQuery().GetType() == azureshared.NetworkVirtualNetwork.String() {
				foundVNetLink = true
				if linkedQuery.GetQuery().GetMethod() != sdp.QueryMethod_GET {
					t.Errorf("Expected virtual network link to use GET method, got %v", linkedQuery.GetQuery().GetMethod())
				}
				if linkedQuery.GetQuery().GetScope() != "sub-id.vnet-rg" {
					t.Errorf("Expected virtual network link scope to be 'sub-id.vnet-rg', got %s", linkedQuery.GetQuery().GetScope())
				}
				if !linkedQuery.GetBlastPropagation().GetIn() {
					t.Error("Expected virtual network link to have In=true")
				}
				if linkedQuery.GetBlastPropagation().GetOut() {
					t.Error("Expected virtual network link to have Out=false")
				}
			}
		}

		if !foundSubnetLink {
			t.Error("Expected to find subnet link in linked item queries")
		}
		if !foundVNetLink {
			t.Error("Expected to find virtual network link in linked item queries")
		}
	})

	t.Run("Get_WithFQDN", func(t *testing.T) {
		fqdn := "test-server.postgres.database.azure.com"
		server := createAzurePostgreSQLFlexibleServer(serverName, "", fqdn)

		mockClient := mocks.NewMockPostgreSQLFlexibleServersClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, serverName, nil).Return(
			armpostgresqlflexibleservers.ServersClientGetResponse{
				Server: *server,
			}, nil)

		testClient := &testPostgreSQLFlexibleServersClient{MockPostgreSQLFlexibleServersClient: mockClient}
		wrapper := manual.NewDBforPostgreSQLFlexibleServer(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], serverName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		// Verify DNS link is present
		foundDNSLink := false
		for _, linkedQuery := range sdpItem.GetLinkedItemQueries() {
			if linkedQuery.GetQuery().GetType() == stdlib.NetworkDNS.String() {
				foundDNSLink = true
				if linkedQuery.GetQuery().GetMethod() != sdp.QueryMethod_SEARCH {
					t.Errorf("Expected DNS link to use SEARCH method, got %v", linkedQuery.GetQuery().GetMethod())
				}
				if linkedQuery.GetQuery().GetQuery() != fqdn {
					t.Errorf("Expected DNS link query to be %s, got %s", fqdn, linkedQuery.GetQuery().GetQuery())
				}
				if linkedQuery.GetQuery().GetScope() != "global" {
					t.Errorf("Expected DNS link scope to be 'global', got %s", linkedQuery.GetQuery().GetScope())
				}
				if !linkedQuery.GetBlastPropagation().GetIn() {
					t.Error("Expected DNS link to have In=true")
				}
				if !linkedQuery.GetBlastPropagation().GetOut() {
					t.Error("Expected DNS link to have Out=true")
				}
			}
		}

		if !foundDNSLink {
			t.Error("Expected to find DNS link in linked item queries")
		}
	})

	t.Run("Get_InvalidQueryParts", func(t *testing.T) {
		mockClient := mocks.NewMockPostgreSQLFlexibleServersClient(ctrl)
		testClient := &testPostgreSQLFlexibleServersClient{MockPostgreSQLFlexibleServersClient: mockClient}

		wrapper := manual.NewDBforPostgreSQLFlexibleServer(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		// Test with empty query
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "", true)
		if qErr == nil {
			t.Error("Expected error when providing empty query, but got nil")
		}
	})

	t.Run("List", func(t *testing.T) {
		server1 := createAzurePostgreSQLFlexibleServer("server-1", "", "")
		server2 := createAzurePostgreSQLFlexibleServer("server-2", "", "")

		mockClient := mocks.NewMockPostgreSQLFlexibleServersClient(ctrl)
		mockPager := &mockPostgreSQLFlexibleServersPager{
			pages: []armpostgresqlflexibleservers.ServersClientListByResourceGroupResponse{
				{
					ServerList: armpostgresqlflexibleservers.ServerList{
						Value: []*armpostgresqlflexibleservers.Server{server1, server2},
					},
				},
			},
		}

		testClient := &testPostgreSQLFlexibleServersClient{
			MockPostgreSQLFlexibleServersClient: mockClient,
			pager:                               mockPager,
		}

		wrapper := manual.NewDBforPostgreSQLFlexibleServer(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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
			if err := item.Validate(); err != nil {
				t.Fatalf("Expected no validation error, got: %v", err)
			}

			if item.GetType() != azureshared.DBforPostgreSQLFlexibleServer.String() {
				t.Errorf("Expected type %s, got %s", azureshared.DBforPostgreSQLFlexibleServer, item.GetType())
			}
		}
	})

	t.Run("List_WithNilName", func(t *testing.T) {
		server1 := createAzurePostgreSQLFlexibleServer("server-1", "", "")
		server2 := &armpostgresqlflexibleservers.Server{
			Name: nil, // Server with nil name should be skipped
			Properties: &armpostgresqlflexibleservers.ServerProperties{
				Version: to.Ptr(armpostgresqlflexibleservers.PostgresMajorVersion("14")),
			},
		}

		mockClient := mocks.NewMockPostgreSQLFlexibleServersClient(ctrl)
		mockPager := &mockPostgreSQLFlexibleServersPager{
			pages: []armpostgresqlflexibleservers.ServersClientListByResourceGroupResponse{
				{
					ServerList: armpostgresqlflexibleservers.ServerList{
						Value: []*armpostgresqlflexibleservers.Server{server1, server2},
					},
				},
			},
		}

		testClient := &testPostgreSQLFlexibleServersClient{
			MockPostgreSQLFlexibleServersClient: mockClient,
			pager:                               mockPager,
		}

		wrapper := manual.NewDBforPostgreSQLFlexibleServer(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

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

	t.Run("ErrorHandling_Get", func(t *testing.T) {
		expectedErr := errors.New("server not found")

		mockClient := mocks.NewMockPostgreSQLFlexibleServersClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, "nonexistent-server", nil).Return(
			armpostgresqlflexibleservers.ServersClientGetResponse{}, expectedErr)

		testClient := &testPostgreSQLFlexibleServersClient{MockPostgreSQLFlexibleServersClient: mockClient}
		wrapper := manual.NewDBforPostgreSQLFlexibleServer(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "nonexistent-server", true)
		if qErr == nil {
			t.Error("Expected error when getting non-existent server, but got nil")
		}
	})

	t.Run("ErrorHandling_List", func(t *testing.T) {
		mockClient := mocks.NewMockPostgreSQLFlexibleServersClient(ctrl)
		// Create a pager that returns an error when NextPage is called
		errorPager := &errorPostgreSQLFlexibleServersPager{}

		testClient := &testPostgreSQLFlexibleServersClient{
			MockPostgreSQLFlexibleServersClient: mockClient,
			pager:                               errorPager,
		}

		wrapper := manual.NewDBforPostgreSQLFlexibleServer(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		listable, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support List operation")
		}

		_, err := listable.List(ctx, wrapper.Scopes()[0], true)
		// The List implementation should return an error when pager.NextPage returns an error
		if err == nil {
			t.Error("Expected error when pager returns error, but got nil")
		}
	})

	t.Run("Get_WithDataEncryption", func(t *testing.T) {
		// Create server with DataEncryption fields
		server := createAzurePostgreSQLFlexibleServer(serverName, "", "")
		primaryKeyURI := "https://test-vault.vault.azure.net/keys/test-key/abc123"
		primaryIdentityID := "/subscriptions/sub-id/resourceGroups/rg-id/providers/Microsoft.ManagedIdentity/userAssignedIdentities/primary-identity"
		geoBackupKeyURI := "https://geo-vault.vault.azure.net/keys/geo-key/def456"
		geoBackupIdentityID := "/subscriptions/sub-id/resourceGroups/rg-id/providers/Microsoft.ManagedIdentity/userAssignedIdentities/geo-identity"

		server.Properties.DataEncryption = &armpostgresqlflexibleservers.DataEncryption{
			PrimaryKeyURI:                   to.Ptr(primaryKeyURI),
			PrimaryUserAssignedIdentityID:   to.Ptr(primaryIdentityID),
			GeoBackupKeyURI:                 to.Ptr(geoBackupKeyURI),
			GeoBackupUserAssignedIdentityID: to.Ptr(geoBackupIdentityID),
		}

		mockClient := mocks.NewMockPostgreSQLFlexibleServersClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, serverName, nil).Return(
			armpostgresqlflexibleservers.ServersClientGetResponse{
				Server: *server,
			}, nil)

		testClient := &testPostgreSQLFlexibleServersClient{MockPostgreSQLFlexibleServersClient: mockClient}
		wrapper := manual.NewDBforPostgreSQLFlexibleServer(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], serverName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		// Verify DataEncryption links are present
		foundPrimaryIdentityLink := false
		foundPrimaryKeyVaultLink := false
		foundPrimaryKeyLink := false
		foundGeoBackupVaultLink := false
		foundGeoBackupKeyLink := false
		foundGeoBackupIdentityLink := false

		for _, linkedQuery := range sdpItem.GetLinkedItemQueries() {
			// Primary User Assigned Identity
			if linkedQuery.GetQuery().GetType() == azureshared.ManagedIdentityUserAssignedIdentity.String() &&
				linkedQuery.GetQuery().GetQuery() == "primary-identity" {
				foundPrimaryIdentityLink = true
				if linkedQuery.GetQuery().GetMethod() != sdp.QueryMethod_GET {
					t.Errorf("Expected primary identity link to use GET method, got %v", linkedQuery.GetQuery().GetMethod())
				}
				if !linkedQuery.GetBlastPropagation().GetIn() {
					t.Error("Expected primary identity link to have In=true")
				}
			}
			// Primary Key Vault Vault
			if linkedQuery.GetQuery().GetType() == azureshared.KeyVaultVault.String() &&
				linkedQuery.GetQuery().GetQuery() == "test-vault" {
				foundPrimaryKeyVaultLink = true
				if linkedQuery.GetQuery().GetMethod() != sdp.QueryMethod_GET {
					t.Errorf("Expected primary vault link to use GET method, got %v", linkedQuery.GetQuery().GetMethod())
				}
			}
			// Primary Key Vault Key
			if linkedQuery.GetQuery().GetType() == azureshared.KeyVaultKey.String() &&
				linkedQuery.GetQuery().GetQuery() == shared.CompositeLookupKey("test-vault", "test-key") {
				foundPrimaryKeyLink = true
				if linkedQuery.GetQuery().GetMethod() != sdp.QueryMethod_GET {
					t.Errorf("Expected primary key link to use GET method, got %v", linkedQuery.GetQuery().GetMethod())
				}
			}
			// Geo Backup Key Vault Vault
			if linkedQuery.GetQuery().GetType() == azureshared.KeyVaultVault.String() &&
				linkedQuery.GetQuery().GetQuery() == "geo-vault" {
				foundGeoBackupVaultLink = true
				if linkedQuery.GetQuery().GetMethod() != sdp.QueryMethod_GET {
					t.Errorf("Expected geo backup vault link to use GET method, got %v", linkedQuery.GetQuery().GetMethod())
				}
			}
			// Geo Backup Key Vault Key
			if linkedQuery.GetQuery().GetType() == azureshared.KeyVaultKey.String() &&
				linkedQuery.GetQuery().GetQuery() == shared.CompositeLookupKey("geo-vault", "geo-key") {
				foundGeoBackupKeyLink = true
				if linkedQuery.GetQuery().GetMethod() != sdp.QueryMethod_GET {
					t.Errorf("Expected geo backup key link to use GET method, got %v", linkedQuery.GetQuery().GetMethod())
				}
			}
			// Geo Backup User Assigned Identity
			if linkedQuery.GetQuery().GetType() == azureshared.ManagedIdentityUserAssignedIdentity.String() &&
				linkedQuery.GetQuery().GetQuery() == "geo-identity" {
				foundGeoBackupIdentityLink = true
				if linkedQuery.GetQuery().GetMethod() != sdp.QueryMethod_GET {
					t.Errorf("Expected geo backup identity link to use GET method, got %v", linkedQuery.GetQuery().GetMethod())
				}
				if !linkedQuery.GetBlastPropagation().GetIn() {
					t.Error("Expected geo backup identity link to have In=true")
				}
			}
		}

		if !foundPrimaryIdentityLink {
			t.Error("Expected to find primary user assigned identity link in linked item queries")
		}
		if !foundPrimaryKeyVaultLink {
			t.Error("Expected to find primary key vault vault link in linked item queries")
		}
		if !foundPrimaryKeyLink {
			t.Error("Expected to find primary key vault key link in linked item queries")
		}
		if !foundGeoBackupVaultLink {
			t.Error("Expected to find geo backup key vault vault link in linked item queries")
		}
		if !foundGeoBackupKeyLink {
			t.Error("Expected to find geo backup key vault key link in linked item queries")
		}
		if !foundGeoBackupIdentityLink {
			t.Error("Expected to find geo backup user assigned identity link in linked item queries")
		}
	})

	t.Run("Get_WithSourceServer", func(t *testing.T) {
		// Create a replica server with SourceServerResourceID
		replicaServerName := "replica-server"
		sourceServerID := "/subscriptions/sub-id/resourceGroups/source-rg/providers/Microsoft.DBforPostgreSQL/flexibleServers/source-server"
		server := createAzurePostgreSQLFlexibleServer(replicaServerName, "", "")
		server.Properties.SourceServerResourceID = to.Ptr(sourceServerID)
		server.Properties.ReplicationRole = to.Ptr(armpostgresqlflexibleservers.ReplicationRoleAsyncReplica)

		mockClient := mocks.NewMockPostgreSQLFlexibleServersClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, replicaServerName, nil).Return(
			armpostgresqlflexibleservers.ServersClientGetResponse{
				Server: *server,
			}, nil)

		testClient := &testPostgreSQLFlexibleServersClient{MockPostgreSQLFlexibleServersClient: mockClient}
		wrapper := manual.NewDBforPostgreSQLFlexibleServer(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], replicaServerName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		// Verify source server link is present
		foundSourceServerLink := false
		for _, linkedQuery := range sdpItem.GetLinkedItemQueries() {
			if linkedQuery.GetQuery().GetType() == azureshared.DBforPostgreSQLFlexibleServer.String() &&
				linkedQuery.GetQuery().GetQuery() == "source-server" {
				foundSourceServerLink = true
				if linkedQuery.GetQuery().GetMethod() != sdp.QueryMethod_GET {
					t.Errorf("Expected source server link to use GET method, got %v", linkedQuery.GetQuery().GetMethod())
				}
				if linkedQuery.GetQuery().GetScope() != "sub-id.source-rg" {
					t.Errorf("Expected source server link scope to be 'sub-id.source-rg', got %s", linkedQuery.GetQuery().GetScope())
				}
				if !linkedQuery.GetBlastPropagation().GetIn() {
					t.Error("Expected source server link to have In=true")
				}
				if linkedQuery.GetBlastPropagation().GetOut() {
					t.Error("Expected source server link to have Out=false")
				}
			}
		}

		if !foundSourceServerLink {
			t.Error("Expected to find source server link in linked item queries")
		}
	})

	t.Run("PotentialLinks", func(t *testing.T) {
		mockClient := mocks.NewMockPostgreSQLFlexibleServersClient(ctrl)
		testClient := &testPostgreSQLFlexibleServersClient{MockPostgreSQLFlexibleServersClient: mockClient}

		wrapper := manual.NewDBforPostgreSQLFlexibleServer(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		potentialLinks := wrapper.PotentialLinks()

		expectedLinks := map[shared.ItemType]bool{
			azureshared.NetworkSubnet:                                          true,
			azureshared.NetworkVirtualNetwork:                                  true,
			azureshared.NetworkPrivateDNSZone:                                  true,
			azureshared.NetworkPrivateEndpoint:                                 true,
			azureshared.DBforPostgreSQLDatabase:                                true,
			azureshared.DBforPostgreSQLFlexibleServerFirewallRule:              true,
			azureshared.DBforPostgreSQLFlexibleServerConfiguration:             true,
			azureshared.DBforPostgreSQLFlexibleServerAdministrator:             true,
			azureshared.DBforPostgreSQLFlexibleServerPrivateEndpointConnection: true,
			azureshared.DBforPostgreSQLFlexibleServerPrivateLinkResource:       true,
			azureshared.DBforPostgreSQLFlexibleServerReplica:                   true,
			azureshared.DBforPostgreSQLFlexibleServerMigration:                 true,
			azureshared.DBforPostgreSQLFlexibleServerBackup:                    true,
			azureshared.DBforPostgreSQLFlexibleServerVirtualEndpoint:           true,
			azureshared.DBforPostgreSQLFlexibleServer:                          true, // For replica-to-source server relationship
			stdlib.NetworkDNS: true,
			azureshared.ManagedIdentityUserAssignedIdentity: true,
			azureshared.KeyVaultVault:                       true,
			azureshared.KeyVaultKey:                         true,
		}

		for expectedType, expectedValue := range expectedLinks {
			if actualValue, exists := potentialLinks[expectedType]; !exists {
				t.Errorf("Expected PotentialLinks to include %s, but it was not found", expectedType)
			} else if actualValue != expectedValue {
				t.Errorf("Expected PotentialLinks[%s] to be %v, got %v", expectedType, expectedValue, actualValue)
			}
		}
	})
}

// createAzurePostgreSQLFlexibleServer creates a mock Azure PostgreSQL Flexible Server for testing
func createAzurePostgreSQLFlexibleServer(serverName, subnetID, fqdn string) *armpostgresqlflexibleservers.Server {
	serverID := "/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.DBforPostgreSQL/flexibleServers/" + serverName

	server := &armpostgresqlflexibleservers.Server{
		Name:     to.Ptr(serverName),
		ID:       to.Ptr(serverID),
		Location: to.Ptr("eastus"),
		Properties: &armpostgresqlflexibleservers.ServerProperties{
			Version: to.Ptr(armpostgresqlflexibleservers.PostgresMajorVersion("14")),
			State:   to.Ptr(armpostgresqlflexibleservers.ServerStateReady),
		},
		SKU: &armpostgresqlflexibleservers.SKU{
			Name: to.Ptr("Standard_B1ms"),
			Tier: to.Ptr(armpostgresqlflexibleservers.SKUTierBurstable),
		},
		Tags: map[string]*string{
			"env": to.Ptr("test"),
		},
	}

	// Add network configuration if subnet ID is provided
	if subnetID != "" {
		server.Properties.Network = &armpostgresqlflexibleservers.Network{
			DelegatedSubnetResourceID: to.Ptr(subnetID),
		}
	}

	// Add FQDN if provided
	if fqdn != "" {
		server.Properties.FullyQualifiedDomainName = to.Ptr(fqdn)
	}

	return server
}
