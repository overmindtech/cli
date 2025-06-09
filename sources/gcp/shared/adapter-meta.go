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

// We have group of functions that are similar in nature, however they cannot simplified into a generic function because
// of the different number of query parts they accept.
// Also, we want to keep the explicit logic for now for the sake of human readability.
func projectLevelEndpointFuncWithSingleQuery(format string) func(queryParts ...string) (EndpointFunc, error) {
	// count number of `%s` in the format string
	if strings.Count(format, "%s") != 2 { // project ID and query
		panic(fmt.Sprintf("format string must contain 2 %%s placeholders: %s", format))
	}
	return func(adapterInitParams ...string) (EndpointFunc, error) {
		if len(adapterInitParams) == 1 && adapterInitParams[0] != "" {
			return func(query string) string {
				if query != "" {
					// query must be an instance
					return fmt.Sprintf(format, adapterInitParams[0], query)
				}
				return ""
			}, nil
		}
		return nil, fmt.Errorf("projectID cannot be empty: %v", adapterInitParams)
	}
}

func projectLevelEndpointFuncWithTwoQueries(format string) func(queryParts ...string) (EndpointFunc, error) {
	// count number of `%s` in the format string
	if strings.Count(format, "%s") != 3 { // project ID, and 2 parts of the query
		panic(fmt.Sprintf("format string must contain 3 %%s placeholders: %s", format))
	}
	return func(adapterInitParams ...string) (EndpointFunc, error) {
		if len(adapterInitParams) == 1 && adapterInitParams[0] != "" {
			return func(query string) string {
				if query != "" {
					// query must be a composite
					queryParts := strings.Split(query, shared.QuerySeparator)
					if len(queryParts) == 2 && queryParts[0] != "" && queryParts[1] != "" {
						return fmt.Sprintf(format, adapterInitParams[0], queryParts[0], queryParts[1])
					}
				}
				return ""
			}, nil
		}
		return nil, fmt.Errorf("projectID and region cannot be empty: %v", adapterInitParams)
	}
}

func projectLevelEndpointFuncWithThreeQueries(format string) func(queryParts ...string) (EndpointFunc, error) {
	// count number of `%s` in the format string
	if strings.Count(format, "%s") != 4 { // project ID, and 3 parts of the query
		panic(fmt.Sprintf("format string must contain 4 %%s placeholders: %s", format))
	}
	return func(adapterInitParams ...string) (EndpointFunc, error) {
		if len(adapterInitParams) == 1 && adapterInitParams[0] != "" {
			return func(query string) string {
				if query != "" {
					// query must be a composite
					queryParts := strings.Split(query, shared.QuerySeparator)
					if len(queryParts) == 3 && queryParts[0] != "" && queryParts[1] != "" && queryParts[2] != "" {
						return fmt.Sprintf(format, adapterInitParams[0], queryParts[0], queryParts[1], queryParts[2])
					}
				}
				return ""
			}, nil
		}
		return nil, fmt.Errorf("projectID and region cannot be empty: %v", adapterInitParams)
	}
}

func zoneLevelEndpointFuncWithSingleQuery(format string) func(queryParts ...string) (EndpointFunc, error) {
	// count number of `%s` in the format string
	if strings.Count(format, "%s") != 3 { // project ID, zone, and query
		panic(fmt.Sprintf("format string must contain 3 %%s placeholders: %s", format))
	}
	return func(adapterInitParams ...string) (EndpointFunc, error) {
		if len(adapterInitParams) == 2 && adapterInitParams[0] != "" && adapterInitParams[1] != "" {
			return func(query string) string {
				if query != "" {
					// query must be an instance
					return fmt.Sprintf(format, adapterInitParams[0], adapterInitParams[1], query)
				}
				return ""
			}, nil
		}
		return nil, fmt.Errorf("projectID and zone cannot be empty: %v", adapterInitParams)
	}
}

func regionalLevelEndpointFuncWithSingleQuery(format string) func(queryParts ...string) (EndpointFunc, error) {
	// count number of `%s` in the format string
	if strings.Count(format, "%s") != 3 { // project ID, region, and query
		panic(fmt.Sprintf("format string must contain 3 %%s placeholders: %s", format))
	}
	return func(adapterInitParams ...string) (EndpointFunc, error) {
		if len(adapterInitParams) == 2 && adapterInitParams[0] != "" && adapterInitParams[1] != "" {
			return func(query string) string {
				if query != "" {
					// query must be an instance
					return fmt.Sprintf(format, adapterInitParams[0], adapterInitParams[1], query)
				}
				return ""
			}, nil
		}
		return nil, fmt.Errorf("projectID and region cannot be empty: %v", adapterInitParams)
	}
}

func zoneLevelEndpointFuncWithTwoQueries(format string) func(queryParts ...string) (EndpointFunc, error) {
	// count number of `%s` in the format string
	if strings.Count(format, "%s") != 4 { // project ID, zone, and 2 parts of the query
		panic(fmt.Sprintf("format string must contain 4 %%s placeholders: %s", format))
	}
	return func(adapterInitParams ...string) (EndpointFunc, error) {
		if len(adapterInitParams) == 2 && adapterInitParams[0] != "" && adapterInitParams[1] != "" {
			return func(query string) string {
				if query != "" {
					// query must be a composite
					queryParts := strings.Split(query, shared.QuerySeparator)
					if len(queryParts) == 2 && queryParts[0] != "" && queryParts[1] != "" {
						return fmt.Sprintf(format, adapterInitParams[0], adapterInitParams[1], queryParts[0], queryParts[1])
					}
				}
				return ""
			}, nil
		}
		return nil, fmt.Errorf("projectID and zone cannot be empty: %v", adapterInitParams)
	}
}

func regionalLevelEndpointFuncWithTwoQueries(format string) func(queryParts ...string) (EndpointFunc, error) {
	// count number of `%s` in the format string
	if strings.Count(format, "%s") != 4 { // project ID, region, and 2 parts of the query
		panic(fmt.Sprintf("format string must contain 4 %%s placeholders: %s", format))
	}
	return func(adapterInitParams ...string) (EndpointFunc, error) {
		if len(adapterInitParams) == 2 && adapterInitParams[0] != "" && adapterInitParams[1] != "" {
			return func(query string) string {
				if query != "" {
					// query must be a composite
					queryParts := strings.Split(query, shared.QuerySeparator)
					if len(queryParts) == 2 && queryParts[0] != "" && queryParts[1] != "" {
						return fmt.Sprintf(format, adapterInitParams[0], adapterInitParams[1], queryParts[0], queryParts[1])
					}
				}
				return ""
			}, nil
		}
		return nil, fmt.Errorf("projectID and region cannot be empty: %v", adapterInitParams)
	}
}

// SDPAssetTypeToAdapterMeta maps GCP asset types to their corresponding adapter metadata.
var SDPAssetTypeToAdapterMeta = map[shared.ItemType]AdapterMeta{
	AIPlatformCustomJob: {
		// There are multiple service endpoints: https://cloud.google.com/vertex-ai/docs/reference/rest#rest_endpoints
		// We stick to the default one for now: https://aiplatform.googleapis.com
		// Other endpoints are in the form of https://{region}-aiplatform.googleapis.com
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_AI,
		Scope:              ScopeProject,
		// Reference: https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.customJobs/get
		// https://aiplatform.googleapis.com/v1/projects/{project}/locations/{location}/customJobs/{customJob}
		GetEndpointBaseURLFunc: projectLevelEndpointFuncWithTwoQueries("https://aiplatform.googleapis.com/v1/projects/%s/locations/%s/customJobs/%s"),
		// Reference: https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.customJobs/list
		// https://aiplatform.googleapis.com/v1/projects/{project}/locations/{location}/customJobs
		SearchEndpointFunc:  projectLevelEndpointFuncWithSingleQuery("https://aiplatform.googleapis.com/v1/projects/%s/locations/%s/customJobs"),
		UniqueAttributeKeys: []string{"locations", "customJobs"},
	},
	AIPlatformPipelineJob: {
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_AI,
		Scope:              ScopeProject,
		//  Format: projects/{project}/locations/{location}/pipelineJobs/{pipelineJob}
		GetEndpointBaseURLFunc: projectLevelEndpointFuncWithTwoQueries("https://aiplatform.googleapis.com/v1/projects/%s/locations/%s/pipelineJobs/%s"),
		SearchEndpointFunc:     projectLevelEndpointFuncWithSingleQuery("https://aiplatform.googleapis.com/v1/projects/%s/locations/%s/pipelineJobs"),
		UniqueAttributeKeys:    []string{"locations", "pipelineJobs"},
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
	BigTableAdminAppProfile: {
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_CONFIGURATION,
		Scope:              ScopeProject,
		// Reference: https://cloud.google.com/bigtable/docs/reference/admin/rest/v2/projects.instances.appProfiles/get
		// GET https://bigtableadmin.googleapis.com/v2/{name=projects/*/instances/*/appProfiles/*}
		GetEndpointBaseURLFunc: projectLevelEndpointFuncWithTwoQueries("https://bigtableadmin.googleapis.com/v2/projects/%s/instances/%s/appProfiles/%s"),
		// Reference: https://cloud.google.com/bigtable/docs/reference/admin/rest/v2/projects.instances.appProfiles/list
		// GET https://bigtableadmin.googleapis.com/v2/{parent=projects/*/instances/*}/appProfiles
		SearchEndpointFunc:  projectLevelEndpointFuncWithSingleQuery("https://bigtableadmin.googleapis.com/v2/projects/%s/instances/%s/appProfiles"),
		UniqueAttributeKeys: []string{"instances", "appProfiles"},
	},
	BigTableAdminBackup: {
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_OTHER,
		Scope:              ScopeProject,
		// Reference: https://cloud.google.com/bigtable/docs/reference/admin/rest/v2/projects.instances.clusters.backups/get
		// GET https://bigtableadmin.googleapis.com/v2/{name=projects/*/instances/*/clusters/*/backups/*}
		GetEndpointBaseURLFunc: projectLevelEndpointFuncWithThreeQueries("https://bigtableadmin.googleapis.com/v2/projects/%s/instances/%s/clusters/%s/backups/%s"),
		// GET https://bigtableadmin.googleapis.com/v2/{parent=projects/*/instances/*/clusters/*}/backups
		SearchEndpointFunc:  projectLevelEndpointFuncWithTwoQueries("https://bigtableadmin.googleapis.com/v2/parent=projects/%s/instances/%s/clusters/%s/backups"),
		UniqueAttributeKeys: []string{"instances", "clusters", "backups"},
	},
	BigTableAdminTable: {
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_DATABASE,
		Scope:              ScopeProject,
		// Reference: https://cloud.google.com/bigtable/docs/reference/admin/rest/v2/projects.instances.tables/get
		// GET https://bigtableadmin.googleapis.com/v2/{name=projects/*/instances/*/tables/*}
		GetEndpointBaseURLFunc: projectLevelEndpointFuncWithTwoQueries("https://bigtableadmin.googleapis.com/v2/projects/%s/instances/%s/tables/%s"),
		// Reference: https://cloud.google.com/bigtable/docs/reference/admin/rest/v2/projects.instances.tables/list
		// GET https://bigtableadmin.googleapis.com/v2/{parent=projects/*/instances/*}/tables
		SearchEndpointFunc:  projectLevelEndpointFuncWithSingleQuery("https://bigtableadmin.googleapis.com/v2/projects/%s/instances/%s/tables"),
		UniqueAttributeKeys: []string{"instances", "tables"},
	},
	ComputeNetwork: {
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
		Scope:              ScopeProject,
		// https://compute.googleapis.com/compute/v1/projects/{project}/global/networks/{network}
		GetEndpointBaseURLFunc: projectLevelEndpointFuncWithSingleQuery("https://compute.googleapis.com/compute/v1/projects/%s/global/networks/%s"),
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
		GetEndpointBaseURLFunc: regionalLevelEndpointFuncWithSingleQuery("https://compute.googleapis.com/compute/v1/projects/%s/regions/%s/subnetworks/%s"),
		// https://compute.googleapis.com/compute/v1/projects/{project}/regions/{region}/subnetworks
		ListEndpointFunc: func(queryParts ...string) (string, error) {
			if len(queryParts) == 2 && queryParts[0] != "" && queryParts[1] != "" {
				return fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/regions/%s/subnetworks", queryParts[0], queryParts[1]), nil
			}
			return "", fmt.Errorf("projectID and region cannot be empty: %v", queryParts)
		},
		UniqueAttributeKeys: []string{"subnetworks"},
	},
	ComputeInstance: {
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_COMPUTE_APPLICATION,
		Scope:              ScopeZonal,
		// https://compute.googleapis.com/compute/v1/projects/{project}/zones/{zone}/instances/{instance}
		GetEndpointBaseURLFunc: zoneLevelEndpointFuncWithSingleQuery("https://compute.googleapis.com/compute/v1/projects/%s/zones/%s/instances/%s"),
		// https://compute.googleapis.com/compute/v1/projects/{project}/zones/{zone}/instances
		ListEndpointFunc: func(queryParts ...string) (string, error) {
			if len(queryParts) == 2 && queryParts[0] != "" && queryParts[1] != "" {
				return fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/zones/%s/instances", queryParts[0], queryParts[1]), nil
			}
			return "", fmt.Errorf("projectID and zone cannot be empty: %v", queryParts)
		},
		UniqueAttributeKeys: []string{"instances"},
	},
	ComputeInstanceTemplate: {
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_COMPUTE_APPLICATION,
		Scope:              ScopeProject,
		// https://compute.googleapis.com/compute/v1/projects/{project}/global/instanceTemplates/{instanceTemplate}
		GetEndpointBaseURLFunc: projectLevelEndpointFuncWithSingleQuery("https://compute.googleapis.com/compute/v1/projects/%s/global/instanceTemplates/%s"),
		// https://compute.googleapis.com/compute/v1/projects/{project}/global/instanceTemplates
		ListEndpointFunc: func(queryParts ...string) (string, error) {
			if len(queryParts) == 1 && queryParts[0] != "" {
				return fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/global/instanceTemplates", queryParts[0]), nil
			}
			return "", fmt.Errorf("projectID cannot be empty: %v", queryParts)
		},
		UniqueAttributeKeys: []string{"instanceTemplates"},
	},
	ComputeRoute: {
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
		Scope:              ScopeProject,
		// https://compute.googleapis.com/compute/v1/projects/{project}/global/routes/{route}
		GetEndpointBaseURLFunc: projectLevelEndpointFuncWithSingleQuery("https://compute.googleapis.com/compute/v1/projects/%s/global/routes/%s"),
		// https://compute.googleapis.com/compute/v1/projects/{project}/global/routes
		ListEndpointFunc: func(queryParts ...string) (string, error) {
			if len(queryParts) == 1 && queryParts[0] != "" {
				return fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/global/routes", queryParts[0]), nil
			}
			return "", fmt.Errorf("projectID cannot be empty: %v", queryParts)
		},
		UniqueAttributeKeys: []string{"routes"},
	},
	ComputeFirewall: {
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
		Scope:              ScopeProject,
		// https://compute.googleapis.com/compute/v1/projects/{project}/global/firewalls/{firewall}
		GetEndpointBaseURLFunc: projectLevelEndpointFuncWithSingleQuery("https://compute.googleapis.com/compute/v1/projects/%s/global/firewalls/%s"),
		// https://compute.googleapis.com/compute/v1/projects/{project}/global/firewalls
		ListEndpointFunc: func(queryParts ...string) (string, error) {
			if len(queryParts) == 1 && queryParts[0] != "" {
				return fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/global/firewalls", queryParts[0]), nil
			}
			return "", fmt.Errorf("projectID cannot be empty: %v", queryParts)
		},
		UniqueAttributeKeys: []string{"firewalls"},
	},
	ComputeProject: {
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_CONFIGURATION,
		Scope:              ScopeProject,
		// https://cloudresourcemanager.googleapis.com/v1/projects/{project}
		/*
			https://cloud.google.com/compute/docs/reference/rest/v1/projects/get
			To decrease latency for this method, you can optionally omit any unneeded information from the response by using a field mask.
			This practice is especially recommended for unused quota information (the quotas field).
			To exclude one or more fields, set your request's fields query parameter to only include the fields you need.
			For example, to only include the id and selfLink fields, add the query parameter ?fields=id,selfLink to your request.
		*/
		// We only need the name field for this adapter
		// This resource won't carry any attributes to link it to other resources.
		// It will always be a linked item from the other resources by its name.
		GetEndpointBaseURLFunc: func(adapterInitParams ...string) (EndpointFunc, error) {
			// We don't use the project ID here, but we need to ensure that the adapter is initialized with a project ID.
			if len(adapterInitParams) == 1 && adapterInitParams[0] != "" {
				return func(query string) string {
					if query != "" {
						// query must be an instance
						return fmt.Sprintf("https://cloudresourcemanager.googleapis.com/v1/projects/%s?fields=name", query)
					}
					return ""
				}, nil
			}
			return nil, fmt.Errorf("projectID cannot be empty: %v", adapterInitParams)
		},
		UniqueAttributeKeys: []string{"projects"},
	},
	IamRole: {
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_SECURITY,
		Scope:              ScopeProject,
		// Reference: https://cloud.google.com/iam/docs/reference/rest/v1/roles/get
		// https://iam.googleapis.com/v1/projects/{PROJECT_ID}/roles/{CUSTOM_ROLE_ID}
		GetEndpointBaseURLFunc: projectLevelEndpointFuncWithSingleQuery("https://iam.googleapis.com/v1/projects/%s/roles/%s"),
		// Reference: https://cloud.google.com/iam/docs/reference/rest/v1/roles/list
		// https://iam.googleapis.com/v1/projects/{PROJECT_ID}/roles
		ListEndpointFunc: func(adapterInitParams ...string) (string, error) {
			if len(adapterInitParams) == 1 && adapterInitParams[0] != "" {
				return fmt.Sprintf("https://iam.googleapis.com/v1/projects/%s/roles", adapterInitParams[0]), nil
			}
			return "", fmt.Errorf("projectID cannot be empty: %v", adapterInitParams)
		},
		UniqueAttributeKeys: []string{"roles"},
	},
	PubSubSubscription: {
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_CONFIGURATION,
		Scope:              ScopeProject,
		// https://pubsub.googleapis.com/v1/projects/{project}/subscriptions/{subscription}
		GetEndpointBaseURLFunc: projectLevelEndpointFuncWithSingleQuery("https://pubsub.googleapis.com/v1/projects/%s/subscriptions/%s"),
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
		GetEndpointBaseURLFunc: projectLevelEndpointFuncWithSingleQuery("https://pubsub.googleapis.com/v1/projects/%s/topics/%s"),
		// https://pubsub.googleapis.com/v1/projects/{project}/topics
		ListEndpointFunc: func(queryParts ...string) (string, error) {
			if len(queryParts) == 1 && queryParts[0] != "" {
				return fmt.Sprintf("https://pubsub.googleapis.com/v1/projects/%s/topics", queryParts[0]), nil
			}
			return "", fmt.Errorf("projectID cannot be empty: %v", queryParts)
		},
		UniqueAttributeKeys: []string{"topics"},
	},
}
