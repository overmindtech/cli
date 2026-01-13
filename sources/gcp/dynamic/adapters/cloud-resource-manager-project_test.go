package adapters_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"google.golang.org/api/cloudresourcemanager/v3"

	"github.com/overmindtech/cli/sdpcache"
	"github.com/overmindtech/cli/sources/gcp/dynamic"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

func TestCloudResourceManagerProject(t *testing.T) {
	ctx := context.Background()
	projectID := "test-project"
	linker := gcpshared.NewLinker()

	project := &cloudresourcemanager.Project{
		Name:        fmt.Sprintf("projects/%s", projectID),
		ProjectId:   projectID,
		DisplayName: "Test Project",
		State:       "ACTIVE",
	}

	sdpItemType := gcpshared.CloudResourceManagerProject

	expectedCallAndResponses := map[string]shared.MockResponse{
		fmt.Sprintf("https://cloudresourcemanager.googleapis.com/v3/projects/%s", projectID): {
			StatusCode: http.StatusOK,
			Body:       project,
		},
	}

	t.Run("Get", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		sdpItem, err := adapter.Get(ctx, projectID, projectID, true)
		if err != nil {
			t.Fatalf("Failed to get project: %v", err)
		}

		if sdpItem.GetType() != sdpItemType.String() {
			t.Errorf("Expected type %s, got %s", sdpItemType.String(), sdpItem.GetType())
		}
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		errorResponses := map[string]shared.MockResponse{
			fmt.Sprintf("https://cloudresourcemanager.googleapis.com/v3/projects/%s", projectID): {
				StatusCode: http.StatusNotFound,
				Body:       map[string]interface{}{"error": "Project not found"},
			},
		}

		httpCli := shared.NewMockHTTPClientProvider(errorResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		_, err = adapter.Get(ctx, projectID, projectID, true)
		if err == nil {
			t.Error("Expected error when getting non-existent project, but got nil")
		}
	})
}
