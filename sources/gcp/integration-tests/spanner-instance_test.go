package integrationtests

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"

	instance "cloud.google.com/go/spanner/admin/instance/apiv1"
	"cloud.google.com/go/spanner/admin/instance/apiv1/instancepb"
	"github.com/googleapis/gax-go/v2/apierror"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sources/gcp/dynamic"
	"github.com/overmindtech/cli/sources/gcp/shared"
)

func TestSpannerInstance(t *testing.T) {
	projectID := os.Getenv("GCP_PROJECT_ID")
	if projectID == "" {
		t.Skip("GCP_PROJECT_ID environment variable not set")
	}

	instanceName := "integration-test-instance"

	ctx := context.Background()

	// Create a new Admin Database client
	client, err := instance.NewInstanceAdminClient(ctx)
	if err != nil {
		t.Fatalf("Failed to create Spanner client: %v", err)
	}

	defer client.Close()

	t.Run("Setup", func(t *testing.T) {
		err := setupSpannerInstance(ctx, client, projectID, instanceName)
		if err != nil {
			t.Fatalf("Failed to setup Spanner Instance: %v", err)
		}
	})
	t.Run("Run", func(t *testing.T) {
		linker := shared.NewLinker()

		gcpHTTPCliWithOtel, err := shared.GCPHTTPClientWithOtel()
		if err != nil {
			t.Fatalf("Failed to create gcp http client with otel")
		}
		adapter, err := dynamic.MakeAdapter(shared.SpannerInstance, linker, gcpHTTPCliWithOtel, projectID)
		if err != nil {
			t.Fatalf("Failed to make adapter for spanner instance: %v", err)
		}
		sdpItem, err := adapter.Get(ctx, projectID, instanceName, true)
		if err != nil {
			t.Fatalf("Failed to get item")
		}

		uniqueAttrKey := sdpItem.GetUniqueAttribute()

		uniqueAttrValue, err := sdpItem.GetAttributes().Get(uniqueAttrKey)
		if err != nil {
			t.Fatalf("Failed to get unique attribute: %v", err)
		}

		if uniqueAttrValue != instanceName {
			t.Fatalf("Expected unique attribute value to be %s, got %s", instanceName, uniqueAttrValue)
		}

		// Check if adapter supports listing
		listable, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support List operation")
		}

		sdpItems, err := listable.List(ctx, projectID, true)
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
		err := deleteSpannerInstance(ctx, client, projectID, instanceName)
		if err != nil {
			t.Fatalf("Failed to delete Spanner Instance: %v", err)
		}
	})
}

func deleteSpannerInstance(ctx context.Context, client *instance.InstanceAdminClient, projectID, instanceName string) error {
	return client.DeleteInstance(ctx, &instancepb.DeleteInstanceRequest{
		Name: fmt.Sprintf("projects/%s/instances/%s", projectID, instanceName),
	})
}

func setupSpannerInstance(ctx context.Context, client *instance.InstanceAdminClient, projectID, instanceName string) error {
	// Implement the setup logic for Spanner Instance setup here
	op, err := client.CreateInstance(ctx, &instancepb.CreateInstanceRequest{
		Parent:     "projects/" + projectID,
		InstanceId: instanceName,
		Instance: &instancepb.Instance{
			Name:        fmt.Sprintf("projects/%s/instances/%s", projectID, instanceName),
			Config:      fmt.Sprintf("projects/%s/instanceConfigs/eur3", projectID),
			DisplayName: instanceName,
			NodeCount:   1,
		},
	})
	if err != nil {
		var apiErr *apierror.APIError
		if errors.As(err, &apiErr) && apiErr.GRPCStatus().Proto().GetCode() == int32(codes.AlreadyExists) {
			log.Printf("Resource already exists in project, skipping creation: %v", err)
			return nil
		}

		return fmt.Errorf("failed to create resource: %w", err)
	}

	if _, err := op.Wait(ctx); err != nil {
		return fmt.Errorf("failed to wait for image creation operation: %w", err)
	}

	log.Printf("Spanner instance %s created successfully in project %s", instanceName, projectID)
	return nil
}
