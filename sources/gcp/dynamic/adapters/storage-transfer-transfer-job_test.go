package adapters_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"cloud.google.com/go/storagetransfer/apiv1/storagetransferpb"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources/gcp/dynamic"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

func TestStorageTransferTransferJob(t *testing.T) {
	ctx := context.Background()
	projectID := "test-project"
	linker := gcpshared.NewLinker()
	jobName := "transferJobs/123456789"
	jobID := "123456789" // Just the ID for the Get query

	job := &storagetransferpb.TransferJob{
		Name:           jobName,
		ServiceAccount: "test-sa@test-project.iam.gserviceaccount.com",
		TransferSpec: &storagetransferpb.TransferSpec{
			DataSource: &storagetransferpb.TransferSpec_GcsDataSource{
				GcsDataSource: &storagetransferpb.GcsData{
					BucketName: "source-bucket",
				},
			},
			DataSink: &storagetransferpb.TransferSpec_GcsDataSink{
				GcsDataSink: &storagetransferpb.GcsData{
					BucketName: "dest-bucket",
				},
			},
		},
		NotificationConfig: &storagetransferpb.NotificationConfig{
			PubsubTopic: fmt.Sprintf("projects/%s/topics/transfer-notifications", projectID),
		},
	}

	// Second job with HTTP data source, intermediate location, and event stream
	jobName2 := "transferJobs/123456790"
	jobID2 := "123456790"
	job2 := &storagetransferpb.TransferJob{
		Name:           jobName2,
		ServiceAccount: "test-sa2@test-project.iam.gserviceaccount.com",
		TransferSpec: &storagetransferpb.TransferSpec{
			DataSource: &storagetransferpb.TransferSpec_HttpDataSource{
				HttpDataSource: &storagetransferpb.HttpData{
					ListUrl: "https://example.com/urllist.tsv",
				},
			},
			DataSink: &storagetransferpb.TransferSpec_GcsDataSink{
				GcsDataSink: &storagetransferpb.GcsData{
					BucketName: "http-dest-bucket",
				},
			},
			IntermediateDataLocation: &storagetransferpb.TransferSpec_GcsIntermediateDataLocation{
				GcsIntermediateDataLocation: &storagetransferpb.GcsData{
					BucketName: "intermediate-bucket",
				},
			},
		},
		EventStream: &storagetransferpb.EventStream{
			Name: fmt.Sprintf("projects/%s/subscriptions/transfer-events", projectID),
		},
	}

	jobList := &storagetransferpb.ListTransferJobsResponse{
		TransferJobs: []*storagetransferpb.TransferJob{job, job2},
	}

	sdpItemType := gcpshared.StorageTransferTransferJob

	expectedCallAndResponses := map[string]shared.MockResponse{
		fmt.Sprintf("https://storagetransfer.googleapis.com/v1/transferJobs/%s?projectId=%s", jobID, projectID): {
			StatusCode: http.StatusOK,
			Body:       job,
		},
		fmt.Sprintf("https://storagetransfer.googleapis.com/v1/transferJobs/%s?projectId=%s", jobID2, projectID): {
			StatusCode: http.StatusOK,
			Body:       job2,
		},
		fmt.Sprintf("https://storagetransfer.googleapis.com/v1/transferJobs?filter={\"projectId\":\"%s\"}", projectID): {
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

		sdpItem, err := adapter.Get(ctx, projectID, jobID, true)
		if err != nil {
			t.Fatalf("Failed to get resource: %v", err)
		}

		if sdpItem.GetType() != sdpItemType.String() {
			t.Errorf("Expected type %s, got %s", sdpItemType.String(), sdpItem.GetType())
		}
		if sdpItem.UniqueAttributeValue() != jobID {
			t.Errorf("Expected unique attribute value '%s', got %s", jobID, sdpItem.UniqueAttributeValue())
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				// transferSpec.gcsDataSource.bucketName
				{
					ExpectedType:   gcpshared.StorageBucket.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "source-bucket",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// transferSpec.gcsDataSink.bucketName
				{
					ExpectedType:   gcpshared.StorageBucket.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "dest-bucket",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// serviceAccount
				{
					ExpectedType:   gcpshared.IAMServiceAccount.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-sa@test-project.iam.gserviceaccount.com",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// notificationConfig.pubsubTopic
				{
					ExpectedType:   gcpshared.PubSubTopic.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "transfer-notifications",
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

	t.Run("Get with HTTP source and intermediate location", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		sdpItem, err := adapter.Get(ctx, projectID, jobID2, true)
		if err != nil {
			t.Fatalf("Failed to get resource: %v", err)
		}

		if sdpItem.GetType() != sdpItemType.String() {
			t.Errorf("Expected type %s, got %s", sdpItemType.String(), sdpItem.GetType())
		}
		if sdpItem.UniqueAttributeValue() != jobID2 {
			t.Errorf("Expected unique attribute value '%s', got %s", jobID2, sdpItem.UniqueAttributeValue())
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				// transferSpec.gcsDataSink.bucketName
				{
					ExpectedType:   gcpshared.StorageBucket.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "http-dest-bucket",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// transferSpec.gcsIntermediateDataLocation.bucketName
				{
					ExpectedType:   gcpshared.StorageBucket.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "intermediate-bucket",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// serviceAccount
				{
					ExpectedType:   gcpshared.IAMServiceAccount.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-sa2@test-project.iam.gserviceaccount.com",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// eventStream.name
				{
					ExpectedType:   gcpshared.PubSubSubscription.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "transfer-events",
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
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		errorResponses := map[string]shared.MockResponse{
			fmt.Sprintf("https://storagetransfer.googleapis.com/v1/transferJobs/%s?projectId=%s", jobID, projectID): {
				StatusCode: http.StatusNotFound,
				Body:       map[string]interface{}{"error": "Transfer job not found"},
			},
		}

		httpCli := shared.NewMockHTTPClientProvider(errorResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		_, err = adapter.Get(ctx, projectID, jobID, true)
		if err == nil {
			t.Error("Expected error when getting non-existent resource, but got nil")
		}
	})
}
