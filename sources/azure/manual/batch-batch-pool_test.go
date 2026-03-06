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

type mockBatchPoolsPager struct {
	pages []armbatch.PoolClientListByBatchAccountResponse
	index int
}

func (m *mockBatchPoolsPager) More() bool {
	return m.index < len(m.pages)
}

func (m *mockBatchPoolsPager) NextPage(ctx context.Context) (armbatch.PoolClientListByBatchAccountResponse, error) {
	if m.index >= len(m.pages) {
		return armbatch.PoolClientListByBatchAccountResponse{}, errors.New("no more pages")
	}
	page := m.pages[m.index]
	m.index++
	return page, nil
}

type errorBatchPoolsPager struct{}

func (e *errorBatchPoolsPager) More() bool {
	return true
}

func (e *errorBatchPoolsPager) NextPage(ctx context.Context) (armbatch.PoolClientListByBatchAccountResponse, error) {
	return armbatch.PoolClientListByBatchAccountResponse{}, errors.New("pager error")
}

type testBatchPoolsClient struct {
	*mocks.MockBatchPoolsClient
	pager clients.BatchPoolsPager
}

func (t *testBatchPoolsClient) ListByBatchAccount(ctx context.Context, resourceGroupName, accountName string) clients.BatchPoolsPager {
	if t.pager != nil {
		return t.pager
	}
	return t.MockBatchPoolsClient.ListByBatchAccount(ctx, resourceGroupName, accountName)
}

func createAzureBatchPool(name string) *armbatch.Pool {
	state := armbatch.PoolProvisioningStateSucceeded
	return &armbatch.Pool{
		ID:   new("/subscriptions/sub/resourceGroups/rg/providers/Microsoft.Batch/batchAccounts/acc/pools/" + name),
		Name: new(name),
		Type: new("Microsoft.Batch/batchAccounts/pools"),
		Properties: &armbatch.PoolProperties{
			VMSize:            new("Standard_D2s_v3"),
			ProvisioningState: &state,
		},
		Tags: map[string]*string{"env": new("test")},
	}
}

func TestBatchBatchPool(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"
	scope := subscriptionID + "." + resourceGroup
	accountName := "test-batch-account"
	poolName := "test-pool"

	t.Run("Get", func(t *testing.T) {
		pool := createAzureBatchPool(poolName)

		mockClient := mocks.NewMockBatchPoolsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, accountName, poolName).Return(
			armbatch.PoolClientGetResponse{
				Pool: *pool,
			}, nil)

		wrapper := manual.NewBatchBatchPool(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(accountName, poolName)
		sdpItem, qErr := adapter.Get(ctx, scope, query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.BatchBatchPool.String() {
			t.Errorf("Expected type %s, got %s", azureshared.BatchBatchPool.String(), sdpItem.GetType())
		}

		if sdpItem.GetUniqueAttribute() != "uniqueAttr" {
			t.Errorf("Expected unique attribute 'uniqueAttr', got %s", sdpItem.GetUniqueAttribute())
		}

		expectedUnique := shared.CompositeLookupKey(accountName, poolName)
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
			}
			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("Get_InvalidQueryParts", func(t *testing.T) {
		mockClient := mocks.NewMockBatchPoolsClient(ctrl)
		wrapper := manual.NewBatchBatchPool(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, scope, accountName, true)
		if qErr == nil {
			t.Error("Expected error when Get with insufficient query parts, but got nil")
		}
	})

	t.Run("Get_ClientError", func(t *testing.T) {
		expectedErr := errors.New("pool not found")
		mockClient := mocks.NewMockBatchPoolsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, accountName, "nonexistent").Return(
			armbatch.PoolClientGetResponse{}, expectedErr)

		wrapper := manual.NewBatchBatchPool(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		query := shared.CompositeLookupKey(accountName, "nonexistent")
		_, qErr := adapter.Get(ctx, scope, query, true)
		if qErr == nil {
			t.Error("Expected error when client returns error, but got nil")
		}
	})

	t.Run("Search", func(t *testing.T) {
		pool1 := createAzureBatchPool("pool-1")
		pool2 := createAzureBatchPool("pool-2")

		mockClient := mocks.NewMockBatchPoolsClient(ctrl)
		pages := []armbatch.PoolClientListByBatchAccountResponse{
			{
				ListPoolsResult: armbatch.ListPoolsResult{
					Value: []*armbatch.Pool{pool1, pool2},
				},
			},
		}
		mockPager := &mockBatchPoolsPager{pages: pages}
		testClient := &testBatchPoolsClient{
			MockBatchPoolsClient: mockClient,
			pager:                mockPager,
		}

		wrapper := manual.NewBatchBatchPool(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
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
		mockClient := mocks.NewMockBatchPoolsClient(ctrl)
		wrapper := manual.NewBatchBatchPool(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		_, qErr := wrapper.Search(ctx, scope)
		if qErr == nil {
			t.Error("Expected error when Search with no query parts, but got nil")
		}
	})

	t.Run("Search_PagerError", func(t *testing.T) {
		mockClient := mocks.NewMockBatchPoolsClient(ctrl)
		errorPager := &errorBatchPoolsPager{}
		testClient := &testBatchPoolsClient{
			MockBatchPoolsClient: mockClient,
			pager:               errorPager,
		}

		wrapper := manual.NewBatchBatchPool(testClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		_, qErr := wrapper.Search(ctx, scope, accountName)
		if qErr == nil {
			t.Error("Expected error when pager returns error, but got nil")
		}
	})

	t.Run("PotentialLinks", func(t *testing.T) {
		mockClient := mocks.NewMockBatchPoolsClient(ctrl)
		wrapper := manual.NewBatchBatchPool(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})

		links := wrapper.PotentialLinks()
		if !links[azureshared.BatchBatchAccount] {
			t.Error("PotentialLinks() should include BatchBatchAccount")
		}
		if !links[azureshared.NetworkSubnet] {
			t.Error("PotentialLinks() should include NetworkSubnet")
		}
		if !links[azureshared.ManagedIdentityUserAssignedIdentity] {
			t.Error("PotentialLinks() should include ManagedIdentityUserAssignedIdentity")
		}
		if !links[azureshared.BatchBatchApplicationPackage] {
			t.Error("PotentialLinks() should include BatchBatchApplicationPackage")
		}
		if !links[azureshared.NetworkPublicIPAddress] {
			t.Error("PotentialLinks() should include NetworkPublicIPAddress")
		}
		if !links[azureshared.StorageAccount] {
			t.Error("PotentialLinks() should include StorageAccount")
		}
		if !links[stdlib.NetworkIP] {
			t.Error("PotentialLinks() should include stdlib.NetworkIP")
		}
		if !links[stdlib.NetworkDNS] {
			t.Error("PotentialLinks() should include stdlib.NetworkDNS")
		}
		if !links[stdlib.NetworkHTTP] {
			t.Error("PotentialLinks() should include stdlib.NetworkHTTP")
		}
	})

	t.Run("ImplementsSearchableAdapter", func(t *testing.T) {
		mockClient := mocks.NewMockBatchPoolsClient(ctrl)
		wrapper := manual.NewBatchBatchPool(mockClient, []azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Error("Adapter should implement SearchableAdapter interface")
		}
	})
}
