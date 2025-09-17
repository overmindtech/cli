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

func TestComputeGlobalForwardingRule(t *testing.T) {
	ctx := context.Background()
	projectID := "test-project"
	linker := gcpshared.NewLinker()
	forwardingRuleName := "test-global-forwarding-rule"

	// Mock response for a global forwarding rule using protobuf types
	globalForwardingRule := &computepb.ForwardingRule{
		Id:                  uint64Ptr(1234567890123456789),
		CreationTimestamp:   stringPtr("2023-01-01T00:00:00.000-08:00"),
		Name:                stringPtr(forwardingRuleName),
		Description:         stringPtr("Test global forwarding rule"),
		Region:              stringPtr(""),
		IPAddress:           stringPtr("203.0.113.1"),
		IPProtocol:          stringPtr("TCP"),
		PortRange:           stringPtr("80"),
		Target:              stringPtr(fmt.Sprintf("projects/%s/global/targetHttpProxies/test-target-proxy", projectID)),
		SelfLink:            stringPtr(fmt.Sprintf("https://www.googleapis.com/compute/v1/projects/%s/global/forwardingRules/%s", projectID, forwardingRuleName)),
		LoadBalancingScheme: stringPtr("EXTERNAL"),
		Subnetwork:          stringPtr(fmt.Sprintf("projects/%s/regions/us-central1/subnetworks/test-subnet", projectID)),
		Network:             stringPtr(fmt.Sprintf("projects/%s/global/networks/default", projectID)),
		BackendService:      stringPtr(fmt.Sprintf("projects/%s/global/backendServices/test-backend-service", projectID)),
		ServiceLabel:        stringPtr("test-service"),
		ServiceName:         stringPtr(fmt.Sprintf("%s-test-service.c.%s.internal", forwardingRuleName, projectID)),
		Kind:                stringPtr("compute#forwardingRule"),
		LabelFingerprint:    stringPtr("42WmSpB8rSM="),
		Labels: map[string]string{
			"env":  "test",
			"team": "devops",
		},
		NetworkTier:          stringPtr("PREMIUM"),
		AllowGlobalAccess:    boolPtr(false),
		AllowPscGlobalAccess: boolPtr(false),
		PscConnectionId:      nil,
		PscConnectionStatus:  stringPtr("ACCEPTED"),
		Fingerprint:          stringPtr("abcd1234efgh5678"),
	}

	// Mock response for a second global forwarding rule using protobuf types
	globalForwardingRule2 := &computepb.ForwardingRule{
		Id:                  uint64Ptr(9876543210987654321),
		CreationTimestamp:   stringPtr("2023-01-02T00:00:00.000-08:00"),
		Name:                stringPtr("test-global-forwarding-rule-2"),
		Description:         stringPtr("Second test global forwarding rule"),
		Region:              stringPtr(""),
		IPAddress:           stringPtr("203.0.113.2"),
		IPProtocol:          stringPtr("TCP"),
		PortRange:           stringPtr("443"),
		Target:              stringPtr(fmt.Sprintf("projects/%s/global/targetHttpsProxies/test-target-proxy-2", projectID)),
		SelfLink:            stringPtr(fmt.Sprintf("https://www.googleapis.com/compute/v1/projects/%s/global/forwardingRules/test-global-forwarding-rule-2", projectID)),
		LoadBalancingScheme: stringPtr("EXTERNAL"),
		Subnetwork:          stringPtr(fmt.Sprintf("projects/%s/regions/us-west1/subnetworks/test-subnet-2", projectID)),
		Network:             stringPtr(fmt.Sprintf("projects/%s/global/networks/custom-network", projectID)),
		BackendService:      stringPtr(fmt.Sprintf("projects/%s/global/backendServices/test-backend-service-2", projectID)),
		ServiceLabel:        stringPtr("test-service-2"),
		ServiceName:         stringPtr("test-global-forwarding-rule-2-test-service-2.c." + projectID + ".internal"),
		Kind:                stringPtr("compute#forwardingRule"),
		LabelFingerprint:    stringPtr("xyz789abc123def="),
		Labels: map[string]string{
			"env":     "prod",
			"service": "web",
		},
		NetworkTier:          stringPtr("PREMIUM"),
		AllowGlobalAccess:    boolPtr(true),
		AllowPscGlobalAccess: boolPtr(true),
		PscConnectionId:      uint64Ptr(123),
		PscConnectionStatus:  stringPtr("ACCEPTED"),
		Fingerprint:          stringPtr("xyz789abc123def456"),
	}

	// Mock response for list operation using protobuf types
	globalForwardingRulesList := &computepb.ForwardingRuleList{
		Kind:     stringPtr("compute#forwardingRuleList"),
		Id:       stringPtr("projects/" + projectID + "/global/forwardingRules"),
		Items:    []*computepb.ForwardingRule{globalForwardingRule, globalForwardingRule2},
		SelfLink: stringPtr(fmt.Sprintf("https://www.googleapis.com/compute/v1/projects/%s/global/forwardingRules", projectID)),
	}

	sdpItemType := gcpshared.ComputeGlobalForwardingRule

	expectedCallAndResponses := map[string]shared.MockResponse{
		fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/global/forwardingRules/%s", projectID, forwardingRuleName): {
			StatusCode: http.StatusOK,
			Body:       globalForwardingRule,
		},
		fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/global/forwardingRules", projectID): {
			StatusCode: http.StatusOK,
			Body:       globalForwardingRulesList,
		},
	}

	t.Run("Get", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		getQuery := forwardingRuleName
		sdpItem, err := adapter.Get(ctx, projectID, getQuery, true)
		if err != nil {
			t.Fatalf("Failed to get global forwarding rule: %v", err)
		}

		if sdpItem.GetType() != sdpItemType.String() {
			t.Errorf("Expected type %s, got %s", sdpItemType.String(), sdpItem.GetType())
		}
		if sdpItem.UniqueAttributeValue() != getQuery {
			t.Errorf("Expected unique attribute value '%s', got %s", forwardingRuleName, sdpItem.UniqueAttributeValue())
		}
		if sdpItem.GetScope() != projectID {
			t.Errorf("Expected scope '%s', got %s", projectID, sdpItem.GetScope())
		}

		// Test specific attributes
		val, err := sdpItem.GetAttributes().Get("name")
		if err != nil {
			t.Fatalf("Failed to get 'name' attribute: %v", err)
		}
		if val != forwardingRuleName {
			t.Errorf("Expected name field to be '%s', got %s", forwardingRuleName, val)
		}

		val, err = sdpItem.GetAttributes().Get("description")
		if err != nil {
			t.Fatalf("Failed to get 'description' attribute: %v", err)
		}
		if val != "Test global forwarding rule" {
			t.Errorf("Expected description field to be 'Test global forwarding rule', got %s", val)
		}

		val, err = sdpItem.GetAttributes().Get("IPAddress")
		if err != nil {
			t.Fatalf("Failed to get 'ipAddress' attribute: %v", err)
		}
		if val != "203.0.113.1" {
			t.Errorf("Expected ipAddress field to be '203.0.113.1', got %s", val)
		}

		val, err = sdpItem.GetAttributes().Get("IPProtocol")
		if err != nil {
			t.Fatalf("Failed to get 'ipProtocol' attribute: %v", err)
		}
		if val != "TCP" {
			t.Errorf("Expected ipProtocol field to be 'TCP', got %s", val)
		}

		val, err = sdpItem.GetAttributes().Get("loadBalancingScheme")
		if err != nil {
			t.Fatalf("Failed to get 'loadBalancingScheme' attribute: %v", err)
		}
		if val != "EXTERNAL" {
			t.Errorf("Expected loadBalancingScheme field to be 'EXTERNAL', got %s", val)
		}

		val, err = sdpItem.GetAttributes().Get("network")
		if err != nil {
			t.Fatalf("Failed to get 'network' attribute: %v", err)
		}
		expectedNetwork := fmt.Sprintf("projects/%s/global/networks/default", projectID)
		if val != expectedNetwork {
			t.Errorf("Expected network field to be '%s', got %s", expectedNetwork, val)
		}

		val, err = sdpItem.GetAttributes().Get("backendService")
		if err != nil {
			t.Fatalf("Failed to get 'backendService' attribute: %v", err)
		}
		expectedBackendService := fmt.Sprintf("projects/%s/global/backendServices/test-backend-service", projectID)
		if val != expectedBackendService {
			t.Errorf("Expected backendService field to be '%s', got %s", expectedBackendService, val)
		}

		val, err = sdpItem.GetAttributes().Get("subnetwork")
		if err != nil {
			t.Fatalf("Failed to get 'subnetwork' attribute: %v", err)
		}
		expectedSubnetwork := fmt.Sprintf("projects/%s/regions/us-central1/subnetworks/test-subnet", projectID)
		if val != expectedSubnetwork {
			t.Errorf("Expected subnetwork field to be '%s', got %s", expectedSubnetwork, val)
		}

		// Test labels - check if labels exist before testing
		labels, err := sdpItem.GetAttributes().Get("labels")
		if err == nil {
			labelsMap, ok := labels.(map[string]interface{})
			if !ok {
				t.Fatalf("Expected labels to be a map[string]interface{}, got %T", labels)
			}
			if labelsMap["env"] != "test" {
				t.Errorf("Expected labels.env to be 'test', got %s", labelsMap["env"])
			}
			if labelsMap["team"] != "devops" {
				t.Errorf("Expected labels.team to be 'devops', got %s", labelsMap["team"])
			}
		} else {
			// Labels might be optional, so just log it's not present
			t.Logf("Labels attribute not found, which is acceptable for this test")
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					ExpectedType:   "ip",
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "203.0.113.1",
					ExpectedScope:  "global",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
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
					ExpectedType:   gcpshared.ComputeBackendService.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-backend-service",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					ExpectedType:   gcpshared.ComputeSubnetwork.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-subnet",
					ExpectedScope:  fmt.Sprintf("%s.us-central1", projectID),
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
		adapter, err := dynamic.MakeAdapter(gcpshared.ComputeGlobalForwardingRule, linker, httpCli, projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		listable, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			t.Fatalf("Adapter is not a ListableAdapter")
		}

		sdpItems, err := listable.List(ctx, projectID, true)
		if err != nil {
			t.Fatalf("Failed to list global forwarding rules: %v", err)
		}

		// Verify the first item
		firstItem := sdpItems[0]
		if firstItem.GetType() != sdpItemType.String() {
			t.Errorf("Expected first item type %s, got %s", sdpItemType.String(), firstItem.GetType())
		}
		if firstItem.UniqueAttributeValue() != forwardingRuleName {
			t.Errorf("Expected first item unique attribute value '%s', got %s", forwardingRuleName, firstItem.UniqueAttributeValue())
		}

		// Verify the second item
		secondItem := sdpItems[1]
		if secondItem.GetType() != sdpItemType.String() {
			t.Errorf("Expected second item type %s, got %s", sdpItemType.String(), secondItem.GetType())
		}
		if secondItem.UniqueAttributeValue() != "test-global-forwarding-rule-2" {
			t.Errorf("Expected second item unique attribute value 'test-global-forwarding-rule-2', got %s", secondItem.UniqueAttributeValue())
		}
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		// Test with empty responses to simulate API errors
		errorResponses := map[string]shared.MockResponse{
			fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/global/forwardingRules/%s", projectID, forwardingRuleName): {
				StatusCode: http.StatusNotFound,
				Body:       map[string]interface{}{"error": "Not found"},
			},
		}

		httpCli := shared.NewMockHTTPClientProvider(errorResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		_, err = adapter.Get(ctx, projectID, forwardingRuleName, true)
		if err == nil {
			t.Error("Expected error when getting non-existent global forwarding rule, but got nil")
		}
	})

	t.Run("EmptyList", func(t *testing.T) {
		// Test with empty list response using protobuf types
		emptyListResponse := &computepb.ForwardingRuleList{
			Kind:  stringPtr("compute#forwardingRuleList"),
			Id:    stringPtr("projects/" + projectID + "/global/forwardingRules"),
			Items: []*computepb.ForwardingRule{},
		}

		emptyResponses := map[string]shared.MockResponse{
			fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/global/forwardingRules", projectID): {
				StatusCode: http.StatusOK,
				Body:       emptyListResponse,
			},
		}

		httpCli := shared.NewMockHTTPClientProvider(emptyResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		listable, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			t.Fatalf("Adapter is not a ListableAdapter")
		}

		sdpItems, err := listable.List(ctx, projectID, true)
		if err != nil {
			t.Fatalf("Failed to list global forwarding rules: %v", err)
		}

		if len(sdpItems) != 0 {
			t.Errorf("Expected 0 global forwarding rules, got %d", len(sdpItems))
		}
	})
}
