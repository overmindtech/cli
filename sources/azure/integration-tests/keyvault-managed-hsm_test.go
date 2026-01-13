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
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/keyvault/armkeyvault"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources/v2"
	log "github.com/sirupsen/logrus"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	"github.com/overmindtech/cli/sources/azure/manual"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
)

const (
	integrationTestManagedHSMName = "ovm-integ-test-hsm"
)

func TestKeyVaultManagedHSMIntegration(t *testing.T) {
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
	managedHSMClient, err := armkeyvault.NewManagedHsmsClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Managed HSM client: %v", err)
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

		// Check if Managed HSM already exists first (quick check)
		existingHSM, err := managedHSMClient.Get(ctx, integrationTestResourceGroup, integrationTestManagedHSMName, nil)
		if err == nil {
			// Managed HSM exists, check if it's ready
			if existingHSM.Properties != nil && existingHSM.Properties.ProvisioningState != nil {
				state := *existingHSM.Properties.ProvisioningState
				if state == "Succeeded" {
					log.Printf("Managed HSM %s already exists and is ready, skipping creation", integrationTestManagedHSMName)
				} else {
					log.Printf("Managed HSM %s exists but in state %s, waiting for it to be ready", integrationTestManagedHSMName, state)
					err = waitForManagedHSMAvailable(ctx, managedHSMClient, integrationTestResourceGroup, integrationTestManagedHSMName)
					if err != nil {
						t.Fatalf("Failed waiting for existing Managed HSM to be ready: %v", err)
					}
				}
			} else {
				log.Printf("Managed HSM %s already exists, verifying availability", integrationTestManagedHSMName)
				err = waitForManagedHSMAvailable(ctx, managedHSMClient, integrationTestResourceGroup, integrationTestManagedHSMName)
				if err != nil {
					t.Fatalf("Failed waiting for Managed HSM to be available: %v", err)
				}
			}
			log.Printf("Setup completed: Managed HSM %s is available", integrationTestManagedHSMName)
		} else {
			// Managed HSM doesn't exist
			// Managed HSM creation takes 30-60 minutes which exceeds test timeout
			// For integration tests, we require the Managed HSM to already exist
			// However, we don't skip the entire test suite - individual tests will skip if needed
			log.Printf("Managed HSM %s does not exist", integrationTestManagedHSMName)
			log.Printf("Managed HSM creation takes 30-60 minutes, which exceeds the test timeout of 5 minutes.")
			log.Printf("Please create the Managed HSM manually or wait for a previous creation to complete.")
			log.Printf("Note: Managed HSMs are only available in specific regions (e.g., eastus2, westus2, westeurope)")
			log.Printf("Tests that require the Managed HSM will be skipped, but ListManagedHSMs will still run.")
		}
	})

	t.Run("Run", func(t *testing.T) {
		t.Run("GetManagedHSM", func(t *testing.T) {
			ctx := t.Context()

			// Try to get the test Managed HSM, skip if it doesn't exist
			_, err := managedHSMClient.Get(ctx, integrationTestResourceGroup, integrationTestManagedHSMName, nil)
			if err != nil {
				t.Skipf("Managed HSM %s does not exist in resource group %s, skipping test. Error: %v", integrationTestManagedHSMName, integrationTestResourceGroup, err)
			}

			log.Printf("Retrieving Managed HSM %s in subscription %s, resource group %s",
				integrationTestManagedHSMName, subscriptionID, integrationTestResourceGroup)

			hsmWrapper := manual.NewKeyVaultManagedHSM(
				clients.NewManagedHSMsClient(managedHSMClient),
				subscriptionID,
				integrationTestResourceGroup,
			)
			scope := hsmWrapper.Scopes()[0]

			hsmAdapter := sources.WrapperToAdapter(hsmWrapper, sdpcache.NewNoOpCache())
			sdpItem, qErr := hsmAdapter.Get(ctx, scope, integrationTestManagedHSMName, true)
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

			if uniqueAttrValue != integrationTestManagedHSMName {
				t.Fatalf("Expected unique attribute value to be %s, got %s", integrationTestManagedHSMName, uniqueAttrValue)
			}

			// Validate the item
			if err := sdpItem.Validate(); err != nil {
				t.Fatalf("SDP item validation failed: %v", err)
			}

			log.Printf("Successfully retrieved Managed HSM %s", integrationTestManagedHSMName)
		})

		t.Run("ListManagedHSMs", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Listing Managed HSMs in subscription %s, resource group %s",
				subscriptionID, integrationTestResourceGroup)

			hsmWrapper := manual.NewKeyVaultManagedHSM(
				clients.NewManagedHSMsClient(managedHSMClient),
				subscriptionID,
				integrationTestResourceGroup,
			)
			scope := hsmWrapper.Scopes()[0]

			hsmAdapter := sources.WrapperToAdapter(hsmWrapper, sdpcache.NewNoOpCache())

			// Check if adapter supports listing
			listable, ok := hsmAdapter.(discovery.ListableAdapter)
			if !ok {
				t.Fatalf("Adapter does not support List operation")
			}

			sdpItems, err := listable.List(ctx, scope, true)
			if err != nil {
				t.Fatalf("Failed to list Managed HSMs: %v", err)
			}

			// Note: len(sdpItems) can be 0 or more, which is valid
			if len(sdpItems) == 0 {
				log.Printf("No Managed HSMs found in resource group %s", integrationTestResourceGroup)
			}

			// Validate all items
			for _, item := range sdpItems {
				if err := item.Validate(); err != nil {
					t.Fatalf("SDP item validation failed: %v", err)
				}
			}

			log.Printf("Successfully listed %d Managed HSMs", len(sdpItems))
		})

		t.Run("VerifyItemAttributes", func(t *testing.T) {
			ctx := t.Context()

			// Try to get the test Managed HSM, skip if it doesn't exist
			_, err := managedHSMClient.Get(ctx, integrationTestResourceGroup, integrationTestManagedHSMName, nil)
			if err != nil {
				t.Skipf("Managed HSM %s does not exist in resource group %s, skipping test. Error: %v", integrationTestManagedHSMName, integrationTestResourceGroup, err)
			}

			hsmWrapper := manual.NewKeyVaultManagedHSM(
				clients.NewManagedHSMsClient(managedHSMClient),
				subscriptionID,
				integrationTestResourceGroup,
			)
			scope := hsmWrapper.Scopes()[0]

			hsmAdapter := sources.WrapperToAdapter(hsmWrapper, sdpcache.NewNoOpCache())
			sdpItem, qErr := hsmAdapter.Get(ctx, scope, integrationTestManagedHSMName, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			// Verify item type
			if sdpItem.GetType() != azureshared.KeyVaultManagedHSM.String() {
				t.Errorf("Expected type %s, got %s", azureshared.KeyVaultManagedHSM, sdpItem.GetType())
			}

			// Verify scope
			expectedScope := fmt.Sprintf("%s.%s", subscriptionID, integrationTestResourceGroup)
			if sdpItem.GetScope() != expectedScope {
				t.Errorf("Expected scope %s, got %s", expectedScope, sdpItem.GetScope())
			}

			// Verify unique attribute
			if sdpItem.GetUniqueAttribute() != "name" {
				t.Errorf("Expected unique attribute 'name', got %s", sdpItem.GetUniqueAttribute())
			}

			// Verify item validation
			if err := sdpItem.Validate(); err != nil {
				t.Fatalf("Item validation failed: %v", err)
			}

			log.Printf("Verified item attributes for Managed HSM %s", integrationTestManagedHSMName)
		})

		t.Run("VerifyLinkedItems", func(t *testing.T) {
			ctx := t.Context()

			// Try to get the test Managed HSM, skip if it doesn't exist
			_, err := managedHSMClient.Get(ctx, integrationTestResourceGroup, integrationTestManagedHSMName, nil)
			if err != nil {
				t.Skipf("Managed HSM %s does not exist in resource group %s, skipping test. Error: %v", integrationTestManagedHSMName, integrationTestResourceGroup, err)
			}

			hsmWrapper := manual.NewKeyVaultManagedHSM(
				clients.NewManagedHSMsClient(managedHSMClient),
				subscriptionID,
				integrationTestResourceGroup,
			)
			scope := hsmWrapper.Scopes()[0]

			hsmAdapter := sources.WrapperToAdapter(hsmWrapper, sdpcache.NewNoOpCache())
			sdpItem, qErr := hsmAdapter.Get(ctx, scope, integrationTestManagedHSMName, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			// Verify that linked items exist (if any)
			linkedQueries := sdpItem.GetLinkedItemQueries()
			if len(linkedQueries) == 0 {
				log.Printf("No linked item queries found for Managed HSM %s (this is valid if the HSM has no private endpoints, network ACLs, or managed identities)", integrationTestManagedHSMName)
			}

			// Verify expected linked item types for Managed HSM
			expectedLinkedTypes := map[string]bool{
				azureshared.NetworkPrivateEndpoint.String():              false,
				azureshared.NetworkSubnet.String():                       false,
				azureshared.ManagedIdentityUserAssignedIdentity.String(): false,
			}

			for _, liq := range linkedQueries {
				query := liq.GetQuery()
				if query == nil {
					t.Error("Linked item query has nil Query")
					continue
				}

				linkedType := query.GetType()
				if _, exists := expectedLinkedTypes[linkedType]; exists {
					expectedLinkedTypes[linkedType] = true
				}

				// Verify query has required fields
				if query.GetType() == "" {
					t.Error("Linked item query has empty Type")
				}
				if query.GetMethod() != sdp.QueryMethod_GET && query.GetMethod() != sdp.QueryMethod_SEARCH {
					t.Errorf("Linked item query has unexpected Method: %v", query.GetMethod())
				}
				if query.GetQuery() == "" {
					t.Error("Linked item query has empty Query")
				}
				if query.GetScope() == "" {
					t.Error("Linked item query has empty Scope")
				}

				// Verify blast propagation is set
				bp := liq.GetBlastPropagation()
				if bp == nil {
					t.Error("Linked item query has nil BlastPropagation")
				} else {
					// Blast propagation should have In and Out set (even if false)
					_ = bp.GetIn()
					_ = bp.GetOut()
				}
			}

			log.Printf("Verified %d linked item queries for Managed HSM %s", len(linkedQueries), integrationTestManagedHSMName)
		})
	})

	t.Run("Teardown", func(t *testing.T) {
		// Optionally delete the Managed HSM
		// Note: We keep the Managed HSM for faster subsequent test runs since creation takes 30-60 minutes
		// The Setup phase instructs users to pre-create the Managed HSM manually, so we don't delete it here
		// Uncomment the following if you want to clean up completely:
		// ctx := t.Context()
		// err := deleteManagedHSM(ctx, managedHSMClient, integrationTestResourceGroup, integrationTestManagedHSMName)
		// if err != nil {
		//     t.Fatalf("Failed to delete Managed HSM: %v", err)
		// }

		// Optionally delete the resource group
		// Note: We keep the resource group for faster subsequent test runs
		// Uncomment the following if you want to clean up completely:
		// err = deleteResourceGroup(ctx, rgClient, integrationTestResourceGroup)
		// if err != nil {
		//     t.Fatalf("Failed to delete resource group: %v", err)
		// }
	})
}

// waitForManagedHSMAvailable waits for a Managed HSM to be fully available
func waitForManagedHSMAvailable(ctx context.Context, client *armkeyvault.ManagedHsmsClient, resourceGroupName, hsmName string) error {
	maxAttempts := 30
	pollInterval := 10 * time.Second

	log.Printf("Waiting for Managed HSM %s to be available via API...", hsmName)

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		resp, err := client.Get(ctx, resourceGroupName, hsmName, nil)
		if err != nil {
			var respErr *azcore.ResponseError
			if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
				log.Printf("Managed HSM %s not yet available (attempt %d/%d), waiting %v...", hsmName, attempt, maxAttempts, pollInterval)
				time.Sleep(pollInterval)
				continue
			}
			return fmt.Errorf("error checking Managed HSM availability: %w", err)
		}

		// Check provisioning state
		if resp.Properties != nil && resp.Properties.ProvisioningState != nil {
			state := *resp.Properties.ProvisioningState
			if state == "Succeeded" {
				log.Printf("Managed HSM %s is available with provisioning state: %s", hsmName, state)
				return nil
			}
			if state == "Failed" {
				return fmt.Errorf("Managed HSM provisioning failed with state: %s", state)
			}
			// Still provisioning, wait and retry
			log.Printf("Managed HSM %s provisioning state: %s (attempt %d/%d), waiting...", hsmName, state, attempt, maxAttempts)
			time.Sleep(pollInterval)
			continue
		}

		// Managed HSM exists but no provisioning state - consider it available
		log.Printf("Managed HSM %s is available", hsmName)
		return nil
	}

	return fmt.Errorf("timeout waiting for Managed HSM %s to be available after %d attempts", hsmName, maxAttempts)
}

// deleteManagedHSM deletes an Azure Managed HSM (idempotent)
// This function is kept for potential use when uncommenting the teardown deletion code
//
//nolint:unused // Intentionally kept for optional teardown cleanup
func deleteManagedHSM(ctx context.Context, client *armkeyvault.ManagedHsmsClient, resourceGroupName, hsmName string) error {
	// Check if Managed HSM exists
	_, err := client.Get(ctx, resourceGroupName, hsmName, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			log.Printf("Managed HSM %s does not exist, skipping deletion", hsmName)
			return nil
		}
		return fmt.Errorf("failed to check if Managed HSM exists: %w", err)
	}

	// Delete the Managed HSM
	// Note: Managed HSMs may require purging after deletion if soft-delete is enabled
	// For integration tests, we'll attempt deletion
	poller, err := client.BeginDelete(ctx, resourceGroupName, hsmName, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			log.Printf("Managed HSM %s does not exist, skipping deletion", hsmName)
			return nil
		}
		return fmt.Errorf("failed to begin deleting Managed HSM: %w", err)
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to delete Managed HSM: %w", err)
	}

	log.Printf("Managed HSM %s deleted successfully", hsmName)
	return nil
}
