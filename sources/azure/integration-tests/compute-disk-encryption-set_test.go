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
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/keyvault/armkeyvault"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/msi/armmsi"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources/v2"
	log "github.com/sirupsen/logrus"
	"k8s.io/utils/ptr"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	"github.com/overmindtech/cli/sources/azure/manual"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/shared"
)

const (
	integrationTestDiskEncryptionSetName = "ovm-integ-test-des"
	integrationTestKeyVaultKeyName       = "ovm-integ-test-des-key"
)

func TestComputeDiskEncryptionSetIntegration(t *testing.T) {
	subscriptionID := os.Getenv("AZURE_SUBSCRIPTION_ID")
	if subscriptionID == "" {
		t.Skip("AZURE_SUBSCRIPTION_ID environment variable not set")
	}

	cred, err := azureshared.NewAzureCredential(t.Context())
	if err != nil {
		t.Fatalf("Failed to create Azure credential: %v", err)
	}

	desClient, err := armcompute.NewDiskEncryptionSetsClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Disk Encryption Sets client: %v", err)
	}

	identityClient, err := armmsi.NewUserAssignedIdentitiesClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create User Assigned Identities client: %v", err)
	}

	keyVaultClient, err := armkeyvault.NewVaultsClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Key Vault client: %v", err)
	}

	rgClient, err := armresources.NewResourceGroupsClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Resource Groups client: %v", err)
	}

	var vaultID string
	var keyURL string
	var identityResourceID string
	var identityPrincipalID string

	t.Run("Setup", func(t *testing.T) {
		ctx := t.Context()

		// Create RG if needed
		if err := createResourceGroup(ctx, rgClient, integrationTestResourceGroup, integrationTestLocation); err != nil {
			t.Fatalf("Failed to create resource group: %v", err)
		}

		// Ensure a Key Vault exists (shared with other tests)
		if err := createKeyVault(ctx, keyVaultClient, integrationTestResourceGroup, integrationTestKeyVaultName, integrationTestLocation); err != nil {
			t.Fatalf("Failed to create Key Vault: %v", err)
		}
		if err := waitForKeyVaultAvailable(ctx, keyVaultClient, integrationTestResourceGroup, integrationTestKeyVaultName); err != nil {
			t.Fatalf("Failed waiting for Key Vault to be available: %v", err)
		}
		vault, err := keyVaultClient.Get(ctx, integrationTestResourceGroup, integrationTestKeyVaultName, nil)
		if err != nil {
			t.Fatalf("Failed to get Key Vault: %v", err)
		}
		if vault.ID == nil || *vault.ID == "" {
			t.Fatalf("Key Vault ID is nil/empty")
		}
		vaultID = *vault.ID

		// Ensure a user-assigned identity exists (shared with other tests)
		if err := createUserAssignedIdentity(ctx, identityClient, integrationTestResourceGroup, integrationTestUserAssignedIdentityName, integrationTestLocation); err != nil {
			t.Fatalf("Failed to create User Assigned Identity: %v", err)
		}
		if err := waitForUserAssignedIdentityAvailable(ctx, identityClient, integrationTestResourceGroup, integrationTestUserAssignedIdentityName); err != nil {
			t.Fatalf("Failed waiting for User Assigned Identity to be available: %v", err)
		}

		identity, err := identityClient.Get(ctx, integrationTestResourceGroup, integrationTestUserAssignedIdentityName, nil)
		if err != nil {
			t.Fatalf("Failed to get User Assigned Identity: %v", err)
		}
		if identity.ID == nil || *identity.ID == "" {
			t.Fatalf("User Assigned Identity ID is nil/empty")
		}
		if identity.Properties == nil || identity.Properties.PrincipalID == nil || *identity.Properties.PrincipalID == "" {
			t.Fatalf("User Assigned Identity principalID is nil/empty")
		}
		identityResourceID = *identity.ID
		identityPrincipalID = *identity.Properties.PrincipalID

		// Ensure a Key Vault key exists (data-plane via Azure CLI).
		keyURL, err = ensureKeyVaultKey(ctx, integrationTestKeyVaultName, integrationTestKeyVaultKeyName)
		if err != nil {
			t.Fatalf("Failed to ensure Key Vault key: %v", err)
		}

		// Grant the identity access to the Key Vault key material. Different vaults may be configured
		// for access-policy or RBAC authorization, so we try both approaches.
		if err := grantKeyVaultCryptoAccess(ctx, integrationTestKeyVaultName, vaultID, identityPrincipalID); err != nil {
			t.Fatalf("Failed to grant Key Vault crypto access to identity: %v", err)
		}

		// Create DES (idempotent) and wait for it to be available.
		if err := createDiskEncryptionSet(ctx, desClient, integrationTestResourceGroup, integrationTestDiskEncryptionSetName, integrationTestLocation, vaultID, keyURL, identityResourceID); err != nil {
			t.Fatalf("Failed to create Disk Encryption Set: %v", err)
		}
		if err := waitForDiskEncryptionSetAvailable(ctx, desClient, integrationTestResourceGroup, integrationTestDiskEncryptionSetName); err != nil {
			t.Fatalf("Failed waiting for Disk Encryption Set to be available: %v", err)
		}
	})

	t.Run("Run", func(t *testing.T) {
		t.Run("GetDiskEncryptionSet", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Retrieving disk encryption set %s in subscription %s, resource group %s",
				integrationTestDiskEncryptionSetName, subscriptionID, integrationTestResourceGroup)

			desWrapper := manual.NewComputeDiskEncryptionSet(
				clients.NewDiskEncryptionSetsClient(desClient),
				subscriptionID,
				integrationTestResourceGroup,
			)
			scope := desWrapper.Scopes()[0]

			desAdapter := sources.WrapperToAdapter(desWrapper, sdpcache.NewNoOpCache())
			sdpItem, qErr := desAdapter.Get(ctx, scope, integrationTestDiskEncryptionSetName, true)
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
			if uniqueAttrValue != integrationTestDiskEncryptionSetName {
				t.Fatalf("Expected unique attribute value to be %s, got %s", integrationTestDiskEncryptionSetName, uniqueAttrValue)
			}

			if err := sdpItem.Validate(); err != nil {
				t.Fatalf("SDP item validation failed: %v", err)
			}
		})

		t.Run("ListDiskEncryptionSets", func(t *testing.T) {
			ctx := t.Context()

			desWrapper := manual.NewComputeDiskEncryptionSet(
				clients.NewDiskEncryptionSetsClient(desClient),
				subscriptionID,
				integrationTestResourceGroup,
			)
			scope := desWrapper.Scopes()[0]

			desAdapter := sources.WrapperToAdapter(desWrapper, sdpcache.NewNoOpCache())

			listable, ok := desAdapter.(discovery.ListableAdapter)
			if !ok {
				t.Fatalf("Adapter does not support List operation")
			}

			sdpItems, err := listable.List(ctx, scope, true)
			if err != nil {
				t.Fatalf("Failed to list Disk Encryption Sets: %v", err)
			}
			if len(sdpItems) < 1 {
				t.Fatalf("Expected at least one Disk Encryption Set, got %d", len(sdpItems))
			}

			var found bool
			for _, item := range sdpItems {
				uniqueAttrKey := item.GetUniqueAttribute()
				if v, err := item.GetAttributes().Get(uniqueAttrKey); err == nil && v == integrationTestDiskEncryptionSetName {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("Expected to find Disk Encryption Set %s in the list", integrationTestDiskEncryptionSetName)
			}
		})

		t.Run("VerifyItemAttributes", func(t *testing.T) {
			ctx := t.Context()

			desWrapper := manual.NewComputeDiskEncryptionSet(
				clients.NewDiskEncryptionSetsClient(desClient),
				subscriptionID,
				integrationTestResourceGroup,
			)
			scope := desWrapper.Scopes()[0]

			desAdapter := sources.WrapperToAdapter(desWrapper, sdpcache.NewNoOpCache())
			sdpItem, qErr := desAdapter.Get(ctx, scope, integrationTestDiskEncryptionSetName, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			if sdpItem.GetType() != azureshared.ComputeDiskEncryptionSet.String() {
				t.Errorf("Expected item type %s, got %s", azureshared.ComputeDiskEncryptionSet, sdpItem.GetType())
			}
			expectedScope := fmt.Sprintf("%s.%s", subscriptionID, integrationTestResourceGroup)
			if sdpItem.GetScope() != expectedScope {
				t.Errorf("Expected scope %s, got %s", expectedScope, sdpItem.GetScope())
			}
			if sdpItem.GetUniqueAttribute() != "name" {
				t.Errorf("Expected unique attribute 'name', got %s", sdpItem.GetUniqueAttribute())
			}
			if err := sdpItem.Validate(); err != nil {
				t.Fatalf("Item validation failed: %v", err)
			}
		})

		t.Run("VerifyLinkedItems", func(t *testing.T) {
			ctx := t.Context()

			desWrapper := manual.NewComputeDiskEncryptionSet(
				clients.NewDiskEncryptionSetsClient(desClient),
				subscriptionID,
				integrationTestResourceGroup,
			)
			scope := desWrapper.Scopes()[0]

			desAdapter := sources.WrapperToAdapter(desWrapper, sdpcache.NewNoOpCache())
			sdpItem, qErr := desAdapter.Get(ctx, scope, integrationTestDiskEncryptionSetName, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			linkedQueries := sdpItem.GetLinkedItemQueries()
			if len(linkedQueries) == 0 {
				t.Fatalf("Expected linked item queries, but got none")
			}

			var hasKeyVaultLink bool
			var hasKeyVaultKeyLink bool
			var hasUserAssignedIdentityLink bool
			var hasDNSLink bool

			for _, liq := range linkedQueries {
				query := liq.GetQuery()
				if query == nil {
					t.Error("Linked item query has nil Query")
					continue
				}

				switch query.GetType() {
				case azureshared.KeyVaultVault.String():
					hasKeyVaultLink = true
					if query.GetMethod() != sdp.QueryMethod_GET {
						t.Errorf("Expected Key Vault link method GET, got %s", query.GetMethod())
					}
					if query.GetQuery() != integrationTestKeyVaultName {
						t.Errorf("Expected Key Vault link query %s, got %s", integrationTestKeyVaultName, query.GetQuery())
					}
					if query.GetScope() != scope {
						t.Errorf("Expected Key Vault link scope %s, got %s", scope, query.GetScope())
					}
					if liq.GetBlastPropagation() == nil {
						t.Error("Key Vault linked item query has nil BlastPropagation")
					} else {
						if liq.GetBlastPropagation().GetIn() != true {
							t.Error("Expected Key Vault BlastPropagation.In to be true")
						}
						if liq.GetBlastPropagation().GetOut() != false {
							t.Error("Expected Key Vault BlastPropagation.Out to be false")
						}
					}
				case azureshared.KeyVaultKey.String():
					hasKeyVaultKeyLink = true
					if query.GetMethod() != sdp.QueryMethod_GET {
						t.Errorf("Expected Key Vault Key link method GET, got %s", query.GetMethod())
					}
					if query.GetQuery() != shared.CompositeLookupKey(integrationTestKeyVaultName, integrationTestKeyVaultKeyName) {
						t.Errorf("Expected Key Vault Key link query %s, got %s", shared.CompositeLookupKey(integrationTestKeyVaultName, integrationTestKeyVaultKeyName), query.GetQuery())
					}
					// Key Vault URI doesn't contain resource group, adapter uses DES scope as best effort
					if query.GetScope() != scope {
						t.Errorf("Expected Key Vault Key link scope %s, got %s", scope, query.GetScope())
					}
					if liq.GetBlastPropagation() == nil {
						t.Error("Key Vault Key linked item query has nil BlastPropagation")
					} else {
						if liq.GetBlastPropagation().GetIn() != true {
							t.Error("Expected Key Vault Key BlastPropagation.In to be true")
						}
						if liq.GetBlastPropagation().GetOut() != false {
							t.Error("Expected Key Vault Key BlastPropagation.Out to be false")
						}
					}
				case azureshared.ManagedIdentityUserAssignedIdentity.String():
					hasUserAssignedIdentityLink = true
					if query.GetMethod() != sdp.QueryMethod_GET {
						t.Errorf("Expected User Assigned Identity link method GET, got %s", query.GetMethod())
					}
					if query.GetQuery() != integrationTestUserAssignedIdentityName {
						t.Errorf("Expected User Assigned Identity link query %s, got %s", integrationTestUserAssignedIdentityName, query.GetQuery())
					}
					if query.GetScope() != scope {
						t.Errorf("Expected User Assigned Identity link scope %s, got %s", scope, query.GetScope())
					}
					if liq.GetBlastPropagation() == nil {
						t.Error("User Assigned Identity linked item query has nil BlastPropagation")
					} else {
						if liq.GetBlastPropagation().GetIn() != true {
							t.Error("Expected User Assigned Identity BlastPropagation.In to be true")
						}
						if liq.GetBlastPropagation().GetOut() != false {
							t.Error("Expected User Assigned Identity BlastPropagation.Out to be false")
						}
					}
				case "dns":
					hasDNSLink = true
					if query.GetMethod() != sdp.QueryMethod_SEARCH {
						t.Errorf("Expected DNS link method SEARCH, got %s", query.GetMethod())
					}
					expectedDNS := azureshared.ExtractDNSFromURL(keyURL)
					if query.GetQuery() != expectedDNS {
						t.Errorf("Expected DNS link query %s, got %s", expectedDNS, query.GetQuery())
					}
					if query.GetScope() != "global" {
						t.Errorf("Expected DNS link scope global, got %s", query.GetScope())
					}
					if liq.GetBlastPropagation() == nil {
						t.Error("DNS linked item query has nil BlastPropagation")
					} else {
						if liq.GetBlastPropagation().GetIn() != true {
							t.Error("Expected DNS BlastPropagation.In to be true")
						}
						if liq.GetBlastPropagation().GetOut() != true {
							t.Error("Expected DNS BlastPropagation.Out to be true")
						}
					}
				default:
					t.Errorf("Unexpected linked item type: %s", query.GetType())
				}
			}

			if !hasKeyVaultLink {
				t.Error("Expected linked query to Key Vault, but didn't find one")
			}
			if !hasUserAssignedIdentityLink {
				t.Error("Expected linked query to User Assigned Identity, but didn't find one")
			}
			if !hasKeyVaultKeyLink {
				t.Error("Expected linked query to Key Vault Key, but didn't find one")
			}
			if !hasDNSLink {
				t.Error("Expected linked query to DNS, but didn't find one")
			}
		})
	})

	t.Run("Teardown", func(t *testing.T) {
		ctx := t.Context()
		if err := deleteDiskEncryptionSet(ctx, desClient, integrationTestResourceGroup, integrationTestDiskEncryptionSetName); err != nil {
			t.Fatalf("Failed to delete Disk Encryption Set: %v", err)
		}
	})
}

func createDiskEncryptionSet(ctx context.Context, client *armcompute.DiskEncryptionSetsClient, resourceGroupName, desName, location, vaultID, keyURL, userAssignedIdentityResourceID string) error {
	// If it exists and is succeeded, skip creation.
	existing, err := client.Get(ctx, resourceGroupName, desName, nil)
	if err == nil {
		if existing.Properties != nil && existing.Properties.ProvisioningState != nil && *existing.Properties.ProvisioningState == "Succeeded" {
			log.Printf("Disk Encryption Set %s already exists and is ready, skipping creation", desName)
			return nil
		}
		log.Printf("Disk Encryption Set %s already exists, will wait for it to be ready", desName)
		return nil
	}

	// New DES creation.
	des := armcompute.DiskEncryptionSet{
		Location: ptr.To(location),
		Identity: &armcompute.EncryptionSetIdentity{
			Type: ptr.To(armcompute.DiskEncryptionSetIdentityTypeUserAssigned),
			UserAssignedIdentities: map[string]*armcompute.UserAssignedIdentitiesValue{
				userAssignedIdentityResourceID: &armcompute.UserAssignedIdentitiesValue{},
			},
		},
		Properties: &armcompute.EncryptionSetProperties{
			EncryptionType: ptr.To(armcompute.DiskEncryptionSetTypeEncryptionAtRestWithCustomerKey),
			ActiveKey: &armcompute.KeyForDiskEncryptionSet{
				KeyURL: ptr.To(keyURL),
				SourceVault: &armcompute.SourceVault{
					ID: ptr.To(vaultID),
				},
			},
			RotationToLatestKeyVersionEnabled: ptr.To(false),
		},
		Tags: map[string]*string{
			"purpose": ptr.To("overmind-integration-tests"),
			"test":    ptr.To("compute-disk-encryption-set"),
		},
	}

	// DES creation can fail briefly due to RBAC propagation; retry a few times.
	var lastErr error
	for attempt := 1; attempt <= 6; attempt++ {
		poller, err := client.BeginCreateOrUpdate(ctx, resourceGroupName, desName, des, nil)
		if err != nil {
			lastErr = err
		} else {
			_, err = poller.PollUntilDone(ctx, nil)
			if err == nil {
				log.Printf("Disk Encryption Set %s created", desName)
				return nil
			}
			lastErr = err
		}

		log.Printf("Disk Encryption Set create attempt %d/6 failed: %v; retrying...", attempt, lastErr)
		time.Sleep(time.Duration(attempt) * 10 * time.Second)
	}

	return fmt.Errorf("failed to create Disk Encryption Set after retries: %w", lastErr)
}

func waitForDiskEncryptionSetAvailable(ctx context.Context, client *armcompute.DiskEncryptionSetsClient, resourceGroupName, desName string) error {
	maxAttempts := 30
	pollInterval := 10 * time.Second

	log.Printf("Waiting for Disk Encryption Set %s to be available via API...", desName)
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		resp, err := client.Get(ctx, resourceGroupName, desName, nil)
		if err != nil {
			var respErr *azcore.ResponseError
			if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
				time.Sleep(pollInterval)
				continue
			}
			return fmt.Errorf("error checking Disk Encryption Set availability: %w", err)
		}

		if resp.Properties != nil && resp.Properties.ProvisioningState != nil {
			state := *resp.Properties.ProvisioningState
			if state == "Succeeded" {
				log.Printf("Disk Encryption Set %s is available with provisioning state: %s", desName, state)
				return nil
			}
			if state == "Failed" {
				return fmt.Errorf("disk encryption set provisioning failed with state: %s", state)
			}
		}

		time.Sleep(pollInterval)
	}

	return fmt.Errorf("timeout waiting for Disk Encryption Set %s to be available after %d attempts", desName, maxAttempts)
}

func deleteDiskEncryptionSet(ctx context.Context, client *armcompute.DiskEncryptionSetsClient, resourceGroupName, desName string) error {
	poller, err := client.BeginDelete(ctx, resourceGroupName, desName, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			log.Printf("Disk Encryption Set %s not found, skipping deletion", desName)
			return nil
		}
		return fmt.Errorf("failed to begin deleting Disk Encryption Set: %w", err)
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to delete Disk Encryption Set: %w", err)
	}

	log.Printf("Disk Encryption Set %s deleted successfully", desName)
	return nil
}
