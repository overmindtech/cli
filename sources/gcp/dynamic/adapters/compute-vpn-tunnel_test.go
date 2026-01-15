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
	"github.com/overmindtech/cli/sources/stdlib"
)

func TestComputeVpnTunnel(t *testing.T) {
	ctx := context.Background()
	projectID := "test-project"
	region := "us-central1"
	linker := gcpshared.NewLinker()
	tunnelName := "test-vpn-tunnel"

	peerIP := "203.0.113.1"
	targetVpnGatewayURL := fmt.Sprintf("projects/%s/regions/%s/targetVpnGateways/test-target-gateway", projectID, region)
	vpnGatewayURL := fmt.Sprintf("projects/%s/regions/%s/vpnGateways/test-gateway", projectID, region)
	peerExternalGatewayURL := fmt.Sprintf("projects/%s/global/externalVpnGateways/test-external-gateway", projectID)
	peerGcpGatewayURL := fmt.Sprintf("projects/%s/regions/%s/vpnGateways/test-peer-gcp-gateway", projectID, region)
	routerURL := fmt.Sprintf("projects/%s/regions/%s/routers/test-router", projectID, region)
	tunnel := &computepb.VpnTunnel{
		Name:                &tunnelName,
		PeerIp:              &peerIP,
		TargetVpnGateway:    &targetVpnGatewayURL,
		VpnGateway:          &vpnGatewayURL,
		PeerExternalGateway: &peerExternalGatewayURL,
		PeerGcpGateway:      &peerGcpGatewayURL,
		Router:              &routerURL,
	}

	tunnelName2 := "test-vpn-tunnel-2"
	tunnel2 := &computepb.VpnTunnel{
		Name: &tunnelName2,
	}

	tunnelList := &computepb.VpnTunnelList{
		Items: []*computepb.VpnTunnel{tunnel, tunnel2},
	}

	sdpItemType := gcpshared.ComputeVpnTunnel

	expectedCallAndResponses := map[string]shared.MockResponse{
		fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/regions/%s/vpnTunnels/%s", projectID, region, tunnelName): {
			StatusCode: http.StatusOK,
			Body:       tunnel,
		},
		fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/regions/%s/vpnTunnels/%s", projectID, region, tunnelName2): {
			StatusCode: http.StatusOK,
			Body:       tunnel2,
		},
		fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/regions/%s/vpnTunnels", projectID, region): {
			StatusCode: http.StatusOK,
			Body:       tunnelList,
		},
	}

	t.Run("Get", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), projectID, region)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		sdpItem, err := adapter.Get(ctx, fmt.Sprintf("%s.%s", projectID, region), tunnelName, true)
		if err != nil {
			t.Fatalf("Failed to get resource: %v", err)
		}

		if sdpItem.GetType() != sdpItemType.String() {
			t.Errorf("Expected type %s, got %s", sdpItemType.String(), sdpItem.GetType())
		}
		if sdpItem.UniqueAttributeValue() != tunnelName {
			t.Errorf("Expected unique attribute value '%s', got %s", tunnelName, sdpItem.UniqueAttributeValue())
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				// Peer IP link
				{
					ExpectedType:   stdlib.NetworkIP.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "203.0.113.1",
					ExpectedScope:  "global",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				// Target VPN Gateway link (Classic VPN)
				{
					ExpectedType:   gcpshared.ComputeTargetVpnGateway.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-target-gateway",
					ExpectedScope:  fmt.Sprintf("%s.%s", projectID, region),
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				// VPN Gateway link (HA VPN)
				{
					ExpectedType:   gcpshared.ComputeVpnGateway.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-gateway",
					ExpectedScope:  fmt.Sprintf("%s.%s", projectID, region),
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				// Peer External Gateway link
				{
					ExpectedType:   gcpshared.ComputeExternalVpnGateway.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-external-gateway",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// Peer GCP Gateway link
				{
					ExpectedType:   gcpshared.ComputeVpnGateway.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-peer-gcp-gateway",
					ExpectedScope:  fmt.Sprintf("%s.%s", projectID, region),
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// Router link
				{
					ExpectedType:   gcpshared.ComputeRouter.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-router",
					ExpectedScope:  fmt.Sprintf("%s.%s", projectID, region),
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
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), projectID, region)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		listable, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			t.Skipf("Adapter for %s does not implement ListableAdapter", sdpItemType)
		}

		sdpItems, err := listable.List(ctx, fmt.Sprintf("%s.%s", projectID, region), true)
		if err != nil {
			t.Fatalf("Failed to list resources: %v", err)
		}

		if len(sdpItems) != 2 {
			t.Errorf("Expected 2 resources, got %d", len(sdpItems))
		}
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		errorResponses := map[string]shared.MockResponse{
			fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/regions/%s/vpnTunnels/%s", projectID, region, tunnelName): {
				StatusCode: http.StatusNotFound,
				Body:       map[string]interface{}{"error": "VPN tunnel not found"},
			},
		}

		httpCli := shared.NewMockHTTPClientProvider(errorResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), projectID, region)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		_, err = adapter.Get(ctx, fmt.Sprintf("%s.%s", projectID, region), tunnelName, true)
		if err == nil {
			t.Error("Expected error when getting non-existent resource, but got nil")
		}
	})
}
