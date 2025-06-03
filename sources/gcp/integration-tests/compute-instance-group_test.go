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

func TestComputeInstanceGroupsIntegration(t *testing.T) {
	projectID := os.Getenv("GCP_PROJECT_ID")
	if projectID == "" {
		t.Skip("GCP_PROJECT_ID environment variable not set")
	}

	zone := os.Getenv("GCP_ZONE")
	if zone == "" {
		t.Skip("GCP_ZONE environment variable not set")
	}

	instanceGroupName := "integration-test-instance-group"

	ctx := context.Background()

	// Create a new Compute InstanceGroups client
	client, err := compute.NewInstanceGroupsRESTClient(ctx)
	if err != nil {
		t.Fatalf("NewInstanceGroupsRESTClient: %v", err)
	}
	defer client.Close()

	t.Run("Setup", func(t *testing.T) {
		err := createInstanceGroup(ctx, client, projectID, zone, instanceGroupName)
		if err != nil {
			t.Fatalf("Failed to create instance group: %v", err)
		}
	})

	t.Run("ListInstanceGroups", func(t *testing.T) {
		log.Printf("Listing instance groups in project %s, zone %s", projectID, zone)

		instanceGroupWrapper := manual.NewComputeInstanceGroup(gcpshared.NewComputeInstanceGroupsClient(client), projectID, zone)
		scope := instanceGroupWrapper.Scopes()[0]

		adapter := sources.WrapperToAdapter(instanceGroupWrapper)
		sdpItems, err := adapter.List(ctx, scope, true)
		if err != nil {
			t.Fatalf("Failed to list instance groups: %v", err)
		}

		if len(sdpItems) < 1 {
			t.Fatalf("Expected at least one instance group, got %d", len(sdpItems))
		}

		var found bool
		for _, item := range sdpItems {
			uniqueAttrKey := item.GetUniqueAttribute()
			v, err := item.GetAttributes().Get(uniqueAttrKey)
			if err == nil && v == instanceGroupName {
				found = true
				break
			}
		}

		if !found {
			t.Fatalf("Expected to find instance group %s in the list of instance groups", instanceGroupName)
		}

		log.Printf("Found %d instance groups in project %s, zone %s", len(sdpItems), projectID, zone)
	})

	t.Run("GetInstanceGroup", func(t *testing.T) {
		log.Printf("Retrieving instance group %s in project %s, zone %s", instanceGroupName, projectID, zone)

		instanceGroupWrapper := manual.NewComputeInstanceGroup(gcpshared.NewComputeInstanceGroupsClient(client), projectID, zone)
		scope := instanceGroupWrapper.Scopes()[0]

		adapter := sources.WrapperToAdapter(instanceGroupWrapper)
		sdpItem, qErr := adapter.Get(ctx, scope, instanceGroupName, true)
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

		if uniqueAttrValue != instanceGroupName {
			t.Fatalf("Expected unique attribute value to be %s, got %s", instanceGroupName, uniqueAttrValue)
		}

		log.Printf("Successfully retrieved instance group %s in project %s, zone %s", instanceGroupName, projectID, zone)
	})

	t.Run("Teardown", func(t *testing.T) {
		req := &computepb.DeleteInstanceGroupRequest{
			Project:       projectID,
			Zone:          zone,
			InstanceGroup: instanceGroupName,
		}

		op, err := client.Delete(ctx, req)
		if err != nil {
			t.Fatalf("Failed to delete instance group: %v", err)
		}

		if err := op.Wait(ctx); err != nil {
			t.Fatalf("Failed to wait for instance group deletion operation: %v", err)
		}

		log.Printf("Instance group %s deleted successfully in project %s, zone %s", instanceGroupName, projectID, zone)
	})
}

func createInstanceGroup(ctx context.Context, client *compute.InstanceGroupsClient, projectID, zone, instanceGroupName string) error {
	instanceGroup := &computepb.InstanceGroup{
		Name: ptr.To(instanceGroupName),
	}

	req := &computepb.InsertInstanceGroupRequest{
		Project:               projectID,
		Zone:                  zone,
		InstanceGroupResource: instanceGroup,
	}

	op, err := client.Insert(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to create instance group: %w", err)
	}

	if err := op.Wait(ctx); err != nil {
		return fmt.Errorf("failed to wait for instance group creation operation: %w", err)
	}

	log.Printf("Instance group %s created successfully in project %s, zone %s", instanceGroupName, projectID, zone)
	return nil
}
