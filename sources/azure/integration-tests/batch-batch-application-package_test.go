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
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/batch/armbatch/v4"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources/v2"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage/v3"
	log "github.com/sirupsen/logrus"

	"github.com/overmindtech/cli/go/discovery"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	"github.com/overmindtech/cli/sources/azure/manual"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/shared"
)

const (
	integrationTestBatchAppPkgAccountName = "ovm-integ-test-sa-pkg"
	integrationTestBatchAppPkgBatchName   = "ovm-integ-test-pkg"
	integrationTestBatchAppName           = "ovm-integ-test-app"
	integrationTestBatchAppPkgVersion     = "1.0"
)

func TestBatchApplicationPackageIntegration(t *testing.T) {
	subscriptionID := os.Getenv("AZURE_SUBSCRIPTION_ID")
	if subscriptionID == "" {
		t.Skip("AZURE_SUBSCRIPTION_ID environment variable not set")
	}

	cred, err := azureshared.NewAzureCredential(t.Context())
	if err != nil {
		t.Fatalf("Failed to create Azure credential: %v", err)
	}

	rgClient, err := armresources.NewResourceGroupsClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Resource Groups client: %v", err)
	}

	saClient, err := armstorage.NewAccountsClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Storage Accounts client: %v", err)
	}

	batchAccountClient, err := armbatch.NewAccountClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Batch Account client: %v", err)
	}

	batchAppClient, err := armbatch.NewApplicationClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Batch Application client: %v", err)
	}

	batchAppPkgClient, err := armbatch.NewApplicationPackageClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Batch Application Package client: %v", err)
	}

	storageAccountName := generateStorageAccountName(integrationTestBatchAppPkgAccountName)
	batchAccountName := generateBatchAccountName(integrationTestBatchAppPkgBatchName)
	setupCompleted := false

	t.Run("Setup", func(t *testing.T) {
		ctx := t.Context()

		err := createResourceGroup(ctx, rgClient, integrationTestResourceGroup, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create resource group: %v", err)
		}

		err = createStorageAccount(ctx, saClient, integrationTestResourceGroup, storageAccountName, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create storage account: %v", err)
		}

		err = waitForStorageAccountAvailable(ctx, saClient, integrationTestResourceGroup, storageAccountName)
		if err != nil {
			t.Fatalf("Failed waiting for storage account: %v", err)
		}

		saResp, err := saClient.GetProperties(ctx, integrationTestResourceGroup, storageAccountName, nil)
		if err != nil {
			t.Fatalf("Failed to get storage account properties: %v", err)
		}
		storageAccountID := *saResp.ID

		err = createBatchAccount(ctx, batchAccountClient, integrationTestResourceGroup, batchAccountName, integrationTestLocation, storageAccountID)
		if err != nil {
			if errors.Is(err, errBatchQuotaExceeded) {
				t.Skipf("Skipping Batch application package integration test due to Azure subscription quota: %v", err)
			}
			t.Fatalf("Failed to create batch account: %v", err)
		}

		err = waitForBatchAccountAvailable(ctx, batchAccountClient, integrationTestResourceGroup, batchAccountName)
		if err != nil {
			t.Fatalf("Failed waiting for batch account: %v", err)
		}

		err = createBatchApplication(ctx, batchAppClient, integrationTestResourceGroup, batchAccountName, integrationTestBatchAppName)
		if err != nil {
			t.Fatalf("Failed to create batch application: %v", err)
		}

		err = createBatchApplicationPackage(ctx, batchAppPkgClient, integrationTestResourceGroup, batchAccountName, integrationTestBatchAppName, integrationTestBatchAppPkgVersion)
		if err != nil {
			t.Fatalf("Failed to create batch application package: %v", err)
		}
		setupCompleted = true
	})

	t.Run("Run", func(t *testing.T) {
		if !setupCompleted {
			t.Skip("Skipping Run: Setup did not complete successfully")
		}

		t.Run("GetApplicationPackage", func(t *testing.T) {
			ctx := t.Context()

			wrapper := manual.NewBatchBatchApplicationPackage(
				clients.NewBatchApplicationPackagesClient(batchAppPkgClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := wrapper.Scopes()[0]
			adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

			query := shared.CompositeLookupKey(batchAccountName, integrationTestBatchAppName, integrationTestBatchAppPkgVersion)
			sdpItem, qErr := adapter.Get(ctx, scope, query, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			if sdpItem == nil {
				t.Fatalf("Expected sdpItem to be non-nil")
			}

			expectedUnique := shared.CompositeLookupKey(batchAccountName, integrationTestBatchAppName, integrationTestBatchAppPkgVersion)
			uniqueAttrKey := sdpItem.GetUniqueAttribute()
			uniqueAttrValue, err := sdpItem.GetAttributes().Get(uniqueAttrKey)
			if err != nil {
				t.Fatalf("Failed to get unique attribute: %v", err)
			}

			if uniqueAttrValue != expectedUnique {
				t.Errorf("Expected unique attribute value %s, got %s", expectedUnique, uniqueAttrValue)
			}

			log.Printf("Successfully retrieved application package %s", integrationTestBatchAppPkgVersion)
		})

		t.Run("SearchApplicationPackages", func(t *testing.T) {
			ctx := t.Context()

			wrapper := manual.NewBatchBatchApplicationPackage(
				clients.NewBatchApplicationPackagesClient(batchAppPkgClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := wrapper.Scopes()[0]
			adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

			searchable, ok := adapter.(discovery.SearchableAdapter)
			if !ok {
				t.Fatalf("Adapter does not support Search operation")
			}

			searchQuery := shared.CompositeLookupKey(batchAccountName, integrationTestBatchAppName)
			sdpItems, err := searchable.Search(ctx, scope, searchQuery, true)
			if err != nil {
				t.Fatalf("Failed to search application packages: %v", err)
			}

			if len(sdpItems) < 1 {
				t.Fatalf("Expected at least one application package, got %d", len(sdpItems))
			}

			expectedUnique := shared.CompositeLookupKey(batchAccountName, integrationTestBatchAppName, integrationTestBatchAppPkgVersion)
			var found bool
			for _, item := range sdpItems {
				uniqueAttrKey := item.GetUniqueAttribute()
				if v, getErr := item.GetAttributes().Get(uniqueAttrKey); getErr == nil && v == expectedUnique {
					found = true
					break
				}
			}

			if !found {
				t.Fatalf("Expected to find application package %s in search results", integrationTestBatchAppPkgVersion)
			}

			log.Printf("Found %d application packages in search results", len(sdpItems))
		})

		t.Run("VerifyLinkedItems", func(t *testing.T) {
			ctx := t.Context()

			wrapper := manual.NewBatchBatchApplicationPackage(
				clients.NewBatchApplicationPackagesClient(batchAppPkgClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := wrapper.Scopes()[0]
			adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

			query := shared.CompositeLookupKey(batchAccountName, integrationTestBatchAppName, integrationTestBatchAppPkgVersion)
			sdpItem, qErr := adapter.Get(ctx, scope, query, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			linkedQueries := sdpItem.GetLinkedItemQueries()
			if len(linkedQueries) == 0 {
				t.Fatalf("Expected linked item queries, but got none")
			}

			for _, liq := range linkedQueries {
				q := liq.GetQuery()
				if q.GetType() == "" {
					t.Error("Linked item query has empty Type")
				}
				if q.GetQuery() == "" {
					t.Error("Linked item query has empty Query")
				}
				if q.GetScope() == "" {
					t.Error("Linked item query has empty Scope")
				}
				if q.GetMethod() != sdp.QueryMethod_GET && q.GetMethod() != sdp.QueryMethod_SEARCH {
					t.Errorf("Linked item query has invalid Method: %s", q.GetMethod())
				}
			}

			// Verify parent application link exists
			var hasAppLink bool
			for _, liq := range linkedQueries {
				if liq.GetQuery().GetType() == azureshared.BatchBatchApplication.String() {
					hasAppLink = true
					break
				}
			}
			if !hasAppLink {
				t.Error("Expected linked query to parent BatchBatchApplication, but didn't find one")
			}

			// Verify parent account link exists
			var hasAccountLink bool
			for _, liq := range linkedQueries {
				if liq.GetQuery().GetType() == azureshared.BatchBatchAccount.String() {
					hasAccountLink = true
					break
				}
			}
			if !hasAccountLink {
				t.Error("Expected linked query to parent BatchBatchAccount, but didn't find one")
			}

			log.Printf("Verified %d linked item queries", len(linkedQueries))
		})

		t.Run("VerifyItemAttributes", func(t *testing.T) {
			ctx := t.Context()

			wrapper := manual.NewBatchBatchApplicationPackage(
				clients.NewBatchApplicationPackagesClient(batchAppPkgClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := wrapper.Scopes()[0]
			adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

			query := shared.CompositeLookupKey(batchAccountName, integrationTestBatchAppName, integrationTestBatchAppPkgVersion)
			sdpItem, qErr := adapter.Get(ctx, scope, query, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			if sdpItem.GetType() != azureshared.BatchBatchApplicationPackage.String() {
				t.Errorf("Expected type %s, got %s", azureshared.BatchBatchApplicationPackage.String(), sdpItem.GetType())
			}

			expectedScope := subscriptionID + "." + integrationTestResourceGroup
			if sdpItem.GetScope() != expectedScope {
				t.Errorf("Expected scope %s, got %s", expectedScope, sdpItem.GetScope())
			}

			if sdpItem.GetUniqueAttribute() != "uniqueAttr" {
				t.Errorf("Expected unique attribute 'uniqueAttr', got %s", sdpItem.GetUniqueAttribute())
			}

			if err := sdpItem.Validate(); err != nil {
				t.Fatalf("Item validation failed: %v", err)
			}
		})
	})

	t.Run("Teardown", func(t *testing.T) {
		ctx := t.Context()

		err := deleteBatchApplicationPackage(ctx, batchAppPkgClient, integrationTestResourceGroup, batchAccountName, integrationTestBatchAppName, integrationTestBatchAppPkgVersion)
		if err != nil {
			t.Logf("Warning: failed to delete application package: %v", err)
		}

		err = deleteBatchApplication(ctx, batchAppClient, integrationTestResourceGroup, batchAccountName, integrationTestBatchAppName)
		if err != nil {
			t.Logf("Warning: failed to delete batch application: %v", err)
		}

		err = deleteBatchAccount(ctx, batchAccountClient, integrationTestResourceGroup, batchAccountName)
		if err != nil {
			t.Logf("Warning: failed to delete batch account: %v", err)
		}

		err = deleteStorageAccount(ctx, saClient, integrationTestResourceGroup, storageAccountName)
		if err != nil {
			t.Logf("Warning: failed to delete storage account: %v", err)
		}
	})
}

func createBatchApplication(ctx context.Context, client *armbatch.ApplicationClient, resourceGroupName, accountName, applicationName string) error {
	_, err := client.Get(ctx, resourceGroupName, accountName, applicationName, nil)
	if err == nil {
		log.Printf("Batch application %s already exists, skipping creation", applicationName)
		return nil
	}

	allowUpdates := true
	_, err = client.Create(ctx, resourceGroupName, accountName, applicationName, armbatch.Application{
		Properties: &armbatch.ApplicationProperties{
			DisplayName:  new("Integration Test Application"),
			AllowUpdates: &allowUpdates,
		},
	}, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusConflict {
			log.Printf("Batch application %s already exists (conflict), skipping creation", applicationName)
			return nil
		}
		return fmt.Errorf("failed to create batch application: %w", err)
	}

	log.Printf("Batch application %s created successfully", applicationName)
	return nil
}

func createBatchApplicationPackage(ctx context.Context, client *armbatch.ApplicationPackageClient, resourceGroupName, accountName, applicationName, versionName string) error {
	_, err := client.Get(ctx, resourceGroupName, accountName, applicationName, versionName, nil)
	if err == nil {
		log.Printf("Batch application package %s already exists, skipping creation", versionName)
		return nil
	}

	_, err = client.Create(ctx, resourceGroupName, accountName, applicationName, versionName, armbatch.ApplicationPackage{}, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusConflict {
			log.Printf("Batch application package %s already exists (conflict), skipping creation", versionName)
			return nil
		}
		return fmt.Errorf("failed to create batch application package: %w", err)
	}

	log.Printf("Batch application package %s created successfully", versionName)

	// Wait briefly for the package to become available
	maxAttempts := 10
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		_, getErr := client.Get(ctx, resourceGroupName, accountName, applicationName, versionName, nil)
		if getErr == nil {
			return nil
		}
		time.Sleep(2 * time.Second)
	}

	return nil
}

func deleteBatchApplicationPackage(ctx context.Context, client *armbatch.ApplicationPackageClient, resourceGroupName, accountName, applicationName, versionName string) error {
	_, err := client.Delete(ctx, resourceGroupName, accountName, applicationName, versionName, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			log.Printf("Batch application package %s not found, skipping deletion", versionName)
			return nil
		}
		return fmt.Errorf("failed to delete batch application package: %w", err)
	}

	log.Printf("Batch application package %s deleted successfully", versionName)
	return nil
}

func deleteBatchApplication(ctx context.Context, client *armbatch.ApplicationClient, resourceGroupName, accountName, applicationName string) error {
	_, err := client.Delete(ctx, resourceGroupName, accountName, applicationName, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			log.Printf("Batch application %s not found, skipping deletion", applicationName)
			return nil
		}
		return fmt.Errorf("failed to delete batch application: %w", err)
	}

	log.Printf("Batch application %s deleted successfully", applicationName)
	return nil
}
