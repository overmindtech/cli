package manual

import (
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/bigquery"

	"github.com/overmindtech/cli/go/discovery"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
	"github.com/overmindtech/cli/sources"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

var BigQueryDatasetLookupByID = shared.NewItemTypeLookup("id", gcpshared.BigQueryDataset)

type BigQueryDatasetWrapper struct {
	client gcpshared.BigQueryDatasetClient
	*gcpshared.ProjectBase
}

// NewBigQueryDataset creates a new bigQueryDatasetWrapper instance.
func NewBigQueryDataset(client gcpshared.BigQueryDatasetClient, locations []gcpshared.LocationInfo) sources.ListStreamableWrapper {
	return &BigQueryDatasetWrapper{
		client: client,
		ProjectBase: gcpshared.NewProjectBase(
			locations,
			sdp.AdapterCategory_ADAPTER_CATEGORY_DATABASE,
			gcpshared.BigQueryDataset,
		),
	}
}

func (b BigQueryDatasetWrapper) IAMPermissions() []string {
	return []string{
		"bigquery.datasets.get",
	}
}

func (b BigQueryDatasetWrapper) PredefinedRole() string {
	return "roles/bigquery.metadataViewer"
}

func (b BigQueryDatasetWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		gcpshared.IAMServiceAccount,
		gcpshared.CloudKMSCryptoKey,
		gcpshared.BigQueryDataset,
		gcpshared.BigQueryConnection,
		gcpshared.BigQueryModel,
		gcpshared.BigQueryRoutine,
		gcpshared.BigQueryTable,
	)
}

func (b BigQueryDatasetWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "google_bigquery_dataset.dataset_id",
		},
		// IAM resources for BigQuery Datasets. These are Terraform-only constructs
		// (no standalone GCP API resource exists). When an IAM binding/member/policy
		// changes, we resolve it to the parent dataset for blast radius analysis.
		//
		// Reference: https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/bigquery_dataset_iam
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "google_bigquery_dataset_iam_binding.dataset_id",
		},
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "google_bigquery_dataset_iam_member.dataset_id",
		},
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "google_bigquery_dataset_iam_policy.dataset_id",
		},
	}
}

func (b BigQueryDatasetWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		BigQueryDatasetLookupByID,
	}
}

func (b BigQueryDatasetWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	location, err := b.LocationFromScope(scope)
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: err.Error(),
		}
	}

	metadata, getErr := b.client.Get(ctx, location.ProjectID, queryParts[0])
	if getErr != nil {
		return nil, gcpshared.QueryError(getErr, scope, b.Type())
	}

	return b.gcpBigQueryDatasetToItem(metadata, location)
}

func (b BigQueryDatasetWrapper) List(ctx context.Context, scope string) ([]*sdp.Item, *sdp.QueryError) {
	location, err := b.LocationFromScope(scope)
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: err.Error(),
		}
	}

	items, listErr := b.client.List(ctx, location.ProjectID, func(ctx context.Context, md *bigquery.DatasetMetadata) (*sdp.Item, *sdp.QueryError) {
		return b.gcpBigQueryDatasetToItem(md, location)
	})
	if listErr != nil {
		return nil, gcpshared.QueryError(listErr, scope, b.Type())
	}

	return items, nil
}

func (b BigQueryDatasetWrapper) ListStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string) {
	location, err := b.LocationFromScope(scope)
	if err != nil {
		stream.SendError(&sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: err.Error(),
		})
		return
	}

	b.client.ListStream(ctx, location.ProjectID, stream, func(ctx context.Context, md *bigquery.DatasetMetadata) (*sdp.Item, *sdp.QueryError) {
		item, qerr := b.gcpBigQueryDatasetToItem(md, location)
		if qerr == nil && item != nil {
			cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
		}
		return item, qerr
	})
}

func (b BigQueryDatasetWrapper) gcpBigQueryDatasetToItem(metadata *bigquery.DatasetMetadata, location gcpshared.LocationInfo) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(metadata, "labels")
	if err != nil {
		return nil, gcpshared.QueryError(err, location.ToScope(), b.Type())
	}

	// The full dataset ID in the form projectID:datasetID.
	parts := strings.Split(metadata.FullID, ":")
	if len(parts) != 2 {
		return nil, gcpshared.QueryError(fmt.Errorf("invalid dataset full ID: %s", metadata.FullID), location.ToScope(), b.Type())
	}

	err = attributes.Set("id", parts[1])
	if err != nil {
		return nil, gcpshared.QueryError(err, location.ToScope(), b.Type())
	}

	sdpItem := &sdp.Item{
		Type:            gcpshared.BigQueryDataset.String(),
		UniqueAttribute: "id",
		Attributes:      attributes,
		Scope:           location.ToScope(),
		Tags:            metadata.Labels,
	}

	// Link to contained models.
	sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   gcpshared.BigQueryModel.String(),
			Method: sdp.QueryMethod_SEARCH,
			Query:  parts[1],
			Scope:  location.ToScope(),
		},
	})

	// Link to contained tables.
	sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   gcpshared.BigQueryTable.String(),
			Method: sdp.QueryMethod_SEARCH,
			Query:  parts[1],
			Scope:  location.ToScope(),
		},
	})

	// Link to contained routines.
	sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   gcpshared.BigQueryRoutine.String(),
			Method: sdp.QueryMethod_SEARCH,
			Query:  parts[1],
			Scope:  location.ToScope(),
		},
	})

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
						Scope:  location.ToScope(),
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
					Scope:  location.ToScope(),
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
					Scope:  location.ProjectID,
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
					Scope:  location.ToScope(),
				},
			})
		}
	}

	return sdpItem, nil
}
