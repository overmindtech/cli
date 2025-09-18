package manual_test

import (
	"context"
	"sync"
	"testing"

	"cloud.google.com/go/kms/apiv1/kmspb"
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

func TestCloudKMSKeyRing(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mocks.NewMockCloudKMSKeyRingClient(ctrl)
	projectID := "test-project-id"
	location := "us"
	keyRingName := "test-keyring"

	t.Run("Get", func(t *testing.T) {
		wrapper := manual.NewCloudKMSKeyRing(mockClient, projectID)

		mockClient.EXPECT().Get(ctx, gomock.Any()).Return(createKeyRing(projectID, location, keyRingName), nil)

		adapter := sources.WrapperToAdapter(wrapper)

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
		wrapper := manual.NewCloudKMSKeyRing(mockClient, projectID)
		adapter := sources.WrapperToAdapter(wrapper)

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
		wrapper := manual.NewCloudKMSKeyRing(mockClient, projectID)
		adapter := sources.WrapperToAdapter(wrapper)

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

		_, ok = adapter.(discovery.ListStreamableAdapter)
		if ok {
			t.Fatalf("Adapter should not support ListStream operation")
		}
	})

	t.Run("List_Unsupported", func(t *testing.T) {
		wrapper := manual.NewCloudKMSKeyRing(mockClient, projectID)
		adapter := sources.WrapperToAdapter(wrapper)

		// Check if adapter supports list - it should not
		_, ok := adapter.(discovery.ListableAdapter)
		if ok {
			t.Fatalf("Expected adapter to not support List operation, but it does")
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
