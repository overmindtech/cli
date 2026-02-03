package integrationtests

import (
	"os"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources/v2"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage/v3"
	log "github.com/sirupsen/logrus"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	"github.com/overmindtech/cli/sources/azure/manual"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
)

// Note: integrationTestSAName is already declared in storage-blob-container_test.go
// Reusing it here since both tests are in the same package

func TestStorageAccountIntegration(t *testing.T) {
	subscriptionID := os.Getenv("AZURE_SUBSCRIPTION_ID")
	if subscriptionID == "" {
		t.Skip("AZURE_SUBSCRIPTION_ID environment variable not set")
	}

	// Initialize Azure credentials using DefaultAzureCredential
	cred, err := azureshared.NewAzureCredential(t.Context())
	if err != nil {
		t.Fatalf("Failed to create Azure credential: %v", err)
	}

	// Create Azure SDK clients
	saClient, err := armstorage.NewAccountsClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Storage Accounts client: %v", err)
	}

	rgClient, err := armresources.NewResourceGroupsClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Resource Groups client: %v", err)
	}

	// Generate unique storage account name (must be globally unique, lowercase, 3-24 chars)
	storageAccountName := generateStorageAccountName(integrationTestSAName)

	t.Run("Setup", func(t *testing.T) {
		ctx := t.Context()

		// Create resource group if it doesn't exist
		err := createResourceGroup(ctx, rgClient, integrationTestResourceGroup, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create resource group: %v", err)
		}

		// Create storage account
		err = createStorageAccount(ctx, saClient, integrationTestResourceGroup, storageAccountName, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create storage account: %v", err)
		}

		// Wait for storage account to be fully available
		err = waitForStorageAccountAvailable(ctx, saClient, integrationTestResourceGroup, storageAccountName)
		if err != nil {
			t.Fatalf("Failed waiting for storage account to be available: %v", err)
		}
	})

	t.Run("Run", func(t *testing.T) {
		t.Run("GetStorageAccount", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Retrieving storage account %s, subscription %s, resource group %s",
				storageAccountName, subscriptionID, integrationTestResourceGroup)

			saWrapper := manual.NewStorageAccount(
				clients.NewStorageAccountsClient(saClient),
				subscriptionID,
				integrationTestResourceGroup,
			)
			scope := saWrapper.Scopes()[0]

			saAdapter := sources.WrapperToAdapter(saWrapper, sdpcache.NewNoOpCache())
			sdpItem, qErr := saAdapter.Get(ctx, scope, storageAccountName, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			if sdpItem == nil {
				t.Fatalf("Expected sdpItem to be non-nil")
			}

			uniqueAttrKey := sdpItem.GetUniqueAttribute()
			uniqueAttrValue, err := sdpItem.GetAttributes().Get(uniqueAttrKey)
			if err != nil {
				t.Fatalf("Failed to get unique attribute: %v", err)
			}

			if uniqueAttrValue != storageAccountName {
				t.Fatalf("Expected unique attribute value to be %s, got %s", storageAccountName, uniqueAttrValue)
			}

			if sdpItem.GetType() != azureshared.StorageAccount.String() {
				t.Fatalf("Expected type %s, got %s", azureshared.StorageAccount, sdpItem.GetType())
			}

			log.Printf("Successfully retrieved storage account %s", storageAccountName)
		})

		t.Run("ListStorageAccounts", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Listing storage accounts in resource group %s", integrationTestResourceGroup)

			saWrapper := manual.NewStorageAccount(
				clients.NewStorageAccountsClient(saClient),
				subscriptionID,
				integrationTestResourceGroup,
			)
			scope := saWrapper.Scopes()[0]

			saAdapter := sources.WrapperToAdapter(saWrapper, sdpcache.NewNoOpCache())

			// Check if adapter supports list
			listable, ok := saAdapter.(discovery.ListableAdapter)
			if !ok {
				t.Fatalf("Adapter does not support List operation")
			}

			sdpItems, err := listable.List(ctx, scope, true)
			if err != nil {
				t.Fatalf("Failed to list storage accounts: %v", err)
			}

			if len(sdpItems) < 1 {
				t.Fatalf("Expected at least one storage account, got %d", len(sdpItems))
			}

			var found bool
			for _, item := range sdpItems {
				uniqueAttrKey := item.GetUniqueAttribute()
				if v, err := item.GetAttributes().Get(uniqueAttrKey); err == nil && v == storageAccountName {
					found = true
					if item.GetType() != azureshared.StorageAccount.String() {
						t.Errorf("Expected type %s, got %s", azureshared.StorageAccount, item.GetType())
					}
					break
				}
			}

			if !found {
				t.Fatalf("Expected to find storage account %s in the list results", storageAccountName)
			}

			log.Printf("Found %d storage accounts in list results", len(sdpItems))
		})

		t.Run("VerifyLinkedItems", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Verifying linked items for storage account %s", storageAccountName)

			saWrapper := manual.NewStorageAccount(
				clients.NewStorageAccountsClient(saClient),
				subscriptionID,
				integrationTestResourceGroup,
			)
			scope := saWrapper.Scopes()[0]

			saAdapter := sources.WrapperToAdapter(saWrapper, sdpcache.NewNoOpCache())
			sdpItem, qErr := saAdapter.Get(ctx, scope, storageAccountName, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			// Verify that linked items exist
			linkedQueries := sdpItem.GetLinkedItemQueries()
			if len(linkedQueries) == 0 {
				t.Fatalf("Expected linked item queries, but got none")
			}

			// Verify expected linked item types
			expectedLinkedTypes := map[string]bool{
				azureshared.StorageBlobContainer.String(): false,
				azureshared.StorageFileShare.String():     false,
				azureshared.StorageTable.String():         false,
				azureshared.StorageQueue.String():         false,
			}

			for _, liq := range linkedQueries {
				linkedType := liq.GetQuery().GetType()
				if _, exists := expectedLinkedTypes[linkedType]; exists {
					expectedLinkedTypes[linkedType] = true

					// Verify the query uses the storage account name
					if liq.GetQuery().GetQuery() != storageAccountName {
						t.Errorf("Expected linked query to use storage account name %s, got %s", storageAccountName, liq.GetQuery().GetQuery())
					}

					// Verify the query method is SEARCH (since we're linking to child resources)
					if liq.GetQuery().GetMethod() != sdp.QueryMethod_SEARCH {
						t.Errorf("Expected linked query method to be SEARCH, got %s", liq.GetQuery().GetMethod())
					}

					// Verify blast propagation (parent to child: In=false, Out=true)
					if liq.GetBlastPropagation() == nil {
						t.Errorf("Expected blast propagation to be set for linked type %s", linkedType)
					} else {
						if liq.GetBlastPropagation().GetIn() != false {
							t.Errorf("Expected blast propagation In=false for linked type %s, got true", linkedType)
						}
						if liq.GetBlastPropagation().GetOut() != true {
							t.Errorf("Expected blast propagation Out=true for linked type %s, got false", linkedType)
						}
					}
				}
			}

			// Verify all expected linked types were found
			for linkedType, found := range expectedLinkedTypes {
				if !found {
					t.Errorf("Expected linked query to %s, but didn't find one", linkedType)
				}
			}

			log.Printf("Verified %d linked item queries for storage account %s", len(linkedQueries), storageAccountName)
		})
	})

	t.Run("Teardown", func(t *testing.T) {
		ctx := t.Context()

		// Delete storage account
		err := deleteStorageAccount(ctx, saClient, integrationTestResourceGroup, storageAccountName)
		if err != nil {
			t.Fatalf("Failed to delete storage account: %v", err)
		}

		// Optionally delete the resource group
		// Note: We keep the resource group for faster subsequent test runs
		// Uncomment the following if you want to clean up completely:
		// err = deleteResourceGroup(ctx, rgClient, integrationTestResourceGroup)
		// if err != nil {
		//     t.Fatalf("Failed to delete resource group: %v", err)
		// }
	})
}
