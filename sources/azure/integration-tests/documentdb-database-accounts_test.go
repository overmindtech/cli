package integrationtests

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/cosmos/armcosmos"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources/v2"
	log "github.com/sirupsen/logrus"
	"k8s.io/utils/ptr"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	"github.com/overmindtech/cli/sources/azure/manual"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
)

const (
	integrationTestCosmosDBAccountName = "ovm-integ-test-cosmos"
)

func TestDocumentDBDatabaseAccountsIntegration(t *testing.T) {
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
	cosmosClient, err := armcosmos.NewDatabaseAccountsClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Cosmos DB client: %v", err)
	}

	rgClient, err := armresources.NewResourceGroupsClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Resource Groups client: %v", err)
	}

	t.Run("Setup", func(t *testing.T) {
		ctx := t.Context()

		// Create resource group if it doesn't exist
		err := createResourceGroup(ctx, rgClient, integrationTestResourceGroup, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create resource group: %v", err)
		}

		// Create Cosmos DB account
		err = createCosmosDBAccount(ctx, cosmosClient, integrationTestResourceGroup, integrationTestCosmosDBAccountName, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create Cosmos DB account: %v", err)
		}

		// Wait for Cosmos DB account to be fully available
		err = waitForCosmosDBAccountAvailable(ctx, cosmosClient, integrationTestResourceGroup, integrationTestCosmosDBAccountName)
		if err != nil {
			t.Fatalf("Failed waiting for Cosmos DB account to be available: %v", err)
		}
	})

	t.Run("Run", func(t *testing.T) {
		t.Run("GetCosmosDBAccount", func(t *testing.T) {
			ctx := t.Context()

			// Try to get the test account, skip if it doesn't exist
			_, err := cosmosClient.Get(ctx, integrationTestResourceGroup, integrationTestCosmosDBAccountName, nil)
			if err != nil {
				t.Skipf("Cosmos DB account %s does not exist in resource group %s, skipping test. Error: %v", integrationTestCosmosDBAccountName, integrationTestResourceGroup, err)
			}

			log.Printf("Retrieving Cosmos DB account %s in subscription %s, resource group %s",
				integrationTestCosmosDBAccountName, subscriptionID, integrationTestResourceGroup)

			cosmosWrapper := manual.NewDocumentDBDatabaseAccounts(
				clients.NewDocumentDBDatabaseAccountsClient(cosmosClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := cosmosWrapper.Scopes()[0]

			cosmosAdapter := sources.WrapperToAdapter(cosmosWrapper, sdpcache.NewNoOpCache())
			sdpItem, qErr := cosmosAdapter.Get(ctx, scope, integrationTestCosmosDBAccountName, true)
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

			if uniqueAttrValue != integrationTestCosmosDBAccountName {
				t.Fatalf("Expected unique attribute value to be %s, got %s", integrationTestCosmosDBAccountName, uniqueAttrValue)
			}

			// Validate the item
			if err := sdpItem.Validate(); err != nil {
				t.Fatalf("SDP item validation failed: %v", err)
			}

			log.Printf("Successfully retrieved Cosmos DB account %s", integrationTestCosmosDBAccountName)
		})

		t.Run("ListCosmosDBAccounts", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Listing Cosmos DB accounts in subscription %s, resource group %s",
				subscriptionID, integrationTestResourceGroup)

			cosmosWrapper := manual.NewDocumentDBDatabaseAccounts(
				clients.NewDocumentDBDatabaseAccountsClient(cosmosClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := cosmosWrapper.Scopes()[0]

			cosmosAdapter := sources.WrapperToAdapter(cosmosWrapper, sdpcache.NewNoOpCache())

			// Check if adapter supports listing
			listable, ok := cosmosAdapter.(discovery.ListableAdapter)
			if !ok {
				t.Fatalf("Adapter does not support List operation")
			}

			sdpItems, err := listable.List(ctx, scope, true)
			if err != nil {
				t.Fatalf("Failed to list Cosmos DB accounts: %v", err)
			}

			// Note: len(sdpItems) can be 0 or more, which is valid
			_ = len(sdpItems)

			// Validate all items
			for _, item := range sdpItems {
				if err := item.Validate(); err != nil {
					t.Fatalf("SDP item validation failed: %v", err)
				}
			}

			log.Printf("Successfully listed %d Cosmos DB accounts", len(sdpItems))
		})
	})

	t.Run("Teardown", func(t *testing.T) {
		ctx := t.Context()

		// Delete Cosmos DB account
		err := deleteCosmosDBAccount(ctx, cosmosClient, integrationTestResourceGroup, integrationTestCosmosDBAccountName)
		if err != nil {
			t.Fatalf("Failed to delete Cosmos DB account: %v", err)
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

// createCosmosDBAccount creates an Azure Cosmos DB account (idempotent)
func createCosmosDBAccount(ctx context.Context, client *armcosmos.DatabaseAccountsClient, resourceGroupName, accountName, location string) error {
	// Check if Cosmos DB account already exists
	_, err := client.Get(ctx, resourceGroupName, accountName, nil)
	if err == nil {
		log.Printf("Cosmos DB account %s already exists, skipping creation", accountName)
		return nil
	}

	// Create the Cosmos DB account
	// Using SQL API as the default, which is the most common
	poller, err := client.BeginCreateOrUpdate(ctx, resourceGroupName, accountName, armcosmos.DatabaseAccountCreateUpdateParameters{
		Location: ptr.To(location),
		Kind:     ptr.To(armcosmos.DatabaseAccountKindGlobalDocumentDB),
		Properties: &armcosmos.DatabaseAccountCreateUpdateProperties{
			DatabaseAccountOfferType: ptr.To("Standard"),
			Locations: []*armcosmos.Location{
				{
					LocationName:     ptr.To(location),
					FailoverPriority: ptr.To[int32](0),
					IsZoneRedundant:  ptr.To(false),
				},
			},
			ConsistencyPolicy: &armcosmos.ConsistencyPolicy{
				DefaultConsistencyLevel: ptr.To(armcosmos.DefaultConsistencyLevelSession),
			},
		},
		Tags: map[string]*string{
			"purpose": ptr.To("overmind-integration-tests"),
			"test":    ptr.To("documentdb-database-accounts"),
		},
	}, nil)
	if err != nil {
		// Check if Cosmos DB account already exists (conflict)
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusConflict {
			log.Printf("Cosmos DB account %s already exists (conflict), skipping creation", accountName)
			return nil
		}
		return fmt.Errorf("failed to begin creating Cosmos DB account: %w", err)
	}

	resp, err := poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to create Cosmos DB account: %w", err)
	}

	// Verify the Cosmos DB account was created successfully
	if resp.Properties == nil || resp.Properties.ProvisioningState == nil {
		return fmt.Errorf("Cosmos DB account created but provisioning state is unknown")
	}

	provisioningState := *resp.Properties.ProvisioningState
	if provisioningState != "Succeeded" {
		return fmt.Errorf("Cosmos DB account provisioning state is %s, expected Succeeded", provisioningState)
	}

	log.Printf("Cosmos DB account %s created successfully with provisioning state: %s", accountName, provisioningState)
	return nil
}

// waitForCosmosDBAccountAvailable waits for a Cosmos DB account to be fully available
func waitForCosmosDBAccountAvailable(ctx context.Context, client *armcosmos.DatabaseAccountsClient, resourceGroupName, accountName string) error {
	maxAttempts := 20
	pollInterval := 10 * time.Second

	for attempt := range maxAttempts {
		resp, err := client.Get(ctx, resourceGroupName, accountName, nil)
		if err != nil {
			return fmt.Errorf("failed to get Cosmos DB account: %w", err)
		}

		if resp.Properties != nil && resp.Properties.ProvisioningState != nil {
			state := *resp.Properties.ProvisioningState
			if state == "Succeeded" {
				log.Printf("Cosmos DB account %s is available", accountName)
				return nil
			}
			log.Printf("Cosmos DB account %s provisioning state: %s (attempt %d/%d)", accountName, state, attempt+1, maxAttempts)
		}

		time.Sleep(pollInterval)
	}

	return fmt.Errorf("Cosmos DB account %s did not become available within the timeout period", accountName)
}

// deleteCosmosDBAccount deletes an Azure Cosmos DB account (idempotent)
func deleteCosmosDBAccount(ctx context.Context, client *armcosmos.DatabaseAccountsClient, resourceGroupName, accountName string) error {
	// Check if Cosmos DB account exists
	_, err := client.Get(ctx, resourceGroupName, accountName, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			log.Printf("Cosmos DB account %s does not exist, skipping deletion", accountName)
			return nil
		}
		return fmt.Errorf("failed to check if Cosmos DB account exists: %w", err)
	}

	// Delete the Cosmos DB account
	poller, err := client.BeginDelete(ctx, resourceGroupName, accountName, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			log.Printf("Cosmos DB account %s does not exist, skipping deletion", accountName)
			return nil
		}
		return fmt.Errorf("failed to begin deleting Cosmos DB account: %w", err)
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to delete Cosmos DB account: %w", err)
	}

	log.Printf("Cosmos DB account %s deleted successfully", accountName)
	return nil
}
