package adapters_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"google.golang.org/api/dataplex/v1"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdpcache"
	"github.com/overmindtech/cli/sources/gcp/dynamic"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

func TestDataplexEntryGroup(t *testing.T) {
	ctx := context.Background()
	projectID := "test-project"
	location := "us-central1"
	linker := gcpshared.NewLinker()
	entryGroupID := "test-entry-group"

	entryGroup := &dataplex.GoogleCloudDataplexV1EntryGroup{
		Name: fmt.Sprintf("projects/%s/locations/%s/entryGroups/%s", projectID, location, entryGroupID),
	}

	entryGroupList := &dataplex.GoogleCloudDataplexV1ListEntryGroupsResponse{
		EntryGroups: []*dataplex.GoogleCloudDataplexV1EntryGroup{entryGroup},
	}

	sdpItemType := gcpshared.DataplexEntryGroup

	expectedCallAndResponses := map[string]shared.MockResponse{
		fmt.Sprintf("https://dataplex.googleapis.com/v1/projects/%s/locations/%s/entryGroups/%s", projectID, location, entryGroupID): {
			StatusCode: http.StatusOK,
			Body:       entryGroup,
		},
		fmt.Sprintf("https://dataplex.googleapis.com/v1/projects/%s/locations/%s/entryGroups", projectID, location): {
			StatusCode: http.StatusOK,
			Body:       entryGroupList,
		},
	}

	t.Run("Get", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		getQuery := shared.CompositeLookupKey(location, entryGroupID)
		sdpItem, err := adapter.Get(ctx, projectID, getQuery, true)
		if err != nil {
			t.Fatalf("Failed to get dataplex entry group: %v", err)
		}

		if sdpItem.GetType() != sdpItemType.String() {
			t.Errorf("Expected type %s, got %s", sdpItemType.String(), sdpItem.GetType())
		}
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

		sdpItems, err := searchable.Search(ctx, projectID, location, true)
		if err != nil {
			t.Fatalf("Failed to search dataplex entry groups: %v", err)
		}

		if len(sdpItems) != 1 {
			t.Errorf("Expected 1 dataplex entry group, got %d", len(sdpItems))
		}
	})

	t.Run("Search with Terraform format", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Skipf("Adapter for %s does not implement SearchableAdapter", sdpItemType)
		}

		// Test Terraform format: projects/[project]/locations/[location]/entryGroups/[entry_group_id]
		terraformQuery := fmt.Sprintf("projects/%s/locations/%s/entryGroups/%s", projectID, location, entryGroupID)
		sdpItems, err := searchable.Search(ctx, projectID, terraformQuery, true)
		if err != nil {
			t.Fatalf("Failed to search resources with Terraform format: %v", err)
		}

		// The search should return only the specific resource matching the Terraform format
		if len(sdpItems) != 1 {
			t.Errorf("Expected 1 resource, got %d", len(sdpItems))
			return
		}

		// Verify the single item returned
		firstItem := sdpItems[0]
		if firstItem.GetType() != sdpItemType.String() {
			t.Errorf("Expected first item type %s, got %s", sdpItemType.String(), firstItem.GetType())
		}
		if firstItem.GetScope() != projectID {
			t.Errorf("Expected first item scope '%s', got %s", projectID, firstItem.GetScope())
		}
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		errorResponses := map[string]shared.MockResponse{
			fmt.Sprintf("https://dataplex.googleapis.com/v1/projects/%s/locations/%s/entryGroups/%s", projectID, location, entryGroupID): {
				StatusCode: http.StatusNotFound,
				Body:       map[string]interface{}{"error": "Entry group not found"},
			},
		}

		httpCli := shared.NewMockHTTPClientProvider(errorResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		getQuery := shared.CompositeLookupKey(location, entryGroupID)
		_, err = adapter.Get(ctx, projectID, getQuery, true)
		if err == nil {
			t.Error("Expected error when getting non-existent dataplex entry group, but got nil")
		}
	})
}
