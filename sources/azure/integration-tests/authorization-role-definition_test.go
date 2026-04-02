package integrationtests

import (
	"fmt"
	"os"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v3"
	log "github.com/sirupsen/logrus"

	"github.com/overmindtech/cli/go/discovery"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	"github.com/overmindtech/cli/sources/azure/manual"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
)

func TestAuthorizationRoleDefinitionIntegration(t *testing.T) {
	subscriptionID := os.Getenv("AZURE_SUBSCRIPTION_ID")
	if subscriptionID == "" {
		t.Skip("AZURE_SUBSCRIPTION_ID environment variable not set")
	}

	cred, err := azureshared.NewAzureCredential(t.Context())
	if err != nil {
		t.Fatalf("Failed to create Azure credential: %v", err)
	}

	roleDefinitionsClient, err := armauthorization.NewRoleDefinitionsClient(cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Role Definitions client: %v", err)
	}

	// Use a built-in role definition ID that always exists: "Reader"
	// The Reader role ID is the same across all Azure subscriptions
	readerRoleDefinitionID := "acdd72a7-3385-48ef-bd42-f606fba81ae7"

	t.Run("Setup", func(t *testing.T) {
		// No setup required for role definitions - they are built-in Azure resources
		log.Printf("Using built-in Reader role definition ID: %s", readerRoleDefinitionID)
	})

	t.Run("Run", func(t *testing.T) {
		t.Run("GetRoleDefinition", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Retrieving role definition %s", readerRoleDefinitionID)

			wrapper := manual.NewAuthorizationRoleDefinition(
				clients.NewRoleDefinitionsClient(roleDefinitionsClient),
				subscriptionID,
			)
			scope := wrapper.Scopes()[0]

			adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())
			sdpItem, qErr := adapter.Get(ctx, scope, readerRoleDefinitionID, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			if sdpItem == nil {
				t.Fatalf("Expected sdpItem to be non-nil")
			}

			if sdpItem.GetType() != azureshared.AuthorizationRoleDefinition.String() {
				t.Errorf("Expected type %s, got %s", azureshared.AuthorizationRoleDefinition.String(), sdpItem.GetType())
			}

			uniqueAttrKey := sdpItem.GetUniqueAttribute()
			if uniqueAttrKey != "name" {
				t.Errorf("Expected unique attribute 'name', got %s", uniqueAttrKey)
			}

			uniqueAttrValue, err := sdpItem.GetAttributes().Get(uniqueAttrKey)
			if err != nil {
				t.Fatalf("Failed to get unique attribute: %v", err)
			}

			if uniqueAttrValue != readerRoleDefinitionID {
				t.Errorf("Expected unique attribute value %s, got %s", readerRoleDefinitionID, uniqueAttrValue)
			}

			if sdpItem.GetScope() != subscriptionID {
				t.Errorf("Expected scope %s, got %s", subscriptionID, sdpItem.GetScope())
			}

			if err := sdpItem.Validate(); err != nil {
				t.Fatalf("Item validation failed: %v", err)
			}

			log.Printf("Successfully retrieved role definition %s", readerRoleDefinitionID)
		})

		t.Run("ListRoleDefinitions", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Listing role definitions in subscription %s", subscriptionID)

			wrapper := manual.NewAuthorizationRoleDefinition(
				clients.NewRoleDefinitionsClient(roleDefinitionsClient),
				subscriptionID,
			)
			scope := wrapper.Scopes()[0]

			adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

			listable, ok := adapter.(discovery.ListableAdapter)
			if !ok {
				t.Fatalf("Adapter does not support List operation")
			}

			sdpItems, err := listable.List(ctx, scope, true)
			if err != nil {
				t.Fatalf("Failed to list role definitions: %v", err)
			}

			// Azure has many built-in role definitions, expect at least a few
			if len(sdpItems) < 5 {
				t.Fatalf("Expected at least 5 role definitions, got %d", len(sdpItems))
			}

			var found bool
			for _, item := range sdpItems {
				uniqueAttrKey := item.GetUniqueAttribute()
				if v, err := item.GetAttributes().Get(uniqueAttrKey); err == nil {
					if v == readerRoleDefinitionID {
						found = true
						break
					}
				}
			}

			if !found {
				t.Fatalf("Expected to find Reader role definition %s in the list results", readerRoleDefinitionID)
			}

			log.Printf("Found %d role definitions in list results", len(sdpItems))
		})

		t.Run("VerifyItemAttributes", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Verifying item attributes for role definition %s", readerRoleDefinitionID)

			wrapper := manual.NewAuthorizationRoleDefinition(
				clients.NewRoleDefinitionsClient(roleDefinitionsClient),
				subscriptionID,
			)
			scope := wrapper.Scopes()[0]

			adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())
			sdpItem, qErr := adapter.Get(ctx, scope, readerRoleDefinitionID, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			// Verify item type
			if sdpItem.GetType() != azureshared.AuthorizationRoleDefinition.String() {
				t.Errorf("Expected item type %s, got %s", azureshared.AuthorizationRoleDefinition.String(), sdpItem.GetType())
			}

			// Verify scope
			if sdpItem.GetScope() != subscriptionID {
				t.Errorf("Expected scope %s, got %s", subscriptionID, sdpItem.GetScope())
			}

			// Verify unique attribute
			if sdpItem.GetUniqueAttribute() != "name" {
				t.Errorf("Expected unique attribute 'name', got %s", sdpItem.GetUniqueAttribute())
			}

			// Verify item validation
			if err := sdpItem.Validate(); err != nil {
				t.Fatalf("Item validation failed: %v", err)
			}

			// Verify role name is Reader
			roleName, err := sdpItem.GetAttributes().Get("properties.roleName")
			if err != nil {
				t.Logf("Warning: Could not get roleName attribute: %v", err)
			} else if roleName != "Reader" {
				t.Errorf("Expected role name 'Reader', got %s", roleName)
			}

			log.Printf("Verified item attributes for role definition %s", readerRoleDefinitionID)
		})

		t.Run("VerifyLinkedItems", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Verifying linked items for role definition %s", readerRoleDefinitionID)

			wrapper := manual.NewAuthorizationRoleDefinition(
				clients.NewRoleDefinitionsClient(roleDefinitionsClient),
				subscriptionID,
			)
			scope := wrapper.Scopes()[0]

			adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())
			sdpItem, qErr := adapter.Get(ctx, scope, readerRoleDefinitionID, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			// Role definitions link to AssignableScopes (subscriptions and resource groups)
			// The built-in Reader role has "/" as its assignable scope, which may not produce links
			// Custom roles would have specific subscription/resource group scopes
			linkedQueries := sdpItem.GetLinkedItemQueries()

			// Verify each linked query has proper attributes
			for _, linkedQuery := range linkedQueries {
				query := linkedQuery.GetQuery()
				if query.GetType() == "" {
					t.Error("Linked item query has empty Type")
				}
				if query.GetMethod() != sdp.QueryMethod_GET && query.GetMethod() != sdp.QueryMethod_SEARCH {
					t.Errorf("Linked item query has invalid Method: %v", query.GetMethod())
				}
				if query.GetQuery() == "" {
					t.Error("Linked item query has empty Query")
				}
				if query.GetScope() == "" {
					t.Error("Linked item query has empty Scope")
				}

				// Verify linked types are either subscription or resource group
				validTypes := map[string]bool{
					azureshared.ResourcesSubscription.String():  true,
					azureshared.ResourcesResourceGroup.String(): true,
				}
				if !validTypes[query.GetType()] {
					t.Errorf("Unexpected linked item type: %s", query.GetType())
				}
			}

			log.Printf("Verified linked items for role definition %s (found %d linked queries)", readerRoleDefinitionID, len(linkedQueries))
		})

		t.Run("VerifyBuiltInRoles", func(t *testing.T) {
			ctx := t.Context()

			// Verify some well-known built-in role definitions exist
			builtInRoles := map[string]string{
				"acdd72a7-3385-48ef-bd42-f606fba81ae7": "Reader",
				"b24988ac-6180-42a0-ab88-20f7382dd24c": "Contributor",
				"8e3af657-a8ff-443c-a75c-2fe8c4bcb635": "Owner",
			}

			wrapper := manual.NewAuthorizationRoleDefinition(
				clients.NewRoleDefinitionsClient(roleDefinitionsClient),
				subscriptionID,
			)
			scope := wrapper.Scopes()[0]
			adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

			for roleID, roleName := range builtInRoles {
				t.Run(fmt.Sprintf("Get%sRole", roleName), func(t *testing.T) {
					sdpItem, qErr := adapter.Get(ctx, scope, roleID, true)
					if qErr != nil {
						t.Fatalf("Failed to get %s role definition: %v", roleName, qErr)
					}

					if sdpItem == nil {
						t.Fatalf("Expected %s role definition to be non-nil", roleName)
					}

					actualRoleName, err := sdpItem.GetAttributes().Get("properties.roleName")
					if err != nil {
						t.Logf("Warning: Could not get roleName attribute for %s: %v", roleName, err)
					} else if actualRoleName != roleName {
						t.Errorf("Expected role name '%s', got '%s'", roleName, actualRoleName)
					}

					log.Printf("Successfully verified built-in role: %s (ID: %s)", roleName, roleID)
				})
			}
		})
	})

	t.Run("Teardown", func(t *testing.T) {
		// No teardown required - role definitions are built-in Azure resources
		log.Printf("No teardown required for role definitions (built-in Azure resources)")
	})
}
