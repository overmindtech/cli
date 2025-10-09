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
	"github.com/overmindtech/cli/sources/stdlib"
)

func TestComputeExternalVpnGateway(t *testing.T) {
	ctx := context.Background()
	projectID := "test-project"
	linker := gcpshared.NewLinker()
	gatewayName := "test-external-vpn-gateway"

	ipAddress := "203.0.113.1"
	gateway := &computepb.ExternalVpnGateway{
		Name: &gatewayName,
		Interfaces: []*computepb.ExternalVpnGatewayInterface{
			{
				IpAddress: &ipAddress,
			},
		},
	}

	gatewayName2 := "test-external-vpn-gateway-2"
	gateway2 := &computepb.ExternalVpnGateway{
		Name: &gatewayName2,
	}

	gatewayList := &computepb.ExternalVpnGatewayList{
		Items: []*computepb.ExternalVpnGateway{gateway, gateway2},
	}

	sdpItemType := gcpshared.ComputeExternalVpnGateway

	expectedCallAndResponses := map[string]shared.MockResponse{
		fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/global/externalVpnGateways/%s", projectID, gatewayName): {
			StatusCode: http.StatusOK,
			Body:       gateway,
		},
		fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/global/externalVpnGateways/%s", projectID, gatewayName2): {
			StatusCode: http.StatusOK,
			Body:       gateway2,
		},
		fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/global/externalVpnGateways", projectID): {
			StatusCode: http.StatusOK,
			Body:       gatewayList,
		},
	}

	t.Run("Get", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		sdpItem, err := adapter.Get(ctx, projectID, gatewayName, true)
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
					ExpectedType:   stdlib.NetworkIP.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "203.0.113.1",
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
			fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/global/externalVpnGateways/%s", projectID, gatewayName): {
				StatusCode: http.StatusNotFound,
				Body:       map[string]interface{}{"error": "Gateway not found"},
			},
		}

		httpCli := shared.NewMockHTTPClientProvider(errorResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		_, err = adapter.Get(ctx, projectID, gatewayName, true)
		if err == nil {
			t.Error("Expected error when getting non-existent resource, but got nil")
		}
	})
}
