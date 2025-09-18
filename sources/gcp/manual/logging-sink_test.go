package manual_test

import (
	"context"
	"sync"
	"testing"

	"cloud.google.com/go/logging/apiv2/loggingpb"
	"go.uber.org/mock/gomock"
	"google.golang.org/api/iterator"

	"github.com/overmindtech/cli/discovery"
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

		// Check if adapter supports listing
		listable, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support List operation")
		}

		sdpItems, err := listable.List(ctx, wrapper.Scopes()[0], true)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if len(sdpItems) != 2 {
			t.Fatalf("Expected 2 items, got: %d", len(sdpItems))
		}

		for _, item := range sdpItems {
			if item.Validate() != nil {
				t.Fatalf("Expected no validation error, got: %v", item.Validate())
			}
		}

		_, ok = adapter.(discovery.SearchStreamableAdapter)
		if ok {
			t.Fatalf("Adapter should not support SearchStream operation")
		}
	})

	t.Run("ListStream", func(t *testing.T) {
		wrapper := manual.NewLoggingSink(mockClient, projectID)

		adapter := sources.WrapperToAdapter(wrapper)

		mockLoggingSinkIterator := mocks.NewMockLoggingSinkIterator(ctrl)

		// add mock implementation here
		mockLoggingSinkIterator.EXPECT().Next().Return(createLoggingSink("sink1", "storage.googleapis.com/my_bucket"), nil)
		mockLoggingSinkIterator.EXPECT().Next().Return(createLoggingSink("sink2", "bigquery.googleapis.com/projects/my-project-id/datasets/my_dataset"), nil)
		mockLoggingSinkIterator.EXPECT().Next().Return(nil, iterator.Done)

		// Mock the ListSinks method
		mockClient.EXPECT().ListSinks(ctx, gomock.Any()).Return(mockLoggingSinkIterator)

		wg := &sync.WaitGroup{}
		wg.Add(2) // we added two items

		var items []*sdp.Item
		mockItemHandler := func(item *sdp.Item) {
			items = append(items, item)
			wg.Done() // signal that we processed an item
		}

		var errs []error
		mockErrorHandler := func(err error) {
			errs = append(errs, err)
		}

		stream := discovery.NewQueryResultStream(mockItemHandler, mockErrorHandler)
		// Check if adapter supports list streaming
		listStreamable, ok := adapter.(discovery.ListStreamableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support ListStream operation")
		}

		listStreamable.ListStream(ctx, wrapper.Scopes()[0], true, stream)
		wg.Wait()

		if len(errs) != 0 {
			t.Fatalf("Expected no errors, got: %v", errs)
		}

		if len(items) != 2 {
			t.Fatalf("Expected 2 items, got: %d", len(items))
		}

		_, ok = adapter.(discovery.SearchStreamableAdapter)
		if ok {
			t.Fatalf("Adapter should not support SearchStream operation")
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
