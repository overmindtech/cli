package manual_test

import (
	"context"
	"errors"
	"sync"
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

// mockAdministratorPager is a simple mock implementation of DBforPostgreSQLFlexibleServerAdministratorPager
type mockAdministratorPager struct {
	pages []armpostgresqlflexibleservers.AdministratorsMicrosoftEntraClientListByServerResponse
	index int
}

func (m *mockAdministratorPager) More() bool {
	return m.index < len(m.pages)
}

func (m *mockAdministratorPager) NextPage(ctx context.Context) (armpostgresqlflexibleservers.AdministratorsMicrosoftEntraClientListByServerResponse, error) {
	if m.index >= len(m.pages) {
		return armpostgresqlflexibleservers.AdministratorsMicrosoftEntraClientListByServerResponse{}, errors.New("no more pages")
	}
	page := m.pages[m.index]
	m.index++
	return page, nil
}

// errorAdministratorPager is a mock pager that always returns an error
type errorAdministratorPager struct{}

func (e *errorAdministratorPager) More() bool {
	return true
}

func (e *errorAdministratorPager) NextPage(ctx context.Context) (armpostgresqlflexibleservers.AdministratorsMicrosoftEntraClientListByServerResponse, error) {
	return armpostgresqlflexibleservers.AdministratorsMicrosoftEntraClientListByServerResponse{}, errors.New("pager error")
}

// testAdministratorClient wraps the mock to implement the correct interface
type testAdministratorClient struct {
	*mocks.MockDBforPostgreSQLFlexibleServerAdministratorClient
	pager clients.DBforPostgreSQLFlexibleServerAdministratorPager
}

func (t *testAdministratorClient) ListByServer(ctx context.Context, resourceGroupName, serverName string) clients.DBforPostgreSQLFlexibleServerAdministratorPager {
	return t.pager
}

func TestDBforPostgreSQLFlexibleServerAdministrator(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"
	serverName := "test-server"
	objectID := "00000000-0000-0000-0000-000000000001"

	t.Run("Get", func(t *testing.T) {
		admin := createAzureAdministrator(objectID)

		mockClient := mocks.NewMockDBforPostgreSQLFlexibleServerAdministratorClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, serverName, objectID).Return(
			armpostgresqlflexibleservers.AdministratorsMicrosoftEntraClientGetResponse{
				AdministratorMicrosoftEntra: *admin,
			}, nil)

		testClient := &testAdministratorClient{MockDBforPostgreSQLFlexibleServerAdministratorClient: mockClient}
		wrapper := manual.NewDBforPostgreSQLFlexibleServerAdministrator(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(serverName, objectID)
		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.DBforPostgreSQLFlexibleServerAdministrator.String() {
			t.Errorf("Expected type %s, got %s", azureshared.DBforPostgreSQLFlexibleServerAdministrator, sdpItem.GetType())
		}

		if sdpItem.GetUniqueAttribute() != "uniqueAttr" {
			t.Errorf("Expected unique attribute 'uniqueAttr', got %s", sdpItem.GetUniqueAttribute())
		}

		expectedUniqueValue := shared.CompositeLookupKey(serverName, objectID)
		if sdpItem.UniqueAttributeValue() != expectedUniqueValue {
			t.Errorf("Expected unique attribute value %s, got %s", expectedUniqueValue, sdpItem.UniqueAttributeValue())
		}

		if sdpItem.GetScope() != subscriptionID+"."+resourceGroup {
			t.Errorf("Expected scope %s, got %s", subscriptionID+"."+resourceGroup, sdpItem.GetScope())
		}

		if err := sdpItem.Validate(); err != nil {
			t.Fatalf("Expected no validation error, got: %v", err)
		}

		t.Run("StaticTests", func(t *testing.T) {
			linkedQueries := sdpItem.GetLinkedItemQueries()
			if len(linkedQueries) != 1 {
				t.Fatalf("Expected 1 linked query, got: %d", len(linkedQueries))
			}

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
		mockClient := mocks.NewMockDBforPostgreSQLFlexibleServerAdministratorClient(ctrl)
		testClient := &testAdministratorClient{MockDBforPostgreSQLFlexibleServerAdministratorClient: mockClient}

		wrapper := manual.NewDBforPostgreSQLFlexibleServerAdministrator(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], serverName, true)
		if qErr == nil {
			t.Error("Expected error when providing insufficient query parts, but got nil")
		}
	})

	t.Run("GetWithEmptyServerName", func(t *testing.T) {
		mockClient := mocks.NewMockDBforPostgreSQLFlexibleServerAdministratorClient(ctrl)
		testClient := &testAdministratorClient{MockDBforPostgreSQLFlexibleServerAdministratorClient: mockClient}

		wrapper := manual.NewDBforPostgreSQLFlexibleServerAdministrator(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey("", objectID)
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr == nil {
			t.Error("Expected error when providing empty server name, but got nil")
		}
	})

	t.Run("GetWithEmptyObjectId", func(t *testing.T) {
		mockClient := mocks.NewMockDBforPostgreSQLFlexibleServerAdministratorClient(ctrl)
		testClient := &testAdministratorClient{MockDBforPostgreSQLFlexibleServerAdministratorClient: mockClient}

		wrapper := manual.NewDBforPostgreSQLFlexibleServerAdministrator(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(serverName, "")
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr == nil {
			t.Error("Expected error when providing empty objectId, but got nil")
		}
	})

	t.Run("Search", func(t *testing.T) {
		admin1 := createAzureAdministrator("00000000-0000-0000-0000-000000000001")
		admin2 := createAzureAdministrator("00000000-0000-0000-0000-000000000002")

		mockClient := mocks.NewMockDBforPostgreSQLFlexibleServerAdministratorClient(ctrl)
		mockPager := &mockAdministratorPager{
			pages: []armpostgresqlflexibleservers.AdministratorsMicrosoftEntraClientListByServerResponse{
				{
					AdministratorMicrosoftEntraList: armpostgresqlflexibleservers.AdministratorMicrosoftEntraList{
						Value: []*armpostgresqlflexibleservers.AdministratorMicrosoftEntra{admin1, admin2},
					},
				},
			},
		}

		testClient := &testAdministratorClient{
			MockDBforPostgreSQLFlexibleServerAdministratorClient: mockClient,
			pager: mockPager,
		}

		wrapper := manual.NewDBforPostgreSQLFlexibleServerAdministrator(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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

			if item.GetType() != azureshared.DBforPostgreSQLFlexibleServerAdministrator.String() {
				t.Errorf("Expected type %s, got %s", azureshared.DBforPostgreSQLFlexibleServerAdministrator, item.GetType())
			}
		}
	})

	t.Run("SearchWithEmptyServerName", func(t *testing.T) {
		mockClient := mocks.NewMockDBforPostgreSQLFlexibleServerAdministratorClient(ctrl)
		testClient := &testAdministratorClient{MockDBforPostgreSQLFlexibleServerAdministratorClient: mockClient}

		wrapper := manual.NewDBforPostgreSQLFlexibleServerAdministrator(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		_, qErr := wrapper.Search(ctx, wrapper.Scopes()[0], "")
		if qErr == nil {
			t.Error("Expected error when providing empty server name, but got nil")
		}
	})

	t.Run("SearchWithNoQueryParts", func(t *testing.T) {
		mockClient := mocks.NewMockDBforPostgreSQLFlexibleServerAdministratorClient(ctrl)
		testClient := &testAdministratorClient{MockDBforPostgreSQLFlexibleServerAdministratorClient: mockClient}

		wrapper := manual.NewDBforPostgreSQLFlexibleServerAdministrator(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		_, qErr := wrapper.Search(ctx, wrapper.Scopes()[0])
		if qErr == nil {
			t.Error("Expected error when providing no query parts, but got nil")
		}
	})

	t.Run("SearchStream", func(t *testing.T) {
		admin1 := createAzureAdministrator("00000000-0000-0000-0000-000000000001")
		admin2 := createAzureAdministrator("00000000-0000-0000-0000-000000000002")

		mockClient := mocks.NewMockDBforPostgreSQLFlexibleServerAdministratorClient(ctrl)
		mockPager := &mockAdministratorPager{
			pages: []armpostgresqlflexibleservers.AdministratorsMicrosoftEntraClientListByServerResponse{
				{
					AdministratorMicrosoftEntraList: armpostgresqlflexibleservers.AdministratorMicrosoftEntraList{
						Value: []*armpostgresqlflexibleservers.AdministratorMicrosoftEntra{admin1, admin2},
					},
				},
			},
		}

		testClient := &testAdministratorClient{
			MockDBforPostgreSQLFlexibleServerAdministratorClient: mockClient,
			pager: mockPager,
		}

		wrapper := manual.NewDBforPostgreSQLFlexibleServerAdministrator(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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

	t.Run("ErrorHandling_Get", func(t *testing.T) {
		expectedErr := errors.New("administrator not found")

		mockClient := mocks.NewMockDBforPostgreSQLFlexibleServerAdministratorClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, serverName, "nonexistent").Return(
			armpostgresqlflexibleservers.AdministratorsMicrosoftEntraClientGetResponse{}, expectedErr)

		testClient := &testAdministratorClient{MockDBforPostgreSQLFlexibleServerAdministratorClient: mockClient}
		wrapper := manual.NewDBforPostgreSQLFlexibleServerAdministrator(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(serverName, "nonexistent")
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr == nil {
			t.Error("Expected error when getting non-existent administrator, but got nil")
		}
	})

	t.Run("ErrorHandling_Search", func(t *testing.T) {
		mockClient := mocks.NewMockDBforPostgreSQLFlexibleServerAdministratorClient(ctrl)
		errorPager := &errorAdministratorPager{}

		testClient := &testAdministratorClient{
			MockDBforPostgreSQLFlexibleServerAdministratorClient: mockClient,
			pager: errorPager,
		}

		wrapper := manual.NewDBforPostgreSQLFlexibleServerAdministrator(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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

	t.Run("Search_AdminWithNilName", func(t *testing.T) {
		validAdmin := createAzureAdministrator("00000000-0000-0000-0000-000000000001")
		nilNameAdmin := &armpostgresqlflexibleservers.AdministratorMicrosoftEntra{
			Name: nil,
		}

		mockClient := mocks.NewMockDBforPostgreSQLFlexibleServerAdministratorClient(ctrl)
		mockPager := &mockAdministratorPager{
			pages: []armpostgresqlflexibleservers.AdministratorsMicrosoftEntraClientListByServerResponse{
				{
					AdministratorMicrosoftEntraList: armpostgresqlflexibleservers.AdministratorMicrosoftEntraList{
						Value: []*armpostgresqlflexibleservers.AdministratorMicrosoftEntra{nilNameAdmin, validAdmin},
					},
				},
			},
		}

		testClient := &testAdministratorClient{
			MockDBforPostgreSQLFlexibleServerAdministratorClient: mockClient,
			pager: mockPager,
		}

		wrapper := manual.NewDBforPostgreSQLFlexibleServerAdministrator(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support Search operation")
		}

		sdpItems, err := searchable.Search(ctx, wrapper.Scopes()[0], serverName, true)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if len(sdpItems) != 1 {
			t.Fatalf("Expected 1 item (nil name should be skipped), got: %d", len(sdpItems))
		}

		expectedUniqueValue := shared.CompositeLookupKey(serverName, "00000000-0000-0000-0000-000000000001")
		if sdpItems[0].UniqueAttributeValue() != expectedUniqueValue {
			t.Errorf("Expected unique value %s, got %s", expectedUniqueValue, sdpItems[0].UniqueAttributeValue())
		}
	})
}

// createAzureAdministrator creates a mock Azure administrator for testing
func createAzureAdministrator(objectID string) *armpostgresqlflexibleservers.AdministratorMicrosoftEntra {
	principalType := armpostgresqlflexibleservers.PrincipalTypeUser
	return &armpostgresqlflexibleservers.AdministratorMicrosoftEntra{
		ID:   new("/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.DBforPostgreSQL/flexibleServers/test-server/administrators/" + objectID),
		Name: new(objectID),
		Type: new("Microsoft.DBforPostgreSQL/flexibleServers/administrators"),
		Properties: &armpostgresqlflexibleservers.AdministratorMicrosoftEntraProperties{
			ObjectID:      new(objectID),
			PrincipalName: new("admin@example.com"),
			PrincipalType: &principalType,
			TenantID:      new("tenant-id"),
		},
	}
}
