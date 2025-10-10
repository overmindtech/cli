package adapters_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"google.golang.org/api/pubsub/v1"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources/gcp/dynamic"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

func TestPubSubSubscription(t *testing.T) {
	ctx := context.Background()
	projectID := "test-project"
	linker := gcpshared.NewLinker()
	subscriptionName := "test-subscription"

	subscription := &pubsub.Subscription{
		Name:  fmt.Sprintf("projects/%s/subscriptions/%s", projectID, subscriptionName),
		Topic: fmt.Sprintf("projects/%s/topics/test-topic", projectID),
		BigqueryConfig: &pubsub.BigQueryConfig{
			Table: "test-project.test_dataset.test_table",
		},
		CloudStorageConfig: &pubsub.CloudStorageConfig{
			Bucket: "test-bucket",
		},
	}

	subscriptionList := &pubsub.ListSubscriptionsResponse{
		Subscriptions: []*pubsub.Subscription{subscription},
	}

	sdpItemType := gcpshared.PubSubSubscription

	expectedCallAndResponses := map[string]shared.MockResponse{
		fmt.Sprintf("https://pubsub.googleapis.com/v1/projects/%s/subscriptions/%s", projectID, subscriptionName): {
			StatusCode: http.StatusOK,
			Body:       subscription,
		},
		fmt.Sprintf("https://pubsub.googleapis.com/v1/projects/%s/subscriptions", projectID): {
			StatusCode: http.StatusOK,
			Body:       subscriptionList,
		},
	}

	t.Run("Get", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		sdpItem, err := adapter.Get(ctx, projectID, subscriptionName, true)
		if err != nil {
			t.Fatalf("Failed to get subscription: %v", err)
		}

		if sdpItem.GetType() != sdpItemType.String() {
			t.Errorf("Expected type %s, got %s", sdpItemType.String(), sdpItem.GetType())
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					// topic
					ExpectedType:   gcpshared.PubSubTopic.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-topic",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					// bigqueryConfig.table
					ExpectedType:   gcpshared.BigQueryTable.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey("test_dataset", "test_table"),
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					// cloudStorageConfig.bucket
					ExpectedType:   gcpshared.StorageBucket.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-bucket",
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
			t.Fatalf("Adapter for %s does not implement ListableAdapter", sdpItemType)
		}

		sdpItems, err := listable.List(ctx, projectID, true)
		if err != nil {
			t.Fatalf("Failed to list subscriptions: %v", err)
		}

		if len(sdpItems) != 1 {
			t.Errorf("Expected 1 subscription, got %d", len(sdpItems))
		}
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		errorResponses := map[string]shared.MockResponse{
			fmt.Sprintf("https://pubsub.googleapis.com/v1/projects/%s/subscriptions/%s", projectID, subscriptionName): {
				StatusCode: http.StatusNotFound,
				Body:       map[string]interface{}{"error": "Subscription not found"},
			},
		}

		httpCli := shared.NewMockHTTPClientProvider(errorResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		_, err = adapter.Get(ctx, projectID, subscriptionName, true)
		if err == nil {
			t.Error("Expected error when getting non-existent subscription, but got nil")
		}
	})
}
