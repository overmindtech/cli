package integrationtests

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources/v2"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage/v3"
	log "github.com/sirupsen/logrus"
	"k8s.io/utils/ptr"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	"github.com/overmindtech/cli/sources/azure/manual"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/shared"
)

const (
	integrationTestSAName        = "ovm-integ-test-sa"
	integrationTestContainerName = "ovm-integ-test-container"
)

func TestStorageBlobContainerIntegration(t *testing.T) {
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

	bcClient, err := armstorage.NewBlobContainersClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Blob Containers client: %v", err)
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

		// Create blob container
		err = createBlobContainer(ctx, bcClient, integrationTestResourceGroup, storageAccountName, integrationTestContainerName)
		if err != nil {
			t.Fatalf("Failed to create blob container: %v", err)
		}
	})

	t.Run("Run", func(t *testing.T) {
		t.Run("GetBlobContainer", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Retrieving blob container %s in storage account %s, subscription %s, resource group %s",
				integrationTestContainerName, storageAccountName, subscriptionID, integrationTestResourceGroup)

			bcWrapper := manual.NewStorageBlobContainer(
				clients.NewBlobContainersClient(bcClient),
				subscriptionID,
				integrationTestResourceGroup,
			)
			scope := bcWrapper.Scopes()[0]

			bcAdapter := sources.WrapperToAdapter(bcWrapper, sdpcache.NewNoOpCache())
			// Get requires storageAccountName and containerName as query parts
			query := storageAccountName + shared.QuerySeparator + integrationTestContainerName
			sdpItem, qErr := bcAdapter.Get(ctx, scope, query, true)
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

			if uniqueAttrValue != integrationTestContainerName {
				t.Fatalf("Expected unique attribute value to be %s, got %s", integrationTestContainerName, uniqueAttrValue)
			}

			log.Printf("Successfully retrieved blob container %s", integrationTestContainerName)
		})

		t.Run("SearchBlobContainers", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Searching blob containers in storage account %s", storageAccountName)

			bcWrapper := manual.NewStorageBlobContainer(
				clients.NewBlobContainersClient(bcClient),
				subscriptionID,
				integrationTestResourceGroup,
			)
			scope := bcWrapper.Scopes()[0]

			bcAdapter := sources.WrapperToAdapter(bcWrapper, sdpcache.NewNoOpCache())

			// Check if adapter supports search
			searchable, ok := bcAdapter.(discovery.SearchableAdapter)
			if !ok {
				t.Fatalf("Adapter does not support Search operation")
			}

			sdpItems, err := searchable.Search(ctx, scope, storageAccountName, true)
			if err != nil {
				t.Fatalf("Failed to search blob containers: %v", err)
			}

			if len(sdpItems) < 1 {
				t.Fatalf("Expected at least one blob container, got %d", len(sdpItems))
			}

			var found bool
			for _, item := range sdpItems {
				uniqueAttrKey := item.GetUniqueAttribute()
				if v, err := item.GetAttributes().Get(uniqueAttrKey); err == nil && v == integrationTestContainerName {
					found = true
					break
				}
			}

			if !found {
				t.Fatalf("Expected to find container %s in the search results", integrationTestContainerName)
			}

			log.Printf("Found %d blob containers in search results", len(sdpItems))
		})

		t.Run("VerifyLinkedItems", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Verifying linked items for blob container %s", integrationTestContainerName)

			bcWrapper := manual.NewStorageBlobContainer(
				clients.NewBlobContainersClient(bcClient),
				subscriptionID,
				integrationTestResourceGroup,
			)
			scope := bcWrapper.Scopes()[0]

			bcAdapter := sources.WrapperToAdapter(bcWrapper, sdpcache.NewNoOpCache())
			query := storageAccountName + shared.QuerySeparator + integrationTestContainerName
			sdpItem, qErr := bcAdapter.Get(ctx, scope, query, true)
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

			log.Printf("Verified %d linked item queries for blob container %s", len(linkedQueries), integrationTestContainerName)
		})
	})

	t.Run("Teardown", func(t *testing.T) {
		ctx := t.Context()

		// Delete blob container
		err := deleteBlobContainer(ctx, bcClient, integrationTestResourceGroup, storageAccountName, integrationTestContainerName)
		if err != nil {
			t.Fatalf("Failed to delete blob container: %v", err)
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

// generateStorageAccountName generates a unique storage account name
// Storage account names must be globally unique, 3-24 characters, lowercase letters and numbers only
func generateStorageAccountName(baseName string) string {
	// Ensure base name is lowercase and valid
	baseName = strings.ToLower(baseName)
	baseName = strings.ReplaceAll(baseName, "-", "")

	// Add random suffix to ensure uniqueness
	rng := rand.New(rand.NewSource(time.Now().UnixNano() + int64(os.Getpid())))
	suffix := fmt.Sprintf("%04d", rng.Intn(10000))

	name := baseName + suffix

	// Ensure length is within limits (3-24 chars)
	if len(name) > 24 {
		name = name[:24]
	}
	if len(name) < 3 {
		name = name + "000"
	}

	return name
}

// createStorageAccount creates an Azure storage account (idempotent)
func createStorageAccount(ctx context.Context, client *armstorage.AccountsClient, resourceGroupName, accountName, location string) error {
	// Check if storage account already exists
	_, err := client.GetProperties(ctx, resourceGroupName, accountName, nil)
	if err == nil {
		log.Printf("Storage account %s already exists, skipping creation", accountName)
		return nil
	}

	// Create the storage account
	poller, err := client.BeginCreate(ctx, resourceGroupName, accountName, armstorage.AccountCreateParameters{
		Location: ptr.To(location),
		Kind:     ptr.To(armstorage.KindStorageV2),
		SKU: &armstorage.SKU{
			Name: ptr.To(armstorage.SKUNameStandardLRS),
		},
		Properties: &armstorage.AccountPropertiesCreateParameters{
			AccessTier: ptr.To(armstorage.AccessTierHot),
		},
		Tags: map[string]*string{
			"purpose": ptr.To("overmind-integration-tests"),
			"test":    ptr.To("storage-blob-container"),
		},
	}, nil)
	if err != nil {
		// Check if storage account already exists (conflict)
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusConflict {
			log.Printf("Storage account %s already exists (conflict), skipping creation", accountName)
			return nil
		}
		return fmt.Errorf("failed to begin creating storage account: %w", err)
	}

	resp, err := poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to create storage account: %w", err)
	}

	// Verify the storage account was created successfully
	if resp.Properties == nil || resp.Properties.ProvisioningState == nil {
		return fmt.Errorf("storage account created but provisioning state is unknown")
	}

	provisioningState := *resp.Properties.ProvisioningState
	if provisioningState != armstorage.ProvisioningStateSucceeded {
		return fmt.Errorf("storage account provisioning state is %s, expected %s", provisioningState, armstorage.ProvisioningStateSucceeded)
	}

	log.Printf("Storage account %s created successfully with provisioning state: %s", accountName, provisioningState)
	return nil
}

// waitForStorageAccountAvailable polls until the storage account is available via the Get API
func waitForStorageAccountAvailable(ctx context.Context, client *armstorage.AccountsClient, resourceGroupName, accountName string) error {
	maxAttempts := 20
	pollInterval := 10 * time.Second

	log.Printf("Waiting for storage account %s to be available via API...", accountName)

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		resp, err := client.GetProperties(ctx, resourceGroupName, accountName, nil)
		if err != nil {
			var respErr *azcore.ResponseError
			if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
				log.Printf("Storage account %s not yet available (attempt %d/%d), waiting %v...", accountName, attempt, maxAttempts, pollInterval)
				time.Sleep(pollInterval)
				continue
			}
			return fmt.Errorf("error checking storage account availability: %w", err)
		}

		// Check provisioning state
		if resp.Properties != nil && resp.Properties.ProvisioningState != nil {
			state := *resp.Properties.ProvisioningState
			if state == armstorage.ProvisioningStateSucceeded {
				log.Printf("Storage account %s is available with provisioning state: %s", accountName, state)
				return nil
			}
			if state == "Failed" {
				return fmt.Errorf("storage account provisioning failed with state: %s", state)
			}
			// Still provisioning, wait and retry
			log.Printf("Storage account %s provisioning state: %s (attempt %d/%d), waiting...", accountName, state, attempt, maxAttempts)
			time.Sleep(pollInterval)
			continue
		}

		// Storage account exists but no provisioning state - consider it available
		log.Printf("Storage account %s is available", accountName)
		return nil
	}

	return fmt.Errorf("timeout waiting for storage account %s to be available after %d attempts", accountName, maxAttempts)
}

// createBlobContainer creates an Azure blob container (idempotent)
func createBlobContainer(ctx context.Context, client *armstorage.BlobContainersClient, resourceGroupName, accountName, containerName string) error {
	// Check if container already exists
	_, err := client.Get(ctx, resourceGroupName, accountName, containerName, nil)
	if err == nil {
		log.Printf("Blob container %s already exists, skipping creation", containerName)
		return nil
	}

	// Create the blob container
	resp, err := client.Create(ctx, resourceGroupName, accountName, containerName, armstorage.BlobContainer{
		ContainerProperties: &armstorage.ContainerProperties{
			PublicAccess: ptr.To(armstorage.PublicAccessNone),
		},
	}, nil)
	if err != nil {
		// Check if container already exists (conflict)
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusConflict {
			log.Printf("Blob container %s already exists (conflict), skipping creation", containerName)
			return nil
		}
		return fmt.Errorf("failed to create blob container: %w", err)
	}

	// Verify the container was created successfully
	if resp.ID == nil {
		return fmt.Errorf("blob container created but ID is unknown")
	}

	log.Printf("Blob container %s created successfully", containerName)
	return nil
}

// deleteBlobContainer deletes an Azure blob container
func deleteBlobContainer(ctx context.Context, client *armstorage.BlobContainersClient, resourceGroupName, accountName, containerName string) error {
	_, err := client.Delete(ctx, resourceGroupName, accountName, containerName, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			log.Printf("Blob container %s not found, skipping deletion", containerName)
			return nil
		}
		return fmt.Errorf("failed to delete blob container: %w", err)
	}

	log.Printf("Blob container %s deleted successfully", containerName)
	return nil
}

// deleteStorageAccount deletes an Azure storage account
func deleteStorageAccount(ctx context.Context, client *armstorage.AccountsClient, resourceGroupName, accountName string) error {
	_, err := client.Delete(ctx, resourceGroupName, accountName, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			log.Printf("Storage account %s not found, skipping deletion", accountName)
			return nil
		}
		return fmt.Errorf("failed to delete storage account: %w", err)
	}

	log.Printf("Storage account %s deleted successfully", accountName)

	// Poll to verify the storage account is actually deleted and Azure has released associated resources.
	// Azure may take some time to fully delete the storage account and release its globally unique name.
	// This ensures subsequent test runs can reuse the same storage account name without conflicts.
	// The polling approach is more efficient than a fixed sleep as it returns as soon as deletion is confirmed.
	err = waitForStorageAccountDeleted(ctx, client, resourceGroupName, accountName)
	if err != nil {
		// Log the error but don't fail - deletion was initiated successfully
		// The polling failure might be due to timeout, but the resource should still be deleted
		log.Printf("Warning: Could not confirm storage account deletion via polling: %v", err)
	}

	return nil
}

// waitForStorageAccountDeleted polls until the storage account is confirmed deleted
// This ensures Azure has released the storage account name and associated resources.
// The wait duration can be configured via AZURE_RESOURCE_DELETE_WAIT_SECONDS environment variable
// (default: 30 seconds max wait time with 2-second polling intervals).
func waitForStorageAccountDeleted(ctx context.Context, client *armstorage.AccountsClient, resourceGroupName, accountName string) error {
	// Allow configuration via environment variable, default to 30 seconds
	maxWaitSeconds := 30
	if envWait := os.Getenv("AZURE_RESOURCE_DELETE_WAIT_SECONDS"); envWait != "" {
		if parsed, err := time.ParseDuration(envWait + "s"); err == nil {
			maxWaitSeconds = int(parsed.Seconds())
		}
	}

	maxAttempts := maxWaitSeconds / 2 // Poll every 2 seconds
	pollInterval := 2 * time.Second

	log.Printf("Verifying storage account %s is deleted (max wait: %d seconds)...", accountName, maxWaitSeconds)

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		_, err := client.GetProperties(ctx, resourceGroupName, accountName, nil)
		if err != nil {
			var respErr *azcore.ResponseError
			if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
				log.Printf("Storage account %s confirmed deleted", accountName)
				return nil
			}
			// Unexpected error - log but continue polling
			log.Printf("Unexpected error checking storage account deletion status: %v (attempt %d/%d)", err, attempt, maxAttempts)
		} else {
			// Storage account still exists
			log.Printf("Storage account %s still exists (attempt %d/%d), waiting %v...", accountName, attempt, maxAttempts, pollInterval)
		}

		if attempt < maxAttempts {
			time.Sleep(pollInterval)
		}
	}

	return fmt.Errorf("timeout waiting for storage account %s to be confirmed deleted after %d attempts", accountName, maxAttempts)
}
