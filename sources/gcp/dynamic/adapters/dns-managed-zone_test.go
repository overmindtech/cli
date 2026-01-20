package adapters_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"google.golang.org/api/dns/v1"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
	"github.com/overmindtech/cli/sources/gcp/dynamic"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

func TestDNSManagedZone(t *testing.T) {
	ctx := context.Background()
	projectID := "test-project"
	linker := gcpshared.NewLinker()
	zoneName := "test-zone"

	managedZone := &dns.ManagedZone{
		Name:    zoneName,
		DnsName: "example.com.",
		PrivateVisibilityConfig: &dns.ManagedZonePrivateVisibilityConfig{
			Networks: []*dns.ManagedZonePrivateVisibilityConfigNetwork{
				{
					NetworkUrl: "https://www.googleapis.com/compute/v1/projects/test-project/global/networks/default",
				},
			},
		},
		ForwardingConfig: &dns.ManagedZoneForwardingConfig{
			TargetNameServers: []*dns.ManagedZoneForwardingConfigNameServerTarget{
				{
					Ipv4Address: "10.0.0.10",
				},
				{
					Ipv6Address: "2001:db8::1",
				},
			},
		},
		PeeringConfig: &dns.ManagedZonePeeringConfig{
			TargetNetwork: &dns.ManagedZonePeeringConfigTargetNetwork{
				NetworkUrl: "https://www.googleapis.com/compute/v1/projects/test-project/global/networks/peering-network",
			},
		},
	}

	zoneList := &dns.ManagedZonesListResponse{
		ManagedZones: []*dns.ManagedZone{managedZone},
	}

	sdpItemType := gcpshared.DNSManagedZone

	expectedCallAndResponses := map[string]shared.MockResponse{
		fmt.Sprintf("https://dns.googleapis.com/dns/v1/projects/%s/managedZones/%s", projectID, zoneName): {
			StatusCode: http.StatusOK,
			Body:       managedZone,
		},
		fmt.Sprintf("https://dns.googleapis.com/dns/v1/projects/%s/managedZones", projectID): {
			StatusCode: http.StatusOK,
			Body:       zoneList,
		},
	}

	t.Run("Get", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		sdpItem, err := adapter.Get(ctx, projectID, zoneName, true)
		if err != nil {
			t.Fatalf("Failed to get DNS managed zone: %v", err)
		}

		if sdpItem.GetType() != sdpItemType.String() {
			t.Errorf("Expected type %s, got %s", sdpItemType.String(), sdpItem.GetType())
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					// dnsName
					ExpectedType:   stdlib.NetworkDNS.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  "example.com.",
					ExpectedScope:  "global",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					// privateVisibilityConfig.networks.networkUrl
					ExpectedType:   gcpshared.ComputeNetwork.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "default",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// TODO: Add test for privateVisibilityConfig.gkeClusters.gkeClusterName → ContainerCluster
				// Requires adapter to define BlastPropagation (currently only has ToSDPItemType)
				{
					// forwardingConfig.targetNameServers.ipv4Address
					ExpectedType:   stdlib.NetworkIP.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "10.0.0.10",
					ExpectedScope:  "global",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					// forwardingConfig.targetNameServers.ipv6Address
					ExpectedType:   stdlib.NetworkIP.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "2001:db8::1",
					ExpectedScope:  "global",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					// peeringConfig.targetNetwork.networkUrl
					ExpectedType:   gcpshared.ComputeNetwork.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "peering-network",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// TODO: Add test for serviceDirectoryConfig.namespace.namespaceUrl → ServiceDirectoryNamespace
				// Requires ServiceDirectoryNamespace adapter to be implemented first
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
			t.Fatalf("Failed to list DNS managed zones: %v", err)
		}

		if len(sdpItems) != 1 {
			t.Errorf("Expected 1 DNS managed zone, got %d", len(sdpItems))
		}
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		errorResponses := map[string]shared.MockResponse{
			fmt.Sprintf("https://dns.googleapis.com/dns/v1/projects/%s/managedZones/%s", projectID, zoneName): {
				StatusCode: http.StatusNotFound,
				Body:       map[string]interface{}{"error": "Managed zone not found"},
			},
		}

		httpCli := shared.NewMockHTTPClientProvider(errorResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		_, err = adapter.Get(ctx, projectID, zoneName, true)
		if err == nil {
			t.Error("Expected error when getting non-existent DNS managed zone, but got nil")
		}
	})
}
