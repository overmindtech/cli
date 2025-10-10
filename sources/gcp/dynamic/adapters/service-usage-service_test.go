package adapters_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"google.golang.org/api/serviceusage/v1"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources/gcp/dynamic"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

func TestServiceUsageService(t *testing.T) {
	ctx := context.Background()
	projectID := "test-project"
	linker := gcpshared.NewLinker()
	serviceName := "compute.googleapis.com"

	service := &serviceusage.GoogleApiServiceusageV1Service{
		Name: fmt.Sprintf("projects/%s/services/%s", projectID, serviceName),
		Config: &serviceusage.GoogleApiServiceusageV1ServiceConfig{
			Name: serviceName,
		},
		State: "ENABLED",
	}

	serviceList := &serviceusage.ListServicesResponse{
		Services: []*serviceusage.GoogleApiServiceusageV1Service{service},
	}

	sdpItemType := gcpshared.ServiceUsageService

	expectedCallAndResponses := map[string]shared.MockResponse{
		fmt.Sprintf("https://serviceusage.googleapis.com/v1/projects/%s/services/%s", projectID, serviceName): {
			StatusCode: http.StatusOK,
			Body:       service,
		},
		fmt.Sprintf("https://serviceusage.googleapis.com/v1/projects/%s/services?filter=state:ENABLED", projectID): {
			StatusCode: http.StatusOK,
			Body:       serviceList,
		},
	}

	t.Run("Get", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		sdpItem, err := adapter.Get(ctx, projectID, serviceName, true)
		if err != nil {
			t.Fatalf("Failed to get service: %v", err)
		}

		if sdpItem.GetType() != sdpItemType.String() {
			t.Errorf("Expected type %s, got %s", sdpItemType.String(), sdpItem.GetType())
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					// config.name
					ExpectedType:   stdlib.NetworkDNS.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  serviceName,
					ExpectedScope:  "global",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
			}

			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("List", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		listable, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			t.Fatalf("Adapter for %s does not implement ListableAdapter", sdpItemType)
		}

		sdpItems, err := listable.List(ctx, projectID, true)
		if err != nil {
			t.Fatalf("Failed to list services: %v", err)
		}

		if len(sdpItems) != 1 {
			t.Errorf("Expected 1 service, got %d", len(sdpItems))
		}
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		errorResponses := map[string]shared.MockResponse{
			fmt.Sprintf("https://serviceusage.googleapis.com/v1/projects/%s/services/%s", projectID, serviceName): {
				StatusCode: http.StatusNotFound,
				Body:       map[string]interface{}{"error": "Service not found"},
			},
		}

		httpCli := shared.NewMockHTTPClientProvider(errorResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		_, err = adapter.Get(ctx, projectID, serviceName, true)
		if err == nil {
			t.Error("Expected error when getting non-existent service, but got nil")
		}
	})
}
