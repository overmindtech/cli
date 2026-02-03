package manual_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage/v3"
	"go.uber.org/mock/gomock"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/manual"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/azure/shared/mocks"
	"github.com/overmindtech/cli/sources/shared"
)

func TestStorageAccount(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	subscriptionID := "test-subscription"
	resourceGroup := "test-rg"

	t.Run("Get", func(t *testing.T) {
		accountName := "teststorageaccount"
		account := createAzureStorageAccount(accountName, "Succeeded")

		mockClient := mocks.NewMockStorageAccountsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, accountName).Return(
			armstorage.AccountsClientGetPropertiesResponse{
				Account: *account,
			}, nil)

		wrapper := manual.NewStorageAccount(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], accountName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != azureshared.StorageAccount.String() {
			t.Errorf("Expected type %s, got %s", azureshared.StorageAccount, sdpItem.GetType())
		}

		if sdpItem.GetUniqueAttribute() != "name" {
			t.Errorf("Expected unique attribute 'name', got %s", sdpItem.GetUniqueAttribute())
		}

		if sdpItem.UniqueAttributeValue() != accountName {
			t.Errorf("Expected unique attribute value %s, got %s", accountName, sdpItem.UniqueAttributeValue())
		}

		if sdpItem.GetTags()["env"] != "test" {
			t.Errorf("Expected tag 'env=test', got: %v", sdpItem.GetTags()["env"])
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					// Storage blob container link
					ExpectedType:   azureshared.StorageBlobContainer.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  accountName,
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  false,
						Out: true,
					},
				},
				{
					// Storage file share link
					ExpectedType:   azureshared.StorageFileShare.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  accountName,
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  false,
						Out: true,
					},
				},
				{
					// Storage table link
					ExpectedType:   azureshared.StorageTable.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  accountName,
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  false,
						Out: true,
					},
				},
				{
					// Storage queue link
					ExpectedType:   azureshared.StorageQueue.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  accountName,
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  false,
						Out: true,
					},
				},
				{
					// Storage private endpoint connection link (child resource)
					ExpectedType:   azureshared.StoragePrivateEndpointConnection.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  accountName,
					ExpectedScope:  subscriptionID + "." + resourceGroup,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					// DNS link from PrimaryEndpoints.Blob
					ExpectedType:   "dns",
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  accountName + ".blob.core.windows.net",
					ExpectedScope:  "global",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					// DNS link from PrimaryEndpoints.Queue
					ExpectedType:   "dns",
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  accountName + ".queue.core.windows.net",
					ExpectedScope:  "global",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					// DNS link from PrimaryEndpoints.Table
					ExpectedType:   "dns",
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  accountName + ".table.core.windows.net",
					ExpectedScope:  "global",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					// DNS link from PrimaryEndpoints.File
					ExpectedType:   "dns",
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  accountName + ".file.core.windows.net",
					ExpectedScope:  "global",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
			}

			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("Get_InvalidQueryParts", func(t *testing.T) {
		mockClient := mocks.NewMockStorageAccountsClient(ctrl)

		wrapper := manual.NewStorageAccount(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		// Test with insufficient query parts (empty)
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "", true)
		if qErr == nil {
			t.Error("Expected error when getting storage account with empty name, but got nil")
		}
	})

	t.Run("List", func(t *testing.T) {
		account1 := createAzureStorageAccount("teststorageaccount1", "Succeeded")
		account2 := createAzureStorageAccount("teststorageaccount2", "Succeeded")

		mockClient := mocks.NewMockStorageAccountsClient(ctrl)
		mockPager := mocks.NewMockStorageAccountsPager(ctrl)

		// Setup pager expectations
		gomock.InOrder(
			mockPager.EXPECT().More().Return(true),
			mockPager.EXPECT().NextPage(ctx).Return(
				armstorage.AccountsClientListByResourceGroupResponse{
					AccountListResult: armstorage.AccountListResult{
						Value: []*armstorage.Account{account1, account2},
					},
				}, nil),
			mockPager.EXPECT().More().Return(false),
		)

		mockClient.EXPECT().List(resourceGroup).Return(mockPager)

		wrapper := manual.NewStorageAccount(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		listable, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support List operation")
		}

		sdpItems, err := listable.List(ctx, wrapper.Scopes()[0], true)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if len(sdpItems) != 2 {
			t.Fatalf("Expected 2 items, got: %d", len(sdpItems))
		}

		for _, item := range sdpItems {
			if item.Validate() != nil {
				t.Fatalf("Expected no validation error, got: %v", item.Validate())
			}

			if item.GetTags()["env"] != "test" {
				t.Fatalf("Expected tag 'env=test', got: %s", item.GetTags()["env"])
			}

			if item.GetType() != azureshared.StorageAccount.String() {
				t.Fatalf("Expected type %s, got: %s", azureshared.StorageAccount, item.GetType())
			}
		}
	})

	t.Run("List_WithNilName", func(t *testing.T) {
		// Create account with nil name to test filtering
		account1 := createAzureStorageAccount("teststorageaccount1", "Succeeded")
		account2 := &armstorage.Account{
			Name:     nil, // Account with nil name should be skipped
			Location: to.Ptr("eastus"),
			Tags: map[string]*string{
				"env": to.Ptr("test"),
			},
			Properties: &armstorage.AccountProperties{
				ProvisioningState: to.Ptr(armstorage.ProvisioningStateSucceeded),
			},
		}

		mockClient := mocks.NewMockStorageAccountsClient(ctrl)
		mockPager := mocks.NewMockStorageAccountsPager(ctrl)

		// Setup pager expectations
		gomock.InOrder(
			mockPager.EXPECT().More().Return(true),
			mockPager.EXPECT().NextPage(ctx).Return(
				armstorage.AccountsClientListByResourceGroupResponse{
					AccountListResult: armstorage.AccountListResult{
						Value: []*armstorage.Account{account1, account2},
					},
				}, nil),
			mockPager.EXPECT().More().Return(false),
		)

		mockClient.EXPECT().List(resourceGroup).Return(mockPager)

		wrapper := manual.NewStorageAccount(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		listable, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support List operation")
		}

		sdpItems, err := listable.List(ctx, wrapper.Scopes()[0], true)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		// Should only return 1 item (account with nil name is skipped)
		if len(sdpItems) != 1 {
			t.Fatalf("Expected 1 item (nil name filtered out), got: %d", len(sdpItems))
		}

		if sdpItems[0].UniqueAttributeValue() != "teststorageaccount1" {
			t.Fatalf("Expected account name 'teststorageaccount1', got: %s", sdpItems[0].UniqueAttributeValue())
		}
	})

	// Note: ListStream test is not included as ListStream is not yet implemented
	// in the storage account adapter. When ListStream is implemented, add a test
	// following the pattern from compute-virtual-machine_test.go

	t.Run("ErrorHandling", func(t *testing.T) {
		expectedErr := errors.New("storage account not found")

		mockClient := mocks.NewMockStorageAccountsClient(ctrl)
		mockClient.EXPECT().Get(ctx, resourceGroup, "nonexistent-account").Return(
			armstorage.AccountsClientGetPropertiesResponse{}, expectedErr)

		wrapper := manual.NewStorageAccount(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "nonexistent-account", true)
		if qErr == nil {
			t.Error("Expected error when getting non-existent storage account, but got nil")
		}
	})

	t.Run("ErrorHandling_List", func(t *testing.T) {
		expectedErr := errors.New("failed to list storage accounts")

		mockClient := mocks.NewMockStorageAccountsClient(ctrl)
		mockPager := mocks.NewMockStorageAccountsPager(ctrl)

		// Setup pager to return error on NextPage
		gomock.InOrder(
			mockPager.EXPECT().More().Return(true),
			mockPager.EXPECT().NextPage(ctx).Return(
				armstorage.AccountsClientListByResourceGroupResponse{}, expectedErr),
		)

		mockClient.EXPECT().List(resourceGroup).Return(mockPager)

		wrapper := manual.NewStorageAccount(mockClient, subscriptionID, resourceGroup)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		listable, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support List operation")
		}

		_, err := listable.List(ctx, wrapper.Scopes()[0], true)
		if err == nil {
			t.Error("Expected error when listing storage accounts fails, but got nil")
		}
	})
}

// createAzureStorageAccount creates a mock Azure storage account for testing
func createAzureStorageAccount(accountName, provisioningState string) *armstorage.Account {
	state := armstorage.ProvisioningState(provisioningState)
	return &armstorage.Account{
		Name:     to.Ptr(accountName),
		Location: to.Ptr("eastus"),
		Kind:     to.Ptr(armstorage.KindStorageV2),
		Tags: map[string]*string{
			"env":     to.Ptr("test"),
			"project": to.Ptr("testing"),
		},
		Properties: &armstorage.AccountProperties{
			ProvisioningState: &state,
			PrimaryEndpoints: &armstorage.Endpoints{
				Blob:  to.Ptr("https://" + accountName + ".blob.core.windows.net/"),
				Queue: to.Ptr("https://" + accountName + ".queue.core.windows.net/"),
				Table: to.Ptr("https://" + accountName + ".table.core.windows.net/"),
				File:  to.Ptr("https://" + accountName + ".file.core.windows.net/"),
			},
		},
	}
}
