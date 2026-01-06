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
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/batch/armbatch"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources/v2"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage/v2"
	log "github.com/sirupsen/logrus"
	"k8s.io/utils/ptr"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	"github.com/overmindtech/cli/sources/azure/manual"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
)

const (
	integrationTestBatchAccountName = "ovm-integ-test-batch"
	integrationTestSANameForBatch   = "ovm-integ-test-sa-batch"
)

func TestBatchAccountIntegration(t *testing.T) {
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
	batchClient, err := armbatch.NewAccountClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Batch Account client: %v", err)
	}

	saClient, err := armstorage.NewAccountsClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Storage Accounts client: %v", err)
	}

	rgClient, err := armresources.NewResourceGroupsClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Resource Groups client: %v", err)
	}

	// Generate unique names (batch account names must be globally unique, 3-24 chars, lowercase alphanumeric)
	batchAccountName := generateBatchAccountName(integrationTestBatchAccountName)
	storageAccountName := generateStorageAccountName(integrationTestSANameForBatch)

	var storageAccountID string

	t.Run("Setup", func(t *testing.T) {
		ctx := t.Context()

		// Create resource group if it doesn't exist
		err := createResourceGroup(ctx, rgClient, integrationTestResourceGroup, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create resource group: %v", err)
		}

		// Create storage account (required for batch account auto-storage)
		err = createStorageAccount(ctx, saClient, integrationTestResourceGroup, storageAccountName, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create storage account: %v", err)
		}

		// Wait for storage account to be fully available
		err = waitForStorageAccountAvailable(ctx, saClient, integrationTestResourceGroup, storageAccountName)
		if err != nil {
			t.Fatalf("Failed waiting for storage account to be available: %v", err)
		}

		// Get storage account ID
		saResp, err := saClient.GetProperties(ctx, integrationTestResourceGroup, storageAccountName, nil)
		if err != nil {
			t.Fatalf("Failed to get storage account properties: %v", err)
		}
		storageAccountID = *saResp.ID

		// Create batch account
		err = createBatchAccount(ctx, batchClient, integrationTestResourceGroup, batchAccountName, integrationTestLocation, storageAccountID)
		if err != nil {
			t.Fatalf("Failed to create batch account: %v", err)
		}

		// Wait for batch account to be fully available
		err = waitForBatchAccountAvailable(ctx, batchClient, integrationTestResourceGroup, batchAccountName)
		if err != nil {
			t.Fatalf("Failed waiting for batch account to be available: %v", err)
		}
	})

	t.Run("Run", func(t *testing.T) {
		t.Run("GetBatchAccount", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Retrieving batch account %s, subscription %s, resource group %s",
				batchAccountName, subscriptionID, integrationTestResourceGroup)

			batchWrapper := manual.NewBatchAccount(
				clients.NewBatchAccountsClient(batchClient),
				subscriptionID,
				integrationTestResourceGroup,
			)
			scope := batchWrapper.Scopes()[0]

			batchAdapter := sources.WrapperToAdapter(batchWrapper)
			sdpItem, qErr := batchAdapter.Get(ctx, scope, batchAccountName, true)
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

			if uniqueAttrValue != batchAccountName {
				t.Fatalf("Expected unique attribute value to be %s, got %s", batchAccountName, uniqueAttrValue)
			}

			if sdpItem.GetType() != azureshared.BatchBatchAccount.String() {
				t.Fatalf("Expected type %s, got %s", azureshared.BatchBatchAccount, sdpItem.GetType())
			}

			log.Printf("Successfully retrieved batch account %s", batchAccountName)
		})

		t.Run("ListBatchAccounts", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Listing batch accounts in resource group %s", integrationTestResourceGroup)

			batchWrapper := manual.NewBatchAccount(
				clients.NewBatchAccountsClient(batchClient),
				subscriptionID,
				integrationTestResourceGroup,
			)
			scope := batchWrapper.Scopes()[0]

			batchAdapter := sources.WrapperToAdapter(batchWrapper)

			// Check if adapter supports list
			listable, ok := batchAdapter.(discovery.ListableAdapter)
			if !ok {
				t.Fatalf("Adapter does not support List operation")
			}

			sdpItems, err := listable.List(ctx, scope, true)
			if err != nil {
				t.Fatalf("Failed to list batch accounts: %v", err)
			}

			if len(sdpItems) < 1 {
				t.Fatalf("Expected at least one batch account, got %d", len(sdpItems))
			}

			var found bool
			for _, item := range sdpItems {
				uniqueAttrKey := item.GetUniqueAttribute()
				if v, err := item.GetAttributes().Get(uniqueAttrKey); err == nil && v == batchAccountName {
					found = true
					if item.GetType() != azureshared.BatchBatchAccount.String() {
						t.Errorf("Expected type %s, got %s", azureshared.BatchBatchAccount, item.GetType())
					}
					break
				}
			}

			if !found {
				t.Fatalf("Expected to find batch account %s in the list results", batchAccountName)
			}

			log.Printf("Found %d batch accounts in list results", len(sdpItems))
		})

		t.Run("VerifyLinkedItems", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Verifying linked items for batch account %s", batchAccountName)

			batchWrapper := manual.NewBatchAccount(
				clients.NewBatchAccountsClient(batchClient),
				subscriptionID,
				integrationTestResourceGroup,
			)
			scope := batchWrapper.Scopes()[0]

			batchAdapter := sources.WrapperToAdapter(batchWrapper)
			sdpItem, qErr := batchAdapter.Get(ctx, scope, batchAccountName, true)
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
				azureshared.StorageAccount.String():                      false, // External resource
				azureshared.BatchBatchApplication.String():               false, // Child resource
				azureshared.BatchBatchPool.String():                      false, // Child resource
				azureshared.BatchBatchCertificate.String():               false, // Child resource
				azureshared.BatchBatchPrivateEndpointConnection.String(): false, // Child resource
				azureshared.BatchBatchPrivateLinkResource.String():       false, // Child resource
				azureshared.BatchBatchDetector.String():                  false, // Child resource
			}

			for _, liq := range linkedQueries {
				linkedType := liq.GetQuery().GetType()
				if _, exists := expectedLinkedTypes[linkedType]; exists {
					expectedLinkedTypes[linkedType] = true

					// Verify the query method
					queryMethod := liq.GetQuery().GetMethod()
					if linkedType == azureshared.StorageAccount.String() {
						// External resources use GET
						if queryMethod != sdp.QueryMethod_GET {
							t.Errorf("Expected linked query method to be GET for %s, got %s", linkedType, queryMethod)
						}
					} else {
						// Child resources use SEARCH
						if queryMethod != sdp.QueryMethod_SEARCH {
							t.Errorf("Expected linked query method to be SEARCH for %s, got %s", linkedType, queryMethod)
						}
					}

					// Verify blast propagation
					if liq.GetBlastPropagation() == nil {
						t.Errorf("Expected blast propagation to be set for linked type %s", linkedType)
					} else {
						bp := liq.GetBlastPropagation()
						if linkedType == azureshared.StorageAccount.String() {
							// Storage account: In=true, Out=false (batch depends on storage)
							if bp.GetIn() != true {
								t.Errorf("Expected blast propagation In=true for storage account, got false")
							}
							if bp.GetOut() != false {
								t.Errorf("Expected blast propagation Out=false for storage account, got true")
							}
						} else {
							// Child resources: In=true, Out=true (tightly coupled)
							if bp.GetIn() != true {
								t.Errorf("Expected blast propagation In=true for %s, got false", linkedType)
							}
							if bp.GetOut() != true {
								t.Errorf("Expected blast propagation Out=true for %s, got false", linkedType)
							}
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

			log.Printf("Verified %d linked item queries for batch account %s", len(linkedQueries), batchAccountName)
		})
	})

	t.Run("Teardown", func(t *testing.T) {
		ctx := t.Context()

		// Delete batch account
		err := deleteBatchAccount(ctx, batchClient, integrationTestResourceGroup, batchAccountName)
		if err != nil {
			t.Fatalf("Failed to delete batch account: %v", err)
		}

		// Delete storage account (function is defined in storage-blob-container_test.go)
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

// generateBatchAccountName generates a unique batch account name
// Batch account names must be globally unique, 3-24 characters, lowercase alphanumeric
func generateBatchAccountName(baseName string) string {
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
		name = name + "000" // pad if too short
	}

	return name
}

// createBatchAccount creates an Azure Batch account (idempotent)
func createBatchAccount(ctx context.Context, client *armbatch.AccountClient, resourceGroupName, accountName, location, storageAccountID string) error {
	// Check if batch account already exists
	_, err := client.Get(ctx, resourceGroupName, accountName, nil)
	if err == nil {
		log.Printf("Batch account %s already exists, skipping creation", accountName)
		return nil
	}

	// Create the batch account
	poller, err := client.BeginCreate(ctx, resourceGroupName, accountName, armbatch.AccountCreateParameters{
		Location: ptr.To(location),
		Properties: &armbatch.AccountCreateProperties{
			AutoStorage: &armbatch.AutoStorageBaseProperties{
				StorageAccountID: ptr.To(storageAccountID),
			},
			PoolAllocationMode: ptr.To(armbatch.PoolAllocationModeBatchService),
		},
		Tags: map[string]*string{
			"purpose": ptr.To("overmind-integration-tests"),
			"test":    ptr.To("batch-account"),
		},
	}, nil)
	if err != nil {
		// Check if batch account already exists (conflict)
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusConflict {
			log.Printf("Batch account %s already exists (conflict), skipping creation", accountName)
			return nil
		}
		return fmt.Errorf("failed to begin creating batch account: %w", err)
	}

	resp, err := poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to create batch account: %w", err)
	}

	// Verify the batch account was created successfully
	if resp.Properties == nil || resp.Properties.ProvisioningState == nil {
		return fmt.Errorf("batch account created but provisioning state is unknown")
	}

	provisioningState := *resp.Properties.ProvisioningState
	if provisioningState != armbatch.ProvisioningStateSucceeded {
		return fmt.Errorf("batch account provisioning state is %s, expected %s", provisioningState, armbatch.ProvisioningStateSucceeded)
	}

	log.Printf("Batch account %s created successfully with provisioning state: %s", accountName, provisioningState)
	return nil
}

// waitForBatchAccountAvailable polls until the batch account is available via the Get API
func waitForBatchAccountAvailable(ctx context.Context, client *armbatch.AccountClient, resourceGroupName, accountName string) error {
	maxAttempts := 20
	pollInterval := 10 * time.Second

	log.Printf("Waiting for batch account %s to be available via API...", accountName)

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		resp, err := client.Get(ctx, resourceGroupName, accountName, nil)
		if err != nil {
			var respErr *azcore.ResponseError
			if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
				log.Printf("Batch account %s not yet available (attempt %d/%d), waiting %v...", accountName, attempt, maxAttempts, pollInterval)
				time.Sleep(pollInterval)
				continue
			}
			return fmt.Errorf("error checking batch account availability: %w", err)
		}

		// Check provisioning state
		if resp.Properties != nil && resp.Properties.ProvisioningState != nil {
			state := *resp.Properties.ProvisioningState
			if state == armbatch.ProvisioningStateSucceeded {
				log.Printf("Batch account %s is available with provisioning state: %s", accountName, state)
				return nil
			}
			if state == armbatch.ProvisioningStateFailed {
				return fmt.Errorf("batch account provisioning failed with state: %s", state)
			}
			// Still provisioning, wait and retry
			log.Printf("Batch account %s provisioning state: %s (attempt %d/%d), waiting...", accountName, state, attempt, maxAttempts)
			time.Sleep(pollInterval)
			continue
		}

		// Batch account exists but no provisioning state - consider it available
		log.Printf("Batch account %s is available", accountName)
		return nil
	}

	return fmt.Errorf("timeout waiting for batch account %s to be available after %d attempts", accountName, maxAttempts)
}

// deleteBatchAccount deletes an Azure Batch account
func deleteBatchAccount(ctx context.Context, client *armbatch.AccountClient, resourceGroupName, accountName string) error {
	log.Printf("Deleting batch account %s...", accountName)

	poller, err := client.BeginDelete(ctx, resourceGroupName, accountName, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			log.Printf("Batch account %s not found, skipping deletion", accountName)
			return nil
		}
		return fmt.Errorf("failed to begin deleting batch account: %w", err)
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to delete batch account: %w", err)
	}

	log.Printf("Batch account %s deleted successfully", accountName)
	return nil
}

