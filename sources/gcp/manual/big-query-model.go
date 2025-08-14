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

func (m BigQueryModelWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		BigQueryDatasetLookupByID,
		BigQueryModelLookupById,
	}
}

func (m BigQueryModelWrapper) Get(ctx context.Context, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	metadata, err := m.client.Get(ctx, m.ProjectBase.ProjectID(), queryParts[0], queryParts[1])
	if err != nil {
		return nil, gcpshared.QueryError(err)
	}
	return m.GCPBigQueryMetadataToItem(queryParts[0], metadata)
}

func (m BigQueryModelWrapper) GCPBigQueryMetadataToItem(dataSetId string, metadata *bigquery.ModelMetadata) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(metadata, "labels")
	if err != nil {
		return nil, gcpshared.QueryError(err)
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
		if row.DataSplitResult != nil && row.DataSplitResult.EvaluationTable.TableId != "" {
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   gcpshared.BigQueryTable.String(),
					Method: sdp.QueryMethod_GET,
					Scope:  m.DefaultScope(),
					Query:  shared.CompositeLookupKey(dataSetId, row.DataSplitResult.EvaluationTable.TableId),
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: false,
				},
			})
		}
	}

	return sdpItem, nil
}

func (m BigQueryModelWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		gcpshared.CloudKMSCryptoKey,
		gcpshared.BigQueryDataset,
		gcpshared.BigQueryTable,
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
	items, err := m.client.List(ctx, m.ProjectBase.ProjectID(), queryParts[0], m.GCPBigQueryMetadataToItem)
	if err != nil {
		return nil, gcpshared.QueryError(err)
	}
	return items, nil
}

func (m BigQueryModelWrapper) SearchStream(ctx context.Context, stream discovery.QueryResultStream, cache *sdpcache.Cache, cacheKey sdpcache.CacheKey, queryParts ...string) {
	m.client.ListStream(ctx, m.ProjectBase.ProjectID(), queryParts[0], stream, func(datasetID string, md *bigquery.ModelMetadata) (*sdp.Item, *sdp.QueryError) {
		item, qerr := m.GCPBigQueryMetadataToItem(datasetID, md)
		if qerr == nil && item != nil {
			cache.StoreItem(item, shared.DefaultCacheDuration, cacheKey)
		}
		return item, qerr
	})
}
