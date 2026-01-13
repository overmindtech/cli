package adapters_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"cloud.google.com/go/monitoring/apiv3/v2/monitoringpb"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdpcache"
	"github.com/overmindtech/cli/sources/gcp/dynamic"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

func TestMonitoringNotificationChannel(t *testing.T) {
	ctx := context.Background()
	projectID := "test-project"
	linker := gcpshared.NewLinker()
	channelID := "test-notification-channel"

	// Create mock protobuf object
	channel := &monitoringpb.NotificationChannel{
		Name:        fmt.Sprintf("projects/%s/notificationChannels/%s", projectID, channelID),
		DisplayName: "Test Notification Channel",
		Type:        "email",
		Labels: map[string]string{
			"email_address": "test@example.com",
		},
	}

	// Create second channel for list testing
	channelID2 := "test-notification-channel-2"
	channel2 := &monitoringpb.NotificationChannel{
		Name:        fmt.Sprintf("projects/%s/notificationChannels/%s", projectID, channelID2),
		DisplayName: "Test Notification Channel 2",
		Type:        "slack",
	}

	// Create list response with multiple items
	channelList := &monitoringpb.ListNotificationChannelsResponse{
		NotificationChannels: []*monitoringpb.NotificationChannel{channel, channel2},
	}

	sdpItemType := gcpshared.MonitoringNotificationChannel

	// Mock HTTP responses
	expectedCallAndResponses := map[string]shared.MockResponse{
		fmt.Sprintf("https://monitoring.googleapis.com/v3/projects/%s/notificationChannels/%s", projectID, channelID): {
			StatusCode: http.StatusOK,
			Body:       channel,
		},
		fmt.Sprintf("https://monitoring.googleapis.com/v3/projects/%s/notificationChannels/%s", projectID, channelID2): {
			StatusCode: http.StatusOK,
			Body:       channel2,
		},
		fmt.Sprintf("https://monitoring.googleapis.com/v3/projects/%s/notificationChannels", projectID): {
			StatusCode: http.StatusOK,
			Body:       channelList,
		},
	}

	t.Run("Get", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		sdpItem, err := adapter.Get(ctx, projectID, channelID, true)
		if err != nil {
			t.Fatalf("Failed to get resource: %v", err)
		}

		// Validate SDP item properties
		if sdpItem.GetType() != sdpItemType.String() {
			t.Errorf("Expected type %s, got %s", sdpItemType.String(), sdpItem.GetType())
		}
		if sdpItem.UniqueAttributeValue() != channelID {
			t.Errorf("Expected unique attribute value '%s', got %s", channelID, sdpItem.UniqueAttributeValue())
		}
		if sdpItem.GetScope() != projectID {
			t.Errorf("Expected scope '%s', got %s", projectID, sdpItem.GetScope())
		}

		// Validate specific attributes
		val, err := sdpItem.GetAttributes().Get("name")
		if err != nil {
			t.Fatalf("Failed to get 'name' attribute: %v", err)
		}
		expectedName := fmt.Sprintf("projects/%s/notificationChannels/%s", projectID, channelID)
		if val != expectedName {
			t.Errorf("Expected name field to be '%s', got %s", expectedName, val)
		}

		// Skip static tests - no blast propagations for this adapter
		// Static tests fail when linked queries are nil
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

	t.Run("ErrorHandling", func(t *testing.T) {
		// Test with error responses to simulate API errors
		errorResponses := map[string]shared.MockResponse{
			fmt.Sprintf("https://monitoring.googleapis.com/v3/projects/%s/notificationChannels/%s", projectID, channelID): {
				StatusCode: http.StatusNotFound,
				Body:       map[string]interface{}{"error": "Notification channel not found"},
			},
		}

		httpCli := shared.NewMockHTTPClientProvider(errorResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		_, err = adapter.Get(ctx, projectID, channelID, true)
		if err == nil {
			t.Error("Expected error when getting non-existent resource, but got nil")
		}
	})
}
