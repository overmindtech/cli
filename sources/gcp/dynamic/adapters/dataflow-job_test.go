package adapters_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/overmindtech/cli/go/discovery"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
	"github.com/overmindtech/cli/sources/gcp/dynamic"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

func TestDataflowJob(t *testing.T) {
	ctx := context.Background()
	projectID := "test-project"
	linker := gcpshared.NewLinker()
	location := "us-central1"
	jobID := "2024-01-15_test-job-id-123"

	dataflowJob := map[string]any{
		"id":               jobID,
		"name":             fmt.Sprintf("projects/%s/locations/%s/jobs/%s", projectID, location, jobID),
		"type":             "JOB_TYPE_STREAMING",
		"currentState":     "JOB_STATE_RUNNING",
		"currentStateTime": "2024-01-15T10:30:00Z",
		"environment": map[string]any{
			"serviceAccountEmail": fmt.Sprintf("dataflow-sa@%s.iam.gserviceaccount.com", projectID),
			"serviceKmsKeyName":   fmt.Sprintf("projects/%s/locations/%s/keyRings/dataflow-ring/cryptoKeys/dataflow-key", projectID, location),
			"workerPools": []any{
				map[string]any{
					"network":     fmt.Sprintf("projects/%s/global/networks/dataflow-network", projectID),
					"subnetwork":  fmt.Sprintf("projects/%s/regions/%s/subnetworks/dataflow-subnet", projectID, location),
					"machineType": "n1-standard-4",
					"numWorkers":  float64(3),
				},
			},
		},
		"jobMetadata": map[string]any{
			"pubsubDetails": []any{
				map[string]any{
					"topic":        fmt.Sprintf("projects/%s/topics/input-topic", projectID),
					"subscription": fmt.Sprintf("projects/%s/subscriptions/input-subscription", projectID),
				},
				map[string]any{
					"topic":        fmt.Sprintf("projects/%s/topics/output-topic", projectID),
					"subscription": fmt.Sprintf("projects/%s/subscriptions/output-subscription", projectID),
				},
			},
			"bigqueryDetails": []any{
				map[string]any{
					"table":   fmt.Sprintf("projects/%s/datasets/analytics/tables/events", projectID),
					"dataset": fmt.Sprintf("projects/%s/datasets/analytics", projectID),
				},
			},
			"spannerDetails": []any{
				map[string]any{
					"instanceId": "spanner-instance-1",
				},
			},
			"bigTableDetails": []any{
				map[string]any{
					"instanceId": "bigtable-instance-1",
				},
			},
		},
	}

	jobID2 := "2024-01-15_test-job-id-456"
	dataflowJob2 := map[string]any{
		"id":           jobID2,
		"name":         fmt.Sprintf("projects/%s/locations/%s/jobs/%s", projectID, location, jobID2),
		"type":         "JOB_TYPE_BATCH",
		"currentState": "JOB_STATE_DONE",
	}

	dataflowJobsList := map[string]any{
		"jobs": []any{dataflowJob, dataflowJob2},
	}

	sdpItemType := gcpshared.DataflowJob

	expectedCallAndResponses := map[string]shared.MockResponse{
		fmt.Sprintf("https://dataflow.googleapis.com/v1b3/projects/%s/locations/%s/jobs/%s", projectID, location, jobID): {
			StatusCode: http.StatusOK,
			Body:       dataflowJob,
		},
		fmt.Sprintf("https://dataflow.googleapis.com/v1b3/projects/%s/locations/%s/jobs", projectID, location): {
			StatusCode: http.StatusOK,
			Body:       dataflowJobsList,
		},
		fmt.Sprintf("https://dataflow.googleapis.com/v1b3/projects/%s/jobs:aggregated", projectID): {
			StatusCode: http.StatusOK,
			Body:       dataflowJobsList,
		},
	}

	t.Run("Get", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		getQuery := shared.CompositeLookupKey(location, jobID)
		sdpItem, err := adapter.Get(ctx, projectID, getQuery, true)
		if err != nil {
			t.Fatalf("Failed to get Dataflow Job: %v", err)
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

		val, err := sdpItem.GetAttributes().Get("name")
		if err != nil {
			t.Fatalf("Failed to get 'name' attribute: %v", err)
		}
		expectedName := fmt.Sprintf("projects/%s/locations/%s/jobs/%s", projectID, location, jobID)
		if val != expectedName {
			t.Errorf("Expected name '%s', got %s", expectedName, val)
		}

		val, err = sdpItem.GetAttributes().Get("currentState")
		if err != nil {
			t.Fatalf("Failed to get 'currentState' attribute: %v", err)
		}
		if val != "JOB_STATE_RUNNING" {
			t.Errorf("Expected currentState 'JOB_STATE_RUNNING', got %s", val)
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				// Pub/Sub topic links (from pubsubDetails array)
				{
					ExpectedType:   gcpshared.PubSubTopic.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "input-topic",
					ExpectedScope:  projectID,
				},
				{
					ExpectedType:   gcpshared.PubSubTopic.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "output-topic",
					ExpectedScope:  projectID,
				},
				// Pub/Sub subscription links (from pubsubDetails array)
				{
					ExpectedType:   gcpshared.PubSubSubscription.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "input-subscription",
					ExpectedScope:  projectID,
				},
				{
					ExpectedType:   gcpshared.PubSubSubscription.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "output-subscription",
					ExpectedScope:  projectID,
				},
				// BigQuery links
				{
					ExpectedType:   gcpshared.BigQueryTable.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey("analytics", "events"),
					ExpectedScope:  projectID,
				},
				{
					ExpectedType:   gcpshared.BigQueryDataset.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "analytics",
					ExpectedScope:  projectID,
				},
				// Spanner instance link (plain name resolves for single-key types)
				{
					ExpectedType:   gcpshared.SpannerInstance.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "spanner-instance-1",
					ExpectedScope:  projectID,
				},
				// Bigtable instance link (plain name resolves for single-key types)
				{
					ExpectedType:   gcpshared.BigTableAdminInstance.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "bigtable-instance-1",
					ExpectedScope:  projectID,
				},
				// IAM service account link
				{
					ExpectedType:   gcpshared.IAMServiceAccount.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  fmt.Sprintf("dataflow-sa@%s.iam.gserviceaccount.com", projectID),
					ExpectedScope:  projectID,
				},
				// KMS crypto key link
				{
					ExpectedType:   gcpshared.CloudKMSCryptoKey.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey(location, "dataflow-ring", "dataflow-key"),
					ExpectedScope:  projectID,
				},
				// Compute network link
				{
					ExpectedType:   gcpshared.ComputeNetwork.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "dataflow-network",
					ExpectedScope:  projectID,
				},
				// Compute subnetwork link (regional — scope includes region)
				{
					ExpectedType:   gcpshared.ComputeSubnetwork.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "dataflow-subnet",
					ExpectedScope:  fmt.Sprintf("%s.%s", projectID, location),
				},
			}

			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("Search", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
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
			t.Fatalf("Failed to search Dataflow Jobs: %v", err)
		}

		if len(sdpItems) != 2 {
			t.Errorf("Expected 2 Dataflow Jobs, got %d", len(sdpItems))
		}

		if len(sdpItems) >= 1 {
			item := sdpItems[0]
			if item.GetType() != sdpItemType.String() {
				t.Errorf("Expected type %s, got %s", sdpItemType.String(), item.GetType())
			}
			expectedUniqueAttr := shared.CompositeLookupKey(location, jobID)
			if item.UniqueAttributeValue() != expectedUniqueAttr {
				t.Errorf("Expected unique attribute value '%s', got %s", expectedUniqueAttr, item.UniqueAttributeValue())
			}
		}

		if len(sdpItems) >= 2 {
			item := sdpItems[1]
			if item.GetType() != sdpItemType.String() {
				t.Errorf("Expected type %s, got %s", sdpItemType.String(), item.GetType())
			}
			expectedUniqueAttr2 := shared.CompositeLookupKey(location, jobID2)
			if item.UniqueAttributeValue() != expectedUniqueAttr2 {
				t.Errorf("Expected unique attribute value '%s', got %s", expectedUniqueAttr2, item.UniqueAttributeValue())
			}
		}
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		errorResponses := map[string]shared.MockResponse{
			fmt.Sprintf("https://dataflow.googleapis.com/v1b3/projects/%s/locations/%s/jobs/%s", projectID, location, jobID): {
				StatusCode: http.StatusNotFound,
				Body:       map[string]any{"error": "Job not found"},
			},
		}

		httpCli := shared.NewMockHTTPClientProvider(errorResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		getQuery := shared.CompositeLookupKey(location, jobID)
		_, err = adapter.Get(ctx, projectID, getQuery, true)
		if err == nil {
			t.Error("Expected error when getting non-existent Dataflow Job, but got nil")
		}
	})

	t.Run("List", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		listable, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			t.Fatalf("Adapter is not a ListableAdapter")
		}

		sdpItems, err := listable.List(ctx, projectID, true)
		if err != nil {
			t.Fatalf("Failed to list Dataflow Jobs: %v", err)
		}

		if len(sdpItems) != 2 {
			t.Errorf("Expected 2 Dataflow Jobs, got %d", len(sdpItems))
		}

		if len(sdpItems) >= 1 {
			item := sdpItems[0]
			expectedUniqueAttr := shared.CompositeLookupKey(location, jobID)
			if item.UniqueAttributeValue() != expectedUniqueAttr {
				t.Errorf("Expected unique attribute value '%s', got %s", expectedUniqueAttr, item.UniqueAttributeValue())
			}
		}

		if len(sdpItems) >= 2 {
			item := sdpItems[1]
			expectedUniqueAttr2 := shared.CompositeLookupKey(location, jobID2)
			if item.UniqueAttributeValue() != expectedUniqueAttr2 {
				t.Errorf("Expected unique attribute value '%s', got %s", expectedUniqueAttr2, item.UniqueAttributeValue())
			}
		}
	})
}
