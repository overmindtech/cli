package adapters_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"cloud.google.com/go/securitycentermanagement/apiv1/securitycentermanagementpb"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
	"github.com/overmindtech/cli/sources/gcp/dynamic"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

func TestSecurityCenterManagementSecurityCenterService(t *testing.T) {
	ctx := context.Background()
	projectID := "test-project"
	location := "global"
	linker := gcpshared.NewLinker()
	serviceName := "container-threat-detection"

	service := &securitycentermanagementpb.SecurityCenterService{
		Name: fmt.Sprintf("projects/%s/locations/%s/securityCenterServices/%s", projectID, location, serviceName),
	}

	serviceName2 := "event-threat-detection"
	service2 := &securitycentermanagementpb.SecurityCenterService{
		Name: fmt.Sprintf("projects/%s/locations/%s/securityCenterServices/%s", projectID, location, serviceName2),
	}

	serviceList := &securitycentermanagementpb.ListSecurityCenterServicesResponse{
		SecurityCenterServices: []*securitycentermanagementpb.SecurityCenterService{service, service2},
	}

	sdpItemType := gcpshared.SecurityCenterManagementSecurityCenterService

	expectedCallAndResponses := map[string]shared.MockResponse{
		fmt.Sprintf("https://securitycentermanagement.googleapis.com/v1/projects/%s/locations/%s/securityCenterServices/%s", projectID, location, serviceName): {
			StatusCode: http.StatusOK,
			Body:       service,
		},
		fmt.Sprintf("https://securitycentermanagement.googleapis.com/v1/projects/%s/locations/%s/securityCenterServices/%s", projectID, location, serviceName2): {
			StatusCode: http.StatusOK,
			Body:       service2,
		},
		fmt.Sprintf("https://securitycentermanagement.googleapis.com/v1/projects/%s/locations/%s/securityCenterServices", projectID, location): {
			StatusCode: http.StatusOK,
			Body:       serviceList,
		},
	}

	t.Run("Get", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		combinedQuery := shared.CompositeLookupKey(location, serviceName)
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
				// Link to parent Project from name field
				// The name field format is: projects/{project}/locations/{location}/securityCenterServices/{service}
				// The manual linker will extract the project ID from the name field
				{
					ExpectedType:   gcpshared.CloudResourceManagerProject.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  projectID,
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
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), projectID)
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

	t.Run("ErrorHandling", func(t *testing.T) {
		errorResponses := map[string]shared.MockResponse{
			fmt.Sprintf("https://securitycentermanagement.googleapis.com/v1/projects/%s/locations/%s/securityCenterServices/%s", projectID, location, serviceName): {
				StatusCode: http.StatusNotFound,
				Body:       map[string]interface{}{"error": "Service not found"},
			},
		}

		httpCli := shared.NewMockHTTPClientProvider(errorResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		combinedQuery := shared.CompositeLookupKey(location, serviceName)
		_, err = adapter.Get(ctx, projectID, combinedQuery, true)
		if err == nil {
			t.Error("Expected error when getting non-existent resource, but got nil")
		}
	})
}
