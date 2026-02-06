package integrationtests

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/keyvault/armkeyvault/v2"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources/v2"
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
	integrationTestSecretName = "ovm-integ-test-secret"
)

func TestKeyVaultSecretIntegration(t *testing.T) {
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

	secretsClient, err := armkeyvault.NewSecretsClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Secrets client: %v", err)
	}

	rgClient, err := armresources.NewResourceGroupsClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Resource Groups client: %v", err)
	}

	// Use the same Key Vault name as the vault integration test
	// Note: integrationTestKeyVaultName is defined in keyvault-vault_test.go
	vaultName := integrationTestKeyVaultName

	t.Run("Setup", func(t *testing.T) {
		ctx := t.Context()

		// Create resource group if it doesn't exist
		err := createResourceGroup(ctx, rgClient, integrationTestResourceGroup, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create resource group: %v", err)
		}

		// Verify Key Vault exists, create if it doesn't
		_, err = keyVaultClient.Get(ctx, integrationTestResourceGroup, vaultName, nil)
		var respErr *azcore.ResponseError
		if err != nil {
			// Check if it's a 404 (not found) - if so, create the vault
			if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
				log.Printf("Key Vault %s does not exist, creating it", vaultName)
				err = createKeyVault(ctx, keyVaultClient, integrationTestResourceGroup, vaultName, integrationTestLocation)
				if err != nil {
					t.Fatalf("Failed to create Key Vault: %v", err)
				}
			} else {
				// Some other error occurred
				t.Fatalf("Failed to check if Key Vault exists: %v", err)
			}
		} else {
			log.Printf("Key Vault %s already exists", vaultName)
		}

		// Wait for Key Vault to be fully available
		err = waitForKeyVaultAvailable(ctx, keyVaultClient, integrationTestResourceGroup, vaultName)
		if err != nil {
			t.Fatalf("Failed waiting for Key Vault to be available: %v", err)
		}

		// Get the Key Vault to retrieve its properties (vault URI)
		vault, err := keyVaultClient.Get(ctx, integrationTestResourceGroup, vaultName, nil)
		if err != nil {
			t.Fatalf("Failed to get Key Vault: %v", err)
		}

		if vault.Properties == nil || vault.Properties.VaultURI == nil {
			t.Fatalf("Key Vault properties or VaultURI is nil")
		}

		// Create secret using Azure CLI (data plane operations require data plane SDK)
		err = createKeyVaultSecret(ctx, vaultName, integrationTestSecretName)
		if err != nil {
			t.Fatalf("Failed to create Key Vault secret: %v", err)
		}
	})

	t.Run("Run", func(t *testing.T) {
		t.Run("GetSecret", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Retrieving secret %s from vault %s in subscription %s, resource group %s",
				integrationTestSecretName, vaultName, subscriptionID, integrationTestResourceGroup)

			secretWrapper := manual.NewKeyVaultSecret(
				clients.NewSecretsClient(secretsClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := secretWrapper.Scopes()[0]

			secretAdapter := sources.WrapperToAdapter(secretWrapper, sdpcache.NewNoOpCache())
			// Get requires vaultName and secretName as query parts
			query := vaultName + shared.QuerySeparator + integrationTestSecretName
			sdpItem, qErr := secretAdapter.Get(ctx, scope, query, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			if sdpItem == nil {
				t.Fatalf("Expected sdpItem to be non-nil")
			}

			uniqueAttrKey := sdpItem.GetUniqueAttribute()
			if uniqueAttrKey != "uniqueAttr" {
				t.Fatalf("Expected unique attribute key to be 'uniqueAttr', got %s", uniqueAttrKey)
			}
			uniqueAttrValue, err := sdpItem.GetAttributes().Get(uniqueAttrKey)
			if err != nil {
				t.Fatalf("Failed to get unique attribute: %v", err)
			}

			expectedUniqueAttrValue := shared.CompositeLookupKey(vaultName, integrationTestSecretName)
			if uniqueAttrValue != expectedUniqueAttrValue {
				t.Fatalf("Expected unique attribute value to be %s, got %s", expectedUniqueAttrValue, uniqueAttrValue)
			}

			// Validate the item
			if err := sdpItem.Validate(); err != nil {
				t.Fatalf("SDP item validation failed: %v", err)
			}

			log.Printf("Successfully retrieved secret %s", integrationTestSecretName)
		})

		t.Run("SearchSecrets", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Searching secrets in vault %s", vaultName)

			secretWrapper := manual.NewKeyVaultSecret(
				clients.NewSecretsClient(secretsClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := secretWrapper.Scopes()[0]

			secretAdapter := sources.WrapperToAdapter(secretWrapper, sdpcache.NewNoOpCache())

			// Check if adapter supports search
			searchable, ok := secretAdapter.(discovery.SearchableAdapter)
			if !ok {
				t.Fatalf("Adapter does not support Search operation")
			}

			sdpItems, err := searchable.Search(ctx, scope, vaultName, true)
			if err != nil {
				t.Fatalf("Failed to search secrets: %v", err)
			}

			if len(sdpItems) < 1 {
				t.Fatalf("Expected at least one secret, got %d", len(sdpItems))
			}

			var found bool
			expectedUniqueAttrValue := shared.CompositeLookupKey(vaultName, integrationTestSecretName)
			for _, item := range sdpItems {
				uniqueAttrKey := item.GetUniqueAttribute()
				if v, err := item.GetAttributes().Get(uniqueAttrKey); err == nil && v == expectedUniqueAttrValue {
					found = true
					break
				}
			}

			if !found {
				t.Fatalf("Expected to find secret %s in the search results", integrationTestSecretName)
			}

			log.Printf("Found %d secrets in search results", len(sdpItems))
		})

		t.Run("VerifyItemAttributes", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Verifying item attributes for secret %s", integrationTestSecretName)

			secretWrapper := manual.NewKeyVaultSecret(
				clients.NewSecretsClient(secretsClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := secretWrapper.Scopes()[0]

			secretAdapter := sources.WrapperToAdapter(secretWrapper, sdpcache.NewNoOpCache())
			query := vaultName + shared.QuerySeparator + integrationTestSecretName
			sdpItem, qErr := secretAdapter.Get(ctx, scope, query, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			// Verify item type
			if sdpItem.GetType() != azureshared.KeyVaultSecret.String() {
				t.Errorf("Expected item type %s, got %s", azureshared.KeyVaultSecret, sdpItem.GetType())
			}

			// Verify scope
			expectedScope := fmt.Sprintf("%s.%s", subscriptionID, integrationTestResourceGroup)
			if sdpItem.GetScope() != expectedScope {
				t.Errorf("Expected scope %s, got %s", expectedScope, sdpItem.GetScope())
			}

			// Verify unique attribute
			if sdpItem.GetUniqueAttribute() != "uniqueAttr" {
				t.Errorf("Expected unique attribute 'uniqueAttr', got %s", sdpItem.GetUniqueAttribute())
			}

			// Verify item validation
			if err := sdpItem.Validate(); err != nil {
				t.Fatalf("Item validation failed: %v", err)
			}

			log.Printf("Verified item attributes for secret %s", integrationTestSecretName)
		})

		t.Run("VerifyLinkedItems", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Verifying linked items for secret %s", integrationTestSecretName)

			secretWrapper := manual.NewKeyVaultSecret(
				clients.NewSecretsClient(secretsClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := secretWrapper.Scopes()[0]

			secretAdapter := sources.WrapperToAdapter(secretWrapper, sdpcache.NewNoOpCache())
			query := vaultName + shared.QuerySeparator + integrationTestSecretName
			sdpItem, qErr := secretAdapter.Get(ctx, scope, query, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			// Verify that linked items exist (Key Vault should be linked)
			linkedQueries := sdpItem.GetLinkedItemQueries()
			if len(linkedQueries) == 0 {
				t.Fatalf("Expected linked item queries, but got none")
			}

			var hasKeyVaultLink bool
			for _, liq := range linkedQueries {
				if liq.GetQuery().GetType() == azureshared.KeyVaultVault.String() {
					hasKeyVaultLink = true
					if liq.GetQuery().GetQuery() != vaultName {
						t.Errorf("Expected linked query to Key Vault %s, got %s", vaultName, liq.GetQuery().GetQuery())
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

			if !hasKeyVaultLink {
				t.Error("Expected linked query to Key Vault, but didn't find one")
			}

			log.Printf("Verified %d linked item queries for secret %s", len(linkedQueries), integrationTestSecretName)
		})
	})

	t.Run("Teardown", func(t *testing.T) {
		ctx := t.Context()

		// Delete secret using Azure CLI
		err := deleteKeyVaultSecret(ctx, vaultName, integrationTestSecretName)
		if err != nil {
			t.Logf("Failed to delete secret: %v", err)
		}

		// Note: We don't delete the Key Vault here as it's shared with keyvault-vault_test.go
		// The Key Vault will be cleaned up by the vault integration test
	})
}

// createKeyVaultSecret creates a Key Vault secret using Azure CLI (idempotent)
func createKeyVaultSecret(ctx context.Context, vaultName, secretName string) error {
	// Check if secret already exists
	cmd := exec.CommandContext(ctx, "az", "keyvault", "secret", "show",
		"--vault-name", vaultName,
		"--name", secretName)
	err := cmd.Run()
	if err == nil {
		log.Printf("Secret %s already exists, skipping creation", secretName)
		return nil
	}

	// Create the secret
	cmd = exec.CommandContext(ctx, "az", "keyvault", "secret", "set",
		"--vault-name", vaultName,
		"--name", secretName,
		"--value", "test-secret-value")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// If the command failed, it might be because the secret already exists
		// Try to show it to confirm
		showCmd := exec.CommandContext(ctx, "az", "keyvault", "secret", "show",
			"--vault-name", vaultName,
			"--name", secretName)
		if showCmd.Run() == nil {
			log.Printf("Secret %s already exists, skipping creation", secretName)
			return nil
		}
		return fmt.Errorf("failed to create secret: %w, output: %s", err, string(output))
	}

	log.Printf("Secret %s created successfully", secretName)
	return nil
}

// deleteKeyVaultSecret deletes a Key Vault secret using Azure CLI (idempotent)
func deleteKeyVaultSecret(ctx context.Context, vaultName, secretName string) error {
	// Check if secret exists first
	showCmd := exec.CommandContext(ctx, "az", "keyvault", "secret", "show",
		"--vault-name", vaultName,
		"--name", secretName)
	showErr := showCmd.Run()
	if showErr != nil {
		// Secret doesn't exist, which is fine - nothing to delete
		// We intentionally ignore showErr here as it indicates the secret doesn't exist
		log.Printf("Secret %s does not exist, skipping deletion", secretName)
		return nil //nolint:nilerr // Returning nil is correct when secret doesn't exist
	}

	// Secret exists, try to delete it
	cmd := exec.CommandContext(ctx, "az", "keyvault", "secret", "delete",
		"--vault-name", vaultName,
		"--name", secretName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to delete secret: %w, output: %s", err, string(output))
	}

	log.Printf("Secret %s deleted successfully", secretName)
	return nil
}
