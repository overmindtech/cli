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
)

const (
	integrationTestUserAssignedIdentityName = "ovm-integ-test-uai"
)

func TestManagedIdentityUserAssignedIdentityIntegration(t *testing.T) {
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
	identityClient, err := armmsi.NewUserAssignedIdentitiesClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create User Assigned Identities client: %v", err)
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

		// Create User Assigned Identity
		err = createUserAssignedIdentity(ctx, identityClient, integrationTestResourceGroup, integrationTestUserAssignedIdentityName, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create User Assigned Identity: %v", err)
		}

		// Wait for User Assigned Identity to be fully available
		err = waitForUserAssignedIdentityAvailable(ctx, identityClient, integrationTestResourceGroup, integrationTestUserAssignedIdentityName)
		if err != nil {
			t.Fatalf("Failed waiting for User Assigned Identity to be available: %v", err)
		}
	})

	t.Run("Run", func(t *testing.T) {
		t.Run("GetUserAssignedIdentity", func(t *testing.T) {
			ctx := t.Context()

			// Try to get the test identity, skip if it doesn't exist
			_, err := identityClient.Get(ctx, integrationTestResourceGroup, integrationTestUserAssignedIdentityName, nil)
			if err != nil {
				t.Skipf("User Assigned Identity %s does not exist in resource group %s, skipping test. Error: %v", integrationTestUserAssignedIdentityName, integrationTestResourceGroup, err)
			}

			log.Printf("Retrieving User Assigned Identity %s in subscription %s, resource group %s",
				integrationTestUserAssignedIdentityName, subscriptionID, integrationTestResourceGroup)

			identityWrapper := manual.NewManagedIdentityUserAssignedIdentity(
				clients.NewUserAssignedIdentitiesClient(identityClient),
				subscriptionID,
				integrationTestResourceGroup,
			)
			scope := identityWrapper.Scopes()[0]

			identityAdapter := sources.WrapperToAdapter(identityWrapper, sdpcache.NewNoOpCache())
			sdpItem, qErr := identityAdapter.Get(ctx, scope, integrationTestUserAssignedIdentityName, true)
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

			if uniqueAttrValue != integrationTestUserAssignedIdentityName {
				t.Fatalf("Expected unique attribute value to be %s, got %s", integrationTestUserAssignedIdentityName, uniqueAttrValue)
			}

			// Validate the item
			if err := sdpItem.Validate(); err != nil {
				t.Fatalf("SDP item validation failed: %v", err)
			}

			log.Printf("Successfully retrieved User Assigned Identity %s", integrationTestUserAssignedIdentityName)
		})

		t.Run("ListUserAssignedIdentities", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Listing User Assigned Identities in subscription %s, resource group %s",
				subscriptionID, integrationTestResourceGroup)

			identityWrapper := manual.NewManagedIdentityUserAssignedIdentity(
				clients.NewUserAssignedIdentitiesClient(identityClient),
				subscriptionID,
				integrationTestResourceGroup,
			)
			scope := identityWrapper.Scopes()[0]

			identityAdapter := sources.WrapperToAdapter(identityWrapper, sdpcache.NewNoOpCache())

			// Check if adapter supports listing
			listable, ok := identityAdapter.(discovery.ListableAdapter)
			if !ok {
				t.Fatalf("Adapter does not support List operation")
			}

			sdpItems, err := listable.List(ctx, scope, true)
			if err != nil {
				t.Fatalf("Failed to list User Assigned Identities: %v", err)
			}

			// Note: len(sdpItems) can be 0 or more, which is valid
			if len(sdpItems) == 0 {
				log.Printf("No User Assigned Identities found in resource group %s", integrationTestResourceGroup)
			}

			// Validate all items
			for _, item := range sdpItems {
				if err := item.Validate(); err != nil {
					t.Fatalf("SDP item validation failed: %v", err)
				}
			}

			// Verify we can find the test identity in the list
			var found bool
			for _, item := range sdpItems {
				uniqueAttrKey := item.GetUniqueAttribute()
				if v, err := item.GetAttributes().Get(uniqueAttrKey); err == nil && v == integrationTestUserAssignedIdentityName {
					found = true
					break
				}
			}

			if !found {
				t.Fatalf("Expected to find identity %s in the list of User Assigned Identities", integrationTestUserAssignedIdentityName)
			}

			log.Printf("Successfully listed %d User Assigned Identities", len(sdpItems))
		})

		t.Run("VerifyLinkedItems", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Verifying linked items for User Assigned Identity %s", integrationTestUserAssignedIdentityName)

			identityWrapper := manual.NewManagedIdentityUserAssignedIdentity(
				clients.NewUserAssignedIdentitiesClient(identityClient),
				subscriptionID,
				integrationTestResourceGroup,
			)
			scope := identityWrapper.Scopes()[0]

			identityAdapter := sources.WrapperToAdapter(identityWrapper, sdpcache.NewNoOpCache())
			sdpItem, qErr := identityAdapter.Get(ctx, scope, integrationTestUserAssignedIdentityName, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			// Verify that linked items exist (federated identity credentials should be linked)
			linkedQueries := sdpItem.GetLinkedItemQueries()
			if len(linkedQueries) == 0 {
				t.Fatalf("Expected linked item queries, but got none")
			}

			var hasFederatedCredentialLink bool
			for _, liq := range linkedQueries {
				switch liq.GetQuery().GetType() {
				case azureshared.ManagedIdentityFederatedIdentityCredential.String():
					hasFederatedCredentialLink = true
					// Verify federated credential link properties
					if liq.GetQuery().GetMethod() != sdp.QueryMethod_SEARCH {
						t.Errorf("Expected federated credential link method to be SEARCH, got %s", liq.GetQuery().GetMethod())
					}
					if liq.GetQuery().GetQuery() != integrationTestUserAssignedIdentityName {
						t.Errorf("Expected federated credential link query to be %s, got %s", integrationTestUserAssignedIdentityName, liq.GetQuery().GetQuery())
					}
					if liq.GetQuery().GetScope() != scope {
						t.Errorf("Expected federated credential link scope to be %s, got %s", scope, liq.GetQuery().GetScope())
					}
					// Verify blast propagation (In: true, Out: true)
					if liq.GetBlastPropagation().GetIn() != true {
						t.Error("Expected federated credential blast propagation In=true, got false")
					}
					if liq.GetBlastPropagation().GetOut() != true {
						t.Error("Expected federated credential blast propagation Out=true, got false")
					}
				default:
					t.Errorf("Unexpected linked item type: %s", liq.GetQuery().GetType())
				}
			}

			if !hasFederatedCredentialLink {
				t.Error("Expected linked query to federated identity credentials, but didn't find one")
			}

			log.Printf("Verified %d linked item queries for User Assigned Identity %s", len(linkedQueries), integrationTestUserAssignedIdentityName)
		})
	})

	t.Run("Teardown", func(t *testing.T) {
		ctx := t.Context()

		// Delete User Assigned Identity
		err := deleteUserAssignedIdentity(ctx, identityClient, integrationTestResourceGroup, integrationTestUserAssignedIdentityName)
		if err != nil {
			t.Fatalf("Failed to delete User Assigned Identity: %v", err)
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

// createUserAssignedIdentity creates an Azure User Assigned Identity (idempotent)
func createUserAssignedIdentity(ctx context.Context, client *armmsi.UserAssignedIdentitiesClient, resourceGroupName, identityName, location string) error {
	// Check if User Assigned Identity already exists
	_, err := client.Get(ctx, resourceGroupName, identityName, nil)
	if err == nil {
		log.Printf("User Assigned Identity %s already exists, skipping creation", identityName)
		return nil
	}

	// Create the User Assigned Identity
	resp, err := client.CreateOrUpdate(ctx, resourceGroupName, identityName, armmsi.Identity{
		Location: ptr.To(location),
		Tags: map[string]*string{
			"purpose": ptr.To("overmind-integration-tests"),
			"test":    ptr.To("managedidentity-user-assigned-identity"),
		},
	}, nil)
	if err != nil {
		// Check if identity already exists (conflict)
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusConflict {
			log.Printf("User Assigned Identity %s already exists (conflict), skipping creation", identityName)
			return nil
		}
		return fmt.Errorf("failed to create User Assigned Identity: %w", err)
	}

	// Verify the identity was created successfully
	if resp.Properties == nil {
		return fmt.Errorf("User Assigned Identity created but properties are nil")
	}

	log.Printf("User Assigned Identity %s created successfully", identityName)
	return nil
}

// waitForUserAssignedIdentityAvailable waits for a User Assigned Identity to be fully available
func waitForUserAssignedIdentityAvailable(ctx context.Context, client *armmsi.UserAssignedIdentitiesClient, resourceGroupName, identityName string) error {
	maxAttempts := 20
	pollInterval := 10 * time.Second

	log.Printf("Waiting for User Assigned Identity %s to be available...", identityName)

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		resp, err := client.Get(ctx, resourceGroupName, identityName, nil)
		if err != nil {
			var respErr *azcore.ResponseError
			if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
				log.Printf("User Assigned Identity %s not yet available (attempt %d/%d), waiting %v...", identityName, attempt, maxAttempts, pollInterval)
				time.Sleep(pollInterval)
				continue
			}
			return fmt.Errorf("error checking User Assigned Identity availability: %w", err)
		}

		// User Assigned Identities don't have a provisioning state like some other resources
		// If we can get the identity and it has properties, it's available
		if resp.Properties != nil {
			log.Printf("User Assigned Identity %s is available", identityName)
			return nil
		}

		log.Printf("Waiting for User Assigned Identity %s to be available (attempt %d/%d)", identityName, attempt, maxAttempts)
		time.Sleep(pollInterval)
	}

	return fmt.Errorf("timeout waiting for User Assigned Identity %s to be available after %d attempts", identityName, maxAttempts)
}

// deleteUserAssignedIdentity deletes an Azure User Assigned Identity (idempotent)
func deleteUserAssignedIdentity(ctx context.Context, client *armmsi.UserAssignedIdentitiesClient, resourceGroupName, identityName string) error {
	// Check if User Assigned Identity exists
	_, err := client.Get(ctx, resourceGroupName, identityName, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			log.Printf("User Assigned Identity %s does not exist, skipping deletion", identityName)
			return nil
		}
		return fmt.Errorf("failed to check if User Assigned Identity exists: %w", err)
	}

	// Delete the User Assigned Identity
	_, err = client.Delete(ctx, resourceGroupName, identityName, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			log.Printf("User Assigned Identity %s does not exist, skipping deletion", identityName)
			return nil
		}
		return fmt.Errorf("failed to delete User Assigned Identity: %w", err)
	}

	log.Printf("User Assigned Identity %s deleted successfully", identityName)
	return nil
}
