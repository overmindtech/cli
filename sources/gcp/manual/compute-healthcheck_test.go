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

func TestComputeHealthCheck(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mocks.NewMockComputeHealthCheckClient(ctrl)
	projectID := "test-project-id"

	t.Run("Get", func(t *testing.T) {
		wrapper := manual.NewComputeHealthCheck(mockClient, projectID)

		mockClient.EXPECT().Get(ctx, gomock.Any()).Return(createHealthCheck("test-healthcheck"), nil)

		adapter := sources.WrapperToAdapter(wrapper)

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], "test-healthcheck", true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		// [SPEC] The default scope is the project ID.
		if sdpItem.GetScope() != "test-project-id" {
			t.Fatalf("Expected scope to be 'test-project-id', got: %s", sdpItem.GetScope())
		}

		// [SPEC] HealthChecks have no linked items.
		if len(sdpItem.GetLinkedItemQueries()) != 0 {
			t.Fatalf("Expected 0 linked item queries, got: %d", len(sdpItem.GetLinkedItemQueries()))
		}

	})

	t.Run("List", func(t *testing.T) {
		wrapper := manual.NewComputeHealthCheck(mockClient, projectID)

		adapter := sources.WrapperToAdapter(wrapper)

		mockComputeHealthCheckIter := mocks.NewMockComputeHealthCheckIterator(ctrl)

		// Mock out items listed from the API.
		mockComputeHealthCheckIter.EXPECT().Next().Return(createHealthCheck("test-healthcheck-1"), nil)
		mockComputeHealthCheckIter.EXPECT().Next().Return(createHealthCheck("test-healthcheck-2"), nil)
		mockComputeHealthCheckIter.EXPECT().Next().Return(nil, iterator.Done)

		// Mock the List method
		mockClient.EXPECT().List(ctx, gomock.Any()).Return(mockComputeHealthCheckIter)

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
		wrapper := manual.NewComputeHealthCheck(mockClient, projectID)
		adapter := sources.WrapperToAdapter(wrapper)

		mockComputeHealthCheckIter := mocks.NewMockComputeHealthCheckIterator(ctrl)
		mockComputeHealthCheckIter.EXPECT().Next().Return(createHealthCheck("test-healthcheck-1"), nil)
		mockComputeHealthCheckIter.EXPECT().Next().Return(createHealthCheck("test-healthcheck-2"), nil)
		mockComputeHealthCheckIter.EXPECT().Next().Return(nil, iterator.Done)

		mockClient.EXPECT().List(ctx, gomock.Any()).Return(mockComputeHealthCheckIter)

		wg := &sync.WaitGroup{}
		wg.Add(2)

		var items []*sdp.Item
		var errs []error
		mockItemHandler := func(item *sdp.Item) { items = append(items, item); wg.Done() }
		mockErrorHandler := func(err error) { errs = append(errs, err) }

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

func createHealthCheck(healthCheckName string) *computepb.HealthCheck {
	return &computepb.HealthCheck{
		Name:             ptr.To(healthCheckName),
		CheckIntervalSec: ptr.To(int32(5)),
		TimeoutSec:       ptr.To(int32(5)),
		Type:             ptr.To("TCP"),
		TcpHealthCheck: &computepb.TCPHealthCheck{
			Port: ptr.To(int32(80)),
		},
	}
}
