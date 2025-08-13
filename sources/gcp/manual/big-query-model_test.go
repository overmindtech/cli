package manual_test

import (
	"context"
	"testing"

	bigquery "cloud.google.com/go/bigquery"
	"go.uber.org/mock/gomock"

	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/gcp/manual"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/gcp/shared/mocks"
	"github.com/overmindtech/cli/sources/shared"
)

func TestBigQueryModel(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mocks.NewMockBigQueryModelClient(ctrl)
	projectID := "test-project"
	datasetID := "test_dataset"
	modelName := "test_model"

	t.Run("Get", func(t *testing.T) {
		wrapper := manual.NewBigQueryModel(mockClient, projectID)

		mockClient.EXPECT().Get(ctx, projectID, datasetID, modelName).Return(createDatasetModel(projectID, modelName), nil)

		adapter := sources.WrapperToAdapter(wrapper)

		query := shared.CompositeLookupKey(datasetID, modelName)

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], query, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}
		if sdpItem == nil {
			t.Fatal("Expected an item, got nil")
		}

		// Cannot test for linked table as you cannot set the model metadata training runs.
		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
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
			}
			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("Search", func(t *testing.T) {
		wrapper := manual.NewBigQueryModel(mockClient, projectID)
		mockClient.EXPECT().List(ctx, projectID, datasetID, gomock.Any()).Return([]*sdp.Item{
			{},
		}, nil)

		adapter := sources.WrapperToAdapter(wrapper)

		sdpItems, qErr := adapter.Search(ctx, wrapper.Scopes()[0], datasetID, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}
		if len(sdpItems) != 1 {
			t.Fatalf("Expected 1 items, got: %d", len(sdpItems))
		}
	})
}

func createDatasetModel(projectID, modelName string) *bigquery.ModelMetadata {
	model := &bigquery.ModelMetadata{
		Name: modelName,
		Type: "LINEAR_REGRESSION",
		Labels: map[string]string{
			"env": "test",
		},
		Location: "US",
		ETag:     "etag123",

		Description: "Test model description",
		EncryptionConfig: &bigquery.EncryptionConfig{
			KMSKeyName: "projects/" + projectID + "/locations/global/keyRings/test-ring/cryptoKeys/test-key",
		},
	}

	return model
}
