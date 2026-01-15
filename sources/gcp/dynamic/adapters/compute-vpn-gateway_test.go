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

func TestComputeVpnGateway(t *testing.T) {
	ctx := context.Background()
	projectID := "test-project"
	region := "us-central1"
	linker := gcpshared.NewLinker()
	gatewayName := "test-vpn-gateway"

	networkURL := fmt.Sprintf("projects/%s/global/networks/default", projectID)
	ipAddress := "203.0.113.1"
	interconnectAttachmentURL := fmt.Sprintf("projects/%s/regions/%s/interconnectAttachments/test-attachment", projectID, region)
	gateway := &computepb.VpnGateway{
		Name:    &gatewayName,
		Network: &networkURL,
		VpnInterfaces: []*computepb.VpnGatewayVpnGatewayInterface{
			{
				IpAddress:              &ipAddress,
				InterconnectAttachment: &interconnectAttachmentURL,
			},
		},
	}

	gatewayName2 := "test-vpn-gateway-2"
	gateway2 := &computepb.VpnGateway{
		Name: &gatewayName2,
	}

	gatewayList := &computepb.VpnGatewayList{
		Items: []*computepb.VpnGateway{gateway, gateway2},
	}

	sdpItemType := gcpshared.ComputeVpnGateway

	expectedCallAndResponses := map[string]shared.MockResponse{
		fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/regions/%s/vpnGateways/%s", projectID, region, gatewayName): {
			StatusCode: http.StatusOK,
			Body:       gateway,
		},
		fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/regions/%s/vpnGateways/%s", projectID, region, gatewayName2): {
			StatusCode: http.StatusOK,
			Body:       gateway2,
		},
		fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/regions/%s/vpnGateways", projectID, region): {
			StatusCode: http.StatusOK,
			Body:       gatewayList,
		},
	}

	t.Run("Get", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), projectID, region)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		sdpItem, err := adapter.Get(ctx, fmt.Sprintf("%s.%s", projectID, region), gatewayName, true)
		if err != nil {
			t.Fatalf("Failed to get resource: %v", err)
		}

		if sdpItem.GetType() != sdpItemType.String() {
			t.Errorf("Expected type %s, got %s", sdpItemType.String(), sdpItem.GetType())
		}
		if sdpItem.UniqueAttributeValue() != gatewayName {
			t.Errorf("Expected unique attribute value '%s', got %s", gatewayName, sdpItem.UniqueAttributeValue())
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					ExpectedType:   gcpshared.ComputeNetwork.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "default",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
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
				{
					ExpectedType:   gcpshared.ComputeInterconnectAttachment.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-attachment",
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
			fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/regions/%s/vpnGateways/%s", projectID, region, gatewayName): {
				StatusCode: http.StatusNotFound,
				Body:       map[string]interface{}{"error": "VPN gateway not found"},
			},
		}

		httpCli := shared.NewMockHTTPClientProvider(errorResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), projectID, region)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		_, err = adapter.Get(ctx, fmt.Sprintf("%s.%s", projectID, region), gatewayName, true)
		if err == nil {
			t.Error("Expected error when getting non-existent resource, but got nil")
		}
	})
}
