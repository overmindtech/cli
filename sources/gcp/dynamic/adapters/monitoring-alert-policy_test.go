package adapters_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"cloud.google.com/go/monitoring/apiv3/v2/monitoringpb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources/gcp/dynamic"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

func TestMonitoringAlertPolicy(t *testing.T) {
	ctx := context.Background()
	projectID := "test-project"
	linker := gcpshared.NewLinker()
	policyID := "test-alert-policy-123"

	// Create mock protobuf object
	alertPolicy := &monitoringpb.AlertPolicy{
		Name:        fmt.Sprintf("projects/%s/alertPolicies/%s", projectID, policyID),
		DisplayName: "Test Alert Policy",
		Documentation: &monitoringpb.AlertPolicy_Documentation{
			Content: "Test alert policy for monitoring",
		},
		NotificationChannels: []string{
			fmt.Sprintf("projects/%s/notificationChannels/test-channel-1", projectID),
			fmt.Sprintf("projects/%s/notificationChannels/test-channel-2", projectID),
		},
		Enabled: wrapperspb.Bool(true),
	}

	// Create second alert policy for list testing
	policyID2 := "test-alert-policy-456"
	alertPolicy2 := &monitoringpb.AlertPolicy{
		Name:        fmt.Sprintf("projects/%s/alertPolicies/%s", projectID, policyID2),
		DisplayName: "Test Alert Policy 2",
		Documentation: &monitoringpb.AlertPolicy_Documentation{
			Content: "Second test alert policy",
		},
		Enabled: wrapperspb.Bool(false),
	}

	// Create list response with multiple items
	alertPolicyList := &monitoringpb.ListAlertPoliciesResponse{
		AlertPolicies: []*monitoringpb.AlertPolicy{alertPolicy, alertPolicy2},
	}

	sdpItemType := gcpshared.MonitoringAlertPolicy

	// Mock HTTP responses
	expectedCallAndResponses := map[string]shared.MockResponse{
		fmt.Sprintf("https://monitoring.googleapis.com/v3/projects/%s/alertPolicies/%s", projectID, policyID): {
			StatusCode: http.StatusOK,
			Body:       alertPolicy,
		},
		fmt.Sprintf("https://monitoring.googleapis.com/v3/projects/%s/alertPolicies/%s", projectID, policyID2): {
			StatusCode: http.StatusOK,
			Body:       alertPolicy2,
		},
		fmt.Sprintf("https://monitoring.googleapis.com/v3/projects/%s/alertPolicies", projectID): {
			StatusCode: http.StatusOK,
			Body:       alertPolicyList,
		},
	}

	t.Run("Get", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		sdpItem, err := adapter.Get(ctx, projectID, policyID, true)
		if err != nil {
			t.Fatalf("Failed to get resource: %v", err)
		}

		// Validate SDP item properties
		if sdpItem.GetType() != sdpItemType.String() {
			t.Errorf("Expected type %s, got %s", sdpItemType.String(), sdpItem.GetType())
		}
		if sdpItem.UniqueAttributeValue() != policyID {
			t.Errorf("Expected unique attribute value '%s', got %s", policyID, sdpItem.UniqueAttributeValue())
		}
		if sdpItem.GetScope() != projectID {
			t.Errorf("Expected scope '%s', got %s", projectID, sdpItem.GetScope())
		}

		// Validate specific attributes
		val, err := sdpItem.GetAttributes().Get("name")
		if err != nil {
			t.Fatalf("Failed to get 'name' attribute: %v", err)
		}
		expectedName := fmt.Sprintf("projects/%s/alertPolicies/%s", projectID, policyID)
		if val != expectedName {
			t.Errorf("Expected name field to be '%s', got %s", expectedName, val)
		}

		// Include static tests - covers ALL blast propagation links
		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				// Notification channel links
				{
					ExpectedType:   gcpshared.MonitoringNotificationChannel.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-channel-1",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					ExpectedType:   gcpshared.MonitoringNotificationChannel.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-channel-2",
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

		// Validate first item
		if len(sdpItems) > 0 {
			firstItem := sdpItems[0]
			if firstItem.GetType() != sdpItemType.String() {
				t.Errorf("Expected first item type %s, got %s", sdpItemType.String(), firstItem.GetType())
			}
			if firstItem.GetScope() != projectID {
				t.Errorf("Expected first item scope '%s', got %s", projectID, firstItem.GetScope())
			}
		}
	})

	t.Run("Search with Terraform format", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Skipf("Adapter for %s does not implement SearchableAdapter", sdpItemType)
		}

		// Test Terraform format: projects/[project]/alertPolicies/[alert_policy_id]
		terraformQuery := fmt.Sprintf("projects/%s/alertPolicies/%s", projectID, policyID)
		sdpItems, err := searchable.Search(ctx, projectID, terraformQuery, true)
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
		if firstItem.GetScope() != projectID {
			t.Errorf("Expected first item scope '%s', got %s", projectID, firstItem.GetScope())
		}
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		// Test with error responses to simulate API errors
		errorResponses := map[string]shared.MockResponse{
			fmt.Sprintf("https://monitoring.googleapis.com/v3/projects/%s/alertPolicies/%s", projectID, policyID): {
				StatusCode: http.StatusNotFound,
				Body:       map[string]interface{}{"error": "Alert policy not found"},
			},
		}

		httpCli := shared.NewMockHTTPClientProvider(errorResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		_, err = adapter.Get(ctx, projectID, policyID, true)
		if err == nil {
			t.Error("Expected error when getting non-existent resource, but got nil")
		}
	})
}
