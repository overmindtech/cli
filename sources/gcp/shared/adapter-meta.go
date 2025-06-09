package shared

import (
	"fmt"
	"strings"

	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources/shared"
)

// Scope defines the scope of a GCP resource.
type Scope string

const (
	ScopeProject  Scope = "project"
	ScopeRegional Scope = "regional"
	ScopeZonal    Scope = "zonal"
)

type EndpointFunc func(query string) string

// AdapterMeta contains metadata for a GCP dynamic adapter.
type AdapterMeta struct {
	Scope                  Scope
	GetEndpointBaseURLFunc func(queryParts ...string) (EndpointFunc, error)
	ListEndpointFunc       func(queryParts ...string) (string, error)
	SearchEndpointFunc     func(queryParts ...string) (EndpointFunc, error)
	SDPAdapterCategory     sdp.AdapterCategory
	UniqueAttributeKeys    []string
}

// SDPAssetTypeToAdapterMeta maps GCP asset types to their corresponding adapter metadata.
var SDPAssetTypeToAdapterMeta = map[shared.ItemType]AdapterMeta{
	ComputeNetwork: {
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
		Scope:              ScopeProject,
		// https://compute.googleapis.com/compute/v1/projects/{project}/global/networks/{network}
		GetEndpointBaseURLFunc: func(adapterInitParams ...string) (EndpointFunc, error) {
			if len(adapterInitParams) == 1 && adapterInitParams[0] != "" {
				return func(query string) string {
					if query != "" {
						return fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/global/networks/%s", adapterInitParams[0], query)
					}
					return ""
				}, nil
			}
			return nil, fmt.Errorf("projectID cannot be empty")
		},
		// https://compute.googleapis.com/compute/v1/projects/{project}/global/networks
		ListEndpointFunc: func(adapterInitParams ...string) (string, error) {
			if len(adapterInitParams) == 1 && adapterInitParams[0] != "" {
				return fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/global/networks", adapterInitParams[0]), nil
			}
			return "", fmt.Errorf("projectID cannot be empty: %v", adapterInitParams)
		},
		UniqueAttributeKeys: []string{"networks"},
	},
	ComputeSubnetwork: {
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
		Scope:              ScopeRegional,
		// https://compute.googleapis.com/compute/v1/projects/{project}/regions/{region}/subnetworks/{subnetwork}
		GetEndpointBaseURLFunc: func(adapterInitParams ...string) (EndpointFunc, error) {
			if len(adapterInitParams) == 2 && adapterInitParams[0] != "" && adapterInitParams[1] != "" {
				return func(query string) string {
					if query != "" {
						return fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/regions/%s/subnetworks/%s", adapterInitParams[0], adapterInitParams[1], query)
					}
					return ""
				}, nil
			}
			return nil, fmt.Errorf("projectID and region cannot be empty: %v", adapterInitParams)
		},
		// https://compute.googleapis.com/compute/v1/projects/{project}/regions/{region}/subnetworks
		ListEndpointFunc: func(queryParts ...string) (string, error) {
			if len(queryParts) == 2 && queryParts[0] != "" && queryParts[1] != "" {
				return fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/regions/%s/subnetworks", queryParts[0], queryParts[1]), nil
			}
			return "", fmt.Errorf("projectID and region cannot be empty: %v", queryParts)
		},
		UniqueAttributeKeys: []string{"subnetworks"},
	},
	BigQueryTable: {
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_DATABASE,
		Scope:              ScopeProject,
		// https://bigquery.googleapis.com/bigquery/v2/projects/{projectId}/datasets/{datasetId}/tables/{tableId}
		GetEndpointBaseURLFunc: func(adapterInitParams ...string) (EndpointFunc, error) {
			if len(adapterInitParams) == 1 && adapterInitParams[0] != "" {
				return func(query string) string {
					// query must be a composite of datasetID and tableID
					queryParts := strings.Split(query, shared.QuerySeparator)
					if len(queryParts) == 1 && queryParts[0] != "" && queryParts[1] != "" {
						return fmt.Sprintf("https://bigquery.googleapis.com/bigquery/v2/projects/%s/datasets/%s/tables/%s", adapterInitParams[0], queryParts[0], queryParts[1])
					}
					return ""
				}, nil
			}
			return nil, fmt.Errorf("projectID cannot be empty: %v", adapterInitParams)
		},
		// https://bigquery.googleapis.com/bigquery/v2/projects/{projectId}/datasets/{datasetId}/tables
		SearchEndpointFunc: func(adapterInitParams ...string) (EndpointFunc, error) {
			if len(adapterInitParams) == 1 && adapterInitParams[0] != "" {
				return func(query string) string {
					if query != "" {
						return fmt.Sprintf("https://bigquery.googleapis.com/bigquery/v2/projects/%s/datasets/%s/tables", adapterInitParams[0], query)
					}
					return ""
				}, nil
			}
			return nil, fmt.Errorf("projectID cannot be empty: %v", adapterInitParams)
		},
		UniqueAttributeKeys: []string{"datasets", "tables"},
	},
	BigQueryDataset: {
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_DATABASE,
		Scope:              ScopeProject,
		// https://bigquery.googleapis.com/bigquery/v2/projects/{projectId}/datasets/{datasetId}
		GetEndpointBaseURLFunc: func(adapterInitParams ...string) (EndpointFunc, error) {
			if len(adapterInitParams) == 1 && adapterInitParams[0] != "" {
				return func(query string) string {
					if query != "" {
						return fmt.Sprintf("https://bigquery.googleapis.com/bigquery/v2/projects/%s/datasets/%s", adapterInitParams[0], query)
					}
					return ""
				}, nil
			}
			return nil, fmt.Errorf("projectID cannot be empty: %v", adapterInitParams)
		},
		// https://bigquery.googleapis.com/bigquery/v2/projects/{projectId}/datasets
		ListEndpointFunc: func(queryParts ...string) (string, error) {
			if len(queryParts) == 1 && queryParts[0] != "" {
				return fmt.Sprintf("https://bigquery.googleapis.com/bigquery/v2/projects/%s/datasets", queryParts[0]), nil
			}
			return "", fmt.Errorf("projectID cannot be empty: %v", queryParts)
		},
		UniqueAttributeKeys: []string{"datasets"},
	},
	PubSubSubscription: {
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_CONFIGURATION,
		Scope:              ScopeProject,
		// https://pubsub.googleapis.com/v1/projects/{project}/subscriptions/{subscription}
		GetEndpointBaseURLFunc: func(adapterInitParams ...string) (EndpointFunc, error) {
			if len(adapterInitParams) == 1 && adapterInitParams[0] != "" {
				return func(query string) string {
					if query != "" {
						return fmt.Sprintf("https://pubsub.googleapis.com/v1/projects/%s/subscriptions/%s", adapterInitParams[0], query)
					}
					return ""
				}, nil
			}
			return nil, fmt.Errorf("projectID cannot be empty: %v", adapterInitParams)
		},
		// https://pubsub.googleapis.com/v1/projects/{project}/subscriptions
		ListEndpointFunc: func(queryParts ...string) (string, error) {
			if len(queryParts) == 1 && queryParts[0] != "" {
				return fmt.Sprintf("https://pubsub.googleapis.com/v1/projects/%s/subscriptions", queryParts[0]), nil
			}
			return "", fmt.Errorf("projectID cannot be empty: %v", queryParts)
		},
		UniqueAttributeKeys: []string{"subscriptions"},
	},
	PubSubTopic: {
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_CONFIGURATION,
		Scope:              ScopeProject,
		// https://pubsub.googleapis.com/v1/projects/{project}/topics/{topic}
		GetEndpointBaseURLFunc: func(adapterInitParams ...string) (EndpointFunc, error) {
			if len(adapterInitParams) == 1 && adapterInitParams[0] != "" {
				return func(query string) string {
					if query != "" {
						return fmt.Sprintf("https://pubsub.googleapis.com/v1/projects/%s/topics/%s", adapterInitParams[0], query)
					}
					return ""
				}, nil
			}
			return nil, fmt.Errorf("projectID cannot be empty: %v", adapterInitParams)
		},
		// https://pubsub.googleapis.com/v1/projects/{project}/topics
		ListEndpointFunc: func(queryParts ...string) (string, error) {
			if len(queryParts) == 1 && queryParts[0] != "" {
				return fmt.Sprintf("https://pubsub.googleapis.com/v1/projects/%s/topics", queryParts[0]), nil
			}
			return "", fmt.Errorf("projectID cannot be empty: %v", queryParts)
		},
		UniqueAttributeKeys: []string{"topics"},
	},
	ComputeInstance: {
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_COMPUTE_APPLICATION,
		Scope:              ScopeZonal,
		// https://compute.googleapis.com/compute/v1/projects/{project}/zones/{zone}/instances/{instance}
		GetEndpointBaseURLFunc: func(adapterInitParams ...string) (EndpointFunc, error) {
			if len(adapterInitParams) == 2 && adapterInitParams[0] != "" && adapterInitParams[1] != "" {
				return func(query string) string {
					if query != "" {
						return fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/zones/%s/instances/%s", adapterInitParams[0], adapterInitParams[1], query)
					}
					return ""
				}, nil
			}
			return nil, fmt.Errorf("projectID and zone cannot be empty: %v", adapterInitParams)
		},
		// https://compute.googleapis.com/compute/v1/projects/{project}/zones/{zone}/instances
		ListEndpointFunc: func(queryParts ...string) (string, error) {
			if len(queryParts) == 2 && queryParts[0] != "" && queryParts[1] != "" {
				return fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/zones/%s/instances", queryParts[0], queryParts[1]), nil
			}
			return "", fmt.Errorf("projectID and zone cannot be empty: %v", queryParts)
		},
		UniqueAttributeKeys: []string{"instances"},
	},
}
