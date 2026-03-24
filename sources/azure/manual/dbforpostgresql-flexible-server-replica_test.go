package manual_test

import (
	"context"
	"errors"
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

type mockDBforPostgreSQLFlexibleServerReplicaPager struct {
	pages []armpostgresqlflexibleservers.ReplicasClientListByServerResponse
	index int
}

func (m *mockDBforPostgreSQLFlexibleServerReplicaPager) More() bool {
	return m.index < len(m.pages)
}

func (m *mockDBforPostgreSQLFlexibleServerReplicaPager) NextPage(ctx context.Context) (armpostgresqlflexibleservers.ReplicasClientListByServerResponse, error) {
	if m.index >= len(m.pages) {
		return armpostgresqlflexibleservers.ReplicasClientListByServerResponse{}, errors.New("no more pages")
	}
	page := m.pages[m.index]
	m.index++
	return page, nil
}

type errorDBforPostgreSQLFlexibleServerReplicaPager struct{}

func (e *errorDBforPostgreSQLFlexibleServerReplicaPager) More() bool {
	return true
}

func (e *errorDBforPostgreSQLFlexibleServerReplicaPager) NextPage(ctx context.Context) (armpostgresqlflexibleservers.ReplicasClientListByServerResponse, error) {
	return armpostgresqlflexibleservers.ReplicasClientListByServerResponse{}, errors.New("pager error")
}

type testDBforPostgreSQLFlexibleServerReplicaClient struct {
	*mocks.MockDBforPostgreSQLFlexibleServerReplicaClient
	pager clients.DBforPostgreSQLFlexibleServerReplicaPager
}

func (t *testDBforPostgreSQLFlexibleServerReplicaClient) ListByServer(ctx context.Context, resourceGroupName, serverName string) clients.DBforPostgreSQLFlexibleServerReplicaPager {
	return t.pager
}

func TestDBforPostgreSQLFlexibleServerReplica(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"
	serverName := "test-server"
	replicaName := "test-replica"

	t.Run("Get", func(t *testing.T) {
		replica := createAzurePostgreSQLFlexibleServerReplica(serverName, replicaName)

		mockClient := mocks.NewMockDBforPostgreSQLFlexibleServerReplicaClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, replicaName).Return(
			armpostgresqlflexibleservers.ServersClientGetResponse{
				Server: *replica,
			}, nil)

		wrapper := manual.NewDBforPostgreSQLFlexibleServerReplica(&testDBforPostgreSQLFlexibleServerReplicaClient{MockDBforPostgreSQLFlexibleServerReplicaClient: mockClient}, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(serverName, replicaName)
		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.DBforPostgreSQLFlexibleServerReplica.String() {
			t.Errorf("Expected type %s, got %s", azureshared.DBforPostgreSQLFlexibleServerReplica, sdpItem.GetType())
		}

		if sdpItem.GetUniqueAttribute() != "uniqueAttr" {
			t.Errorf("Expected unique attribute 'uniqueAttr', got %s", sdpItem.GetUniqueAttribute())
		}

		expectedUniqueAttrValue := shared.CompositeLookupKey(serverName, replicaName)
		if sdpItem.UniqueAttributeValue() != expectedUniqueAttrValue {
			t.Errorf("Expected unique attribute value %s, got %s", expectedUniqueAttrValue, sdpItem.UniqueAttributeValue())
		}

		if sdpItem.GetScope() != subscriptionID+"."+resourceGroup {
			t.Errorf("Expected scope %s, got %s", subscriptionID+"."+resourceGroup, sdpItem.GetScope())
		}

		if sdpItem.GetHealth() != sdp.Health_HEALTH_OK {
			t.Errorf("Expected health OK, got %v", sdpItem.GetHealth())
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
					ExpectedType:   stdlib.NetworkDNS.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  "test-replica.postgres.database.azure.com",
					ExpectedScope:  "global",
				},
			}
			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("GetWithInsufficientQueryParts", func(t *testing.T) {
		mockClient := mocks.NewMockDBforPostgreSQLFlexibleServerReplicaClient(ctrl)
		wrapper := manual.NewDBforPostgreSQLFlexibleServerReplica(&testDBforPostgreSQLFlexibleServerReplicaClient{MockDBforPostgreSQLFlexibleServerReplicaClient: mockClient}, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], serverName, true)
		if qErr == nil {
			t.Error("Expected error when providing only serverName (1 query part), but got nil")
		}
	})

	t.Run("GetWithEmptyServerName", func(t *testing.T) {
		mockClient := mocks.NewMockDBforPostgreSQLFlexibleServerReplicaClient(ctrl)
		wrapper := manual.NewDBforPostgreSQLFlexibleServerReplica(&testDBforPostgreSQLFlexibleServerReplicaClient{MockDBforPostgreSQLFlexibleServerReplicaClient: mockClient}, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey("", replicaName)
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr == nil {
			t.Error("Expected error when serverName is empty, but got nil")
		}
	})

	t.Run("GetWithEmptyReplicaName", func(t *testing.T) {
		mockClient := mocks.NewMockDBforPostgreSQLFlexibleServerReplicaClient(ctrl)
		wrapper := manual.NewDBforPostgreSQLFlexibleServerReplica(&testDBforPostgreSQLFlexibleServerReplicaClient{MockDBforPostgreSQLFlexibleServerReplicaClient: mockClient}, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(serverName, "")
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr == nil {
			t.Error("Expected error when replicaName is empty, but got nil")
		}
	})

	t.Run("Search", func(t *testing.T) {
		replica1 := createAzurePostgreSQLFlexibleServerReplica(serverName, "replica1")
		replica2 := createAzurePostgreSQLFlexibleServerReplica(serverName, "replica2")

		mockClient := mocks.NewMockDBforPostgreSQLFlexibleServerReplicaClient(ctrl)
		pager := &mockDBforPostgreSQLFlexibleServerReplicaPager{
			pages: []armpostgresqlflexibleservers.ReplicasClientListByServerResponse{
				{
					ServerList: armpostgresqlflexibleservers.ServerList{
						Value: []*armpostgresqlflexibleservers.Server{replica1, replica2},
					},
				},
			},
		}

		testClient := &testDBforPostgreSQLFlexibleServerReplicaClient{
			MockDBforPostgreSQLFlexibleServerReplicaClient: mockClient,
			pager: pager,
		}
		wrapper := manual.NewDBforPostgreSQLFlexibleServerReplica(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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
		replica1 := createAzurePostgreSQLFlexibleServerReplica(serverName, "replica1")

		mockClient := mocks.NewMockDBforPostgreSQLFlexibleServerReplicaClient(ctrl)
		pager := &mockDBforPostgreSQLFlexibleServerReplicaPager{
			pages: []armpostgresqlflexibleservers.ReplicasClientListByServerResponse{
				{
					ServerList: armpostgresqlflexibleservers.ServerList{
						Value: []*armpostgresqlflexibleservers.Server{replica1},
					},
				},
			},
		}

		testClient := &testDBforPostgreSQLFlexibleServerReplicaClient{
			MockDBforPostgreSQLFlexibleServerReplicaClient: mockClient,
			pager: pager,
		}
		wrapper := manual.NewDBforPostgreSQLFlexibleServerReplica(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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
		mockClient := mocks.NewMockDBforPostgreSQLFlexibleServerReplicaClient(ctrl)
		wrapper := manual.NewDBforPostgreSQLFlexibleServerReplica(&testDBforPostgreSQLFlexibleServerReplicaClient{MockDBforPostgreSQLFlexibleServerReplicaClient: mockClient}, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		_, qErr := wrapper.Search(ctx, wrapper.Scopes()[0])
		if qErr == nil {
			t.Error("Expected error when providing no query parts, but got nil")
		}
	})

	t.Run("SearchWithEmptyServerName", func(t *testing.T) {
		mockClient := mocks.NewMockDBforPostgreSQLFlexibleServerReplicaClient(ctrl)
		wrapper := manual.NewDBforPostgreSQLFlexibleServerReplica(&testDBforPostgreSQLFlexibleServerReplicaClient{MockDBforPostgreSQLFlexibleServerReplicaClient: mockClient}, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		_, qErr := wrapper.Search(ctx, wrapper.Scopes()[0], "")
		if qErr == nil {
			t.Error("Expected error when serverName is empty, but got nil")
		}
	})

	t.Run("SearchStreamWithEmptyServerName", func(t *testing.T) {
		mockClient := mocks.NewMockDBforPostgreSQLFlexibleServerReplicaClient(ctrl)
		wrapper := manual.NewDBforPostgreSQLFlexibleServerReplica(&testDBforPostgreSQLFlexibleServerReplicaClient{MockDBforPostgreSQLFlexibleServerReplicaClient: mockClient}, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		searchStreamable := wrapper.(sources.SearchStreamableWrapper)

		stream := discovery.NewRecordingQueryResultStream()
		searchStreamable.SearchStream(ctx, stream, sdpcache.NewNoOpCache(), sdpcache.CacheKey{}, wrapper.Scopes()[0], "")
		errs := stream.GetErrors()
		if len(errs) == 0 {
			t.Error("Expected error when serverName is empty, but got none")
		}
	})

	t.Run("ErrorHandling_Get", func(t *testing.T) {
		expectedErr := errors.New("replica not found")

		mockClient := mocks.NewMockDBforPostgreSQLFlexibleServerReplicaClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, "nonexistent-replica").Return(
			armpostgresqlflexibleservers.ServersClientGetResponse{}, expectedErr)

		wrapper := manual.NewDBforPostgreSQLFlexibleServerReplica(&testDBforPostgreSQLFlexibleServerReplicaClient{MockDBforPostgreSQLFlexibleServerReplicaClient: mockClient}, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(serverName, "nonexistent-replica")
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr == nil {
			t.Error("Expected error when getting non-existent replica, but got nil")
		}
	})

	t.Run("ErrorHandling_Search", func(t *testing.T) {
		mockClient := mocks.NewMockDBforPostgreSQLFlexibleServerReplicaClient(ctrl)
		errorPager := &errorDBforPostgreSQLFlexibleServerReplicaPager{}
		testClient := &testDBforPostgreSQLFlexibleServerReplicaClient{
			MockDBforPostgreSQLFlexibleServerReplicaClient: mockClient,
			pager: errorPager,
		}

		wrapper := manual.NewDBforPostgreSQLFlexibleServerReplica(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		_, qErr := wrapper.Search(ctx, wrapper.Scopes()[0], serverName)
		if qErr == nil {
			t.Error("Expected error from Search when pager returns error, but got nil")
		}
	})

	t.Run("PotentialLinks", func(t *testing.T) {
		mockClient := mocks.NewMockDBforPostgreSQLFlexibleServerReplicaClient(ctrl)
		wrapper := manual.NewDBforPostgreSQLFlexibleServerReplica(&testDBforPostgreSQLFlexibleServerReplicaClient{MockDBforPostgreSQLFlexibleServerReplicaClient: mockClient}, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		potentialLinks := wrapper.PotentialLinks()

		expectedLinks := []shared.ItemType{
			azureshared.DBforPostgreSQLFlexibleServer,
			azureshared.NetworkSubnet,
			azureshared.NetworkVirtualNetwork,
			azureshared.NetworkPrivateDNSZone,
			azureshared.NetworkPrivateEndpoint,
			azureshared.ManagedIdentityUserAssignedIdentity,
			azureshared.KeyVaultVault,
			azureshared.KeyVaultKey,
			stdlib.NetworkDNS,
		}

		for _, expected := range expectedLinks {
			if !potentialLinks[expected] {
				t.Errorf("Expected PotentialLinks to include %s", expected)
			}
		}
	})

	t.Run("HealthMapping", func(t *testing.T) {
		testCases := []struct {
			state          armpostgresqlflexibleservers.ServerState
			expectedHealth sdp.Health
		}{
			{armpostgresqlflexibleservers.ServerStateReady, sdp.Health_HEALTH_OK},
			{armpostgresqlflexibleservers.ServerStateStarting, sdp.Health_HEALTH_PENDING},
			{armpostgresqlflexibleservers.ServerStateStopping, sdp.Health_HEALTH_PENDING},
			{armpostgresqlflexibleservers.ServerStateUpdating, sdp.Health_HEALTH_PENDING},
			{armpostgresqlflexibleservers.ServerStateDisabled, sdp.Health_HEALTH_WARNING},
			{armpostgresqlflexibleservers.ServerStateStopped, sdp.Health_HEALTH_WARNING},
			{armpostgresqlflexibleservers.ServerStateDropping, sdp.Health_HEALTH_ERROR},
		}

		for _, tc := range testCases {
			t.Run(string(tc.state), func(t *testing.T) {
				replica := createAzurePostgreSQLFlexibleServerReplicaWithState(serverName, replicaName, tc.state)

				mockClient := mocks.NewMockDBforPostgreSQLFlexibleServerReplicaClient(ctrl)
				mockClient.EXPECT().Get(ctx, resourceGroup, replicaName).Return(
					armpostgresqlflexibleservers.ServersClientGetResponse{
						Server: *replica,
					}, nil)

				wrapper := manual.NewDBforPostgreSQLFlexibleServerReplica(&testDBforPostgreSQLFlexibleServerReplicaClient{MockDBforPostgreSQLFlexibleServerReplicaClient: mockClient}, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
				adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

				query := shared.CompositeLookupKey(serverName, replicaName)
				sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
				if qErr != nil {
					t.Fatalf("Expected no error, got: %v", qErr)
				}

				if sdpItem.GetHealth() != tc.expectedHealth {
					t.Errorf("Expected health %v for state %s, got %v", tc.expectedHealth, tc.state, sdpItem.GetHealth())
				}
			})
		}
	})
}

func createAzurePostgreSQLFlexibleServerReplica(serverName, replicaName string) *armpostgresqlflexibleservers.Server {
	return createAzurePostgreSQLFlexibleServerReplicaWithState(serverName, replicaName, armpostgresqlflexibleservers.ServerStateReady)
}

func createAzurePostgreSQLFlexibleServerReplicaWithState(serverName, replicaName string, state armpostgresqlflexibleservers.ServerState) *armpostgresqlflexibleservers.Server {
	replicaID := "/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.DBforPostgreSQL/flexibleServers/" + replicaName
	sourceServerID := "/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.DBforPostgreSQL/flexibleServers/" + serverName
	replicationRole := armpostgresqlflexibleservers.ReplicationRoleAsyncReplica
	fqdn := replicaName + ".postgres.database.azure.com"
	return &armpostgresqlflexibleservers.Server{
		Name:     &replicaName,
		ID:       &replicaID,
		Type:     new(string),
		Location: new(string),
		Properties: &armpostgresqlflexibleservers.ServerProperties{
			State:                    &state,
			ReplicationRole:          &replicationRole,
			SourceServerResourceID:   &sourceServerID,
			FullyQualifiedDomainName: &fqdn,
		},
	}
}
