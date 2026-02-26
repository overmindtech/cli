package manual_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage/v3"
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

type mockEncryptionScopesPager struct {
	pages []armstorage.EncryptionScopesClientListResponse
	index int
}

func (m *mockEncryptionScopesPager) More() bool {
	return m.index < len(m.pages)
}

func (m *mockEncryptionScopesPager) NextPage(ctx context.Context) (armstorage.EncryptionScopesClientListResponse, error) {
	if m.index >= len(m.pages) {
		return armstorage.EncryptionScopesClientListResponse{}, errors.New("no more pages")
	}
	page := m.pages[m.index]
	m.index++
	return page, nil
}

type errorEncryptionScopesPager struct{}

func (e *errorEncryptionScopesPager) More() bool {
	return true
}

func (e *errorEncryptionScopesPager) NextPage(ctx context.Context) (armstorage.EncryptionScopesClientListResponse, error) {
	return armstorage.EncryptionScopesClientListResponse{}, errors.New("pager error")
}

type testEncryptionScopesClient struct {
	*mocks.MockEncryptionScopesClient
	pager clients.EncryptionScopesPager
}

func (t *testEncryptionScopesClient) List(ctx context.Context, resourceGroupName, accountName string) clients.EncryptionScopesPager {
	return t.pager
}

func TestStorageEncryptionScope(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"
	storageAccountName := "teststorageaccount"
	encryptionScopeName := "test-encryption-scope"

	t.Run("Get", func(t *testing.T) {
		encScope := createAzureEncryptionScope(encryptionScopeName)

		mockClient := mocks.NewMockEncryptionScopesClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, storageAccountName, encryptionScopeName).Return(
			armstorage.EncryptionScopesClientGetResponse{
				EncryptionScope: *encScope,
			}, nil)

		testClient := &testEncryptionScopesClient{MockEncryptionScopesClient: mockClient}
		wrapper := manual.NewStorageEncryptionScope(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(storageAccountName, encryptionScopeName)
		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.StorageEncryptionScope.String() {
			t.Errorf("Expected type %s, got %s", azureshared.StorageEncryptionScope.String(), sdpItem.GetType())
		}

		if sdpItem.GetUniqueAttribute() != "uniqueAttr" {
			t.Errorf("Expected unique attribute 'uniqueAttr', got %s", sdpItem.GetUniqueAttribute())
		}

		if sdpItem.UniqueAttributeValue() != shared.CompositeLookupKey(storageAccountName, encryptionScopeName) {
			t.Errorf("Expected unique attribute value %s, got %s", shared.CompositeLookupKey(storageAccountName, encryptionScopeName), sdpItem.UniqueAttributeValue())
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

			linkedQuery := linkedQueries[0]
			if linkedQuery.GetQuery().GetType() != azureshared.StorageAccount.String() {
				t.Errorf("Expected linked query type %s, got %s", azureshared.StorageAccount.String(), linkedQuery.GetQuery().GetType())
			}
			if linkedQuery.GetQuery().GetMethod() != sdp.QueryMethod_GET {
				t.Errorf("Expected linked query method GET, got %s", linkedQuery.GetQuery().GetMethod())
			}
			if linkedQuery.GetQuery().GetQuery() != storageAccountName {
				t.Errorf("Expected linked query %s, got %s", storageAccountName, linkedQuery.GetQuery().GetQuery())
			}
		})
	})

	t.Run("Get_InvalidQueryParts", func(t *testing.T) {
		mockClient := mocks.NewMockEncryptionScopesClient(ctrl)
		testClient := &testEncryptionScopesClient{MockEncryptionScopesClient: mockClient}

		wrapper := manual.NewStorageEncryptionScope(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], storageAccountName, true)
		if qErr == nil {
			t.Error("Expected error when providing insufficient query parts, but got nil")
		}
	})

	t.Run("Search", func(t *testing.T) {
		scope1 := createAzureEncryptionScope("scope-1")
		scope2 := createAzureEncryptionScope("scope-2")

		mockClient := mocks.NewMockEncryptionScopesClient(ctrl)
		mockPager := &mockEncryptionScopesPager{
			pages: []armstorage.EncryptionScopesClientListResponse{
				{
					EncryptionScopeListResult: armstorage.EncryptionScopeListResult{
						Value: []*armstorage.EncryptionScope{scope1, scope2},
					},
				},
			},
		}

		testClient := &testEncryptionScopesClient{
			MockEncryptionScopesClient: mockClient,
			pager:                     mockPager,
		}

		wrapper := manual.NewStorageEncryptionScope(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support Search operation")
		}

		sdpItems, err := searchable.Search(ctx, wrapper.Scopes()[0], storageAccountName, true)
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
			if item.GetType() != azureshared.StorageEncryptionScope.String() {
				t.Errorf("Expected type %s, got %s", azureshared.StorageEncryptionScope.String(), item.GetType())
			}
		}
	})

	t.Run("Search_InvalidQueryParts", func(t *testing.T) {
		mockClient := mocks.NewMockEncryptionScopesClient(ctrl)
		testClient := &testEncryptionScopesClient{MockEncryptionScopesClient: mockClient}

		wrapper := manual.NewStorageEncryptionScope(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		_, qErr := wrapper.Search(ctx, wrapper.Scopes()[0])
		if qErr == nil {
			t.Error("Expected error when providing no query parts, but got nil")
		}
	})

	t.Run("Search_ScopeWithNilName", func(t *testing.T) {
		mockClient := mocks.NewMockEncryptionScopesClient(ctrl)
		validScope := createAzureEncryptionScope("valid-scope")
		mockPager := &mockEncryptionScopesPager{
			pages: []armstorage.EncryptionScopesClientListResponse{
				{
					EncryptionScopeListResult: armstorage.EncryptionScopeListResult{
						Value: []*armstorage.EncryptionScope{
							{Name: nil},
							validScope,
						},
					},
				},
			},
		}

		testClient := &testEncryptionScopesClient{
			MockEncryptionScopesClient: mockClient,
			pager:                     mockPager,
		}

		wrapper := manual.NewStorageEncryptionScope(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support Search operation")
		}

		sdpItems, err := searchable.Search(ctx, wrapper.Scopes()[0], storageAccountName, true)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if len(sdpItems) != 1 {
			t.Fatalf("Expected 1 item, got: %d", len(sdpItems))
		}

		if sdpItems[0].UniqueAttributeValue() != shared.CompositeLookupKey(storageAccountName, "valid-scope") {
			t.Errorf("Expected unique value %s, got %s", shared.CompositeLookupKey(storageAccountName, "valid-scope"), sdpItems[0].UniqueAttributeValue())
		}
	})

	t.Run("ErrorHandling_Get", func(t *testing.T) {
		expectedErr := errors.New("encryption scope not found")

		mockClient := mocks.NewMockEncryptionScopesClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, storageAccountName, "nonexistent-scope").Return(
			armstorage.EncryptionScopesClientGetResponse{}, expectedErr)

		testClient := &testEncryptionScopesClient{MockEncryptionScopesClient: mockClient}
		wrapper := manual.NewStorageEncryptionScope(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := storageAccountName + shared.QuerySeparator + "nonexistent-scope"
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr == nil {
			t.Error("Expected error when getting non-existent encryption scope, but got nil")
		}
	})

	t.Run("ErrorHandling_Search", func(t *testing.T) {
		mockClient := mocks.NewMockEncryptionScopesClient(ctrl)
		errorPager := &errorEncryptionScopesPager{}

		testClient := &testEncryptionScopesClient{
			MockEncryptionScopesClient: mockClient,
			pager:                     errorPager,
		}

		wrapper := manual.NewStorageEncryptionScope(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support Search operation")
		}

		_, err := searchable.Search(ctx, wrapper.Scopes()[0], storageAccountName, true)
		if err == nil {
			t.Error("Expected error from pager when NextPage returns an error, but got nil")
		}
	})
}

func createAzureEncryptionScope(scopeName string) *armstorage.EncryptionScope {
	return &armstorage.EncryptionScope{
		ID:   to.Ptr("/subscriptions/test-subscription/resourceGroups/test-rg/providers/Microsoft.Storage/storageAccounts/teststorageaccount/encryptionScopes/" + scopeName),
		Name: to.Ptr(scopeName),
		Type: to.Ptr("Microsoft.Storage/storageAccounts/encryptionScopes"),
		EncryptionScopeProperties: &armstorage.EncryptionScopeProperties{
			Source: to.Ptr(armstorage.EncryptionScopeSourceMicrosoftStorage),
			State:  to.Ptr(armstorage.EncryptionScopeStateEnabled),
		},
	}
}
