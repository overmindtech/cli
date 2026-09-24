package adapters_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"google.golang.org/api/run/v2"

	"github.com/overmindtech/cli/go/discovery"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
	"github.com/overmindtech/cli/sources/gcp/dynamic"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

func TestRunRevision(t *testing.T) {
	ctx := context.Background()
	projectID := "test-project"
	location := "us-central1"
	serviceName := "test-service"
	revisionName := "test-revision"
	linker := gcpshared.NewLinker()
	const (
		plaintextEnvSentinel = "ovm-run-rev-env-sentinel-8b6f"
		plaintextEnvName     = "DATABASE_URL"
		secretEnvName        = "API_KEY"
		secretRef            = "projects/test-project/secrets/rev-api-key"
	)

	revision := &run.GoogleCloudRunV2Revision{
		Name:           fmt.Sprintf("projects/%s/locations/%s/services/%s/revisions/%s", projectID, location, serviceName, revisionName),
		ServiceAccount: "run-sa@test-project.iam.gserviceaccount.com",
		Service:        fmt.Sprintf("projects/%s/locations/%s/services/%s", projectID, location, serviceName),
		Containers: []*run.GoogleCloudRunV2Container{
			{
				Image: fmt.Sprintf("%s-docker.pkg.dev/%s/repo/image:latest", location, projectID),
				Env: []*run.GoogleCloudRunV2EnvVar{
					{
						Name:  plaintextEnvName,
						Value: plaintextEnvSentinel,
					},
					{
						Name: secretEnvName,
						ValueSource: &run.GoogleCloudRunV2EnvVarSource{
							SecretKeyRef: &run.GoogleCloudRunV2SecretKeySelector{
								Secret: secretRef,
							},
						},
					},
				},
			},
		},
	}

	revisionList := &run.GoogleCloudRunV2ListRevisionsResponse{
		Revisions: []*run.GoogleCloudRunV2Revision{revision},
	}

	sdpItemType := gcpshared.RunRevision

	expectedCallAndResponses := map[string]shared.MockResponse{
		fmt.Sprintf("https://run.googleapis.com/v2/projects/%s/locations/%s/services/%s/revisions/%s", projectID, location, serviceName, revisionName): {
			StatusCode: http.StatusOK,
			Body:       revision,
		},
		fmt.Sprintf("https://run.googleapis.com/v2/projects/%s/locations/%s/services/%s/revisions", projectID, location, serviceName): {
			StatusCode: http.StatusOK,
			Body:       revisionList,
		},
	}

	t.Run("Get", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		getQuery := shared.CompositeLookupKey(location, serviceName, revisionName)
		sdpItem, err := adapter.Get(ctx, projectID, getQuery, true)
		if err != nil {
			t.Fatalf("Failed to get revision: %v", err)
		}

		if sdpItem.GetType() != sdpItemType.String() {
			t.Errorf("Expected type %s, got %s", sdpItemType.String(), sdpItem.GetType())
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
		if !strings.Contains(string(attrsJSON), secretRef) {
			t.Errorf("expected secret ref to remain in attributes: %s", attrsJSON)
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					// service
					ExpectedType:   gcpshared.RunService.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey(location, serviceName),
					ExpectedScope:  projectID,
				},
				{
					// serviceAccount
					ExpectedType:   gcpshared.IAMServiceAccount.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "run-sa@test-project.iam.gserviceaccount.com",
					ExpectedScope:  projectID,
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
			t.Fatalf("Adapter for %s does not implement SearchableAdapter", sdpItemType)
		}

		searchQuery := shared.CompositeLookupKey(location, serviceName)
		sdpItems, err := searchable.Search(ctx, projectID, searchQuery, true)
		if err != nil {
			t.Fatalf("Failed to search revisions: %v", err)
		}

		if len(sdpItems) != 1 {
			t.Errorf("Expected 1 revision, got %d", len(sdpItems))
		}
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		errorResponses := map[string]shared.MockResponse{
			fmt.Sprintf("https://run.googleapis.com/v2/projects/%s/locations/%s/services/%s/revisions/%s", projectID, location, serviceName, revisionName): {
				StatusCode: http.StatusNotFound,
				Body:       map[string]any{"error": "Revision not found"},
			},
		}

		httpCli := shared.NewMockHTTPClientProvider(errorResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		getQuery := shared.CompositeLookupKey(location, serviceName, revisionName)
		_, err = adapter.Get(ctx, projectID, getQuery, true)
		if err == nil {
			t.Error("Expected error when getting non-existent revision, but got nil")
		}
	})
}
