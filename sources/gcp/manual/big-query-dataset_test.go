package manual_test

import (
	"context"
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

func TestBigQueryDataset(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mocks.NewMockBigQueryDatasetClient(ctrl)
	projectID := "test-project"
	datasetID := "test_dataset"

	t.Run("Get", func(t *testing.T) {
		wrapper := manual.NewBigQueryDataset(mockClient, projectID)

		mockClient.EXPECT().Get(ctx, projectID, datasetID).Return(createDataset(projectID, datasetID), nil)

		adapter := sources.WrapperToAdapter(wrapper)

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], datasetID, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					ExpectedType:   gcpshared.IAMServiceAccount.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-user@example.com",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					ExpectedType:   gcpshared.CloudKMSCryptoKey.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey("global", "test-ring", "test-key"),
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					ExpectedType:   gcpshared.BigQueryDataset.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  datasetID,
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  false,
						Out: true,
					},
				},
				{
					ExpectedType:   gcpshared.BigQueryConnection.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey("global", "test-connection"),
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

	t.Run("List", func(t *testing.T) {
		wrapper := manual.NewBigQueryDataset(mockClient, projectID)
		adapter := sources.WrapperToAdapter(wrapper)

		mockClient.EXPECT().List(ctx, projectID, gomock.Any()).Return([]*sdp.Item{
			{},
			{},
		}, nil)

		// Check if adapter supports listing
		listable, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support List operation")
		}

		sdpItems, err := listable.List(ctx, wrapper.Scopes()[0], true)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		expectedCount := 2
		actualCount := len(sdpItems)
		if actualCount != expectedCount {
			t.Fatalf("Expected %d items, got: %d", expectedCount, actualCount)
		}

		_, ok = adapter.(discovery.SearchableAdapter)
		if ok {
			t.Fatalf("Expected adapter to not support Search operation, but it does")
		}

		_, ok = adapter.(discovery.SearchStreamableAdapter)
		if ok {
			t.Fatalf("Adapter should not support SearchStream operation")
		}
	})
}

// createDataset creates a BigQuery Dataset for testing.
func createDataset(projectID, datasetID string) *bigquery.DatasetMetadata {
	return &bigquery.DatasetMetadata{
		Name:        datasetID,
		FullID:      projectID + ":" + datasetID,
		Location:    "EU",
		Description: "Test dataset for unit tests",
		Labels: map[string]string{
			"env": "test",
		},
		Access: []*bigquery.AccessEntry{
			{
				Role:       bigquery.ReaderRole,
				EntityType: bigquery.UserEmailEntity,
				Entity:     "test-user@example.com",
				Dataset: &bigquery.DatasetAccessEntry{
					Dataset: &bigquery.Dataset{
						ProjectID: projectID,
						DatasetID: datasetID,
					},
				},
			},
		},
		DefaultEncryptionConfig: &bigquery.EncryptionConfig{
			KMSKeyName: "projects/" + projectID + "/locations/global/keyRings/test-ring/cryptoKeys/test-key",
		},
		ExternalDatasetReference: &bigquery.ExternalDatasetReference{
			// projects/{projectId}/locations/{locationId}/connections/{connectionId}
			Connection: "projects/" + projectID + "/locations/global/connections/test-connection",
		},
	}
}
