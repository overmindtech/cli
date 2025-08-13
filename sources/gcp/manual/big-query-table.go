package manual

import (
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/bigquery"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

var (
	BigQueryTableLookupByID = shared.NewItemTypeLookup("id", gcpshared.BigQueryTable)
)

type BigQueryTableWrapper struct {
	client gcpshared.BigQueryTableClient

	*gcpshared.ProjectBase
}

// NewBigQueryTable creates a new bigQueryTable instance
func NewBigQueryTable(client gcpshared.BigQueryTableClient, projectID string) sources.SearchableWrapper {
	return &BigQueryTableWrapper{
		client: client,
		ProjectBase: gcpshared.NewProjectBase(
			projectID,
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

// PotentialLinks returns the potential links for the BigQuery dataset wrapper
func (b BigQueryTableWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		gcpshared.CloudKMSCryptoKey,
		gcpshared.BigQueryDataset,
		gcpshared.BigQueryConnection,
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
func (b BigQueryTableWrapper) Get(ctx context.Context, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	// O: dataset ID
	// 1: table ID
	metadata, err := b.client.Get(ctx, b.ProjectID(), queryParts[0], queryParts[1])
	if err != nil {
		return nil, gcpshared.QueryError(err)
	}

	return b.GCPBigQueryTableToItem(metadata)
}

func (b BigQueryTableWrapper) Search(ctx context.Context, queryParts ...string) ([]*sdp.Item, *sdp.QueryError) {
	// queryParts[0]: Dataset ID
	items, err := b.client.List(ctx, b.ProjectID(), queryParts[0], b.GCPBigQueryTableToItem)
	if err != nil {
		return nil, err
	}

	return items, nil
}

func (b BigQueryTableWrapper) SearchStream(ctx context.Context, stream discovery.QueryResultStream, queryParts ...string) {
	// queryParts[0]: Dataset ID
	b.client.ListStream(ctx, b.ProjectID(), queryParts[0], stream, b.GCPBigQueryTableToItem)
}

func (b BigQueryTableWrapper) GCPBigQueryTableToItem(metadata *bigquery.TableMetadata) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(metadata, "labels")
	if err != nil {
		return nil, gcpshared.QueryError(err)
	}

	// The full dataset ID in the form projectID:datasetID.tableID
	parts := strings.Split(strings.TrimPrefix(metadata.FullID, b.ProjectID()+":"), ".")
	if len(parts) != 2 {
		return nil, gcpshared.QueryError(fmt.Errorf("invalid table full ID: %s", metadata.FullID))
	}

	// O: dataset ID
	// 1: table ID
	err = attributes.Set("id", strings.Join(parts, shared.QuerySeparator))
	if err != nil {
		return nil, gcpshared.QueryError(err)
	}

	sdpItem := &sdp.Item{
		Type:            gcpshared.BigQueryTable.String(),
		UniqueAttribute: "id",
		Attributes:      attributes,
		Scope:           b.DefaultScope(),
		Tags:            metadata.Labels,
	}

	sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   gcpshared.BigQueryDataset.String(),
			Method: sdp.QueryMethod_GET,
			Query:  parts[0], // dataset ID
			Scope:  b.ProjectID(),
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
					Scope:  b.ProjectID(),
				},
				BlastPropagation: &sdp.BlastPropagation{
					// Tightly coupled.
					In:  true,
					Out: false,
				},
			})
		}
	}

	if metadata.ExternalDataConfig != nil && metadata.ExternalDataConfig.ConnectionID != "" {
		// The connection specifying the credentials to be used to read external storage, such as Azure Blob, Cloud Storage, or S3.
		// The connectionId can have the form
		// {projectId}.{locationId};{connectionId} or
		// projects/{projectId}/locations/{locationId}/connections/{connectionId}
		var projectID, location, connectionID string
		values := gcpshared.ExtractPathParams(metadata.ExternalDataConfig.ConnectionID, "projects", "locations", "connections")
		if len(values) == 3 {
			projectID = values[0]
			location = values[1]
			connectionID = values[2]
		} else {
			// {projectId}.{locationId};{connectionId}
			resParts := strings.Split(metadata.ExternalDataConfig.ConnectionID, ".")
			if len(resParts) == 2 {
				projectID = resParts[0]
				// {locationId};{connectionId}
				colParts := strings.Split(resParts[1], ";")
				if len(colParts) == 2 {
					location = colParts[0]
					connectionID = colParts[1]
				}
			}
		}
		if projectID != "" && location != "" && connectionID != "" {
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   gcpshared.BigQueryConnection.String(),
					Method: sdp.QueryMethod_GET,
					Query:  shared.CompositeLookupKey(location, connectionID),
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

	return sdpItem, nil
}
