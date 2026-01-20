package adapters_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"google.golang.org/api/run/v2"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
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

	revision := &run.GoogleCloudRunV2Revision{
		Name:           fmt.Sprintf("projects/%s/locations/%s/services/%s/revisions/%s", projectID, location, serviceName, revisionName),
		ServiceAccount: "run-sa@test-project.iam.gserviceaccount.com",
		Service:        fmt.Sprintf("projects/%s/locations/%s/services/%s", projectID, location, serviceName),
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

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					// service
					ExpectedType:   gcpshared.RunService.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey(location, serviceName),
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					// serviceAccount
					ExpectedType:   gcpshared.IAMServiceAccount.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "run-sa@test-project.iam.gserviceaccount.com",
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
				Body:       map[string]interface{}{"error": "Revision not found"},
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
