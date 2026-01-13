package adapters_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"cloud.google.com/go/resourcemanager/apiv3/resourcemanagerpb"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
	"github.com/overmindtech/cli/sources/gcp/dynamic"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

func TestCloudResourceManagerTagValue(t *testing.T) {
	ctx := context.Background()
	projectID := "test-project"
	linker := gcpshared.NewLinker()
	tagValueID := "123456789"
	tagKeyID := "987654321"

	tagValue := &resourcemanagerpb.TagValue{
		Name:   fmt.Sprintf("tagValues/%s", tagValueID),
		Parent: fmt.Sprintf("tagKeys/%s", tagKeyID),
	}

	tagValueID2 := "123456790"
	tagValue2 := &resourcemanagerpb.TagValue{
		Name:   fmt.Sprintf("tagValues/%s", tagValueID2),
		Parent: fmt.Sprintf("tagKeys/%s", tagKeyID),
	}

	tagValueList := &resourcemanagerpb.ListTagValuesResponse{
		TagValues: []*resourcemanagerpb.TagValue{tagValue, tagValue2},
	}

	sdpItemType := gcpshared.CloudResourceManagerTagValue

	expectedCallAndResponses := map[string]shared.MockResponse{
		fmt.Sprintf("https://cloudresourcemanager.googleapis.com/v3/tagValues/%s", tagValueID): {
			StatusCode: http.StatusOK,
			Body:       tagValue,
		},
		fmt.Sprintf("https://cloudresourcemanager.googleapis.com/v3/tagValues/%s", tagValueID2): {
			StatusCode: http.StatusOK,
			Body:       tagValue2,
		},
		fmt.Sprintf("https://cloudresourcemanager.googleapis.com/v3/tagValues?parent=tagKeys/%s", tagKeyID): {
			StatusCode: http.StatusOK,
			Body:       tagValueList,
		},
	}

	t.Run("Get", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		sdpItem, err := adapter.Get(ctx, projectID, tagValueID, true)
		if err != nil {
			t.Fatalf("Failed to get resource: %v", err)
		}

		if sdpItem.GetType() != sdpItemType.String() {
			t.Errorf("Expected type %s, got %s", sdpItemType.String(), sdpItem.GetType())
		}
		if sdpItem.UniqueAttributeValue() != tagValueID {
			t.Errorf("Expected unique attribute value '%s', got %s", tagValueID, sdpItem.UniqueAttributeValue())
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					ExpectedType:   gcpshared.CloudResourceManagerTagKey.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  tagKeyID,
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
			t.Skipf("Adapter for %s does not implement SearchableAdapter", sdpItemType)
		}

		sdpItems, err := searchable.Search(ctx, projectID, tagKeyID, true)
		if err != nil {
			t.Fatalf("Failed to search resources: %v", err)
		}

		if len(sdpItems) != 2 {
			t.Errorf("Expected 2 resources, got %d", len(sdpItems))
		}
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		errorResponses := map[string]shared.MockResponse{
			fmt.Sprintf("https://cloudresourcemanager.googleapis.com/v3/tagValues/%s", tagValueID): {
				StatusCode: http.StatusNotFound,
				Body:       map[string]interface{}{"error": "Tag value not found"},
			},
		}

		httpCli := shared.NewMockHTTPClientProvider(errorResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		_, err = adapter.Get(ctx, projectID, tagValueID, true)
		if err == nil {
			t.Error("Expected error when getting non-existent resource, but got nil")
		}
	})
}
