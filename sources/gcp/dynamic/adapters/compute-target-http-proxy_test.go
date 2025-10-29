package adapters_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"cloud.google.com/go/compute/apiv1/computepb"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources/gcp/dynamic"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

func TestComputeTargetHttpProxy(t *testing.T) {
	ctx := context.Background()
	projectID := "test-project"
	linker := gcpshared.NewLinker()
	proxyName := "test-http-proxy"

	urlMapURL := fmt.Sprintf("projects/%s/global/urlMaps/test-url-map", projectID)
	proxy := &computepb.TargetHttpProxy{
		Name:   &proxyName,
		UrlMap: &urlMapURL,
	}

	proxyName2 := "test-http-proxy-2"
	proxy2 := &computepb.TargetHttpProxy{
		Name: &proxyName2,
	}

	proxyList := &computepb.TargetHttpProxyList{
		Items: []*computepb.TargetHttpProxy{proxy, proxy2},
	}

	sdpItemType := gcpshared.ComputeTargetHttpProxy

	expectedCallAndResponses := map[string]shared.MockResponse{
		fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/global/targetHttpProxies/%s", projectID, proxyName): {
			StatusCode: http.StatusOK,
			Body:       proxy,
		},
		fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/global/targetHttpProxies/%s", projectID, proxyName2): {
			StatusCode: http.StatusOK,
			Body:       proxy2,
		},
		fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/global/targetHttpProxies", projectID): {
			StatusCode: http.StatusOK,
			Body:       proxyList,
		},
	}

	t.Run("Get", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		sdpItem, err := adapter.Get(ctx, projectID, proxyName, true)
		if err != nil {
			t.Fatalf("Failed to get resource: %v", err)
		}

		if sdpItem.GetType() != sdpItemType.String() {
			t.Errorf("Expected type %s, got %s", sdpItemType.String(), sdpItem.GetType())
		}
		if sdpItem.UniqueAttributeValue() != proxyName {
			t.Errorf("Expected unique attribute value '%s', got %s", proxyName, sdpItem.UniqueAttributeValue())
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					ExpectedType:   gcpshared.ComputeUrlMap.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-url-map",
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
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		listable, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			t.Skipf("Adapter for %s does not implement ListableAdapter", sdpItemType)
		}

		sdpItems, err := listable.List(ctx, projectID, true)
		if err != nil {
			t.Fatalf("Failed to list resources: %v", err)
		}

		if len(sdpItems) != 2 {
			t.Errorf("Expected 2 resources, got %d", len(sdpItems))
		}
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		errorResponses := map[string]shared.MockResponse{
			fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/global/targetHttpProxies/%s", projectID, proxyName): {
				StatusCode: http.StatusNotFound,
				Body:       map[string]interface{}{"error": "Proxy not found"},
			},
		}

		httpCli := shared.NewMockHTTPClientProvider(errorResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		_, err = adapter.Get(ctx, projectID, proxyName, true)
		if err == nil {
			t.Error("Expected error when getting non-existent resource, but got nil")
		}
	})
}
