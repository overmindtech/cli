package integrationtests

import (
	"context"
	"os"
	"testing"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sources/gcp/dynamic"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

func TestComputeInstanceTemplateIntegration(t *testing.T) {
	projectID := os.Getenv("GCP_PROJECT_ID")
	if projectID == "" {
		t.Skip("GCP_PROJECT_ID environment variable not set")
	}

	ctx := context.Background()

	t.Run("Setup", func(t *testing.T) {
		t.Logf("We will test existing instance templates in project '%s'", projectID)
	})

	t.Run("Run", func(t *testing.T) {
		t.Logf("Running test for Compute Instance Templates")

		sdpItemType := gcpshared.ComputeInstanceTemplate
		meta := gcpshared.SDPAssetTypeToAdapterMeta[sdpItemType]

		gcpHTTPCliWithOtel, err := gcpshared.GCPHTTPClientWithOtel()
		if err != nil {
			t.Fatalf("Failed to create GCP HTTP client: %v", err)
		}

		// Instance templates are global resources, no region needed
		adapter, err := dynamic.MakeAdapter(sdpItemType, meta, gcpshared.NewLinker(), gcpHTTPCliWithOtel, projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		listable, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			t.Fatalf("Adapter for %s does not implement ListableAdapter", sdpItemType)
		}

		// For global resources, scope is just the project ID
		scope := projectID
		sdpItems, err := listable.List(ctx, scope, true)
		if err != nil {
			t.Fatalf("Failed to list instance templates: %v", err)
		}

		if len(sdpItems) == 0 {
			t.Logf("No instance templates found in project %s", projectID)
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
				t.Errorf("Expected sdpItem to be non-nil for instance template %s", uniqueAttrValue)
				continue
			}

			if err := sdpItem.Validate(); err != nil {
				t.Errorf("SDP item validation failed for %s: %v", uniqueAttrValue, err)
			}
		}
	})

	t.Run("Teardown", func(t *testing.T) {
		t.Logf("No teardown needed for Compute Instance Template test as we only performed read operations")
	})
}
