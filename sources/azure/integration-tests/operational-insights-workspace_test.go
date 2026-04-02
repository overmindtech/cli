package integrationtests

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/operationalinsights/armoperationalinsights"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources/v2"
	log "github.com/sirupsen/logrus"

	"github.com/overmindtech/cli/go/discovery"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	"github.com/overmindtech/cli/sources/azure/manual"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
)

const (
	integrationTestWorkspaceName = "ovm-integ-test-workspace"
)

// errOperationalInsightsAuthorizationFailed is a sentinel error for authorization failures
var errOperationalInsightsAuthorizationFailed = errors.New("authorization failed for Operational Insights resource provider")

func TestOperationalInsightsWorkspaceIntegration(t *testing.T) {
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
	workspacesClient, err := armoperationalinsights.NewWorkspacesClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Workspaces client: %v", err)
	}

	rgClient, err := armresources.NewResourceGroupsClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Resource Groups client: %v", err)
	}

	var setupCompleted bool

	t.Run("Setup", func(t *testing.T) {
		ctx := t.Context()

		// Create resource group if it doesn't exist
		err := createResourceGroup(ctx, rgClient, integrationTestResourceGroup, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create resource group: %v", err)
		}

		// Create workspace
		err = createOperationalInsightsWorkspace(ctx, workspacesClient, integrationTestResourceGroup, integrationTestWorkspaceName, integrationTestLocation)
		if err != nil {
			if errors.Is(err, errOperationalInsightsAuthorizationFailed) {
				t.Skipf("Skipping test: %v (service principal lacks permission to register Microsoft.OperationalInsights resource provider)", err)
			}
			t.Fatalf("Failed to create workspace: %v", err)
		}

		// Wait for workspace to be fully available
		err = waitForOperationalInsightsWorkspaceAvailable(ctx, workspacesClient, integrationTestResourceGroup, integrationTestWorkspaceName)
		if err != nil {
			t.Fatalf("Failed waiting for workspace to be available: %v", err)
		}

		setupCompleted = true
	})

	t.Run("Run", func(t *testing.T) {
		if !setupCompleted {
			t.Skip("Skipping Run: Setup did not complete successfully")
		}

		t.Run("GetWorkspace", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Retrieving workspace %s in subscription %s, resource group %s",
				integrationTestWorkspaceName, subscriptionID, integrationTestResourceGroup)

			workspaceWrapper := manual.NewOperationalInsightsWorkspace(
				clients.NewOperationalInsightsWorkspaceClient(workspacesClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := workspaceWrapper.Scopes()[0]

			workspaceAdapter := sources.WrapperToAdapter(workspaceWrapper, sdpcache.NewNoOpCache())
			sdpItem, qErr := workspaceAdapter.Get(ctx, scope, integrationTestWorkspaceName, true)
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

			if uniqueAttrValue != integrationTestWorkspaceName {
				t.Fatalf("Expected unique attribute value to be %s, got %s", integrationTestWorkspaceName, uniqueAttrValue)
			}

			log.Printf("Successfully retrieved workspace %s", integrationTestWorkspaceName)
		})

		t.Run("ListWorkspaces", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Listing workspaces in subscription %s, resource group %s",
				subscriptionID, integrationTestResourceGroup)

			workspaceWrapper := manual.NewOperationalInsightsWorkspace(
				clients.NewOperationalInsightsWorkspaceClient(workspacesClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := workspaceWrapper.Scopes()[0]

			workspaceAdapter := sources.WrapperToAdapter(workspaceWrapper, sdpcache.NewNoOpCache())

			// Check if adapter supports listing
			listable, ok := workspaceAdapter.(discovery.ListableAdapter)
			if !ok {
				t.Fatalf("Adapter does not support List operation")
			}

			sdpItems, err := listable.List(ctx, scope, true)
			if err != nil {
				t.Fatalf("Failed to list workspaces: %v", err)
			}

			if len(sdpItems) < 1 {
				t.Fatalf("Expected at least one workspace, got %d", len(sdpItems))
			}

			var found bool
			for _, item := range sdpItems {
				uniqueAttrKey := item.GetUniqueAttribute()
				if v, err := item.GetAttributes().Get(uniqueAttrKey); err == nil && v == integrationTestWorkspaceName {
					found = true
					break
				}
			}

			if !found {
				t.Fatalf("Expected to find workspace %s in the list of workspaces", integrationTestWorkspaceName)
			}

			log.Printf("Found %d workspaces in resource group %s", len(sdpItems), integrationTestResourceGroup)
		})

		t.Run("VerifyItemAttributes", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Verifying item attributes for workspace %s", integrationTestWorkspaceName)

			workspaceWrapper := manual.NewOperationalInsightsWorkspace(
				clients.NewOperationalInsightsWorkspaceClient(workspacesClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := workspaceWrapper.Scopes()[0]

			workspaceAdapter := sources.WrapperToAdapter(workspaceWrapper, sdpcache.NewNoOpCache())
			sdpItem, qErr := workspaceAdapter.Get(ctx, scope, integrationTestWorkspaceName, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			// Verify item type
			if sdpItem.GetType() != azureshared.OperationalInsightsWorkspace.String() {
				t.Errorf("Expected item type %s, got %s", azureshared.OperationalInsightsWorkspace, sdpItem.GetType())
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

			log.Printf("Verified item attributes for workspace %s", integrationTestWorkspaceName)
		})

		t.Run("VerifyLinkedItems", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Verifying linked items for workspace %s", integrationTestWorkspaceName)

			workspaceWrapper := manual.NewOperationalInsightsWorkspace(
				clients.NewOperationalInsightsWorkspaceClient(workspacesClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := workspaceWrapper.Scopes()[0]

			workspaceAdapter := sources.WrapperToAdapter(workspaceWrapper, sdpcache.NewNoOpCache())
			sdpItem, qErr := workspaceAdapter.Get(ctx, scope, integrationTestWorkspaceName, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			// Verify that linked items exist (if any)
			linkedQueries := sdpItem.GetLinkedItemQueries()
			log.Printf("Found %d linked item queries for workspace %s", len(linkedQueries), integrationTestWorkspaceName)

			// For a standalone workspace without private link, there may not be any linked items
			// But we should verify the structure is correct if links exist
			for _, liq := range linkedQueries {
				query := liq.GetQuery()
				if query == nil {
					t.Error("Linked item query has nil Query")
					continue
				}

				// Verify query has required fields
				if query.GetType() == "" {
					t.Error("Linked item query has empty Type")
				}
				// Method should be GET or SEARCH (not empty)
				if query.GetMethod() == sdp.QueryMethod_GET || query.GetMethod() == sdp.QueryMethod_SEARCH {
					// Valid method
				} else {
					t.Errorf("Linked item query has unexpected Method: %v", query.GetMethod())
				}
				if query.GetQuery() == "" {
					t.Error("Linked item query has empty Query")
				}
				if query.GetScope() == "" {
					t.Error("Linked item query has empty Scope")
				}

				log.Printf("Verified linked item query: Type=%s, Method=%s, Query=%s, Scope=%s",
					query.GetType(), query.GetMethod(), query.GetQuery(), query.GetScope())
			}
		})
	})

	t.Run("Teardown", func(t *testing.T) {
		ctx := t.Context()

		// Delete workspace
		err := deleteOperationalInsightsWorkspace(ctx, workspacesClient, integrationTestResourceGroup, integrationTestWorkspaceName)
		if err != nil {
			t.Fatalf("Failed to delete workspace: %v", err)
		}

		// Note: We keep the resource group for faster subsequent test runs
	})
}

// createOperationalInsightsWorkspace creates an Azure Log Analytics workspace (idempotent)
func createOperationalInsightsWorkspace(ctx context.Context, client *armoperationalinsights.WorkspacesClient, resourceGroupName, workspaceName, location string) error {
	// Check if workspace already exists
	existingWorkspace, err := client.Get(ctx, resourceGroupName, workspaceName, nil)
	if err == nil {
		// Workspace exists, check its state
		if existingWorkspace.Properties != nil && existingWorkspace.Properties.ProvisioningState != nil {
			state := *existingWorkspace.Properties.ProvisioningState
			if state == armoperationalinsights.WorkspaceEntityStatusSucceeded {
				log.Printf("Workspace %s already exists with state %s, skipping creation", workspaceName, state)
				return nil
			}
			log.Printf("Workspace %s exists but in state %s, will wait for it", workspaceName, state)
		} else {
			log.Printf("Workspace %s already exists, skipping creation", workspaceName)
			return nil
		}
	}

	// Create the workspace
	retentionDays := int32(30)
	poller, err := client.BeginCreateOrUpdate(ctx, resourceGroupName, workspaceName, armoperationalinsights.Workspace{
		Location: new(location),
		Properties: &armoperationalinsights.WorkspaceProperties{
			RetentionInDays: &retentionDays,
		},
		Tags: map[string]*string{
			"purpose": new("overmind-integration-tests"),
			"test":    new("operational-insights-workspace"),
		},
	}, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) {
			// Check for authorization failure (resource provider not registered)
			if respErr.StatusCode == http.StatusForbidden && respErr.ErrorCode == "AuthorizationFailed" {
				return fmt.Errorf("%w: %s", errOperationalInsightsAuthorizationFailed, respErr.Error())
			}
			// Check for missing resource provider registration
			if strings.Contains(respErr.Error(), "register/action") {
				return fmt.Errorf("%w: %s", errOperationalInsightsAuthorizationFailed, respErr.Error())
			}
			// Check if workspace already exists (conflict)
			if respErr.StatusCode == http.StatusConflict {
				// Verify conflict is real before treating it as success.
				if _, getErr := client.Get(ctx, resourceGroupName, workspaceName, nil); getErr == nil {
					log.Printf("Workspace %s already exists (conflict), skipping", workspaceName)
					return nil
				}
				return fmt.Errorf("workspace %s conflict but not retrievable: %w", workspaceName, err)
			}
		}
		return fmt.Errorf("failed to begin creating workspace: %w", err)
	}

	resp, err := poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to create workspace: %w", err)
	}

	// Verify the workspace was created successfully
	if resp.Properties == nil || resp.Properties.ProvisioningState == nil {
		return fmt.Errorf("workspace created but provisioning state is unknown")
	}

	provisioningState := *resp.Properties.ProvisioningState
	if provisioningState != armoperationalinsights.WorkspaceEntityStatusSucceeded {
		return fmt.Errorf("workspace provisioning state is %s, expected Succeeded", provisioningState)
	}

	log.Printf("Workspace %s created successfully with provisioning state: %s", workspaceName, provisioningState)
	return nil
}

// waitForOperationalInsightsWorkspaceAvailable polls until the workspace is available via the Get API
func waitForOperationalInsightsWorkspaceAvailable(ctx context.Context, client *armoperationalinsights.WorkspacesClient, resourceGroupName, workspaceName string) error {
	maxAttempts := 20
	pollInterval := 5 * time.Second
	maxNotFoundAttempts := 5
	notFoundCount := 0

	log.Printf("Waiting for workspace %s to be available via API...", workspaceName)

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		resp, err := client.Get(ctx, resourceGroupName, workspaceName, nil)
		if err != nil {
			var respErr *azcore.ResponseError
			if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
				notFoundCount++
				if notFoundCount >= maxNotFoundAttempts {
					return fmt.Errorf("workspace %s not found after %d attempts", workspaceName, notFoundCount)
				}
				log.Printf("Workspace %s not yet available (attempt %d/%d), waiting %v...", workspaceName, attempt, maxAttempts, pollInterval)
				time.Sleep(pollInterval)
				continue
			}
			return fmt.Errorf("error checking workspace availability: %w", err)
		}
		notFoundCount = 0

		// Check provisioning state
		if resp.Properties != nil && resp.Properties.ProvisioningState != nil {
			state := *resp.Properties.ProvisioningState
			if state == armoperationalinsights.WorkspaceEntityStatusSucceeded {
				log.Printf("Workspace %s is available with provisioning state: %s", workspaceName, state)
				return nil
			}
			if state == armoperationalinsights.WorkspaceEntityStatusFailed {
				return fmt.Errorf("workspace provisioning failed with state: %s", state)
			}
			// Still provisioning, wait and retry
			log.Printf("Workspace %s provisioning state: %s (attempt %d/%d), waiting...", workspaceName, state, attempt, maxAttempts)
			time.Sleep(pollInterval)
			continue
		}

		// Workspace exists but no provisioning state - consider it available
		log.Printf("Workspace %s is available", workspaceName)
		return nil
	}

	return fmt.Errorf("timeout waiting for workspace %s to be available after %d attempts", workspaceName, maxAttempts)
}

// deleteOperationalInsightsWorkspace deletes an Azure Log Analytics workspace
func deleteOperationalInsightsWorkspace(ctx context.Context, client *armoperationalinsights.WorkspacesClient, resourceGroupName, workspaceName string) error {
	poller, err := client.BeginDelete(ctx, resourceGroupName, workspaceName, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			log.Printf("Workspace %s not found, skipping deletion", workspaceName)
			return nil
		}
		return fmt.Errorf("failed to begin deleting workspace: %w", err)
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to delete workspace: %w", err)
	}

	log.Printf("Workspace %s deleted successfully", workspaceName)
	return nil
}
