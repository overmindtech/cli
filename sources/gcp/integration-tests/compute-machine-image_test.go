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

func TestComputeMachineImagesIntegration(t *testing.T) {
	projectID := os.Getenv("GCP_PROJECT_ID")
	if projectID == "" {
		t.Skip("GCP_PROJECT_ID environment variable not set")
	}

	zone := os.Getenv("GCP_ZONE")
	if zone == "" {
		t.Skip("GCP_ZONE environment variable not set")
	}

	region := os.Getenv("GCP_REGION")
	if region == "" {
		t.Skip("GCP_REGION environment variable not set")
	}

	machineImageName := "integration-test-machine-image"
	sourceInstanceName := "integration-test-instance"

	ctx := context.Background()

	// Create a new Compute Machine Images client
	client, err := compute.NewMachineImagesRESTClient(ctx)
	if err != nil {
		t.Fatalf("NewMachineImagesRESTClient: %v", err)
	}
	defer client.Close()

	instanceClient, err := compute.NewInstancesRESTClient(ctx)
	if err != nil {
		t.Fatalf("NewInstancesRESTClient: %v", err)
	}
	defer instanceClient.Close()

	t.Run("Setup", func(t *testing.T) {
		err = createComputeInstance(ctx, instanceClient, projectID, zone, sourceInstanceName, "default", "default", region)
		if err != nil {
			t.Fatalf("Failed to create source instance: %v", err)
		}

		err := createComputeMachineImage(ctx, client, projectID, machineImageName, sourceInstanceName)
		if err != nil {
			t.Fatalf("Failed to create compute machine image: %v", err)
		}
	})

	t.Run("ListMachineImages", func(t *testing.T) {
		log.Printf("Listing machine images in project %s", projectID)

		machineImagesWrapper := manual.NewComputeMachineImage(gcpshared.NewComputeMachineImageClient(client), projectID)
		scope := machineImagesWrapper.Scopes()[0]

		machineImagesAdapter := sources.WrapperToAdapter(machineImagesWrapper)
		sdpItems, err := machineImagesAdapter.List(ctx, scope, true)
		if err != nil {
			t.Fatalf("Failed to list compute machine images: %v", err)
		}

		if len(sdpItems) < 1 {
			t.Fatalf("Expected at least one compute machine image, got %d", len(sdpItems))
		}

		var found bool
		for _, item := range sdpItems {
			uniqueAttrKey := item.GetUniqueAttribute()
			if v, err := item.GetAttributes().Get(uniqueAttrKey); err == nil && v == machineImageName {
				found = true
				break
			}
		}

		if !found {
			t.Fatalf("Expected to find machine image %s in the list of compute machine images", machineImageName)
		}

		log.Printf("Found %d machine images in project %s", len(sdpItems), projectID)
	})

	t.Run("GetMachineImage", func(t *testing.T) {
		log.Printf("Retrieving machine image %s in project %s", machineImageName, projectID)

		machineImagesWrapper := manual.NewComputeMachineImage(gcpshared.NewComputeMachineImageClient(client), projectID)
		scope := machineImagesWrapper.Scopes()[0]

		machineImagesAdapter := sources.WrapperToAdapter(machineImagesWrapper)
		sdpItem, qErr := machineImagesAdapter.Get(ctx, scope, machineImageName, true)
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

		if uniqueAttrValue != machineImageName {
			t.Fatalf("Expected unique attribute value to be %s, got %s", machineImageName, uniqueAttrValue)
		}

		log.Printf("Successfully retrieved machine image %s in project %s", machineImageName, projectID)
	})

	t.Run("Teardown", func(t *testing.T) {
		req := &computepb.DeleteMachineImageRequest{
			Project:      projectID,
			MachineImage: machineImageName,
		}

		op, err := client.Delete(ctx, req)
		if err != nil {
			t.Fatalf("Failed to delete compute machine image: %v", err)
		}

		if err := op.Wait(ctx); err != nil {
			t.Fatalf("Failed to wait for machine image deletion operation: %v", err)
		}

		log.Printf("Compute machine image %s deleted successfully", machineImageName)

		instanceReq := &computepb.DeleteInstanceRequest{
			Project:  projectID,
			Zone:     zone,
			Instance: sourceInstanceName,
		}

		instanceOp, err := instanceClient.Delete(ctx, instanceReq)
		if err != nil {
			t.Fatalf("Failed to delete source instance: %v", err)
		}

		if err := instanceOp.Wait(ctx); err != nil {
			t.Fatalf("Failed to wait for source instance deletion operation: %v", err)
		}

		log.Printf("Source instance %s deleted successfully in project %s, zone %s", sourceInstanceName, projectID, zone)
	})
}

// createComputeMachineImage creates a GCP Compute Machine Image with the given parameters.
func createComputeMachineImage(ctx context.Context, client *compute.MachineImagesClient, projectID, machineImageName, sourceInstanceName string) error {
	machineImage := &computepb.MachineImage{
		Name: ptr.To(machineImageName),
		SourceInstance: ptr.To(fmt.Sprintf(
			"projects/%s/zones/%s/instances/%s",
			projectID, "us-central1-a", sourceInstanceName,
		)),
		Labels: map[string]string{
			"test": "integration",
		},
	}

	req := &computepb.InsertMachineImageRequest{
		Project:              projectID,
		MachineImageResource: machineImage,
	}

	op, err := client.Insert(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to create machine image: %w", err)
	}

	if err := op.Wait(ctx); err != nil {
		return fmt.Errorf("failed to wait for machine image creation operation: %w", err)
	}

	log.Printf("Machine image %s created successfully in project %s", machineImageName, projectID)
	return nil
}
