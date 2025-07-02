package integrationtests

import (
	"context"
	"os"
	"testing"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sources/gcp/dynamic"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

func TestComputeNetworkIntegration(t *testing.T) {
	projectID := os.Getenv("GCP_PROJECT_ID")
	if projectID == "" {
		t.Skip("GCP_PROJECT_ID environment variable not set")
	}

	ctx := context.Background()

	networkName := "default" // Use an existing network for testing

	t.Run("Setup", func(t *testing.T) {
		t.Logf("We will use the default network '%s' in project '%s' for testing", networkName, projectID)
	})

	t.Run("Run", func(t *testing.T) {
		t.Logf("Running test for Compute Network: %s", networkName)

		sdpItemType := gcpshared.ComputeNetwork
		meta := gcpshared.SDPAssetTypeToAdapterMeta[sdpItemType]

		gcpHTTPCliWithOtel, err := gcpshared.GCPHTTPClientWithOtel()
		if err != nil {
			t.Fatalf("Failed to create GCP HTTP client: %v", err)
		}

		adapter, err := dynamic.MakeAdapter(sdpItemType, meta, gcpshared.NewLinker(), gcpHTTPCliWithOtel, projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		listable, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			t.Fatalf("Adapter for %s does not implement ListableAdapter", sdpItemType)
		}

		sdpItems, err := listable.List(ctx, projectID, true)
		if err != nil {
			t.Fatalf("Failed to list networks: %v", err)
		}

		for _, sdp := range sdpItems {
			uniqueAttrVal, err := sdp.GetAttributes().Get(sdp.GetUniqueAttribute())
			if err != nil {
				t.Errorf("Failed to get unique attribute for %s: %v", sdp.GetUniqueAttribute(), err)
			}

			uniqueAttrValue, ok := uniqueAttrVal.(string)
			if !ok {
				t.Errorf("Unique attribute value for %s is not a string: %v", sdp.GetUniqueAttribute(), uniqueAttrVal)
				continue
			}

			sdpItem, qErr := adapter.Get(ctx, projectID, uniqueAttrValue, true)
			if qErr != nil {
				t.Errorf("Expected no error, got: %v", qErr)
			}

			if sdpItem == nil {
				t.Errorf("Expected sdpItem to be non-nil for network %s", sdp.GetUniqueAttribute())
			}

			if err := sdpItem.Validate(); err != nil {
				t.Errorf("SDP item validation failed for %s: %v", sdp.GetUniqueAttribute(), err)
			}
		}
	})

	t.Run("Teardown", func(t *testing.T) {
		t.Logf("Skipping teardown for Compute Network test as we are using the default network '%s'", networkName)
	})
}
