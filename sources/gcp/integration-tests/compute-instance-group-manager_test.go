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

func TestComputeInstanceGroupManagerIntegration(t *testing.T) {
	projectID := os.Getenv("GCP_PROJECT_ID")
	if projectID == "" {
		t.Skip("GCP_PROJECT_ID environment variable not set")
	}

	zone := os.Getenv("GCP_ZONE")
	if zone == "" {
		t.Skip("GCP_ZONE environment variable not set")
	}

	instanceGroupManagerName := "overmind-test-instance-group-manager"
	templateName := "overmind-integration-test-template"

	ctx := context.Background()

	instanceGroupManagerClient, err := compute.NewInstanceGroupManagersRESTClient(ctx)
	if err != nil {
		t.Fatalf("NewRegionInstanceGroupManagersRESTClient: %v", err)
	}
	defer instanceGroupManagerClient.Close()

	instanceTemplatesClient, err := compute.NewInstanceTemplatesRESTClient(ctx)
	if err != nil {
		t.Fatalf("NewInstanceTemplatesRESTClient: %v", err)
	}
	defer instanceTemplatesClient.Close()

	// Setup: create instance template and instance group manager
	t.Run("Setup", func(t *testing.T) {
		err := createInstanceTemplate(ctx, instanceTemplatesClient, projectID, templateName)
		if err != nil {
			t.Fatalf("Failed to create instance template: %v", err)
		}
		err = createInstanceGroupManager(ctx, instanceGroupManagerClient, projectID, zone, instanceGroupManagerName, templateName)
		if err != nil {
			t.Fatalf("Failed to create instance group manager: %v", err)
		}
	})

	t.Run("Run", func(t *testing.T) {
		log.Printf("Running integration test for Compute Instance Group Manager in project %s, zone %s", projectID, zone)

		instanceGroupManagerWrapper := manual.NewComputeInstanceGroupManager(gcpshared.NewComputeInstanceGroupManagerClient(instanceGroupManagerClient), projectID, zone)
		scope := instanceGroupManagerWrapper.Scopes()[0]

		instanceGroupManagerAdapter := sources.WrapperToAdapter(instanceGroupManagerWrapper)
		sdpItem, qErr := instanceGroupManagerAdapter.Get(ctx, scope, instanceGroupManagerName, true)
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

		if uniqueAttrValue != instanceGroupManagerName {
			t.Fatalf("Expected unique attribute value to be %s, got %s", instanceGroupManagerName, uniqueAttrValue)
		}
		// [SPEC] The only two linked item queries being created at the moment are one Instance Template and Instance Group
		{
			if len(sdpItem.GetLinkedItemQueries()) != 2 {
				t.Fatalf("Expected 1 linked item query, got: %d", len(sdpItem.GetLinkedItemQueries()))
			}

			// [SPEC] Ensure Instance Template is present
			linkedItem := sdpItem.GetLinkedItemQueries()[0]
			if linkedItem.GetQuery().GetType() != gcpshared.ComputeInstanceTemplate.String() {
				t.Fatalf("Expected linked item type to be %s, got: %s", gcpshared.ComputeInstanceTemplate, linkedItem.GetQuery().GetType())
			}

			if linkedItem.GetQuery().GetQuery() != templateName {
				t.Fatalf("Expected linked item query to be %s, got: %s", instanceGroupManagerName, linkedItem.GetQuery().GetQuery())
			}

			if linkedItem.GetQuery().GetScope() != projectID {
				t.Fatalf("Expected linked item scope to be %s, got: %s", projectID, linkedItem.GetQuery().GetScope())
			}
		}

		sdpItems, err := instanceGroupManagerAdapter.List(ctx, scope, true)
		if err != nil {
			t.Fatalf("Failed to list instance group managers: %v", err)
		}

		if len(sdpItems) < 1 {
			t.Fatalf("Expected at least one instance group manager, got %d", len(sdpItems))
		}

		var found bool
		for _, item := range sdpItems {
			if v, err := item.GetAttributes().Get(uniqueAttrKey); err == nil && v == instanceGroupManagerName {
				found = true
				break
			}
		}

		if !found {
			t.Fatalf("Expected to find instance group manager %s in the list", instanceGroupManagerName)
		}
	})

	t.Run("Teardown", func(t *testing.T) {
		err := deleteInstanceGroupManager(ctx, instanceGroupManagerClient, projectID, zone, instanceGroupManagerName)
		if err != nil {
			t.Fatalf("Failed to delete instance group manager: %v", err)
		}
		err = deleteInstanceTemplate(ctx, instanceTemplatesClient, projectID, templateName)
		if err != nil {
			t.Fatalf("Failed to delete instance template: %v", err)
		}
	})
}

// createInstanceTemplate creates a GCP Compute Engine instance template.
func createInstanceTemplate(ctx context.Context, client *compute.InstanceTemplatesClient, projectID, templateName string) error {
	template := &computepb.InstanceTemplate{
		Name: ptr.To(templateName),
		Properties: &computepb.InstanceProperties{
			MachineType: ptr.To("e2-micro"),
			Disks: []*computepb.AttachedDisk{
				{
					Boot:       ptr.To(true),
					AutoDelete: ptr.To(true),
					Type:       ptr.To("PERSISTENT"),
					InitializeParams: &computepb.AttachedDiskInitializeParams{
						SourceImage: ptr.To("projects/debian-cloud/global/images/family/debian-11"),
					},
				},
			},
			NetworkInterfaces: []*computepb.NetworkInterface{
				{
					Network: ptr.To("global/networks/default"),
				},
			},
		},
	}

	req := &computepb.InsertInstanceTemplateRequest{
		Project:                  projectID,
		InstanceTemplateResource: template,
	}

	op, err := client.Insert(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to create instance template: %w", err)
	}
	if err := op.Wait(ctx); err != nil {
		return fmt.Errorf("failed to wait for instance template creation: %w", err)
	}
	log.Printf("Instance template %s created successfully in project %s", templateName, projectID)
	return nil
}

// deleteInstanceTemplate deletes a GCP Compute Engine instance template.
func deleteInstanceTemplate(ctx context.Context, client *compute.InstanceTemplatesClient, projectID, templateName string) error {
	req := &computepb.DeleteInstanceTemplateRequest{
		Project:          projectID,
		InstanceTemplate: templateName,
	}
	op, err := client.Delete(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to delete instance template: %w", err)
	}
	if err := op.Wait(ctx); err != nil {
		return fmt.Errorf("failed to wait for instance template deletion: %w", err)
	}
	log.Printf("Instance template %s deleted successfully", templateName)
	return nil
}

// createInstanceGroupManager creates a GCP Compute Engine instance group manager.
func createInstanceGroupManager(ctx context.Context, client *compute.InstanceGroupManagersClient, projectID, zone, instanceGroupManagerName, templateName string) error {
	instanceGroupManager := &computepb.InstanceGroupManager{
		Name:             ptr.To(instanceGroupManagerName),
		InstanceTemplate: ptr.To(fmt.Sprintf("projects/%s/global/instanceTemplates/%s", projectID, templateName)),
		TargetSize:       ptr.To(int32(1)),
	}

	req := &computepb.InsertInstanceGroupManagerRequest{
		Project:                      projectID,
		Zone:                         zone,
		InstanceGroupManagerResource: instanceGroupManager,
	}

	op, err := client.Insert(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to create instance group manager: %w", err)
	}
	if err := op.Wait(ctx); err != nil {
		return fmt.Errorf("failed to wait for instance group manager creation: %w", err)
	}
	log.Printf("Instance group manager %s created successfully in", instanceGroupManagerName)
	return nil
}

// deleteInstanceGroupManager deletes a GCP Compute Engine instance group manager.
func deleteInstanceGroupManager(ctx context.Context, client *compute.InstanceGroupManagersClient, projectID, zone, instanceGroupManagerName string) error {
	req := &computepb.DeleteInstanceGroupManagerRequest{
		Project:              projectID,
		Zone:                 zone,
		InstanceGroupManager: instanceGroupManagerName,
	}
	op, err := client.Delete(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to delete instance group manager: %w", err)
	}
	if err := op.Wait(ctx); err != nil {
		return fmt.Errorf("failed to wait for instance group manager deletion: %w", err)
	}
	log.Printf("Instance group manager %s deleted successfully", instanceGroupManagerName)
	return nil
}
