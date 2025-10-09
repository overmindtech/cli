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

func TestComputeRouter(t *testing.T) {
	ctx := context.Background()
	projectID := "test-project"
	region := "us-central1"
	linker := gcpshared.NewLinker()
	routerName := "test-router"

	// Create mock protobuf object
	router := &computepb.Router{
		Name:        stringPtr(routerName),
		Description: stringPtr("Test Router"),
		Network:     stringPtr(fmt.Sprintf("projects/%s/global/networks/default", projectID)),
		Region:      stringPtr(fmt.Sprintf("projects/%s/regions/%s", projectID, region)),
		Interfaces: []*computepb.RouterInterface{
			{
				Name:                         stringPtr("interface-1"),
				LinkedInterconnectAttachment: stringPtr(fmt.Sprintf("projects/%s/regions/%s/interconnectAttachments/test-attachment", projectID, region)),
				PrivateIpAddress:             stringPtr("10.0.0.1"),
				Subnetwork:                   stringPtr(fmt.Sprintf("projects/%s/regions/%s/subnetworks/test-subnet", projectID, region)),
				LinkedVpnTunnel:              stringPtr(fmt.Sprintf("projects/%s/regions/%s/vpnTunnels/test-tunnel", projectID, region)),
			},
		},
		BgpPeers: []*computepb.RouterBgpPeer{
			{
				Name:                   stringPtr("bgp-peer-1"),
				PeerIpAddress:          stringPtr("192.168.1.1"),
				IpAddress:              stringPtr("192.168.1.2"),
				Ipv4NexthopAddress:     stringPtr("192.168.1.3"),
				PeerIpv4NexthopAddress: stringPtr("192.168.1.4"),
			},
		},
		Nats: []*computepb.RouterNat{
			{
				Name:        stringPtr("nat-1"),
				NatIps:      []string{"203.0.113.1", "203.0.113.2"},
				DrainNatIps: []string{"203.0.113.3"},
				Subnetworks: []*computepb.RouterNatSubnetworkToNat{
					{
						Name: stringPtr(fmt.Sprintf("projects/%s/regions/%s/subnetworks/nat-subnet", projectID, region)),
					},
				},
				Nat64Subnetworks: []*computepb.RouterNatSubnetworkToNat64{
					{
						Name: stringPtr(fmt.Sprintf("projects/%s/regions/%s/subnetworks/nat64-subnet", projectID, region)),
					},
				},
			},
		},
	}

	// Create second router for list testing
	routerName2 := "test-router-2"
	router2 := &computepb.Router{
		Name:        stringPtr(routerName2),
		Description: stringPtr("Test Router 2"),
		Network:     stringPtr(fmt.Sprintf("projects/%s/global/networks/default", projectID)),
		Region:      stringPtr(fmt.Sprintf("projects/%s/regions/%s", projectID, region)),
	}

	// Create list response with multiple items
	routerList := &computepb.RouterList{
		Items: []*computepb.Router{router, router2},
	}

	sdpItemType := gcpshared.ComputeRouter

	// Mock HTTP responses
	expectedCallAndResponses := map[string]shared.MockResponse{
		fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/regions/%s/routers/%s", projectID, region, routerName): {
			StatusCode: http.StatusOK,
			Body:       router,
		},
		fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/regions/%s/routers/%s", projectID, region, routerName2): {
			StatusCode: http.StatusOK,
			Body:       router2,
		},
		fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/regions/%s/routers", projectID, region): {
			StatusCode: http.StatusOK,
			Body:       routerList,
		},
	}

	t.Run("Get", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, projectID, region)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		sdpItem, err := adapter.Get(ctx, fmt.Sprintf("%s.%s", projectID, region), routerName, true)
		if err != nil {
			t.Fatalf("Failed to get resource: %v", err)
		}

		// Validate SDP item properties
		if sdpItem.GetType() != sdpItemType.String() {
			t.Errorf("Expected type %s, got %s", sdpItemType.String(), sdpItem.GetType())
		}
		if sdpItem.UniqueAttributeValue() != routerName {
			t.Errorf("Expected unique attribute value '%s', got %s", routerName, sdpItem.UniqueAttributeValue())
		}
		expectedScope := fmt.Sprintf("%s.%s", projectID, region)
		if sdpItem.GetScope() != expectedScope {
			t.Errorf("Expected scope '%s', got %s", expectedScope, sdpItem.GetScope())
		}

		// Validate specific attributes
		val, err := sdpItem.GetAttributes().Get("name")
		if err != nil {
			t.Fatalf("Failed to get 'name' attribute: %v", err)
		}
		if val != routerName {
			t.Errorf("Expected name field to be '%s', got %s", routerName, val)
		}

		// Include static tests - covers ALL blast propagation links
		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				// Network link
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
				// Interface private IP address
				{
					ExpectedType:   stdlib.NetworkIP.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "10.0.0.1",
					ExpectedScope:  "global",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				// Interface subnetwork
				{
					ExpectedType:   gcpshared.ComputeSubnetwork.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-subnet",
					ExpectedScope:  fmt.Sprintf("%s.%s", projectID, region),
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// VPN tunnel link
				{
					ExpectedType:   gcpshared.ComputeVpnTunnel.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-tunnel",
					ExpectedScope:  fmt.Sprintf("%s.%s", projectID, region),
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				// BGP peer IP addresses
				{
					ExpectedType:   stdlib.NetworkIP.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "192.168.1.1",
					ExpectedScope:  "global",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					ExpectedType:   stdlib.NetworkIP.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "192.168.1.2",
					ExpectedScope:  "global",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					ExpectedType:   stdlib.NetworkIP.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "192.168.1.3",
					ExpectedScope:  "global",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					ExpectedType:   stdlib.NetworkIP.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "192.168.1.4",
					ExpectedScope:  "global",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				// NAT IP addresses
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
					ExpectedType:   stdlib.NetworkIP.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "203.0.113.2",
					ExpectedScope:  "global",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					ExpectedType:   stdlib.NetworkIP.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "203.0.113.3",
					ExpectedScope:  "global",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				// NAT subnetworks
				{
					ExpectedType:   gcpshared.ComputeSubnetwork.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "nat-subnet",
					ExpectedScope:  fmt.Sprintf("%s.%s", projectID, region),
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					ExpectedType:   gcpshared.ComputeSubnetwork.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "nat64-subnet",
					ExpectedScope:  fmt.Sprintf("%s.%s", projectID, region),
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
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, projectID, region)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		listable, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			t.Skipf("Adapter for %s does not implement ListableAdapter", sdpItemType)
		}

		expectedScope := fmt.Sprintf("%s.%s", projectID, region)
		sdpItems, err := listable.List(ctx, expectedScope, true)
		if err != nil {
			t.Fatalf("Failed to list resources: %v", err)
		}

		if len(sdpItems) != 2 {
			t.Errorf("Expected 2 resources, got %d", len(sdpItems))
		}

		firstItem := sdpItems[0]
		if firstItem.GetType() != sdpItemType.String() {
			t.Errorf("Expected first item type %s, got %s", sdpItemType.String(), firstItem.GetType())
		}
		if firstItem.GetScope() != expectedScope {
			t.Errorf("Expected first item scope '%s', got %s", expectedScope, firstItem.GetScope())
		}

	})

	t.Run("Search with Terraform format", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, projectID, region)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Skipf("Adapter for %s does not implement SearchableAdapter", sdpItemType)
		}

		// Test Terraform format: projects/[project]/regions/[region]/routers/[router]
		terraformQuery := fmt.Sprintf("projects/%s/regions/%s/routers/%s", projectID, region, routerName)
		expectedScope := fmt.Sprintf("%s.%s", projectID, region)
		sdpItems, err := searchable.Search(ctx, expectedScope, terraformQuery, true)
		if err != nil {
			t.Fatalf("Failed to search resources with Terraform format: %v", err)
		}

		// The search should return only the specific resource matching the Terraform format
		if len(sdpItems) != 1 {
			t.Errorf("Expected 1 resource, got %d", len(sdpItems))
			return
		}

		// Verify the single item returned
		firstItem := sdpItems[0]
		if firstItem.GetType() != sdpItemType.String() {
			t.Errorf("Expected first item type %s, got %s", sdpItemType.String(), firstItem.GetType())
		}
		if firstItem.GetScope() != expectedScope {
			t.Errorf("Expected first item scope '%s', got %s", expectedScope, firstItem.GetScope())
		}
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		// Test with error responses to simulate API errors
		errorResponses := map[string]shared.MockResponse{
			fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/regions/%s/routers/%s", projectID, region, routerName): {
				StatusCode: http.StatusNotFound,
				Body:       map[string]interface{}{"error": "Router not found"},
			},
		}

		httpCli := shared.NewMockHTTPClientProvider(errorResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, projectID, region)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		expectedScope := fmt.Sprintf("%s.%s", projectID, region)
		_, err = adapter.Get(ctx, expectedScope, routerName, true)
		if err == nil {
			t.Error("Expected error when getting non-existent resource, but got nil")
		}
	})
}
