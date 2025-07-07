package manual

import (
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/bigquery"

	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

var (
	BigQueryDatasetLookupByID = shared.NewItemTypeLookup("id", gcpshared.BigQueryDataset)
)

type BigQueryDatasetWrapper struct {
	client gcpshared.BigQueryDatasetClient

	*gcpshared.ProjectBase
}

// NewBigQueryDataset creates a new bigQueryDatasetWrapper instance
func NewBigQueryDataset(client gcpshared.BigQueryDatasetClient, projectID string) sources.ListableWrapper {
	return &BigQueryDatasetWrapper{
		client: client,
		ProjectBase: gcpshared.NewProjectBase(
			projectID,
			sdp.AdapterCategory_ADAPTER_CATEGORY_DATABASE,
			gcpshared.BigQueryDataset,
		),
	}
}

func (b BigQueryDatasetWrapper) IAMPermissions() []string {
	// https://cloud.google.com/bigquery/docs/access-control
	// There is no specific permission for listing datasets, so we use the get permission
	// TODO: Confirm if this is sufficient for listing datasets and their metadata
	return []string{
		"bigquery.datasets.get",
	}
}

// PotentialLinks returns the potential links for the BigQuery dataset wrapper
func (b BigQueryDatasetWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		gcpshared.IAMServiceAccount,
		gcpshared.CloudKMSCryptoKey,
		gcpshared.BigQueryDataset,
		gcpshared.BigQueryConnection,
	)
}

// TerraformMappings returns the Terraform mappings for the BigQuery dataset wrapper
func (b BigQueryDatasetWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "google_bigquery_dataset.dataset_id",
		},
	}
}

// GetLookups returns the lookups for the BigQuery dataset
func (b BigQueryDatasetWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		BigQueryDatasetLookupByID,
	}
}

// Get retrieves a BigQuery dataset by its ID
func (b BigQueryDatasetWrapper) Get(ctx context.Context, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	metadata, err := b.client.Get(ctx, b.ProjectID(), queryParts[0])
	if err != nil {
		return nil, gcpshared.QueryError(err)
	}

	return b.GCPBigQueryDatasetToItem(ctx, metadata)
}

func (b BigQueryDatasetWrapper) List(ctx context.Context) ([]*sdp.Item, *sdp.QueryError) {
	items, err := b.client.List(ctx, b.ProjectID(), b.GCPBigQueryDatasetToItem)
	if err != nil {
		return nil, err
	}

	return items, nil
}

func (b BigQueryDatasetWrapper) GCPBigQueryDatasetToItem(ctx context.Context, metadata *bigquery.DatasetMetadata) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(metadata, "labels")
	if err != nil {
		return nil, gcpshared.QueryError(err)
	}

	// The full dataset ID in the form projectID:datasetID.
	parts := strings.Split(metadata.FullID, ":")
	if len(parts) != 2 {
		return nil, gcpshared.QueryError(fmt.Errorf("invalid dataset full ID: %s", metadata.FullID))
	}

	err = attributes.Set("id", parts[1])
	if err != nil {
		return nil, gcpshared.QueryError(err)
	}

	sdpItem := &sdp.Item{
		Type:            gcpshared.BigQueryDataset.String(),
		UniqueAttribute: "id",
		Attributes:      attributes,
		Scope:           b.DefaultScope(),
		Tags:            metadata.Labels,
	}

	for _, access := range metadata.Access {
		if access.EntityType == bigquery.GroupEmailEntity ||
			access.EntityType == bigquery.UserEmailEntity ||
			access.EntityType == bigquery.IAMMemberEntity {
			if access.Entity != "" {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   gcpshared.IAMServiceAccount.String(),
						Method: sdp.QueryMethod_GET,
						Query:  access.Entity,
						Scope:  b.DefaultScope(),
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				})
			}
		}

		if access.Dataset != nil && access.Dataset.Dataset != nil {
			// Link to the dataset that this access applies to
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   gcpshared.BigQueryDataset.String(),
					Method: sdp.QueryMethod_GET,
					Query:  access.Dataset.Dataset.DatasetID,
					Scope:  b.DefaultScope(),
				},
				BlastPropagation: &sdp.BlastPropagation{
					/*
						A grant authorizing all resources of a particular type in a particular dataset access to this dataset.
						Only views are supported for now.
						The role field is not required when this field is set.
						If that dataset is deleted and re-created, its access needs to be granted again via an update operation.
					*/
					In:  false,
					Out: true,
				},
			})

		}
	}

	if metadata.DefaultEncryptionConfig != nil {
		// Link to the KMS key used for default encryption
		values := gcpshared.ExtractPathParams(metadata.DefaultEncryptionConfig.KMSKeyName, "locations", "keyRings", "cryptoKeys")
		if len(values) == 3 && values[0] != "" && values[1] != "" && values[2] != "" {
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   gcpshared.CloudKMSCryptoKey.String(),
					Method: sdp.QueryMethod_GET,
					Query:  shared.CompositeLookupKey(values...),
					Scope:  b.ProjectID(),
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: false,
				},
			})
		}
	}

	if metadata.ExternalDatasetReference != nil && metadata.ExternalDatasetReference.Connection != "" {
		// Link to the external dataset reference
		// Format: projects/{projectId}/locations/{locationId}/connections/{connectionId}
		values := gcpshared.ExtractPathParams(metadata.ExternalDatasetReference.Connection, "locations", "connections")
		if len(values) == 2 && values[0] != "" && values[1] != "" {
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   gcpshared.BigQueryConnection.String(),
					Method: sdp.QueryMethod_GET,
					Query:  shared.CompositeLookupKey(values...),
					Scope:  b.DefaultScope(),
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
