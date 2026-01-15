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
	BigQueryModelLookupById = shared.NewItemTypeLookup("id", gcpshared.BigQueryModel)
)

// BigQueryModelWrapper is a wrapper for the BigQueryModelClient that implements the sources.SearchableWrapper interface
type BigQueryModelWrapper struct {
	client gcpshared.BigQueryModelClient
	*gcpshared.ProjectBase
}

// NewBigQueryModel creates a new BigQueryModelWrapper instance
func NewBigQueryModel(client gcpshared.BigQueryModelClient, projectID string) sources.SearchableWrapper {
	return &BigQueryModelWrapper{
		client: client,
		ProjectBase: gcpshared.NewProjectBase(
			projectID,
			sdp.AdapterCategory_ADAPTER_CATEGORY_DATABASE,
			gcpshared.BigQueryModel,
		),
	}
}

func (m BigQueryModelWrapper) IAMPermissions() []string {
	return []string{
		"bigquery.models.getMetadata",
		"bigquery.models.list",
	}
}

func (m BigQueryModelWrapper) PredefinedRole() string {
	// https://cloud.google.com/iam/docs/roles-permissions/bigquery#bigquery.metadataViewer
	return "roles/bigquery.metadataViewer"
}

func (m BigQueryModelWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		BigQueryDatasetLookupByID,
		BigQueryModelLookupById,
	}
}

func (m BigQueryModelWrapper) Get(ctx context.Context, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	metadata, err := m.client.Get(ctx, m.ProjectBase.ProjectID(), queryParts[0], queryParts[1])
	if err != nil {
		return nil, gcpshared.QueryError(err, m.DefaultScope(), m.Type())
	}
	return m.GCPBigQueryMetadataToItem(ctx, queryParts[0], metadata)
}

func (m BigQueryModelWrapper) GCPBigQueryMetadataToItem(ctx context.Context, dataSetId string, metadata *bigquery.ModelMetadata) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(metadata, "labels")
	if err != nil {
		return nil, gcpshared.QueryError(err, m.DefaultScope(), m.Type())
	}

	sdpItem := &sdp.Item{
		Type:            gcpshared.BigQueryModel.String(),
		UniqueAttribute: "Name",
		Attributes:      attributes,
		Scope:           m.DefaultScope(),
		Tags:            metadata.Labels,
	}

	sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   gcpshared.BigQueryDataset.String(),
			Method: sdp.QueryMethod_GET,
			Scope:  m.DefaultScope(),
			Query:  dataSetId,
		},
		// Model is in a dataset, if dataset is deleted, model is deleted.
		// If the model is deleted, the dataset is not deleted.
		BlastPropagation: &sdp.BlastPropagation{
			In:  false,
			Out: true,
		},
	})

	if metadata.EncryptionConfig != nil && metadata.EncryptionConfig.KMSKeyName != "" {
		values := gcpshared.ExtractPathParams(metadata.EncryptionConfig.KMSKeyName, "locations", "keyRings", "cryptoKeys")
		if len(values) == 3 && values[0] != "" && values[1] != "" && values[2] != "" {
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{

				Query: &sdp.Query{
					Type:   gcpshared.CloudKMSCryptoKey.String(),
					Method: sdp.QueryMethod_GET,
					Scope:  m.ProjectID(),
					Query:  shared.CompositeLookupKey(values...),
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: false,
				},
			})
		}
	}

	for _, row := range metadata.RawTrainingRuns() {
		if row.DataSplitResult != nil {
			// Link to evaluation table (already existed)
			if row.DataSplitResult.EvaluationTable != nil && row.DataSplitResult.EvaluationTable.TableId != "" {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   gcpshared.BigQueryTable.String(),
						Method: sdp.QueryMethod_GET,
						Scope:  m.DefaultScope(),
						Query:  shared.CompositeLookupKey(dataSetId, row.DataSplitResult.EvaluationTable.TableId),
					},
					// If the evaluation table is deleted or updated: The model's evaluation results may become invalid or inaccessible. If the model is updated: The table remains unaffected.
					BlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				})
			}

			// Link to training table
			if row.DataSplitResult.TrainingTable != nil && row.DataSplitResult.TrainingTable.TableId != "" {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   gcpshared.BigQueryTable.String(),
						Method: sdp.QueryMethod_GET,
						Scope:  m.DefaultScope(),
						Query:  shared.CompositeLookupKey(dataSetId, row.DataSplitResult.TrainingTable.TableId),
					},
					// If the training table is deleted or updated: The model's training data may become invalid or inaccessible. If the model is updated: The table remains unaffected.
					BlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				})
			}

			// Link to test table
			if row.DataSplitResult.TestTable != nil && row.DataSplitResult.TestTable.TableId != "" {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   gcpshared.BigQueryTable.String(),
						Method: sdp.QueryMethod_GET,
						Scope:  m.DefaultScope(),
						Query:  shared.CompositeLookupKey(dataSetId, row.DataSplitResult.TestTable.TableId),
					},
					// If the test table is deleted or updated: The model's test results may become invalid or inaccessible. If the model is updated: The table remains unaffected.
					BlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				})
			}
		}
	}

	// TODO: Link to BigQuery Connection and Vertex AI Endpoint for remote models
	// RemoteModelInfo (containing connection and endpoint fields) is not directly accessible
	// in the Go SDK's ModelMetadata struct. To implement these links, we would need to:
	// 1. Use the REST API directly to fetch model metadata, or
	// 2. Wait for the Go SDK to expose RemoteModelInfo fields, or
	// 3. Access the raw JSON response if available
	// Connection format: projects/{projectId}/locations/{locationId}/connections/{connectionId}
	// Endpoint format: https://{location}-aiplatform.googleapis.com/v1/projects/{project}/locations/{location}/endpoints/{endpoint_id}

	return sdpItem, nil
}

func (m BigQueryModelWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		gcpshared.CloudKMSCryptoKey,
		gcpshared.BigQueryDataset,
		gcpshared.BigQueryTable,
		gcpshared.BigQueryConnection,
		gcpshared.AIPlatformEndpoint,
	)
}

func (m BigQueryModelWrapper) SearchLookups() []sources.ItemTypeLookups {
	return []sources.ItemTypeLookups{
		{
			BigQueryModelLookupById,
		},
	}
}

func (m BigQueryModelWrapper) Search(ctx context.Context, queryParts ...string) ([]*sdp.Item, *sdp.QueryError) {
	items, err := m.client.List(ctx, m.ProjectBase.ProjectID(), queryParts[0], func(datasetID string, md *bigquery.ModelMetadata) (*sdp.Item, *sdp.QueryError) {
		return m.GCPBigQueryMetadataToItem(ctx, datasetID, md)
	})
	if err != nil {
		return nil, gcpshared.QueryError(err, m.DefaultScope(), m.Type())
	}
	return items, nil
}

func (m BigQueryModelWrapper) SearchStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, queryParts ...string) {
	m.client.ListStream(ctx, m.ProjectBase.ProjectID(), queryParts[0], stream, func(datasetID string, md *bigquery.ModelMetadata) (*sdp.Item, *sdp.QueryError) {
		item, qerr := m.GCPBigQueryMetadataToItem(ctx, datasetID, md)
		if qerr == nil && item != nil {
			cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
		}
		return item, qerr
	})
}
