package adapters_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"cloud.google.com/go/dataplex/apiv1/dataplexpb"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources/gcp/dynamic"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

func TestDataplexDataScan(t *testing.T) {
	ctx := context.Background()
	projectID := "test-project"
	location := "us-central1"
	linker := gcpshared.NewLinker()
	dataScanName := "test-data-scan"

	// Create mock protobuf object with storage bucket resource
	bucketName := "test-bucket"
	dataScan := &dataplexpb.DataScan{
		Name: fmt.Sprintf("projects/%s/locations/%s/dataScans/%s", projectID, location, dataScanName),
		Data: &dataplexpb.DataSource{
			Source: &dataplexpb.DataSource_Resource{
				Resource: bucketName,
			},
		},
	}

	// Create second data scan for search testing
	dataScanName2 := "test-data-scan-2"
	dataScan2 := &dataplexpb.DataScan{
		Name: fmt.Sprintf("projects/%s/locations/%s/dataScans/%s", projectID, location, dataScanName2),
		Data: &dataplexpb.DataSource{
			Source: &dataplexpb.DataSource_Resource{
				Resource: "test-bucket",
			},
		},
	}

	// Create list response with multiple items
	dataScanList := &dataplexpb.ListDataScansResponse{
		DataScans: []*dataplexpb.DataScan{dataScan, dataScan2},
	}

	sdpItemType := gcpshared.DataplexDataScan

	// Mock HTTP responses
	expectedCallAndResponses := map[string]shared.MockResponse{
		fmt.Sprintf("https://dataplex.googleapis.com/v1/projects/%s/locations/%s/dataScans/%s", projectID, location, dataScanName): {
			StatusCode: http.StatusOK,
			Body:       dataScan,
		},
		fmt.Sprintf("https://dataplex.googleapis.com/v1/projects/%s/locations/%s/dataScans/%s", projectID, location, dataScanName2): {
			StatusCode: http.StatusOK,
			Body:       dataScan2,
		},
		fmt.Sprintf("https://dataplex.googleapis.com/v1/projects/%s/locations/%s/dataScans", projectID, location): {
			StatusCode: http.StatusOK,
			Body:       dataScanList,
		},
	}

	t.Run("Get", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		combinedQuery := shared.CompositeLookupKey(location, dataScanName)
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

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				// Storage bucket link
				{
					ExpectedType:   gcpshared.StorageBucket.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  bucketName,
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// Note: data.entity link also exists but DataplexEntity adapter doesn't exist yet
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

	t.Run("Search with Terraform format", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Skipf("Adapter for %s does not implement SearchableAdapter", sdpItemType)
		}

		terraformQuery := fmt.Sprintf("projects/%s/locations/%s/dataScans/%s", projectID, location, dataScanName)
		sdpItems, err := searchable.Search(ctx, projectID, terraformQuery, true)
		if err != nil {
			t.Fatalf("Failed to search resources with Terraform format: %v", err)
		}

		if len(sdpItems) != 1 {
			t.Errorf("Expected 1 resource, got %d", len(sdpItems))
		}
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		errorResponses := map[string]shared.MockResponse{
			fmt.Sprintf("https://dataplex.googleapis.com/v1/projects/%s/locations/%s/dataScans/%s", projectID, location, dataScanName): {
				StatusCode: http.StatusNotFound,
				Body:       map[string]interface{}{"error": "Data scan not found"},
			},
		}

		httpCli := shared.NewMockHTTPClientProvider(errorResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		combinedQuery := shared.CompositeLookupKey(location, dataScanName)
		_, err = adapter.Get(ctx, projectID, combinedQuery, true)
		if err == nil {
			t.Error("Expected error when getting non-existent resource, but got nil")
		}
	})
}
