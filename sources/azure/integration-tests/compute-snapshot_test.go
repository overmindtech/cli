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

	"github.com/overmindtech/cli/go/discovery"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	"github.com/overmindtech/cli/sources/azure/manual"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
)

const (
	integrationTestSnapshotName    = "ovm-integ-test-snapshot"
	integrationTestDiskForSnapName = "ovm-integ-test-disk-for-snap"
)

func TestComputeSnapshotIntegration(t *testing.T) {
	subscriptionID := os.Getenv("AZURE_SUBSCRIPTION_ID")
	if subscriptionID == "" {
		t.Skip("AZURE_SUBSCRIPTION_ID environment variable not set")
	}

	cred, err := azureshared.NewAzureCredential(t.Context())
	if err != nil {
		t.Fatalf("Failed to create Azure credential: %v", err)
	}

	snapshotClient, err := armcompute.NewSnapshotsClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Snapshots client: %v", err)
	}

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

		err := createResourceGroup(ctx, rgClient, integrationTestResourceGroup, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create resource group: %v", err)
		}

		// Create a disk to snapshot from
		err = createDisk(ctx, diskClient, integrationTestResourceGroup, integrationTestDiskForSnapName, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create disk: %v", err)
		}

		err = waitForDiskAvailable(ctx, diskClient, integrationTestResourceGroup, integrationTestDiskForSnapName)
		if err != nil {
			t.Fatalf("Failed waiting for disk to be available: %v", err)
		}

		// Get disk ID for snapshot creation
		diskResp, err := diskClient.Get(ctx, integrationTestResourceGroup, integrationTestDiskForSnapName, nil)
		if err != nil {
			t.Fatalf("Failed to get disk: %v", err)
		}

		// Create snapshot from the disk
		err = createSnapshot(ctx, snapshotClient, integrationTestResourceGroup, integrationTestSnapshotName, integrationTestLocation, *diskResp.ID)
		if err != nil {
			t.Fatalf("Failed to create snapshot: %v", err)
		}

		err = waitForSnapshotAvailable(ctx, snapshotClient, integrationTestResourceGroup, integrationTestSnapshotName)
		if err != nil {
			t.Fatalf("Failed waiting for snapshot to be available: %v", err)
		}
	})

	t.Run("Run", func(t *testing.T) {
		ctx := t.Context()
		_, err := snapshotClient.Get(ctx, integrationTestResourceGroup, integrationTestSnapshotName, nil)
		if err != nil {
			var respErr *azcore.ResponseError
			if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
				t.Skipf("Snapshot %s does not exist - Setup may have failed. Skipping Run tests.", integrationTestSnapshotName)
			}
		}

		t.Run("GetSnapshot", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Retrieving snapshot %s in subscription %s, resource group %s",
				integrationTestSnapshotName, subscriptionID, integrationTestResourceGroup)

			snapshotWrapper := manual.NewComputeSnapshot(
				clients.NewSnapshotsClient(snapshotClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := snapshotWrapper.Scopes()[0]

			snapshotAdapter := sources.WrapperToAdapter(snapshotWrapper, sdpcache.NewNoOpCache())
			sdpItem, qErr := snapshotAdapter.Get(ctx, scope, integrationTestSnapshotName, true)
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

			if uniqueAttrValue != integrationTestSnapshotName {
				t.Fatalf("Expected unique attribute value to be %s, got %s", integrationTestSnapshotName, uniqueAttrValue)
			}

			if sdpItem.GetType() != azureshared.ComputeSnapshot.String() {
				t.Fatalf("Expected type %s, got %s", azureshared.ComputeSnapshot, sdpItem.GetType())
			}

			log.Printf("Successfully retrieved snapshot %s", integrationTestSnapshotName)
		})

		t.Run("ListSnapshots", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Listing snapshots in subscription %s, resource group %s",
				subscriptionID, integrationTestResourceGroup)

			snapshotWrapper := manual.NewComputeSnapshot(
				clients.NewSnapshotsClient(snapshotClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := snapshotWrapper.Scopes()[0]

			snapshotAdapter := sources.WrapperToAdapter(snapshotWrapper, sdpcache.NewNoOpCache())

			listable, ok := snapshotAdapter.(discovery.ListableAdapter)
			if !ok {
				t.Fatalf("Adapter does not support List operation")
			}

			sdpItems, err := listable.List(ctx, scope, true)
			if err != nil {
				t.Fatalf("Failed to list snapshots: %v", err)
			}

			if len(sdpItems) < 1 {
				t.Fatalf("Expected at least one snapshot, got %d", len(sdpItems))
			}

			var found bool
			for _, item := range sdpItems {
				uniqueAttrKey := item.GetUniqueAttribute()
				if v, err := item.GetAttributes().Get(uniqueAttrKey); err == nil && v == integrationTestSnapshotName {
					found = true
					if item.GetType() != azureshared.ComputeSnapshot.String() {
						t.Errorf("Expected type %s, got %s", azureshared.ComputeSnapshot, item.GetType())
					}
					break
				}
			}

			if !found {
				t.Fatalf("Expected to find snapshot %s in the list of snapshots", integrationTestSnapshotName)
			}

			log.Printf("Found %d snapshots in resource group %s", len(sdpItems), integrationTestResourceGroup)
		})

		t.Run("VerifyItemAttributes", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Verifying item attributes for snapshot %s", integrationTestSnapshotName)

			snapshotWrapper := manual.NewComputeSnapshot(
				clients.NewSnapshotsClient(snapshotClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := snapshotWrapper.Scopes()[0]

			snapshotAdapter := sources.WrapperToAdapter(snapshotWrapper, sdpcache.NewNoOpCache())
			sdpItem, qErr := snapshotAdapter.Get(ctx, scope, integrationTestSnapshotName, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			if sdpItem.GetType() != azureshared.ComputeSnapshot.String() {
				t.Errorf("Expected item type %s, got %s", azureshared.ComputeSnapshot, sdpItem.GetType())
			}

			expectedScope := fmt.Sprintf("%s.%s", subscriptionID, integrationTestResourceGroup)
			if sdpItem.GetScope() != expectedScope {
				t.Errorf("Expected scope %s, got %s", expectedScope, sdpItem.GetScope())
			}

			if sdpItem.GetUniqueAttribute() != "name" {
				t.Errorf("Expected unique attribute 'name', got %s", sdpItem.GetUniqueAttribute())
			}

			if sdpItem.GetHealth() != sdp.Health_HEALTH_OK {
				t.Errorf("Expected health OK, got %s", sdpItem.GetHealth())
			}

			if err := sdpItem.Validate(); err != nil {
				t.Fatalf("Item validation failed: %v", err)
			}

			log.Printf("Verified item attributes for snapshot %s", integrationTestSnapshotName)
		})

		t.Run("VerifyLinkedItems", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Verifying linked items for snapshot %s", integrationTestSnapshotName)

			snapshotWrapper := manual.NewComputeSnapshot(
				clients.NewSnapshotsClient(snapshotClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, integrationTestResourceGroup)},
			)
			scope := snapshotWrapper.Scopes()[0]

			snapshotAdapter := sources.WrapperToAdapter(snapshotWrapper, sdpcache.NewNoOpCache())
			sdpItem, qErr := snapshotAdapter.Get(ctx, scope, integrationTestSnapshotName, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			linkedQueries := sdpItem.GetLinkedItemQueries()
			log.Printf("Found %d linked item queries for snapshot %s", len(linkedQueries), integrationTestSnapshotName)

			// The snapshot was created from a disk, so we expect a link to the source disk
			var hasDiskLink bool
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
				}

				if query.GetType() == azureshared.ComputeDisk.String() {
					hasDiskLink = true
					if query.GetMethod() != sdp.QueryMethod_GET {
						t.Errorf("Expected disk link method to be GET, got %s", query.GetMethod())
					}
					if query.GetQuery() != integrationTestDiskForSnapName {
						t.Errorf("Expected disk link query to be %s, got %s", integrationTestDiskForSnapName, query.GetQuery())
					}
					if bp != nil {
						if bp.GetIn() != true {
							t.Error("Expected disk blast propagation In=true, got false")
						}
						if bp.GetOut() != false {
							t.Error("Expected disk blast propagation Out=false, got true")
						}
					}
				}

				log.Printf("Verified linked item query: Type=%s, Method=%s, Query=%s, Scope=%s",
					query.GetType(), query.GetMethod(), query.GetQuery(), query.GetScope())
			}

			if !hasDiskLink {
				t.Error("Expected to find a link to the source disk")
			}

			log.Printf("Verified %d linked item queries for snapshot %s", len(linkedQueries), integrationTestSnapshotName)
		})
	})

	t.Run("Teardown", func(t *testing.T) {
		ctx := t.Context()

		// Delete snapshot first
		err := deleteSnapshot(ctx, snapshotClient, integrationTestResourceGroup, integrationTestSnapshotName)
		if err != nil {
			t.Fatalf("Failed to delete snapshot: %v", err)
		}

		// Delete the source disk
		err = deleteDisk(ctx, diskClient, integrationTestResourceGroup, integrationTestDiskForSnapName)
		if err != nil {
			t.Fatalf("Failed to delete disk: %v", err)
		}
	})
}

// createSnapshot creates an Azure snapshot from a source disk (idempotent)
func createSnapshot(ctx context.Context, client *armcompute.SnapshotsClient, resourceGroupName, snapshotName, location, sourceDiskID string) error {
	existingSnapshot, err := client.Get(ctx, resourceGroupName, snapshotName, nil)
	if err == nil {
		if existingSnapshot.Properties != nil && existingSnapshot.Properties.ProvisioningState != nil {
			state := *existingSnapshot.Properties.ProvisioningState
			if state == "Succeeded" {
				log.Printf("Snapshot %s already exists with state %s, skipping creation", snapshotName, state)
				return nil
			}
			log.Printf("Snapshot %s exists but in state %s, will wait for it", snapshotName, state)
		} else {
			log.Printf("Snapshot %s already exists, skipping creation", snapshotName)
			return nil
		}
	}

	poller, err := client.BeginCreateOrUpdate(ctx, resourceGroupName, snapshotName, armcompute.Snapshot{
		Location: ptr.To(location),
		Properties: &armcompute.SnapshotProperties{
			CreationData: &armcompute.CreationData{
				CreateOption:    ptr.To(armcompute.DiskCreateOptionCopy),
				SourceResourceID: ptr.To(sourceDiskID),
			},
		},
		Tags: map[string]*string{
			"purpose": ptr.To("overmind-integration-tests"),
			"test":    ptr.To("compute-snapshot"),
		},
	}, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusConflict {
			log.Printf("Snapshot %s already exists (conflict), skipping creation", snapshotName)
			return nil
		}
		return fmt.Errorf("failed to begin creating snapshot: %w", err)
	}

	resp, err := poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to create snapshot: %w", err)
	}

	if resp.Properties == nil || resp.Properties.ProvisioningState == nil {
		return fmt.Errorf("snapshot created but provisioning state is unknown")
	}

	provisioningState := *resp.Properties.ProvisioningState
	if provisioningState != "Succeeded" {
		return fmt.Errorf("snapshot provisioning state is %s, expected Succeeded", provisioningState)
	}

	log.Printf("Snapshot %s created successfully with provisioning state: %s", snapshotName, provisioningState)
	return nil
}

// waitForSnapshotAvailable polls until the snapshot is available via the Get API
func waitForSnapshotAvailable(ctx context.Context, client *armcompute.SnapshotsClient, resourceGroupName, snapshotName string) error {
	maxAttempts := 20
	pollInterval := 5 * time.Second

	log.Printf("Waiting for snapshot %s to be available via API...", snapshotName)

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		resp, err := client.Get(ctx, resourceGroupName, snapshotName, nil)
		if err != nil {
			var respErr *azcore.ResponseError
			if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
				log.Printf("Snapshot %s not yet available (attempt %d/%d), waiting %v...", snapshotName, attempt, maxAttempts, pollInterval)
				time.Sleep(pollInterval)
				continue
			}
			return fmt.Errorf("error checking snapshot availability: %w", err)
		}

		if resp.Properties != nil && resp.Properties.ProvisioningState != nil {
			state := *resp.Properties.ProvisioningState
			if state == "Succeeded" {
				log.Printf("Snapshot %s is available with provisioning state: %s", snapshotName, state)
				return nil
			}
			if state == "Failed" {
				return fmt.Errorf("snapshot provisioning failed with state: %s", state)
			}
			log.Printf("Snapshot %s provisioning state: %s (attempt %d/%d), waiting...", snapshotName, state, attempt, maxAttempts)
			time.Sleep(pollInterval)
			continue
		}

		log.Printf("Snapshot %s is available", snapshotName)
		return nil
	}

	return fmt.Errorf("timeout waiting for snapshot %s to be available after %d attempts", snapshotName, maxAttempts)
}

// deleteSnapshot deletes an Azure snapshot
func deleteSnapshot(ctx context.Context, client *armcompute.SnapshotsClient, resourceGroupName, snapshotName string) error {
	poller, err := client.BeginDelete(ctx, resourceGroupName, snapshotName, nil)
	if err != nil {
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			log.Printf("Snapshot %s not found, skipping deletion", snapshotName)
			return nil
		}
		return fmt.Errorf("failed to begin deleting snapshot: %w", err)
	}

	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to delete snapshot: %w", err)
	}

	log.Printf("Snapshot %s deleted successfully", snapshotName)
	return nil
}
