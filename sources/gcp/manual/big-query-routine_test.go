package manual_test

import (
	"context"
	"testing"
	"time"

	"cloud.google.com/go/bigquery"
	"go.uber.org/mock/gomock"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/gcp/manual"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/gcp/shared/mocks"
	"github.com/overmindtech/cli/sources/shared"
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
		wrapper := manual.NewBigQueryRoutine(mockClient, projectID)

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
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				},
			}
			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("Get error", func(t *testing.T) {
		wrapper := manual.NewBigQueryRoutine(mockClient, projectID)
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())
		mockClient.EXPECT().Get(ctx, projectID, datasetID, routineID).Return(nil, assert.AnError)
		_, qErr := adapter.Get(ctx, wrapper.Scopes()[0], shared.CompositeLookupKey(datasetID, routineID), true)
		if qErr == nil {
			t.Fatalf("Expected error, got nil")
		}
	})

	t.Run("Search", func(t *testing.T) {
		wrapper := manual.NewBigQueryRoutine(mockClient, projectID)
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
		wrapper := manual.NewBigQueryRoutine(mockClient, projectID)
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
