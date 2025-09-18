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
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

func TestComputeImageIntegration(t *testing.T) {
	projectID := os.Getenv("GCP_PROJECT_ID")
	if projectID == "" {
		t.Skip("GCP_PROJECT_ID environment variable not set")
	}

	zone := os.Getenv("GCP_ZONE")
	if zone == "" {
		t.Skip("GCP_ZONE environment variable not set")
	}

	imageName := "integration-test-image"
	diskName := "integration-test-disk"

	ctx := context.Background()

	// Create a new Compute Images client
	client, err := compute.NewImagesRESTClient(ctx)
	if err != nil {
		t.Fatalf("NewImagesRESTClient: %v", err)
	}
	defer client.Close()

	diskClient, err := compute.NewDisksRESTClient(ctx)
	if err != nil {
		t.Fatalf("NewDisksRESTClient: %v", err)
	}
	defer diskClient.Close()

	t.Run("Setup", func(t *testing.T) {
		err = createDisk(ctx, diskClient, projectID, zone, diskName)
		if err != nil {
			t.Fatalf("Failed to create source disk: %v", err)
		}

		err := createComputeImage(ctx, client, projectID, zone, imageName, diskName)
		if err != nil {
			t.Fatalf("Failed to create compute image: %v", err)
		}
	})

	t.Run("ListImages", func(t *testing.T) {
		log.Printf("Listing images in project %s", projectID)

		imagesWrapper := manual.NewComputeImage(gcpshared.NewComputeImagesClient(client), projectID)
		scope := imagesWrapper.Scopes()[0]

		imagesAdapter := sources.WrapperToAdapter(imagesWrapper)

		// Check if adapter supports listing
		listable, ok := imagesAdapter.(discovery.ListableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support List operation")
		}

		sdpItems, err := listable.List(ctx, scope, true)
		if err != nil {
			t.Fatalf("Failed to list compute images: %v", err)
		}

		if len(sdpItems) < 1 {
			t.Fatalf("Expected at least one compute image, got %d", len(sdpItems))
		}

		var found bool
		for _, item := range sdpItems {
			uniqueAttrKey := item.GetUniqueAttribute()
			if v, err := item.GetAttributes().Get(uniqueAttrKey); err == nil && v == imageName {
				found = true
				break
			}
		}

		if !found {
			t.Fatalf("Expected to find images %s in the list of compute images", imageName)
		}

		log.Printf("Found %d images in project %s", len(sdpItems), projectID)
	})

	t.Run("GetImage", func(t *testing.T) {
		log.Printf("Retrieving image %s in project %s", imageName, projectID)

		imagesWrapper := manual.NewComputeImage(gcpshared.NewComputeImagesClient(client), projectID)
		scope := imagesWrapper.Scopes()[0]

		imagesAdapter := sources.WrapperToAdapter(imagesWrapper)
		sdpItem, qErr := imagesAdapter.Get(ctx, scope, imageName, true)
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

		if uniqueAttrValue != imageName {
			t.Fatalf("Expected unique attribute value to be %s, got %s", imageName, uniqueAttrValue)
		}

		log.Printf("Successfully retrieved image %s in project %s", imageName, projectID)
	})

	t.Run("Teardown", func(t *testing.T) {
		err := deleteImage(ctx, client, projectID, zone, imageName)
		if err != nil {
			t.Fatalf("Failed to delete compute image: %v", err)
		}
	})

}

// createComputeImage creates a GCP Compute Image with the given parameters.
func createComputeImage(ctx context.Context, client *compute.ImagesClient, projectID, zone, imageName, diskName string) error {
	image := &computepb.Image{
		Name: ptr.To(imageName),
		SourceDisk: ptr.To(fmt.Sprintf(
			"projects/%s/zones/%s/disks/%s",
			projectID, zone, diskName,
		)),
		Labels: map[string]string{
			"test": "integration",
		},
	}

	// Create the image
	req := &computepb.InsertImageRequest{
		Project:       projectID,
		ImageResource: image,
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
		return fmt.Errorf("failed to wait for image creation operation: %w", err)
	}

	log.Printf("Image %s created successfully in project %s", imageName, projectID)
	return nil
}

func deleteImage(ctx context.Context, client *compute.ImagesClient, projectID, zone, imageName string) error {
	req := &computepb.DeleteImageRequest{
		Project: projectID,
		Image:   imageName,
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
		return fmt.Errorf("failed to wait for image deletion operation: %w", err)
	}

	log.Printf("Compute image %s deleted successfully in project %s", imageName, projectID)
	return nil
}
