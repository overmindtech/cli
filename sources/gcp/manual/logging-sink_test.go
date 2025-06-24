package manual_test

import (
	"context"
	"testing"

	"cloud.google.com/go/logging/apiv2/loggingpb"
	"go.uber.org/mock/gomock"
	"google.golang.org/api/iterator"

	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/gcp/manual"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/gcp/shared/mocks"
	"github.com/overmindtech/cli/sources/shared"
)

func TestNewLoggingSink(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mocks.NewMockLoggingConfigClient(ctrl)
	projectID := "my-project-id"

	t.Run("Get", func(t *testing.T) {
		type testCase struct {
			name              string
			destination       string
			expectedQueryTest shared.QueryTest
		}
		testCases := []testCase{
			{
				name:        "Cloud Storage Bucket",
				destination: "storage.googleapis.com/my_bucket",
				expectedQueryTest: shared.QueryTest{
					ExpectedType:   gcpshared.StorageBucket.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "my_bucket",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
			},
			{
				name:        "BigQuery Dataset",
				destination: "bigquery.googleapis.com/projects/my-project-id/datasets/my_dataset",
				expectedQueryTest: shared.QueryTest{
					ExpectedType:   gcpshared.BigQueryDataset.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "my_dataset",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
			},
			{
				name:        "Pub/Sub Topic",
				destination: "pubsub.googleapis.com/projects/my-project-id/topics/my_topic",
				expectedQueryTest: shared.QueryTest{
					ExpectedType:   gcpshared.PubSubTopic.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "my_topic",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
			},
			{
				name:        "Logging Bucket",
				destination: "logging.googleapis.com/projects/my-project-id/locations/global/buckets/my_bucket",
				expectedQueryTest: shared.QueryTest{
					ExpectedType:   gcpshared.LoggingBucket.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey("global", "my_bucket"),
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				wrapper := manual.NewLoggingSink(mockClient, projectID)

				mockClient.EXPECT().GetSink(ctx, gomock.Any()).Return(createLoggingSink("my-sink", tc.destination), nil)

				adapter := sources.WrapperToAdapter(wrapper)

				sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "my-sink", true)
				if qErr != nil {
					t.Fatalf("Expected no error, got: %v", qErr)
				}

				uniqAttr := sdpItem.GetUniqueAttribute()
				uniqAttrVal, err := sdpItem.GetAttributes().Get(uniqAttr)
				if err != nil {
					t.Fatalf("Expected to find unique attribute %s, got error: %v", uniqAttr, err)
				}

				if uniqAttrVal.(string) != "my-sink" {
					t.Errorf("Expected unique attribute value to be 'my-sink', got: %s", uniqAttrVal)
				}

				t.Run("StaticTests", func(t *testing.T) {
					queryTests := shared.QueryTests{tc.expectedQueryTest}
					shared.RunStaticTests(t, adapter, sdpItem, queryTests)
				})
			})
		}
	})

	t.Run("List", func(t *testing.T) {
		wrapper := manual.NewLoggingSink(mockClient, projectID)

		mockLoggingSinkIterator := mocks.NewMockLoggingSinkIterator(ctrl)

		mockLoggingSinkIterator.EXPECT().Next().Return(createLoggingSink("sink1", "storage.googleapis.com/my_bucket"), nil)
		mockLoggingSinkIterator.EXPECT().Next().Return(createLoggingSink("sink2", "bigquery.googleapis.com/projects/my-project-id/datasets/my_dataset"), nil)
		mockLoggingSinkIterator.EXPECT().Next().Return(nil, iterator.Done) // End of iteration

		mockClient.EXPECT().ListSinks(ctx, gomock.Any()).Return(mockLoggingSinkIterator)

		adapter := sources.WrapperToAdapter(wrapper)

		sdpItems, qErr := adapter.List(ctx, wrapper.Scopes()[0], true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if len(sdpItems) != 2 {
			t.Fatalf("Expected 2 items, got: %d", len(sdpItems))
		}

		for _, item := range sdpItems {
			if item.Validate() != nil {
				t.Fatalf("Expected no validation error, got: %v", item.Validate())
			}
		}
	})

}

func createLoggingSink(name, destination string) *loggingpb.LogSink {
	return &loggingpb.LogSink{
		Name:        name,
		Destination: destination,
		Filter:      "severity>=ERROR",
	}
}
