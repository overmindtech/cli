package manual_test

import (
	"context"
	"errors"
	"slices"
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

type mockSqlElasticPoolPager struct {
	pages []armsql.ElasticPoolsClientListByServerResponse
	index int
}

func (m *mockSqlElasticPoolPager) More() bool {
	return m.index < len(m.pages)
}

func (m *mockSqlElasticPoolPager) NextPage(ctx context.Context) (armsql.ElasticPoolsClientListByServerResponse, error) {
	if m.index >= len(m.pages) {
		return armsql.ElasticPoolsClientListByServerResponse{}, errors.New("no more pages")
	}
	page := m.pages[m.index]
	m.index++
	return page, nil
}

type errorSqlElasticPoolPager struct{}

func (e *errorSqlElasticPoolPager) More() bool {
	return true
}

func (e *errorSqlElasticPoolPager) NextPage(ctx context.Context) (armsql.ElasticPoolsClientListByServerResponse, error) {
	return armsql.ElasticPoolsClientListByServerResponse{}, errors.New("pager error")
}

type testSqlElasticPoolClient struct {
	*mocks.MockSqlElasticPoolClient
	pager clients.SqlElasticPoolPager
}

func (t *testSqlElasticPoolClient) ListByServer(ctx context.Context, resourceGroupName, serverName string) clients.SqlElasticPoolPager {
	return t.pager
}

func TestSqlElasticPool(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"
	serverName := "test-server"
	elasticPoolName := "test-pool"

	t.Run("Get", func(t *testing.T) {
		pool := createAzureSqlElasticPool(serverName, elasticPoolName)

		mockClient := mocks.NewMockSqlElasticPoolClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, serverName, elasticPoolName).Return(
			armsql.ElasticPoolsClientGetResponse{
				ElasticPool: *pool,
			}, nil)

		wrapper := manual.NewSqlElasticPool(&testSqlElasticPoolClient{MockSqlElasticPoolClient: mockClient}, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(serverName, elasticPoolName)
		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.SQLElasticPool.String() {
			t.Errorf("Expected type %s, got %s", azureshared.SQLElasticPool.String(), sdpItem.GetType())
		}

		if sdpItem.GetUniqueAttribute() != "uniqueAttr" {
			t.Errorf("Expected unique attribute 'uniqueAttr', got %s", sdpItem.GetUniqueAttribute())
		}

		expectedUniqueAttrValue := shared.CompositeLookupKey(serverName, elasticPoolName)
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
					ExpectedType:   azureshared.SQLDatabase.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  serverName,
					ExpectedScope:  subscriptionID + "." + resourceGroup,
				},
			}
			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("GetWithInsufficientQueryParts", func(t *testing.T) {
		mockClient := mocks.NewMockSqlElasticPoolClient(ctrl)
		wrapper := manual.NewSqlElasticPool(&testSqlElasticPoolClient{MockSqlElasticPoolClient: mockClient}, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], serverName, true)
		if qErr == nil {
			t.Error("Expected error when providing only serverName (1 query part), but got nil")
		}
	})

	t.Run("GetWithEmptyName", func(t *testing.T) {
		mockClient := mocks.NewMockSqlElasticPoolClient(ctrl)
		wrapper := manual.NewSqlElasticPool(&testSqlElasticPoolClient{MockSqlElasticPoolClient: mockClient}, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(serverName, "")
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr == nil {
			t.Error("Expected error when elastic pool name is empty, but got nil")
		}
	})

	t.Run("Search", func(t *testing.T) {
		pool1 := createAzureSqlElasticPool(serverName, "pool-1")
		pool2 := createAzureSqlElasticPool(serverName, "pool-2")

		mockClient := mocks.NewMockSqlElasticPoolClient(ctrl)
		pager := &mockSqlElasticPoolPager{
			pages: []armsql.ElasticPoolsClientListByServerResponse{
				{
					ElasticPoolListResult: armsql.ElasticPoolListResult{
						Value: []*armsql.ElasticPool{pool1, pool2},
					},
				},
			},
		}

		testClient := &testSqlElasticPoolClient{
			MockSqlElasticPoolClient: mockClient,
			pager:                    pager,
		}

		wrapper := manual.NewSqlElasticPool(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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
		pool := createAzureSqlElasticPool(serverName, elasticPoolName)

		mockClient := mocks.NewMockSqlElasticPoolClient(ctrl)
		pager := &mockSqlElasticPoolPager{
			pages: []armsql.ElasticPoolsClientListByServerResponse{
				{
					ElasticPoolListResult: armsql.ElasticPoolListResult{
						Value: []*armsql.ElasticPool{pool},
					},
				},
			},
		}

		testClient := &testSqlElasticPoolClient{
			MockSqlElasticPoolClient: mockClient,
			pager:                    pager,
		}
		wrapper := manual.NewSqlElasticPool(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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
		mockClient := mocks.NewMockSqlElasticPoolClient(ctrl)
		wrapper := manual.NewSqlElasticPool(&testSqlElasticPoolClient{MockSqlElasticPoolClient: mockClient}, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		_, qErr := wrapper.Search(ctx, wrapper.Scopes()[0])
		if qErr == nil {
			t.Error("Expected error when providing no query parts, but got nil")
		}
	})

	t.Run("ErrorHandling_Get", func(t *testing.T) {
		expectedErr := errors.New("elastic pool not found")

		mockClient := mocks.NewMockSqlElasticPoolClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, serverName, "nonexistent-pool").Return(
			armsql.ElasticPoolsClientGetResponse{}, expectedErr)

		wrapper := manual.NewSqlElasticPool(&testSqlElasticPoolClient{MockSqlElasticPoolClient: mockClient}, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(serverName, "nonexistent-pool")
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr == nil {
			t.Error("Expected error when getting non-existent elastic pool, but got nil")
		}
	})

	t.Run("ErrorHandling_Search", func(t *testing.T) {
		mockClient := mocks.NewMockSqlElasticPoolClient(ctrl)
		errorPager := &errorSqlElasticPoolPager{}
		testClient := &testSqlElasticPoolClient{
			MockSqlElasticPoolClient: mockClient,
			pager:                    errorPager,
		}

		wrapper := manual.NewSqlElasticPool(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		_, qErr := wrapper.Search(ctx, wrapper.Scopes()[0], serverName)
		if qErr == nil {
			t.Error("Expected error from Search when pager returns error, but got nil")
		}
	})

	t.Run("InterfaceCompliance", func(t *testing.T) {
		mockClient := mocks.NewMockSqlElasticPoolClient(ctrl)
		wrapper := manual.NewSqlElasticPool(&testSqlElasticPoolClient{MockSqlElasticPoolClient: mockClient}, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		w := wrapper.(sources.Wrapper)

		permissions := w.IAMPermissions()
		if len(permissions) == 0 {
			t.Error("Expected IAMPermissions to return at least one permission")
		}
		expectedPermission := "Microsoft.Sql/servers/elasticPools/read"
		if !slices.Contains(permissions, expectedPermission) {
			t.Errorf("Expected IAMPermissions to include %s", expectedPermission)
		}

		potentialLinks := w.PotentialLinks()
		if !potentialLinks[azureshared.SQLServer] {
			t.Error("Expected PotentialLinks to include SQLServer")
		}
		if !potentialLinks[azureshared.SQLDatabase] {
			t.Error("Expected PotentialLinks to include SQLDatabase")
		}
		if !potentialLinks[azureshared.MaintenanceMaintenanceConfiguration] {
			t.Error("Expected PotentialLinks to include MaintenanceMaintenanceConfiguration")
		}

		mappings := w.TerraformMappings()
		if len(mappings) == 0 {
			t.Error("Expected TerraformMappings to return at least one mapping")
		}
		foundMapping := false
		for _, mapping := range mappings {
			if mapping.GetTerraformQueryMap() == "azurerm_mssql_elasticpool.id" {
				foundMapping = true
				break
			}
		}
		if !foundMapping {
			t.Error("Expected TerraformMappings to include 'azurerm_mssql_elasticpool.id' mapping")
		}
	})
}

func createAzureSqlElasticPool(serverName, elasticPoolName string) *armsql.ElasticPool {
	poolID := "/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.Sql/servers/" + serverName + "/elasticPools/" + elasticPoolName
	state := armsql.ElasticPoolStateReady
	return &armsql.ElasticPool{
		Name: &elasticPoolName,
		ID:   &poolID,
		Properties: &armsql.ElasticPoolProperties{
			State: &state,
		},
	}
}
