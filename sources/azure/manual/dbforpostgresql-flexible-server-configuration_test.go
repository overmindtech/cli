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

// mockConfigurationsPager is a mock implementation of PostgreSQLConfigurationsPager
type mockConfigurationsPager struct {
	pages []armpostgresqlflexibleservers.ConfigurationsClientListByServerResponse
	index int
}

func (m *mockConfigurationsPager) More() bool {
	return m.index < len(m.pages)
}

func (m *mockConfigurationsPager) NextPage(ctx context.Context) (armpostgresqlflexibleservers.ConfigurationsClientListByServerResponse, error) {
	if m.index >= len(m.pages) {
		return armpostgresqlflexibleservers.ConfigurationsClientListByServerResponse{}, errors.New("no more pages")
	}
	page := m.pages[m.index]
	m.index++
	return page, nil
}

// errorConfigurationsPager is a mock pager that always returns an error
type errorConfigurationsPager struct{}

func (e *errorConfigurationsPager) More() bool {
	return true
}

func (e *errorConfigurationsPager) NextPage(ctx context.Context) (armpostgresqlflexibleservers.ConfigurationsClientListByServerResponse, error) {
	return armpostgresqlflexibleservers.ConfigurationsClientListByServerResponse{}, errors.New("pager error")
}

// testConfigurationsClient wraps the mock to implement the correct interface
type testConfigurationsClient struct {
	*mocks.MockPostgreSQLConfigurationsClient
	pager clients.PostgreSQLConfigurationsPager
}

func (t *testConfigurationsClient) NewListByServerPager(resourceGroupName string, serverName string, options *armpostgresqlflexibleservers.ConfigurationsClientListByServerOptions) clients.PostgreSQLConfigurationsPager {
	return t.pager
}

func TestDBforPostgreSQLFlexibleServerConfiguration(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"
	serverName := "test-server"
	configurationName := "shared_buffers"

	t.Run("Get", func(t *testing.T) {
		configuration := createAzureConfiguration(configurationName)

		mockClient := mocks.NewMockPostgreSQLConfigurationsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, serverName, configurationName, nil).Return(
			armpostgresqlflexibleservers.ConfigurationsClientGetResponse{
				Configuration: *configuration,
			}, nil)

		testClient := &testConfigurationsClient{MockPostgreSQLConfigurationsClient: mockClient}
		wrapper := manual.NewDBforPostgreSQLFlexibleServerConfiguration(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(serverName, configurationName)
		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.DBforPostgreSQLFlexibleServerConfiguration.String() {
			t.Errorf("Expected type %s, got %s", azureshared.DBforPostgreSQLFlexibleServerConfiguration, sdpItem.GetType())
		}

		if sdpItem.GetUniqueAttribute() != "uniqueAttr" {
			t.Errorf("Expected unique attribute 'uniqueAttr', got %s", sdpItem.GetUniqueAttribute())
		}

		if sdpItem.UniqueAttributeValue() != shared.CompositeLookupKey(serverName, configurationName) {
			t.Errorf("Expected unique attribute value %s, got %s", shared.CompositeLookupKey(serverName, configurationName), sdpItem.UniqueAttributeValue())
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

	t.Run("Search", func(t *testing.T) {
		config1 := createAzureConfiguration("shared_buffers")
		config2 := createAzureConfiguration("work_mem")

		mockClient := mocks.NewMockPostgreSQLConfigurationsClient(ctrl)
		mockPager := &mockConfigurationsPager{
			pages: []armpostgresqlflexibleservers.ConfigurationsClientListByServerResponse{
				{
					ConfigurationList: armpostgresqlflexibleservers.ConfigurationList{
						Value: []*armpostgresqlflexibleservers.Configuration{config1, config2},
					},
				},
			},
		}

		testClient := &testConfigurationsClient{
			MockPostgreSQLConfigurationsClient: mockClient,
			pager:                              mockPager,
		}

		wrapper := manual.NewDBforPostgreSQLFlexibleServerConfiguration(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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

			if item.GetType() != azureshared.DBforPostgreSQLFlexibleServerConfiguration.String() {
				t.Errorf("Expected type %s, got %s", azureshared.DBforPostgreSQLFlexibleServerConfiguration, item.GetType())
			}
		}
	})

	t.Run("SearchStream", func(t *testing.T) {
		config1 := createAzureConfiguration("shared_buffers")
		config2 := createAzureConfiguration("work_mem")

		mockClient := mocks.NewMockPostgreSQLConfigurationsClient(ctrl)
		mockPager := &mockConfigurationsPager{
			pages: []armpostgresqlflexibleservers.ConfigurationsClientListByServerResponse{
				{
					ConfigurationList: armpostgresqlflexibleservers.ConfigurationList{
						Value: []*armpostgresqlflexibleservers.Configuration{config1, config2},
					},
				},
			},
		}

		testClient := &testConfigurationsClient{
			MockPostgreSQLConfigurationsClient: mockClient,
			pager:                              mockPager,
		}

		wrapper := manual.NewDBforPostgreSQLFlexibleServerConfiguration(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		searchStreamable, ok := adapter.(discovery.SearchStreamableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support SearchStream operation")
		}

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

		searchStreamable.SearchStream(ctx, wrapper.Scopes()[0], serverName, true, stream)
		wg.Wait()

		if len(errs) != 0 {
			t.Fatalf("Expected no errors, got: %v", errs)
		}

		if len(items) != 2 {
			t.Fatalf("Expected 2 items, got: %d", len(items))
		}
	})

	t.Run("GetWithInsufficientQueryParts", func(t *testing.T) {
		mockClient := mocks.NewMockPostgreSQLConfigurationsClient(ctrl)
		testClient := &testConfigurationsClient{MockPostgreSQLConfigurationsClient: mockClient}

		wrapper := manual.NewDBforPostgreSQLFlexibleServerConfiguration(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], serverName, true)
		if qErr == nil {
			t.Error("Expected error when providing insufficient query parts, but got nil")
		}
	})

	t.Run("GetWithEmptyServerName", func(t *testing.T) {
		mockClient := mocks.NewMockPostgreSQLConfigurationsClient(ctrl)
		testClient := &testConfigurationsClient{MockPostgreSQLConfigurationsClient: mockClient}

		wrapper := manual.NewDBforPostgreSQLFlexibleServerConfiguration(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey("", configurationName)
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr == nil {
			t.Error("Expected error when providing empty server name, but got nil")
		}
	})

	t.Run("GetWithEmptyConfigurationName", func(t *testing.T) {
		mockClient := mocks.NewMockPostgreSQLConfigurationsClient(ctrl)
		testClient := &testConfigurationsClient{MockPostgreSQLConfigurationsClient: mockClient}

		wrapper := manual.NewDBforPostgreSQLFlexibleServerConfiguration(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(serverName, "")
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr == nil {
			t.Error("Expected error when providing empty configuration name, but got nil")
		}
	})

	t.Run("SearchWithEmptyServerName", func(t *testing.T) {
		mockClient := mocks.NewMockPostgreSQLConfigurationsClient(ctrl)
		testClient := &testConfigurationsClient{MockPostgreSQLConfigurationsClient: mockClient}

		wrapper := manual.NewDBforPostgreSQLFlexibleServerConfiguration(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		_, qErr := wrapper.Search(ctx, wrapper.Scopes()[0], "")
		if qErr == nil {
			t.Error("Expected error when providing empty server name, but got nil")
		}
	})

	t.Run("SearchWithNoQueryParts", func(t *testing.T) {
		mockClient := mocks.NewMockPostgreSQLConfigurationsClient(ctrl)
		testClient := &testConfigurationsClient{MockPostgreSQLConfigurationsClient: mockClient}

		wrapper := manual.NewDBforPostgreSQLFlexibleServerConfiguration(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support Search operation")
		}

		_, err := searchable.Search(ctx, wrapper.Scopes()[0], "", true)
		if err == nil {
			t.Error("Expected error when providing no query parts, but got nil")
		}
	})

	t.Run("Search_ConfigurationWithNilName", func(t *testing.T) {
		configWithName := createAzureConfiguration("shared_buffers")

		mockClient := mocks.NewMockPostgreSQLConfigurationsClient(ctrl)
		mockPager := &mockConfigurationsPager{
			pages: []armpostgresqlflexibleservers.ConfigurationsClientListByServerResponse{
				{
					ConfigurationList: armpostgresqlflexibleservers.ConfigurationList{
						Value: []*armpostgresqlflexibleservers.Configuration{
							{Name: nil}, // Configuration with nil name should be skipped
							configWithName,
						},
					},
				},
			},
		}

		testClient := &testConfigurationsClient{
			MockPostgreSQLConfigurationsClient: mockClient,
			pager:                              mockPager,
		}

		wrapper := manual.NewDBforPostgreSQLFlexibleServerConfiguration(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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
			t.Fatalf("Expected 1 item, got: %d", len(sdpItems))
		}

		if sdpItems[0].UniqueAttributeValue() != shared.CompositeLookupKey(serverName, "shared_buffers") {
			t.Errorf("Expected configuration name '%s', got %s", shared.CompositeLookupKey(serverName, "shared_buffers"), sdpItems[0].UniqueAttributeValue())
		}
	})

	t.Run("ErrorHandling_Get", func(t *testing.T) {
		expectedErr := errors.New("configuration not found")

		mockClient := mocks.NewMockPostgreSQLConfigurationsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, serverName, "nonexistent", nil).Return(
			armpostgresqlflexibleservers.ConfigurationsClientGetResponse{}, expectedErr)

		testClient := &testConfigurationsClient{MockPostgreSQLConfigurationsClient: mockClient}
		wrapper := manual.NewDBforPostgreSQLFlexibleServerConfiguration(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(serverName, "nonexistent")
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr == nil {
			t.Error("Expected error when getting non-existent configuration, but got nil")
		}
	})

	t.Run("ErrorHandling_Search", func(t *testing.T) {
		mockClient := mocks.NewMockPostgreSQLConfigurationsClient(ctrl)
		errorPager := &errorConfigurationsPager{}

		testClient := &testConfigurationsClient{
			MockPostgreSQLConfigurationsClient: mockClient,
			pager:                              errorPager,
		}

		wrapper := manual.NewDBforPostgreSQLFlexibleServerConfiguration(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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
}

// createAzureConfiguration creates a mock Azure configuration for testing
func createAzureConfiguration(name string) *armpostgresqlflexibleservers.Configuration {
	dataType := armpostgresqlflexibleservers.ConfigurationDataTypeInteger
	return &armpostgresqlflexibleservers.Configuration{
		ID:   new("/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.DBforPostgreSQL/flexibleServers/test-server/configurations/" + name),
		Name: new(name),
		Type: new("Microsoft.DBforPostgreSQL/flexibleServers/configurations"),
		Properties: &armpostgresqlflexibleservers.ConfigurationProperties{
			Value:         new("128MB"),
			DefaultValue:  new("128MB"),
			DataType:      &dataType,
			AllowedValues: new("16384-2097152"),
			Source:        new("system-default"),
			Description:   new("Sets the amount of memory the database server uses for shared memory buffers."),
		},
	}
}
