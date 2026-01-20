package integrationtests

import (
	"context"
	"os"
	"strings"
	"testing"

	"cloud.google.com/go/bigquery"
	"go.uber.org/mock/gomock"

	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/gcp/manual"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

func TestBigQueryModel(t *testing.T) {
	projectID := os.Getenv("GCP_PROJECT_ID")
	if projectID == "" {
		t.Skip("GCP_PROJECT_ID environment variable is not set, skipping BigQuery model tests")
	}
	t.Parallel()

	dataSet := "test_dataset"
	model := "test_model"
	routine := "test_routine"
	table := "test_table"

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

		routineQuery := "CREATE OR REPLACE FUNCTION `" + projectID + "." + dataSet + "." + routine + "`(input INT64)\n" +
			"RETURNS INT64\n" +
			"AS (\n" +
			"  input + 1\n" +
			");"

		routineOp, err := client.Query(routineQuery).Run(ctx)
		if err != nil {
			t.Fatalf("Failed to create routine: %v", err)
		}
		if _, err := routineOp.Wait(ctx); err != nil {
			t.Fatalf("Failed to wait for routine creation: %v", err)
		}

		routineItem := client.Dataset(dataSet).Routine(routine)
		if _, err := routineItem.Metadata(ctx); err != nil {
			t.Fatalf("Failed to retrieve routine metadata: %v", err)
		}
		t.Logf("Routine created: %s", routine)

		tableItem := client.Dataset(dataSet).Table(table)
		err = tableItem.Create(ctx, &bigquery.TableMetadata{
			Name:        table,
			Description: "Test table for integration tests",
			Schema: bigquery.Schema{
				{Name: "id", Type: bigquery.IntegerFieldType, Required: true},
				{Name: "name", Type: bigquery.StringFieldType},
			},
		})
		if err != nil && !strings.Contains(err.Error(), "Already Exists") {
			t.Fatalf("Failed to create table %s: %v", table, err)
		}
		if _, err := tableItem.Metadata(ctx); err != nil {
			t.Fatalf("Failed to retrieve table metadata: %v", err)
		}
		t.Logf("Table created: %s", table)
	})
	t.Run("Get", func(t *testing.T) {
		bigqueryClient := gcpshared.NewBigQueryModelClient(client)
		adapter := manual.NewBigQueryModel(bigqueryClient, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		sdpItem, err := adapter.Get(ctx, adapter.Scopes()[0], dataSet, model)
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

		searchable, ok := adapter.(sources.SearchableWrapper)
		if !ok {
			t.Fatalf("Expected adapter to support search")
		}

		sdpItems, err := searchable.Search(ctx, adapter.Scopes()[0], dataSet)
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
	t.Run("GetRoutine", func(t *testing.T) {
		routineClient := gcpshared.NewBigQueryRoutineClient(client)
		adapter := manual.NewBigQueryRoutine(routineClient, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})

		sdpItem, err := adapter.Get(ctx, adapter.Scopes()[0], dataSet, routine)
		if err != nil {
			t.Fatalf("Failed to get routine: %v", err)
		}
		if sdpItem == nil {
			t.Fatal("Expected a routine item, got nil")
		}

		uniqueAttrKey := sdpItem.GetUniqueAttribute()
		uniqueAttrValue, attrErr := sdpItem.GetAttributes().Get(uniqueAttrKey)
		if attrErr != nil {
			t.Fatalf("Failed to get routine unique attribute: %v", attrErr)
		}

		expectedUniqueAttrValue := shared.CompositeLookupKey(dataSet, routine)
		if uniqueAttrValue != expectedUniqueAttrValue {
			t.Fatalf("Expected routine unique attribute value to be %s, got %v", expectedUniqueAttrValue, uniqueAttrValue)
		}

		searchable, ok := adapter.(sources.SearchableWrapper)
		if !ok {
			t.Fatalf("Expected adapter to support search")
		}

		sdpItems, err := searchable.Search(ctx, adapter.Scopes()[0], dataSet)
		if err != nil {
			t.Fatalf("Failed to search routines: %v", err)
		}
		if len(sdpItems) < 1 {
			t.Fatalf("Expected at least one routine in dataset, got %d", len(sdpItems))
		}

		var found bool
		for _, item := range sdpItems {
			if v, err := item.GetAttributes().Get(uniqueAttrKey); err == nil && v == expectedUniqueAttrValue {
				found = true
				break
			}
		}

		if !found {
			t.Fatalf("Expected to find routine %s in the list of dataset routines", routine)
		}
	})

	t.Run("GetDataset", func(t *testing.T) {
		datasetClient := gcpshared.NewBigQueryDatasetClient(client)
		adapter := manual.NewBigQueryDataset(datasetClient, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})

		sdpItem, err := adapter.Get(ctx, adapter.Scopes()[0], dataSet)
		if err != nil {
			t.Fatalf("Failed to get dataset: %v", err)
		}
		if sdpItem == nil {
			t.Fatal("Expected a dataset item, got nil")
		}

		expectedScope := projectID
		modelLinkFound := false
		routineLinkFound := false
		tableLinkFound := false
		for _, linkedItem := range sdpItem.GetLinkedItemQueries() {
			query := linkedItem.GetQuery()
			if query == nil {
				continue
			}

			switch query.GetType() {
			case gcpshared.BigQueryModel.String():
				if query.GetMethod() != sdp.QueryMethod_SEARCH {
					t.Fatalf("Expected model link method to be %s, got %s", sdp.QueryMethod_SEARCH, query.GetMethod())
				}
				if query.GetQuery() != dataSet {
					t.Fatalf("Expected model link query to be %s, got %s", dataSet, query.GetQuery())
				}
				if query.GetScope() != expectedScope {
					t.Fatalf("Expected model link scope to be %s, got %s", expectedScope, query.GetScope())
				}
				modelLinkFound = true
			case gcpshared.BigQueryRoutine.String():
				if query.GetMethod() != sdp.QueryMethod_SEARCH {
					t.Fatalf("Expected routine link method to be %s, got %s", sdp.QueryMethod_SEARCH, query.GetMethod())
				}
				if query.GetQuery() != dataSet {
					t.Fatalf("Expected routine link query to be %s, got %s", dataSet, query.GetQuery())
				}
				if query.GetScope() != expectedScope {
					t.Fatalf("Expected routine link scope to be %s, got %s", expectedScope, query.GetScope())
				}
				routineLinkFound = true
			case gcpshared.BigQueryTable.String():
				if query.GetMethod() != sdp.QueryMethod_SEARCH {
					t.Fatalf("Expected table link method to be %s, got %s", sdp.QueryMethod_SEARCH, query.GetMethod())
				}
				if query.GetQuery() != dataSet {
					t.Fatalf("Expected table link query to be %s, got %s", dataSet, query.GetQuery())
				}
				if query.GetScope() != expectedScope {
					t.Fatalf("Expected table link scope to be %s, got %s", expectedScope, query.GetScope())
				}
				tableLinkFound = true
			}
		}

		if !modelLinkFound {
			t.Fatalf("Expected dataset %s to include a link to its models", dataSet)
		}
		if !routineLinkFound {
			t.Fatalf("Expected dataset %s to include a link to its routines", dataSet)
		}
		if !tableLinkFound {
			t.Fatalf("Expected dataset %s to include a link to its tables", dataSet)
		}
	})
	t.Run("GetTable", func(t *testing.T) {
		tableClient := gcpshared.NewBigQueryTableClient(client)
		adapter := manual.NewBigQueryTable(tableClient, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})

		sdpItem, err := adapter.Get(ctx, adapter.Scopes()[0], dataSet, table)
		if err != nil {
			t.Fatalf("Failed to get table: %v", err)
		}
		if sdpItem == nil {
			t.Fatal("Expected a table item, got nil")
		}

		uniqueAttrKey := sdpItem.GetUniqueAttribute()
		uniqueAttrValue, attrErr := sdpItem.GetAttributes().Get(uniqueAttrKey)
		if attrErr != nil {
			t.Fatalf("Failed to get table unique attribute: %v", attrErr)
		}
		expectedUniqueAttrValue := shared.CompositeLookupKey(dataSet, table)
		if uniqueAttrValue != expectedUniqueAttrValue {
			t.Fatalf("Expected table unique attribute value to be %s, got %v", expectedUniqueAttrValue, uniqueAttrValue)
		}

		searchable, ok := adapter.(sources.SearchableWrapper)
		if !ok {
			t.Fatalf("Expected adapter to support search")
		}

		sdpItems, err := searchable.Search(ctx, adapter.Scopes()[0], dataSet)
		if err != nil {
			t.Fatalf("Failed to search tables: %v", err)
		}
		if len(sdpItems) < 1 {
			t.Fatalf("Expected at least one table in dataset, got %d", len(sdpItems))
		}

		found := false
		for _, item := range sdpItems {
			if v, err := item.GetAttributes().Get(uniqueAttrKey); err == nil && v == expectedUniqueAttrValue {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("Expected to find table %s in the list of dataset tables", table)
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
