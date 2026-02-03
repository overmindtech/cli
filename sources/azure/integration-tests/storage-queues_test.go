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
	integrationTestQueueName = "ovm-integ-test-queue"
)

func TestStorageQueuesIntegration(t *testing.T) {
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

	queueClient, err := armstorage.NewQueueClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Queue client: %v", err)
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

		// Create queue
		err = createQueue(ctx, queueClient, integrationTestResourceGroup, storageAccountName, integrationTestQueueName)
		if err != nil {
			t.Fatalf("Failed to create queue: %v", err)
		}
	})

	t.Run("Run", func(t *testing.T) {
		t.Run("GetQueue", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Retrieving queue %s in storage account %s, subscription %s, resource group %s",
				integrationTestQueueName, storageAccountName, subscriptionID, integrationTestResourceGroup)

			queueWrapper := manual.NewStorageQueues(
				clients.NewQueuesClient(queueClient),
				subscriptionID,
				integrationTestResourceGroup,
			)
			scope := queueWrapper.Scopes()[0]

			queueAdapter := sources.WrapperToAdapter(queueWrapper, sdpcache.NewNoOpCache())
			// Get requires storageAccountName and queueName as query parts
			query := shared.CompositeLookupKey(storageAccountName, integrationTestQueueName)
			sdpItem, qErr := queueAdapter.Get(ctx, scope, query, true)
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

			expectedID := shared.CompositeLookupKey(storageAccountName, integrationTestQueueName)
			if uniqueAttrValue != expectedID {
				t.Fatalf("Expected unique attribute value to be %s, got %s", expectedID, uniqueAttrValue)
			}

			log.Printf("Successfully retrieved queue %s", integrationTestQueueName)
		})

		t.Run("SearchQueues", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Searching queues in storage account %s", storageAccountName)

			queueWrapper := manual.NewStorageQueues(
				clients.NewQueuesClient(queueClient),
				subscriptionID,
				integrationTestResourceGroup,
			)
			scope := queueWrapper.Scopes()[0]

			queueAdapter := sources.WrapperToAdapter(queueWrapper, sdpcache.NewNoOpCache())

			// Check if adapter supports search
			searchable, ok := queueAdapter.(discovery.SearchableAdapter)
			if !ok {
				t.Fatalf("Adapter does not support Search operation")
			}

			sdpItems, err := searchable.Search(ctx, scope, storageAccountName, true)
			if err != nil {
				t.Fatalf("Failed to search queues: %v", err)
			}

			if len(sdpItems) < 1 {
				t.Fatalf("Expected at least one queue, got %d", len(sdpItems))
			}

			var found bool
			for _, item := range sdpItems {
				uniqueAttrKey := item.GetUniqueAttribute()
				expectedID := shared.CompositeLookupKey(storageAccountName, integrationTestQueueName)
				if v, err := item.GetAttributes().Get(uniqueAttrKey); err == nil && v == expectedID {
					found = true
					break
				}
			}

			if !found {
				t.Fatalf("Expected to find queue %s in the search results", integrationTestQueueName)
			}

			log.Printf("Found %d queues in search results", len(sdpItems))
		})

		t.Run("VerifyItemAttributes", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Verifying item attributes for queue %s", integrationTestQueueName)

			queueWrapper := manual.NewStorageQueues(
				clients.NewQueuesClient(queueClient),
				subscriptionID,
				integrationTestResourceGroup,
			)
			scope := queueWrapper.Scopes()[0]

			queueAdapter := sources.WrapperToAdapter(queueWrapper, sdpcache.NewNoOpCache())
			query := shared.CompositeLookupKey(storageAccountName, integrationTestQueueName)
			sdpItem, qErr := queueAdapter.Get(ctx, scope, query, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			// Verify item type
			if sdpItem.GetType() != azureshared.StorageQueue.String() {
				t.Errorf("Expected item type %s, got %s", azureshared.StorageQueue, sdpItem.GetType())
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

			log.Printf("Verified item attributes for queue %s", integrationTestQueueName)
		})

		t.Run("VerifyLinkedItems", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Verifying linked items for queue %s", integrationTestQueueName)

			queueWrapper := manual.NewStorageQueues(
				clients.NewQueuesClient(queueClient),
				subscriptionID,
				integrationTestResourceGroup,
			)
			scope := queueWrapper.Scopes()[0]

			queueAdapter := sources.WrapperToAdapter(queueWrapper, sdpcache.NewNoOpCache())
			query := shared.CompositeLookupKey(storageAccountName, integrationTestQueueName)
			sdpItem, qErr := queueAdapter.Get(ctx, scope, query, true)
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

			log.Printf("Verified %d linked item queries for queue %s", len(linkedQueries), integrationTestQueueName)
		})
	})

	t.Run("Teardown", func(t *testing.T) {
		ctx := t.Context()

		// Delete queue
		err := deleteQueue(ctx, queueClient, integrationTestResourceGroup, storageAccountName, integrationTestQueueName)
		if err != nil {
			t.Fatalf("Failed to delete queue: %v", err)
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

// createQueue creates an Azure storage queue (idempotent)
func createQueue(ctx context.Context, client *armstorage.QueueClient, resourceGroupName, accountName, queueName string) error {
	// Check if queue already exists
	_, err := client.Get(ctx, resourceGroupName, accountName, queueName, nil)
	if err == nil {
		log.Printf("Queue %s already exists, skipping creation", queueName)
		return nil
	}

	// Create the queue
	// Queues don't require any properties, they can be created with an empty QueueProperties
	resp, err := client.Create(ctx, resourceGroupName, accountName, queueName, armstorage.Queue{
		QueueProperties: &armstorage.QueueProperties{
			// Metadata is optional, can be nil
		},
	}, nil)
	if err != nil {
		// Check if queue already exists (conflict)
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusConflict {
			log.Printf("Queue %s already exists (conflict), skipping creation", queueName)
			return nil
		}
		return fmt.Errorf("failed to create queue: %w", err)
	}

	// Verify the queue was created successfully
	if resp.ID == nil {
		return fmt.Errorf("queue created but ID is unknown")
	}

	log.Printf("Queue %s created successfully", queueName)
	return nil
}

// deleteQueue deletes an Azure storage queue
func deleteQueue(ctx context.Context, client *armstorage.QueueClient, resourceGroupName, accountName, queueName string) error {
	_, err := client.Delete(ctx, resourceGroupName, accountName, queueName, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			log.Printf("Queue %s not found, skipping deletion", queueName)
			return nil
		}
		return fmt.Errorf("failed to delete queue: %w", err)
	}

	log.Printf("Queue %s deleted successfully", queueName)
	return nil
}
