package integrationtests

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/msi/armmsi"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources/v2"
	log "github.com/sirupsen/logrus"

	"github.com/overmindtech/cli/go/discovery"
	"github.com/overmindtech/cli/go/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	"github.com/overmindtech/cli/sources/azure/manual"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/shared"
)

const (
	integrationTestIdentityName    = "ovm-integ-test-identity"
	integrationTestFedCredName     = "ovm-integ-test-fed-cred"
	integrationTestFedCredIssuer   = "https://token.actions.githubusercontent.com"
	integrationTestFedCredSubject  = "repo:overmindtech/test-repo:ref:refs/heads/main"
	integrationTestFedCredAudience = "api://AzureADTokenExchange"
)

func TestManagedIdentityFederatedIdentityCredentialIntegration(t *testing.T) {
	subscriptionID := os.Getenv("AZURE_SUBSCRIPTION_ID")
	if subscriptionID == "" {
		t.Skip("AZURE_SUBSCRIPTION_ID environment variable not set")
	}

	cred, err := azureshared.NewAzureCredential(t.Context())
	if err != nil {
		t.Fatalf("Failed to create Azure credential: %v", err)
	}

	uaiClient, err := armmsi.NewUserAssignedIdentitiesClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create User Assigned Identities client: %v", err)
	}

	ficClient, err := armmsi.NewFederatedIdentityCredentialsClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Federated Identity Credentials client: %v", err)
	}

	rgClient, err := armresources.NewResourceGroupsClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Resource Groups client: %v", err)
	}

	var setupCompleted bool

	t.Run("Setup", func(t *testing.T) {
		ctx := t.Context()

		err := createResourceGroup(ctx, rgClient, integrationTestResourceGroup, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create resource group: %v", err)
		}

		err = createUserAssignedIdentity(ctx, uaiClient, integrationTestResourceGroup, integrationTestIdentityName, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create user assigned identity: %v", err)
		}

		err = waitForUserAssignedIdentityAvailable(ctx, uaiClient, integrationTestResourceGroup, integrationTestIdentityName)
		if err != nil {
			t.Fatalf("Failed waiting for user assigned identity to be available: %v", err)
		}

		err = createFederatedIdentityCredential(ctx, ficClient, integrationTestResourceGroup, integrationTestIdentityName, integrationTestFedCredName)
		if err != nil {
			t.Fatalf("Failed to create federated identity credential: %v", err)
		}

		setupCompleted = true
	})

	t.Run("Run", func(t *testing.T) {
		if !setupCompleted {
			t.Skip("Skipping Run: Setup did not complete successfully")
		}

		t.Run("GetFederatedIdentityCredential", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Retrieving federated identity credential %s for identity %s, subscription %s, resource group %s",
				integrationTestFedCredName, integrationTestIdentityName, subscriptionID, integrationTestResourceGroup)

			wrapper := manual.NewManagedIdentityFederatedIdentityCredential(
				clients.NewFederatedIdentityCredentialsClient(ficClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := wrapper.Scopes()[0]

			adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())
			query := shared.CompositeLookupKey(integrationTestIdentityName, integrationTestFedCredName)
			sdpItem, qErr := adapter.Get(ctx, scope, query, true)
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

			expectedUniqueValue := shared.CompositeLookupKey(integrationTestIdentityName, integrationTestFedCredName)
			if uniqueAttrValue != expectedUniqueValue {
				t.Errorf("Expected unique attribute value %s, got %s", expectedUniqueValue, uniqueAttrValue)
			}

			log.Printf("Successfully retrieved federated identity credential %s", integrationTestFedCredName)
		})

		t.Run("SearchFederatedIdentityCredentials", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Searching federated identity credentials for identity %s", integrationTestIdentityName)

			wrapper := manual.NewManagedIdentityFederatedIdentityCredential(
				clients.NewFederatedIdentityCredentialsClient(ficClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := wrapper.Scopes()[0]

			adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

			searchable, ok := adapter.(discovery.SearchableAdapter)
			if !ok {
				t.Fatalf("Adapter does not support Search operation")
			}

			sdpItems, err := searchable.Search(ctx, scope, integrationTestIdentityName, true)
			if err != nil {
				t.Fatalf("Failed to search federated identity credentials: %v", err)
			}

			if len(sdpItems) < 1 {
				t.Fatalf("Expected at least one federated identity credential, got %d", len(sdpItems))
			}

			var found bool
			expectedValue := shared.CompositeLookupKey(integrationTestIdentityName, integrationTestFedCredName)
			for _, item := range sdpItems {
				uniqueAttrKey := item.GetUniqueAttribute()
				if v, err := item.GetAttributes().Get(uniqueAttrKey); err == nil && v == expectedValue {
					found = true
					break
				}
			}

			if !found {
				t.Fatalf("Expected to find credential %s in the search results", integrationTestFedCredName)
			}

			log.Printf("Found %d federated identity credentials in search results", len(sdpItems))
		})

		t.Run("VerifyLinkedItems", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Verifying linked items for federated identity credential %s", integrationTestFedCredName)

			wrapper := manual.NewManagedIdentityFederatedIdentityCredential(
				clients.NewFederatedIdentityCredentialsClient(ficClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := wrapper.Scopes()[0]

			adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())
			query := shared.CompositeLookupKey(integrationTestIdentityName, integrationTestFedCredName)
			sdpItem, qErr := adapter.Get(ctx, scope, query, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			linkedQueries := sdpItem.GetLinkedItemQueries()
			if len(linkedQueries) == 0 {
				t.Fatalf("Expected linked item queries, but got none")
			}

			var hasIdentityLink bool
			var hasDNSLink bool
			for _, liq := range linkedQueries {
				query := liq.GetQuery()
				if query.GetType() == "" {
					t.Error("Linked query has empty Type")
				}
				if query.GetQuery() == "" {
					t.Error("Linked query has empty Query")
				}
				if query.GetScope() == "" {
					t.Error("Linked query has empty Scope")
				}

				if query.GetType() == azureshared.ManagedIdentityUserAssignedIdentity.String() {
					hasIdentityLink = true
					if query.GetQuery() != integrationTestIdentityName {
						t.Errorf("Expected linked query to identity %s, got %s", integrationTestIdentityName, query.GetQuery())
					}
				}

				if query.GetType() == "dns" {
					hasDNSLink = true
					if query.GetQuery() != "token.actions.githubusercontent.com" {
						t.Errorf("Expected DNS query to token.actions.githubusercontent.com, got %s", query.GetQuery())
					}
					if query.GetScope() != "global" {
						t.Errorf("Expected DNS query scope to be global, got %s", query.GetScope())
					}
				}
			}

			if !hasIdentityLink {
				t.Error("Expected linked query to user assigned identity, but didn't find one")
			}

			if !hasDNSLink {
				t.Error("Expected linked query to DNS (from Issuer URL), but didn't find one")
			}

			log.Printf("Verified %d linked item queries for federated identity credential %s", len(linkedQueries), integrationTestFedCredName)
		})

		t.Run("VerifyItemAttributes", func(t *testing.T) {
			ctx := t.Context()

			wrapper := manual.NewManagedIdentityFederatedIdentityCredential(
				clients.NewFederatedIdentityCredentialsClient(ficClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := wrapper.Scopes()[0]

			adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())
			query := shared.CompositeLookupKey(integrationTestIdentityName, integrationTestFedCredName)
			sdpItem, qErr := adapter.Get(ctx, scope, query, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			if sdpItem.GetType() != azureshared.ManagedIdentityFederatedIdentityCredential.String() {
				t.Errorf("Expected type %s, got %s", azureshared.ManagedIdentityFederatedIdentityCredential, sdpItem.GetType())
			}

			expectedScope := subscriptionID + "." + integrationTestResourceGroup
			if sdpItem.GetScope() != expectedScope {
				t.Errorf("Expected scope %s, got %s", expectedScope, sdpItem.GetScope())
			}

			if sdpItem.GetUniqueAttribute() != "uniqueAttr" {
				t.Errorf("Expected unique attribute 'uniqueAttr', got %s", sdpItem.GetUniqueAttribute())
			}

			if err := sdpItem.Validate(); err != nil {
				t.Errorf("Item validation failed: %v", err)
			}

			log.Printf("Verified item attributes for federated identity credential %s", integrationTestFedCredName)
		})
	})

	t.Run("Teardown", func(t *testing.T) {
		ctx := t.Context()

		err := deleteFederatedIdentityCredential(ctx, ficClient, integrationTestResourceGroup, integrationTestIdentityName, integrationTestFedCredName)
		if err != nil {
			t.Fatalf("Failed to delete federated identity credential: %v", err)
		}

		err = deleteUserAssignedIdentity(ctx, uaiClient, integrationTestResourceGroup, integrationTestIdentityName)
		if err != nil {
			t.Fatalf("Failed to delete user assigned identity: %v", err)
		}
	})
}

func createFederatedIdentityCredential(ctx context.Context, client *armmsi.FederatedIdentityCredentialsClient, resourceGroupName, identityName, credentialName string) error {
	_, err := client.Get(ctx, resourceGroupName, identityName, credentialName, nil)
	if err == nil {
		log.Printf("Federated identity credential %s already exists, skipping creation", credentialName)
		return nil
	}

	_, err = client.CreateOrUpdate(ctx, resourceGroupName, identityName, credentialName, armmsi.FederatedIdentityCredential{
		Properties: &armmsi.FederatedIdentityCredentialProperties{
			Issuer:    new(integrationTestFedCredIssuer),
			Subject:   new(integrationTestFedCredSubject),
			Audiences: []*string{new(integrationTestFedCredAudience)},
		},
	}, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusConflict {
			if _, getErr := client.Get(ctx, resourceGroupName, identityName, credentialName, nil); getErr == nil {
				log.Printf("Federated identity credential %s already exists (conflict), skipping creation", credentialName)
				return nil
			}
			return fmt.Errorf("federated identity credential %s conflict but not retrievable: %w", credentialName, err)
		}
		return fmt.Errorf("failed to create federated identity credential: %w", err)
	}

	log.Printf("Federated identity credential %s created successfully", credentialName)
	return nil
}

func deleteFederatedIdentityCredential(ctx context.Context, client *armmsi.FederatedIdentityCredentialsClient, resourceGroupName, identityName, credentialName string) error {
	_, err := client.Delete(ctx, resourceGroupName, identityName, credentialName, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			log.Printf("Federated identity credential %s not found, skipping deletion", credentialName)
			return nil
		}
		return fmt.Errorf("failed to delete federated identity credential: %w", err)
	}

	log.Printf("Federated identity credential %s deleted successfully", credentialName)
	return nil
}
