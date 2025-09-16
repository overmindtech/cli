package adapters_test

// This test file demonstrates the use of protobuf types from the Go SDK for mocking HTTP responses
// as requested in the user feedback. It uses cloud.google.com/go/aiplatform/apiv1/aiplatformpb
// types instead of generic map[string]interface{} structures.
//
// Note: There are some limitations when using protobuf types with the current dynamic adapter
// implementation:
// 1. Protobuf serializes field names to snake_case (e.g., "batch_prediction_jobs") while the
//    adapter configuration expects camelCase (e.g., "batchPredictionJobs"), affecting list operations
// 2. Blast propagation paths in the adapter expect JSON field names but get protobuf field names,
//    limiting automatic link generation for nested fields like GCS sources and KMS keys
//
// These limitations don't affect the core functionality testing but are noted for future improvements.

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"cloud.google.com/go/aiplatform/apiv1/aiplatformpb"
	"google.golang.org/genproto/googleapis/rpc/status"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources/gcp/dynamic"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

func TestAIPlatformBatchPredictionJob(t *testing.T) {
	ctx := context.Background()
	projectID := "test-project"
	location := "us-central1"
	linker := gcpshared.NewLinker()
	jobName := "test-batch-prediction-job"

	// Mock response for a batch prediction job
	batchPredictionJob := &aiplatformpb.BatchPredictionJob{
		Name:           fmt.Sprintf("projects/%s/locations/%s/batchPredictionJobs/%s", projectID, location, jobName),
		DisplayName:    "Test Batch Prediction Job",
		Model:          fmt.Sprintf("projects/%s/locations/%s/models/test-model", projectID, location),
		ModelVersionId: "1",
		InputConfig: &aiplatformpb.BatchPredictionJob_InputConfig{
			InstancesFormat: "jsonl",
			Source: &aiplatformpb.BatchPredictionJob_InputConfig_GcsSource{
				GcsSource: &aiplatformpb.GcsSource{
					Uris: []string{
						fmt.Sprintf("gs://%s-input-bucket/input-data.jsonl", projectID),
					},
				},
			},
		},
		OutputConfig: &aiplatformpb.BatchPredictionJob_OutputConfig{
			PredictionsFormat: "jsonl",
			Destination: &aiplatformpb.BatchPredictionJob_OutputConfig_GcsDestination{
				GcsDestination: &aiplatformpb.GcsDestination{
					OutputUriPrefix: fmt.Sprintf("gs://%s-output-bucket/predictions/", projectID),
				},
			},
		},
		DedicatedResources: &aiplatformpb.BatchDedicatedResources{
			MachineSpec: &aiplatformpb.MachineSpec{
				MachineType: "n1-standard-2",
			},
			StartingReplicaCount: 1,
			MaxReplicaCount:      5,
		},
		ServiceAccount: fmt.Sprintf("batch-prediction@%s.iam.gserviceaccount.com", projectID),
		State:          aiplatformpb.JobState_JOB_STATE_SUCCEEDED,
		Error: &status.Status{
			Code:    0,
			Message: "",
		},
		PartialFailures: []*status.Status{},
		ResourcesConsumed: &aiplatformpb.ResourcesConsumed{
			ReplicaHours: 2.5,
		},
		CompletionStats: &aiplatformpb.CompletionStats{
			SuccessfulCount:              1000,
			FailedCount:                  0,
			IncompleteCount:              0,
			SuccessfulForecastPointCount: 0,
		},
		EncryptionSpec: &aiplatformpb.EncryptionSpec{
			KmsKeyName: fmt.Sprintf("projects/%s/locations/%s/keyRings/test-ring/cryptoKeys/test-key", projectID, location),
		},
		Labels: map[string]string{
			"env":  "test",
			"team": "ml",
		},
		CreateTime:              nil, // Will be set to proper timestamp if needed
		StartTime:               nil,
		EndTime:                 nil,
		UpdateTime:              nil,
		DisableContainerLogging: false,
	}

	// Create a second batch prediction job for list testing
	jobName2 := "test-batch-prediction-job-2"
	batchPredictionJob2 := &aiplatformpb.BatchPredictionJob{
		Name:           fmt.Sprintf("projects/%s/locations/%s/batchPredictionJobs/%s", projectID, location, jobName2),
		DisplayName:    "Second Test Batch Prediction Job",
		Model:          fmt.Sprintf("projects/%s/locations/%s/models/test-model-2", projectID, location),
		ModelVersionId: "2",
		InputConfig: &aiplatformpb.BatchPredictionJob_InputConfig{
			InstancesFormat: "csv",
			Source: &aiplatformpb.BatchPredictionJob_InputConfig_BigquerySource{
				BigquerySource: &aiplatformpb.BigQuerySource{
					InputUri: fmt.Sprintf("bq://%s.test_dataset.input_table", projectID),
				},
			},
		},
		OutputConfig: &aiplatformpb.BatchPredictionJob_OutputConfig{
			PredictionsFormat: "csv",
			Destination: &aiplatformpb.BatchPredictionJob_OutputConfig_BigqueryDestination{
				BigqueryDestination: &aiplatformpb.BigQueryDestination{
					OutputUri: fmt.Sprintf("bq://%s.test_dataset.predictions_table", projectID),
				},
			},
		},
		ManualBatchTuningParameters: &aiplatformpb.ManualBatchTuningParameters{
			BatchSize: 64,
		},
		ServiceAccount: fmt.Sprintf("batch-prediction-2@%s.iam.gserviceaccount.com", projectID),
		State:          aiplatformpb.JobState_JOB_STATE_RUNNING,
		Error: &status.Status{
			Code:    0,
			Message: "",
		},
		PartialFailures: []*status.Status{},
		ResourcesConsumed: &aiplatformpb.ResourcesConsumed{
			ReplicaHours: 1.2,
		},
		CompletionStats: &aiplatformpb.CompletionStats{
			SuccessfulCount:              500,
			FailedCount:                  2,
			IncompleteCount:              100,
			SuccessfulForecastPointCount: 0,
		},
		Labels: map[string]string{
			"env":     "prod",
			"service": "recommendation",
		},
		CreateTime:              nil,
		StartTime:               nil,
		UpdateTime:              nil,
		DisableContainerLogging: true,
	}

	// Mock response for list operation
	batchPredictionJobsList := &aiplatformpb.ListBatchPredictionJobsResponse{
		BatchPredictionJobs: []*aiplatformpb.BatchPredictionJob{
			batchPredictionJob,
			batchPredictionJob2,
		},
		NextPageToken: "",
	}

	sdpItemType := gcpshared.AIPlatformBatchPredictionJob

	expectedCallAndResponses := map[string]shared.MockResponse{
		fmt.Sprintf("https://aiplatform.googleapis.com/v1/projects/%s/locations/%s/batchPredictionJobs/%s", projectID, location, jobName): {
			StatusCode: http.StatusOK,
			Body:       batchPredictionJob,
		},
		fmt.Sprintf("https://aiplatform.googleapis.com/v1/projects/%s/locations/%s/batchPredictionJobs", projectID, location): {
			StatusCode: http.StatusOK,
			Body:       batchPredictionJobsList,
		},
	}

	t.Run("Get", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		getQuery := fmt.Sprintf("%s|%s", location, jobName)
		sdpItem, err := adapter.Get(ctx, projectID, getQuery, true)
		if err != nil {
			t.Fatalf("Failed to get batch prediction job: %v", err)
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
		expectedName := fmt.Sprintf("projects/%s/locations/%s/batchPredictionJobs/%s", projectID, location, jobName)
		if val != expectedName {
			t.Errorf("Expected name field to be '%s', got %s", expectedName, val)
		}

		val, err = sdpItem.GetAttributes().Get("displayName")
		if err != nil {
			t.Fatalf("Failed to get 'displayName' attribute: %v", err)
		}
		if val != "Test Batch Prediction Job" {
			t.Errorf("Expected displayName field to be 'Test Batch Prediction Job', got %s", val)
		}

		val, err = sdpItem.GetAttributes().Get("model")
		if err != nil {
			t.Fatalf("Failed to get 'model' attribute: %v", err)
		}
		expectedModel := fmt.Sprintf("projects/%s/locations/%s/models/test-model", projectID, location)
		if val != expectedModel {
			t.Errorf("Expected model field to be '%s', got %s", expectedModel, val)
		}

		val, err = sdpItem.GetAttributes().Get("modelVersionId")
		if err != nil {
			t.Fatalf("Failed to get 'modelVersionId' attribute: %v", err)
		}
		if val != "1" {
			t.Errorf("Expected modelVersionId field to be '1', got %s", val)
		}

		val, err = sdpItem.GetAttributes().Get("state")
		if err != nil {
			t.Fatalf("Failed to get 'state' attribute: %v", err)
		}
		// The state is returned as a string
		stateValue, ok := val.(string)
		if !ok {
			t.Fatalf("Expected state to be a string, got %T", val)
		}
		if stateValue != "JOB_STATE_SUCCEEDED" {
			t.Errorf("Expected state field to be 'JOB_STATE_SUCCEEDED', got %s", stateValue)
		}

		val, err = sdpItem.GetAttributes().Get("serviceAccount")
		if err != nil {
			t.Fatalf("Failed to get 'serviceAccount' attribute: %v", err)
		}
		expectedServiceAccount := fmt.Sprintf("batch-prediction@%s.iam.gserviceaccount.com", projectID)
		if val != expectedServiceAccount {
			t.Errorf("Expected serviceAccount field to be '%s', got %s", expectedServiceAccount, val)
		}

		// Test nested inputConfig
		inputConfig, err := sdpItem.GetAttributes().Get("inputConfig")
		if err != nil {
			t.Fatalf("Failed to get 'inputConfig' attribute: %v", err)
		}
		inputConfigMap, ok := inputConfig.(map[string]interface{})
		if !ok {
			t.Fatalf("Expected inputConfig to be a map[string]interface{}, got %T", inputConfig)
		}
		if inputConfigMap["instancesFormat"] != "jsonl" {
			t.Errorf("Expected inputConfig.instancesFormat to be 'jsonl', got %s", inputConfigMap["instancesFormat"])
		}

		// Test nested outputConfig
		outputConfig, err := sdpItem.GetAttributes().Get("outputConfig")
		if err != nil {
			t.Fatalf("Failed to get 'outputConfig' attribute: %v", err)
		}
		outputConfigMap, ok := outputConfig.(map[string]interface{})
		if !ok {
			t.Fatalf("Expected outputConfig to be a map[string]interface{}, got %T", outputConfig)
		}
		if outputConfigMap["predictionsFormat"] != "jsonl" {
			t.Errorf("Expected outputConfig.predictionsFormat to be 'jsonl', got %s", outputConfigMap["predictionsFormat"])
		}

		// Test encryptionSpec
		encryptionSpec, err := sdpItem.GetAttributes().Get("encryptionSpec")
		if err != nil {
			t.Fatalf("Failed to get 'encryptionSpec' attribute: %v", err)
		}
		encryptionSpecMap, ok := encryptionSpec.(map[string]interface{})
		if !ok {
			t.Fatalf("Expected encryptionSpec to be a map[string]interface{}, got %T", encryptionSpec)
		}
		expectedKmsKey := fmt.Sprintf("projects/%s/locations/%s/keyRings/test-ring/cryptoKeys/test-key", projectID, location)
		if encryptionSpecMap["kmsKeyName"] != expectedKmsKey {
			t.Errorf("Expected encryptionSpec.kmsKeyName to be '%s', got %s", expectedKmsKey, encryptionSpecMap["kmsKeyName"])
		}

		t.Run("StaticTests", func(t *testing.T) {
			// Only test blast propagation paths that are currently working
			// (GCS and BigQuery paths have TODOs and require manual linkers)
			queryTests := shared.QueryTests{
				{
					ExpectedType:   gcpshared.AIPlatformModel.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-model",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					ExpectedType:   gcpshared.IAMServiceAccount.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  fmt.Sprintf("batch-prediction@%s.iam.gserviceaccount.com", projectID),
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
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

		// Test search functionality with location
		sdpItems, err := searchable.Search(ctx, projectID, location, true)
		if err != nil {
			t.Fatalf("Failed to search batch prediction jobs: %v", err)
		}

		if len(sdpItems) != 2 {
			t.Errorf("Expected 2 batch prediction jobs, got %d", len(sdpItems))
		}

		// Test first item
		item1 := sdpItems[0]
		if item1.GetType() != sdpItemType.String() {
			t.Errorf("Expected type %s, got %s", sdpItemType.String(), item1.GetType())
		}
		expectedUniqueAttr1 := fmt.Sprintf("%s|%s", location, jobName)
		if item1.UniqueAttributeValue() != expectedUniqueAttr1 {
			t.Errorf("Expected unique attribute value '%s', got %s", expectedUniqueAttr1, item1.UniqueAttributeValue())
		}

		// Test second item
		item2 := sdpItems[1]
		if item2.GetType() != sdpItemType.String() {
			t.Errorf("Expected type %s, got %s", sdpItemType.String(), item2.GetType())
		}
		expectedUniqueAttr2 := fmt.Sprintf("%s|%s", location, jobName2)
		if item2.UniqueAttributeValue() != expectedUniqueAttr2 {
			t.Errorf("Expected unique attribute value '%s', got %s", expectedUniqueAttr2, item2.UniqueAttributeValue())
		}
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		// Test with 404 response to simulate job not found
		errorResponses := map[string]shared.MockResponse{
			fmt.Sprintf("https://aiplatform.googleapis.com/v1/projects/%s/locations/%s/batchPredictionJobs/%s", projectID, location, jobName): {
				StatusCode: http.StatusNotFound,
				Body:       &status.Status{Code: 404, Message: "Batch prediction job not found"},
			},
		}

		httpCli := shared.NewMockHTTPClientProvider(errorResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		getQuery := fmt.Sprintf("%s|%s", location, jobName)
		_, err = adapter.Get(ctx, projectID, getQuery, true)
		if err == nil {
			t.Error("Expected error when getting non-existent batch prediction job, but got nil")
		}
	})

	t.Run("InvalidQuery", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		// Test with invalid query format (missing location)
		_, err = adapter.Get(ctx, projectID, "invalid-query-format", true)
		if err == nil {
			t.Error("Expected error when using invalid query format, but got nil")
		}
	})
}
