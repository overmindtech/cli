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
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/keyvault/armkeyvault/v2"
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
	integrationTestKeyVaultName = "ovm-integ-test-kv"
)

func TestKeyVaultVaultIntegration(t *testing.T) {
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
	keyVaultClient, err := armkeyvault.NewVaultsClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Key Vault client: %v", err)
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

		// Create Key Vault
		err = createKeyVault(ctx, keyVaultClient, integrationTestResourceGroup, integrationTestKeyVaultName, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create Key Vault: %v", err)
		}

		// Wait for Key Vault to be fully available
		err = waitForKeyVaultAvailable(ctx, keyVaultClient, integrationTestResourceGroup, integrationTestKeyVaultName)
		if err != nil {
			t.Fatalf("Failed waiting for Key Vault to be available: %v", err)
		}
	})

	t.Run("Run", func(t *testing.T) {
		t.Run("GetKeyVault", func(t *testing.T) {
			ctx := t.Context()

			// Try to get the test vault, skip if it doesn't exist
			_, err := keyVaultClient.Get(ctx, integrationTestResourceGroup, integrationTestKeyVaultName, nil)
			if err != nil {
				t.Skipf("Key Vault %s does not exist in resource group %s, skipping test. Error: %v", integrationTestKeyVaultName, integrationTestResourceGroup, err)
			}

			log.Printf("Retrieving Key Vault %s in subscription %s, resource group %s",
				integrationTestKeyVaultName, subscriptionID, integrationTestResourceGroup)

			kvWrapper := manual.NewKeyVaultVault(
				clients.NewVaultsClient(keyVaultClient),
				subscriptionID,
				integrationTestResourceGroup,
			)
			scope := kvWrapper.Scopes()[0]

			kvAdapter := sources.WrapperToAdapter(kvWrapper, sdpcache.NewNoOpCache())
			sdpItem, qErr := kvAdapter.Get(ctx, scope, integrationTestKeyVaultName, true)
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

			if uniqueAttrValue != integrationTestKeyVaultName {
				t.Fatalf("Expected unique attribute value to be %s, got %s", integrationTestKeyVaultName, uniqueAttrValue)
			}

			// Validate the item
			if err := sdpItem.Validate(); err != nil {
				t.Fatalf("SDP item validation failed: %v", err)
			}

			log.Printf("Successfully retrieved Key Vault %s", integrationTestKeyVaultName)
		})

		t.Run("ListKeyVaults", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Listing Key Vaults in subscription %s, resource group %s",
				subscriptionID, integrationTestResourceGroup)

			kvWrapper := manual.NewKeyVaultVault(
				clients.NewVaultsClient(keyVaultClient),
				subscriptionID,
				integrationTestResourceGroup,
			)
			scope := kvWrapper.Scopes()[0]

			kvAdapter := sources.WrapperToAdapter(kvWrapper, sdpcache.NewNoOpCache())

			// Check if adapter supports listing
			listable, ok := kvAdapter.(discovery.ListableAdapter)
			if !ok {
				t.Fatalf("Adapter does not support List operation")
			}

			sdpItems, err := listable.List(ctx, scope, true)
			if err != nil {
				t.Fatalf("Failed to list Key Vaults: %v", err)
			}

			// Note: len(sdpItems) can be 0 or more, which is valid
			if len(sdpItems) == 0 {
				log.Printf("No Key Vaults found in resource group %s", integrationTestResourceGroup)
			}

			// Validate all items
			for _, item := range sdpItems {
				if err := item.Validate(); err != nil {
					t.Fatalf("SDP item validation failed: %v", err)
				}
			}

			log.Printf("Successfully listed %d Key Vaults", len(sdpItems))
		})
	})

	t.Run("Teardown", func(t *testing.T) {
		// We intentionally keep the Key Vault by default.
		//
		// Key Vault names are globally unique and (by default) soft-deleted on removal.
		// Deleting the vault in tests frequently causes subsequent runs to fail because the
		// name is still held by the soft-deleted vault, and recreating requires a purge.
		//
		// To opt into cleanup, set CLEANUP_AZURE_INTEGRATION_TESTS=true.
		if os.Getenv("CLEANUP_AZURE_INTEGRATION_TESTS") == "true" {
			ctx := t.Context()
			err := deleteKeyVault(ctx, keyVaultClient, integrationTestResourceGroup, integrationTestKeyVaultName)
			if err != nil {
				t.Fatalf("Failed to delete Key Vault: %v", err)
			}
		} else {
			log.Printf("Skipping Key Vault deletion (set CLEANUP_AZURE_INTEGRATION_TESTS=true to enable)")
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

// createKeyVault creates an Azure Key Vault (idempotent)
func createKeyVault(ctx context.Context, client *armkeyvault.VaultsClient, resourceGroupName, vaultName, location string) error {
	// Check if Key Vault already exists
	_, err := client.Get(ctx, resourceGroupName, vaultName, nil)
	if err == nil {
		log.Printf("Key Vault %s already exists, skipping creation", vaultName)
		return nil
	}

	// Get the tenant ID from environment variable
	tenantID := os.Getenv("AZURE_TENANT_ID")
	if tenantID == "" {
		return fmt.Errorf("AZURE_TENANT_ID environment variable not set, required for Key Vault creation")
	}

	// Create a context with timeout for the entire Key Vault creation operation
	// Key Vault creation can hang if the Microsoft.KeyVault resource provider is not registered
	createCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	// Create the Key Vault.
	// Key Vault names must be globally unique and 3-24 characters
	// They can only contain alphanumeric characters and hyphens
	params := armkeyvault.VaultCreateOrUpdateParameters{
		Location: ptr.To(location),
		Properties: &armkeyvault.VaultProperties{
			TenantID: ptr.To(tenantID),
			SKU: &armkeyvault.SKU{
				Family: ptr.To(armkeyvault.SKUFamilyA),
				Name:   ptr.To(armkeyvault.SKUNameStandard),
			},
			AccessPolicies: []*armkeyvault.AccessPolicyEntry{
				// For integration tests, we create with minimal configuration.
			},
		},
		Tags: map[string]*string{
			"purpose": ptr.To("overmind-integration-tests"),
			"test":    ptr.To("keyvault-vault"),
		},
	}

	// We allow one remediation pass for the common failure mode:
	// the vault was soft-deleted previously, so the name is still held and create returns 409.
	for attempt := 1; attempt <= 2; attempt++ {
		poller, err := client.BeginCreateOrUpdate(createCtx, resourceGroupName, vaultName, params, nil)
		if err != nil {
			// Check if context timed out
			if errors.Is(err, context.DeadlineExceeded) {
				return fmt.Errorf("timeout starting Key Vault creation (this may indicate the Microsoft.KeyVault resource provider is not registered or the operation is taking too long): %w", err)
			}

			var respErr *azcore.ResponseError
			if errors.As(err, &respErr) && respErr.StatusCode == http.StatusConflict {
				// Key Vault uses soft-delete by default; 409 commonly means a deleted vault is still holding the name.
				// Attempt to purge the deleted vault (if it exists) and retry creation.
				if attempt == 1 {
					if purgeErr := purgeSoftDeletedKeyVault(createCtx, client, vaultName, location); purgeErr != nil {
						return fmt.Errorf("key vault name conflict for %s and purge failed: %w", vaultName, purgeErr)
					}
					continue
				}
				return fmt.Errorf("key vault name conflict for %s (it may be soft-deleted and require purge before recreate): %w", vaultName, err)
			}

			return fmt.Errorf("failed to begin creating Key Vault: %w", err)
		}

		// Use the same timeout context for polling
		resp, err := poller.PollUntilDone(createCtx, nil)
		if err != nil {
			// Check if context timed out
			if errors.Is(err, context.DeadlineExceeded) {
				return fmt.Errorf("timeout waiting for Key Vault creation to complete (this may indicate the Microsoft.KeyVault resource provider is not registered): %w", err)
			}
			return fmt.Errorf("failed to create Key Vault: %w", err)
		}

		// Verify the Key Vault was created successfully
		if resp.Properties == nil {
			return fmt.Errorf("Key Vault created but properties are nil")
		}

		log.Printf("Key Vault %s created successfully", vaultName)
		return nil
	}

	return fmt.Errorf("failed to create Key Vault %s: exhausted retries", vaultName)
}

// waitForKeyVaultAvailable waits for a Key Vault to be fully available
func waitForKeyVaultAvailable(ctx context.Context, client *armkeyvault.VaultsClient, resourceGroupName, vaultName string) error {
	maxAttempts := 20
	pollInterval := 10 * time.Second

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		resp, err := client.Get(ctx, resourceGroupName, vaultName, nil)
		if err != nil {
			var respErr *azcore.ResponseError
			if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
				log.Printf("Key Vault %s not yet available (attempt %d/%d), waiting %v...", vaultName, attempt, maxAttempts, pollInterval)
				time.Sleep(pollInterval)
				continue
			}
			return fmt.Errorf("failed to get Key Vault: %w", err)
		}

		// Key Vaults don't have a provisioning state like other resources
		// If we can get the vault, it's available
		if resp.Properties != nil {
			log.Printf("Key Vault %s is available", vaultName)
			return nil
		}

		log.Printf("Waiting for Key Vault %s to be available (attempt %d/%d)", vaultName, attempt, maxAttempts)
		time.Sleep(pollInterval)
	}

	return fmt.Errorf("Key Vault %s did not become available within the timeout period", vaultName)
}

func purgeSoftDeletedKeyVault(ctx context.Context, client *armkeyvault.VaultsClient, vaultName, location string) error {
	// Check if the vault is soft-deleted (this is the usual reason for 409 conflicts).
	_, err := client.GetDeleted(ctx, vaultName, location, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			// The caller may have passed the wrong location. Try to locate the deleted vault and
			// determine its original location.
			pager := client.NewListDeletedPager(nil)
			for pager.More() {
				page, pageErr := pager.NextPage(ctx)
				if pageErr != nil {
					return fmt.Errorf("failed to list deleted vaults while resolving conflict for %s: %w", vaultName, pageErr)
				}
				for _, v := range page.Value {
					if v == nil || v.Name == nil || *v.Name != vaultName {
						continue
					}
					if v.Properties != nil && v.Properties.Location != nil && *v.Properties.Location != "" {
						location = *v.Properties.Location
						log.Printf("Found soft-deleted Key Vault %s in location %s via ListDeleted", vaultName, location)
						goto purge
					}
					// If we can't determine location, we still can't purge with the SDK.
					return fmt.Errorf("soft-deleted Key Vault %s found but location was empty; cannot purge automatically", vaultName)
				}
			}

			// Not a soft-deleted vault in this subscription (or not visible); the name may be held elsewhere.
			return fmt.Errorf("vault name %s is not soft-deleted in subscription/location (may be held by another subscription/tenant): %w", vaultName, err)
		}
		return fmt.Errorf("failed to check deleted Key Vault %s in %s: %w", vaultName, location, err)
	}

purge:
	log.Printf("Key Vault %s is soft-deleted in %s; purging to allow recreation", vaultName, location)
	poller, err := client.BeginPurgeDeleted(ctx, vaultName, location, nil)
	if err != nil {
		return fmt.Errorf("failed to begin purging soft-deleted Key Vault %s: %w", vaultName, err)
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to purge soft-deleted Key Vault %s: %w", vaultName, err)
	}

	log.Printf("Soft-deleted Key Vault %s purged successfully", vaultName)
	return nil
}

// deleteKeyVault deletes an Azure Key Vault (idempotent)
func deleteKeyVault(ctx context.Context, client *armkeyvault.VaultsClient, resourceGroupName, vaultName string) error {
	// Check if Key Vault exists
	_, err := client.Get(ctx, resourceGroupName, vaultName, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			log.Printf("Key Vault %s does not exist, skipping deletion", vaultName)
			return nil
		}
		return fmt.Errorf("failed to check if Key Vault exists: %w", err)
	}

	// Delete the Key Vault
	// Note: Key Vaults may require soft-delete to be disabled first
	// For integration tests, we'll attempt deletion
	_, err = client.Delete(ctx, resourceGroupName, vaultName, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			log.Printf("Key Vault %s does not exist, skipping deletion", vaultName)
			return nil
		}
		return fmt.Errorf("failed to delete Key Vault: %w", err)
	}

	log.Printf("Key Vault %s deleted successfully", vaultName)
	return nil
}
