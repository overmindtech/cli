package adapters_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"google.golang.org/api/compute/v1"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
	"github.com/overmindtech/cli/sources/gcp/dynamic"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

func TestComputeRoute(t *testing.T) {
	ctx := context.Background()
	projectID := "test-project"
	linker := gcpshared.NewLinker()
	routeName := "test-route"

	route := &compute.Route{
		Name:             routeName,
		Network:          "https://www.googleapis.com/compute/v1/projects/test-project/global/networks/default",
		NextHopNetwork:   "https://www.googleapis.com/compute/v1/projects/test-project/global/networks/peer-network",
		NextHopIp:        "10.0.0.1",
		NextHopInstance:  "https://www.googleapis.com/compute/v1/projects/test-project/zones/us-central1-a/instances/test-instance",
		NextHopVpnTunnel: "https://www.googleapis.com/compute/v1/projects/test-project/regions/us-central1/vpnTunnels/test-tunnel",
	}

	routeList := &compute.RouteList{
		Items: []*compute.Route{route},
	}

	sdpItemType := gcpshared.ComputeRoute

	expectedCallAndResponses := map[string]shared.MockResponse{
		fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/global/routes/%s", projectID, routeName): {
			StatusCode: http.StatusOK,
			Body:       route,
		},
		fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/global/routes", projectID): {
			StatusCode: http.StatusOK,
			Body:       routeList,
		},
	}

	t.Run("Get", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		sdpItem, err := adapter.Get(ctx, projectID, routeName, true)
		if err != nil {
			t.Fatalf("Failed to get route: %v", err)
		}

		if sdpItem.GetType() != sdpItemType.String() {
			t.Errorf("Expected type %s, got %s", sdpItemType.String(), sdpItem.GetType())
		}
		if sdpItem.UniqueAttributeValue() != routeName {
			t.Errorf("Expected unique attribute value '%s', got %s", routeName, sdpItem.UniqueAttributeValue())
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					// network
					ExpectedType:   gcpshared.ComputeNetwork.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "default",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					// nextHopNetwork
					ExpectedType:   gcpshared.ComputeNetwork.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "peer-network",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					// nextHopIp
					ExpectedType:   stdlib.NetworkIP.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "10.0.0.1",
					ExpectedScope:  "global",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					// nextHopVpnTunnel
					ExpectedType:   gcpshared.ComputeVpnTunnel.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-tunnel",
					ExpectedScope:  "test-project.us-central1",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					// nextHopInstance
					ExpectedType:   gcpshared.ComputeInstance.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-instance",
					ExpectedScope:  "test-project.us-central1-a",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// TODO: Add test for nextHopGateway → ComputeGateway
				// Requires ComputeGateway adapter to be implemented first
				// TODO: Add test for nextHopHub → NetworkConnectivityHub
				// Requires NetworkConnectivityHub adapter to be implemented first
			}

			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
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
			t.Fatalf("Failed to list routes: %v", err)
		}

		if len(sdpItems) != 1 {
			t.Errorf("Expected 1 route, got %d", len(sdpItems))
		}
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		errorResponses := map[string]shared.MockResponse{
			fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/global/routes/%s", projectID, routeName): {
				StatusCode: http.StatusNotFound,
				Body:       map[string]interface{}{"error": "Route not found"},
			},
		}

		httpCli := shared.NewMockHTTPClientProvider(errorResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		_, err = adapter.Get(ctx, projectID, routeName, true)
		if err == nil {
			t.Error("Expected error when getting non-existent route, but got nil")
		}
	})
}
