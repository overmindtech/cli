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
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources/v2"
	log "github.com/sirupsen/logrus"
	"k8s.io/utils/ptr"

	"github.com/overmindtech/workspace/discovery"
	"github.com/overmindtech/workspace/sdp-go"
	"github.com/overmindtech/workspace/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	"github.com/overmindtech/cli/sources/azure/manual"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
)

const (
	integrationTestDiskAccessName = "ovm-integ-test-disk-access"
)

func TestComputeDiskAccessIntegration(t *testing.T) {
	subscriptionID := os.Getenv("AZURE_SUBSCRIPTION_ID")
	if subscriptionID == "" {
		t.Skip("AZURE_SUBSCRIPTION_ID environment variable not set")
	}

	cred, err := azureshared.NewAzureCredential(t.Context())
	if err != nil {
		t.Fatalf("Failed to create Azure credential: %v", err)
	}

	diskAccessClient, err := armcompute.NewDiskAccessesClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Disk Accesses client: %v", err)
	}

	rgClient, err := armresources.NewResourceGroupsClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Resource Groups client: %v", err)
	}

	t.Run("Setup", func(t *testing.T) {
		ctx := t.Context()

		err := createResourceGroup(ctx, rgClient, integrationTestResourceGroup, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create resource group: %v", err)
		}

		err = createDiskAccess(ctx, diskAccessClient, integrationTestResourceGroup, integrationTestDiskAccessName, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create disk access: %v", err)
		}

		err = waitForDiskAccessAvailable(ctx, diskAccessClient, integrationTestResourceGroup, integrationTestDiskAccessName)
		if err != nil {
			t.Fatalf("Failed waiting for disk access to be available: %v", err)
		}
	})

	t.Run("Run", func(t *testing.T) {
		t.Run("GetDiskAccess", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Retrieving disk access %s in subscription %s, resource group %s",
				integrationTestDiskAccessName, subscriptionID, integrationTestResourceGroup)

			diskAccessWrapper := manual.NewComputeDiskAccess(
				clients.NewDiskAccessesClient(diskAccessClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := diskAccessWrapper.Scopes()[0]

			diskAccessAdapter := sources.WrapperToAdapter(diskAccessWrapper, sdpcache.NewNoOpCache())
			sdpItem, qErr := diskAccessAdapter.Get(ctx, scope, integrationTestDiskAccessName, true)
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

			if uniqueAttrValue != integrationTestDiskAccessName {
				t.Fatalf("Expected unique attribute value to be %s, got %s", integrationTestDiskAccessName, uniqueAttrValue)
			}

			log.Printf("Successfully retrieved disk access %s", integrationTestDiskAccessName)
		})

		t.Run("ListDiskAccesses", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Listing disk accesses in subscription %s, resource group %s",
				subscriptionID, integrationTestResourceGroup)

			diskAccessWrapper := manual.NewComputeDiskAccess(
				clients.NewDiskAccessesClient(diskAccessClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := diskAccessWrapper.Scopes()[0]

			diskAccessAdapter := sources.WrapperToAdapter(diskAccessWrapper, sdpcache.NewNoOpCache())

			listable, ok := diskAccessAdapter.(discovery.ListableAdapter)
			if !ok {
				t.Fatalf("Adapter does not support List operation")
			}

			sdpItems, err := listable.List(ctx, scope, true)
			if err != nil {
				t.Fatalf("Failed to list disk accesses: %v", err)
			}

			if len(sdpItems) < 1 {
				t.Fatalf("Expected at least one disk access, got %d", len(sdpItems))
			}

			var found bool
			for _, item := range sdpItems {
				uniqueAttrKey := item.GetUniqueAttribute()
				if v, err := item.GetAttributes().Get(uniqueAttrKey); err == nil && v == integrationTestDiskAccessName {
					found = true
					break
				}
			}

			if !found {
				t.Fatalf("Expected to find disk access %s in the list of disk accesses", integrationTestDiskAccessName)
			}

			log.Printf("Found %d disk accesses in resource group %s", len(sdpItems), integrationTestResourceGroup)
		})

		t.Run("VerifyItemAttributes", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Verifying item attributes for disk access %s", integrationTestDiskAccessName)

			diskAccessWrapper := manual.NewComputeDiskAccess(
				clients.NewDiskAccessesClient(diskAccessClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := diskAccessWrapper.Scopes()[0]

			diskAccessAdapter := sources.WrapperToAdapter(diskAccessWrapper, sdpcache.NewNoOpCache())
			sdpItem, qErr := diskAccessAdapter.Get(ctx, scope, integrationTestDiskAccessName, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			if sdpItem.GetType() != azureshared.ComputeDiskAccess.String() {
				t.Errorf("Expected item type %s, got %s", azureshared.ComputeDiskAccess.String(), sdpItem.GetType())
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

			log.Printf("Verified item attributes for disk access %s", integrationTestDiskAccessName)
		})

		t.Run("VerifyLinkedItems", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Verifying linked items for disk access %s", integrationTestDiskAccessName)

			diskAccessWrapper := manual.NewComputeDiskAccess(
				clients.NewDiskAccessesClient(diskAccessClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := diskAccessWrapper.Scopes()[0]

			diskAccessAdapter := sources.WrapperToAdapter(diskAccessWrapper, sdpcache.NewNoOpCache())
			sdpItem, qErr := diskAccessAdapter.Get(ctx, scope, integrationTestDiskAccessName, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			linkedQueries := sdpItem.GetLinkedItemQueries()
			log.Printf("Found %d linked item queries for disk access %s", len(linkedQueries), integrationTestDiskAccessName)

			// Disk access always has at least one linked query: ComputeDiskAccessPrivateEndpointConnection (SEARCH)
			if len(linkedQueries) < 1 {
				t.Errorf("Expected at least one linked item query (private endpoint connection), got %d", len(linkedQueries))
			}

			for _, liq := range linkedQueries {
				query := liq.GetQuery()
				if query == nil {
					t.Error("Linked item query has nil Query")
					continue
				}

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

				bp := liq.GetBlastPropagation()
				if bp == nil {
					t.Error("Linked item query has nil BlastPropagation")
					log.Printf("Verified linked item query: Type=%s, Method=%s, Query=%s, Scope=%s (BlastPropagation nil)",
						query.GetType(), query.GetMethod(), query.GetQuery(), query.GetScope())
				} else {
					log.Printf("Verified linked item query: Type=%s, Method=%s, Query=%s, Scope=%s, In=%v, Out=%v",
						query.GetType(), query.GetMethod(), query.GetQuery(), query.GetScope(),
						bp.GetIn(), bp.GetOut())
				}
			}
		})
	})

	t.Run("Teardown", func(t *testing.T) {
		ctx := t.Context()

		err := deleteDiskAccess(ctx, diskAccessClient, integrationTestResourceGroup, integrationTestDiskAccessName)
		if err != nil {
			t.Fatalf("Failed to delete disk access: %v", err)
		}
	})
}

// createDiskAccess creates an Azure disk access resource (idempotent).
func createDiskAccess(ctx context.Context, client *armcompute.DiskAccessesClient, resourceGroupName, diskAccessName, location string) error {
	existing, err := client.Get(ctx, resourceGroupName, diskAccessName, nil)
	if err == nil {
		if existing.Properties != nil && existing.Properties.ProvisioningState != nil {
			state := *existing.Properties.ProvisioningState
			if state == "Succeeded" {
				log.Printf("Disk access %s already exists with state %s, skipping creation", diskAccessName, state)
				return nil
			}
			log.Printf("Disk access %s exists but in state %s, will wait for it", diskAccessName, state)
		} else {
			log.Printf("Disk access %s already exists, skipping creation", diskAccessName)
			return nil
		}
	}

	poller, err := client.BeginCreateOrUpdate(ctx, resourceGroupName, diskAccessName, armcompute.DiskAccess{
		Location: ptr.To(location),
		Tags: map[string]*string{
			"purpose": ptr.To("overmind-integration-tests"),
			"test":    ptr.To("compute-disk-access"),
		},
	}, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusConflict {
			log.Printf("Disk access %s already exists (conflict), skipping creation", diskAccessName)
			return nil
		}
		return fmt.Errorf("failed to begin creating disk access: %w", err)
	}

	resp, err := poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to create disk access: %w", err)
	}

	if resp.Properties != nil && resp.Properties.ProvisioningState != nil {
		state := *resp.Properties.ProvisioningState
		if state != "Succeeded" {
			return fmt.Errorf("disk access provisioning state is %s, expected Succeeded", state)
		}
		log.Printf("Disk access %s created successfully with provisioning state: %s", diskAccessName, state)
	} else {
		log.Printf("Disk access %s created successfully", diskAccessName)
	}

	return nil
}

// waitForDiskAccessAvailable polls until the disk access is available via the Get API.
func waitForDiskAccessAvailable(ctx context.Context, client *armcompute.DiskAccessesClient, resourceGroupName, diskAccessName string) error {
	maxAttempts := 20
	pollInterval := 5 * time.Second

	log.Printf("Waiting for disk access %s to be available via API...", diskAccessName)

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		resp, err := client.Get(ctx, resourceGroupName, diskAccessName, nil)
		if err != nil {
			var respErr *azcore.ResponseError
			if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
				log.Printf("Disk access %s not yet available (attempt %d/%d), waiting %v...", diskAccessName, attempt, maxAttempts, pollInterval)
				time.Sleep(pollInterval)
				continue
			}
			return fmt.Errorf("error checking disk access availability: %w", err)
		}

		if resp.Properties != nil && resp.Properties.ProvisioningState != nil {
			state := *resp.Properties.ProvisioningState
			if state == "Succeeded" {
				log.Printf("Disk access %s is available with provisioning state: %s", diskAccessName, state)
				return nil
			}
			if state == "Failed" {
				return fmt.Errorf("disk access provisioning failed with state: %s", state)
			}
			log.Printf("Disk access %s provisioning state: %s (attempt %d/%d), waiting...", diskAccessName, state, attempt, maxAttempts)
			time.Sleep(pollInterval)
			continue
		}

		log.Printf("Disk access %s is available", diskAccessName)
		return nil
	}

	return fmt.Errorf("timeout waiting for disk access %s to be available after %d attempts", diskAccessName, maxAttempts)
}

// deleteDiskAccess deletes an Azure disk access resource.
func deleteDiskAccess(ctx context.Context, client *armcompute.DiskAccessesClient, resourceGroupName, diskAccessName string) error {
	poller, err := client.BeginDelete(ctx, resourceGroupName, diskAccessName, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			log.Printf("Disk access %s not found, skipping deletion", diskAccessName)
			return nil
		}
		return fmt.Errorf("failed to begin deleting disk access: %w", err)
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to delete disk access: %w", err)
	}

	log.Printf("Disk access %s deleted successfully", diskAccessName)
	return nil
}
