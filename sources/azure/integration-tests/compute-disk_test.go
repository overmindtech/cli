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
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v6"
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
	integrationTestDiskName = "ovm-integ-test-disk"
)

func TestComputeDiskIntegration(t *testing.T) {
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
	diskClient, err := armcompute.NewDisksClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Disks client: %v", err)
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

		// Create disk
		err = createDisk(ctx, diskClient, integrationTestResourceGroup, integrationTestDiskName, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create disk: %v", err)
		}

		// Wait for disk to be fully available
		err = waitForDiskAvailable(ctx, diskClient, integrationTestResourceGroup, integrationTestDiskName)
		if err != nil {
			t.Fatalf("Failed waiting for disk to be available: %v", err)
		}
	})

	t.Run("Run", func(t *testing.T) {
		t.Run("GetDisk", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Retrieving disk %s in subscription %s, resource group %s",
				integrationTestDiskName, subscriptionID, integrationTestResourceGroup)

			diskWrapper := manual.NewComputeDisk(
				clients.NewDisksClient(diskClient),
				subscriptionID,
				integrationTestResourceGroup,
			)
			scope := diskWrapper.Scopes()[0]

			diskAdapter := sources.WrapperToAdapter(diskWrapper, sdpcache.NewNoOpCache())
			sdpItem, qErr := diskAdapter.Get(ctx, scope, integrationTestDiskName, true)
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

			if uniqueAttrValue != integrationTestDiskName {
				t.Fatalf("Expected unique attribute value to be %s, got %s", integrationTestDiskName, uniqueAttrValue)
			}

			log.Printf("Successfully retrieved disk %s", integrationTestDiskName)
		})

		t.Run("ListDisks", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Listing disks in subscription %s, resource group %s",
				subscriptionID, integrationTestResourceGroup)

			diskWrapper := manual.NewComputeDisk(
				clients.NewDisksClient(diskClient),
				subscriptionID,
				integrationTestResourceGroup,
			)
			scope := diskWrapper.Scopes()[0]

			diskAdapter := sources.WrapperToAdapter(diskWrapper, sdpcache.NewNoOpCache())

			// Check if adapter supports listing
			listable, ok := diskAdapter.(discovery.ListableAdapter)
			if !ok {
				t.Fatalf("Adapter does not support List operation")
			}

			sdpItems, err := listable.List(ctx, scope, true)
			if err != nil {
				t.Fatalf("Failed to list disks: %v", err)
			}

			if len(sdpItems) < 1 {
				t.Fatalf("Expected at least one disk, got %d", len(sdpItems))
			}

			var found bool
			for _, item := range sdpItems {
				uniqueAttrKey := item.GetUniqueAttribute()
				if v, err := item.GetAttributes().Get(uniqueAttrKey); err == nil && v == integrationTestDiskName {
					found = true
					break
				}
			}

			if !found {
				t.Fatalf("Expected to find disk %s in the list of disks", integrationTestDiskName)
			}

			log.Printf("Found %d disks in resource group %s", len(sdpItems), integrationTestResourceGroup)
		})

		t.Run("VerifyItemAttributes", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Verifying item attributes for disk %s", integrationTestDiskName)

			diskWrapper := manual.NewComputeDisk(
				clients.NewDisksClient(diskClient),
				subscriptionID,
				integrationTestResourceGroup,
			)
			scope := diskWrapper.Scopes()[0]

			diskAdapter := sources.WrapperToAdapter(diskWrapper, sdpcache.NewNoOpCache())
			sdpItem, qErr := diskAdapter.Get(ctx, scope, integrationTestDiskName, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			// Verify item type
			if sdpItem.GetType() != azureshared.ComputeDisk.String() {
				t.Errorf("Expected item type %s, got %s", azureshared.ComputeDisk, sdpItem.GetType())
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

			log.Printf("Verified item attributes for disk %s", integrationTestDiskName)
		})

		t.Run("VerifyLinkedItems", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Verifying linked items for disk %s", integrationTestDiskName)

			diskWrapper := manual.NewComputeDisk(
				clients.NewDisksClient(diskClient),
				subscriptionID,
				integrationTestResourceGroup,
			)
			scope := diskWrapper.Scopes()[0]

			diskAdapter := sources.WrapperToAdapter(diskWrapper, sdpcache.NewNoOpCache())
			sdpItem, qErr := diskAdapter.Get(ctx, scope, integrationTestDiskName, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			// Verify that linked items exist (if any)
			linkedQueries := sdpItem.GetLinkedItemQueries()
			log.Printf("Found %d linked item queries for disk %s", len(linkedQueries), integrationTestDiskName)

			// For a standalone empty disk, there may not be any linked items
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

				// Verify blast propagation is set
				bp := liq.GetBlastPropagation()
				if bp == nil {
					t.Error("Linked item query has nil BlastPropagation")
				} else {
					// Blast propagation should have In and Out set (even if false)
					_ = bp.GetIn()
					_ = bp.GetOut()
				}

				log.Printf("Verified linked item query: Type=%s, Method=%s, Query=%s, Scope=%s, In=%v, Out=%v",
					query.GetType(), query.GetMethod(), query.GetQuery(), query.GetScope(),
					bp.GetIn(), bp.GetOut())
			}
		})
	})

	t.Run("Teardown", func(t *testing.T) {
		ctx := t.Context()

		// Delete disk
		err := deleteDisk(ctx, diskClient, integrationTestResourceGroup, integrationTestDiskName)
		if err != nil {
			t.Fatalf("Failed to delete disk: %v", err)
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

// createDisk creates an Azure managed disk (idempotent)
func createDisk(ctx context.Context, client *armcompute.DisksClient, resourceGroupName, diskName, location string) error {
	// Check if disk already exists
	existingDisk, err := client.Get(ctx, resourceGroupName, diskName, nil)
	if err == nil {
		// Disk exists, check its state
		if existingDisk.Properties != nil && existingDisk.Properties.ProvisioningState != nil {
			state := *existingDisk.Properties.ProvisioningState
			if state == "Succeeded" {
				log.Printf("Disk %s already exists with state %s, skipping creation", diskName, state)
				return nil
			}
			log.Printf("Disk %s exists but in state %s, will wait for it", diskName, state)
		} else {
			log.Printf("Disk %s already exists, skipping creation", diskName)
			return nil
		}
	}

	// Create an empty disk (DiskCreateOptionEmpty)
	// This is the simplest type of disk to create for testing
	poller, err := client.BeginCreateOrUpdate(ctx, resourceGroupName, diskName, armcompute.Disk{
		Location: ptr.To(location),
		Properties: &armcompute.DiskProperties{
			CreationData: &armcompute.CreationData{
				CreateOption: ptr.To(armcompute.DiskCreateOptionEmpty),
			},
			DiskSizeGB: ptr.To[int32](10), // 10 GB disk
		},
		SKU: &armcompute.DiskSKU{
			Name: ptr.To(armcompute.DiskStorageAccountTypesStandardLRS),
		},
		Tags: map[string]*string{
			"purpose": ptr.To("overmind-integration-tests"),
			"test":    ptr.To("compute-disk"),
		},
	}, nil)
	if err != nil {
		// Check if disk already exists (conflict)
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusConflict {
			log.Printf("Disk %s already exists (conflict), skipping creation", diskName)
			return nil
		}
		return fmt.Errorf("failed to begin creating disk: %w", err)
	}

	resp, err := poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to create disk: %w", err)
	}

	// Verify the disk was created successfully
	if resp.Properties == nil || resp.Properties.ProvisioningState == nil {
		return fmt.Errorf("disk created but provisioning state is unknown")
	}

	provisioningState := *resp.Properties.ProvisioningState
	if provisioningState != "Succeeded" {
		return fmt.Errorf("disk provisioning state is %s, expected Succeeded", provisioningState)
	}

	log.Printf("Disk %s created successfully with provisioning state: %s", diskName, provisioningState)
	return nil
}

// waitForDiskAvailable polls until the disk is available via the Get API
// This is needed because even after creation succeeds, there can be a delay before the disk is queryable
func waitForDiskAvailable(ctx context.Context, client *armcompute.DisksClient, resourceGroupName, diskName string) error {
	maxAttempts := 20
	pollInterval := 5 * time.Second

	log.Printf("Waiting for disk %s to be available via API...", diskName)

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		resp, err := client.Get(ctx, resourceGroupName, diskName, nil)
		if err != nil {
			var respErr *azcore.ResponseError
			if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
				log.Printf("Disk %s not yet available (attempt %d/%d), waiting %v...", diskName, attempt, maxAttempts, pollInterval)
				time.Sleep(pollInterval)
				continue
			}
			return fmt.Errorf("error checking disk availability: %w", err)
		}

		// Check provisioning state
		if resp.Properties != nil && resp.Properties.ProvisioningState != nil {
			state := *resp.Properties.ProvisioningState
			if state == "Succeeded" {
				log.Printf("Disk %s is available with provisioning state: %s", diskName, state)
				return nil
			}
			if state == "Failed" {
				return fmt.Errorf("disk provisioning failed with state: %s", state)
			}
			// Still provisioning, wait and retry
			log.Printf("Disk %s provisioning state: %s (attempt %d/%d), waiting...", diskName, state, attempt, maxAttempts)
			time.Sleep(pollInterval)
			continue
		}

		// Disk exists but no provisioning state - consider it available
		log.Printf("Disk %s is available", diskName)
		return nil
	}

	return fmt.Errorf("timeout waiting for disk %s to be available after %d attempts", diskName, maxAttempts)
}

// deleteDisk deletes an Azure managed disk
func deleteDisk(ctx context.Context, client *armcompute.DisksClient, resourceGroupName, diskName string) error {
	poller, err := client.BeginDelete(ctx, resourceGroupName, diskName, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			log.Printf("Disk %s not found, skipping deletion", diskName)
			return nil
		}
		return fmt.Errorf("failed to begin deleting disk: %w", err)
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to delete disk: %w", err)
	}

	log.Printf("Disk %s deleted successfully", diskName)
	return nil
}
