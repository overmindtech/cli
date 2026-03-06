package manual_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/batch/armbatch/v4"
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

// mockBatchApplicationsPager is a mock implementation of BatchApplicationsPager.
type mockBatchApplicationsPager struct {
	pages []armbatch.ApplicationClientListResponse
	index int
}

func (m *mockBatchApplicationsPager) More() bool {
	return m.index < len(m.pages)
}

func (m *mockBatchApplicationsPager) NextPage(ctx context.Context) (armbatch.ApplicationClientListResponse, error) {
	if m.index >= len(m.pages) {
		return armbatch.ApplicationClientListResponse{}, errors.New("no more pages")
	}
	page := m.pages[m.index]
	m.index++
	return page, nil
}

// errorBatchApplicationsPager is a mock pager that always returns an error.
type errorBatchApplicationsPager struct{}

func (e *errorBatchApplicationsPager) More() bool {
	return true
}

func (e *errorBatchApplicationsPager) NextPage(ctx context.Context) (armbatch.ApplicationClientListResponse, error) {
	return armbatch.ApplicationClientListResponse{}, errors.New("pager error")
}

// testBatchApplicationsClient wraps the mock and injects a pager from List().
type testBatchApplicationsClient struct {
	*mocks.MockBatchApplicationsClient
	pager clients.BatchApplicationsPager
}

func (t *testBatchApplicationsClient) List(ctx context.Context, resourceGroupName, accountName string) clients.BatchApplicationsPager {
	if t.pager != nil {
		return t.pager
	}
	return t.MockBatchApplicationsClient.List(ctx, resourceGroupName, accountName)
}

func createAzureBatchApplication(name string) *armbatch.Application {
	allowUpdates := true
	return &armbatch.Application{
		ID:   new("/subscriptions/sub/resourceGroups/rg/providers/Microsoft.Batch/batchAccounts/acc/applications/" + name),
		Name: new(name),
		Type: new("Microsoft.Batch/batchAccounts/applications"),
		Properties: &armbatch.ApplicationProperties{
			DisplayName:   new("Test application " + name),
			AllowUpdates:  &allowUpdates,
		},
		Tags: map[string]*string{"env": new("test")},
	}
}

func TestBatchBatchApplication(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"
	scope := subscriptionID + "." + resourceGroup
	accountName := "test-batch-account"
	applicationName := "test-app"

	t.Run("Get", func(t *testing.T) {
		app := createAzureBatchApplication(applicationName)

		mockClient := mocks.NewMockBatchApplicationsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, accountName, applicationName).Return(
			armbatch.ApplicationClientGetResponse{
				Application: *app,
			}, nil)

		wrapper := manual.NewBatchBatchApplication(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(accountName, applicationName)
		sdpItem, qErr := adapter.Get(ctx, scope, query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.BatchBatchApplication.String() {
			t.Errorf("Expected type %s, got %s", azureshared.BatchBatchApplication.String(), sdpItem.GetType())
		}

		if sdpItem.GetUniqueAttribute() != "uniqueAttr" {
			t.Errorf("Expected unique attribute 'uniqueAttr', got %s", sdpItem.GetUniqueAttribute())
		}

		expectedUnique := shared.CompositeLookupKey(accountName, applicationName)
		if sdpItem.UniqueAttributeValue() != expectedUnique {
			t.Errorf("Expected unique attribute value %s, got %s", expectedUnique, sdpItem.UniqueAttributeValue())
		}

		if sdpItem.GetScope() != scope {
			t.Errorf("Expected scope %s, got %s", scope, sdpItem.GetScope())
		}

		if err := sdpItem.Validate(); err != nil {
			t.Fatalf("Expected valid item, got: %v", err)
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{ExpectedType: azureshared.BatchBatchAccount.String(), ExpectedMethod: sdp.QueryMethod_GET, ExpectedQuery: accountName, ExpectedScope: scope},
				{ExpectedType: azureshared.BatchBatchApplicationPackage.String(), ExpectedMethod: sdp.QueryMethod_SEARCH, ExpectedQuery: shared.CompositeLookupKey(accountName, applicationName), ExpectedScope: scope},
			}
			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("Get_InvalidQueryParts", func(t *testing.T) {
		mockClient := mocks.NewMockBatchApplicationsClient(ctrl)
		wrapper := manual.NewBatchBatchApplication(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, scope, accountName, true)
		if qErr == nil {
			t.Error("Expected error when Get with insufficient query parts, but got nil")
		}
	})

	t.Run("Get_ClientError", func(t *testing.T) {
		expectedErr := errors.New("application not found")
		mockClient := mocks.NewMockBatchApplicationsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, accountName, "nonexistent").Return(
			armbatch.ApplicationClientGetResponse{}, expectedErr)

		wrapper := manual.NewBatchBatchApplication(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(accountName, "nonexistent")
		_, qErr := adapter.Get(ctx, scope, query, true)
		if qErr == nil {
			t.Error("Expected error when client returns error, but got nil")
		}
	})

	t.Run("Search", func(t *testing.T) {
		app1 := createAzureBatchApplication("app-1")
		app2 := createAzureBatchApplication("app-2")

		mockClient := mocks.NewMockBatchApplicationsClient(ctrl)
		pages := []armbatch.ApplicationClientListResponse{
			{
				ListApplicationsResult: armbatch.ListApplicationsResult{
					Value: []*armbatch.Application{app1, app2},
				},
			},
		}
		mockPager := &mockBatchApplicationsPager{pages: pages}
		testClient := &testBatchApplicationsClient{
			MockBatchApplicationsClient: mockClient,
			pager:                       mockPager,
		}

		wrapper := manual.NewBatchBatchApplication(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support Search operation")
		}

		sdpItems, err := searchable.Search(ctx, scope, accountName, true)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if len(sdpItems) != 2 {
			t.Fatalf("Expected 2 items, got: %d", len(sdpItems))
		}

		for _, item := range sdpItems {
			if err := item.Validate(); err != nil {
				t.Errorf("Expected valid item, got: %v", err)
			}
		}
	})

	t.Run("Search_InvalidQueryParts", func(t *testing.T) {
		mockClient := mocks.NewMockBatchApplicationsClient(ctrl)
		wrapper := manual.NewBatchBatchApplication(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		_, qErr := wrapper.Search(ctx, scope)
		if qErr == nil {
			t.Error("Expected error when Search with no query parts, but got nil")
		}
	})

	t.Run("Search_PagerError", func(t *testing.T) {
		mockClient := mocks.NewMockBatchApplicationsClient(ctrl)
		errorPager := &errorBatchApplicationsPager{}
		testClient := &testBatchApplicationsClient{
			MockBatchApplicationsClient: mockClient,
			pager:                       errorPager,
		}

		wrapper := manual.NewBatchBatchApplication(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		_, qErr := wrapper.Search(ctx, scope, accountName)
		if qErr == nil {
			t.Error("Expected error when pager returns error, but got nil")
		}
	})

	t.Run("PotentialLinks", func(t *testing.T) {
		mockClient := mocks.NewMockBatchApplicationsClient(ctrl)
		wrapper := manual.NewBatchBatchApplication(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		links := wrapper.PotentialLinks()
		if !links[azureshared.BatchBatchAccount] {
			t.Error("PotentialLinks() should include BatchBatchAccount")
		}
		if !links[azureshared.BatchBatchApplicationPackage] {
			t.Error("PotentialLinks() should include BatchBatchApplicationPackage")
		}
	})

	t.Run("ImplementsSearchableAdapter", func(t *testing.T) {
		mockClient := mocks.NewMockBatchApplicationsClient(ctrl)
		wrapper := manual.NewBatchBatchApplication(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Error("Adapter should implement SearchableAdapter interface")
		}
	})
}
