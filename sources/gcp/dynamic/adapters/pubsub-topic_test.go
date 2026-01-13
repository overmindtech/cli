package adapters_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"google.golang.org/api/pubsub/v1"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
	"github.com/overmindtech/cli/sources/gcp/dynamic"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

func TestPubSubTopic(t *testing.T) {
	ctx := context.Background()
	projectID := "test-project"
	linker := gcpshared.NewLinker()
	topicName := "test-topic"

	topic := &pubsub.Topic{
		Name:       fmt.Sprintf("projects/%s/topics/%s", projectID, topicName),
		KmsKeyName: "projects/test-project/locations/global/keyRings/my-keyring/cryptoKeys/my-key",
		IngestionDataSourceSettings: &pubsub.IngestionDataSourceSettings{
			CloudStorage: &pubsub.CloudStorage{
				Bucket: "ingestion-bucket",
			},
		},
	}

	topicList := &pubsub.ListTopicsResponse{
		Topics: []*pubsub.Topic{topic},
	}

	sdpItemType := gcpshared.PubSubTopic

	expectedCallAndResponses := map[string]shared.MockResponse{
		fmt.Sprintf("https://pubsub.googleapis.com/v1/projects/%s/topics/%s", projectID, topicName): {
			StatusCode: http.StatusOK,
			Body:       topic,
		},
		fmt.Sprintf("https://pubsub.googleapis.com/v1/projects/%s/topics", projectID): {
			StatusCode: http.StatusOK,
			Body:       topicList,
		},
	}

	t.Run("Get", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		sdpItem, err := adapter.Get(ctx, projectID, topicName, true)
		if err != nil {
			t.Fatalf("Failed to get topic: %v", err)
		}

		if sdpItem.GetType() != sdpItemType.String() {
			t.Errorf("Expected type %s, got %s", sdpItemType.String(), sdpItem.GetType())
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					// kmsKeyName
					ExpectedType:   gcpshared.CloudKMSCryptoKey.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey("global", "my-keyring", "my-key"),
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					// ingestionDataSourceSettings.cloudStorage.bucket
					ExpectedType:   gcpshared.StorageBucket.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "ingestion-bucket",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// TODO: Add tests for AWS Kinesis ingestion settings (streamAr, consumerArn, awsRoleArn)
				// Requires cross-cloud linking setup
			}

			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("List", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		listable, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			t.Fatalf("Adapter for %s does not implement ListableAdapter", sdpItemType)
		}

		sdpItems, err := listable.List(ctx, projectID, true)
		if err != nil {
			t.Fatalf("Failed to list topics: %v", err)
		}

		if len(sdpItems) != 1 {
			t.Errorf("Expected 1 topic, got %d", len(sdpItems))
		}
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		errorResponses := map[string]shared.MockResponse{
			fmt.Sprintf("https://pubsub.googleapis.com/v1/projects/%s/topics/%s", projectID, topicName): {
				StatusCode: http.StatusNotFound,
				Body:       map[string]interface{}{"error": "Topic not found"},
			},
		}

		httpCli := shared.NewMockHTTPClientProvider(errorResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		_, err = adapter.Get(ctx, projectID, topicName, true)
		if err == nil {
			t.Error("Expected error when getting non-existent topic, but got nil")
		}
	})
}
