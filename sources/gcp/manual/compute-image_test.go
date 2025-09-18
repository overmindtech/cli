package manual_test

import (
	"context"
	"sync"
	"testing"

	"cloud.google.com/go/compute/apiv1/computepb"
	"go.uber.org/mock/gomock"
	"google.golang.org/api/iterator"
	"k8s.io/utils/ptr"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/gcp/manual"
	"github.com/overmindtech/cli/sources/gcp/shared/mocks"
)

func TestComputeImage(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mocks.NewMockComputeImagesClient(ctrl)
	projectID := "test-project-id"

	t.Run("Get", func(t *testing.T) {
		wrapper := manual.NewComputeImage(mockClient, projectID)

		mockClient.EXPECT().Get(ctx, gomock.Any()).Return(createComputeImage("test-image", computepb.Image_READY), nil)

		adapter := sources.WrapperToAdapter(wrapper)

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "test-image", true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetTags()["env"] != "test" {
			t.Fatalf("Expected tag 'env=test', got: %v", sdpItem.GetTags()["env"])
		}
	})

	t.Run("HealthCheck", func(t *testing.T) {
		type testCase struct {
			name     string
			input    computepb.Image_Status
			expected sdp.Health
		}
		testCases := []testCase{
			{
				name:     "Undefined",
				input:    computepb.Image_UNDEFINED_STATUS,
				expected: sdp.Health_HEALTH_UNKNOWN,
			},
			{
				name:     "Deleting",
				input:    computepb.Image_DELETING,
				expected: sdp.Health_HEALTH_PENDING,
			},
			{
				name:     "Failed",
				input:    computepb.Image_FAILED,
				expected: sdp.Health_HEALTH_ERROR,
			},
			{
				name:     "Pending",
				input:    computepb.Image_PENDING,
				expected: sdp.Health_HEALTH_PENDING,
			},
			{
				name:     "Healthy",
				input:    computepb.Image_READY,
				expected: sdp.Health_HEALTH_OK,
			},
		}

		mockClient = mocks.NewMockComputeImagesClient(ctrl)

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				wrapper := manual.NewComputeImage(mockClient, projectID)
				adapter := sources.WrapperToAdapter(wrapper)

				mockClient.EXPECT().Get(ctx, gomock.Any()).Return(createComputeImage("test-instance", tc.input), nil)

				sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "test-instance", true)
				if qErr != nil {
					t.Fatalf("Expected no error, got: %v", qErr)
				}

				if sdpItem.GetHealth() != tc.expected {
					t.Fatalf("Expected health %s, got: %s", tc.expected, sdpItem.GetHealth())
				}
			})
		}
	})

	t.Run("List", func(t *testing.T) {
		wrapper := manual.NewComputeImage(mockClient, projectID)

		adapter := sources.WrapperToAdapter(wrapper)

		mockComputeIterator := mocks.NewMockComputeImageIterator(ctrl)

		mockComputeIterator.EXPECT().Next().Return(createComputeImage("test-image-1", computepb.Image_READY), nil)
		mockComputeIterator.EXPECT().Next().Return(createComputeImage("test-image-2", computepb.Image_READY), nil)
		mockComputeIterator.EXPECT().Next().Return(nil, iterator.Done)

		mockClient.EXPECT().List(ctx, gomock.Any()).Return(mockComputeIterator)

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

			if item.GetTags()["env"] != "test" {
				t.Fatalf("Expected tag 'env=test', got: %s", item.GetTags()["env"])
			}
		}

		_, ok = adapter.(discovery.SearchStreamableAdapter)
		if ok {
			t.Fatalf("Adapter should not support SearchStream operation")
		}
	})

	t.Run("ListStream", func(t *testing.T) {
		wrapper := manual.NewComputeImage(mockClient, projectID)

		adapter := sources.WrapperToAdapter(wrapper)

		mockComputeIterator := mocks.NewMockComputeImageIterator(ctrl)

		// add mock implementation here
		mockComputeIterator.EXPECT().Next().Return(createComputeImage("test-image-1", computepb.Image_READY), nil)
		mockComputeIterator.EXPECT().Next().Return(createComputeImage("test-image-2", computepb.Image_READY), nil)
		mockComputeIterator.EXPECT().Next().Return(nil, iterator.Done)

		// Mock the List method
		mockClient.EXPECT().List(ctx, gomock.Any()).Return(mockComputeIterator)

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

func createComputeImage(imageName string, status computepb.Image_Status) *computepb.Image {
	return &computepb.Image{
		Name:   ptr.To(imageName),
		Labels: map[string]string{"env": "test"},
		Status: ptr.To(status.String()),
	}
}
