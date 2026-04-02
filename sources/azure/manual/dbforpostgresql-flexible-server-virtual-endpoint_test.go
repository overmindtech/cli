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

type mockDBforPostgreSQLFlexibleServerVirtualEndpointPager struct {
	pages []armpostgresqlflexibleservers.VirtualEndpointsClientListByServerResponse
	index int
}

func (m *mockDBforPostgreSQLFlexibleServerVirtualEndpointPager) More() bool {
	return m.index < len(m.pages)
}

func (m *mockDBforPostgreSQLFlexibleServerVirtualEndpointPager) NextPage(ctx context.Context) (armpostgresqlflexibleservers.VirtualEndpointsClientListByServerResponse, error) {
	if m.index >= len(m.pages) {
		return armpostgresqlflexibleservers.VirtualEndpointsClientListByServerResponse{}, errors.New("no more pages")
	}
	page := m.pages[m.index]
	m.index++
	return page, nil
}

type errorDBforPostgreSQLFlexibleServerVirtualEndpointPager struct{}

func (e *errorDBforPostgreSQLFlexibleServerVirtualEndpointPager) More() bool {
	return true
}

func (e *errorDBforPostgreSQLFlexibleServerVirtualEndpointPager) NextPage(ctx context.Context) (armpostgresqlflexibleservers.VirtualEndpointsClientListByServerResponse, error) {
	return armpostgresqlflexibleservers.VirtualEndpointsClientListByServerResponse{}, errors.New("pager error")
}

type testDBforPostgreSQLFlexibleServerVirtualEndpointClient struct {
	*mocks.MockDBforPostgreSQLFlexibleServerVirtualEndpointClient
	pager clients.DBforPostgreSQLFlexibleServerVirtualEndpointPager
}

func (t *testDBforPostgreSQLFlexibleServerVirtualEndpointClient) ListByServer(ctx context.Context, resourceGroupName, serverName string) clients.DBforPostgreSQLFlexibleServerVirtualEndpointPager {
	return t.pager
}

func TestDBforPostgreSQLFlexibleServerVirtualEndpoint(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"
	serverName := "test-server"
	virtualEndpointName := "test-virtual-endpoint"

	t.Run("Get", func(t *testing.T) {
		virtualEndpoint := createAzurePostgreSQLFlexibleServerVirtualEndpoint(serverName, virtualEndpointName)

		mockClient := mocks.NewMockDBforPostgreSQLFlexibleServerVirtualEndpointClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, serverName, virtualEndpointName).Return(
			armpostgresqlflexibleservers.VirtualEndpointsClientGetResponse{
				VirtualEndpoint: *virtualEndpoint,
			}, nil)

		wrapper := manual.NewDBforPostgreSQLFlexibleServerVirtualEndpoint(&testDBforPostgreSQLFlexibleServerVirtualEndpointClient{MockDBforPostgreSQLFlexibleServerVirtualEndpointClient: mockClient}, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(serverName, virtualEndpointName)
		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.DBforPostgreSQLFlexibleServerVirtualEndpoint.String() {
			t.Errorf("Expected type %s, got %s", azureshared.DBforPostgreSQLFlexibleServerVirtualEndpoint, sdpItem.GetType())
		}

		if sdpItem.GetUniqueAttribute() != "uniqueAttr" {
			t.Errorf("Expected unique attribute 'uniqueAttr', got %s", sdpItem.GetUniqueAttribute())
		}

		expectedUniqueAttrValue := shared.CompositeLookupKey(serverName, virtualEndpointName)
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
				{
					ExpectedType:   azureshared.DBforPostgreSQLFlexibleServer.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "member-server-1",
					ExpectedScope:  subscriptionID + "." + resourceGroup,
				},
				{
					ExpectedType:   stdlib.NetworkDNS.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  "test-endpoint.postgres.database.azure.com",
					ExpectedScope:  "global",
				},
			}
			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("GetWithInsufficientQueryParts", func(t *testing.T) {
		mockClient := mocks.NewMockDBforPostgreSQLFlexibleServerVirtualEndpointClient(ctrl)
		wrapper := manual.NewDBforPostgreSQLFlexibleServerVirtualEndpoint(&testDBforPostgreSQLFlexibleServerVirtualEndpointClient{MockDBforPostgreSQLFlexibleServerVirtualEndpointClient: mockClient}, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], serverName, true)
		if qErr == nil {
			t.Error("Expected error when providing only serverName (1 query part), but got nil")
		}
	})

	t.Run("GetWithEmptyServerName", func(t *testing.T) {
		mockClient := mocks.NewMockDBforPostgreSQLFlexibleServerVirtualEndpointClient(ctrl)
		wrapper := manual.NewDBforPostgreSQLFlexibleServerVirtualEndpoint(&testDBforPostgreSQLFlexibleServerVirtualEndpointClient{MockDBforPostgreSQLFlexibleServerVirtualEndpointClient: mockClient}, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey("", virtualEndpointName)
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr == nil {
			t.Error("Expected error when serverName is empty, but got nil")
		}
	})

	t.Run("GetWithEmptyVirtualEndpointName", func(t *testing.T) {
		mockClient := mocks.NewMockDBforPostgreSQLFlexibleServerVirtualEndpointClient(ctrl)
		wrapper := manual.NewDBforPostgreSQLFlexibleServerVirtualEndpoint(&testDBforPostgreSQLFlexibleServerVirtualEndpointClient{MockDBforPostgreSQLFlexibleServerVirtualEndpointClient: mockClient}, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(serverName, "")
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr == nil {
			t.Error("Expected error when virtualEndpointName is empty, but got nil")
		}
	})

	t.Run("Search", func(t *testing.T) {
		virtualEndpoint1 := createAzurePostgreSQLFlexibleServerVirtualEndpoint(serverName, "vep1")
		virtualEndpoint2 := createAzurePostgreSQLFlexibleServerVirtualEndpoint(serverName, "vep2")

		mockClient := mocks.NewMockDBforPostgreSQLFlexibleServerVirtualEndpointClient(ctrl)
		pager := &mockDBforPostgreSQLFlexibleServerVirtualEndpointPager{
			pages: []armpostgresqlflexibleservers.VirtualEndpointsClientListByServerResponse{
				{
					VirtualEndpointsList: armpostgresqlflexibleservers.VirtualEndpointsList{
						Value: []*armpostgresqlflexibleservers.VirtualEndpoint{virtualEndpoint1, virtualEndpoint2},
					},
				},
			},
		}

		testClient := &testDBforPostgreSQLFlexibleServerVirtualEndpointClient{
			MockDBforPostgreSQLFlexibleServerVirtualEndpointClient: mockClient,
			pager: pager,
		}
		wrapper := manual.NewDBforPostgreSQLFlexibleServerVirtualEndpoint(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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
		virtualEndpoint1 := createAzurePostgreSQLFlexibleServerVirtualEndpoint(serverName, "vep1")

		mockClient := mocks.NewMockDBforPostgreSQLFlexibleServerVirtualEndpointClient(ctrl)
		pager := &mockDBforPostgreSQLFlexibleServerVirtualEndpointPager{
			pages: []armpostgresqlflexibleservers.VirtualEndpointsClientListByServerResponse{
				{
					VirtualEndpointsList: armpostgresqlflexibleservers.VirtualEndpointsList{
						Value: []*armpostgresqlflexibleservers.VirtualEndpoint{virtualEndpoint1},
					},
				},
			},
		}

		testClient := &testDBforPostgreSQLFlexibleServerVirtualEndpointClient{
			MockDBforPostgreSQLFlexibleServerVirtualEndpointClient: mockClient,
			pager: pager,
		}
		wrapper := manual.NewDBforPostgreSQLFlexibleServerVirtualEndpoint(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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
		mockClient := mocks.NewMockDBforPostgreSQLFlexibleServerVirtualEndpointClient(ctrl)
		wrapper := manual.NewDBforPostgreSQLFlexibleServerVirtualEndpoint(&testDBforPostgreSQLFlexibleServerVirtualEndpointClient{MockDBforPostgreSQLFlexibleServerVirtualEndpointClient: mockClient}, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		_, qErr := wrapper.Search(ctx, wrapper.Scopes()[0])
		if qErr == nil {
			t.Error("Expected error when providing no query parts, but got nil")
		}
	})

	t.Run("SearchWithEmptyServerName", func(t *testing.T) {
		mockClient := mocks.NewMockDBforPostgreSQLFlexibleServerVirtualEndpointClient(ctrl)
		wrapper := manual.NewDBforPostgreSQLFlexibleServerVirtualEndpoint(&testDBforPostgreSQLFlexibleServerVirtualEndpointClient{MockDBforPostgreSQLFlexibleServerVirtualEndpointClient: mockClient}, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		_, qErr := wrapper.Search(ctx, wrapper.Scopes()[0], "")
		if qErr == nil {
			t.Error("Expected error when serverName is empty, but got nil")
		}
	})

	t.Run("ErrorHandling_Get", func(t *testing.T) {
		expectedErr := errors.New("virtual endpoint not found")

		mockClient := mocks.NewMockDBforPostgreSQLFlexibleServerVirtualEndpointClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, serverName, "nonexistent-vep").Return(
			armpostgresqlflexibleservers.VirtualEndpointsClientGetResponse{}, expectedErr)

		wrapper := manual.NewDBforPostgreSQLFlexibleServerVirtualEndpoint(&testDBforPostgreSQLFlexibleServerVirtualEndpointClient{MockDBforPostgreSQLFlexibleServerVirtualEndpointClient: mockClient}, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(serverName, "nonexistent-vep")
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr == nil {
			t.Error("Expected error when getting non-existent virtual endpoint, but got nil")
		}
	})

	t.Run("ErrorHandling_Search", func(t *testing.T) {
		mockClient := mocks.NewMockDBforPostgreSQLFlexibleServerVirtualEndpointClient(ctrl)
		errorPager := &errorDBforPostgreSQLFlexibleServerVirtualEndpointPager{}
		testClient := &testDBforPostgreSQLFlexibleServerVirtualEndpointClient{
			MockDBforPostgreSQLFlexibleServerVirtualEndpointClient: mockClient,
			pager: errorPager,
		}

		wrapper := manual.NewDBforPostgreSQLFlexibleServerVirtualEndpoint(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		_, qErr := wrapper.Search(ctx, wrapper.Scopes()[0], serverName)
		if qErr == nil {
			t.Error("Expected error from Search when pager returns error, but got nil")
		}
	})

	t.Run("PotentialLinks", func(t *testing.T) {
		mockClient := mocks.NewMockDBforPostgreSQLFlexibleServerVirtualEndpointClient(ctrl)
		wrapper := manual.NewDBforPostgreSQLFlexibleServerVirtualEndpoint(&testDBforPostgreSQLFlexibleServerVirtualEndpointClient{MockDBforPostgreSQLFlexibleServerVirtualEndpointClient: mockClient}, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		potentialLinks := wrapper.PotentialLinks()

		expectedLinks := map[shared.ItemType]bool{
			azureshared.DBforPostgreSQLFlexibleServer: true,
			stdlib.NetworkDNS:                         true,
		}

		for linkType := range expectedLinks {
			if !potentialLinks[linkType] {
				t.Errorf("Expected PotentialLinks to include %s", linkType)
			}
		}
	})
}

func createAzurePostgreSQLFlexibleServerVirtualEndpoint(serverName, virtualEndpointName string) *armpostgresqlflexibleservers.VirtualEndpoint {
	virtualEndpointID := "/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.DBforPostgreSQL/flexibleServers/" + serverName + "/virtualEndpoints/" + virtualEndpointName
	endpointType := armpostgresqlflexibleservers.VirtualEndpointTypeReadWrite
	return &armpostgresqlflexibleservers.VirtualEndpoint{
		Name: new(virtualEndpointName),
		ID:   new(virtualEndpointID),
		Type: new("Microsoft.DBforPostgreSQL/flexibleServers/virtualEndpoints"),
		Properties: &armpostgresqlflexibleservers.VirtualEndpointResourceProperties{
			EndpointType: &endpointType,
			Members:      []*string{new("member-server-1")},
			VirtualEndpoints: []*string{
				new("test-endpoint.postgres.database.azure.com"),
			},
		},
	}
}
