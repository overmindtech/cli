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

func TestComputeGlobalAddress(t *testing.T) {
	ctx := context.Background()
	projectID := "test-project"
	linker := gcpshared.NewLinker()
	addressName := "test-global-address"
	globalAddress := &computepb.Address{
		Name:        &addressName,
		Description: stringPtr("Test global address for load balancer"),
		Address:     stringPtr("203.0.113.12"),
		AddressType: stringPtr("EXTERNAL"),
		Status:      stringPtr("RESERVED"),
		Network:     stringPtr("global/networks/test-network"),
		Labels: map[string]string{
			"env":  "test",
			"team": "networking",
		},
		Region:            stringPtr("global"),
		NetworkTier:       stringPtr("PREMIUM"),
		CreationTimestamp: stringPtr("2023-01-15T10:30:00.000-08:00"),
		Id:                uint64Ptr(1234567890123456789),
		Kind:              stringPtr("compute#globalAddress"),
		SelfLink:          stringPtr(fmt.Sprintf("https://www.googleapis.com/compute/v1/projects/%s/global/addresses/%s", projectID, addressName)),
	}

	// Create a second global address for list testing
	addressName2 := "test-global-address-2"
	globalAddress2 := &computepb.Address{
		Name:        &addressName2,
		Description: stringPtr("Second test global address"),
		Address:     stringPtr("203.0.113.13"),
		AddressType: stringPtr("EXTERNAL"),
		Status:      stringPtr("RESERVED"),
		Network:     stringPtr("global/networks/test-network-2"),
		Labels: map[string]string{
			"env":  "prod",
			"team": "networking",
		},
		Region:            stringPtr("global"),
		NetworkTier:       stringPtr("PREMIUM"),
		CreationTimestamp: stringPtr("2023-01-16T11:45:00.000-08:00"),
		Id:                uint64Ptr(1234567890123456790),
		Kind:              stringPtr("compute#globalAddress"),
		SelfLink:          stringPtr(fmt.Sprintf("https://www.googleapis.com/compute/v1/projects/%s/global/addresses/%s", projectID, addressName2)),
	}

	globalAddresses := &computepb.AddressList{
		Items: []*computepb.Address{globalAddress, globalAddress2},
	}

	sdpItemType := gcpshared.ComputeGlobalAddress

	expectedCallAndResponses := map[string]shared.MockResponse{
		fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/global/addresses/%s", projectID, addressName): {
			StatusCode: http.StatusOK,
			Body:       globalAddress,
		},
		fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/global/addresses", projectID): {
			StatusCode: http.StatusOK,
			Body:       globalAddresses,
		},
	}

	t.Run("Get", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		getQuery := addressName
		sdpItem, err := adapter.Get(ctx, projectID, getQuery, true)
		if err != nil {
			t.Fatalf("Failed to get global address: %v", err)
		}

		if sdpItem.GetType() != sdpItemType.String() {
			t.Errorf("Expected type %s, got %s", sdpItemType.String(), sdpItem.GetType())
		}
		if sdpItem.UniqueAttributeValue() != getQuery {
			t.Errorf("Expected unique attribute value '%s', got %s", addressName, sdpItem.UniqueAttributeValue())
		}
		if sdpItem.GetScope() != projectID {
			t.Errorf("Expected scope '%s', got %s", projectID, sdpItem.GetScope())
		}
		val, err := sdpItem.GetAttributes().Get("name")
		if err != nil {
			t.Fatalf("Failed to get 'name' attribute: %v", err)
		}
		if val != addressName {
			t.Errorf("Expected name field to be '%s', got %s", addressName, val)
		}
		val, err = sdpItem.GetAttributes().Get("description")
		if err != nil {
			t.Fatalf("Failed to get 'description' attribute: %v", err)
		}
		if val != "Test global address for load balancer" {
			t.Errorf("Expected description field to be 'Test global address for load balancer', got %s", val)
		}
		val, err = sdpItem.GetAttributes().Get("address")
		if err != nil {
			t.Fatalf("Failed to get 'address' attribute: %v", err)
		}
		if val != "203.0.113.12" {
			t.Errorf("Expected address field to be '203.0.113.12', got %s", val)
		}
		val, err = sdpItem.GetAttributes().Get("address_type")
		if err != nil {
			t.Fatalf("Failed to get 'address_type' attribute: %v", err)
		}
		if val != "EXTERNAL" {
			t.Errorf("Expected address_type field to be 'EXTERNAL', got %s", val)
		}
		val, err = sdpItem.GetAttributes().Get("status")
		if err != nil {
			t.Fatalf("Failed to get 'status' attribute: %v", err)
		}
		if val != "RESERVED" {
			t.Errorf("Expected status field to be 'RESERVED', got %s", val)
		}
		val, err = sdpItem.GetAttributes().Get("network")
		if err != nil {
			t.Fatalf("Failed to get 'network' attribute: %v", err)
		}
		if val != "global/networks/test-network" {
			t.Errorf("Expected network field to be 'global/networks/test-network', got %s", val)
		}
		val, err = sdpItem.GetAttributes().Get("network_tier")
		if err != nil {
			t.Fatalf("Failed to get 'network_tier' attribute: %v", err)
		}
		if val != "PREMIUM" {
			t.Errorf("Expected network_tier field to be 'PREMIUM', got %s", val)
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					ExpectedType:   gcpshared.ComputeNetwork.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-network",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					ExpectedType:   stdlib.NetworkIP.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "203.0.113.12",
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
		adapter, err := dynamic.MakeAdapter(gcpshared.ComputeGlobalAddress, linker, httpCli, projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		listable, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			t.Fatalf("Adapter is not a ListableAdapter")
		}

		sdpItems, err := listable.List(ctx, projectID, true)
		if err != nil {
			t.Fatalf("Failed to list global addresses: %v", err)
		}

		if len(sdpItems) != 2 {
			t.Errorf("Expected 2 global addresses, got %d", len(sdpItems))
		}
	})
}

// Helper functions for pointer creation
func stringPtr(s string) *string {
	return &s
}

func uint64Ptr(u uint64) *uint64 {
	return &u
}
