package adapters_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"cloud.google.com/go/compute/apiv1/computepb"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
	"github.com/overmindtech/cli/sources/gcp/dynamic"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

func TestComputeUrlMap(t *testing.T) {
	ctx := context.Background()
	projectID := "test-project"
	linker := gcpshared.NewLinker()
	urlMapName := "test-url-map"

	defaultService := fmt.Sprintf("projects/%s/global/backendServices/test-backend", projectID)
	pathMatcherDefaultService := fmt.Sprintf("projects/%s/global/backendServices/test-path-matcher-backend", projectID)
	pathRuleService := fmt.Sprintf("projects/%s/global/backendServices/test-path-rule-backend", projectID)
	weightedBackendService := fmt.Sprintf("projects/%s/global/backendServices/test-weighted-backend", projectID)
	mirrorBackendService := fmt.Sprintf("projects/%s/global/backendServices/test-mirror-backend", projectID)
	routeRuleService := fmt.Sprintf("projects/%s/global/backendServices/test-route-rule-backend", projectID)
	pathMatcherWeightedBackend := fmt.Sprintf("projects/%s/global/backendServices/test-pm-weighted-backend", projectID)
	pathMatcherMirrorBackend := fmt.Sprintf("projects/%s/global/backendServices/test-pm-mirror-backend", projectID)
	pathRuleWeightedBackend := fmt.Sprintf("projects/%s/global/backendServices/test-pr-weighted-backend", projectID)
	pathRuleMirrorBackend := fmt.Sprintf("projects/%s/global/backendServices/test-pr-mirror-backend", projectID)
	routeRuleWeightedBackend := fmt.Sprintf("projects/%s/global/backendServices/test-rr-weighted-backend", projectID)
	routeRuleMirrorBackend := fmt.Sprintf("projects/%s/global/backendServices/test-rr-mirror-backend", projectID)
	pathMatcherName := "path-matcher-1"
	pathPattern := "/api/*"
	priority := int32(100)
	weight := uint32(100)

	urlMap := &computepb.UrlMap{
		Name:           &urlMapName,
		DefaultService: &defaultService,
		DefaultRouteAction: &computepb.HttpRouteAction{
			WeightedBackendServices: []*computepb.WeightedBackendService{
				{
					BackendService: &weightedBackendService,
					Weight:         &weight,
				},
			},
			RequestMirrorPolicy: &computepb.RequestMirrorPolicy{
				BackendService: &mirrorBackendService,
			},
		},
		PathMatchers: []*computepb.PathMatcher{
			{
				Name:           &pathMatcherName,
				DefaultService: &pathMatcherDefaultService,
				DefaultRouteAction: &computepb.HttpRouteAction{
					WeightedBackendServices: []*computepb.WeightedBackendService{
						{
							BackendService: &pathMatcherWeightedBackend,
							Weight:         &weight,
						},
					},
					RequestMirrorPolicy: &computepb.RequestMirrorPolicy{
						BackendService: &pathMatcherMirrorBackend,
					},
				},
				PathRules: []*computepb.PathRule{
					{
						Paths:   []string{pathPattern},
						Service: &pathRuleService,
						RouteAction: &computepb.HttpRouteAction{
							WeightedBackendServices: []*computepb.WeightedBackendService{
								{
									BackendService: &pathRuleWeightedBackend,
									Weight:         &weight,
								},
							},
							RequestMirrorPolicy: &computepb.RequestMirrorPolicy{
								BackendService: &pathRuleMirrorBackend,
							},
						},
					},
				},
				RouteRules: []*computepb.HttpRouteRule{
					{
						Priority: &priority,
						Service:  &routeRuleService,
						RouteAction: &computepb.HttpRouteAction{
							WeightedBackendServices: []*computepb.WeightedBackendService{
								{
									BackendService: &routeRuleWeightedBackend,
									Weight:         &weight,
								},
							},
							RequestMirrorPolicy: &computepb.RequestMirrorPolicy{
								BackendService: &routeRuleMirrorBackend,
							},
						},
					},
				},
			},
		},
	}

	urlMapName2 := "test-url-map-2"
	urlMap2 := &computepb.UrlMap{
		Name: &urlMapName2,
	}

	urlMapList := &computepb.UrlMapList{
		Items: []*computepb.UrlMap{urlMap, urlMap2},
	}

	sdpItemType := gcpshared.ComputeUrlMap

	expectedCallAndResponses := map[string]shared.MockResponse{
		fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/global/urlMaps/%s", projectID, urlMapName): {
			StatusCode: http.StatusOK,
			Body:       urlMap,
		},
		fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/global/urlMaps/%s", projectID, urlMapName2): {
			StatusCode: http.StatusOK,
			Body:       urlMap2,
		},
		fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/global/urlMaps", projectID): {
			StatusCode: http.StatusOK,
			Body:       urlMapList,
		},
	}

	t.Run("Get", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		sdpItem, err := adapter.Get(ctx, projectID, urlMapName, true)
		if err != nil {
			t.Fatalf("Failed to get resource: %v", err)
		}

		if sdpItem.GetType() != sdpItemType.String() {
			t.Errorf("Expected type %s, got %s", sdpItemType.String(), sdpItem.GetType())
		}
		if sdpItem.UniqueAttributeValue() != urlMapName {
			t.Errorf("Expected unique attribute value '%s', got %s", urlMapName, sdpItem.UniqueAttributeValue())
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				// Default service link
				{
					ExpectedType:   gcpshared.ComputeBackendService.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-backend",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// Path matcher default service link
				{
					ExpectedType:   gcpshared.ComputeBackendService.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-path-matcher-backend",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// Path rule service link
				{
					ExpectedType:   gcpshared.ComputeBackendService.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-path-rule-backend",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// Default route action weighted backend service link
				{
					ExpectedType:   gcpshared.ComputeBackendService.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-weighted-backend",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// Default route action request mirror backend service link
				{
					ExpectedType:   gcpshared.ComputeBackendService.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-mirror-backend",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// Path matcher default route action weighted backend service link
				{
					ExpectedType:   gcpshared.ComputeBackendService.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-pm-weighted-backend",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// Path matcher default route action request mirror backend service link
				{
					ExpectedType:   gcpshared.ComputeBackendService.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-pm-mirror-backend",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// Path rule route action weighted backend service link
				{
					ExpectedType:   gcpshared.ComputeBackendService.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-pr-weighted-backend",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// Path rule route action request mirror backend service link
				{
					ExpectedType:   gcpshared.ComputeBackendService.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-pr-mirror-backend",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// Route rule service link
				{
					ExpectedType:   gcpshared.ComputeBackendService.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-route-rule-backend",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// Route rule route action weighted backend service link
				{
					ExpectedType:   gcpshared.ComputeBackendService.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-rr-weighted-backend",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// Route rule route action request mirror backend service link
				{
					ExpectedType:   gcpshared.ComputeBackendService.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-rr-mirror-backend",
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
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), projectID)
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
			fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/global/urlMaps/%s", projectID, urlMapName): {
				StatusCode: http.StatusNotFound,
				Body:       map[string]interface{}{"error": "URL map not found"},
			},
		}

		httpCli := shared.NewMockHTTPClientProvider(errorResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		_, err = adapter.Get(ctx, projectID, urlMapName, true)
		if err == nil {
			t.Error("Expected error when getting non-existent resource, but got nil")
		}
	})
}
