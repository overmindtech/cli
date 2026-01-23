package manual_test

import (
	"context"
	"strings"
	"sync"
	"testing"

	"cloud.google.com/go/kms/apiv1/kmspb"
	"go.uber.org/mock/gomock"
	"google.golang.org/api/iterator"
	locationpb "google.golang.org/genproto/googleapis/cloud/location"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/gcp/manual"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/gcp/shared/mocks"
	"github.com/overmindtech/cli/sources/shared"
)

func TestCloudKMSKeyRing(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mocks.NewMockCloudKMSKeyRingClient(ctrl)
	projectID := "test-project-id"
	location := "us"
	keyRingName := "test-keyring"

	t.Run("Get", func(t *testing.T) {
		wrapper := manual.NewCloudKMSKeyRing(mockClient, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})

		mockClient.EXPECT().Get(ctx, gomock.Any()).Return(createKeyRing(projectID, location, keyRingName), nil)

		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], shared.CompositeLookupKey(location, keyRingName), true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					ExpectedType:   gcpshared.IAMPolicy.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "us|test-keyring",
					ExpectedScope:  "test-project-id",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					ExpectedType:   gcpshared.CloudKMSCryptoKey.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  "us|test-keyring",
					ExpectedScope:  "test-project-id",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  false,
						Out: true,
					},
				},
			}

			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("Search", func(t *testing.T) {
		wrapper := manual.NewCloudKMSKeyRing(mockClient, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		mockIterator := mocks.NewMockCloudKMSKeyRingIterator(ctrl)

		mockIterator.EXPECT().Next().Return(createKeyRing(projectID, location, "test-keyring-1"), nil)
		mockIterator.EXPECT().Next().Return(createKeyRing(projectID, location, "test-keyring-2"), nil)
		mockIterator.EXPECT().Next().Return(nil, iterator.Done)

		mockClient.EXPECT().Search(ctx, gomock.Any()).Return(mockIterator)

		// Check if adapter supports searching
		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support Search operation")
		}

		sdpItems, err := searchable.Search(ctx, wrapper.Scopes()[0], location, true)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		expectedCount := 2
		actualCount := len(sdpItems)
		if actualCount != expectedCount {
			t.Fatalf("Expected %d items, got: %d", expectedCount, actualCount)
		}
		for _, item := range sdpItems {
			if item.Validate() != nil {
				t.Fatalf("Expected no validation error, got: %v", item.Validate())
			}
		}
	})

	t.Run("SearchStream", func(t *testing.T) {
		wrapper := manual.NewCloudKMSKeyRing(mockClient, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		mockIterator := mocks.NewMockCloudKMSKeyRingIterator(ctrl)

		mockIterator.EXPECT().Next().Return(createKeyRing(projectID, location, "test-keyring-1"), nil)
		mockIterator.EXPECT().Next().Return(createKeyRing(projectID, location, "test-keyring-2"), nil)
		mockIterator.EXPECT().Next().Return(nil, iterator.Done)

		mockClient.EXPECT().Search(ctx, gomock.Any()).Return(mockIterator)

		var items []*sdp.Item
		var errs []error
		wg := &sync.WaitGroup{}
		wg.Add(2)

		mockItemHandler := func(item *sdp.Item) {
			items = append(items, item)
			wg.Done()
		}
		mockErrorHandler := func(err error) {
			errs = append(errs, err)
		}

		stream := discovery.NewQueryResultStream(mockItemHandler, mockErrorHandler)
		// Check if adapter supports search streaming
		searchStreamable, ok := adapter.(discovery.SearchStreamableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support SearchStream operation")
		}

		searchStreamable.SearchStream(ctx, wrapper.Scopes()[0], location, true, stream)
		wg.Wait()

		if len(errs) > 0 {
			t.Fatalf("Expected no errors, got: %v", errs)
		}
		if len(items) != 2 {
			t.Fatalf("Expected 2 items, got: %d", len(items))
		}
		for _, item := range items {
			if item.Validate() != nil {
				t.Fatalf("Expected no validation error, got: %v", item.Validate())
			}
		}

		// Verify adapter supports ListStream
		_, ok = adapter.(discovery.ListStreamableAdapter)
		if !ok {
			t.Fatalf("Adapter should support ListStream operation")
		}
	})

	t.Run("List", func(t *testing.T) {
		wrapper := manual.NewCloudKMSKeyRing(mockClient, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		// Mock ListLocations
		mockLocationIterator := mocks.NewMockCloudKMSLocationIterator(ctrl)
		mockLocationIterator.EXPECT().Next().Return(createLocation(projectID, "us-central1"), nil)
		mockLocationIterator.EXPECT().Next().Return(createLocation(projectID, "europe-west1"), nil)
		mockLocationIterator.EXPECT().Next().Return(nil, iterator.Done)

		mockClient.EXPECT().ListLocations(ctx, gomock.Any()).Return(mockLocationIterator)

		// Mock Search for first location (us-central1)
		mockKeyRingIterator1 := mocks.NewMockCloudKMSKeyRingIterator(ctrl)
		mockKeyRingIterator1.EXPECT().Next().Return(createKeyRing(projectID, "us-central1", "keyring-1"), nil)
		mockKeyRingIterator1.EXPECT().Next().Return(nil, iterator.Done)

		// Mock Search for second location (europe-west1)
		mockKeyRingIterator2 := mocks.NewMockCloudKMSKeyRingIterator(ctrl)
		mockKeyRingIterator2.EXPECT().Next().Return(createKeyRing(projectID, "europe-west1", "keyring-2"), nil)
		mockKeyRingIterator2.EXPECT().Next().Return(createKeyRing(projectID, "europe-west1", "keyring-3"), nil)
		mockKeyRingIterator2.EXPECT().Next().Return(nil, iterator.Done)

	// Expect Search calls for both locations
	// Use gomock.Any() for ctx because the pool with context wraps it with cancellation
	mockClient.EXPECT().Search(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, req *kmspb.ListKeyRingsRequest, opts ...any) gcpshared.CloudKMSKeyRingIterator {
		if strings.Contains(req.GetParent(), "us-central1") {
			return mockKeyRingIterator1
		}
		return mockKeyRingIterator2
	}).Times(2)

		// Check if adapter supports listing
		listable, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support List operation")
		}

		sdpItems, err := listable.List(ctx, wrapper.Scopes()[0], true)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		// Expect 3 items total (1 from us-central1, 2 from europe-west1)
		if len(sdpItems) != 3 {
			t.Fatalf("Expected 3 items, got: %d", len(sdpItems))
		}

		for _, item := range sdpItems {
			if item.Validate() != nil {
				t.Fatalf("Expected no validation error, got: %v", item.Validate())
			}
		}
	})

	t.Run("ListStream", func(t *testing.T) {
		wrapper := manual.NewCloudKMSKeyRing(mockClient, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		// Mock ListLocations
		mockLocationIterator := mocks.NewMockCloudKMSLocationIterator(ctrl)
		mockLocationIterator.EXPECT().Next().Return(createLocation(projectID, "us-central1"), nil)
		mockLocationIterator.EXPECT().Next().Return(createLocation(projectID, "europe-west1"), nil)
		mockLocationIterator.EXPECT().Next().Return(nil, iterator.Done)

		mockClient.EXPECT().ListLocations(ctx, gomock.Any()).Return(mockLocationIterator)

		// Mock Search for first location (us-central1)
		mockKeyRingIterator1 := mocks.NewMockCloudKMSKeyRingIterator(ctrl)
		mockKeyRingIterator1.EXPECT().Next().Return(createKeyRing(projectID, "us-central1", "keyring-1"), nil)
		mockKeyRingIterator1.EXPECT().Next().Return(nil, iterator.Done)

		// Mock Search for second location (europe-west1)
		mockKeyRingIterator2 := mocks.NewMockCloudKMSKeyRingIterator(ctrl)
		mockKeyRingIterator2.EXPECT().Next().Return(createKeyRing(projectID, "europe-west1", "keyring-2"), nil)
		mockKeyRingIterator2.EXPECT().Next().Return(createKeyRing(projectID, "europe-west1", "keyring-3"), nil)
		mockKeyRingIterator2.EXPECT().Next().Return(nil, iterator.Done)

	// Expect Search calls for both locations (order may vary due to parallelism)
	// Use gomock.Any() for ctx because the pool with context wraps it with cancellation
	mockClient.EXPECT().Search(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, req *kmspb.ListKeyRingsRequest, opts ...any) gcpshared.CloudKMSKeyRingIterator {
		if strings.Contains(req.GetParent(), "us-central1") {
			return mockKeyRingIterator1
		}
		return mockKeyRingIterator2
	}).Times(2)

		var items []*sdp.Item
		var itemsMu sync.Mutex
		var errs []error
		var errsMu sync.Mutex
		wg := &sync.WaitGroup{}
		wg.Add(3) // 3 total items expected

		mockItemHandler := func(item *sdp.Item) {
			itemsMu.Lock()
			items = append(items, item)
			itemsMu.Unlock()
			wg.Done()
		}
		mockErrorHandler := func(err error) {
			errsMu.Lock()
			errs = append(errs, err)
			errsMu.Unlock()
		}

		stream := discovery.NewQueryResultStream(mockItemHandler, mockErrorHandler)

		// Check if adapter supports list streaming
		listStreamable, ok := adapter.(discovery.ListStreamableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support ListStream operation")
		}

		listStreamable.ListStream(ctx, wrapper.Scopes()[0], true, stream)
		wg.Wait()

		if len(errs) > 0 {
			t.Fatalf("Expected no errors, got: %v", errs)
		}
		if len(items) != 3 {
			t.Fatalf("Expected 3 items, got: %d", len(items))
		}
		for _, item := range items {
			if item.Validate() != nil {
				t.Fatalf("Expected no validation error, got: %v", item.Validate())
			}
		}
	})
}

// createKeyRing creates a KeyRing with the specified project, location, and keyRing name.
func createKeyRing(projectID, location, keyRingName string) *kmspb.KeyRing {
	return &kmspb.KeyRing{
		Name:       "projects/" + projectID + "/locations/" + location + "/keyRings/" + keyRingName,
		CreateTime: nil, // You can set a timestamp if needed
	}
}

// createLocation creates a Location with the specified project and location ID.
func createLocation(projectID, locationID string) *locationpb.Location {
	return &locationpb.Location{
		Name:        "projects/" + projectID + "/locations/" + locationID,
		LocationId:  locationID,
		DisplayName: locationID,
	}
}
