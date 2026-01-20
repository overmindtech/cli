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

var BigQueryModelLookupById = shared.NewItemTypeLookup("id", gcpshared.BigQueryModel)

// BigQueryModelWrapper is a wrapper for the BigQueryModelClient that implements the sources.SearchableWrapper interface
type BigQueryModelWrapper struct {
	client gcpshared.BigQueryModelClient
	*gcpshared.ProjectBase
}

// NewBigQueryModel creates a new BigQueryModelWrapper instance
func NewBigQueryModel(client gcpshared.BigQueryModelClient, locations []gcpshared.LocationInfo) sources.SearchStreamableWrapper {
	return &BigQueryModelWrapper{
		client: client,
		ProjectBase: gcpshared.NewProjectBase(
			locations,
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

func (m BigQueryModelWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	location, err := m.LocationFromScope(scope)
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: err.Error(),
		}
	}

	metadata, err := m.client.Get(ctx, location.ProjectID, queryParts[0], queryParts[1])
	if err != nil {
		return nil, gcpshared.QueryError(err, scope, m.Type())
	}
	return m.GCPBigQueryMetadataToItem(ctx, location, queryParts[0], metadata)
}

func (m BigQueryModelWrapper) GCPBigQueryMetadataToItem(ctx context.Context, location gcpshared.LocationInfo, dataSetId string, metadata *bigquery.ModelMetadata) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(metadata, "labels")
	if err != nil {
		return nil, gcpshared.QueryError(err, location.ToScope(), m.Type())
	}

	sdpItem := &sdp.Item{
		Type:            gcpshared.BigQueryModel.String(),
		UniqueAttribute: "Name",
		Attributes:      attributes,
		Scope:           location.ToScope(),
		Tags:            metadata.Labels,
	}

	sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   gcpshared.BigQueryDataset.String(),
			Method: sdp.QueryMethod_GET,
			Scope:  location.ProjectID,
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
					Scope:  location.ProjectID,
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
						Scope:  location.ProjectID,
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
						Scope:  location.ProjectID,
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
						Scope:  location.ProjectID,
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

func (m BigQueryModelWrapper) Search(ctx context.Context, scope string, queryParts ...string) ([]*sdp.Item, *sdp.QueryError) {
	location, err := m.LocationFromScope(scope)
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: err.Error(),
		}
	}

	items, listErr := m.client.List(ctx, location.ProjectID, queryParts[0], func(datasetID string, md *bigquery.ModelMetadata) (*sdp.Item, *sdp.QueryError) {
		return m.GCPBigQueryMetadataToItem(ctx, location, datasetID, md)
	})
	return items, listErr
}

func (m BigQueryModelWrapper) SearchStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string, queryParts ...string) {
	location, err := m.LocationFromScope(scope)
	if err != nil {
		stream.SendError(&sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: err.Error(),
		})
		return
	}

	m.client.ListStream(ctx, location.ProjectID, queryParts[0], stream, func(datasetID string, md *bigquery.ModelMetadata) (*sdp.Item, *sdp.QueryError) {
		item, qerr := m.GCPBigQueryMetadataToItem(ctx, location, datasetID, md)
		if qerr == nil && item != nil {
			cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
		}
		return item, qerr
	})
}
