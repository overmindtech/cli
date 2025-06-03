package integrationtests

import (
	"context"
	"errors"
	"fmt"
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

func TestComputeHealthCheckIntegration(t *testing.T) {
	projectID := os.Getenv("GCP_PROJECT_ID")
	if projectID == "" {
		t.Skip("GCP_PROJECT_ID environment variable not set")
	}

	healthCheckName := "integration-test-healthcheck"

	ctx := context.Background()

	// Create a new Compute HealthCheck client
	client, err := compute.NewHealthChecksRESTClient(ctx)
	if err != nil {
		t.Fatalf("NewHealthChecksRESTClient: %v", err)
	}
	defer client.Close()

	t.Run("Setup", func(t *testing.T) {
		err := createComputeHealthCheck(ctx, client, projectID, healthCheckName)
		if err != nil {
			t.Fatalf("Failed to create compute health check: %v", err)
		}
	})

	t.Run("Run", func(t *testing.T) {
		log.Printf("Running integration test for Compute HealthCheck in project %s", projectID)

		healthCheckWrapper := manual.NewComputeHealthCheck(gcpshared.NewComputeHealthCheckClient(client), projectID)
		scope := healthCheckWrapper.Scopes()[0]

		healthCheckAdapter := sources.WrapperToAdapter(healthCheckWrapper)

		// [SPEC] GET against a valid resource name will return an SDP item wrapping the
		// available resource.
		sdpItem, err := healthCheckAdapter.Get(ctx, scope, healthCheckName, true)
		if err != nil {
			t.Fatalf("healthCheckAdapter.Get returned unexpected error: %v", err)
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

		if uniqueAttrValue != healthCheckName {
			t.Fatalf("Expected unique attribute value to be %s, got %s", healthCheckName, uniqueAttrValue)
		}

		// [SPEC] HealthChecks have no linked items.
		if len(sdpItem.GetLinkedItemQueries()) != 0 {
			t.Fatalf("Expected 0 linked item queries, got: %d", len(sdpItem.GetLinkedItemQueries()))
		}

		// [SPEC] The LIST operation for health checks will list all health checks in a given
		// scope.
		sdpItems, err := healthCheckAdapter.List(ctx, scope, true)
		if err != nil {
			t.Fatalf("Failed to list compute health checks: %v", err)
		}

		if len(sdpItems) < 1 {
			t.Fatalf("Expected at least one compute health check, got %d", len(sdpItems))
		}

		// The LIST operation result should include our health check.
		found := false
		for _, item := range sdpItems {
			if v, err := item.GetAttributes().Get(uniqueAttrKey); err == nil && v == healthCheckName {
				found = true
				break
			}
		}

		if !found {
			t.Fatalf("Expected to find health check %s in list, but it was not found", healthCheckName)
		}
	})

	t.Run("Teardown", func(t *testing.T) {
		err := deleteComputeHealthCheck(ctx, client, projectID, healthCheckName)
		if err != nil {
			t.Errorf("Warning: failed to delete compute health check: %v", err)
		}
	})
}

// createComputeHealthCheck creates a GCP Compute HealthCheck with the given parameters.
func createComputeHealthCheck(ctx context.Context, client *compute.HealthChecksClient, projectID, healthCheckName string) error {
	healthCheck := &computepb.HealthCheck{
		Name:             ptr.To(healthCheckName),
		CheckIntervalSec: ptr.To(int32(5)),
		TimeoutSec:       ptr.To(int32(5)),
		Type:             ptr.To("TCP"),
		TcpHealthCheck: &computepb.TCPHealthCheck{
			Port: ptr.To(int32(80)),
		},
	}

	req := &computepb.InsertHealthCheckRequest{
		Project:             projectID,
		HealthCheckResource: healthCheck,
	}

	op, err := client.Insert(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to create health check: %w", err)
	}

	if err := op.Wait(ctx); err != nil {
		return fmt.Errorf("failed to wait for health check creation operation: %w", err)
	}

	log.Printf("Health check %s created successfully in project %s", healthCheckName, projectID)
	return nil
}

// deleteComputeHealthCheck deletes a GCP Compute HealthCheck.
func deleteComputeHealthCheck(ctx context.Context, client *compute.HealthChecksClient, projectID, healthCheckName string) error {
	req := &computepb.DeleteHealthCheckRequest{
		Project:     projectID,
		HealthCheck: healthCheckName,
	}

	op, err := client.Delete(ctx, req)
	var apiErr *apierror.APIError
	if errors.As(err, &apiErr) && apiErr.HTTPCode() == 404 {
		log.Printf("Health check %s not found in project %s", healthCheckName, projectID)
		return nil
	}

	if err != nil {
		return fmt.Errorf("failed to delete health check: %w", err)
	}

	if err := op.Wait(ctx); err != nil {
		return fmt.Errorf("failed to wait for health check deletion operation: %w", err)
	}

	log.Printf("Health check %s deleted successfully in project %s", healthCheckName, projectID)
	return nil
}
