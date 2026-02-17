package manual_test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"cloud.google.com/go/iam/admin/apiv1/adminpb"
	"go.uber.org/mock/gomock"

	"github.com/overmindtech/cli/go/discovery"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/gcp/manual"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/gcp/shared/mocks"
	"github.com/overmindtech/cli/sources/shared"
)

func TestIAMServiceAccountKey(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mocks.NewMockIAMServiceAccountKeyClient(ctrl)
	projectID := "test-project-id"

	testServiceAccount := "test-sa@test-project-id.iam.gserviceaccount.com"
	testKeyName := "1234567890abcdef"
	testKeyFullName := "projects/test-project-id/serviceAccounts/test-sa@test-project-id.iam.gserviceaccount.com/keys/1234567890abcdef"

	t.Run("Get", func(t *testing.T) {
		wrapper := manual.NewIAMServiceAccountKey(mockClient, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})

		mockClient.EXPECT().Get(ctx, gomock.Any()).Return(createServiceAccountKey(testKeyFullName), nil)

		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], shared.CompositeLookupKey(testServiceAccount, testKeyName), true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					ExpectedType:             gcpshared.IAMServiceAccount.String(),
					ExpectedMethod:           sdp.QueryMethod_GET,
					ExpectedQuery:            testServiceAccount,
					ExpectedScope:            projectID,
				},
			}

			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("Search", func(t *testing.T) {
		wrapper := manual.NewIAMServiceAccountKey(mockClient, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		mockClient.EXPECT().Search(ctx, gomock.Any()).Return(&adminpb.ListServiceAccountKeysResponse{
			Keys: []*adminpb.ServiceAccountKey{
				createServiceAccountKey(testKeyFullName),
			},
		}, nil)

		// Check if adapter supports searching
		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support Search operation")
		}

		sdpItems, err := searchable.Search(ctx, wrapper.Scopes()[0], testServiceAccount, true)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		expectedCount := 1
		actualCount := len(sdpItems)
		if actualCount != expectedCount {
			t.Fatalf("Expected %d items, got: %d", expectedCount, actualCount)
		}

		for _, item := range sdpItems {
			if err := item.Validate(); err != nil {
				t.Fatalf("Expected no validation error, got: %v", err)
			}
		}
	})

	t.Run("SearchCachesNotFoundWithMemoryCache", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockClient := mocks.NewMockIAMServiceAccountKeyClient(ctrl)
		projectID := "cache-test-project"
		scope := projectID
		query := "nonexistent-sa@cache-test-project.iam.gserviceaccount.com"

		mockClient.EXPECT().Search(ctx, gomock.Any()).Return(&adminpb.ListServiceAccountKeysResponse{Keys: nil}, nil).Times(1)

		wrapper := manual.NewIAMServiceAccountKey(mockClient, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		cache := sdpcache.NewMemoryCache()
		adapter := sources.WrapperToAdapter(wrapper, cache)
		discAdapter := adapter.(discovery.Adapter)
		searchable := adapter.(discovery.SearchableAdapter)

		items, err := searchable.Search(ctx, scope, query, false)
		if err != nil {
			t.Fatalf("first Search: unexpected error: %v", err)
		}
		if len(items) != 0 {
			t.Errorf("first Search: expected 0 items, got %d", len(items))
		}

		cacheHit, _, _, qErr, done := cache.Lookup(ctx, discAdapter.Name(), sdp.QueryMethod_SEARCH, scope, discAdapter.Type(), query, false)
		done()
		if !cacheHit {
			t.Fatal("expected cache hit for Search after first call")
		}
		if qErr == nil || qErr.GetErrorType() != sdp.QueryError_NOTFOUND {
			t.Fatalf("expected cached NOTFOUND for Search, got %v", qErr)
		}

		items, err = searchable.Search(ctx, scope, query, false)
		if err != nil {
			t.Fatalf("second Search: unexpected error: %v", err)
		}
		if len(items) != 0 {
			t.Errorf("second Search: expected 0 items, got %d", len(items))
		}
	})

	t.Run("SearchWithTerraformQueryMap", func(t *testing.T) {
		wrapper := manual.NewIAMServiceAccountKey(mockClient, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})

		mockClient.EXPECT().Get(ctx, gomock.Any()).Return(createServiceAccountKey(testKeyFullName), nil)

		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		// projects/{{project}}/serviceAccounts/{{account}}/keys/{{key}}
		terraformResourceID := fmt.Sprintf("projects/%s/serviceAccounts/%s/keys/%s", projectID, testServiceAccount, testKeyName)

		// Check if adapter supports searching
		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support Search operation")
		}

		sdpItems, err := searchable.Search(ctx, wrapper.Scopes()[0], terraformResourceID, true)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		expectedCount := 1
		actualCount := len(sdpItems)
		if actualCount != expectedCount {
			t.Fatalf("Expected %d items, got: %d", expectedCount, actualCount)
		}

		if err := sdpItems[0].Validate(); err != nil {
			t.Fatalf("Expected no validation error, got: %v", err)
		}
	})

	t.Run("SearchStream", func(t *testing.T) {
		wrapper := manual.NewIAMServiceAccountKey(mockClient, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		mockClient.EXPECT().Search(ctx, gomock.Any()).Return(&adminpb.ListServiceAccountKeysResponse{
			Keys: []*adminpb.ServiceAccountKey{
				createServiceAccountKey(testKeyFullName),
			},
		}, nil)

		var items []*sdp.Item
		var errs []error
		wg := &sync.WaitGroup{}
		wg.Add(1)

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

		searchStreamable.SearchStream(ctx, wrapper.Scopes()[0], testServiceAccount, true, stream)
		wg.Wait()

		if len(errs) > 0 {
			t.Fatalf("Expected no errors, got: %v", errs)
		}
		if len(items) != 1 {
			t.Fatalf("Expected 1 item, got: %d", len(items))
		}
		for _, item := range items {
			if err := item.Validate(); err != nil {
				t.Fatalf("Expected no validation error, got: %v", err)
			}
		}

		_, ok = adapter.(discovery.ListStreamableAdapter)
		if ok {
			t.Fatalf("Adapter should not support ListStream operation")
		}
	})

	t.Run("List_Unsupported", func(t *testing.T) {
		wrapper := manual.NewIAMServiceAccountKey(mockClient, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		// Check if adapter supports list - it should not
		_, ok := adapter.(discovery.ListableAdapter)
		if ok {
			t.Fatalf("Expected adapter to not support List operation, but it does")
		}
	})
}

// createServiceAccountKey creates a ServiceAccountKey with the specified name.
func createServiceAccountKey(name string) *adminpb.ServiceAccountKey {
	return &adminpb.ServiceAccountKey{
		Name: name,
	}
}
