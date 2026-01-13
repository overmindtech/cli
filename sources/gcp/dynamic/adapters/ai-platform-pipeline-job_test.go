package adapters_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"google.golang.org/api/aiplatform/v1"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
	"github.com/overmindtech/cli/sources/gcp/dynamic"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

func TestAIPlatformPipelineJob(t *testing.T) {
	ctx := context.Background()
	projectID := "test-project"
	linker := gcpshared.NewLinker()
	jobID := "test-pipeline-job"

	pipelineJob := &aiplatform.GoogleCloudAiplatformV1PipelineJob{
		Name:           fmt.Sprintf("projects/%s/locations/global/pipelineJobs/%s", projectID, jobID),
		ServiceAccount: "aiplatform-sa@test-project.iam.gserviceaccount.com",
		Network:        fmt.Sprintf("projects/%s/global/networks/default", projectID),
		EncryptionSpec: &aiplatform.GoogleCloudAiplatformV1EncryptionSpec{
			KmsKeyName: "projects/test-project/locations/global/keyRings/my-keyring/cryptoKeys/my-key",
		},
	}

	jobList := &aiplatform.GoogleCloudAiplatformV1ListPipelineJobsResponse{
		PipelineJobs: []*aiplatform.GoogleCloudAiplatformV1PipelineJob{pipelineJob},
	}

	sdpItemType := gcpshared.AIPlatformPipelineJob

	expectedCallAndResponses := map[string]shared.MockResponse{
		fmt.Sprintf("https://aiplatform.googleapis.com/v1/projects/%s/locations/global/pipelineJobs/%s", projectID, jobID): {
			StatusCode: http.StatusOK,
			Body:       pipelineJob,
		},
		fmt.Sprintf("https://aiplatform.googleapis.com/v1/projects/%s/locations/global/pipelineJobs", projectID): {
			StatusCode: http.StatusOK,
			Body:       jobList,
		},
	}

	t.Run("Get", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		sdpItem, err := adapter.Get(ctx, projectID, jobID, true)
		if err != nil {
			t.Fatalf("Failed to get pipeline job: %v", err)
		}

		if sdpItem.GetType() != sdpItemType.String() {
			t.Errorf("Expected type %s, got %s", sdpItemType.String(), sdpItem.GetType())
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					// serviceAccount
					ExpectedType:   gcpshared.IAMServiceAccount.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "aiplatform-sa@test-project.iam.gserviceaccount.com",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					// network
					ExpectedType:   gcpshared.ComputeNetwork.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "default",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					// encryptionSpec.kmsKeyName
					ExpectedType:   gcpshared.CloudKMSCryptoKey.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey("global", "my-keyring", "my-key"),
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
			t.Fatalf("Adapter for %s does not implement ListableAdapter", sdpItemType)
		}

		sdpItems, err := listable.List(ctx, projectID, true)
		if err != nil {
			t.Fatalf("Failed to list pipeline jobs: %v", err)
		}

		if len(sdpItems) != 1 {
			t.Errorf("Expected 1 pipeline job, got %d", len(sdpItems))
		}
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		errorResponses := map[string]shared.MockResponse{
			fmt.Sprintf("https://aiplatform.googleapis.com/v1/projects/%s/locations/global/pipelineJobs/%s", projectID, jobID): {
				StatusCode: http.StatusNotFound,
				Body:       map[string]interface{}{"error": "Pipeline job not found"},
			},
		}

		httpCli := shared.NewMockHTTPClientProvider(errorResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		_, err = adapter.Get(ctx, projectID, jobID, true)
		if err == nil {
			t.Error("Expected error when getting non-existent pipeline job, but got nil")
		}
	})
}
