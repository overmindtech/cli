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
)

type mockDBforPostgreSQLFlexibleServerBackupPager struct {
	pages []armpostgresqlflexibleservers.BackupsAutomaticAndOnDemandClientListByServerResponse
	index int
}

func (m *mockDBforPostgreSQLFlexibleServerBackupPager) More() bool {
	return m.index < len(m.pages)
}

func (m *mockDBforPostgreSQLFlexibleServerBackupPager) NextPage(ctx context.Context) (armpostgresqlflexibleservers.BackupsAutomaticAndOnDemandClientListByServerResponse, error) {
	if m.index >= len(m.pages) {
		return armpostgresqlflexibleservers.BackupsAutomaticAndOnDemandClientListByServerResponse{}, errors.New("no more pages")
	}
	page := m.pages[m.index]
	m.index++
	return page, nil
}

type errorDBforPostgreSQLFlexibleServerBackupPager struct{}

func (e *errorDBforPostgreSQLFlexibleServerBackupPager) More() bool {
	return true
}

func (e *errorDBforPostgreSQLFlexibleServerBackupPager) NextPage(ctx context.Context) (armpostgresqlflexibleservers.BackupsAutomaticAndOnDemandClientListByServerResponse, error) {
	return armpostgresqlflexibleservers.BackupsAutomaticAndOnDemandClientListByServerResponse{}, errors.New("pager error")
}

type testDBforPostgreSQLFlexibleServerBackupClient struct {
	*mocks.MockDBforPostgreSQLFlexibleServerBackupClient
	pager clients.DBforPostgreSQLFlexibleServerBackupPager
}

func (t *testDBforPostgreSQLFlexibleServerBackupClient) ListByServer(ctx context.Context, resourceGroupName, serverName string) clients.DBforPostgreSQLFlexibleServerBackupPager {
	return t.pager
}

func TestDBforPostgreSQLFlexibleServerBackup(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"
	serverName := "test-server"
	backupName := "test-backup"

	t.Run("Get", func(t *testing.T) {
		backup := createAzurePostgreSQLFlexibleServerBackup(serverName, backupName)

		mockClient := mocks.NewMockDBforPostgreSQLFlexibleServerBackupClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, serverName, backupName).Return(
			armpostgresqlflexibleservers.BackupsAutomaticAndOnDemandClientGetResponse{
				BackupAutomaticAndOnDemand: *backup,
			}, nil)

		wrapper := manual.NewDBforPostgreSQLFlexibleServerBackup(&testDBforPostgreSQLFlexibleServerBackupClient{MockDBforPostgreSQLFlexibleServerBackupClient: mockClient}, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(serverName, backupName)
		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.DBforPostgreSQLFlexibleServerBackup.String() {
			t.Errorf("Expected type %s, got %s", azureshared.DBforPostgreSQLFlexibleServerBackup, sdpItem.GetType())
		}

		if sdpItem.GetUniqueAttribute() != "uniqueAttr" {
			t.Errorf("Expected unique attribute 'uniqueAttr', got %s", sdpItem.GetUniqueAttribute())
		}

		expectedUniqueAttrValue := shared.CompositeLookupKey(serverName, backupName)
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
			}
			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("GetWithInsufficientQueryParts", func(t *testing.T) {
		mockClient := mocks.NewMockDBforPostgreSQLFlexibleServerBackupClient(ctrl)
		wrapper := manual.NewDBforPostgreSQLFlexibleServerBackup(&testDBforPostgreSQLFlexibleServerBackupClient{MockDBforPostgreSQLFlexibleServerBackupClient: mockClient}, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], serverName, true)
		if qErr == nil {
			t.Error("Expected error when providing only serverName (1 query part), but got nil")
		}
	})

	t.Run("GetWithEmptyServerName", func(t *testing.T) {
		mockClient := mocks.NewMockDBforPostgreSQLFlexibleServerBackupClient(ctrl)
		wrapper := manual.NewDBforPostgreSQLFlexibleServerBackup(&testDBforPostgreSQLFlexibleServerBackupClient{MockDBforPostgreSQLFlexibleServerBackupClient: mockClient}, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey("", backupName)
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr == nil {
			t.Error("Expected error when serverName is empty, but got nil")
		}
	})

	t.Run("GetWithEmptyBackupName", func(t *testing.T) {
		mockClient := mocks.NewMockDBforPostgreSQLFlexibleServerBackupClient(ctrl)
		wrapper := manual.NewDBforPostgreSQLFlexibleServerBackup(&testDBforPostgreSQLFlexibleServerBackupClient{MockDBforPostgreSQLFlexibleServerBackupClient: mockClient}, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(serverName, "")
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr == nil {
			t.Error("Expected error when backupName is empty, but got nil")
		}
	})

	t.Run("Search", func(t *testing.T) {
		backup1 := createAzurePostgreSQLFlexibleServerBackup(serverName, "backup1")
		backup2 := createAzurePostgreSQLFlexibleServerBackup(serverName, "backup2")

		mockClient := mocks.NewMockDBforPostgreSQLFlexibleServerBackupClient(ctrl)
		pager := &mockDBforPostgreSQLFlexibleServerBackupPager{
			pages: []armpostgresqlflexibleservers.BackupsAutomaticAndOnDemandClientListByServerResponse{
				{
					BackupAutomaticAndOnDemandList: armpostgresqlflexibleservers.BackupAutomaticAndOnDemandList{
						Value: []*armpostgresqlflexibleservers.BackupAutomaticAndOnDemand{backup1, backup2},
					},
				},
			},
		}

		testClient := &testDBforPostgreSQLFlexibleServerBackupClient{
			MockDBforPostgreSQLFlexibleServerBackupClient: mockClient,
			pager: pager,
		}
		wrapper := manual.NewDBforPostgreSQLFlexibleServerBackup(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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
		backup1 := createAzurePostgreSQLFlexibleServerBackup(serverName, "backup1")

		mockClient := mocks.NewMockDBforPostgreSQLFlexibleServerBackupClient(ctrl)
		pager := &mockDBforPostgreSQLFlexibleServerBackupPager{
			pages: []armpostgresqlflexibleservers.BackupsAutomaticAndOnDemandClientListByServerResponse{
				{
					BackupAutomaticAndOnDemandList: armpostgresqlflexibleservers.BackupAutomaticAndOnDemandList{
						Value: []*armpostgresqlflexibleservers.BackupAutomaticAndOnDemand{backup1},
					},
				},
			},
		}

		testClient := &testDBforPostgreSQLFlexibleServerBackupClient{
			MockDBforPostgreSQLFlexibleServerBackupClient: mockClient,
			pager: pager,
		}
		wrapper := manual.NewDBforPostgreSQLFlexibleServerBackup(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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
		mockClient := mocks.NewMockDBforPostgreSQLFlexibleServerBackupClient(ctrl)
		wrapper := manual.NewDBforPostgreSQLFlexibleServerBackup(&testDBforPostgreSQLFlexibleServerBackupClient{MockDBforPostgreSQLFlexibleServerBackupClient: mockClient}, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		_, qErr := wrapper.Search(ctx, wrapper.Scopes()[0])
		if qErr == nil {
			t.Error("Expected error when providing no query parts, but got nil")
		}
	})

	t.Run("ErrorHandling_Get", func(t *testing.T) {
		expectedErr := errors.New("backup not found")

		mockClient := mocks.NewMockDBforPostgreSQLFlexibleServerBackupClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, serverName, "nonexistent-backup").Return(
			armpostgresqlflexibleservers.BackupsAutomaticAndOnDemandClientGetResponse{}, expectedErr)

		wrapper := manual.NewDBforPostgreSQLFlexibleServerBackup(&testDBforPostgreSQLFlexibleServerBackupClient{MockDBforPostgreSQLFlexibleServerBackupClient: mockClient}, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(serverName, "nonexistent-backup")
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr == nil {
			t.Error("Expected error when getting non-existent backup, but got nil")
		}
	})

	t.Run("ErrorHandling_Search", func(t *testing.T) {
		mockClient := mocks.NewMockDBforPostgreSQLFlexibleServerBackupClient(ctrl)
		errorPager := &errorDBforPostgreSQLFlexibleServerBackupPager{}
		testClient := &testDBforPostgreSQLFlexibleServerBackupClient{
			MockDBforPostgreSQLFlexibleServerBackupClient: mockClient,
			pager: errorPager,
		}

		wrapper := manual.NewDBforPostgreSQLFlexibleServerBackup(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		_, qErr := wrapper.Search(ctx, wrapper.Scopes()[0], serverName)
		if qErr == nil {
			t.Error("Expected error from Search when pager returns error, but got nil")
		}
	})

	t.Run("PotentialLinks", func(t *testing.T) {
		mockClient := mocks.NewMockDBforPostgreSQLFlexibleServerBackupClient(ctrl)
		wrapper := manual.NewDBforPostgreSQLFlexibleServerBackup(&testDBforPostgreSQLFlexibleServerBackupClient{MockDBforPostgreSQLFlexibleServerBackupClient: mockClient}, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		potentialLinks := wrapper.PotentialLinks()

		if !potentialLinks[azureshared.DBforPostgreSQLFlexibleServer] {
			t.Error("Expected PotentialLinks to include DBforPostgreSQLFlexibleServer")
		}
	})
}

func createAzurePostgreSQLFlexibleServerBackup(serverName, backupName string) *armpostgresqlflexibleservers.BackupAutomaticAndOnDemand {
	backupID := "/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.DBforPostgreSQL/flexibleServers/" + serverName + "/backups/" + backupName
	backupType := armpostgresqlflexibleservers.BackupTypeFull
	return &armpostgresqlflexibleservers.BackupAutomaticAndOnDemand{
		Name: new(backupName),
		ID:   new(backupID),
		Type: new("Microsoft.DBforPostgreSQL/flexibleServers/backups"),
		Properties: &armpostgresqlflexibleservers.BackupAutomaticAndOnDemandProperties{
			BackupType: &backupType,
			Source:     new("Automatic"),
		},
	}
}
