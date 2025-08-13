package manual_test

import (
	"context"
	"fmt"
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

func TestCloudKMSCryptoKey(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mocks.NewMockCloudKMSCryptoKeyClient(ctrl)
	projectID := "test-project-id"

	t.Run("Get", func(t *testing.T) {
		wrapper := manual.NewCloudKMSCryptoKey(mockClient, projectID)

		mockClient.EXPECT().Get(ctx, gomock.Any()).Return(
			createCryptoKey(
				"projects/test-project-id/locations/global/keyRings/test-keyring/cryptoKeys/test-key",
				"1",
				kmspb.CryptoKeyVersion_ENABLED,
			), nil)

		adapter := sources.WrapperToAdapter(wrapper)

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], shared.CompositeLookupKey("location", "keyRing", "cryptoKey"), true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		expectedTag := "test"
		actualTag := sdpItem.GetTags()["env"]
		if actualTag != expectedTag {
			t.Fatalf("Expected tag 'env=%s', got: %v", expectedTag, actualTag)
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					ExpectedType:   gcpshared.CloudKMSCryptoKeyVersion.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "global|test-keyring|test-key|1",
					ExpectedScope:  "test-project-id",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					ExpectedType:   gcpshared.CloudKMSEKMConnection.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "global|valid-ekm-connection",
					ExpectedScope:  "test-project-id",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					ExpectedType:   gcpshared.IAMPolicy.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "global|test-keyring|test-key",
					ExpectedScope:  "test-project-id",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
				{
					ExpectedType:   gcpshared.CloudKMSKeyRing.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "global|test-keyring",
					ExpectedScope:  "test-project-id",
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
			}

			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("Search", func(t *testing.T) {
		wrapper := manual.NewCloudKMSCryptoKey(mockClient, projectID)
		adapter := sources.WrapperToAdapter(wrapper)

		mockCryptoKeyIterator := mocks.NewMockCloudKMSCryptoKeyIterator(ctrl)

		mockCryptoKeyIterator.EXPECT().Next().Return(
			createCryptoKey(
				"projects/test-project-id/locations/global/keyRings/test-key-ring/cryptoKeys/test-key-1",
				"1",
				kmspb.CryptoKeyVersion_ENABLED,
			), nil)
		mockCryptoKeyIterator.EXPECT().Next().Return(
			createCryptoKey(
				"projects/test-project-id/locations/global/keyRings/test-key-ring/cryptoKeys/test-key-2",
				"1",
				kmspb.CryptoKeyVersion_ENABLED,
			), nil)
		// This one is for a different key ring and should be filtered out.
		mockCryptoKeyIterator.EXPECT().Next().Return(
			createCryptoKey(
				"projects/test-project-id/locations/global/keyRings/other-key-ring/cryptoKeys/test-key-3",
				"1",
				kmspb.CryptoKeyVersion_ENABLED,
			), nil)
		mockCryptoKeyIterator.EXPECT().Next().Return(nil, iterator.Done)

		mockClient.EXPECT().List(ctx, gomock.Any()).Return(mockCryptoKeyIterator)

		// [SPEC] Search filters by the key ring. It will list all crypto keys
		// any crypto keys that are not using the given key ring.
		sdpItems, err := adapter.Search(ctx, wrapper.Scopes()[0], shared.CompositeLookupKey("location", "key-ring"), true)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		// 2 of 3 are filtered in.
		if len(sdpItems) != 3 {
			t.Fatalf("Expected 2 items, got: %d", len(sdpItems))
		}

		for _, item := range sdpItems {
			if item.Validate() != nil {
				t.Fatalf("Expected no validation error, got: %v", item.Validate())
			}

			attributes := item.GetAttributes()
			_, err := attributes.Get("name")
			if err != nil {
				t.Fatalf("Failed to get name attribute: %v", err)
			}
		}
	})

	t.Run("SearchStream", func(t *testing.T) {
		wrapper := manual.NewCloudKMSCryptoKey(mockClient, projectID)
		adapter := sources.WrapperToAdapter(wrapper)

		mockCryptoKeyIterator := mocks.NewMockCloudKMSCryptoKeyIterator(ctrl)

		// add mock implementation here
		mockCryptoKeyIterator.EXPECT().Next().Return(
			createCryptoKey(
				"projects/test-project-id/locations/global/keyRings/test-key-ring/cryptoKeys/test-key-1",
				"1",
				kmspb.CryptoKeyVersion_ENABLED,
			), nil)
		mockCryptoKeyIterator.EXPECT().Next().Return(
			createCryptoKey(
				"projects/test-project-id/locations/global/keyRings/test-key-ring/cryptoKeys/test-key-2",
				"1",
				kmspb.CryptoKeyVersion_ENABLED,
			), nil)
		mockCryptoKeyIterator.EXPECT().Next().Return(nil, iterator.Done)

		// Mock the List method
		mockClient.EXPECT().List(ctx, gomock.Any()).Return(mockCryptoKeyIterator)

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
		adapter.SearchStream(ctx, wrapper.Scopes()[0], shared.CompositeLookupKey("global", "test-key-ring"), true, stream)
		wg.Wait()

		if len(errs) != 0 {
			t.Fatalf("Expected no errors, got: %v", errs)
		}

		if len(items) != 2 {
			t.Fatalf("Expected 2 items, got: %d", len(items))
		}
	})
}

// createCryptoKey creates a CryptoKey with the specified name, primary version, and state.
func createCryptoKey(name, versionNumber string, state kmspb.CryptoKeyVersion_CryptoKeyVersionState) *kmspb.CryptoKey {
	var primary *kmspb.CryptoKeyVersion
	if versionNumber != "" {
		primary = &kmspb.CryptoKeyVersion{
			Name:            fmt.Sprintf("%s/cryptoKeyVersions/%s", name, versionNumber),
			State:           state,
			ProtectionLevel: kmspb.ProtectionLevel_EXTERNAL_VPC,
		}
	}
	return &kmspb.CryptoKey{
		Name:             name,
		Primary:          primary,
		CryptoKeyBackend: "projects/test-project-id/locations/global/ekmConnections/valid-ekm-connection",
		Labels:           map[string]string{"env": "test"},
	}
}
