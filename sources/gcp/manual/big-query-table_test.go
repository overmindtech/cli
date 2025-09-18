package manual_test

import (
	"context"
	"fmt"
	"testing"

	"cloud.google.com/go/bigquery"
	"go.uber.org/mock/gomock"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/gcp/manual"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/gcp/shared/mocks"
	"github.com/overmindtech/cli/sources/shared"
)

func TestBigQueryTable(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mocks.NewMockBigQueryTableClient(ctrl)
	projectID := "test-project"
	datasetID := "test_dataset"
	tableID := "test_table"

	t.Run("Get", func(t *testing.T) {
		wrapper := manual.NewBigQueryTable(mockClient, projectID)

		mockClient.EXPECT().Get(ctx, projectID, datasetID, tableID).Return(createTableMetadata(projectID, datasetID, tableID, projectID+".us;test-connection"), nil)

		adapter := sources.WrapperToAdapter(wrapper)

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], shared.CompositeLookupKey(datasetID, tableID), true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != gcpshared.BigQueryTable.String() {
			t.Fatalf("Expected type %s, got: %s", gcpshared.BigQueryTable.String(), sdpItem.GetType())
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
				{
					ExpectedType:   gcpshared.CloudKMSCryptoKey.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey("us", "test-ring", "test-key"),
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					ExpectedType:   gcpshared.BigQueryConnection.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey("us", "test-connection"),
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

	t.Run("Get with alternative connection id", func(t *testing.T) {
		wrapper := manual.NewBigQueryTable(mockClient, projectID)

		mockClient.EXPECT().Get(ctx, projectID, datasetID, tableID).Return(createTableMetadata(projectID, datasetID, tableID, fmt.Sprintf("projects/%s/locations/us/connections/test-connection", projectID)), nil)

		adapter := sources.WrapperToAdapter(wrapper)

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], shared.CompositeLookupKey(datasetID, tableID), true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem.GetType() != gcpshared.BigQueryTable.String() {
			t.Fatalf("Expected type %s, got: %s", gcpshared.BigQueryTable.String(), sdpItem.GetType())
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
				{
					ExpectedType:   gcpshared.CloudKMSCryptoKey.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey("us", "test-ring", "test-key"),
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					ExpectedType:   gcpshared.BigQueryConnection.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey("us", "test-connection"),
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

	t.Run("Search", func(t *testing.T) {
		wrapper := manual.NewBigQueryTable(mockClient, projectID)
		adapter := sources.WrapperToAdapter(wrapper)

		// Mock the List function to call the converter with each table
		mockClient.EXPECT().List(
			gomock.Any(),
			projectID,
			datasetID,
			gomock.Any(),
		).DoAndReturn(func(ctx context.Context, projectID, datasetID string, converter func(*bigquery.TableMetadata) (*sdp.Item, *sdp.QueryError)) ([]*sdp.Item, *sdp.QueryError) {
			items := make([]*sdp.Item, 0, 2)

			table1 := createTableMetadata(projectID, datasetID, "table1", projectID+".us;test-connection")
			item1, qErr := converter(table1)
			if qErr != nil {
				return nil, qErr
			}
			items = append(items, item1)

			table2 := createTableMetadata(projectID, datasetID, "table2", projectID+".us;test-connection")
			item2, qErr := converter(table2)
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

	t.Run("SearchWithTerraformMapping", func(t *testing.T) {
		wrapper := manual.NewBigQueryTable(mockClient, projectID)
		adapter := sources.WrapperToAdapter(wrapper)

		// Mock the List function to call the converter with each table
		mockClient.EXPECT().Get(ctx, projectID, datasetID, tableID).
			Return(createTableMetadata(
				projectID,
				datasetID,
				tableID,
				fmt.Sprintf("projects/%s/locations/us/connections/test-connection", projectID),
			), nil)

		terraformMapping := fmt.Sprintf("projects/%s/datasets/%s/tables/%s", projectID, datasetID, tableID)
		// Check if adapter supports searching
		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support Search operation")
		}

		sdpItems, err := searchable.Search(ctx, wrapper.Scopes()[0], terraformMapping, true)
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

	t.Run("List_Unsupported", func(t *testing.T) {
		wrapper := manual.NewBigQueryTable(mockClient, projectID)
		adapter := sources.WrapperToAdapter(wrapper)

		// Check if adapter supports list - it should not
		_, ok := adapter.(discovery.ListableAdapter)
		if ok {
			t.Fatalf("Expected adapter to not support List operation, but it does")
		}

		// Check if adapter supports ListStream - it should not
		_, ok = adapter.(discovery.ListStreamableAdapter)
		if ok {
			t.Fatalf("Adapter should not support ListStream operation")
		}
	})
}

// createTableMetadata creates a BigQuery TableMetadata for testing.
func createTableMetadata(projectID, datasetID, tableID, connectionID string) *bigquery.TableMetadata {
	return &bigquery.TableMetadata{
		Name:     tableID,
		FullID:   projectID + ":" + datasetID + "." + tableID,
		Type:     "TABLE",
		Location: "US",
		Labels:   map[string]string{"env": "test"},
		EncryptionConfig: &bigquery.EncryptionConfig{
			KMSKeyName: "projects/" + projectID + "/locations/us/keyRings/test-ring/cryptoKeys/test-key",
		},
		ExternalDataConfig: &bigquery.ExternalDataConfig{
			ConnectionID: connectionID,
		},
	}
}
