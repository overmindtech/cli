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
)

func TestAIPlatformModelDeploymentMonitoringJob(t *testing.T) {
	ctx := context.Background()
	projectID := "test-project"
	location := "us-central1"
	linker := gcpshared.NewLinker()
	jobName := "test-monitoring-job"

	job := &aiplatformpb.ModelDeploymentMonitoringJob{
		Name: fmt.Sprintf("projects/%s/locations/%s/modelDeploymentMonitoringJobs/%s", projectID, location, jobName),
		EncryptionSpec: &aiplatformpb.EncryptionSpec{
			KmsKeyName: "projects/test-project/locations/global/keyRings/test-ring/cryptoKeys/test-key",
		},
		Endpoint: fmt.Sprintf("projects/%s/locations/%s/endpoints/test-endpoint", projectID, location),
		ModelDeploymentMonitoringObjectiveConfigs: []*aiplatformpb.ModelDeploymentMonitoringObjectiveConfig{
			{
				DeployedModelId: "deployed-model-123",
			},
		},
		ModelMonitoringAlertConfig: &aiplatformpb.ModelMonitoringAlertConfig{
			NotificationChannels: []string{
				fmt.Sprintf("projects/%s/notificationChannels/alert-channel-1", projectID),
				fmt.Sprintf("projects/%s/notificationChannels/alert-channel-2", projectID),
			},
		},
	}

	jobName2 := "test-monitoring-job-2"
	job2 := &aiplatformpb.ModelDeploymentMonitoringJob{
		Name: fmt.Sprintf("projects/%s/locations/%s/modelDeploymentMonitoringJobs/%s", projectID, location, jobName2),
	}

	jobList := &aiplatformpb.ListModelDeploymentMonitoringJobsResponse{
		ModelDeploymentMonitoringJobs: []*aiplatformpb.ModelDeploymentMonitoringJob{job, job2},
	}

	sdpItemType := gcpshared.AIPlatformModelDeploymentMonitoringJob

	expectedCallAndResponses := map[string]shared.MockResponse{
		fmt.Sprintf("https://aiplatform.googleapis.com/v1/projects/%s/locations/%s/modelDeploymentMonitoringJobs/%s", projectID, location, jobName): {
			StatusCode: http.StatusOK,
			Body:       job,
		},
		fmt.Sprintf("https://aiplatform.googleapis.com/v1/projects/%s/locations/%s/modelDeploymentMonitoringJobs/%s", projectID, location, jobName2): {
			StatusCode: http.StatusOK,
			Body:       job2,
		},
		fmt.Sprintf("https://aiplatform.googleapis.com/v1/projects/%s/locations/%s/modelDeploymentMonitoringJobs", projectID, location): {
			StatusCode: http.StatusOK,
			Body:       jobList,
		},
	}

	t.Run("Get", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		combinedQuery := shared.CompositeLookupKey(location, jobName)
		sdpItem, err := adapter.Get(ctx, projectID, combinedQuery, true)
		if err != nil {
			t.Fatalf("Failed to get resource: %v", err)
		}

		if sdpItem.GetType() != sdpItemType.String() {
			t.Errorf("Expected type %s, got %s", sdpItemType.String(), sdpItem.GetType())
		}
		if sdpItem.UniqueAttributeValue() != combinedQuery {
			t.Errorf("Expected unique attribute value '%s', got %s", combinedQuery, sdpItem.UniqueAttributeValue())
		}

		// Include static tests - covers ALL blast propagation links
		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				// KMS encryption key link
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
				// AI Platform Endpoint link (bidirectional)
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
				// Deployed Model ID link (AI Platform Model)
				{
					ExpectedType:   gcpshared.AIPlatformModel.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "deployed-model-123",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// Notification Channel 1 link
				{
					ExpectedType:   gcpshared.MonitoringNotificationChannel.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "alert-channel-1",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// Notification Channel 2 link
				{
					ExpectedType:   gcpshared.MonitoringNotificationChannel.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "alert-channel-2",
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
			t.Skipf("Adapter for %s does not implement SearchableAdapter", sdpItemType)
		}

		sdpItems, err := searchable.Search(ctx, projectID, location, true)
		if err != nil {
			t.Fatalf("Failed to search resources: %v", err)
		}

		if len(sdpItems) != 2 {
			t.Errorf("Expected 2 resources, got %d", len(sdpItems))
		}
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		errorResponses := map[string]shared.MockResponse{
			fmt.Sprintf("https://aiplatform.googleapis.com/v1/projects/%s/locations/%s/modelDeploymentMonitoringJobs/%s", projectID, location, jobName): {
				StatusCode: http.StatusNotFound,
				Body:       map[string]interface{}{"error": "Monitoring job not found"},
			},
		}

		httpCli := shared.NewMockHTTPClientProvider(errorResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		combinedQuery := shared.CompositeLookupKey(location, jobName)
		_, err = adapter.Get(ctx, projectID, combinedQuery, true)
		if err == nil {
			t.Error("Expected error when getting non-existent resource, but got nil")
		}
	})
}
