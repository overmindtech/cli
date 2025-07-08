package integrationtests

import (
	"context"
	"os"
	"strings"
	"testing"

	"cloud.google.com/go/bigquery"
	"go.uber.org/mock/gomock"

	"github.com/overmindtech/cli/sources/gcp/manual"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

func TestBigQueryModel(t *testing.T) {
	projectID := os.Getenv("GCP_PROJECT_ID")
	if projectID == "" {
		t.Skip("GCP_PROJECT_ID environment variable is not set, skipping BigQuery model tests")
	}

	dataSet := "test_dataset"
	model := "test_model"

	ctx := context.Background()
	ctrl := gomock.NewController(t)
	client, err := bigquery.NewClient(ctx, projectID)
	if err != nil {
		t.Fatalf("Failed to create BigQuery client: %v", err)
	}
	defer client.Close()

	defer ctrl.Finish()
	t.Run("Setup", func(t *testing.T) {
		datasetItem := client.Dataset(dataSet)
		err := datasetItem.Create(ctx, &bigquery.DatasetMetadata{
			Name:        dataSet,
			Description: "Test dataset for model integration tests",
		})
		if err != nil && !strings.Contains(err.Error(), "Already Exists") {
			t.Fatalf("Failed to create dataset %s: %v", dataSet, err)
		}
		t.Logf("Dataset %s created successfully", dataSet)

		query := "CREATE OR REPLACE MODEL `" + projectID + "." + dataSet + "." + model + "` OPTIONS " +
			`(model_type='LOGISTIC_REG',
             labels=['animal_label']
             ) AS
            SELECT
              1 AS feature_dummy, -- A dummy feature for 'cats'
              'cats' AS animal_label -- The primary label we want to output
            UNION ALL
            SELECT
              2 AS feature_dummy, -- A different dummy feature for the second label
              'dogs' AS animal_label; -- A second, dummy label to satisfy the classification requirement`

		op, err := client.Query(query).Run(ctx)
		if err != nil {
			t.Fatalf("Failed to create model: %v", err)
		}
		if _, err := op.Wait(ctx); err != nil {
			t.Fatalf("Failed to wait for model creation: %v", err)
		}
		modelItem := client.Dataset(dataSet).Model(model)
		modelMetadata, err := modelItem.Update(ctx, bigquery.ModelMetadataToUpdate{
			Name:        model,
			Description: "Test model description",
		}, "")
		if err != nil {
			t.Fatalf("Failed to create model: %v", err)
		}
		t.Logf("Model created: %s", modelMetadata.Name)
	})
	t.Run("Get", func(t *testing.T) {
		bigqueryClient := gcpshared.NewBigQueryModelClient(client, dataSet)
		adapter := manual.NewBigQueryModel(bigqueryClient, projectID)
		sdpItem, err := adapter.Get(ctx, dataSet, model)
		if err != nil {
			t.Fatalf("Failed to get item: %v", err)
		}
		if sdpItem == nil {
			t.Fatal("Expected an item, got nil")
		}
		uniqueAttrKey := sdpItem.GetUniqueAttribute()

		uniqueAttrValue, attrErr := sdpItem.GetAttributes().Get(uniqueAttrKey)
		if attrErr != nil {
			t.Fatalf("Failed to get unique attribute: %v", err)
		}

		if uniqueAttrValue != model {
			t.Fatalf("Expected unique attribute value to be %s, got %s", model, uniqueAttrValue)
		}

		sdpItems, err := adapter.Search(ctx, dataSet)
		if err != nil {
			t.Fatalf("Failed to search items: %v", err)
		}
		if len(sdpItems) < 1 {
			t.Fatalf("Expected at least one model in dataset, got %d", len(sdpItems))
		}

		var found bool
		for _, item := range sdpItems {
			if v, err := item.GetAttributes().Get(uniqueAttrKey); err == nil && v == model {
				found = true
				break
			}
		}

		if !found {
			t.Fatalf("Expected to find model %s in the list of dataset models", model)
		}
	})
	t.Run("Teardown", func(t *testing.T) {
		// Cleanup resources if needed
		err := client.Dataset(dataSet).DeleteWithContents(ctx)
		if err != nil {
			t.Fatalf("Failed to delete dataset %s: %v", dataSet, err)
		} else {
			t.Logf("Dataset %s deleted successfully", dataSet)
		}
	})
}
