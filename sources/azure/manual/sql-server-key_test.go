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

// mockSqlServerKeysPager is a simple mock implementation of SqlServerKeysPager
type mockSqlServerKeysPager struct {
	pages []armsql.ServerKeysClientListByServerResponse
	index int
}

func (m *mockSqlServerKeysPager) More() bool {
	return m.index < len(m.pages)
}

func (m *mockSqlServerKeysPager) NextPage(ctx context.Context) (armsql.ServerKeysClientListByServerResponse, error) {
	if m.index >= len(m.pages) {
		return armsql.ServerKeysClientListByServerResponse{}, errors.New("no more pages")
	}
	page := m.pages[m.index]
	m.index++
	return page, nil
}

// errorSqlServerKeysPager is a mock pager that always returns an error
type errorSqlServerKeysPager struct{}

func (e *errorSqlServerKeysPager) More() bool {
	return true
}

func (e *errorSqlServerKeysPager) NextPage(ctx context.Context) (armsql.ServerKeysClientListByServerResponse, error) {
	return armsql.ServerKeysClientListByServerResponse{}, errors.New("pager error")
}

// testSqlServerKeysClient wraps the mock to implement the correct interface
type testSqlServerKeysClient struct {
	*mocks.MockSqlServerKeysClient
	pager clients.SqlServerKeysPager
}

func (t *testSqlServerKeysClient) NewListByServerPager(resourceGroupName, serverName string) clients.SqlServerKeysPager {
	return t.pager
}

func TestSqlServerKey(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"
	serverName := "test-server"
	keyName := "test-key"

	t.Run("Get", func(t *testing.T) {
		serverKey := createAzureSqlServerKey(serverName, keyName, "")

		mockClient := mocks.NewMockSqlServerKeysClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, serverName, keyName).Return(
			armsql.ServerKeysClientGetResponse{
				ServerKey: *serverKey,
			}, nil)

		testClient := &testSqlServerKeysClient{MockSqlServerKeysClient: mockClient}
		wrapper := manual.NewSqlServerKey(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		// Get requires serverName and keyName as query parts
		query := shared.CompositeLookupKey(serverName, keyName)
		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.SQLServerKey.String() {
			t.Errorf("Expected type %s, got %s", azureshared.SQLServerKey, sdpItem.GetType())
		}

		if sdpItem.GetUniqueAttribute() != "uniqueAttr" {
			t.Errorf("Expected unique attribute 'uniqueAttr', got %s", sdpItem.GetUniqueAttribute())
		}

		expectedUniqueAttrValue := shared.CompositeLookupKey(serverName, keyName)
		if sdpItem.UniqueAttributeValue() != expectedUniqueAttrValue {
			t.Errorf("Expected unique attribute value %s, got %s", expectedUniqueAttrValue, sdpItem.UniqueAttributeValue())
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
				{
					// SQLServer link (parent)
					ExpectedType:   azureshared.SQLServer.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  serverName,
					ExpectedScope:  subscriptionID + "." + resourceGroup,
				},
			}

			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("Get_WithKeyVaultKey", func(t *testing.T) {
		keyVaultKeyURI := "https://my-vault.vault.azure.net/keys/my-key/12345"
		serverKey := createAzureSqlServerKey(serverName, keyName, keyVaultKeyURI)

		mockClient := mocks.NewMockSqlServerKeysClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, serverName, keyName).Return(
			armsql.ServerKeysClientGetResponse{
				ServerKey: *serverKey,
			}, nil)

		testClient := &testSqlServerKeysClient{MockSqlServerKeysClient: mockClient}
		wrapper := manual.NewSqlServerKey(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(serverName, keyName)
		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
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
					// KeyVaultKey link
					ExpectedType:   azureshared.KeyVaultKey.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey("my-vault", "my-key"),
					ExpectedScope:  subscriptionID + "." + resourceGroup,
				},
			}

			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("GetWithInsufficientQueryParts", func(t *testing.T) {
		mockClient := mocks.NewMockSqlServerKeysClient(ctrl)
		testClient := &testSqlServerKeysClient{MockSqlServerKeysClient: mockClient}

		wrapper := manual.NewSqlServerKey(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		// Test with insufficient query parts (only server name)
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], serverName, true)
		if qErr == nil {
			t.Error("Expected error when providing insufficient query parts, but got nil")
		}
	})

	t.Run("GetWithEmptyServerName", func(t *testing.T) {
		mockClient := mocks.NewMockSqlServerKeysClient(ctrl)
		testClient := &testSqlServerKeysClient{MockSqlServerKeysClient: mockClient}

		wrapper := manual.NewSqlServerKey(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		// Test with empty server name
		query := shared.CompositeLookupKey("", keyName)
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr == nil {
			t.Error("Expected error when providing empty server name, but got nil")
		}
	})

	t.Run("GetWithEmptyKeyName", func(t *testing.T) {
		mockClient := mocks.NewMockSqlServerKeysClient(ctrl)
		testClient := &testSqlServerKeysClient{MockSqlServerKeysClient: mockClient}

		wrapper := manual.NewSqlServerKey(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		// Test with empty key name
		query := shared.CompositeLookupKey(serverName, "")
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr == nil {
			t.Error("Expected error when providing empty key name, but got nil")
		}
	})

	t.Run("Search", func(t *testing.T) {
		serverKey1 := createAzureSqlServerKey(serverName, "key-1", "")
		serverKey2 := createAzureSqlServerKey(serverName, "key-2", "")

		mockClient := mocks.NewMockSqlServerKeysClient(ctrl)
		mockPager := &mockSqlServerKeysPager{
			pages: []armsql.ServerKeysClientListByServerResponse{
				{
					ServerKeyListResult: armsql.ServerKeyListResult{
						Value: []*armsql.ServerKey{serverKey1, serverKey2},
					},
				},
			},
		}

		testClient := &testSqlServerKeysClient{
			MockSqlServerKeysClient: mockClient,
			pager:                   mockPager,
		}

		wrapper := manual.NewSqlServerKey(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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

			if item.GetType() != azureshared.SQLServerKey.String() {
				t.Errorf("Expected type %s, got %s", azureshared.SQLServerKey, item.GetType())
			}
		}
	})

	t.Run("SearchStream", func(t *testing.T) {
		serverKey1 := createAzureSqlServerKey(serverName, "key-1", "")
		serverKey2 := createAzureSqlServerKey(serverName, "key-2", "")

		mockClient := mocks.NewMockSqlServerKeysClient(ctrl)
		mockPager := &mockSqlServerKeysPager{
			pages: []armsql.ServerKeysClientListByServerResponse{
				{
					ServerKeyListResult: armsql.ServerKeyListResult{
						Value: []*armsql.ServerKey{serverKey1, serverKey2},
					},
				},
			},
		}

		testClient := &testSqlServerKeysClient{
			MockSqlServerKeysClient: mockClient,
			pager:                   mockPager,
		}

		wrapper := manual.NewSqlServerKey(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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

	t.Run("Search_WithNilName", func(t *testing.T) {
		serverKey1 := createAzureSqlServerKey(serverName, "key-1", "")
		serverKey2 := &armsql.ServerKey{
			Name:     nil, // Key with nil name should be skipped
			Location: new("eastus"),
			ID:       new("/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.Sql/servers/test-server/keys/key-2"),
			Properties: &armsql.ServerKeyProperties{
				ServerKeyType: new(armsql.ServerKeyTypeServiceManaged),
			},
		}

		mockClient := mocks.NewMockSqlServerKeysClient(ctrl)
		mockPager := &mockSqlServerKeysPager{
			pages: []armsql.ServerKeysClientListByServerResponse{
				{
					ServerKeyListResult: armsql.ServerKeyListResult{
						Value: []*armsql.ServerKey{serverKey1, serverKey2},
					},
				},
			},
		}

		testClient := &testSqlServerKeysClient{
			MockSqlServerKeysClient: mockClient,
			pager:                   mockPager,
		}

		wrapper := manual.NewSqlServerKey(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support Search operation")
		}

		sdpItems, err := searchable.Search(ctx, wrapper.Scopes()[0], serverName, true)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		// Should only return 1 item (key with nil name is skipped)
		if len(sdpItems) != 1 {
			t.Fatalf("Expected 1 item (nil name filtered out), got: %d", len(sdpItems))
		}

		if sdpItems[0].UniqueAttributeValue() != shared.CompositeLookupKey(serverName, "key-1") {
			t.Fatalf("Expected key name 'key-1', got: %s", sdpItems[0].UniqueAttributeValue())
		}
	})

	t.Run("Search_InvalidQueryParts", func(t *testing.T) {
		mockClient := mocks.NewMockSqlServerKeysClient(ctrl)
		testClient := &testSqlServerKeysClient{MockSqlServerKeysClient: mockClient}

		wrapper := manual.NewSqlServerKey(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		// Test Search directly with no query parts - should return error before calling NewListByServerPager
		_, qErr := wrapper.Search(ctx, wrapper.Scopes()[0])
		if qErr == nil {
			t.Error("Expected error when providing no query parts, but got nil")
		}
	})

	t.Run("SearchWithEmptyServerName", func(t *testing.T) {
		mockClient := mocks.NewMockSqlServerKeysClient(ctrl)
		testClient := &testSqlServerKeysClient{MockSqlServerKeysClient: mockClient}

		wrapper := manual.NewSqlServerKey(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		// Test Search with empty server name
		_, qErr := wrapper.Search(ctx, wrapper.Scopes()[0], "")
		if qErr == nil {
			t.Error("Expected error when providing empty server name in Search, but got nil")
		}
	})

	t.Run("ErrorHandling_Get", func(t *testing.T) {
		expectedErr := errors.New("key not found")

		mockClient := mocks.NewMockSqlServerKeysClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, serverName, "nonexistent-key").Return(
			armsql.ServerKeysClientGetResponse{}, expectedErr)

		testClient := &testSqlServerKeysClient{MockSqlServerKeysClient: mockClient}
		wrapper := manual.NewSqlServerKey(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(serverName, "nonexistent-key")
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr == nil {
			t.Error("Expected error when getting non-existent key, but got nil")
		}
	})

	t.Run("ErrorHandling_Search", func(t *testing.T) {
		mockClient := mocks.NewMockSqlServerKeysClient(ctrl)
		// Create a pager that returns an error when NextPage is called
		errorPager := &errorSqlServerKeysPager{}

		testClient := &testSqlServerKeysClient{
			MockSqlServerKeysClient: mockClient,
			pager:                   errorPager,
		}

		wrapper := manual.NewSqlServerKey(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support Search operation")
		}

		_, err := searchable.Search(ctx, wrapper.Scopes()[0], serverName, true)
		// The Search implementation should return an error when pager.NextPage returns an error
		if err == nil {
			t.Error("Expected error from pager when NextPage returns an error, but got nil")
		}
	})

	t.Run("InterfaceCompliance", func(t *testing.T) {
		mockClient := mocks.NewMockSqlServerKeysClient(ctrl)
		testClient := &testSqlServerKeysClient{MockSqlServerKeysClient: mockClient}
		wrapper := manual.NewSqlServerKey(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		// Cast to sources.Wrapper to access interface methods
		w := wrapper.(sources.Wrapper)

		// Verify IAMPermissions
		permissions := w.IAMPermissions()
		if len(permissions) == 0 {
			t.Error("Expected IAMPermissions to return at least one permission")
		}
		expectedPermission := "Microsoft.Sql/servers/keys/read"
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
		if !potentialLinks[azureshared.KeyVaultKey] {
			t.Error("Expected PotentialLinks to include KeyVaultKey")
		}

		// Verify PredefinedRole using type assertion to the searchable wrapper
		if sw, ok := wrapper.(interface{ PredefinedRole() string }); ok {
			role := sw.PredefinedRole()
			if role != "Reader" {
				t.Errorf("Expected PredefinedRole to be 'Reader', got %s", role)
			}
		}
	})
}

// createAzureSqlServerKey creates a mock Azure SQL Server Key for testing
func createAzureSqlServerKey(serverName, keyName, keyVaultKeyURI string) *armsql.ServerKey {
	keyID := "/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.Sql/servers/" + serverName + "/keys/" + keyName

	keyType := armsql.ServerKeyTypeServiceManaged
	if keyVaultKeyURI != "" {
		keyType = armsql.ServerKeyTypeAzureKeyVault
	}

	serverKey := &armsql.ServerKey{
		Name:     new(keyName),
		Location: new("eastus"),
		ID:       new(keyID),
		Properties: &armsql.ServerKeyProperties{
			ServerKeyType: new(keyType),
		},
	}

	if keyVaultKeyURI != "" {
		serverKey.Properties.URI = new(keyVaultKeyURI)
	}

	return serverKey
}
