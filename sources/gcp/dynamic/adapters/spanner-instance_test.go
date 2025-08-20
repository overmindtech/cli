package adapters_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"cloud.google.com/go/spanner/admin/instance/apiv1/instancepb"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources/gcp/dynamic"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

func TestSpannerInstance(t *testing.T) {
	ctx := context.Background()
	projectID := "test-project"
	linker := gcpshared.NewLinker()
	instanceName := "test-instance"
	spannerInstance := &instancepb.Instance{
		Name:        fmt.Sprintf("projects/%s/instances/%s", projectID, instanceName),
		DisplayName: "Test Spanner Instance",
		Config:      "projects/test-project/instanceConfigs/regional-us-central1",
		NodeCount:   3,
		State:       instancepb.Instance_READY,
		Labels: map[string]string{
			"env":  "test",
			"team": "devops",
		},
		ProcessingUnits: 1000,
	}

	spannerInstances := &instancepb.ListInstancesResponse{
		Instances: []*instancepb.Instance{spannerInstance},
	}

	sdpItemType := gcpshared.SpannerInstance

	expectedCallAndResponses := map[string]shared.MockResponse{
		fmt.Sprintf("https://spanner.googleapis.com/v1/projects/%s/instances/%s", projectID, instanceName): {
			StatusCode: http.StatusOK,
			Body:       spannerInstance,
		},
		fmt.Sprintf("https://spanner.googleapis.com/v1/projects/%s/instances", projectID): {
			StatusCode: http.StatusOK,
			Body:       spannerInstances,
		},
	}

	t.Run("Get", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		getQuery := instanceName
		sdpItem, err := adapter.Get(ctx, projectID, getQuery, true)
		if err != nil {
			t.Fatalf("Failed to get Spanner instance: %v", err)
		}

		if sdpItem.GetType() != sdpItemType.String() {
			t.Errorf("Expected type %s, got %s", sdpItemType.String(), sdpItem.GetType())
		}
		if sdpItem.UniqueAttributeValue() != getQuery {
			t.Errorf("Expected unique attribute value '%s', got %s", instanceName, sdpItem.UniqueAttributeValue())
		}
		if sdpItem.GetScope() != projectID {
			t.Errorf("Expected scope '%s', got %s", projectID, sdpItem.GetScope())
		}
		val, err := sdpItem.GetAttributes().Get("name")
		if err != nil {
			t.Fatalf("Failed to get 'name' attribute: %v", err)
		}
		if val != fmt.Sprintf("projects/%s/instances/%s", projectID, instanceName) {
			t.Errorf("Expected name field to be 'projects/%s/instances/%s', got %s", projectID, instanceName, val)
		}
		val, err = sdpItem.GetAttributes().Get("display_name")
		if err != nil {
			t.Fatalf("Failed to get 'display_name' attribute: %v", err)
		}
		if val != "Test Spanner Instance" {
			t.Errorf("Expected display_name field to be 'Test Spanner Instance', got %s", val)
		}
		val, err = sdpItem.GetAttributes().Get("config")
		if err != nil {
			t.Fatalf("Failed to get 'config' attribute: %v", err)
		}
		if val != "projects/test-project/instanceConfigs/regional-us-central1" {
			t.Errorf("Expected config field to be 'projects/test-project/instanceConfigs/regional-us-central1', got %s", val)
		}
		val, err = sdpItem.GetAttributes().Get("node_count")
		if err != nil {
			t.Fatalf("Failed to get 'node_count' attribute: %v", err)
		}
		converted, ok := val.(float64)
		if !ok {
			t.Fatalf("Expected node_count to be a float64, got %T", val)
		}
		if converted != 3 {
			t.Errorf("Expected node_count field to be '3', got %s", val)
		}
		val, err = sdpItem.GetAttributes().Get("state")
		if err != nil {
			t.Fatalf("Failed to get 'state' attribute: %v", err)
		}
		converted, ok = val.(float64)
		if !ok {
			t.Fatalf("Expected state to be a float64, got %T", val)
		}
		if instancepb.Instance_State(converted) != instancepb.Instance_READY {
			t.Errorf("Expected state field to be 'READY', got %s", val)
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					ExpectedType:   gcpshared.SpannerInstanceConfig.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "regional-us-central1",
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

	t.Run("List", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(gcpshared.SpannerInstance, linker, httpCli, projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		listable, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			t.Fatalf("Adapter is not a ListableAdapter")
		}

		sdpItems, err := listable.List(ctx, projectID, true)
		if err != nil {
			t.Fatalf("Failed to list Spanner instances: %v", err)
		}

		if len(sdpItems) != 1 {
			t.Errorf("Expected 1 Spanner instance, got %d", len(sdpItems))
		}
	})
}
