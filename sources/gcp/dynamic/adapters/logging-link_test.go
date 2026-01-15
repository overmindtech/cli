package adapters_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"cloud.google.com/go/logging/apiv2/loggingpb"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
	"github.com/overmindtech/cli/sources/gcp/dynamic"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

func TestLoggingLink(t *testing.T) {
	ctx := context.Background()
	projectID := "test-project"
	location := "global"
	bucketName := "test-bucket"
	linkName := "test-link"
	linker := gcpshared.NewLinker()

	link := &loggingpb.Link{
		Name: fmt.Sprintf("projects/%s/locations/%s/buckets/%s/links/%s", projectID, location, bucketName, linkName),
		BigqueryDataset: &loggingpb.BigQueryDataset{
			DatasetId: fmt.Sprintf("bigquery.googleapis.com/projects/%s/datasets/test_dataset", projectID),
		},
	}

	linkList := &loggingpb.ListLinksResponse{
		Links: []*loggingpb.Link{link},
	}

	sdpItemType := gcpshared.LoggingLink

	expectedCallAndResponses := map[string]shared.MockResponse{
		fmt.Sprintf("https://logging.googleapis.com/v2/projects/%s/locations/%s/buckets/%s/links/%s", projectID, location, bucketName, linkName): {
			StatusCode: http.StatusOK,
			Body:       link,
		},
		fmt.Sprintf("https://logging.googleapis.com/v2/projects/%s/locations/%s/buckets/%s/links", projectID, location, bucketName): {
			StatusCode: http.StatusOK,
			Body:       linkList,
		},
	}

	t.Run("Get", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		getQuery := shared.CompositeLookupKey(location, bucketName, linkName)
		sdpItem, err := adapter.Get(ctx, projectID, getQuery, true)
		if err != nil {
			t.Fatalf("Failed to get logging link: %v", err)
		}

		if sdpItem.GetType() != sdpItemType.String() {
			t.Errorf("Expected type %s, got %s", sdpItemType.String(), sdpItem.GetType())
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					// name (LoggingBucket)
					ExpectedType:   gcpshared.LoggingBucket.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey(location, bucketName),
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					// bigqueryDataset.datasetId
					ExpectedType:   gcpshared.BigQueryDataset.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test_dataset",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
			}

			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("Search", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter for %s does not implement SearchableAdapter", sdpItemType)
		}

		searchQuery := shared.CompositeLookupKey(location, bucketName)
		sdpItems, err := searchable.Search(ctx, projectID, searchQuery, true)
		if err != nil {
			t.Fatalf("Failed to search logging links: %v", err)
		}

		if len(sdpItems) != 1 {
			t.Errorf("Expected 1 logging link, got %d", len(sdpItems))
		}
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		errorResponses := map[string]shared.MockResponse{
			fmt.Sprintf("https://logging.googleapis.com/v2/projects/%s/locations/%s/buckets/%s/links/%s", projectID, location, bucketName, linkName): {
				StatusCode: http.StatusNotFound,
				Body:       map[string]interface{}{"error": "Link not found"},
			},
		}

		httpCli := shared.NewMockHTTPClientProvider(errorResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		getQuery := shared.CompositeLookupKey(location, bucketName, linkName)
		_, err = adapter.Get(ctx, projectID, getQuery, true)
		if err == nil {
			t.Error("Expected error when getting non-existent logging link, but got nil")
		}
	})
}
