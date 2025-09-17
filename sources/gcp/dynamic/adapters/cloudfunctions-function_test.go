package adapters_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"cloud.google.com/go/functions/apiv2/functionspb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources/gcp/dynamic"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

func TestCloudFunctionsFunction(t *testing.T) {
	ctx := context.Background()
	projectID := "test-project"
	linker := gcpshared.NewLinker()
	location := "us-central1"
	functionName := "test-function"

	// Mock response for a Cloud Function
	cloudFunction := &functionspb.Function{
		Name:        fmt.Sprintf("projects/%s/locations/%s/functions/%s", projectID, location, functionName),
		Description: "Test Cloud Function for HTTP requests",
		UpdateTime: &timestamppb.Timestamp{
			Seconds: 1673784600, // 2023-01-15T10:30:00Z
		},
		Labels: map[string]string{
			"env":  "test",
			"team": "backend",
		},
		State:      functionspb.Function_ACTIVE,
		Url:        fmt.Sprintf("https://%s-%s.cloudfunctions.net/test-function", location, projectID),
		KmsKeyName: fmt.Sprintf("projects/%s/locations/%s/keyRings/test-ring/cryptoKeys/test-key", projectID, location),
		ServiceConfig: &functionspb.ServiceConfig{
			ServiceAccountEmail: fmt.Sprintf("test-function@%s.iam.gserviceaccount.com", projectID),
			VpcConnector:        fmt.Sprintf("projects/%s/locations/%s/connectors/test-connector", projectID, location),
			Service:             fmt.Sprintf("projects/%s/locations/%s/services/test-function-service", projectID, location),
			EnvironmentVariables: map[string]string{
				"ENV": "test",
			},
		},
		BuildConfig: &functionspb.BuildConfig{
			Runtime:          "python39",
			EntryPoint:       "main",
			DockerRepository: fmt.Sprintf("projects/%s/locations/%s/repositories/test-docker-repo", projectID, location),
			WorkerPool:       fmt.Sprintf("projects/%s/locations/%s/workerPools/test-worker-pool", projectID, location),
			Source: &functionspb.Source{
				Source: &functionspb.Source_StorageSource{
					StorageSource: &functionspb.StorageSource{
						Bucket: "test-bucket",
						Object: "function-source.zip",
					},
				},
			},
			SourceProvenance: &functionspb.SourceProvenance{
				ResolvedStorageSource: &functionspb.StorageSource{
					Bucket: "test-resolved-bucket",
					Object: "resolved-function-source.zip",
				},
			},
		},
		EventTrigger: &functionspb.EventTrigger{
			PubsubTopic:         fmt.Sprintf("projects/%s/topics/test-topic", projectID),
			ServiceAccountEmail: fmt.Sprintf("event-trigger@%s.iam.gserviceaccount.com", projectID),
			Trigger:             fmt.Sprintf("projects/%s/locations/%s/triggers/test-trigger", projectID, location),
		},
	}

	// Mock response for a second Cloud Function
	functionName2 := "test-function-2"
	cloudFunction2 := &functionspb.Function{
		Name:        fmt.Sprintf("projects/%s/locations/%s/functions/%s", projectID, location, functionName2),
		Description: "Second test Cloud Function for Pub/Sub events",
		UpdateTime: &timestamppb.Timestamp{
			Seconds: 1673871900, // 2023-01-16T11:45:00Z
		},
		Labels: map[string]string{
			"env":     "prod",
			"service": "event-processor",
		},
		State: functionspb.Function_ACTIVE,
		BuildConfig: &functionspb.BuildConfig{
			Runtime:    "nodejs18",
			EntryPoint: "handler",
			Source: &functionspb.Source{
				Source: &functionspb.Source_StorageSource{
					StorageSource: &functionspb.StorageSource{
						Bucket: "test-bucket-2",
						Object: "function-source-2.zip",
					},
				},
			},
		},
	}

	// Mock response for list operation
	cloudFunctionsList := &functionspb.ListFunctionsResponse{
		Functions: []*functionspb.Function{
			cloudFunction,
			cloudFunction2,
		},
		NextPageToken: "",
	}

	sdpItemType := gcpshared.CloudFunctionsFunction

	expectedCallAndResponses := map[string]shared.MockResponse{
		fmt.Sprintf("https://cloudfunctions.googleapis.com/v2/projects/%s/locations/%s/functions/%s", projectID, location, functionName): {
			StatusCode: http.StatusOK,
			Body:       cloudFunction,
		},
		fmt.Sprintf("https://cloudfunctions.googleapis.com/v2/projects/%s/locations/%s/functions", projectID, location): {
			StatusCode: http.StatusOK,
			Body:       cloudFunctionsList,
		},
	}

	t.Run("Get", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		getQuery := shared.CompositeLookupKey(location, functionName)
		sdpItem, err := adapter.Get(ctx, projectID, getQuery, true)
		if err != nil {
			t.Fatalf("Failed to get Cloud Function: %v", err)
		}

		if sdpItem.GetType() != sdpItemType.String() {
			t.Errorf("Expected type %s, got %s", sdpItemType.String(), sdpItem.GetType())
		}
		if sdpItem.UniqueAttributeValue() != getQuery {
			t.Errorf("Expected unique attribute value '%s', got %s", getQuery, sdpItem.UniqueAttributeValue())
		}
		if sdpItem.GetScope() != projectID {
			t.Errorf("Expected scope '%s', got %s", projectID, sdpItem.GetScope())
		}

		// Test specific attributes
		val, err := sdpItem.GetAttributes().Get("name")
		if err != nil {
			t.Fatalf("Failed to get 'name' attribute: %v", err)
		}
		expectedName := fmt.Sprintf("projects/%s/locations/%s/functions/%s", projectID, location, functionName)
		if val != expectedName {
			t.Errorf("Expected name field to be '%s', got %s", expectedName, val)
		}

		val, err = sdpItem.GetAttributes().Get("description")
		if err != nil {
			t.Fatalf("Failed to get 'description' attribute: %v", err)
		}
		if val != "Test Cloud Function for HTTP requests" {
			t.Errorf("Expected description field to be 'Test Cloud Function for HTTP requests', got %s", val)
		}

		val, err = sdpItem.GetAttributes().Get("state")
		if err != nil {
			t.Fatalf("Failed to get 'state' attribute: %v", err)
		}
		if val != "ACTIVE" {
			t.Errorf("Expected state field to be 'ACTIVE', got %s", val)
		}

		// Test buildConfig.runtime attribute (nested in v2 API)
		buildConfig, err := sdpItem.GetAttributes().Get("buildConfig")
		if err != nil {
			t.Fatalf("Failed to get 'buildConfig' attribute: %v", err)
		}
		buildConfigMap, ok := buildConfig.(map[string]interface{})
		if !ok {
			t.Fatalf("Expected buildConfig to be a map, got %T", buildConfig)
		}
		if buildConfigMap["runtime"] != "python39" {
			t.Errorf("Expected buildConfig.runtime to be 'python39', got %s", buildConfigMap["runtime"])
		}
		if buildConfigMap["entryPoint"] != "main" {
			t.Errorf("Expected buildConfig.entryPoint to be 'main', got %s", buildConfigMap["entryPoint"])
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				// Test KMS key link
				{
					ExpectedType:   gcpshared.CloudKMSCryptoKey.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey(location, "test-ring", "test-key"),
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// Test storage bucket link (buildConfig.source.storageSource.bucket)
				{
					ExpectedType:   gcpshared.StorageBucket.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-bucket",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// Test service account link (serviceConfig.serviceAccountEmail)
				{
					ExpectedType:   gcpshared.IAMServiceAccount.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  fmt.Sprintf("test-function@%s.iam.gserviceaccount.com", projectID),
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// Test Pub/Sub topic link (eventTrigger.pubsubTopic)
				{
					ExpectedType:   gcpshared.PubSubTopic.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-topic",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// Test event trigger service account link (eventTrigger.serviceAccountEmail)
				{
					ExpectedType:   gcpshared.IAMServiceAccount.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  fmt.Sprintf("event-trigger@%s.iam.gserviceaccount.com", projectID),
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// Test Eventarc trigger link (eventTrigger.trigger)
				{
					ExpectedType:   gcpshared.EventarcTrigger.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey(location, "test-trigger"),
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// Test Cloud Run service link (serviceConfig.service)
				{
					ExpectedType:   gcpshared.RunService.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey(location, "test-function-service"),
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// Test Artifact Registry repository link (buildConfig.dockerRepository)
				{
					ExpectedType:   gcpshared.ArtifactRegistryRepository.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey(location, "test-docker-repo"),
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// Test Cloud Run Worker Pool link (buildConfig.workerPool)
				{
					ExpectedType:   gcpshared.RunWorkerPool.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey(location, "test-worker-pool"),
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// Test resolved storage bucket link (buildConfig.sourceProvenance.resolvedStorageSource.bucket)
				{
					ExpectedType:   gcpshared.StorageBucket.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-resolved-bucket",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// Note: serviceConfig.vpcConnector test case omitted because gcp-vpc-access-connector adapter doesn't exist
			}

			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("Search", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter is not a SearchableAdapter")
		}

		searchQuery := location
		sdpItems, err := searchable.Search(ctx, projectID, searchQuery, true)
		if err != nil {
			t.Fatalf("Failed to search Cloud Functions: %v", err)
		}

		if len(sdpItems) != 2 {
			t.Errorf("Expected 2 Cloud Functions, got %d", len(sdpItems))
		}

		if len(sdpItems) >= 1 {
			item := sdpItems[0]
			if item.GetType() != sdpItemType.String() {
				t.Errorf("Expected type %s, got %s", sdpItemType.String(), item.GetType())
			}
			expectedUniqueAttr := shared.CompositeLookupKey(location, functionName)
			if item.UniqueAttributeValue() != expectedUniqueAttr {
				t.Errorf("Expected unique attribute value '%s', got %s", expectedUniqueAttr, item.UniqueAttributeValue())
			}
		}

		if len(sdpItems) >= 2 {
			item := sdpItems[1]
			if item.GetType() != sdpItemType.String() {
				t.Errorf("Expected type %s, got %s", sdpItemType.String(), item.GetType())
			}
			expectedUniqueAttr2 := shared.CompositeLookupKey(location, functionName2)
			if item.UniqueAttributeValue() != expectedUniqueAttr2 {
				t.Errorf("Expected unique attribute value '%s', got %s", expectedUniqueAttr2, item.UniqueAttributeValue())
			}
		}
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		// Test with error responses to simulate API errors
		errorResponses := map[string]shared.MockResponse{
			fmt.Sprintf("https://cloudfunctions.googleapis.com/v2/projects/%s/locations/%s/functions/%s", projectID, location, functionName): {
				StatusCode: http.StatusNotFound,
				Body:       map[string]interface{}{"error": "Function not found"},
			},
		}

		httpCli := shared.NewMockHTTPClientProvider(errorResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		getQuery := shared.CompositeLookupKey(location, functionName)
		_, err = adapter.Get(ctx, projectID, getQuery, true)
		if err == nil {
			t.Error("Expected error when getting non-existent Cloud Function, but got nil")
		}
	})
}
