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
	"github.com/overmindtech/cli/sources/stdlib"
)

type mockBatchApplicationPackagesPager struct {
	pages []armbatch.ApplicationPackageClientListResponse
	index int
}

func (m *mockBatchApplicationPackagesPager) More() bool {
	return m.index < len(m.pages)
}

func (m *mockBatchApplicationPackagesPager) NextPage(ctx context.Context) (armbatch.ApplicationPackageClientListResponse, error) {
	if m.index >= len(m.pages) {
		return armbatch.ApplicationPackageClientListResponse{}, errors.New("no more pages")
	}
	page := m.pages[m.index]
	m.index++
	return page, nil
}

type errorBatchApplicationPackagesPager struct{}

func (e *errorBatchApplicationPackagesPager) More() bool {
	return true
}

func (e *errorBatchApplicationPackagesPager) NextPage(ctx context.Context) (armbatch.ApplicationPackageClientListResponse, error) {
	return armbatch.ApplicationPackageClientListResponse{}, errors.New("pager error")
}

type testBatchApplicationPackagesClient struct {
	*mocks.MockBatchApplicationPackagesClient
	pager clients.BatchApplicationPackagesPager
}

func (t *testBatchApplicationPackagesClient) List(ctx context.Context, resourceGroupName, accountName, applicationName string) clients.BatchApplicationPackagesPager {
	if t.pager != nil {
		return t.pager
	}
	return t.MockBatchApplicationPackagesClient.List(ctx, resourceGroupName, accountName, applicationName)
}

func createAzureBatchApplicationPackage(versionName string) *armbatch.ApplicationPackage {
	state := armbatch.PackageStateActive
	return &armbatch.ApplicationPackage{
		ID:   new("/subscriptions/sub/resourceGroups/rg/providers/Microsoft.Batch/batchAccounts/acc/applications/app/versions/" + versionName),
		Name: new(versionName),
		Type: new("Microsoft.Batch/batchAccounts/applications/versions"),
		Properties: &armbatch.ApplicationPackageProperties{
			State:      &state,
			Format:     new("zip"),
			StorageURL: new("https://teststorage.blob.core.windows.net/packages/" + versionName + ".zip"),
		},
		Tags: map[string]*string{"env": new("test")},
	}
}

func TestBatchBatchApplicationPackage(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"
	scope := subscriptionID + "." + resourceGroup
	accountName := "test-batch-account"
	applicationName := "test-app"
	versionName := "1.0"

	t.Run("Get", func(t *testing.T) {
		pkg := createAzureBatchApplicationPackage(versionName)

		mockClient := mocks.NewMockBatchApplicationPackagesClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, accountName, applicationName, versionName).Return(
			armbatch.ApplicationPackageClientGetResponse{
				ApplicationPackage: *pkg,
			}, nil)

		wrapper := manual.NewBatchBatchApplicationPackage(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(accountName, applicationName, versionName)
		sdpItem, qErr := adapter.Get(ctx, scope, query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.BatchBatchApplicationPackage.String() {
			t.Errorf("Expected type %s, got %s", azureshared.BatchBatchApplicationPackage.String(), sdpItem.GetType())
		}

		if sdpItem.GetUniqueAttribute() != "uniqueAttr" {
			t.Errorf("Expected unique attribute 'uniqueAttr', got %s", sdpItem.GetUniqueAttribute())
		}

		expectedUnique := shared.CompositeLookupKey(accountName, applicationName, versionName)
		if sdpItem.UniqueAttributeValue() != expectedUnique {
			t.Errorf("Expected unique attribute value %s, got %s", expectedUnique, sdpItem.UniqueAttributeValue())
		}

		if sdpItem.GetScope() != scope {
			t.Errorf("Expected scope %s, got %s", scope, sdpItem.GetScope())
		}

		if err := sdpItem.Validate(); err != nil {
			t.Fatalf("Expected valid item, got: %v", err)
		}

		if sdpItem.GetHealth() != sdp.Health_HEALTH_OK {
			t.Errorf("Expected health OK for active package, got %s", sdpItem.GetHealth())
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{ExpectedType: azureshared.BatchBatchApplication.String(), ExpectedMethod: sdp.QueryMethod_GET, ExpectedQuery: shared.CompositeLookupKey(accountName, applicationName), ExpectedScope: scope},
				{ExpectedType: azureshared.BatchBatchAccount.String(), ExpectedMethod: sdp.QueryMethod_GET, ExpectedQuery: accountName, ExpectedScope: scope},
				{ExpectedType: stdlib.NetworkDNS.String(), ExpectedMethod: sdp.QueryMethod_SEARCH, ExpectedQuery: "teststorage.blob.core.windows.net", ExpectedScope: "global"},
			}
			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("Get_InvalidQueryParts", func(t *testing.T) {
		mockClient := mocks.NewMockBatchApplicationPackagesClient(ctrl)
		wrapper := manual.NewBatchBatchApplicationPackage(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		// Only 2 parts instead of 3
		query := shared.CompositeLookupKey(accountName, applicationName)
		_, qErr := adapter.Get(ctx, scope, query, true)
		if qErr == nil {
			t.Error("Expected error when Get with insufficient query parts, but got nil")
		}
	})

	t.Run("Get_ClientError", func(t *testing.T) {
		expectedErr := errors.New("application package not found")
		mockClient := mocks.NewMockBatchApplicationPackagesClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, accountName, applicationName, "nonexistent").Return(
			armbatch.ApplicationPackageClientGetResponse{}, expectedErr)

		wrapper := manual.NewBatchBatchApplicationPackage(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(accountName, applicationName, "nonexistent")
		_, qErr := adapter.Get(ctx, scope, query, true)
		if qErr == nil {
			t.Error("Expected error when client returns error, but got nil")
		}
	})

	t.Run("Search", func(t *testing.T) {
		pkg1 := createAzureBatchApplicationPackage("1.0")
		pkg2 := createAzureBatchApplicationPackage("2.0")

		mockClient := mocks.NewMockBatchApplicationPackagesClient(ctrl)
		pages := []armbatch.ApplicationPackageClientListResponse{
			{
				ListApplicationPackagesResult: armbatch.ListApplicationPackagesResult{
					Value: []*armbatch.ApplicationPackage{pkg1, pkg2},
				},
			},
		}
		mockPager := &mockBatchApplicationPackagesPager{pages: pages}
		testClient := &testBatchApplicationPackagesClient{
			MockBatchApplicationPackagesClient: mockClient,
			pager:                              mockPager,
		}

		wrapper := manual.NewBatchBatchApplicationPackage(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support Search operation")
		}

		sdpItems, err := searchable.Search(ctx, scope, shared.CompositeLookupKey(accountName, applicationName), true)
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

	t.Run("SearchStream", func(t *testing.T) {
		pkg1 := createAzureBatchApplicationPackage("1.0")
		pkg2 := createAzureBatchApplicationPackage("2.0")

		mockClient := mocks.NewMockBatchApplicationPackagesClient(ctrl)
		pages := []armbatch.ApplicationPackageClientListResponse{
			{
				ListApplicationPackagesResult: armbatch.ListApplicationPackagesResult{
					Value: []*armbatch.ApplicationPackage{pkg1, pkg2},
				},
			},
		}
		mockPager := &mockBatchApplicationPackagesPager{pages: pages}
		testClient := &testBatchApplicationPackagesClient{
			MockBatchApplicationPackagesClient: mockClient,
			pager:                              mockPager,
		}

		wrapper := manual.NewBatchBatchApplicationPackage(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		searchStreamable, ok := adapter.(discovery.SearchStreamableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support SearchStream operation")
		}

		var items []*sdp.Item
		var errs []error
		stream := discovery.NewQueryResultStream(
			func(item *sdp.Item) { items = append(items, item) },
			func(err error) { errs = append(errs, err) },
		)

		searchStreamable.SearchStream(ctx, scope, shared.CompositeLookupKey(accountName, applicationName), true, stream)

		if len(errs) != 0 {
			t.Fatalf("Expected no errors, got: %v", errs)
		}

		if len(items) != 2 {
			t.Fatalf("Expected 2 items, got: %d", len(items))
		}
	})

	t.Run("Search_InvalidQueryParts", func(t *testing.T) {
		mockClient := mocks.NewMockBatchApplicationPackagesClient(ctrl)
		wrapper := manual.NewBatchBatchApplicationPackage(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		// No query parts
		_, qErr := wrapper.Search(ctx, scope)
		if qErr == nil {
			t.Error("Expected error when Search with no query parts, but got nil")
		}
	})

	t.Run("Search_PagerError", func(t *testing.T) {
		mockClient := mocks.NewMockBatchApplicationPackagesClient(ctrl)
		errorPager := &errorBatchApplicationPackagesPager{}
		testClient := &testBatchApplicationPackagesClient{
			MockBatchApplicationPackagesClient: mockClient,
			pager:                              errorPager,
		}

		wrapper := manual.NewBatchBatchApplicationPackage(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		_, qErr := wrapper.Search(ctx, scope, accountName, applicationName)
		if qErr == nil {
			t.Error("Expected error when pager returns error, but got nil")
		}
	})

	t.Run("Search_NilNameSkipped", func(t *testing.T) {
		validPkg := createAzureBatchApplicationPackage("1.0")
		nilNamePkg := &armbatch.ApplicationPackage{
			Name: nil,
		}

		mockClient := mocks.NewMockBatchApplicationPackagesClient(ctrl)
		pages := []armbatch.ApplicationPackageClientListResponse{
			{
				ListApplicationPackagesResult: armbatch.ListApplicationPackagesResult{
					Value: []*armbatch.ApplicationPackage{nilNamePkg, validPkg},
				},
			},
		}
		mockPager := &mockBatchApplicationPackagesPager{pages: pages}
		testClient := &testBatchApplicationPackagesClient{
			MockBatchApplicationPackagesClient: mockClient,
			pager:                              mockPager,
		}

		wrapper := manual.NewBatchBatchApplicationPackage(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		items, qErr := wrapper.Search(ctx, scope, accountName, applicationName)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if len(items) != 1 {
			t.Fatalf("Expected 1 item (nil-name skipped), got: %d", len(items))
		}
	})

	t.Run("PotentialLinks", func(t *testing.T) {
		mockClient := mocks.NewMockBatchApplicationPackagesClient(ctrl)
		wrapper := manual.NewBatchBatchApplicationPackage(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		links := wrapper.PotentialLinks()
		if !links[azureshared.BatchBatchApplication] {
			t.Error("PotentialLinks() should include BatchBatchApplication")
		}
		if !links[azureshared.BatchBatchAccount] {
			t.Error("PotentialLinks() should include BatchBatchAccount")
		}
		if !links[stdlib.NetworkDNS] {
			t.Error("PotentialLinks() should include stdlib.NetworkDNS")
		}
	})

	t.Run("HealthPending", func(t *testing.T) {
		pkg := createAzureBatchApplicationPackage(versionName)
		state := armbatch.PackageStatePending
		pkg.Properties.State = &state

		mockClient := mocks.NewMockBatchApplicationPackagesClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, accountName, applicationName, versionName).Return(
			armbatch.ApplicationPackageClientGetResponse{
				ApplicationPackage: *pkg,
			}, nil)

		wrapper := manual.NewBatchBatchApplicationPackage(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(accountName, applicationName, versionName)
		sdpItem, qErr := adapter.Get(ctx, scope, query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetHealth() != sdp.Health_HEALTH_PENDING {
			t.Errorf("Expected health PENDING for pending package, got %s", sdpItem.GetHealth())
		}
	})

	t.Run("GetWithoutStorageURL", func(t *testing.T) {
		pkg := createAzureBatchApplicationPackage(versionName)
		pkg.Properties.StorageURL = nil

		mockClient := mocks.NewMockBatchApplicationPackagesClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, accountName, applicationName, versionName).Return(
			armbatch.ApplicationPackageClientGetResponse{
				ApplicationPackage: *pkg,
			}, nil)

		wrapper := manual.NewBatchBatchApplicationPackage(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(accountName, applicationName, versionName)
		sdpItem, qErr := adapter.Get(ctx, scope, query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		// Should have 2 linked queries (application + account) but no DNS link
		linkedQueries := sdpItem.GetLinkedItemQueries()
		for _, liq := range linkedQueries {
			if liq.GetQuery().GetType() == stdlib.NetworkDNS.String() {
				t.Error("Expected no DNS linked query when StorageURL is nil")
			}
		}
	})

	t.Run("ImplementsSearchableAdapter", func(t *testing.T) {
		mockClient := mocks.NewMockBatchApplicationPackagesClient(ctrl)
		wrapper := manual.NewBatchBatchApplicationPackage(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Error("Adapter should implement SearchableAdapter interface")
		}
	})
}
