package integrationtests

import (
	"context"
	"fmt"
	"os"
	"testing"

	compute "cloud.google.com/go/compute/apiv1"
	"cloud.google.com/go/compute/apiv1/computepb"
	log "github.com/sirupsen/logrus"
	"k8s.io/utils/ptr"

	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/gcp/manual"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

func TestComputeInstantSnapshotsIntegration(t *testing.T) {
	projectID := os.Getenv("GCP_PROJECT_ID")
	if projectID == "" {
		t.Skip("GCP_PROJECT_ID environment variable not set")
	}

	zone := os.Getenv("GCP_ZONE")
	if zone == "" {
		t.Skip("GCP_ZONE environment variable not set")
	}

	snapshotName := "integration-test-instant-snapshot"
	diskName := "integration-test-disk-for-snapshot"
	diskFullName := fmt.Sprintf(
		"projects/%s/zones/%s/disks/%s",
		projectID, zone, diskName,
	)

	ctx := context.Background()

	// Create a new Compute InstantSnapshots client
	client, err := compute.NewInstantSnapshotsRESTClient(ctx)
	if err != nil {
		t.Fatalf("NewInstantSnapshotsRESTClient: %v", err)
	}
	defer client.Close()

	diskClient, err := compute.NewDisksRESTClient(ctx)
	if err != nil {
		t.Fatalf("NewDisksRESTClient: %v", err)
	}
	defer diskClient.Close()

	t.Run("Setup", func(t *testing.T) {
		err = createDisk(ctx, diskClient, projectID, zone, diskName)
		if err != nil {
			t.Fatalf("Failed to create source disk: %v", err)
		}

		err := createInstantSnapshot(ctx, client, projectID, zone, snapshotName, diskFullName)
		if err != nil {
			t.Fatalf("Failed to create instant snapshot: %v", err)
		}
	})

	t.Run("ListInstantSnapshots", func(t *testing.T) {
		log.Printf("Listing instant snapshots in project %s, zone %s", projectID, zone)

		snapshotsWrapper := manual.NewComputeInstantSnapshot(gcpshared.NewComputeInstantSnapshotsClient(client), projectID, zone)
		scope := snapshotsWrapper.Scopes()[0]

		snapshotsAdapter := sources.WrapperToAdapter(snapshotsWrapper)
		sdpItems, err := snapshotsAdapter.List(ctx, scope, true)
		if err != nil {
			t.Fatalf("Failed to list instant snapshots: %v", err)
		}

		if len(sdpItems) < 1 {
			t.Fatalf("Expected at least one instant snapshot, got %d", len(sdpItems))
		}

		var found bool
		for _, item := range sdpItems {
			uniqueAttrKey := item.GetUniqueAttribute()
			if v, err := item.GetAttributes().Get(uniqueAttrKey); err == nil && v == snapshotName {
				found = true
				break
			}
		}

		if !found {
			t.Fatalf("Expected to find snapshot %s in the list of instant snapshots", snapshotName)
		}

		log.Printf("Found %d instant snapshots in project %s, zone %s", len(sdpItems), projectID, zone)
	})

	t.Run("GetInstantSnapshot", func(t *testing.T) {
		log.Printf("Retrieving instant snapshot %s in project %s, zone %s", snapshotName, projectID, zone)

		snapshotsWrapper := manual.NewComputeInstantSnapshot(gcpshared.NewComputeInstantSnapshotsClient(client), projectID, zone)
		scope := snapshotsWrapper.Scopes()[0]

		snapshotsAdapter := sources.WrapperToAdapter(snapshotsWrapper)
		sdpItem, qErr := snapshotsAdapter.Get(ctx, scope, snapshotName, true)
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

		if uniqueAttrValue != snapshotName {
			t.Fatalf("Expected unique attribute value to be %s, got %s", snapshotName, uniqueAttrValue)
		}

		// [SPEC] The only two linked item queries being created at the moment are one Instance Template and Instance Group
		{
			if len(sdpItem.GetLinkedItemQueries()) != 1 {
				t.Fatalf("Expected 1 linked item query, got: %d", len(sdpItem.GetLinkedItemQueries()))
			}

			// [SPEC] Ensure Source Disk is linked
			linkedItem := sdpItem.GetLinkedItemQueries()[0]
			if linkedItem.GetQuery().GetType() != gcpshared.ComputeDisk.String() {
				t.Fatalf("Expected linked item type to be %s, got: %s", gcpshared.ComputeDisk, linkedItem.GetQuery().GetType())
			}

			if linkedItem.GetQuery().GetQuery() != diskName {
				t.Fatalf("Expected linked item query to be %s, got: %s", diskName, linkedItem.GetQuery().GetQuery())
			}

			if linkedItem.GetQuery().GetScope() != gcpshared.ZonalScope(projectID, zone) {
				t.Fatalf("Expected linked item scope to be %s, got: %s", gcpshared.ZonalScope(projectID, zone), linkedItem.GetQuery().GetScope())
			}
		}

		log.Printf("Successfully retrieved instant snapshot %s in project %s, zone %s", snapshotName, projectID, zone)
	})

	t.Run("Teardown", func(t *testing.T) {
		req := &computepb.DeleteInstantSnapshotRequest{
			Project:         projectID,
			Zone:            zone,
			InstantSnapshot: snapshotName,
		}

		op, err := client.Delete(ctx, req)
		if err != nil {
			t.Fatalf("Failed to delete instant snapshot: %v", err)
		}

		if err := op.Wait(ctx); err != nil {
			t.Fatalf("Failed to wait for instant snapshot deletion operation: %v", err)
		}

		log.Printf("Instant snapshot %s deleted successfully", snapshotName)

		diskReq := &computepb.DeleteDiskRequest{
			Project: projectID,
			Zone:    zone,
			Disk:    diskName,
		}

		diskOp, err := diskClient.Delete(ctx, diskReq)
		if err != nil {
			t.Fatalf("Failed to delete disk: %v", err)
		}

		if err := diskOp.Wait(ctx); err != nil {
			t.Fatalf("Failed to wait for disk deletion operation: %v", err)
		}

		log.Printf("Disk %s deleted successfully in project %s, zone %s", diskName, projectID, zone)
	})
}

// createInstantSnapshot creates a GCP Compute Instant Snapshot with the given parameters.
func createInstantSnapshot(ctx context.Context, client *compute.InstantSnapshotsClient, projectID, zone, snapshotName, diskName string) error {
	snapshot := &computepb.InstantSnapshot{
		Name:       ptr.To(snapshotName),
		SourceDisk: ptr.To(diskName),
		Labels: map[string]string{
			"test": "integration",
		},
	}

	// Create the instant snapshot
	req := &computepb.InsertInstantSnapshotRequest{
		Project:                 projectID,
		Zone:                    zone,
		InstantSnapshotResource: snapshot,
	}

	op, err := client.Insert(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to create instant snapshot: %w", err)
	}

	if err := op.Wait(ctx); err != nil {
		return fmt.Errorf("failed to wait for instant snapshot creation operation: %w", err)
	}

	log.Printf("Instant snapshot %s created successfully in project %s, zone %s", snapshotName, projectID, zone)
	return nil
}
