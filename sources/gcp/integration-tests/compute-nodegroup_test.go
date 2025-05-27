package integrationtests

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	compute "cloud.google.com/go/compute/apiv1"
	"cloud.google.com/go/compute/apiv1/computepb"
	"github.com/googleapis/gax-go/v2/apierror"
	log "github.com/sirupsen/logrus"
	"k8s.io/utils/ptr"

	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/gcp/adapters"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

// The scope of this integration test should cover nodegroups, nodes, and node templates.

func TestComputeNodeGroupIntegration(t *testing.T) {
	projectID := os.Getenv("GCP_PROJECT_ID")
	if projectID == "" {
		t.Skip("GCP_PROJECT_ID environment variable not set")
	}

	zone := os.Getenv("GCP_ZONE")
	if zone == "" {
		t.Skip("GCP_ZONE environment variable not set")
	}

	region := zone[:strings.LastIndex(zone, "-")]

	// Can replace with an environment-specific ID later.
	suffix := "default"

	// Nodegroup -> Node Template
	nodeTemplateName := "overmind-integration-test-node-template-" + suffix
	nodeGroupName := "overmind-integration-test-node-group-" + suffix
	ctx := context.Background()

	// Create a new Compute Engine client
	client, err := compute.NewNodeGroupsRESTClient(ctx)
	if err != nil {
		t.Fatalf("NewNodeGroupsRESTClient: %v", err)
	}
	defer client.Close()

	ntClient, err := compute.NewNodeTemplatesRESTClient(ctx)
	if err != nil {
		t.Fatalf("NewNodeTemplatesRESTClient: %v", err)
	}
	defer ntClient.Close()

	t.Run("Setup", func(t *testing.T) {
		err := createComputeNodeTemplate(ctx, ntClient, projectID, region, nodeTemplateName)
		if err != nil {
			t.Fatalf("Failed to create compute node template: %v", err)
		}

		fullNodeTemplateName := "projects/" + projectID + "/regions/" + region + "/nodeTemplates/" + nodeTemplateName

		err = createComputeNodeGroup(ctx, client, fullNodeTemplateName, projectID, zone, nodeGroupName)
		if err != nil {
			t.Fatalf("Failed to create compute node group: %v", err)
		}
	})

	t.Run("Run", func(t *testing.T) {
		log.Printf("Running integration test for Compute Node Group in project %s, zone %s", projectID, zone)

		nodeGroupWrapper := adapters.NewComputeNodeGroup(gcpshared.NewComputeNodeGroupClient(client), projectID, zone)
		scope := nodeGroupWrapper.Scopes()[0]

		nodeGroupAdapter := sources.WrapperToAdapter(nodeGroupWrapper)

		// [SPEC] GET against a valid resource name will return an SDP item wrapping the
		// available resource.
		sdpItem, err := nodeGroupAdapter.Get(ctx, scope, nodeGroupName, true)
		if err != nil {
			t.Fatalf("nodeGroupAdapter.Get returned unexpected error: %v", err)
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

		if uniqueAttrValue != nodeGroupName {
			t.Fatalf("Expected unique attribute value to be %s, got %s", nodeGroupName, uniqueAttrValue)
		}

		// [SPEC] The only linked item query is one Node Template.
		{
			if len(sdpItem.GetLinkedItemQueries()) != 1 {
				t.Fatalf("Expected 1 linked item query, got: %d", len(sdpItem.GetLinkedItemQueries()))
			}

			linkedItem := sdpItem.GetLinkedItemQueries()[0]
			if linkedItem.GetQuery().GetType() != adapters.ComputeNodeTemplate.String() {
				t.Fatalf("Expected linked item type to be %s, got: %s", adapters.ComputeNodeTemplate.String(), linkedItem.GetQuery().GetType())
			}

			if linkedItem.GetQuery().GetQuery() != nodeTemplateName {
				t.Fatalf("Expected linked item query to be %s, got: %s", nodeTemplateName, linkedItem.GetQuery().GetQuery())
			}

			expectedScope := gcpshared.RegionalScope(projectID, region)
			if linkedItem.GetQuery().GetScope() != expectedScope {
				t.Fatalf("Expected linked item scope to be %s, got: %s", expectedScope, linkedItem.GetQuery().GetScope())
			}
		}

		// [SPEC] The LIST operation for node groups will list all node groups in a given
		// scope.
		sdpItems, err := nodeGroupAdapter.List(ctx, scope, true)
		if err != nil {
			t.Fatalf("Failed to list compute node groups: %v", err)
		}

		if len(sdpItems) < 1 {
			t.Fatalf("Expected at least one compute node group, got %d", len(sdpItems))
		}

		// The LIST operation result should include our node group.
		found := false
		for _, item := range sdpItems {
			if v, err := item.GetAttributes().Get(uniqueAttrKey); err == nil && v == nodeGroupName {
				found = true
				break
			}
		}

		if !found {
			t.Fatalf("Expected to find node group %s in list, but it was not found", nodeGroupName)
		}
	})

	t.Run("Teardown", func(t *testing.T) {
		err := deleteComputeNodeGroup(ctx, client, projectID, zone, nodeGroupName)
		if err != nil {
			t.Errorf("Warning: failed to delete compute node group: %v", err)
		}

		err = deleteComputeNodeTemplate(ctx, ntClient, projectID, region, nodeTemplateName)
		if err != nil {
			t.Errorf("Warning: failed to delete node template: %v", err)
		}
	})
}

// Create a compute node template in GCP to test against.
func createComputeNodeTemplate(ctx context.Context, client *compute.NodeTemplatesClient, projectID, region, name string) error {
	// Create a new node template
	nodeTemplate := &computepb.NodeTemplate{
		Name:     ptr.To(name),
		NodeType: ptr.To("c2-node-60-240"),
	}

	// Create the node template
	req := &computepb.InsertNodeTemplateRequest{
		Project:              projectID,
		NodeTemplateResource: nodeTemplate,
		Region:               region,
	}

	op, err := client.Insert(ctx, req)
	if err != nil {
		return fmt.Errorf("Failed to create node template: %w", err)
	}

	// Wait for the operation to complete
	if err := op.Wait(ctx); err != nil {
		return fmt.Errorf("Failed to wait for node template operation: %w", err)
	}

	log.Printf("Node template %s created successfully in project %s", name, projectID)
	return nil
}

// Delete a compute node template.
func deleteComputeNodeTemplate(ctx context.Context, client *compute.NodeTemplatesClient, projectID, region, name string) error {
	req := &computepb.DeleteNodeTemplateRequest{
		Project:      projectID,
		Region:       region,
		NodeTemplate: name,
	}

	op, err := client.Delete(ctx, req)
	var apiErr *apierror.APIError
	if errors.As(err, &apiErr) && apiErr.HTTPCode() == 404 {
		log.Printf("Node template %s not found in project %s", name, projectID)
		return nil
	}

	if err != nil {
		return fmt.Errorf("failed to delete node template: %w", err)
	}

	if err := op.Wait(ctx); err != nil {
		return fmt.Errorf("failed to wait for node template deletion operation: %w", err)
	}

	log.Printf("Node template %s deleted successfully in project %s", name, projectID)
	return nil
}

// Create a compute node group in GCP using the given node template.
func createComputeNodeGroup(ctx context.Context, client *compute.NodeGroupsClient, nodeTemplate, projectID, zone, name string) error {
	// Create a new node group
	nodeGroup := &computepb.NodeGroup{
		Name:         ptr.To(name),
		NodeTemplate: ptr.To(nodeTemplate),
		AutoscalingPolicy: &computepb.NodeGroupAutoscalingPolicy{
			Mode:     ptr.To(computepb.NodeGroupAutoscalingPolicy_OFF.String()),
			MinNodes: ptr.To(int32(0)),
			MaxNodes: ptr.To(int32(1)),
		},
	}

	req := &computepb.InsertNodeGroupRequest{
		Project:           projectID,
		Zone:              zone,
		NodeGroupResource: nodeGroup,
	}

	op, err := client.Insert(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to create node group: %w", err)
	}

	if err := op.Wait(ctx); err != nil {
		return fmt.Errorf("failed to wait for node group creation operation: %w", err)
	}

	log.Printf("Node group %s created successfully in project %s, zone %s", name, projectID, zone)
	return nil
}

// Delete a compute node group in GCP.
func deleteComputeNodeGroup(ctx context.Context, client *compute.NodeGroupsClient, projectID, zone, name string) error {
	req := &computepb.DeleteNodeGroupRequest{
		Project:   projectID,
		Zone:      zone,
		NodeGroup: name,
	}

	op, err := client.Delete(ctx, req)
	var apiErr *apierror.APIError
	if errors.As(err, &apiErr) && apiErr.HTTPCode() == 404 {
		log.Printf("Node group %s not found in project %s, zone %s", name, projectID, zone)
		return nil
	}

	if err != nil {
		return fmt.Errorf("failed to delete node group: %w", err)
	}

	if err := op.Wait(ctx); err != nil {
		return fmt.Errorf("failed to wait for node group deletion operation: %w", err)
	}

	log.Printf("Node group %s deleted successfully in project %s, zone %s", name, projectID, zone)
	return nil
}
