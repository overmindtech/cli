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

func TestComputeNetworkEndpointGroup(t *testing.T) {
	ctx := context.Background()
	projectID := "test-project"
	zone := "us-central1-a"
	linker := gcpshared.NewLinker()
	negName := "test-neg"

	networkURL := fmt.Sprintf("projects/%s/global/networks/default", projectID)
	subnetworkURL := fmt.Sprintf("projects/%s/regions/us-central1/subnetworks/default", projectID)
	cloudRunService := fmt.Sprintf("projects/%s/locations/us-central1/services/test-cloud-run-service", projectID)
	appEngineService := "test-app-engine-service"
	cloudFunctionName := fmt.Sprintf("projects/%s/locations/us-central1/functions/test-cloud-function", projectID)

	neg := &computepb.NetworkEndpointGroup{
		Name:       &negName,
		Network:    &networkURL,
		Subnetwork: &subnetworkURL,
		CloudRun: &computepb.NetworkEndpointGroupCloudRun{
			Service: &cloudRunService,
		},
		AppEngine: &computepb.NetworkEndpointGroupAppEngine{
			Service: &appEngineService,
		},
		CloudFunction: &computepb.NetworkEndpointGroupCloudFunction{
			Function: &cloudFunctionName,
		},
	}

	negName2 := "test-neg-2"
	neg2 := &computepb.NetworkEndpointGroup{
		Name: &negName2,
	}

	negList := &computepb.NetworkEndpointGroupList{
		Items: []*computepb.NetworkEndpointGroup{neg, neg2},
	}

	sdpItemType := gcpshared.ComputeNetworkEndpointGroup

	expectedCallAndResponses := map[string]shared.MockResponse{
		fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/zones/%s/networkEndpointGroups/%s", projectID, zone, negName): {
			StatusCode: http.StatusOK,
			Body:       neg,
		},
		fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/zones/%s/networkEndpointGroups/%s", projectID, zone, negName2): {
			StatusCode: http.StatusOK,
			Body:       neg2,
		},
		fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/zones/%s/networkEndpointGroups", projectID, zone): {
			StatusCode: http.StatusOK,
			Body:       negList,
		},
	}

	t.Run("Get", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, projectID, zone)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		sdpItem, err := adapter.Get(ctx, fmt.Sprintf("%s.%s", projectID, zone), negName, true)
		if err != nil {
			t.Fatalf("Failed to get resource: %v", err)
		}

		if sdpItem.GetType() != sdpItemType.String() {
			t.Errorf("Expected type %s, got %s", sdpItemType.String(), sdpItem.GetType())
		}
		if sdpItem.UniqueAttributeValue() != negName {
			t.Errorf("Expected unique attribute value '%s', got %s", negName, sdpItem.UniqueAttributeValue())
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				// Network link
				{
					ExpectedType:   gcpshared.ComputeNetwork.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "default",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// Subnetwork link
				{
					ExpectedType:   gcpshared.ComputeSubnetwork.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "default",
					ExpectedScope:  fmt.Sprintf("%s.us-central1", projectID),
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// Cloud Run service link
				{
					ExpectedType:   gcpshared.RunService.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey("us-central1", "test-cloud-run-service"),
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// Note: App Engine service link test omitted because gcp-app-engine-service adapter doesn't exist yet
				// Cloud Function link
				{
					ExpectedType:   gcpshared.CloudFunctionsFunction.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey("us-central1", "test-cloud-function"),
					ExpectedScope:  projectID,
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

		sdpItems, err := listable.List(ctx, fmt.Sprintf("%s.%s", projectID, zone), true)
		if err != nil {
			t.Fatalf("Failed to list resources: %v", err)
		}

		if len(sdpItems) != 2 {
			t.Errorf("Expected 2 resources, got %d", len(sdpItems))
		}
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		errorResponses := map[string]shared.MockResponse{
			fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/zones/%s/networkEndpointGroups/%s", projectID, zone, negName): {
				StatusCode: http.StatusNotFound,
				Body:       map[string]interface{}{"error": "NEG not found"},
			},
		}

		httpCli := shared.NewMockHTTPClientProvider(errorResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, projectID, zone)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		_, err = adapter.Get(ctx, fmt.Sprintf("%s.%s", projectID, zone), negName, true)
		if err == nil {
			t.Error("Expected error when getting non-existent resource, but got nil")
		}
	})
}
