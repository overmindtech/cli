package integrationtests

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"testing"

	compute "cloud.google.com/go/compute/apiv1"
	"cloud.google.com/go/compute/apiv1/computepb"
	"github.com/googleapis/gax-go/v2/apierror"
	log "github.com/sirupsen/logrus"
	"k8s.io/utils/ptr"

	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/gcp/manual"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

func TestComputeAutoscalerIntegration(t *testing.T) {
	projectID := os.Getenv("GCP_PROJECT_ID")
	if projectID == "" {
		t.Skip("GCP_PROJECT_ID environment variable not set")
	}

	zone := os.Getenv("GCP_ZONE")
	if zone == "" {
		t.Skip("GCP_ZONE environment variable not set")
	}

	// Can replace with an environment-specific ID later.
	suffix := "default"

	// 3 resources to create:
	// Autoscaler -> Instance Group Manager -> Instance Template
	instanceTemplateName := "overmind-integration-test-instance-template-" + suffix
	instanceGroupManagerName := "overmind-integration-test-igm-" + suffix
	autoscalerName := "overmind-integration-test-autoscaler-" + suffix
	ctx := context.Background()

	// Create a new Compute Engine client
	client, err := compute.NewAutoscalersRESTClient(ctx)
	if err != nil {
		t.Fatalf("NewAutoscalersRESTClient: %v", err)
	}
	defer client.Close()

	itClient, err := compute.NewInstanceTemplatesRESTClient(ctx)
	if err != nil {
		t.Fatalf("NewInstanceTemplatesRESTClient: %v", err)
	}
	defer itClient.Close()

	igmClient, err := compute.NewInstanceGroupManagersRESTClient(ctx)
	if err != nil {
		t.Fatalf("NewInstanceGroupManagersRESTClient: %v", err)
	}
	defer igmClient.Close()

	t.Run("Setup", func(t *testing.T) {
		err := createComputeInstanceTemplate(ctx, itClient, projectID, instanceTemplateName)
		if err != nil {
			t.Fatalf("Failed to create compute instance template: %v", err)
		}

		err = createInstanceGroupManager(ctx, igmClient, projectID, zone, instanceGroupManagerName, instanceTemplateName)
		if err != nil {
			t.Fatalf("Failed to create instance group manager: %v", err)
		}

		fullIgmName := "projects/" + projectID + "/zones/" + zone + "/instanceGroupManagers/" + instanceGroupManagerName

		err = createComputeAutoscaler(ctx, client, fullIgmName, projectID, zone, autoscalerName)
		if err != nil {
			t.Fatalf("Failed to create compute autoscaler: %v", err)
		}
	})

	t.Run("Run", func(t *testing.T) {
		log.Printf("Running integration test for Compute Autoscaler in project %s, zone %s", projectID, zone)

		autoscalerWrapper := manual.NewComputeAutoscaler(gcpshared.NewComputeAutoscalerClient(client), projectID, zone)
		scope := autoscalerWrapper.Scopes()[0]

		autoscalerAdapter := sources.WrapperToAdapter(autoscalerWrapper)

		// [SPEC] GET against a valid resource name will return an SDP item wrapping the
		// available resource.
		sdpItem, err := autoscalerAdapter.Get(ctx, scope, autoscalerName, true)
		if err != nil {
			t.Fatalf("autoscalerAdapter.Get returned unexpected error: %v", err)
		}
		if sdpItem == nil {
			t.Fatalf("Expected sdpItem to be non-nil")
		}

		// [SPEC] The attributes contained in the SDP item directly match the attributes
		// from the GCP API.
		uniqueAttrKey := sdpItem.GetUniqueAttribute()
		uniqueAttrValue, err := sdpItem.GetAttributes().Get(uniqueAttrKey)
		if err != nil {
			t.Fatalf("Failed to get unique attribute: %v", err)
		}

		if uniqueAttrValue != autoscalerName {
			t.Fatalf("Expected unique attribute value to be %s, got %s", autoscalerName, uniqueAttrValue)
		}

		// [SPEC] The only linked item query is one Instance Group Manager.
		{
			if len(sdpItem.GetLinkedItemQueries()) != 1 {
				t.Fatalf("Expected 1 linked item query, got: %d", len(sdpItem.GetLinkedItemQueries()))
			}

			linkedItem := sdpItem.GetLinkedItemQueries()[0]
			if linkedItem.GetQuery().GetType() != gcpshared.ComputeInstanceGroupManager.String() {
				t.Fatalf("Expected linked item type to be %s, got: %s", gcpshared.ComputeInstanceGroupManager, linkedItem.GetQuery().GetType())
			}

			if linkedItem.GetQuery().GetQuery() != instanceGroupManagerName {
				t.Fatalf("Expected linked item query to be %s, got: %s", instanceGroupManagerName, linkedItem.GetQuery().GetQuery())
			}

			expectedScope := gcpshared.ZonalScope(projectID, zone)
			if linkedItem.GetQuery().GetScope() != expectedScope {
				t.Fatalf("Expected linked item scope to be %s, got: %s", expectedScope, linkedItem.GetQuery().GetScope())
			}
		}

		// [SPEC] The LIST operation for autoscalers will list all autoscalers in a given
		// scope.
		sdpItems, err := autoscalerAdapter.List(ctx, scope, true)
		if err != nil {
			t.Fatalf("Failed to list compute autoscalers: %v", err)
		}

		if len(sdpItems) < 1 {
			t.Fatalf("Expected at least one compute autoscaler, got %d", len(sdpItems))
		}

		// The LIST operation result should include our autoscaler.
		found := false
		for _, item := range sdpItems {
			if v, err := item.GetAttributes().Get(uniqueAttrKey); err == nil && v == autoscalerName {
				found = true
				break
			}
		}

		if !found {
			t.Fatalf("Expected to find autoscaler %s in list, but it was not found", autoscalerName)
		}
	})

	t.Run("Teardown", func(t *testing.T) {
		err := deleteComputeAutoscaler(ctx, client, projectID, zone, autoscalerName)
		if err != nil {
			t.Errorf("Warning: failed to delete compute autoscaler: %v", err)
		}

		err = deleteInstanceGroupManager(ctx, igmClient, projectID, zone, instanceGroupManagerName)
		if err != nil {
			t.Errorf("Warning: failed to delete instance group manager: %v", err)
		}

		err = deleteComputeInstanceTemplate(ctx, itClient, projectID, instanceTemplateName)
		if err != nil {
			t.Errorf("Warning: failed to delete compute instance template: %v", err)
		}
	})
}

// Create a compute instance template in GCP to test against. Uses a common Debian image
// and basic network configuration.
func createComputeInstanceTemplate(ctx context.Context, client *compute.InstanceTemplatesClient, projectID, name string) error {
	// Create a new instance template
	instanceTemplate := &computepb.InstanceTemplate{
		Name: ptr.To(name),
		Properties: &computepb.InstanceProperties{
			Disks: []*computepb.AttachedDisk{
				{
					AutoDelete: ptr.To(true),
					Boot:       ptr.To(true),
					DeviceName: ptr.To(name),
					InitializeParams: &computepb.AttachedDiskInitializeParams{
						DiskSizeGb:  ptr.To(int64(10)),
						DiskType:    ptr.To("pd-balanced"),
						SourceImage: ptr.To("projects/debian-cloud/global/images/debian-12-bookworm-v20250415"),
					},
					Mode: ptr.To("READ_WRITE"),
					Type: ptr.To("PERSISTENT"),

					// Labels? Tags?
				},
			},
			NetworkInterfaces: []*computepb.NetworkInterface{
				{
					AccessConfigs: []*computepb.AccessConfig{
						{
							Kind:        ptr.To("compute#accessConfig"),
							Name:        ptr.To("External NAT"),
							NetworkTier: ptr.To("PREMIUM"),
							Type:        ptr.To("ONE_TO_ONE_NAT"),
						},
					},
					Network:   ptr.To("projects/" + projectID + "/global/networks/default"),
					StackType: ptr.To("IPV4_ONLY"),
				},
			},
			MachineType: ptr.To("e2-micro"),
			Tags: &computepb.Tags{
				Items: []string{"overmind-test"},
			},
		},
	}

	// Create the instance template
	req := &computepb.InsertInstanceTemplateRequest{
		Project:                  projectID,
		InstanceTemplateResource: instanceTemplate,
	}

	op, err := client.Insert(ctx, req)
	if err != nil {
		var apiErr *apierror.APIError
		if errors.As(err, &apiErr) && apiErr.HTTPCode() == http.StatusConflict {
			log.Printf("Resource already exists in project, skipping creation: %v", err)
			return nil
		}

		return fmt.Errorf("failed to create resource: %w", err)
	}

	// Wait for the operation to complete
	if err := op.Wait(ctx); err != nil {
		return fmt.Errorf("Failed to wait for instance template operation: %w", err)
	}

	log.Printf("Instance template %s created successfully in project %s", name, projectID)
	return nil
}

// Delete a compute instance template.
func deleteComputeInstanceTemplate(ctx context.Context, client *compute.InstanceTemplatesClient, projectID, name string) error {
	req := &computepb.DeleteInstanceTemplateRequest{
		Project:          projectID,
		InstanceTemplate: name,
	}

	op, err := client.Delete(ctx, req)
	if err != nil {
		var apiErr *apierror.APIError
		if errors.As(err, &apiErr) && apiErr.HTTPCode() == http.StatusNotFound {
			log.Printf("Failed to find resource to delete: %v", err)
			return nil
		}

		return fmt.Errorf("failed to delete resource: %w", err)
	}

	if err := op.Wait(ctx); err != nil {
		return fmt.Errorf("failed to wait for instance template deletion operation: %w", err)
	}

	log.Printf("Instance template %s deleted successfully in project %s", name, projectID)
	return nil
}

// Create a compute autoscaler in GCP targeting the given instance group manager.
func createComputeAutoscaler(ctx context.Context, client *compute.AutoscalersClient, targetedInstanceGroupManager, projectID, zone, name string) error {
	// Create a new autoscaler
	autoscaler := &computepb.Autoscaler{
		Name:   ptr.To(name),
		Target: &targetedInstanceGroupManager,
		AutoscalingPolicy: &computepb.AutoscalingPolicy{
			MinNumReplicas: ptr.To(int32(0)),
			MaxNumReplicas: ptr.To(int32(1)),
			CpuUtilization: &computepb.AutoscalingPolicyCpuUtilization{
				UtilizationTarget: ptr.To(float64(0.6)),
			},
		},
	}

	req := &computepb.InsertAutoscalerRequest{
		Project:            projectID,
		Zone:               zone,
		AutoscalerResource: autoscaler,
	}

	op, err := client.Insert(ctx, req)
	if err != nil {
		var apiErr *apierror.APIError
		if errors.As(err, &apiErr) && apiErr.HTTPCode() == http.StatusConflict {
			log.Printf("Resource already exists in project, skipping creation: %v", err)
			return nil
		}

		return fmt.Errorf("failed to create resource: %w", err)
	}

	if err := op.Wait(ctx); err != nil {
		return fmt.Errorf("failed to wait for autoscaler creation operation: %w", err)
	}

	log.Printf("Autoscaler %s created successfully in project %s, zone %s", name, projectID, zone)
	return nil
}

// Delete a compute autoscaler in GCP.
func deleteComputeAutoscaler(ctx context.Context, client *compute.AutoscalersClient, projectID, zone, name string) error {
	req := &computepb.DeleteAutoscalerRequest{
		Project:    projectID,
		Zone:       zone,
		Autoscaler: name,
	}

	op, err := client.Delete(ctx, req)
	if err != nil {
		var apiErr *apierror.APIError
		if errors.As(err, &apiErr) && apiErr.HTTPCode() == http.StatusNotFound {
			log.Printf("Failed to find resource to delete: %v", err)
			return nil
		}

		return fmt.Errorf("failed to delete resource: %w", err)
	}

	if err := op.Wait(ctx); err != nil {
		return fmt.Errorf("failed to wait for autoscaler deletion operation: %w", err)
	}

	log.Printf("Autoscaler %s deleted successfully in project %s, zone %s", name, projectID, zone)
	return nil
}
