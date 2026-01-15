package adapters_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"cloud.google.com/go/aiplatform/apiv1/aiplatformpb"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
	"github.com/overmindtech/cli/sources/gcp/dynamic"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

func TestAIPlatformModel(t *testing.T) {
	ctx := context.Background()
	projectID := "test-project"
	linker := gcpshared.NewLinker()
	modelName := "test-model"

	// Create mock protobuf object
	model := &aiplatformpb.Model{
		Name:        fmt.Sprintf("projects/%s/locations/global/models/%s", projectID, modelName),
		DisplayName: "Test Model",
		Description: "Test AI Platform Model",
		EncryptionSpec: &aiplatformpb.EncryptionSpec{
			KmsKeyName: "projects/test-project/locations/global/keyRings/test-ring/cryptoKeys/test-key",
		},
		ContainerSpec: &aiplatformpb.ModelContainerSpec{
			ImageUri: "us-central1-docker.pkg.dev/test-project/test-repo/test-image:latest",
		},
		PipelineJob: "projects/test-project/locations/global/pipelineJobs/test-pipeline",
		ArtifactUri: fmt.Sprintf("gs://%s-model-artifacts/model/", projectID),
		DeployedModels: []*aiplatformpb.DeployedModelRef{
			{
				Endpoint: "projects/test-project/locations/global/endpoints/test-endpoint",
			},
		},
	}

	// Create second model for list testing
	modelName2 := "test-model-2"
	model2 := &aiplatformpb.Model{
		Name:        fmt.Sprintf("projects/%s/locations/global/models/%s", projectID, modelName2),
		DisplayName: "Test Model 2",
		Description: "Test AI Platform Model 2",
	}

	// Create list response with multiple items
	modelList := &aiplatformpb.ListModelsResponse{
		Models: []*aiplatformpb.Model{model, model2},
	}

	sdpItemType := gcpshared.AIPlatformModel

	// Mock HTTP responses
	expectedCallAndResponses := map[string]shared.MockResponse{
		fmt.Sprintf("https://aiplatform.googleapis.com/v1/projects/%s/locations/global/models/%s", projectID, modelName): {
			StatusCode: http.StatusOK,
			Body:       model,
		},
		fmt.Sprintf("https://aiplatform.googleapis.com/v1/projects/%s/locations/global/models/%s", projectID, modelName2): {
			StatusCode: http.StatusOK,
			Body:       model2,
		},
		fmt.Sprintf("https://aiplatform.googleapis.com/v1/projects/%s/locations/global/models", projectID): {
			StatusCode: http.StatusOK,
			Body:       modelList,
		},
	}

	t.Run("Get", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		sdpItem, err := adapter.Get(ctx, projectID, modelName, true)
		if err != nil {
			t.Fatalf("Failed to get resource: %v", err)
		}

		// Validate SDP item properties
		if sdpItem.GetType() != sdpItemType.String() {
			t.Errorf("Expected type %s, got %s", sdpItemType.String(), sdpItem.GetType())
		}
		if sdpItem.UniqueAttributeValue() != modelName {
			t.Errorf("Expected unique attribute value '%s', got %s", modelName, sdpItem.UniqueAttributeValue())
		}
		if sdpItem.GetScope() != projectID {
			t.Errorf("Expected scope '%s', got %s", projectID, sdpItem.GetScope())
		}

		// Validate specific attributes
		val, err := sdpItem.GetAttributes().Get("name")
		if err != nil {
			t.Fatalf("Failed to get 'name' attribute: %v", err)
		}
		expectedName := fmt.Sprintf("projects/%s/locations/global/models/%s", projectID, modelName)
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
				// Pipeline job link
				{
					ExpectedType:   gcpshared.AIPlatformPipelineJob.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-pipeline",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// Deployed model endpoint link
				{
					ExpectedType:   gcpshared.AIPlatformEndpoint.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-endpoint",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				// Storage bucket link (artifactUri)
				{
					ExpectedType:   gcpshared.StorageBucket.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  fmt.Sprintf("%s-model-artifacts", projectID),
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
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), projectID)
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
			fmt.Sprintf("https://aiplatform.googleapis.com/v1/projects/%s/locations/global/models/%s", projectID, modelName): {
				StatusCode: http.StatusNotFound,
				Body:       map[string]interface{}{"error": "Model not found"},
			},
		}

		httpCli := shared.NewMockHTTPClientProvider(errorResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		_, err = adapter.Get(ctx, projectID, modelName, true)
		if err == nil {
			t.Error("Expected error when getting non-existent resource, but got nil")
		}
	})
}
