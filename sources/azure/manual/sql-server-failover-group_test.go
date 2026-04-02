package manual_test

import (
	"context"
	"errors"
	"slices"
	"sync"
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

// mockSqlFailoverGroupsPager is a simple mock implementation of SqlFailoverGroupsPager
type mockSqlFailoverGroupsPager struct {
	pages []armsql.FailoverGroupsClientListByServerResponse
	index int
}

func (m *mockSqlFailoverGroupsPager) More() bool {
	return m.index < len(m.pages)
}

func (m *mockSqlFailoverGroupsPager) NextPage(ctx context.Context) (armsql.FailoverGroupsClientListByServerResponse, error) {
	if m.index >= len(m.pages) {
		return armsql.FailoverGroupsClientListByServerResponse{}, errors.New("no more pages")
	}
	page := m.pages[m.index]
	m.index++
	return page, nil
}

// errorSqlFailoverGroupsPager is a mock pager that always returns an error
type errorSqlFailoverGroupsPager struct{}

func (e *errorSqlFailoverGroupsPager) More() bool {
	return true
}

func (e *errorSqlFailoverGroupsPager) NextPage(ctx context.Context) (armsql.FailoverGroupsClientListByServerResponse, error) {
	return armsql.FailoverGroupsClientListByServerResponse{}, errors.New("pager error")
}

// testSqlFailoverGroupsClient wraps the mock to implement the correct interface
type testSqlFailoverGroupsClient struct {
	*mocks.MockSqlFailoverGroupsClient
	pager clients.SqlFailoverGroupsPager
}

func (t *testSqlFailoverGroupsClient) ListByServer(ctx context.Context, resourceGroupName, serverName string) clients.SqlFailoverGroupsPager {
	return t.pager
}

func TestSqlServerFailoverGroup(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"
	serverName := "test-server"
	failoverGroupName := "test-failover-group"

	t.Run("Get", func(t *testing.T) {
		failoverGroup := createAzureSqlServerFailoverGroup(subscriptionID, resourceGroup, serverName, failoverGroupName)

		mockClient := mocks.NewMockSqlFailoverGroupsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, serverName, failoverGroupName).Return(
			armsql.FailoverGroupsClientGetResponse{
				FailoverGroup: *failoverGroup,
			}, nil)

		testClient := &testSqlFailoverGroupsClient{MockSqlFailoverGroupsClient: mockClient}
		wrapper := manual.NewSqlServerFailoverGroup(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(serverName, failoverGroupName)
		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.SQLServerFailoverGroup.String() {
			t.Errorf("Expected type %s, got %s", azureshared.SQLServerFailoverGroup, sdpItem.GetType())
		}

		if sdpItem.GetUniqueAttribute() != "uniqueAttr" {
			t.Errorf("Expected unique attribute 'uniqueAttr', got %s", sdpItem.GetUniqueAttribute())
		}

		expectedUniqueAttrValue := shared.CompositeLookupKey(serverName, failoverGroupName)
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
					// SQLServer link (parent)
					ExpectedType:   azureshared.SQLServer.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  serverName,
					ExpectedScope:  subscriptionID + "." + resourceGroup,
				},
				{
					// Partner server link
					ExpectedType:   azureshared.SQLServer.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "partner-server",
					ExpectedScope:  subscriptionID + ".partner-rg",
				},
				{
					// Database link
					ExpectedType:   azureshared.SQLDatabase.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey(serverName, "test-database"),
					ExpectedScope:  subscriptionID + "." + resourceGroup,
				},
			}

			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("Get_WithInsufficientQueryParts", func(t *testing.T) {
		mockClient := mocks.NewMockSqlFailoverGroupsClient(ctrl)
		testClient := &testSqlFailoverGroupsClient{MockSqlFailoverGroupsClient: mockClient}

		wrapper := manual.NewSqlServerFailoverGroup(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		// Only provide serverName without failoverGroupName
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], serverName, true)
		if qErr == nil {
			t.Error("Expected error when providing insufficient query parts, but got nil")
		}
	})

	t.Run("GetWithEmptyServerName", func(t *testing.T) {
		mockClient := mocks.NewMockSqlFailoverGroupsClient(ctrl)
		testClient := &testSqlFailoverGroupsClient{MockSqlFailoverGroupsClient: mockClient}

		wrapper := manual.NewSqlServerFailoverGroup(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		// Provide empty server name and valid failover group name
		// Call wrapper.Get directly to get *sdp.QueryError
		_, qErr := wrapper.Get(ctx, wrapper.Scopes()[0], "", failoverGroupName)
		if qErr == nil {
			t.Fatal("Expected error when serverName is empty, but got nil")
		}
		if qErr.GetErrorString() != "serverName cannot be empty" {
			t.Errorf("Expected error string 'serverName cannot be empty', got: %s", qErr.GetErrorString())
		}
	})

	t.Run("GetWithEmptyFailoverGroupName", func(t *testing.T) {
		mockClient := mocks.NewMockSqlFailoverGroupsClient(ctrl)
		testClient := &testSqlFailoverGroupsClient{MockSqlFailoverGroupsClient: mockClient}

		wrapper := manual.NewSqlServerFailoverGroup(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		// Provide valid server name and empty failover group name
		// Call wrapper.Get directly to get *sdp.QueryError
		_, qErr := wrapper.Get(ctx, wrapper.Scopes()[0], serverName, "")
		if qErr == nil {
			t.Fatal("Expected error when failoverGroupName is empty, but got nil")
		}
		if qErr.GetErrorString() != "failoverGroupName cannot be empty" {
			t.Errorf("Expected error string 'failoverGroupName cannot be empty', got: %s", qErr.GetErrorString())
		}
	})

	t.Run("Search", func(t *testing.T) {
		failoverGroup1 := createAzureSqlServerFailoverGroup(subscriptionID, resourceGroup, serverName, "failover-group-1")
		failoverGroup2 := createAzureSqlServerFailoverGroup(subscriptionID, resourceGroup, serverName, "failover-group-2")

		mockClient := mocks.NewMockSqlFailoverGroupsClient(ctrl)
		mockPager := &mockSqlFailoverGroupsPager{
			pages: []armsql.FailoverGroupsClientListByServerResponse{
				{
					FailoverGroupListResult: armsql.FailoverGroupListResult{
						Value: []*armsql.FailoverGroup{failoverGroup1, failoverGroup2},
					},
				},
			},
		}

		testClient := &testSqlFailoverGroupsClient{
			MockSqlFailoverGroupsClient: mockClient,
			pager:                       mockPager,
		}

		wrapper := manual.NewSqlServerFailoverGroup(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support Search operation")
		}

		sdpItems, err := searchable.Search(ctx, wrapper.Scopes()[0], serverName, true)
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

			if item.GetType() != azureshared.SQLServerFailoverGroup.String() {
				t.Errorf("Expected type %s, got %s", azureshared.SQLServerFailoverGroup, item.GetType())
			}
		}
	})

	t.Run("SearchStream", func(t *testing.T) {
		failoverGroup1 := createAzureSqlServerFailoverGroup(subscriptionID, resourceGroup, serverName, "failover-group-1")
		failoverGroup2 := createAzureSqlServerFailoverGroup(subscriptionID, resourceGroup, serverName, "failover-group-2")

		mockClient := mocks.NewMockSqlFailoverGroupsClient(ctrl)
		mockPager := &mockSqlFailoverGroupsPager{
			pages: []armsql.FailoverGroupsClientListByServerResponse{
				{
					FailoverGroupListResult: armsql.FailoverGroupListResult{
						Value: []*armsql.FailoverGroup{failoverGroup1, failoverGroup2},
					},
				},
			},
		}

		testClient := &testSqlFailoverGroupsClient{
			MockSqlFailoverGroupsClient: mockClient,
			pager:                       mockPager,
		}

		wrapper := manual.NewSqlServerFailoverGroup(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		wg := &sync.WaitGroup{}
		wg.Add(2)

		var items []*sdp.Item
		mockItemHandler := func(item *sdp.Item) {
			items = append(items, item)
			wg.Done()
		}

		var errs []error
		mockErrorHandler := func(err error) {
			errs = append(errs, err)
		}

		stream := discovery.NewQueryResultStream(mockItemHandler, mockErrorHandler)

		searchStreamable, ok := adapter.(discovery.SearchStreamableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support SearchStream operation")
		}

		searchStreamable.SearchStream(ctx, wrapper.Scopes()[0], serverName, true, stream)
		wg.Wait()

		if len(errs) != 0 {
			t.Fatalf("Expected no errors, got: %v", errs)
		}

		if len(items) != 2 {
			t.Fatalf("Expected 2 items, got: %d", len(items))
		}
	})

	t.Run("SearchWithEmptyServerName", func(t *testing.T) {
		mockClient := mocks.NewMockSqlFailoverGroupsClient(ctrl)
		testClient := &testSqlFailoverGroupsClient{MockSqlFailoverGroupsClient: mockClient}

		wrapper := manual.NewSqlServerFailoverGroup(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		// Test Search directly with empty server name
		_, qErr := wrapper.Search(ctx, wrapper.Scopes()[0], "")
		if qErr == nil {
			t.Error("Expected error when serverName is empty, but got nil")
		}
	})

	t.Run("Search_InvalidQueryParts", func(t *testing.T) {
		mockClient := mocks.NewMockSqlFailoverGroupsClient(ctrl)
		testClient := &testSqlFailoverGroupsClient{MockSqlFailoverGroupsClient: mockClient}

		wrapper := manual.NewSqlServerFailoverGroup(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		// Test Search directly with no query parts
		_, qErr := wrapper.Search(ctx, wrapper.Scopes()[0])
		if qErr == nil {
			t.Error("Expected error when providing no query parts, but got nil")
		}
	})

	t.Run("Search_WithNilName", func(t *testing.T) {
		failoverGroup1 := createAzureSqlServerFailoverGroup(subscriptionID, resourceGroup, serverName, "failover-group-1")
		failoverGroup2 := &armsql.FailoverGroup{
			Name: nil, // FailoverGroup with nil name should be skipped
			ID:   new("/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.Sql/servers/test-server/failoverGroups/failover-group-2"),
			Tags: map[string]*string{
				"env": new("test"),
			},
		}

		mockClient := mocks.NewMockSqlFailoverGroupsClient(ctrl)
		mockPager := &mockSqlFailoverGroupsPager{
			pages: []armsql.FailoverGroupsClientListByServerResponse{
				{
					FailoverGroupListResult: armsql.FailoverGroupListResult{
						Value: []*armsql.FailoverGroup{failoverGroup1, failoverGroup2},
					},
				},
			},
		}

		testClient := &testSqlFailoverGroupsClient{
			MockSqlFailoverGroupsClient: mockClient,
			pager:                       mockPager,
		}

		wrapper := manual.NewSqlServerFailoverGroup(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support Search operation")
		}

		sdpItems, err := searchable.Search(ctx, wrapper.Scopes()[0], serverName, true)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		// Should only return 1 item (failover group with nil name is skipped)
		if len(sdpItems) != 1 {
			t.Fatalf("Expected 1 item (nil name filtered out), got: %d", len(sdpItems))
		}
	})

	t.Run("ErrorHandling_Get", func(t *testing.T) {
		expectedErr := errors.New("failover group not found")

		mockClient := mocks.NewMockSqlFailoverGroupsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, serverName, "nonexistent-failover-group").Return(
			armsql.FailoverGroupsClientGetResponse{}, expectedErr)

		testClient := &testSqlFailoverGroupsClient{MockSqlFailoverGroupsClient: mockClient}
		wrapper := manual.NewSqlServerFailoverGroup(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(serverName, "nonexistent-failover-group")
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr == nil {
			t.Error("Expected error when getting non-existent failover group, but got nil")
		}
	})

	t.Run("ErrorHandling_Search", func(t *testing.T) {
		mockClient := mocks.NewMockSqlFailoverGroupsClient(ctrl)
		errorPager := &errorSqlFailoverGroupsPager{}

		testClient := &testSqlFailoverGroupsClient{
			MockSqlFailoverGroupsClient: mockClient,
			pager:                       errorPager,
		}

		wrapper := manual.NewSqlServerFailoverGroup(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support Search operation")
		}

		_, err := searchable.Search(ctx, wrapper.Scopes()[0], serverName, true)
		if err == nil {
			t.Error("Expected error from pager when NextPage returns an error, but got nil")
		}
	})

	t.Run("InterfaceCompliance", func(t *testing.T) {
		mockClient := mocks.NewMockSqlFailoverGroupsClient(ctrl)
		testClient := &testSqlFailoverGroupsClient{MockSqlFailoverGroupsClient: mockClient}
		wrapper := manual.NewSqlServerFailoverGroup(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		w := wrapper.(sources.Wrapper)

		// Verify IAMPermissions
		permissions := w.IAMPermissions()
		if len(permissions) == 0 {
			t.Error("Expected IAMPermissions to return at least one permission")
		}
		expectedPermission := "Microsoft.Sql/servers/failoverGroups/read"
		found := slices.Contains(permissions, expectedPermission)
		if !found {
			t.Errorf("Expected IAMPermissions to include %s", expectedPermission)
		}

		// Verify PotentialLinks
		potentialLinks := w.PotentialLinks()
		if len(potentialLinks) == 0 {
			t.Error("Expected PotentialLinks to return at least one link")
		}
		if !potentialLinks[azureshared.SQLServer] {
			t.Error("Expected PotentialLinks to include SQLServer")
		}
		if !potentialLinks[azureshared.SQLDatabase] {
			t.Error("Expected PotentialLinks to include SQLDatabase")
		}
	})
}

// createAzureSqlServerFailoverGroup creates a mock Azure SQL Server Failover Group for testing
func createAzureSqlServerFailoverGroup(subscriptionID, resourceGroup, serverName, failoverGroupName string) *armsql.FailoverGroup {
	failoverGroupID := "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Sql/servers/" + serverName + "/failoverGroups/" + failoverGroupName
	partnerServerID := "/subscriptions/" + subscriptionID + "/resourceGroups/partner-rg/providers/Microsoft.Sql/servers/partner-server"
	databaseID := "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Sql/servers/" + serverName + "/databases/test-database"

	replicationState := ""

	return &armsql.FailoverGroup{
		Name:     new(failoverGroupName),
		Location: new("eastus"),
		Tags: map[string]*string{
			"env": new("test"),
		},
		ID: new(failoverGroupID),
		Properties: &armsql.FailoverGroupProperties{
			ReplicationState: &replicationState,
			PartnerServers: []*armsql.PartnerInfo{
				{
					ID:       new(partnerServerID),
					Location: new("westus"),
				},
			},
			Databases: []*string{
				new(databaseID),
			},
			ReadWriteEndpoint: &armsql.FailoverGroupReadWriteEndpoint{
				FailoverPolicy: new(armsql.ReadWriteEndpointFailoverPolicyAutomatic),
			},
		},
	}
}
