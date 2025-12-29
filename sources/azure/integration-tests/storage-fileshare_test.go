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
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage/v2"
	log "github.com/sirupsen/logrus"
	"k8s.io/utils/ptr"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	"github.com/overmindtech/cli/sources/azure/manual"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/shared"
)

const (
	integrationTestShareName = "ovm-integ-test-share"
)

func TestStorageFileShareIntegration(t *testing.T) {
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

	fsClient, err := armstorage.NewFileSharesClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create File Shares client: %v", err)
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

		// Create file share
		err = createFileShare(ctx, fsClient, integrationTestResourceGroup, storageAccountName, integrationTestShareName)
		if err != nil {
			t.Fatalf("Failed to create file share: %v", err)
		}
	})

	t.Run("Run", func(t *testing.T) {
		t.Run("GetFileShare", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Retrieving file share %s in storage account %s, subscription %s, resource group %s",
				integrationTestShareName, storageAccountName, subscriptionID, integrationTestResourceGroup)

			fsWrapper := manual.NewStorageFileShare(
				clients.NewFileSharesClient(fsClient),
				subscriptionID,
				integrationTestResourceGroup,
			)
			scope := fsWrapper.Scopes()[0]

			fsAdapter := sources.WrapperToAdapter(fsWrapper)
			// Get requires storageAccountName and shareName as query parts
			query := storageAccountName + shared.QuerySeparator + integrationTestShareName
			sdpItem, qErr := fsAdapter.Get(ctx, scope, query, true)
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

			if uniqueAttrValue != shared.CompositeLookupKey(storageAccountName, integrationTestShareName) {
				t.Errorf("Expected unique attribute value %s, got %s", shared.CompositeLookupKey(storageAccountName, integrationTestShareName), uniqueAttrValue)
			}

			log.Printf("Successfully retrieved file share %s", integrationTestShareName)
		})

		t.Run("SearchFileShares", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Searching file shares in storage account %s", storageAccountName)

			fsWrapper := manual.NewStorageFileShare(
				clients.NewFileSharesClient(fsClient),
				subscriptionID,
				integrationTestResourceGroup,
			)
			scope := fsWrapper.Scopes()[0]

			fsAdapter := sources.WrapperToAdapter(fsWrapper)

			// Check if adapter supports search
			searchable, ok := fsAdapter.(discovery.SearchableAdapter)
			if !ok {
				t.Fatalf("Adapter does not support Search operation")
			}

			sdpItems, err := searchable.Search(ctx, scope, storageAccountName, true)
			if err != nil {
				t.Fatalf("Failed to search file shares: %v", err)
			}

			if len(sdpItems) < 1 {
				t.Fatalf("Expected at least one file share, got %d", len(sdpItems))
			}

			var found bool
			for _, item := range sdpItems {
				uniqueAttrKey := item.GetUniqueAttribute()
				if v, err := item.GetAttributes().Get(uniqueAttrKey); err == nil && v == shared.CompositeLookupKey(storageAccountName, integrationTestShareName) {
					found = true
					break
				}
			}

			if !found {
				t.Fatalf("Expected to find share %s in the search results", integrationTestShareName)
			}

			log.Printf("Found %d file shares in search results", len(sdpItems))
		})

		t.Run("VerifyLinkedItems", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Verifying linked items for file share %s", integrationTestShareName)

			fsWrapper := manual.NewStorageFileShare(
				clients.NewFileSharesClient(fsClient),
				subscriptionID,
				integrationTestResourceGroup,
			)
			scope := fsWrapper.Scopes()[0]

			fsAdapter := sources.WrapperToAdapter(fsWrapper)
			query := storageAccountName + shared.QuerySeparator + integrationTestShareName
			sdpItem, qErr := fsAdapter.Get(ctx, scope, query, true)
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
					break
				}
			}

			if !hasStorageAccountLink {
				t.Error("Expected linked query to storage account, but didn't find one")
			}

			log.Printf("Verified %d linked item queries for file share %s", len(linkedQueries), integrationTestShareName)
		})
	})

	t.Run("Teardown", func(t *testing.T) {
		ctx := t.Context()

		// Delete file share
		err := deleteFileShare(ctx, fsClient, integrationTestResourceGroup, storageAccountName, integrationTestShareName)
		if err != nil {
			t.Fatalf("Failed to delete file share: %v", err)
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

// createFileShare creates an Azure file share (idempotent)
func createFileShare(ctx context.Context, client *armstorage.FileSharesClient, resourceGroupName, accountName, shareName string) error {
	// Check if file share already exists
	_, err := client.Get(ctx, resourceGroupName, accountName, shareName, nil)
	if err == nil {
		log.Printf("File share %s already exists, skipping creation", shareName)
		return nil
	}

	// Create the file share
	// File shares require a quota (size in GB)
	resp, err := client.Create(ctx, resourceGroupName, accountName, shareName, armstorage.FileShare{
		FileShareProperties: &armstorage.FileShareProperties{
			ShareQuota: ptr.To(int32(1)), // 1GB minimum quota
		},
	}, nil)
	if err != nil {
		// Check if file share already exists (conflict)
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusConflict {
			log.Printf("File share %s already exists (conflict), skipping creation", shareName)
			return nil
		}
		return fmt.Errorf("failed to create file share: %w", err)
	}

	// Verify the file share was created successfully
	if resp.ID == nil {
		return fmt.Errorf("file share created but ID is unknown")
	}

	log.Printf("File share %s created successfully", shareName)
	return nil
}

// deleteFileShare deletes an Azure file share
func deleteFileShare(ctx context.Context, client *armstorage.FileSharesClient, resourceGroupName, accountName, shareName string) error {
	_, err := client.Delete(ctx, resourceGroupName, accountName, shareName, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			log.Printf("File share %s not found, skipping deletion", shareName)
			return nil
		}
		return fmt.Errorf("failed to delete file share: %w", err)
	}

	log.Printf("File share %s deleted successfully", shareName)
	return nil
}
