package adapters_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"google.golang.org/api/logging/v2"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sources/gcp/dynamic"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

func TestLoggingSavedQuery(t *testing.T) {
	ctx := context.Background()
	projectID := "test-project"
	location := "global"
	linker := gcpshared.NewLinker()
	queryName := "test-query"

	savedQuery := &logging.SavedQuery{
		Name:        fmt.Sprintf("projects/%s/locations/%s/savedQueries/%s", projectID, location, queryName),
		DisplayName: "Test Query",
	}

	queryList := &logging.ListSavedQueriesResponse{
		SavedQueries: []*logging.SavedQuery{savedQuery},
	}

	sdpItemType := gcpshared.LoggingSavedQuery

	expectedCallAndResponses := map[string]shared.MockResponse{
		fmt.Sprintf("https://logging.googleapis.com/v2/projects/%s/locations/%s/savedQueries/%s", projectID, location, queryName): {
			StatusCode: http.StatusOK,
			Body:       savedQuery,
		},
		fmt.Sprintf("https://logging.googleapis.com/v2/projects/%s/locations/%s/savedQueries", projectID, location): {
			StatusCode: http.StatusOK,
			Body:       queryList,
		},
	}

	t.Run("Get", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		getQuery := shared.CompositeLookupKey(location, queryName)
		sdpItem, err := adapter.Get(ctx, projectID, getQuery, true)
		if err != nil {
			t.Fatalf("Failed to get saved query: %v", err)
		}

		if sdpItem.GetType() != sdpItemType.String() {
			t.Errorf("Expected type %s, got %s", sdpItemType.String(), sdpItem.GetType())
		}
	})

	t.Run("Search", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter for %s does not implement SearchableAdapter", sdpItemType)
		}

		sdpItems, err := searchable.Search(ctx, projectID, location, true)
		if err != nil {
			t.Fatalf("Failed to search saved queries: %v", err)
		}

		if len(sdpItems) != 1 {
			t.Errorf("Expected 1 saved query, got %d", len(sdpItems))
		}
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		errorResponses := map[string]shared.MockResponse{
			fmt.Sprintf("https://logging.googleapis.com/v2/projects/%s/locations/%s/savedQueries/%s", projectID, location, queryName): {
				StatusCode: http.StatusNotFound,
				Body:       map[string]interface{}{"error": "Saved query not found"},
			},
		}

		httpCli := shared.NewMockHTTPClientProvider(errorResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		getQuery := shared.CompositeLookupKey(location, queryName)
		_, err = adapter.Get(ctx, projectID, getQuery, true)
		if err == nil {
			t.Error("Expected error when getting non-existent saved query, but got nil")
		}
	})
}
