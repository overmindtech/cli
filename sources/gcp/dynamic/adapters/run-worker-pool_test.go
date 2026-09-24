package adapters_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"cloud.google.com/go/run/apiv2/runpb"

	"github.com/overmindtech/cli/go/sdpcache"
	"github.com/overmindtech/cli/sources/gcp/dynamic"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

func TestRunWorkerPool(t *testing.T) {
	ctx := context.Background()
	projectID := "test-project"
	location := "us-central1"
	linker := gcpshared.NewLinker()
	workerPoolName := "test-worker-pool"
	const (
		plaintextEnvSentinel = "ovm-run-wp-env-sentinel-3d5c"
		plaintextEnvName     = "DATABASE_URL"
	)

	workerPool := &runpb.WorkerPool{
		Name: fmt.Sprintf("projects/%s/locations/%s/workerPools/%s", projectID, location, workerPoolName),
		Template: &runpb.WorkerPoolRevisionTemplate{
			Containers: []*runpb.Container{
				{
					Image: fmt.Sprintf("%s-docker.pkg.dev/%s/repo/image:latest", location, projectID),
					Env: []*runpb.EnvVar{
						{
							Name: plaintextEnvName,
							Values: &runpb.EnvVar_Value{
								Value: plaintextEnvSentinel,
							},
						},
					},
				},
			},
		},
	}

	sdpItemType := gcpshared.RunWorkerPool

	expectedCallAndResponses := map[string]shared.MockResponse{
		fmt.Sprintf("https://run.googleapis.com/v2/projects/%s/locations/%s/workerPools/%s", projectID, location, workerPoolName): {
			StatusCode: http.StatusOK,
			Body:       workerPool,
		},
	}

	t.Run("Get", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		combinedQuery := shared.CompositeLookupKey(location, workerPoolName)
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

		attrMap := sdpItem.GetAttributes().GetAttrStruct().AsMap()
		attrsJSON, marshalErr := json.Marshal(attrMap)
		if marshalErr != nil {
			t.Fatalf("marshal attributes: %v", marshalErr)
		}
		if strings.Contains(string(attrsJSON), plaintextEnvSentinel) {
			t.Errorf("plaintext env value leaked in attributes: %s", attrsJSON)
		}
		if !strings.Contains(string(attrsJSON), plaintextEnvName) {
			t.Errorf("expected env name %q to remain in attributes: %s", plaintextEnvName, attrsJSON)
		}
	})
}
