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
	"github.com/overmindtech/cli/sources/gcp/adapters"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

func TestComputeDisksIntegration(t *testing.T) {
	projectID := os.Getenv("GCP_PROJECT_ID")
	if projectID == "" {
		t.Skip("GCP_PROJECT_ID environment variable not set")
	}

	zone := os.Getenv("GCP_ZONE")
	if zone == "" {
		t.Skip("GCP_ZONE environment variable not set")
	}

	diskName := "integration-test-disk"

	ctx := context.Background()

	// Create a new Compute Disks client
	diskClient, err := compute.NewDisksRESTClient(ctx)
	if err != nil {
		t.Fatalf("NewDisksRESTClient: %v", err)
	}
	defer diskClient.Close()

	t.Run("Setup", func(t *testing.T) {
		err := createDisk(ctx, diskClient, projectID, zone, diskName)
		if err != nil {
			t.Fatalf("Failed to create disk: %v", err)
		}
	})

	t.Run("ListDisks", func(t *testing.T) {
		log.Printf("Listing disks in project %s, zone %s", projectID, zone)

		disksWrapper := adapters.NewComputeDisk(gcpshared.NewComputeDiskClient(diskClient), projectID, zone)
		scope := disksWrapper.Scopes()[0]

		disksAdapter := sources.WrapperToAdapter(disksWrapper)
		sdpItems, err := disksAdapter.List(ctx, scope, true)
		if err != nil {
			t.Fatalf("Failed to list compute disks: %v", err)
		}

		if len(sdpItems) < 1 {
			t.Fatalf("Expected at least one compute disk, got %d", len(sdpItems))
		}

		var found bool
		for _, item := range sdpItems {
			uniqueAttrKey := item.GetUniqueAttribute()
			if v, err := item.GetAttributes().Get(uniqueAttrKey); err == nil && v == diskName {
				found = true
				break
			}
		}

		if !found {
			t.Fatalf("Expected to find disk %s in the list of compute disks", diskName)
		}

		log.Printf("Found %d disks in project %s, zone %s", len(sdpItems), projectID, zone)
	})

	t.Run("GetDisk", func(t *testing.T) {
		log.Printf("Retrieving disk %s in project %s, zone %s", diskName, projectID, zone)

		disksWrapper := adapters.NewComputeDisk(gcpshared.NewComputeDiskClient(diskClient), projectID, zone)
		scope := disksWrapper.Scopes()[0]

		disksAdapter := sources.WrapperToAdapter(disksWrapper)
		sdpItem, qErr := disksAdapter.Get(ctx, scope, diskName, true)
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

		if uniqueAttrValue != diskName {
			t.Fatalf("Expected unique attribute value to be %s, got %s", diskName, uniqueAttrValue)
		}

		log.Printf("Successfully retrieved disk %s in project %s, zone %s", diskName, projectID, zone)
	})

	t.Run("Teardown", func(t *testing.T) {
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

func createDisk(ctx context.Context, client *compute.DisksClient, projectID, zone, diskName string) error {
	disk := &computepb.Disk{
		Name:   ptr.To(diskName),
		SizeGb: ptr.To(int64(10)),
		Type: ptr.To(fmt.Sprintf(
			"projects/%s/zones/%s/diskTypes/pd-standard",
			projectID, zone,
		)),
		Labels: map[string]string{
			"test": "integration",
		},
	}

	req := &computepb.InsertDiskRequest{
		Project:      projectID,
		Zone:         zone,
		DiskResource: disk,
	}

	op, err := client.Insert(ctx, req)
	if err != nil {
		return fmt.Errorf("Failed to create disk: %w", err)
	}

	if err := op.Wait(ctx); err != nil {
		return fmt.Errorf("Failed to wait for disk creation operation: %w", err)
	}

	log.Printf("Disk %s created successfully in project %s, zone %s", diskName, projectID, zone)
	return nil
}
