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

func TestComputeInstanceIntegration(t *testing.T) {
	projectID := os.Getenv("GCP_PROJECT_ID")
	if projectID == "" {
		t.Skip("GCP_PROJECT_ID environment variable not set")
	}

	zone := os.Getenv("GCP_ZONE")
	if zone == "" {
		t.Skip("GCP_ZONE environment variable not set")
	}

	instanceName := "integration-test-instance"

	ctx := context.Background()

	// Create a new Compute Instance client
	client, err := compute.NewInstancesRESTClient(ctx)

	if err != nil {
		t.Fatalf("NewInstancesRESTClient: %v", err)
	}

	defer client.Close()

	t.Run("Setup", func(t *testing.T) {
		err := createComputeInstance(ctx, client, projectID, zone, instanceName)
		if err != nil {
			t.Fatalf("Failed to create compute instance: %v", err)
		}
	})

	t.Run("Run", func(t *testing.T) {
		log.Printf("Running integration test for Compute Instance in project %s, zone %s", projectID, zone)

		instanceWrapper := adapters.NewComputeInstance(gcpshared.NewComputeInstanceClient(client), projectID, zone)
		scope := instanceWrapper.Scopes()[0]

		instanceAdapter := sources.WrapperToAdapter(instanceWrapper)
		sdpItem, qErr := instanceAdapter.Get(ctx, scope, instanceName, true)
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

		if uniqueAttrValue != instanceName {
			t.Fatalf("Expected unique attribute value to be %s, got %s", instanceName, uniqueAttrValue)
		}

		sdpItems, err := instanceAdapter.List(ctx, scope, true)
		if err != nil {
			t.Fatalf("Failed to list compute instances: %v", err)
		}

		if len(sdpItems) < 1 {
			t.Fatalf("Expected at least one compute instance, got %d", len(sdpItems))
		}

		var found bool
		for _, item := range sdpItems {
			if v, err := item.GetAttributes().Get(uniqueAttrKey); err == nil && v == instanceName {
				found = true
				break
			}
		}

		if !found {
			t.Fatalf("Expected to find instance %s in the list of compute instances", instanceName)
		}
	})

	t.Run("Teardown", func(t *testing.T) {
		req := &computepb.DeleteInstanceRequest{
			Project:  projectID,
			Zone:     zone,
			Instance: instanceName,
		}

		op, err := client.Delete(ctx, req)
		if err != nil {
			t.Fatalf("Failed to delete compute instance: %v", err)
		}

		if err := op.Wait(ctx); err != nil {
			t.Fatalf("Failed to wait for instance deletion operation: %v", err)
		}

		log.Printf("Compute instance %s deleted successfully", instanceName)
	})
}

// createComputeInstance creates a GCP Compute Instance with the given parameters.
func createComputeInstance(ctx context.Context, client *compute.InstancesClient, projectID, zone, instanceName string) error {
	// Define the instance configuration
	instance := &computepb.Instance{
		Name:        ptr.To(instanceName),
		MachineType: ptr.To(fmt.Sprintf("zones/%s/machineTypes/e2-micro", zone)),
		Disks: []*computepb.AttachedDisk{
			{
				Boot:       ptr.To(true),
				AutoDelete: ptr.To(true),
				InitializeParams: &computepb.AttachedDiskInitializeParams{
					SourceImage: ptr.To("projects/debian-cloud/global/images/debian-12-bookworm-v20250415"),
					DiskSizeGb:  ptr.To(int64(10)),
				},
			},
		},
		NetworkInterfaces: []*computepb.NetworkInterface{
			{
				StackType: ptr.To("IPV4_ONLY"),
				//Subnetwork: ptr.To("default"),
			},
		},
		Labels: map[string]string{
			"test": "integration",
		},
	}

	// Create the instance
	req := &computepb.InsertInstanceRequest{
		Project:          projectID,
		Zone:             zone,
		InstanceResource: instance,
	}

	op, err := client.Insert(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to create instance: %w", err)
	}

	// Wait for the operation to complete
	if err := op.Wait(ctx); err != nil {
		return fmt.Errorf("failed to wait for instance creation operation: %w", err)
	}

	log.Printf("Instance %s created successfully in project %s, zone %s", instanceName, projectID, zone)
	return nil
}
