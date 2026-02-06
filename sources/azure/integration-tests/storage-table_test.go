package integrationtests

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources/v2"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage/v3"
	log "github.com/sirupsen/logrus"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	"github.com/overmindtech/cli/sources/azure/manual"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/shared"
)

const (
	integrationTestTableName = "ovm-integ-test-table"
)

func TestStorageTableIntegration(t *testing.T) {
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

	tableClient, err := armstorage.NewTableClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Table client: %v", err)
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

		// Create table
		err = createTable(ctx, tableClient, integrationTestResourceGroup, storageAccountName, integrationTestTableName)
		if err != nil {
			t.Fatalf("Failed to create table: %v", err)
		}
	})

	t.Run("Run", func(t *testing.T) {
		t.Run("GetTable", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Retrieving table %s in storage account %s, subscription %s, resource group %s",
				integrationTestTableName, storageAccountName, subscriptionID, integrationTestResourceGroup)

			tableWrapper := manual.NewStorageTable(
				clients.NewTablesClient(tableClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := tableWrapper.Scopes()[0]

			tableAdapter := sources.WrapperToAdapter(tableWrapper, sdpcache.NewNoOpCache())
			// Get requires storageAccountName and tableName as query parts
			query := shared.CompositeLookupKey(storageAccountName, integrationTestTableName)
			sdpItem, qErr := tableAdapter.Get(ctx, scope, query, true)
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

			expectedID := shared.CompositeLookupKey(storageAccountName, integrationTestTableName)
			if uniqueAttrValue != expectedID {
				t.Fatalf("Expected unique attribute value to be %s, got %s", expectedID, uniqueAttrValue)
			}

			log.Printf("Successfully retrieved table %s", integrationTestTableName)
		})

		t.Run("SearchTables", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Searching tables in storage account %s", storageAccountName)

			tableWrapper := manual.NewStorageTable(
				clients.NewTablesClient(tableClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := tableWrapper.Scopes()[0]

			tableAdapter := sources.WrapperToAdapter(tableWrapper, sdpcache.NewNoOpCache())

			// Check if adapter supports search
			searchable, ok := tableAdapter.(discovery.SearchableAdapter)
			if !ok {
				t.Fatalf("Adapter does not support Search operation")
			}

			sdpItems, err := searchable.Search(ctx, scope, storageAccountName, true)
			if err != nil {
				t.Fatalf("Failed to search tables: %v", err)
			}

			if len(sdpItems) < 1 {
				t.Fatalf("Expected at least one table, got %d", len(sdpItems))
			}

			var found bool
			for _, item := range sdpItems {
				uniqueAttrKey := item.GetUniqueAttribute()
				expectedID := shared.CompositeLookupKey(storageAccountName, integrationTestTableName)
				if v, err := item.GetAttributes().Get(uniqueAttrKey); err == nil && v == expectedID {
					found = true
					break
				}
			}

			if !found {
				t.Fatalf("Expected to find table %s in the search results", integrationTestTableName)
			}

			log.Printf("Found %d tables in search results", len(sdpItems))
		})

		t.Run("VerifyItemAttributes", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Verifying item attributes for table %s", integrationTestTableName)

			tableWrapper := manual.NewStorageTable(
				clients.NewTablesClient(tableClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := tableWrapper.Scopes()[0]

			tableAdapter := sources.WrapperToAdapter(tableWrapper, sdpcache.NewNoOpCache())
			query := shared.CompositeLookupKey(storageAccountName, integrationTestTableName)
			sdpItem, qErr := tableAdapter.Get(ctx, scope, query, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			// Verify item type
			if sdpItem.GetType() != azureshared.StorageTable.String() {
				t.Errorf("Expected item type %s, got %s", azureshared.StorageTable, sdpItem.GetType())
			}

			// Verify scope
			expectedScope := fmt.Sprintf("%s.%s", subscriptionID, integrationTestResourceGroup)
			if sdpItem.GetScope() != expectedScope {
				t.Errorf("Expected scope %s, got %s", expectedScope, sdpItem.GetScope())
			}

			// Verify unique attribute
			if sdpItem.GetUniqueAttribute() != "id" {
				t.Errorf("Expected unique attribute 'id', got %s", sdpItem.GetUniqueAttribute())
			}

			// Verify item validation
			if err := sdpItem.Validate(); err != nil {
				t.Fatalf("Item validation failed: %v", err)
			}

			log.Printf("Verified item attributes for table %s", integrationTestTableName)
		})

		t.Run("VerifyLinkedItems", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Verifying linked items for table %s", integrationTestTableName)

			tableWrapper := manual.NewStorageTable(
				clients.NewTablesClient(tableClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := tableWrapper.Scopes()[0]

			tableAdapter := sources.WrapperToAdapter(tableWrapper, sdpcache.NewNoOpCache())
			query := shared.CompositeLookupKey(storageAccountName, integrationTestTableName)
			sdpItem, qErr := tableAdapter.Get(ctx, scope, query, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			// Verify that linked items exist (storage account should be linked)
			linkedQueries := sdpItem.GetLinkedItemQueries()
			if len(linkedQueries) == 0 {
				t.Fatalf("Expected linked item queries, but got none")
			}

			var hasStorageAccountLink bool
			for _, liq := range linkedQueries {
				if liq.GetQuery().GetType() == azureshared.StorageAccount.String() {
					hasStorageAccountLink = true
					if liq.GetQuery().GetQuery() != storageAccountName {
						t.Errorf("Expected linked query to storage account %s, got %s", storageAccountName, liq.GetQuery().GetQuery())
					}
					if liq.GetBlastPropagation().GetIn() != true {
						t.Error("Expected BlastPropagation.In to be true")
					}
					if liq.GetBlastPropagation().GetOut() != false {
						t.Error("Expected BlastPropagation.Out to be false")
					}
					break
				}
			}

			if !hasStorageAccountLink {
				t.Error("Expected linked query to storage account, but didn't find one")
			}

			log.Printf("Verified %d linked item queries for table %s", len(linkedQueries), integrationTestTableName)
		})
	})

	t.Run("Teardown", func(t *testing.T) {
		ctx := t.Context()

		// Delete table
		err := deleteTable(ctx, tableClient, integrationTestResourceGroup, storageAccountName, integrationTestTableName)
		if err != nil {
			t.Fatalf("Failed to delete table: %v", err)
		}

		// Delete storage account
		err = deleteStorageAccount(ctx, saClient, integrationTestResourceGroup, storageAccountName)
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

// createTable creates an Azure storage table (idempotent)
func createTable(ctx context.Context, client *armstorage.TableClient, resourceGroupName, accountName, tableName string) error {
	// Check if table already exists
	_, err := client.Get(ctx, resourceGroupName, accountName, tableName, nil)
	if err == nil {
		log.Printf("Table %s already exists, skipping creation", tableName)
		return nil
	}

	// Create the table
	// Tables don't require any properties
	resp, err := client.Create(ctx, resourceGroupName, accountName, tableName, &armstorage.TableClientCreateOptions{
		Parameters: &armstorage.Table{
			TableProperties: &armstorage.TableProperties{},
		},
	})
	if err != nil {
		// Check if table already exists (conflict)
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusConflict {
			log.Printf("Table %s already exists (conflict), skipping creation", tableName)
			return nil
		}
		return fmt.Errorf("failed to create table: %w", err)
	}

	// Verify the table was created successfully
	if resp.ID == nil {
		return fmt.Errorf("table created but ID is unknown")
	}

	log.Printf("Table %s created successfully", tableName)
	return nil
}

// deleteTable deletes an Azure storage table
func deleteTable(ctx context.Context, client *armstorage.TableClient, resourceGroupName, accountName, tableName string) error {
	_, err := client.Delete(ctx, resourceGroupName, accountName, tableName, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			log.Printf("Table %s not found, skipping deletion", tableName)
			return nil
		}
		return fmt.Errorf("failed to delete table: %w", err)
	}

	log.Printf("Table %s deleted successfully", tableName)
	return nil
}
