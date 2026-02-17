package manual_test

import (
	"context"
	"testing"
	"time"

	"cloud.google.com/go/bigquery"
	"go.uber.org/mock/gomock"

	"github.com/overmindtech/cli/go/discovery"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/gcp/manual"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/gcp/shared/mocks"
	"github.com/overmindtech/cli/sources/shared"
	"github.com/overmindtech/cli/sources/stdlib"
	"github.com/stretchr/testify/assert"
)

func TestBigQueryRoutine(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mocks.NewMockBigQueryRoutineClient(ctrl)
	projectID := "test-project"
	datasetID := "test_dataset"
	routineID := "test_routine"

	t.Run("Get", func(t *testing.T) {
		wrapper := manual.NewBigQueryRoutine(mockClient, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})

		mockClient.EXPECT().Get(ctx, projectID, datasetID, routineID).Return(createRoutineMetadata("test routine"), nil)

		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], shared.CompositeLookupKey(datasetID, routineID), true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != gcpshared.BigQueryRoutine.String() {
			t.Fatalf("Expected type %s, got: %s", gcpshared.BigQueryRoutine.String(), sdpItem.GetType())
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					ExpectedType:   gcpshared.BigQueryDataset.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  datasetID,
					ExpectedScope:  projectID,
				},
				// Imported library GCS bucket link
				{
					ExpectedType:   gcpshared.StorageBucket.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "bucket",
					ExpectedScope:  projectID,
				},
				// Remote function connection link
				{
					ExpectedType:   gcpshared.BigQueryConnection.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "us|example-conn",
					ExpectedScope:  "example",
				},
				// Remote function HTTP endpoint link
				{
					ExpectedType:   stdlib.NetworkHTTP.String(),
					ExpectedMethod: sdp.QueryMethod_SEARCH,
					ExpectedQuery:  "https://example.com/run",
					ExpectedScope:  "global",
				},
			}
			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("Get error", func(t *testing.T) {
		wrapper := manual.NewBigQueryRoutine(mockClient, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())
		mockClient.EXPECT().Get(ctx, projectID, datasetID, routineID).Return(nil, assert.AnError)
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], shared.CompositeLookupKey(datasetID, routineID), true)
		if qErr == nil {
			t.Fatalf("Expected error, got nil")
		}
	})

	t.Run("Search", func(t *testing.T) {
		wrapper := manual.NewBigQueryRoutine(mockClient, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		// Mock the List function to call the converter with each routine
		mockClient.EXPECT().List(
			gomock.Any(),
			projectID,
			datasetID,
			gomock.Any(),
		).DoAndReturn(func(ctx context.Context, projectID string, datasetID string, converter func(routine *bigquery.RoutineMetadata, datasetID, routineID string) (*sdp.Item, *sdp.QueryError)) ([]*sdp.Item, *sdp.QueryError) {
			items := make([]*sdp.Item, 0, 2)

			routine1 := createRoutineMetadata("test routine 1")
			item1, qErr := converter(routine1, datasetID, "routine1")
			if qErr != nil {
				return nil, qErr
			}
			items = append(items, item1)

			routine2 := createRoutineMetadata("test routine 2")
			item2, qErr := converter(routine2, datasetID, "routine2")
			if qErr != nil {
				return nil, qErr
			}
			items = append(items, item2)

			return items, nil
		})

		// Check if adapter supports searching
		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support Search operation")
		}

		sdpItems, err := searchable.Search(ctx, wrapper.Scopes()[0], datasetID, true)
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

	t.Run("Search error", func(t *testing.T) {
		wrapper := manual.NewBigQueryRoutine(mockClient, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		// Mock the List function to call the converter with each routine
		mockClient.EXPECT().List(
			gomock.Any(),
			projectID,
			datasetID,
			gomock.Any(),
		).Return(nil, &sdp.QueryError{ErrorType: sdp.QueryError_OTHER, ErrorString: "test error"})

		// Check if adapter supports searching
		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support Search operation")
		}

		_, err := searchable.Search(ctx, wrapper.Scopes()[0], datasetID, true)
		if err == nil {
			t.Fatalf("Expected error, got nil")
		}
	})

	t.Run("SearchCachesNotFoundWithMemoryCache", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockClient := mocks.NewMockBigQueryRoutineClient(ctrl)
		projectID := "cache-test-project"
		scope := projectID
		datasetID := "empty_dataset"
		query := datasetID

		mockClient.EXPECT().List(gomock.Any(), projectID, datasetID, gomock.Any()).Return([]*sdp.Item{}, nil).Times(1)

		wrapper := manual.NewBigQueryRoutine(mockClient, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
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

	t.Run("Search with terraform format", func(t *testing.T) {
		wrapper := manual.NewBigQueryRoutine(mockClient, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		// Use terraform-style path format
		terraformStyleQuery := "projects/test-project/datasets/test_dataset/routines/test_routine"

		// Mock Get (called internally when terraform format is detected)
		mockClient.EXPECT().Get(ctx, projectID, datasetID, routineID).Return(createRoutineMetadata("terraform format test"), nil)

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support Search operation")
		}

		items, qErr := searchable.Search(ctx, wrapper.Scopes()[0], terraformStyleQuery, true)
		if qErr != nil {
			t.Fatalf("Expected no error with terraform format, got: %v", qErr)
		}
		if len(items) != 1 {
			t.Fatalf("Expected 1 item, got: %d", len(items))
		}
		if items[0].GetType() != gcpshared.BigQueryRoutine.String() {
			t.Fatalf("Expected type %s, got: %s", gcpshared.BigQueryRoutine.String(), items[0].GetType())
		}
	})

	t.Run("Search with legacy pipe format", func(t *testing.T) {
		wrapper := manual.NewBigQueryRoutine(mockClient, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		// Use legacy dataset ID format
		legacyQuery := datasetID

		// Mock the List function
		mockClient.EXPECT().List(
			gomock.Any(),
			projectID,
			datasetID,
			gomock.Any(),
		).DoAndReturn(func(ctx context.Context, projectID string, datasetID string, converter func(routine *bigquery.RoutineMetadata, datasetID, routineID string) (*sdp.Item, *sdp.QueryError)) ([]*sdp.Item, *sdp.QueryError) {
			items := make([]*sdp.Item, 0, 1)
			routine := createRoutineMetadata("legacy format test")
			item, qErr := converter(routine, datasetID, routineID)
			if qErr != nil {
				return nil, qErr
			}
			items = append(items, item)
			return items, nil
		})

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support Search operation")
		}

		items, qErr := searchable.Search(ctx, wrapper.Scopes()[0], legacyQuery, true)
		if qErr != nil {
			t.Fatalf("Expected no error with legacy format, got: %v", qErr)
		}
		if len(items) != 1 {
			t.Fatalf("Expected 1 item, got: %d", len(items))
		}
	})
}

func createRoutineMetadata(description string) *bigquery.RoutineMetadata {
	return &bigquery.RoutineMetadata{
		Type:             bigquery.ScalarFunctionRoutine,
		CreationTime:     time.Unix(1710000000, 0),
		LastModifiedTime: time.Unix(1710003600, 0),
		Language:         "SQL",
		Description:      description,
		Body:             "BEGIN SELECT 1; END;",
		Arguments: []*bigquery.RoutineArgument{
			{
				Name: "input_num",
				Kind: "FIXED_TYPE",
				Mode: "IN",
				DataType: &bigquery.StandardSQLDataType{
					TypeKind: "INT64",
				},
			},
		},
		ReturnType: &bigquery.StandardSQLDataType{
			TypeKind: "INT64",
		},
		DataGovernanceType: string(bigquery.Deterministic),
		ImportedLibraries:  []string{"gs://bucket/lib.js"},
		RemoteFunctionOptions: &bigquery.RemoteFunctionOptions{
			Connection: "projects/example/locations/us/connections/example-conn",
			Endpoint:   "https://example.com/run",
		},
	}
}
