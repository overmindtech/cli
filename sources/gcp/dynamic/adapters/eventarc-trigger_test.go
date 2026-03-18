package adapters_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"cloud.google.com/go/eventarc/apiv1/eventarcpb"

	"github.com/overmindtech/cli/go/discovery"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
	"github.com/overmindtech/cli/sources/gcp/dynamic"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

func TestEventarcTrigger(t *testing.T) {
	ctx := context.Background()
	projectID := "test-project"
	location := "us-central1"
	linker := gcpshared.NewLinker()
	triggerName := "test-trigger"

	trigger := &eventarcpb.Trigger{
		Name:           fmt.Sprintf("projects/%s/locations/%s/triggers/%s", projectID, location, triggerName),
		ServiceAccount: fmt.Sprintf("test-sa@%s.iam.gserviceaccount.com", projectID),
	}

	triggerName2 := "test-trigger-2"
	trigger2 := &eventarcpb.Trigger{
		Name: fmt.Sprintf("projects/%s/locations/%s/triggers/%s", projectID, location, triggerName2),
	}

	triggerList := &eventarcpb.ListTriggersResponse{
		Triggers: []*eventarcpb.Trigger{trigger, trigger2},
	}

	sdpItemType := gcpshared.EventarcTrigger

	expectedCallAndResponses := map[string]shared.MockResponse{
		fmt.Sprintf("https://eventarc.googleapis.com/v1/projects/%s/locations/%s/triggers/%s", projectID, location, triggerName): {
			StatusCode: http.StatusOK,
			Body:       trigger,
		},
		fmt.Sprintf("https://eventarc.googleapis.com/v1/projects/%s/locations/%s/triggers/%s", projectID, location, triggerName2): {
			StatusCode: http.StatusOK,
			Body:       trigger2,
		},
		fmt.Sprintf("https://eventarc.googleapis.com/v1/projects/%s/locations/%s/triggers", projectID, location): {
			StatusCode: http.StatusOK,
			Body:       triggerList,
		},
		fmt.Sprintf("https://eventarc.googleapis.com/v1/projects/%s/locations/-/triggers", projectID): {
			StatusCode: http.StatusOK,
			Body:       triggerList,
		},
	}

	t.Run("Get", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		combinedQuery := shared.CompositeLookupKey(location, triggerName)
		sdpItem, err := adapter.Get(ctx, projectID, combinedQuery, true)
		if err != nil {
			t.Fatalf("Failed to get Eventarc trigger: %v", err)
		}

		if sdpItem.GetType() != sdpItemType.String() {
			t.Errorf("Expected type %s, got %s", sdpItemType.String(), sdpItem.GetType())
		}
		if sdpItem.UniqueAttributeValue() != combinedQuery {
			t.Errorf("Expected unique attribute value '%s', got %s", combinedQuery, sdpItem.UniqueAttributeValue())
		}
		if sdpItem.GetScope() != projectID {
			t.Errorf("Expected scope '%s', got %s", projectID, sdpItem.GetScope())
		}

		val, err := sdpItem.GetAttributes().Get("name")
		if err != nil {
			t.Fatalf("Failed to get 'name' attribute: %v", err)
		}
		expectedName := fmt.Sprintf("projects/%s/locations/%s/triggers/%s", projectID, location, triggerName)
		if val != expectedName {
			t.Errorf("Expected name field to be '%s', got %s", expectedName, val)
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					ExpectedType:   gcpshared.IAMServiceAccount.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  fmt.Sprintf("test-sa@%s.iam.gserviceaccount.com", projectID),
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

		sdpItems, err := searchable.Search(ctx, projectID, location, true)
		if err != nil {
			t.Fatalf("Failed to search Eventarc triggers: %v", err)
		}

		if len(sdpItems) != 2 {
			t.Errorf("Expected 2 Eventarc triggers, got %d", len(sdpItems))
		}

		if len(sdpItems) >= 1 {
			item := sdpItems[0]
			if item.GetType() != sdpItemType.String() {
				t.Errorf("Expected type %s, got %s", sdpItemType.String(), item.GetType())
			}
			if item.GetScope() != projectID {
				t.Errorf("Expected scope '%s', got %s", projectID, item.GetScope())
			}
		}
	})

	t.Run("List", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		listable, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			t.Fatalf("Adapter for %s does not implement ListableAdapter", sdpItemType)
		}

		sdpItems, err := listable.List(ctx, projectID, true)
		if err != nil {
			t.Fatalf("Failed to list Eventarc triggers: %v", err)
		}

		if len(sdpItems) != 2 {
			t.Errorf("Expected 2 Eventarc triggers, got %d", len(sdpItems))
		}

		if len(sdpItems) >= 1 {
			item := sdpItems[0]
			if item.GetType() != sdpItemType.String() {
				t.Errorf("Expected type %s, got %s", sdpItemType.String(), item.GetType())
			}
			if item.GetScope() != projectID {
				t.Errorf("Expected scope '%s', got %s", projectID, item.GetScope())
			}
		}
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		errorResponses := map[string]shared.MockResponse{
			fmt.Sprintf("https://eventarc.googleapis.com/v1/projects/%s/locations/%s/triggers/%s", projectID, location, triggerName): {
				StatusCode: http.StatusNotFound,
				Body:       map[string]any{"error": "Trigger not found"},
			},
		}

		httpCli := shared.NewMockHTTPClientProvider(errorResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		combinedQuery := shared.CompositeLookupKey(location, triggerName)
		_, err = adapter.Get(ctx, projectID, combinedQuery, true)
		if err == nil {
			t.Error("Expected error when getting non-existent Eventarc trigger, but got nil")
		}
	})
}
