package manual

import (
	"context"

	"cloud.google.com/go/bigquery"
	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
	"github.com/overmindtech/cli/sources"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

var (
	BigQueryRoutineLookupByID = shared.NewItemTypeLookup("id", gcpshared.BigQueryRoutine)
)

type BigQueryRoutineWrapper struct {
	client gcpshared.BigQueryRoutineClient

	*gcpshared.ProjectBase
}

func NewBigQueryRoutine(client gcpshared.BigQueryRoutineClient, projectID string) sources.SearchableWrapper {
	return &BigQueryRoutineWrapper{
		client: client,
		ProjectBase: gcpshared.NewProjectBase(
			projectID,
			sdp.AdapterCategory_ADAPTER_CATEGORY_DATABASE,
			gcpshared.BigQueryRoutine),
	}
}

func (b BigQueryRoutineWrapper) IAMPermissions() []string {
	return []string{
		"bigquery.routines.get",
		"bigquery.routines.list",
	}
}

func (b BigQueryRoutineWrapper) PredefinedRole() string {
	return "roles/bigquery.metadataViewer"
}

// PotentialLinks returns the potential links for the BigQuery routine wrapper
func (b BigQueryRoutineWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		gcpshared.BigQueryDataset,
	)
}

// TerraformMappings returns the Terraform mappings for the BigQuery routine wrapper
func (b BigQueryRoutineWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod: sdp.QueryMethod_GET,
			// https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/bigquery_routine
			// projects/{{project}}/datasets/{{dataset_id}}/routines/{{routine_id}}
			TerraformQueryMap: "google_bigquery_routine.routine_id",
		},
	}
}

// GetLookups returns the lookups for the BigQuery routine
func (b BigQueryRoutineWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		BigQueryDatasetLookupByID,
		BigQueryRoutineLookupByID,
	}
}

func (b BigQueryRoutineWrapper) SearchLookups() []sources.ItemTypeLookups {
	return []sources.ItemTypeLookups{

		{
			BigQueryRoutineLookupByID,
		},
	}
}

// Get retrieves a BigQuery routine by its ID
func (b BigQueryRoutineWrapper) Get(ctx context.Context, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	// 0: dataset ID
	// 1: routine ID
	metadata, err := b.client.Get(ctx, b.ProjectID(), queryParts[0], queryParts[1])
	if err != nil {
		return nil, gcpshared.QueryError(err, b.DefaultScope(), b.Type())
	}
	return b.GCPBigQueryRoutineToItem(metadata, queryParts[0], queryParts[1])
}

func (b BigQueryRoutineWrapper) Search(ctx context.Context, queryParts ...string) ([]*sdp.Item, *sdp.QueryError) {
	items, err := b.client.List(ctx, b.ProjectID(), queryParts[0], b.GCPBigQueryRoutineToItem)
	if err != nil {
		return nil, gcpshared.QueryError(err, b.DefaultScope(), b.Type())
	}

	return items, nil
}

func (b BigQueryRoutineWrapper) SearchStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, queryParts ...string) {
}

func (b BigQueryRoutineWrapper) GCPBigQueryRoutineToItem(metadata *bigquery.RoutineMetadata, datasetID, routineID string) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(metadata, "")
	if err != nil {
		return nil, gcpshared.QueryError(err, b.DefaultScope(), b.Type())
	}

	err = attributes.Set("id", shared.CompositeLookupKey(datasetID, routineID))
	if err != nil {
		return nil, gcpshared.QueryError(err, b.DefaultScope(), b.Type())
	}

	sdpItem := &sdp.Item{
		Type:            gcpshared.BigQueryRoutine.String(),
		UniqueAttribute: "id",
		Attributes:      attributes,
		Scope:           b.DefaultScope(),
		Tags:            make(map[string]string),
	}

	sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   gcpshared.BigQueryDataset.String(),
			Method: sdp.QueryMethod_GET,
			Query:  datasetID,
			Scope:  b.ProjectID(),
		},
		BlastPropagation: &sdp.BlastPropagation{
			In:  true,
			Out: true,
		},
	})

	//NOTE: optional feature for the future - parse routine_definition to identify referenced tables/views/connections and add links. Out-of-scope for initial version.

	return sdpItem, nil
}
