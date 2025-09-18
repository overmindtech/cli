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

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/gcp/manual"
	"github.com/overmindtech/cli/sources/gcp/shared"
)

func TestComputeAddressIntegration(t *testing.T) {
	projectID := os.Getenv("GCP_PROJECT_ID")
	if projectID == "" {
		t.Skip("GCP_PROJECT_ID environment variable not set")
	}

	region := os.Getenv("GCP_REGION")
	if region == "" {
		t.Skip("GCP_REGION environment variable not set")
	}

	addressName := "overmind-test-address"

	ctx := context.Background()

	client, err := compute.NewAddressesRESTClient(ctx)
	if err != nil {
		t.Fatalf("NewAddressesRESTClient: %v", err)
	}

	defer client.Close()

	t.Run("Setup", func(t *testing.T) {
		err := createComputeAddress(ctx, client, projectID, region, addressName)
		if err != nil {
			t.Fatalf("Failed to create compute address: %v", err)
		}
	})

	t.Run("Run", func(t *testing.T) {
		log.Printf("Running integration test for Compute Address in project %s, region %s", projectID, region)

		addressWrapper := manual.NewComputeAddress(shared.NewComputeAddressClient(client), projectID, region)
		scope := addressWrapper.Scopes()[0]

		addressAdapter := sources.WrapperToAdapter(addressWrapper)
		sdpItem, qErr := addressAdapter.Get(ctx, scope, addressName, true)
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

		if uniqueAttrValue != addressName {
			t.Fatalf("Expected unique attribute value to be %s, got %s", addressName, uniqueAttrValue)
		}

		// Check if adapter supports listing
		listable, ok := addressAdapter.(discovery.ListableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support List operation")
		}

		sdpItems, err := listable.List(ctx, scope, true)
		if err != nil {
			t.Fatalf("Failed to list compute addresses: %v", err)
		}

		if len(sdpItems) < 1 {
			t.Fatalf("Expected at least one compute addresses, got %d", len(sdpItems))
		}

		var found bool
		for _, item := range sdpItems {
			if v, err := item.GetAttributes().Get(uniqueAttrKey); err == nil && v == addressName {
				found = true
				break
			}
		}

		if !found {
			t.Fatalf("Expected to find address %s in the list of compute addresses", addressName)
		}
	})

	t.Run("Teardown", func(t *testing.T) {
		err := deleteComputeAddress(ctx, client, region, projectID, addressName)
		if err != nil {
			t.Fatal(err)
		}
	})
}

// createComputeAddress creates a GCP Compute Engine address with the given parameters.
func createComputeAddress(ctx context.Context, client *compute.AddressesClient, projectID, region, addressName string) error {
	// Define the address configuration
	address := &computepb.Address{
		Name: ptr.To(addressName),
		Labels: map[string]string{
			"test": "integration",
		},
		NetworkTier: ptr.To("PREMIUM"),
		Region:      ptr.To(region),
	}

	// Create the address
	req := &computepb.InsertAddressRequest{
		Project:         projectID,
		Region:          region,
		AddressResource: address,
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
		return fmt.Errorf("failed to wait for address creation operation: %w", err)
	}

	log.Printf("Address %s created successfully in project %s, region %s", addressName, projectID, region)
	return nil
}

// Delete a compute address template.
func deleteComputeAddress(ctx context.Context, client *compute.AddressesClient, region, projectID, addressName string) error {
	req := &computepb.DeleteAddressRequest{
		Project: projectID,
		Region:  region,
		Address: addressName,
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
		return fmt.Errorf("failed to wait for address deletion operation: %w", err)
	}

	log.Printf("Compute address %s deleted successfully", addressName)
	return nil
}
