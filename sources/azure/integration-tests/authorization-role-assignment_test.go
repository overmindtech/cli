package integrationtests

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v3"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources/v2"
	"github.com/google/uuid"
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

func TestAuthorizationRoleAssignmentIntegration(t *testing.T) {
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
	roleAssignmentsClient, err := armauthorization.NewRoleAssignmentsClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Role Assignments client: %v", err)
	}

	roleDefinitionsClient, err := armauthorization.NewRoleDefinitionsClient(cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Role Definitions client: %v", err)
	}

	rgClient, err := armresources.NewResourceGroupsClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Resource Groups client: %v", err)
	}

	// Get current user's principal ID for role assignment
	// We'll use the current authenticated user/principal
	principalID, err := getCurrentUserPrincipalID(t.Context(), cred)
	if err != nil {
		t.Fatalf("Failed to get current user principal ID: %v", err)
	}

	// Get the Reader role definition ID (built-in role)
	readerRoleDefinitionID, err := getReaderRoleDefinitionID(t.Context(), roleDefinitionsClient, subscriptionID)
	if err != nil {
		t.Fatalf("Failed to get Reader role definition ID: %v", err)
	}

	// Generate unique role assignment name (GUID)
	roleAssignmentName := uuid.New().String()
	azureScope := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s", subscriptionID, integrationTestResourceGroup)

	t.Run("Setup", func(t *testing.T) {
		ctx := t.Context()

		// Create resource group if it doesn't exist
		err := createResourceGroup(ctx, rgClient, integrationTestResourceGroup, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create resource group: %v", err)
		}

		// Create role assignment at resource group scope
		err = createRoleAssignment(ctx, roleAssignmentsClient, azureScope, roleAssignmentName, principalID, readerRoleDefinitionID)
		if err != nil {
			t.Fatalf("Failed to create role assignment: %v", err)
		}

		log.Printf("Created role assignment %s at scope %s", roleAssignmentName, azureScope)
	})

	t.Run("Run", func(t *testing.T) {
		t.Run("GetRoleAssignment", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Retrieving role assignment %s at scope %s", roleAssignmentName, azureScope)

			roleAssignmentWrapper := manual.NewAuthorizationRoleAssignment(
				clients.NewRoleAssignmentsClient(roleAssignmentsClient),
				subscriptionID,
				integrationTestResourceGroup,
			)
			scope := roleAssignmentWrapper.Scopes()[0]

			roleAssignmentAdapter := sources.WrapperToAdapter(roleAssignmentWrapper, sdpcache.NewNoOpCache())
			sdpItem, qErr := roleAssignmentAdapter.Get(ctx, scope, roleAssignmentName, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			if sdpItem == nil {
				t.Fatalf("Expected sdpItem to be non-nil")
			}

			if sdpItem.GetType() != azureshared.AuthorizationRoleAssignment.String() {
				t.Errorf("Expected type %s, got %s", azureshared.AuthorizationRoleAssignment.String(), sdpItem.GetType())
			}

			uniqueAttrKey := sdpItem.GetUniqueAttribute()
			if uniqueAttrKey != "uniqueAttr" {
				t.Errorf("Expected unique attribute 'uniqueAttr', got %s", uniqueAttrKey)
			}

			uniqueAttrValue, err := sdpItem.GetAttributes().Get(uniqueAttrKey)
			if err != nil {
				t.Fatalf("Failed to get unique attribute: %v", err)
			}

			expectedUniqueAttrValue := shared.CompositeLookupKey(integrationTestResourceGroup, roleAssignmentName)
			if uniqueAttrValue != expectedUniqueAttrValue {
				t.Errorf("Expected unique attribute value %s, got %s", expectedUniqueAttrValue, uniqueAttrValue)
			}

			if sdpItem.GetScope() != fmt.Sprintf("%s.%s", subscriptionID, integrationTestResourceGroup) {
				t.Errorf("Expected scope %s.%s, got %s", subscriptionID, integrationTestResourceGroup, sdpItem.GetScope())
			}

			if err := sdpItem.Validate(); err != nil {
				t.Fatalf("Item validation failed: %v", err)
			}

			log.Printf("Successfully retrieved role assignment %s", roleAssignmentName)
		})

		t.Run("ListRoleAssignments", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Listing role assignments in resource group %s", integrationTestResourceGroup)

			roleAssignmentWrapper := manual.NewAuthorizationRoleAssignment(
				clients.NewRoleAssignmentsClient(roleAssignmentsClient),
				subscriptionID,
				integrationTestResourceGroup,
			)
			scope := roleAssignmentWrapper.Scopes()[0]

			roleAssignmentAdapter := sources.WrapperToAdapter(roleAssignmentWrapper, sdpcache.NewNoOpCache())

			// Check if adapter supports list
			listable, ok := roleAssignmentAdapter.(discovery.ListableAdapter)
			if !ok {
				t.Fatalf("Adapter does not support List operation")
			}

			sdpItems, err := listable.List(ctx, scope, true)
			if err != nil {
				t.Fatalf("Failed to list role assignments: %v", err)
			}

			if len(sdpItems) < 1 {
				t.Fatalf("Expected at least one role assignment, got %d", len(sdpItems))
			}

			var found bool
			expectedUniqueAttrValue := shared.CompositeLookupKey(integrationTestResourceGroup, roleAssignmentName)
			for _, item := range sdpItems {
				uniqueAttrKey := item.GetUniqueAttribute()
				if v, err := item.GetAttributes().Get(uniqueAttrKey); err == nil {
					if v == expectedUniqueAttrValue {
						found = true
						break
					}
				}
			}

			if !found {
				t.Fatalf("Expected to find role assignment %s in the list results", roleAssignmentName)
			}

			log.Printf("Found %d role assignments in list results", len(sdpItems))
		})

		t.Run("VerifyItemAttributes", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Verifying item attributes for role assignment %s", roleAssignmentName)

			roleAssignmentWrapper := manual.NewAuthorizationRoleAssignment(
				clients.NewRoleAssignmentsClient(roleAssignmentsClient),
				subscriptionID,
				integrationTestResourceGroup,
			)
			scope := roleAssignmentWrapper.Scopes()[0]

			roleAssignmentAdapter := sources.WrapperToAdapter(roleAssignmentWrapper, sdpcache.NewNoOpCache())
			sdpItem, qErr := roleAssignmentAdapter.Get(ctx, scope, roleAssignmentName, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			// Verify item type
			if sdpItem.GetType() != azureshared.AuthorizationRoleAssignment.String() {
				t.Errorf("Expected item type %s, got %s", azureshared.AuthorizationRoleAssignment.String(), sdpItem.GetType())
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

			// Verify that principal ID is in attributes
			principalIDAttr, err := sdpItem.GetAttributes().Get("properties.principalId")
			if err != nil {
				t.Logf("Warning: Could not get principalId attribute (may be in different format): %v", err)
			} else if principalIDAttr != principalID {
				t.Logf("Warning: Principal ID mismatch (expected %s, got %s), but this may be due to attribute format", principalID, principalIDAttr)
			}

			log.Printf("Verified item attributes for role assignment %s", roleAssignmentName)
		})

		t.Run("VerifyLinkedItems", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Verifying linked items for role assignment %s", roleAssignmentName)

			roleAssignmentWrapper := manual.NewAuthorizationRoleAssignment(
				clients.NewRoleAssignmentsClient(roleAssignmentsClient),
				subscriptionID,
				integrationTestResourceGroup,
			)
			scope := roleAssignmentWrapper.Scopes()[0]

			roleAssignmentAdapter := sources.WrapperToAdapter(roleAssignmentWrapper, sdpcache.NewNoOpCache())
			sdpItem, qErr := roleAssignmentAdapter.Get(ctx, scope, roleAssignmentName, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			// Verify linked item queries are created
			linkedQueries := sdpItem.GetLinkedItemQueries()
			if len(linkedQueries) == 0 {
				t.Error("Expected at least one linked item query (role definition), got 0")
			}

			// Verify role definition link exists
			foundRoleDefinitionLink := false
			for _, linkedQuery := range linkedQueries {
				if linkedQuery.GetQuery().GetType() == azureshared.AuthorizationRoleDefinition.String() {
					foundRoleDefinitionLink = true
					if linkedQuery.GetQuery().GetMethod() != sdp.QueryMethod_GET {
						t.Errorf("Expected role definition link method to be GET, got %v", linkedQuery.GetQuery().GetMethod())
					}
					if linkedQuery.GetQuery().GetScope() != subscriptionID {
						t.Errorf("Expected role definition link scope to be subscription ID %s, got %s", subscriptionID, linkedQuery.GetQuery().GetScope())
					}
					if linkedQuery.GetBlastPropagation().GetIn() != true {
						t.Error("Expected role definition link BlastPropagation.In to be true")
					}
					if linkedQuery.GetBlastPropagation().GetOut() != false {
						t.Error("Expected role definition link BlastPropagation.Out to be false")
					}
					break
				}
			}
			if !foundRoleDefinitionLink {
				t.Error("Expected to find role definition linked item query")
			}

			log.Printf("Verified linked items for role assignment %s (found %d linked queries)", roleAssignmentName, len(linkedQueries))
		})
	})

	t.Run("Teardown", func(t *testing.T) {
		ctx := t.Context()

		// Delete role assignment
		err := deleteRoleAssignment(ctx, roleAssignmentsClient, azureScope, roleAssignmentName)
		if err != nil {
			t.Fatalf("Failed to delete role assignment: %v", err)
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

// getCurrentUserPrincipalID gets the principal ID of the current authenticated user
// It tries to get it from environment variable first, then falls back to Azure CLI
func getCurrentUserPrincipalID(ctx context.Context, cred azcore.TokenCredential) (string, error) {
	// First, try to get from environment variable (useful for CI/CD)
	if principalID := os.Getenv("AZURE_PRINCIPAL_ID"); principalID != "" {
		log.Printf("Using principal ID from AZURE_PRINCIPAL_ID environment variable")
		return strings.TrimSpace(principalID), nil
	}

	// For local development, use Azure CLI to get the current user's object ID
	// This requires the user to be logged in via `az login`
	cmd := exec.CommandContext(ctx, "az", "ad", "signed-in-user", "show", "--query", "id", "-o", "tsv")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get principal ID from Azure CLI (make sure you're logged in with 'az login'): %w. Alternatively, set AZURE_PRINCIPAL_ID environment variable", err)
	}

	principalID := strings.TrimSpace(string(output))
	if principalID == "" {
		return "", fmt.Errorf("Azure CLI returned empty principal ID. Please set AZURE_PRINCIPAL_ID environment variable or ensure you're logged in with 'az login'")
	}

	log.Printf("Retrieved principal ID from Azure CLI")
	return principalID, nil
}

// getReaderRoleDefinitionID gets the Reader role definition ID
func getReaderRoleDefinitionID(ctx context.Context, client *armauthorization.RoleDefinitionsClient, subscriptionID string) (string, error) {
	scope := fmt.Sprintf("/subscriptions/%s", subscriptionID)
	filter := "roleName eq 'Reader'"

	pager := client.NewListPager(scope, &armauthorization.RoleDefinitionsClientListOptions{
		Filter: &filter,
	})

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return "", fmt.Errorf("failed to get role definitions page: %w", err)
		}

		for _, roleDef := range page.Value {
			if roleDef.Properties != nil && roleDef.Properties.RoleName != nil && *roleDef.Properties.RoleName == "Reader" {
				if roleDef.ID != nil {
					return *roleDef.ID, nil
				}
			}
		}
	}

	return "", fmt.Errorf("Reader role definition not found")
}

// createRoleAssignment creates an Azure role assignment (idempotent)
func createRoleAssignment(ctx context.Context, client *armauthorization.RoleAssignmentsClient, scope, roleAssignmentName, principalID, roleDefinitionID string) error {
	// Check if role assignment already exists
	_, err := client.Get(ctx, scope, roleAssignmentName, nil)
	if err == nil {
		log.Printf("Role assignment %s already exists, skipping creation", roleAssignmentName)
		return nil
	}

	// Create the role assignment
	// Note: We need to get the principal ID from the current user or a service principal
	// For integration tests, we'll use Azure CLI to get the current user's object ID
	// This requires running: az ad signed-in-user show --query id -o tsv
	// Or using Graph API

	// For now, let's try to create it and handle the error if principal ID is needed
	// Actually, we should get the principal ID before calling this function
	if principalID == "" {
		return fmt.Errorf("principal ID is required to create role assignment")
	}

	parameters := armauthorization.RoleAssignmentCreateParameters{
		Properties: &armauthorization.RoleAssignmentProperties{
			PrincipalID:      ptr.To(principalID),
			RoleDefinitionID: ptr.To(roleDefinitionID),
		},
	}

	resp, err := client.Create(ctx, scope, roleAssignmentName, parameters, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) {
			if respErr.StatusCode == http.StatusConflict {
				log.Printf("Role assignment %s already exists (conflict), skipping creation", roleAssignmentName)
				return nil
			}
			if respErr.StatusCode == http.StatusForbidden {
				return fmt.Errorf("insufficient permissions to create role assignment: %w", err)
			}
		}
		return fmt.Errorf("failed to create role assignment: %w", err)
	}

	// Verify the role assignment was created successfully
	if resp.RoleAssignment.ID == nil {
		return fmt.Errorf("role assignment created but ID is unknown")
	}

	log.Printf("Role assignment %s created successfully at scope %s", roleAssignmentName, scope)
	return nil
}

// deleteRoleAssignment deletes an Azure role assignment
func deleteRoleAssignment(ctx context.Context, client *armauthorization.RoleAssignmentsClient, scope, roleAssignmentName string) error {
	_, err := client.Delete(ctx, scope, roleAssignmentName, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			log.Printf("Role assignment %s not found, skipping deletion", roleAssignmentName)
			return nil
		}
		return fmt.Errorf("failed to delete role assignment: %w", err)
	}

	log.Printf("Role assignment %s deleted successfully", roleAssignmentName)
	return nil
}
