package manual

import (
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/bigquery"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
	"github.com/overmindtech/cli/sources"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

var BigQueryTableLookupByID = shared.NewItemTypeLookup("id", gcpshared.BigQueryTable)

type BigQueryTableWrapper struct {
	client gcpshared.BigQueryTableClient

	*gcpshared.ProjectBase
}

// NewBigQueryTable creates a new bigQueryTable instance
func NewBigQueryTable(client gcpshared.BigQueryTableClient, locations []gcpshared.LocationInfo) sources.SearchStreamableWrapper {
	return &BigQueryTableWrapper{
		client: client,
		ProjectBase: gcpshared.NewProjectBase(
			locations,
			sdp.AdapterCategory_ADAPTER_CATEGORY_DATABASE,
			gcpshared.BigQueryTable,
		),
	}
}

func (b BigQueryTableWrapper) IAMPermissions() []string {
	return []string{
		"bigquery.tables.get",
		"bigquery.tables.list",
	}
}

func (b BigQueryTableWrapper) PredefinedRole() string {
	return "roles/bigquery.metadataViewer"
}

// PotentialLinks returns the potential links for the BigQuery table wrapper
func (b BigQueryTableWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		gcpshared.CloudKMSCryptoKey,
		gcpshared.BigQueryDataset,
		gcpshared.BigQueryConnection,
		gcpshared.StorageBucket,
		gcpshared.BigQueryTable,
	)
}

// TerraformMappings returns the Terraform mappings for the BigQuery dataset wrapper
func (b BigQueryTableWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod: sdp.QueryMethod_GET,
			// https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/bigquery_table
			// projects/{{project}}/datasets/{{dataset}}/tables/{{name}}
			TerraformQueryMap: "google_bigquery_table.id",
		},
	}
}

// GetLookups returns the lookups for the BigQuery dataset
func (b BigQueryTableWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		BigQueryDatasetLookupByID,
		BigQueryTableLookupByID,
	}
}

func (b BigQueryTableWrapper) SearchLookups() []sources.ItemTypeLookups {
	return []sources.ItemTypeLookups{
		{
			BigQueryDatasetLookupByID,
		},
	}
}

// Get retrieves a BigQuery dataset by its ID
func (b BigQueryTableWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	location, err := b.LocationFromScope(scope)
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: err.Error(),
		}
	}

	// O: dataset ID
	// 1: table ID
	metadata, err := b.client.Get(ctx, location.ProjectID, queryParts[0], queryParts[1])
	if err != nil {
		return nil, gcpshared.QueryError(err, scope, b.Type())
	}

	return b.GCPBigQueryTableToItem(location, metadata)
}

func (b BigQueryTableWrapper) Search(ctx context.Context, scope string, queryParts ...string) ([]*sdp.Item, *sdp.QueryError) {
	location, err := b.LocationFromScope(scope)
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: err.Error(),
		}
	}

	// queryParts[0]: Dataset ID
	items, listErr := b.client.List(ctx, location.ProjectID, queryParts[0], func(md *bigquery.TableMetadata) (*sdp.Item, *sdp.QueryError) {
		return b.GCPBigQueryTableToItem(location, md)
	})
	return items, listErr
}

func (b BigQueryTableWrapper) SearchStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string, queryParts ...string) {
	location, err := b.LocationFromScope(scope)
	if err != nil {
		stream.SendError(&sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: err.Error(),
		})
		return
	}

	// queryParts[0]: Dataset ID
	b.client.ListStream(ctx, location.ProjectID, queryParts[0], stream, func(md *bigquery.TableMetadata) (*sdp.Item, *sdp.QueryError) {
		item, qerr := b.GCPBigQueryTableToItem(location, md)
		if qerr == nil && item != nil {
			cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
		}
		return item, qerr
	})
}

func (b BigQueryTableWrapper) GCPBigQueryTableToItem(location gcpshared.LocationInfo, metadata *bigquery.TableMetadata) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(metadata, "labels")
	if err != nil {
		return nil, gcpshared.QueryError(err, location.ToScope(), b.Type())
	}

	// The full dataset ID in the form projectID:datasetID.tableID
	parts := strings.Split(strings.TrimPrefix(metadata.FullID, location.ProjectID+":"), ".")
	if len(parts) != 2 {
		return nil, gcpshared.QueryError(fmt.Errorf("invalid table full ID: %s", metadata.FullID), location.ToScope(), b.Type())
	}

	// O: dataset ID
	// 1: table ID
	err = attributes.Set("id", strings.Join(parts, shared.QuerySeparator))
	if err != nil {
		return nil, gcpshared.QueryError(err, location.ToScope(), b.Type())
	}

	sdpItem := &sdp.Item{
		Type:            gcpshared.BigQueryTable.String(),
		UniqueAttribute: "id",
		Attributes:      attributes,
		Scope:           location.ToScope(),
		Tags:            metadata.Labels,
	}

	sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   gcpshared.BigQueryDataset.String(),
			Method: sdp.QueryMethod_GET,
			Query:  parts[0], // dataset ID
			Scope:  location.ProjectID,
		},
		BlastPropagation: &sdp.BlastPropagation{
			// Tightly coupled.
			In:  true,
			Out: true,
		},
	})

	if metadata.EncryptionConfig != nil && metadata.EncryptionConfig.KMSKeyName != "" {
		// The KMS key used to encrypt the table.
		// The KMS key name can have the form
		// projects/{projectId}/locations/{locationId}/keyRings/{keyRingId}/cryptoKeys/{cryptoKeyId}
		values := gcpshared.ExtractPathParams(metadata.EncryptionConfig.KMSKeyName, "locations", "keyRings", "cryptoKeys")
		if len(values) == 3 && values[0] != "" && values[1] != "" && values[2] != "" {
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   gcpshared.CloudKMSCryptoKey.String(),
					Method: sdp.QueryMethod_GET,
					Query:  shared.CompositeLookupKey(values...),
					Scope:  location.ProjectID,
				},
				BlastPropagation: &sdp.BlastPropagation{
					// Tightly coupled.
					In:  true,
					Out: false,
				},
			})
		}
	}

	if metadata.ExternalDataConfig != nil {
		if metadata.ExternalDataConfig.ConnectionID != "" {
			// The connection specifying the credentials to be used to read external storage, such as Azure Blob, Cloud Storage, or S3.
			// The connectionId can have the form
			// {projectId}.{locationId};{connectionId} or
			// projects/{projectId}/locations/{locationId}/connections/{connectionId}
			var projectID, connectionLocation, connectionID string
			values := gcpshared.ExtractPathParams(metadata.ExternalDataConfig.ConnectionID, "projects", "locations", "connections")
			if len(values) == 3 {
				projectID = values[0]
				connectionLocation = values[1]
				connectionID = values[2]
			} else {
				// {projectId}.{locationId};{connectionId}
				resParts := strings.Split(metadata.ExternalDataConfig.ConnectionID, ".")
				if len(resParts) == 2 {
					projectID = resParts[0]
					// {locationId};{connectionId}
					colParts := strings.Split(resParts[1], ";")
					if len(colParts) == 2 {
						connectionLocation = colParts[0]
						connectionID = colParts[1]
					}
				}
			}
			if projectID != "" && connectionLocation != "" && connectionID != "" {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   gcpshared.BigQueryConnection.String(),
						Method: sdp.QueryMethod_GET,
						Query:  shared.CompositeLookupKey(connectionLocation, connectionID),
						Scope:  projectID,
					},
					BlastPropagation: &sdp.BlastPropagation{
						// Tightly coupled.
						In:  true,
						Out: true,
					},
				})
			}
		}

		// Link to Storage Buckets referenced in source URIs (gs:// URIs).
		// Format: gs://bucket-name/path/to/file or gs://bucket-name/path/* (wildcard allowed after bucket name).
		// GET https://storage.googleapis.com/storage/v1/b/{bucket}
		// https://cloud.google.com/storage/docs/json_api/v1/buckets/get
		if len(metadata.ExternalDataConfig.SourceURIs) > 0 {
			// Use a map to deduplicate bucket names
			bucketMap := make(map[string]bool)
			for _, sourceURI := range metadata.ExternalDataConfig.SourceURIs {
				if sourceURI != "" {
					// Use the StorageBucket linker to extract bucket name from various URI formats
					if linkFunc, ok := gcpshared.ManualAdapterLinksByAssetType[gcpshared.StorageBucket]; ok {
						// The linker handles gs:// URIs and extracts bucket names
						linkedQuery := linkFunc(location.ProjectID, location.ToScope(), sourceURI, &sdp.BlastPropagation{
							// If the Storage Bucket is deleted or updated: The external table may fail to read data. If the table is updated: The bucket remains unaffected.
							In:  true,
							Out: false,
						})
						if linkedQuery != nil {
							// Create a unique key from query and scope to deduplicate
							bucketKey := fmt.Sprintf("%s|%s", linkedQuery.GetQuery().GetQuery(), linkedQuery.GetQuery().GetScope())
							if !bucketMap[bucketKey] {
								bucketMap[bucketKey] = true
								sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, linkedQuery)
							}
						}
					}
				}
			}
		}
	}

	// Link to base table if this is a snapshot.
	// The base table from which this snapshot was created.
	// GET https://bigquery.googleapis.com/bigquery/v2/projects/{projectId}/datasets/{datasetId}/tables/{tableId}
	// https://cloud.google.com/bigquery/docs/reference/rest/v2/tables/get
	if metadata.SnapshotDefinition != nil && metadata.SnapshotDefinition.BaseTableReference != nil {
		baseTableRef := metadata.SnapshotDefinition.BaseTableReference
		if baseTableRef.ProjectID != "" && baseTableRef.DatasetID != "" && baseTableRef.TableID != "" {
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   gcpshared.BigQueryTable.String(),
					Method: sdp.QueryMethod_GET,
					Query:  shared.CompositeLookupKey(baseTableRef.DatasetID, baseTableRef.TableID),
					Scope:  baseTableRef.ProjectID,
				},
				BlastPropagation: &sdp.BlastPropagation{
					// If the base table is deleted or updated: The snapshot may become invalid or inaccessible. If the snapshot is updated: The base table remains unaffected.
					In:  true,
					Out: false,
				},
			})
		}
	}

	// Link to base table if this is a clone.
	// The base table from which this clone was created.
	// GET https://bigquery.googleapis.com/bigquery/v2/projects/{projectId}/datasets/{datasetId}/tables/{tableId}
	// https://cloud.google.com/bigquery/docs/reference/rest/v2/tables/get
	if metadata.CloneDefinition != nil && metadata.CloneDefinition.BaseTableReference != nil {
		baseTableRef := metadata.CloneDefinition.BaseTableReference
		if baseTableRef.ProjectID != "" && baseTableRef.DatasetID != "" && baseTableRef.TableID != "" {
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   gcpshared.BigQueryTable.String(),
					Method: sdp.QueryMethod_GET,
					Query:  shared.CompositeLookupKey(baseTableRef.DatasetID, baseTableRef.TableID),
					Scope:  baseTableRef.ProjectID,
				},
				BlastPropagation: &sdp.BlastPropagation{
					// If the base table is deleted or updated: The clone may become invalid or inaccessible. If the clone is updated: The base table remains unaffected.
					In:  true,
					Out: false,
				},
			})
		}
	}

	// Note: Replicas field is not available in the Go client library's TableMetadata struct,
	// even though it exists in the REST API. If needed in the future, we would need to access
	// the raw REST API response or wait for the Go client library to expose this field.

	return sdpItem, nil
}
