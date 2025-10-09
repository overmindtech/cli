package adapters_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"cloud.google.com/go/aiplatform/apiv1/aiplatformpb"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources/gcp/dynamic"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

func TestAIPlatformEndpoint(t *testing.T) {
	ctx := context.Background()
	projectID := "test-project"
	linker := gcpshared.NewLinker()
	endpointName := "test-endpoint"

	// Create mock protobuf object
	endpoint := &aiplatformpb.Endpoint{
		Name:        fmt.Sprintf("projects/%s/locations/global/endpoints/%s", projectID, endpointName),
		DisplayName: "Test Endpoint",
		Description: "Test AI Platform Endpoint",
		Network:     "projects/test-project/global/networks/default",
		EncryptionSpec: &aiplatformpb.EncryptionSpec{
			KmsKeyName: "projects/test-project/locations/global/keyRings/test-ring/cryptoKeys/test-key",
		},
		DeployedModels: []*aiplatformpb.DeployedModel{
			{
				Model: "projects/test-project/locations/global/models/test-model",
			},
		},
		ModelDeploymentMonitoringJob: "projects/test-project/locations/global/modelDeploymentMonitoringJobs/test-job",
		DedicatedEndpointDns:         "test-endpoint.aiplatform.googleapis.com",
		PredictRequestResponseLoggingConfig: &aiplatformpb.PredictRequestResponseLoggingConfig{
			BigqueryDestination: &aiplatformpb.BigQueryDestination{
				OutputUri: "bq://test-project.test_dataset.test_table",
			},
		},
	}

	// Create second endpoint for list testing
	endpointName2 := "test-endpoint-2"
	endpoint2 := &aiplatformpb.Endpoint{
		Name:        fmt.Sprintf("projects/%s/locations/global/endpoints/%s", projectID, endpointName2),
		DisplayName: "Test Endpoint 2",
		Description: "Test AI Platform Endpoint 2",
	}

	// Create list response with multiple items
	endpointList := &aiplatformpb.ListEndpointsResponse{
		Endpoints: []*aiplatformpb.Endpoint{endpoint, endpoint2},
	}

	sdpItemType := gcpshared.AIPlatformEndpoint

	// Mock HTTP responses
	expectedCallAndResponses := map[string]shared.MockResponse{
		fmt.Sprintf("https://aiplatform.googleapis.com/v1/projects/%s/locations/global/endpoints/%s", projectID, endpointName): {
			StatusCode: http.StatusOK,
			Body:       endpoint,
		},
		fmt.Sprintf("https://aiplatform.googleapis.com/v1/projects/%s/locations/global/endpoints/%s", projectID, endpointName2): {
			StatusCode: http.StatusOK,
			Body:       endpoint2,
		},
		fmt.Sprintf("https://aiplatform.googleapis.com/v1/projects/%s/locations/global/endpoints", projectID): {
			StatusCode: http.StatusOK,
			Body:       endpointList,
		},
	}

	t.Run("Get", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		sdpItem, err := adapter.Get(ctx, projectID, endpointName, true)
		if err != nil {
			t.Fatalf("Failed to get resource: %v", err)
		}

		// Validate SDP item properties
		if sdpItem.GetType() != sdpItemType.String() {
			t.Errorf("Expected type %s, got %s", sdpItemType.String(), sdpItem.GetType())
		}
		if sdpItem.UniqueAttributeValue() != endpointName {
			t.Errorf("Expected unique attribute value '%s', got %s", endpointName, sdpItem.UniqueAttributeValue())
		}
		if sdpItem.GetScope() != projectID {
			t.Errorf("Expected scope '%s', got %s", projectID, sdpItem.GetScope())
		}

		// Validate specific attributes
		val, err := sdpItem.GetAttributes().Get("name")
		if err != nil {
			t.Fatalf("Failed to get 'name' attribute: %v", err)
		}
		expectedName := fmt.Sprintf("projects/%s/locations/global/endpoints/%s", projectID, endpointName)
		if val != expectedName {
			t.Errorf("Expected name field to be '%s', got %s", expectedName, val)
		}

		// Include static tests - covers ALL blast propagation links
		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				// KMS key link
				{
					ExpectedType:   gcpshared.CloudKMSCryptoKey.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey("global", "test-ring", "test-key"),
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
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
				// Deployed model link
				{
					ExpectedType:   gcpshared.AIPlatformModel.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-model",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				// Model deployment monitoring job link
				{
					ExpectedType:   gcpshared.AIPlatformModelDeploymentMonitoringJob.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey("global", "test-job"),
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				// Dedicated endpoint DNS link
				{
					ExpectedType:   stdlib.NetworkDNS.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  "test-endpoint.aiplatform.googleapis.com",
					ExpectedScope:  "global",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				// BigQuery table link
				{
					ExpectedType:   gcpshared.BigQueryTable.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey("test_dataset", "test_table"),
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
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		listable, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			t.Skipf("Adapter for %s does not implement ListableAdapter", sdpItemType)
		}

		sdpItems, err := listable.List(ctx, projectID, true)
		if err != nil {
			t.Fatalf("Failed to list resources: %v", err)
		}

		if len(sdpItems) != 2 {
			t.Errorf("Expected 2 resources, got %d", len(sdpItems))
		}

		// Validate first item
		if len(sdpItems) > 0 {
			firstItem := sdpItems[0]
			if firstItem.GetType() != sdpItemType.String() {
				t.Errorf("Expected first item type %s, got %s", sdpItemType.String(), firstItem.GetType())
			}
			if firstItem.GetScope() != projectID {
				t.Errorf("Expected first item scope '%s', got %s", projectID, firstItem.GetScope())
			}
		}
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		// Test with error responses to simulate API errors
		errorResponses := map[string]shared.MockResponse{
			fmt.Sprintf("https://aiplatform.googleapis.com/v1/projects/%s/locations/global/endpoints/%s", projectID, endpointName): {
				StatusCode: http.StatusNotFound,
				Body:       map[string]interface{}{"error": "Endpoint not found"},
			},
		}

		httpCli := shared.NewMockHTTPClientProvider(errorResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		_, err = adapter.Get(ctx, projectID, endpointName, true)
		if err == nil {
			t.Error("Expected error when getting non-existent resource, but got nil")
		}
	})
}
