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

func TestComputeSnapshotsIntegration(t *testing.T) {
	projectID := os.Getenv("GCP_PROJECT_ID")
	if projectID == "" {
		t.Skip("GCP_PROJECT_ID environment variable not set")
	}

	zone := os.Getenv("GCP_ZONE")
	if zone == "" {
		t.Skip("GCP_ZONE environment variable not set")
	}

	snapshotName := "integration-test-snapshot"
	diskName := "integration-test-disk-for-snapshot"

	ctx := context.Background()

	// Create a new Compute Snapshots client
	client, err := compute.NewSnapshotsRESTClient(ctx)
	if err != nil {
		t.Fatalf("NewSnapshotsRESTClient: %v", err)
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

		err := createComputeSnapshot(ctx, client, projectID, zone, snapshotName, diskName)
		if err != nil {
			t.Fatalf("Failed to create compute snapshot: %v", err)
		}
	})

	t.Run("ListSnapshots", func(t *testing.T) {
		log.Printf("Listing snapshots in project %s", projectID)

		snapshotsWrapper := manual.NewComputeSnapshot(gcpshared.NewComputeSnapshotsClient(client), projectID)
		scope := snapshotsWrapper.Scopes()[0]

		snapshotsAdapter := sources.WrapperToAdapter(snapshotsWrapper)
		sdpItems, err := snapshotsAdapter.List(ctx, scope, true)
		if err != nil {
			t.Fatalf("Failed to list compute snapshots: %v", err)
		}

		if len(sdpItems) < 1 {
			t.Fatalf("Expected at least one compute snapshot, got %d", len(sdpItems))
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
			t.Fatalf("Expected to find snapshot %s in the list of compute snapshots", snapshotName)
		}

		log.Printf("Found %d snapshots in project %s", len(sdpItems), projectID)
	})

	t.Run("GetSnapshot", func(t *testing.T) {
		log.Printf("Retrieving snapshot %s in project %s", snapshotName, projectID)

		snapshotsWrapper := manual.NewComputeSnapshot(gcpshared.NewComputeSnapshotsClient(client), projectID)
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

		log.Printf("Successfully retrieved snapshot %s in project %s", snapshotName, projectID)
	})

	t.Run("Teardown", func(t *testing.T) {
		req := &computepb.DeleteSnapshotRequest{
			Project:  projectID,
			Snapshot: snapshotName,
		}

		op, err := client.Delete(ctx, req)
		if err != nil {
			t.Fatalf("Failed to delete compute snapshot: %v", err)
		}

		if err := op.Wait(ctx); err != nil {
			t.Fatalf("Failed to wait for snapshot deletion operation: %v", err)
		}

		log.Printf("Compute snapshot %s deleted successfully", snapshotName)

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

// createComputeSnapshot creates a GCP Compute Snapshot with the given parameters.
func createComputeSnapshot(ctx context.Context, client *compute.SnapshotsClient, projectID, zone, snapshotName, diskName string) error {
	snapshot := &computepb.Snapshot{
		Name: ptr.To(snapshotName),
		SourceDisk: ptr.To(fmt.Sprintf(
			"projects/%s/zones/%s/disks/%s",
			projectID, zone, diskName,
		)),
		Labels: map[string]string{
			"test": "integration",
		},
		StorageLocations: []string{"us-central1"},
	}

	// Create the snapshot
	req := &computepb.InsertSnapshotRequest{
		Project:          projectID,
		SnapshotResource: snapshot,
	}

	op, err := client.Insert(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to create snapshot: %w", err)
	}

	if err := op.Wait(ctx); err != nil {
		return fmt.Errorf("failed to wait for snapshot creation operation: %w", err)
	}

	log.Printf("Snapshot %s created successfully in project %s", snapshotName, projectID)
	return nil
}
