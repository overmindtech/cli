package manual

import (
	"context"
	"strings"

	"cloud.google.com/go/bigquery"
	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
	"github.com/overmindtech/cli/sources"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

var BigQueryRoutineLookupByID = shared.NewItemTypeLookup("id", gcpshared.BigQueryRoutine)

type BigQueryRoutineWrapper struct {
	client gcpshared.BigQueryRoutineClient

	*gcpshared.ProjectBase
}

func NewBigQueryRoutine(client gcpshared.BigQueryRoutineClient, locations []gcpshared.LocationInfo) sources.SearchStreamableWrapper {
	return &BigQueryRoutineWrapper{
		client: client,
		ProjectBase: gcpshared.NewProjectBase(
			locations,
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
		gcpshared.StorageBucket,
		gcpshared.BigQueryConnection,
		stdlib.NetworkHTTP,
	)
}

// TerraformMappings returns the Terraform mappings for the BigQuery routine wrapper
func (b BigQueryRoutineWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod: sdp.QueryMethod_SEARCH,
			// https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/bigquery_routine
			// ID format: projects/{{project}}/datasets/{{dataset_id}}/routines/{{routine_id}}
			// The framework automatically intercepts queries starting with "projects/" and converts
			// them to GET operations by extracting the last N path parameters (based on GetLookups count).
			TerraformQueryMap: "google_bigquery_routine.id",
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
func (b BigQueryRoutineWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	location, err := b.LocationFromScope(scope)
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: err.Error(),
		}
	}
	// 0: dataset ID
	// 1: routine ID
	metadata, getErr := b.client.Get(ctx, location.ProjectID, queryParts[0], queryParts[1])
	if getErr != nil {
		return nil, gcpshared.QueryError(getErr, scope, b.Type())
	}
	return b.gcpBigQueryRoutineToItem(metadata, queryParts[0], queryParts[1], location)
}

func (b BigQueryRoutineWrapper) Search(ctx context.Context, scope string, queryParts ...string) ([]*sdp.Item, *sdp.QueryError) {
	location, err := b.LocationFromScope(scope)
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: err.Error(),
		}
	}

	toItem := func(metadata *bigquery.RoutineMetadata, datasetID, routineID string) (*sdp.Item, *sdp.QueryError) {
		return b.gcpBigQueryRoutineToItem(metadata, datasetID, routineID, location)
	}

	items, listErr := b.client.List(ctx, location.ProjectID, queryParts[0], toItem)
	if listErr != nil {
		return nil, gcpshared.QueryError(listErr, scope, b.Type())
	}

	return items, nil
}

func (b BigQueryRoutineWrapper) SearchStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string, queryParts ...string) {
	location, err := b.LocationFromScope(scope)
	if err != nil {
		stream.SendError(&sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: err.Error(),
		})
		return
	}

	toItem := func(metadata *bigquery.RoutineMetadata, datasetID, routineID string) (*sdp.Item, *sdp.QueryError) {
		item, qerr := b.gcpBigQueryRoutineToItem(metadata, datasetID, routineID, location)
		if qerr == nil && item != nil {
			cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
		}
		return item, qerr
	}

	items, listErr := b.client.List(ctx, location.ProjectID, queryParts[0], toItem)
	if listErr != nil {
		stream.SendError(gcpshared.QueryError(listErr, scope, b.Type()))
		return
	}

	for _, item := range items {
		stream.SendItem(item)
	}
}

func (b BigQueryRoutineWrapper) gcpBigQueryRoutineToItem(metadata *bigquery.RoutineMetadata, datasetID, routineID string, location gcpshared.LocationInfo) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(metadata, "")
	if err != nil {
		return nil, gcpshared.QueryError(err, location.ToScope(), b.Type())
	}

	err = attributes.Set("id", shared.CompositeLookupKey(datasetID, routineID))
	if err != nil {
		return nil, gcpshared.QueryError(err, location.ToScope(), b.Type())
	}

	sdpItem := &sdp.Item{
		Type:            gcpshared.BigQueryRoutine.String(),
		UniqueAttribute: "id",
		Attributes:      attributes,
		Scope:           location.ToScope(),
		Tags:            make(map[string]string),
	}

	sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   gcpshared.BigQueryDataset.String(),
			Method: sdp.QueryMethod_GET,
			Query:  datasetID,
			Scope:  location.ProjectID,
		},
		BlastPropagation: &sdp.BlastPropagation{
			In:  true,
			Out: true,
		},
	})

	// Link to imported libraries (GCS buckets) for JavaScript routines
	// Format: gs://bucket-name/path/to/file.js
	if len(metadata.ImportedLibraries) > 0 {
		blastPropagation := &sdp.BlastPropagation{
			In:  true,
			Out: false,
		}
		if linkFunc, ok := gcpshared.ManualAdapterLinksByAssetType[gcpshared.StorageBucket]; ok {
			for _, libraryURI := range metadata.ImportedLibraries {
				if libraryURI != "" {
					linkedQuery := linkFunc(location.ProjectID, location.ToScope(), libraryURI, blastPropagation)
					if linkedQuery != nil {
						sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, linkedQuery)
					}
				}
			}
		}
	}

	// Link to BigQuery Connection used for remote function authentication
	// Format: projects/{projectId}/locations/{locationId}/connections/{connectionId}
	// or: {projectId}.{locationId};{connectionId}
	if metadata.RemoteFunctionOptions != nil && metadata.RemoteFunctionOptions.Connection != "" {
		var projectID, location, connectionID string
		values := gcpshared.ExtractPathParams(metadata.RemoteFunctionOptions.Connection, "projects", "locations", "connections")
		if len(values) == 3 {
			projectID = values[0]
			location = values[1]
			connectionID = values[2]
		} else {
			// Try short format: {projectId}.{locationId};{connectionId}
			resParts := strings.Split(metadata.RemoteFunctionOptions.Connection, ".")
			if len(resParts) == 2 {
				projectID = resParts[0]
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
					// If the Connection is deleted or updated: The remote function may fail to authenticate. If the routine is updated: The connection remains unaffected.
					In:  true,
					Out: false,
				},
			})
		}
	}

	// Link to HTTP endpoint for remote function calls
	// Format: https://example.com/run or http://example.com/run
	if metadata.RemoteFunctionOptions != nil && metadata.RemoteFunctionOptions.Endpoint != "" {
		endpoint := strings.TrimSpace(metadata.RemoteFunctionOptions.Endpoint)
		if strings.HasPrefix(endpoint, "http://") || strings.HasPrefix(endpoint, "https://") {
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   stdlib.NetworkHTTP.String(),
					Method: sdp.QueryMethod_SEARCH,
					Query:  endpoint,
					Scope:  "global",
				},
				BlastPropagation: &sdp.BlastPropagation{
					// If the HTTP endpoint is unreachable: The remote function will fail to execute. If the routine is updated: The endpoint remains unaffected.
					In:  true,
					Out: false,
				},
			})
		}
	}

	// NOTE: SparkOptions and ExternalRuntimeOptions are not currently available in the Go SDK's RoutineMetadata struct,
	// even though they exist in the REST API. If the Go SDK is updated to include these fields in the future,
	// we should add links for:
	// - sparkOptions.connection (BigQuery Connection)
	// - sparkOptions.mainFileUri, pyFileUris, jarUris, fileUris, archiveUris (GCS buckets)
	// - externalRuntimeOptions.runtimeConnection (BigQuery Connection)

	// NOTE: optional feature for the future - parse routine_definition to identify referenced tables/views/connections and add links. Out-of-scope for initial version.

	return sdpItem, nil
}
