package integrationtests

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sources/gcp/dynamic"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

func TestComputeSubnetworkIntegration(t *testing.T) {
	projectID := os.Getenv("GCP_PROJECT_ID")
	if projectID == "" {
		t.Skip("GCP_PROJECT_ID environment variable not set")
	}

	region := os.Getenv("GCP_REGION")
	if region == "" {
		region = "us-central1" // Default region if not specified
		t.Logf("GCP_REGION environment variable not set, using default: %s", region)
	}

	ctx := context.Background()

	// We'll use the default subnetwork for testing
	subnetworkName := "default" // Default subnetworks are created for default networks

	t.Run("Setup", func(t *testing.T) {
		t.Logf("We will use the default subnetwork '%s' in region '%s' of project '%s' for testing",
			subnetworkName, region, projectID)
	})

	t.Run("Run", func(t *testing.T) {
		t.Logf("Running test for Compute Subnetwork: %s", subnetworkName)

		sdpItemType := gcpshared.ComputeSubnetwork

		gcpHTTPCliWithOtel, err := gcpshared.GCPHTTPClientWithOtel()
		if err != nil {
			t.Fatalf("Failed to create GCP HTTP client: %v", err)
		}

		// For subnetworks, we need to include the region as an initialization parameter
		adapter, err := dynamic.MakeAdapter(sdpItemType, gcpshared.NewLinker(), gcpHTTPCliWithOtel, projectID, region)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		listable, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			t.Fatalf("Adapter for %s does not implement ListableAdapter", sdpItemType)
		}

		scope := fmt.Sprintf("%s.%s", projectID, region)
		sdpItems, err := listable.List(ctx, scope, true)
		if err != nil {
			t.Fatalf("Failed to list subnetworks in region %s: %v", region, err)
		}

		if len(sdpItems) == 0 {
			t.Logf("No subnetworks found in project %s and region %s", projectID, region)
			return
		}

		for _, sdp := range sdpItems {
			uniqueAttrVal, err := sdp.GetAttributes().Get(sdp.GetUniqueAttribute())
			if err != nil {
				t.Errorf("Failed to get unique attribute for %s: %v", sdp.GetUniqueAttribute(), err)
				continue
			}

			uniqueAttrValue, ok := uniqueAttrVal.(string)
			if !ok {
				t.Errorf("Unique attribute value for %s is not a string: %v", sdp.GetUniqueAttribute(), uniqueAttrVal)
				continue
			}

			sdpItem, qErr := adapter.Get(ctx, scope, uniqueAttrValue, true)
			if qErr != nil {
				t.Errorf("Expected no error, got: %v", qErr)
				continue
			}

			if sdpItem == nil {
				t.Errorf("Expected sdpItem to be non-nil for subnetwork %s", uniqueAttrValue)
				continue
			}

			if err := sdpItem.Validate(); err != nil {
				t.Errorf("SDP item validation failed for %s: %v", uniqueAttrValue, err)
			}
		}
	})

	t.Run("Teardown", func(t *testing.T) {
		t.Logf("Skipping teardown for Compute Subnetwork test as we are using the default subnetwork '%s'", subnetworkName)
	})
}
