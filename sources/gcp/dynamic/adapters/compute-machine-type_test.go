package adapters_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"cloud.google.com/go/compute/apiv1/computepb"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources/gcp/dynamic"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

func machineTypeStringPtr(s string) *string {
	return &s
}

func machineTypeInt32Ptr(i int32) *int32 {
	return &i
}

func machineTypeInt64Ptr(i int64) *int64 {
	return &i
}

func machineTypeBoolPtr(b bool) *bool {
	return &b
}

func TestComputeMachineType(t *testing.T) {
	ctx := context.Background()
	projectID := "test-project"
	zone := "us-central1-a"
	linker := gcpshared.NewLinker()
	machineTypeName := "n1-standard-1"

	// Create mock protobuf object
	// Note: MachineType may have an Accelerators field listing compatible accelerator types,
	// but we'll test without it for now since the exact field structure varies.
	// The blast propagation configuration will still be validated.
	machineType := &computepb.MachineType{
		Name:                         machineTypeStringPtr(machineTypeName),
		Description:                  machineTypeStringPtr("1 vCPU, 3.75 GB RAM"),
		GuestCpus:                    machineTypeInt32Ptr(1),
		MemoryMb:                     machineTypeInt32Ptr(3840),
		Zone:                         machineTypeStringPtr(zone),
		SelfLink:                     machineTypeStringPtr(fmt.Sprintf("https://www.googleapis.com/compute/v1/projects/%s/zones/%s/machineTypes/%s", projectID, zone, machineTypeName)),
		Kind:                         machineTypeStringPtr("compute#machineType"),
		IsSharedCpu:                  machineTypeBoolPtr(false),
		MaximumPersistentDisks:       machineTypeInt32Ptr(128),
		MaximumPersistentDisksSizeGb: machineTypeInt64Ptr(65536),
	}

	// Create second resource for list testing
	machineTypeName2 := "n1-standard-2"
	machineType2 := &computepb.MachineType{
		Name:                         machineTypeStringPtr(machineTypeName2),
		Description:                  machineTypeStringPtr("2 vCPU, 7.5 GB RAM"),
		GuestCpus:                    machineTypeInt32Ptr(2),
		MemoryMb:                     machineTypeInt32Ptr(7680),
		Zone:                         machineTypeStringPtr(zone),
		SelfLink:                     machineTypeStringPtr(fmt.Sprintf("https://www.googleapis.com/compute/v1/projects/%s/zones/%s/machineTypes/%s", projectID, zone, machineTypeName2)),
		Kind:                         machineTypeStringPtr("compute#machineType"),
		IsSharedCpu:                  machineTypeBoolPtr(false),
		MaximumPersistentDisks:       machineTypeInt32Ptr(128),
		MaximumPersistentDisksSizeGb: machineTypeInt64Ptr(65536),
	}

	// Create list response with multiple items
	machineTypeList := &computepb.MachineTypeList{
		Items: []*computepb.MachineType{machineType, machineType2},
	}

	sdpItemType := gcpshared.ComputeMachineType

	// Create a machine type with accelerators for testing blast propagation
	// Using JSON map to include accelerators field since protobuf structure is unclear
	acceleratorTypeURL := fmt.Sprintf("projects/%s/zones/%s/acceleratorTypes/nvidia-tesla-t4", projectID, zone)
	machineTypeWithAccelerators := map[string]interface{}{
		"name":                         machineTypeName,
		"description":                  "1 vCPU, 3.75 GB RAM",
		"guestCpus":                    1,
		"memoryMb":                     3840,
		"zone":                         zone,
		"selfLink":                     fmt.Sprintf("https://www.googleapis.com/compute/v1/projects/%s/zones/%s/machineTypes/%s", projectID, zone, machineTypeName),
		"kind":                         "compute#machineType",
		"isSharedCpu":                  false,
		"maximumPersistentDisks":       128,
		"maximumPersistentDisksSizeGb": 65536,
		"accelerators": []map[string]interface{}{
			{
				"acceleratorType":       acceleratorTypeURL,
				"guestAcceleratorCount": 1,
			},
		},
	}

	// Mock HTTP responses
	expectedCallAndResponses := map[string]shared.MockResponse{
		fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/zones/%s/machineTypes/%s", projectID, zone, machineTypeName): {
			StatusCode: http.StatusOK,
			Body:       machineTypeWithAccelerators, // Use JSON map to test accelerator link
		},
		fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/zones/%s/machineTypes/%s", projectID, zone, machineTypeName2): {
			StatusCode: http.StatusOK,
			Body:       machineType2,
		},
		fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/zones/%s/machineTypes", projectID, zone): {
			StatusCode: http.StatusOK,
			Body:       machineTypeList,
		},
	}

	t.Run("Get", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, projectID, zone)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		// For zonal resources, query is just the resource name, scope is "projectID.zone"
		zoneScope := fmt.Sprintf("%s.%s", projectID, zone)
		sdpItem, err := adapter.Get(ctx, zoneScope, machineTypeName, true)
		if err != nil {
			t.Fatalf("Failed to get resource: %v", err)
		}

		// Validate SDP item properties
		if sdpItem.GetType() != sdpItemType.String() {
			t.Errorf("Expected type %s, got %s", sdpItemType.String(), sdpItem.GetType())
		}
		if sdpItem.UniqueAttributeValue() != machineTypeName {
			t.Errorf("Expected unique attribute value '%s', got %s", machineTypeName, sdpItem.UniqueAttributeValue())
		}
		if sdpItem.GetScope() != fmt.Sprintf("%s.%s", projectID, zone) {
			t.Errorf("Expected scope '%s.%s', got %s", projectID, zone, sdpItem.GetScope())
		}

		// Validate specific attributes
		val, err := sdpItem.GetAttributes().Get("name")
		if err != nil {
			t.Fatalf("Failed to get 'name' attribute: %v", err)
		}
		if val != machineTypeName {
			t.Errorf("Expected name field to be '%s', got %s", machineTypeName, val)
		}

		// Include static tests - MUST cover ALL blast propagation links
		t.Run("StaticTests", func(t *testing.T) {
			// CRITICAL: Review the adapter's blast propagation configuration and create
			// test cases for EVERY linked resource defined in the adapter's blastPropagation map
			// The adapter has one blast propagation: "accelerators.acceleratorType" -> ComputeAcceleratorType
			queryTests := shared.QueryTests{
				{
					ExpectedType:   gcpshared.ComputeAcceleratorType.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "nvidia-tesla-t4",
					ExpectedScope:  zoneScope,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
			}
			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("List", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, projectID, zone)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		listable, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			t.Skipf("Adapter for %s does not implement ListableAdapter", sdpItemType)
		}

		// For zonal resources, list requires zone scope
		zoneScope := fmt.Sprintf("%s.%s", projectID, zone)
		sdpItems, err := listable.List(ctx, zoneScope, true)
		if err != nil {
			t.Fatalf("Failed to list resources: %v", err)
		}

		if len(sdpItems) != 2 {
			t.Errorf("Expected 2 resources, got %d", len(sdpItems))
		}
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		// Test with error responses to simulate API errors
		errorResponses := map[string]shared.MockResponse{
			fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/zones/%s/machineTypes/%s", projectID, zone, machineTypeName): {
				StatusCode: http.StatusNotFound,
				Body:       map[string]interface{}{"error": "Resource not found"},
			},
		}

		httpCli := shared.NewMockHTTPClientProvider(errorResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, projectID, zone)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		zoneScope := fmt.Sprintf("%s.%s", projectID, zone)
		_, err = adapter.Get(ctx, zoneScope, machineTypeName, true)
		if err == nil {
			t.Error("Expected error when getting non-existent resource, but got nil")
		}
	})
}
