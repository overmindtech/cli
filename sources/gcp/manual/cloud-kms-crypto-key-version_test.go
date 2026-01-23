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
	"github.com/overmindtech/cli/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/gcp/manual"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/gcp/shared/mocks"
	"github.com/overmindtech/cli/sources/shared"
)

func TestCloudKMSCryptoKeyVersion(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mocks.NewMockCloudKMSCryptoKeyVersionClient(ctrl)
	projectID := "test-project-id"
	location := "us"
	keyRingName := "test-keyring"
	cryptoKeyName := "test-key"

	t.Run("Get", func(t *testing.T) {
		wrapper := manual.NewCloudKMSCryptoKeyVersion(mockClient, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})

		mockClient.EXPECT().Get(ctx, gomock.Any()).Return(
			createCryptoKeyVersion(projectID, location, keyRingName, cryptoKeyName, "1", kmspb.CryptoKeyVersion_ENABLED),
			nil,
		)

		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], shared.CompositeLookupKey(location, keyRingName, cryptoKeyName, "1"), true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetHealth() != sdp.Health_HEALTH_OK {
			t.Fatalf("Expected health OK for ENABLED version, got: %v", sdpItem.GetHealth())
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					ExpectedType:   gcpshared.CloudKMSCryptoKey.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "us|test-keyring|test-key",
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

	t.Run("Get_WithImportJob", func(t *testing.T) {
		wrapper := manual.NewCloudKMSCryptoKeyVersion(mockClient, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		versionWithImport := createCryptoKeyVersion(projectID, location, keyRingName, cryptoKeyName, "2", kmspb.CryptoKeyVersion_ENABLED)
		versionWithImport.ImportJob = "projects/test-project-id/locations/us/keyRings/test-keyring/importJobs/test-import-job"

		mockClient.EXPECT().Get(ctx, gomock.Any()).Return(versionWithImport, nil)

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], shared.CompositeLookupKey(location, keyRingName, cryptoKeyName, "2"), true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		// Verify ImportJob link is present
		foundImportJobLink := false
		for _, link := range sdpItem.GetLinkedItemQueries() {
			if link.GetQuery().GetType() == gcpshared.CloudKMSImportJob.String() {
				foundImportJobLink = true
				expectedQuery := "us|test-keyring|test-import-job"
				if link.GetQuery().GetQuery() != expectedQuery {
					t.Fatalf("Expected ImportJob query '%s', got: %s", expectedQuery, link.GetQuery().GetQuery())
				}
			}
		}
		if !foundImportJobLink {
			t.Fatalf("Expected ImportJob link to be present")
		}
	})

	t.Run("Get_WithEKMConnection", func(t *testing.T) {
		wrapper := manual.NewCloudKMSCryptoKeyVersion(mockClient, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		versionWithEKM := createCryptoKeyVersion(projectID, location, keyRingName, cryptoKeyName, "3", kmspb.CryptoKeyVersion_ENABLED)
		versionWithEKM.ProtectionLevel = kmspb.ProtectionLevel_EXTERNAL_VPC
		versionWithEKM.ExternalProtectionLevelOptions = &kmspb.ExternalProtectionLevelOptions{
			EkmConnectionKeyPath: "projects/test-project-id/locations/us/ekmConnections/test-ekm-connection",
		}

		mockClient.EXPECT().Get(ctx, gomock.Any()).Return(versionWithEKM, nil)

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], shared.CompositeLookupKey(location, keyRingName, cryptoKeyName, "3"), true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		// Verify EKM connection link is present
		foundEKMLink := false
		for _, link := range sdpItem.GetLinkedItemQueries() {
			if link.GetQuery().GetType() == gcpshared.CloudKMSEKMConnection.String() {
				foundEKMLink = true
				expectedQuery := "us|test-ekm-connection"
				if link.GetQuery().GetQuery() != expectedQuery {
					t.Fatalf("Expected EKM connection query '%s', got: %s", expectedQuery, link.GetQuery().GetQuery())
				}
			}
		}
		if !foundEKMLink {
			t.Fatalf("Expected EKM connection link to be present")
		}
	})

	t.Run("Get_HealthStates", func(t *testing.T) {
		wrapper := manual.NewCloudKMSCryptoKeyVersion(mockClient, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		testCases := []struct {
			state          kmspb.CryptoKeyVersion_CryptoKeyVersionState
			expectedHealth sdp.Health
		}{
			{kmspb.CryptoKeyVersion_ENABLED, sdp.Health_HEALTH_OK},
			{kmspb.CryptoKeyVersion_DISABLED, sdp.Health_HEALTH_WARNING},
			{kmspb.CryptoKeyVersion_DESTROYED, sdp.Health_HEALTH_ERROR},
			{kmspb.CryptoKeyVersion_DESTROY_SCHEDULED, sdp.Health_HEALTH_ERROR},
			{kmspb.CryptoKeyVersion_PENDING_GENERATION, sdp.Health_HEALTH_PENDING},
			{kmspb.CryptoKeyVersion_PENDING_IMPORT, sdp.Health_HEALTH_PENDING},
			{kmspb.CryptoKeyVersion_IMPORT_FAILED, sdp.Health_HEALTH_ERROR},
			{kmspb.CryptoKeyVersion_GENERATION_FAILED, sdp.Health_HEALTH_ERROR},
		}

		for _, tc := range testCases {
			t.Run(tc.state.String(), func(t *testing.T) {
				mockClient.EXPECT().Get(ctx, gomock.Any()).Return(
					createCryptoKeyVersion(projectID, location, keyRingName, cryptoKeyName, "1", tc.state),
					nil,
				)

				sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], shared.CompositeLookupKey(location, keyRingName, cryptoKeyName, "1"), true)
				if qErr != nil {
					t.Fatalf("Expected no error, got: %v", qErr)
				}

				if sdpItem.GetHealth() != tc.expectedHealth {
					t.Fatalf("Expected health %v for state %v, got: %v", tc.expectedHealth, tc.state, sdpItem.GetHealth())
				}
			})
		}
	})

	t.Run("Search", func(t *testing.T) {
		wrapper := manual.NewCloudKMSCryptoKeyVersion(mockClient, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		mockIterator := mocks.NewMockCloudKMSCryptoKeyVersionIterator(ctrl)

		mockIterator.EXPECT().Next().Return(
			createCryptoKeyVersion(projectID, location, keyRingName, cryptoKeyName, "1", kmspb.CryptoKeyVersion_ENABLED),
			nil,
		)
		mockIterator.EXPECT().Next().Return(
			createCryptoKeyVersion(projectID, location, keyRingName, cryptoKeyName, "2", kmspb.CryptoKeyVersion_DISABLED),
			nil,
		)
		mockIterator.EXPECT().Next().Return(
			createCryptoKeyVersion(projectID, location, keyRingName, cryptoKeyName, "3", kmspb.CryptoKeyVersion_DESTROYED),
			nil,
		)
		mockIterator.EXPECT().Next().Return(nil, iterator.Done)

		mockClient.EXPECT().List(ctx, gomock.Any()).Return(mockIterator)

		// Check if adapter supports searching
		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support Search operation")
		}

		sdpItems, err := searchable.Search(ctx, wrapper.Scopes()[0], shared.CompositeLookupKey(location, keyRingName, cryptoKeyName), true)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		expectedCount := 3
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
		wrapper := manual.NewCloudKMSCryptoKeyVersion(mockClient, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		mockIterator := mocks.NewMockCloudKMSCryptoKeyVersionIterator(ctrl)

		mockIterator.EXPECT().Next().Return(
			createCryptoKeyVersion(projectID, location, keyRingName, cryptoKeyName, "1", kmspb.CryptoKeyVersion_ENABLED),
			nil,
		)
		mockIterator.EXPECT().Next().Return(
			createCryptoKeyVersion(projectID, location, keyRingName, cryptoKeyName, "2", kmspb.CryptoKeyVersion_DISABLED),
			nil,
		)
		mockIterator.EXPECT().Next().Return(nil, iterator.Done)

		mockClient.EXPECT().List(ctx, gomock.Any()).Return(mockIterator)

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

		searchStreamable.SearchStream(ctx, wrapper.Scopes()[0], shared.CompositeLookupKey(location, keyRingName, cryptoKeyName), true, stream)
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

		// Verify adapter does not support ListStream
		_, ok = adapter.(discovery.ListStreamableAdapter)
		if ok {
			t.Fatalf("Adapter should not support ListStream operation")
		}
	})

	t.Run("List_Unsupported", func(t *testing.T) {
		wrapper := manual.NewCloudKMSCryptoKeyVersion(mockClient, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		// Check if adapter supports list - it should not
		_, ok := adapter.(discovery.ListableAdapter)
		if ok {
			t.Fatalf("Expected adapter to not support List operation, but it does")
		}
	})
}

// createCryptoKeyVersion creates a CryptoKeyVersion with the specified parameters.
func createCryptoKeyVersion(projectID, location, keyRing, cryptoKey, version string, state kmspb.CryptoKeyVersion_CryptoKeyVersionState) *kmspb.CryptoKeyVersion {
	return &kmspb.CryptoKeyVersion{
		Name: fmt.Sprintf("projects/%s/locations/%s/keyRings/%s/cryptoKeys/%s/cryptoKeyVersions/%s",
			projectID, location, keyRing, cryptoKey, version),
		State:           state,
		ProtectionLevel: kmspb.ProtectionLevel_SOFTWARE,
		Algorithm:       kmspb.CryptoKeyVersion_GOOGLE_SYMMETRIC_ENCRYPTION,
	}
}
