package dynamic

import (
	"fmt"

	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

// Scope defines the scope of a GCP resource.
type Scope string

const (
	ScopeGlobal   Scope = "global"
	ScopeRegional Scope = "regional"
	ScopeZonal    Scope = "zonal"
)

// AdapterMeta contains metadata for a GCP dynamic adapter.
type AdapterMeta struct {
	Scope                  Scope
	GetEndpointBaseURLFunc func(queryParts ...string) (string, error)
	ListEndpointFunc       func(queryParts ...string) (string, error)
	SearchEndpointFunc     func(queryParts ...string) (string, error)
	SDPAdapterCategory     sdp.AdapterCategory
}

// sdpAssetTypeToAdapterMeta maps GCP asset types to their corresponding adapter metadata.
var sdpAssetTypeToAdapterMeta = map[shared.ItemType]AdapterMeta{
	gcpshared.ComputeNetwork: {
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
		Scope:              ScopeGlobal,
		// https://compute.googleapis.com/compute/v1/projects/{project}/global/networks/{network}
		GetEndpointBaseURLFunc: func(queryParts ...string) (string, error) {
			if len(queryParts) == 1 && queryParts[0] != "" {
				return fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/global/networks/", queryParts[0]), nil
			}
			return "", fmt.Errorf("projectID cannot be empty")
		},
		// https://compute.googleapis.com/compute/v1/projects/{project}/global/networks
		ListEndpointFunc: func(queryParts ...string) (string, error) {
			if len(queryParts) == 1 && queryParts[0] != "" {
				return fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/global/networks", queryParts[0]), nil
			}
			return "", fmt.Errorf("projectID cannot be empty: %v", queryParts)
		},
	},
	gcpshared.ComputeSubnetwork: {
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
		Scope:              ScopeRegional,
		// https://compute.googleapis.com/compute/v1/projects/{project}/regions/{region}/subnetworks/{subnetwork}
		GetEndpointBaseURLFunc: func(queryParts ...string) (string, error) {
			if len(queryParts) == 2 && queryParts[0] != "" && queryParts[1] != "" {
				return fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/regions/%s/subnetworks/", queryParts[0], queryParts[1]), nil
			}
			return "", fmt.Errorf("projectID and region cannot be empty: %v", queryParts)
		},
		// https://compute.googleapis.com/compute/v1/projects/{project}/regions/{region}/subnetworks
		ListEndpointFunc: func(queryParts ...string) (string, error) {
			if len(queryParts) == 2 && queryParts[0] != "" && queryParts[1] != "" {
				return fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/regions/%s/subnetworks", queryParts[0], queryParts[1]), nil
			}
			return "", fmt.Errorf("projectID and region cannot be empty: %v", queryParts)
		},
	},
	gcpshared.BigQueryTable: {
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_DATABASE,
		Scope:              ScopeGlobal,
		// https://bigquery.googleapis.com/bigquery/v2/projects/{projectId}/datasets/{datasetId}/tables/{tableId}
		GetEndpointBaseURLFunc: func(queryParts ...string) (string, error) {
			if len(queryParts) == 2 && queryParts[0] != "" && queryParts[1] != "" {
				return fmt.Sprintf("https://bigquery.googleapis.com/bigquery/v2/projects/%s/datasets/%s/tables/", queryParts[0], queryParts[1]), nil
			}
			return "", fmt.Errorf("projectID and datasetID cannot be empty: %v", queryParts)
		},
		// https://bigquery.googleapis.com/bigquery/v2/projects/{projectId}/datasets/{datasetId}/tables
		SearchEndpointFunc: func(queryParts ...string) (string, error) {
			if len(queryParts) == 2 && queryParts[0] != "" && queryParts[1] != "" {
				return fmt.Sprintf("https://bigquery.googleapis.com/bigquery/v2/projects/%s/datasets/%s/tables", queryParts[0], queryParts[1]), nil
			}
			return "", fmt.Errorf("projectID and datasetID cannot be empty: %v", queryParts)
		},
	},
	gcpshared.BigQueryDataset: {
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_DATABASE,
		Scope:              ScopeGlobal,
		// https://bigquery.googleapis.com/bigquery/v2/projects/{projectId}/datasets/{datasetId}
		GetEndpointBaseURLFunc: func(queryParts ...string) (string, error) {
			if len(queryParts) == 1 && queryParts[0] != "" {
				return fmt.Sprintf("https://bigquery.googleapis.com/bigquery/v2/projects/%s/datasets/", queryParts[0]), nil
			}
			return "", fmt.Errorf("projectID cannot be empty: %v", queryParts)
		},
		// https://bigquery.googleapis.com/bigquery/v2/projects/{projectId}/datasets
		ListEndpointFunc: func(queryParts ...string) (string, error) {
			if len(queryParts) == 1 && queryParts[0] != "" {
				return fmt.Sprintf("https://bigquery.googleapis.com/bigquery/v2/projects/%s/datasets", queryParts[0]), nil
			}
			return "", fmt.Errorf("projectID cannot be empty: %v", queryParts)
		},
	},
	gcpshared.PubSubSubscription: {
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_CONFIGURATION,
		Scope:              ScopeGlobal,
		// https://pubsub.googleapis.com/v1/projects/{project}/subscriptions/{subscription}
		GetEndpointBaseURLFunc: func(queryParts ...string) (string, error) {
			if len(queryParts) == 1 && queryParts[0] != "" {
				return fmt.Sprintf("https://pubsub.googleapis.com/v1/projects/%s/subscriptions/", queryParts[0]), nil
			}
			return "", fmt.Errorf("projectID cannot be empty: %v", queryParts)
		},
		// https://pubsub.googleapis.com/v1/projects/{project}/subscriptions
		ListEndpointFunc: func(queryParts ...string) (string, error) {
			if len(queryParts) == 1 && queryParts[0] != "" {
				return fmt.Sprintf("https://pubsub.googleapis.com/v1/projects/%s/subscriptions", queryParts[0]), nil
			}
			return "", fmt.Errorf("projectID cannot be empty: %v", queryParts)
		},
	},
	gcpshared.PubSubTopic: {
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_CONFIGURATION,
		Scope:              ScopeGlobal,
		// https://pubsub.googleapis.com/v1/projects/{project}/topics/{topic}
		GetEndpointBaseURLFunc: func(queryParts ...string) (string, error) {
			if len(queryParts) == 1 && queryParts[0] != "" {
				return fmt.Sprintf("https://pubsub.googleapis.com/v1/projects/%s/topics/", queryParts[0]), nil
			}
			return "", fmt.Errorf("projectID cannot be empty: %v", queryParts)
		},
		// https://pubsub.googleapis.com/v1/projects/{project}/topics
		ListEndpointFunc: func(queryParts ...string) (string, error) {
			if len(queryParts) == 1 && queryParts[0] != "" {
				return fmt.Sprintf("https://pubsub.googleapis.com/v1/projects/%s/topics", queryParts[0]), nil
			}
			return "", fmt.Errorf("projectID cannot be empty: %v", queryParts)
		},
	},
}
