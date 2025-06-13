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
	// We will normally generate the search description from the UniqueAttributeKeys
	// but we allow it to be overridden for specific adapters.
	SearchDescription   string
	SDPAdapterCategory  sdp.AdapterCategory
	UniqueAttributeKeys []string
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

func projectLevelEndpointFuncWithFourQueries(format string) func(queryParts ...string) (EndpointFunc, error) {
	// count number of `%s` in the format string
	if strings.Count(format, "%s") != 5 { // project ID, and 4 parts of the query
		panic(fmt.Sprintf("format string must contain 5 %%s placeholders: %s", format))
	}
	return func(adapterInitParams ...string) (EndpointFunc, error) {
		if len(adapterInitParams) == 1 && adapterInitParams[0] != "" {
			return func(query string) string {
				if query != "" {
					// query must be a composite
					queryParts := strings.Split(query, shared.QuerySeparator)
					if len(queryParts) == 4 && queryParts[0] != "" && queryParts[1] != "" && queryParts[2] != "" && queryParts[3] != "" {
						return fmt.Sprintf(format, adapterInitParams[0], queryParts[0], queryParts[1], queryParts[2], queryParts[3])
					}
				}
				return ""
			}, nil
		}
		return nil, fmt.Errorf("projectID cannot be empty: %v", adapterInitParams)
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

func projectLevelListFunc(format string) func(adapterInitParams ...string) (string, error) {
	if strings.Count(format, "%s") != 1 {
		panic(fmt.Sprintf("format string must contain 1 %%s placeholder: %s", format))
	}
	return func(adapterInitParams ...string) (string, error) {
		if len(adapterInitParams) == 1 && adapterInitParams[0] != "" {
			return fmt.Sprintf(format, adapterInitParams[0]), nil
		}
		return "", fmt.Errorf("projectID cannot be empty: %v", adapterInitParams)
	}
}

func regionLevelListFunc(format string) func(adapterInitParams ...string) (string, error) {
	// count number of `%s` in the format string
	if strings.Count(format, "%s") != 2 { // project ID and region
		panic(fmt.Sprintf("format string must contain 2 %%s placeholders: %s", format))
	}
	return func(adapterInitParams ...string) (string, error) {
		if len(adapterInitParams) == 2 && adapterInitParams[0] != "" && adapterInitParams[1] != "" {
			return fmt.Sprintf(format, adapterInitParams[0], adapterInitParams[1]), nil
		}
		return "", fmt.Errorf("projectID and region cannot be empty: %v", adapterInitParams)
	}
}

func zoneLevelListFunc(format string) func(adapterInitParams ...string) (string, error) {
	// count number of `%s` in the format string
	if strings.Count(format, "%s") != 2 { // project ID and zone
		panic(fmt.Sprintf("format string must contain 2 %%s placeholders: %s", format))
	}
	return func(adapterInitParams ...string) (string, error) {
		if len(adapterInitParams) == 2 && adapterInitParams[0] != "" && adapterInitParams[1] != "" {
			return fmt.Sprintf(format, adapterInitParams[0], adapterInitParams[1]), nil
		}
		return "", fmt.Errorf("projectID and zone cannot be empty: %v", adapterInitParams)
	}
}

// SDPAssetTypeToAdapterMeta maps GCP asset types to their corresponding adapter metadata.
var SDPAssetTypeToAdapterMeta = map[shared.ItemType]AdapterMeta{
	AIPlatformCustomJob: {
		// There are multiple service endpoints: https://cloud.google.com/vertex-ai/docs/reference/rest#rest_endpoints
		// We stick to the default one for now: https://aiplatform.googleapis.com
		// Other endpoints are in the form of https://{region}-aiplatform.googleapis.com
		// If we use the default endpoint the location must be set to `global`.
		// So, for simplicity, we can get custom jobs by their name globally, list globally,
		// otherwise we have to check the validity of the location and use the regional endpoint.
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_AI,
		Scope:              ScopeProject,
		// Vertex AI API must be enabled for the project!
		// Reference: https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.customJobs/get
		// https://aiplatform.googleapis.com/v1/projects/{project}/locations/{location}/customJobs/{customJob}
		GetEndpointBaseURLFunc: projectLevelEndpointFuncWithSingleQuery("https://aiplatform.googleapis.com/v1/projects/%s/locations/global/customJobs/%s"),
		// Reference: https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.customJobs/list
		// https://aiplatform.googleapis.com/v1/projects/{project}/locations/{location}/customJobs
		// Expected location is `global` for the default endpoint.
		ListEndpointFunc:    projectLevelListFunc("https://aiplatform.googleapis.com/v1/projects/%s/locations/global/customJobs"),
		UniqueAttributeKeys: []string{"customJobs"},
	},
	AIPlatformPipelineJob: {
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_AI,
		Scope:              ScopeProject,
		// When using the default endpoint, the location must be set to `global`.
		//  Format: projects/{project}/locations/{location}/pipelineJobs/{pipelineJob}
		GetEndpointBaseURLFunc: projectLevelEndpointFuncWithSingleQuery("https://aiplatform.googleapis.com/v1/projects/%s/locations/global/pipelineJobs/%s"),
		// Reference: https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.pipelineJobs/list
		ListEndpointFunc:    projectLevelListFunc("https://aiplatform.googleapis.com/v1/projects/%s/locations/global/pipelineJobs"),
		UniqueAttributeKeys: []string{"pipelineJobs"},
	},
	ArtifactRegistryDockerImage: {
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_STORAGE,
		Scope:              ScopeProject,
		// Reference: https://cloud.google.com/artifact-registry/docs/reference/rest/v1/projects.locations.repositories.dockerImages/get?rep_location=global
		// GET https://artifactregistry.googleapis.com/v1/{name=projects/*/locations/*/repositories/*/dockerImages/*}
		// IAM permissions: artifactregistry.dockerImages.get
		GetEndpointBaseURLFunc: projectLevelEndpointFuncWithThreeQueries("https://artifactregistry.googleapis.com/v1/projects/%s/locations/%s/repositories/%s/dockerImages/%s"),
		// Reference: https://cloud.google.com/artifact-registry/docs/reference/rest/v1/projects.locations.repositories.dockerImages/list?rep_location=global
		// GET https://artifactregistry.googleapis.com/v1/{parent=projects/*/locations/*/repositories/*}/dockerImages
		// IAM permissions: artifactregistry.dockerImages.list
		SearchEndpointFunc: func(adapterInitParams ...string) (EndpointFunc, error) {
			if len(adapterInitParams) == 1 && adapterInitParams[0] != "" {
				return func(query string) string {
					if strings.Contains(query, "/") {
						// That means this is coming from terraform mapping, and the query is in the form of
						// projects/{{project}}/locations/{{location}}/repository/{{repository_id}}/dockerImages/{{docker_image}}
						// We need to extract the relevant parts and construct the URL accordingly
						parts := strings.Split(strings.TrimPrefix(query, "/"), "/")
						if len(parts) == 8 {
							// 3: location
							// 5: repository_id
							// 7: docker_image
							return fmt.Sprintf("https://artifactregistry.googleapis.com/v1/projects/%s/locations/%s/repositories/%s/dockerImages/%s", adapterInitParams[0], parts[3], parts[5], parts[7])
						}
						return ""
					}
					if query != "" {
						// This is a regular query coming from user interaction, and it should be in the form of
						// {{location}}|{{repository_id}}
						queryParts := strings.Split(query, shared.QuerySeparator)
						if len(queryParts) == 2 && queryParts[0] != "" && queryParts[1] != "" {
							return fmt.Sprintf("https://artifactregistry.googleapis.com/v1/projects/%s/locations/%s/repositories/%s/dockerImages", adapterInitParams[0], queryParts[0], queryParts[1])
						}
						return ""
					}
					return ""
				}, nil
			}
			return nil, fmt.Errorf("projectID cannot be empty: %v", adapterInitParams)
		},
		SearchDescription:   "Search for Docker images in Artifact Registry. Use the format {{location}}|{{repository_id}} or projects/{{project}}/locations/{{location}}/repository/{{repository_id}}/dockerImages/{{docker_image}} which is supported for terraform mappings.",
		UniqueAttributeKeys: []string{"locations", "repositories", "dockerImages"},
	},
	BigQueryDataset: {
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_DATABASE,
		Scope:              ScopeProject,
		// https://bigquery.googleapis.com/bigquery/v2/projects/{projectId}/datasets/{datasetId}
		GetEndpointBaseURLFunc: projectLevelEndpointFuncWithSingleQuery("https://bigquery.googleapis.com/bigquery/v2/projects/%s/datasets/%s"),
		// Reference: https://cloud.google.com/bigquery/docs/reference/rest/v2/datasets/list
		// https://bigquery.googleapis.com/bigquery/v2/projects/{projectId}/datasets
		ListEndpointFunc:    projectLevelListFunc("https://bigquery.googleapis.com/bigquery/v2/projects/%s/datasets"),
		UniqueAttributeKeys: []string{"datasets"},
	},
	BigQueryTable: {
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_DATABASE,
		Scope:              ScopeProject,
		// https://bigquery.googleapis.com/bigquery/v2/projects/{projectId}/datasets/{datasetId}/tables/{tableId}
		GetEndpointBaseURLFunc: projectLevelEndpointFuncWithTwoQueries("https://bigquery.googleapis.com/bigquery/v2/projects/%s/datasets/%s/tables/%s"),
		// Reference: https://cloud.google.com/bigquery/docs/reference/rest/v2/tables/list
		// https://bigquery.googleapis.com/bigquery/v2/projects/{projectId}/datasets/{datasetId}/tables
		// TODO: Update this for => https://linear.app/overmind/issue/ENG-580/handle-terraform-mappings-in-search-method
		// id => projects/{{project}}/datasets/{{dataset}}/tables/{{table}}
		SearchEndpointFunc:  projectLevelEndpointFuncWithSingleQuery("https://bigquery.googleapis.com/bigquery/v2/projects/%s/datasets/%s/tables"),
		SearchDescription:   "Search for BigQuery tables in a dataset. Use the format {{dataset}} or projects/{{project}}/datasets/{{dataset}}/tables/{{table}} which is supported for terraform mappings.",
		UniqueAttributeKeys: []string{"datasets", "tables"},
	},
	BigTableAdminAppProfile: {
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_CONFIGURATION,
		Scope:              ScopeProject,
		// Reference: https://cloud.google.com/bigtable/docs/reference/admin/rest/v2/projects.instances.appProfiles/get
		// GET https://bigtableadmin.googleapis.com/v2/{name=projects/*/instances/*/appProfiles/*}
		GetEndpointBaseURLFunc: projectLevelEndpointFuncWithTwoQueries("https://bigtableadmin.googleapis.com/v2/projects/%s/instances/%s/appProfiles/%s"),
		// Reference: https://cloud.google.com/bigtable/docs/reference/admin/rest/v2/projects.instances.appProfiles/list
		// GET https://bigtableadmin.googleapis.com/v2/{parent=projects/*/instances/*}/appProfiles
		// TODO: Update this for => https://linear.app/overmind/issue/ENG-580/handle-terraform-mappings-in-search-method
		// id => projects/{{project}}/instances/{{instance}}/appProfiles/{{app_profile_id}}
		SearchEndpointFunc:  projectLevelEndpointFuncWithSingleQuery("https://bigtableadmin.googleapis.com/v2/projects/%s/instances/%s/appProfiles"),
		SearchDescription:   "Search for BigTable App Profiles in an instance. Use the format {{instance}} or projects/{{project}}/instances/{{instance}}/appProfiles/{{app_profile_id}} which is supported for terraform mappings.",
		UniqueAttributeKeys: []string{"instances", "appProfiles"},
	},
	BigTableAdminBackup: {
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_OTHER,
		Scope:              ScopeProject,
		// Reference: https://cloud.google.com/bigtable/docs/reference/admin/rest/v2/projects.instances.clusters.backups/get
		// GET https://bigtableadmin.googleapis.com/v2/{name=projects/*/instances/*/clusters/*/backups/*}
		GetEndpointBaseURLFunc: projectLevelEndpointFuncWithThreeQueries("https://bigtableadmin.googleapis.com/v2/projects/%s/instances/%s/clusters/%s/backups/%s"),
		// GET https://bigtableadmin.googleapis.com/v2/{parent=projects/*/instances/*/clusters/*}/backups
		SearchEndpointFunc:  projectLevelEndpointFuncWithTwoQueries("https://bigtableadmin.googleapis.com/v2/projects/%s/instances/%s/clusters/%s/backups"),
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
		// TODO: Update this for => https://linear.app/overmind/issue/ENG-580/handle-terraform-mappings-in-search-method
		// id => projects/{{project}}/instances/{{instance_name}}/tables/{{name}}
		SearchEndpointFunc:  projectLevelEndpointFuncWithSingleQuery("https://bigtableadmin.googleapis.com/v2/projects/%s/instances/%s/tables"),
		SearchDescription:   "Search for BigTable tables in an instance. Use the format {{instance_name}} or projects/{{project}}/instances/{{instance_name}}/tables/{{name}} which is supported for terraform mappings.",
		UniqueAttributeKeys: []string{"instances", "tables"},
	},
	CloudBillingBillingInfo: {
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_CONFIGURATION,
		Scope:              ScopeProject,
		// Reference: https://cloud.google.com/billing/docs/reference/rest/v1/projects/getBillingInfo
		// Gets the billing information for a project.
		// GET https://cloudbilling.googleapis.com/v1/{name=projects/*}/billingInfo
		// IAM permissions: resourcemanager.projects.get
		GetEndpointBaseURLFunc: func(adapterInitParams ...string) (EndpointFunc, error) {
			if len(adapterInitParams) == 1 && adapterInitParams[0] != "" {
				return func(query string) string {
					if query != "" {
						return fmt.Sprintf("https://cloudbilling.googleapis.com/v1/projects/%s/billingInfo", query)
					}
					return ""
				}, nil
			}
			return nil, fmt.Errorf("projectID cannot be empty: %v", adapterInitParams)
		},
		UniqueAttributeKeys: []string{"billingInfo"},
	},
	CloudBuildBuild: {
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_CONFIGURATION,
		Scope:              ScopeProject,
		// Reference: https://cloud.google.com/build/docs/api/reference/rest/v1/projects.builds/get
		// GET https://cloudbuild.googleapis.com/v1/projects/{projectId}/builds/{id}
		GetEndpointBaseURLFunc: projectLevelEndpointFuncWithSingleQuery("https://cloudbuild.googleapis.com/v1/projects/%s/builds/%s"),
		// Reference: https://cloud.google.com/build/docs/api/reference/rest/v1/projects.builds/list
		// GET https://cloudbuild.googleapis.com/v1/projects/{projectId}/builds
		ListEndpointFunc:    projectLevelListFunc("https://cloudbuild.googleapis.com/v1/projects/%s/builds"),
		UniqueAttributeKeys: []string{"builds"},
	},
	CloudResourceManagerProject: {
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_CONFIGURATION,
		Scope:              ScopeProject,
		// Reference: https://cloud.google.com/resource-manager/reference/rest/v3/projects/get
		// GET https://cloudresourcemanager.googleapis.com/v3/projects/*
		// IAM permissions: resourcemanager.projects.get
		GetEndpointBaseURLFunc: func(adapterInitParams ...string) (EndpointFunc, error) {
			if len(adapterInitParams) == 1 && adapterInitParams[0] != "" {
				return func(query string) string {
					if query != "" {
						return fmt.Sprintf("https://cloudresourcemanager.googleapis.com/v3/projects/%s", query)
					}
					return ""
				}, nil
			}
			return nil, fmt.Errorf("projectID cannot be empty: %v", adapterInitParams)
		},
		UniqueAttributeKeys: []string{"projects"},
	},
	ComputeFirewall: {
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
		Scope:              ScopeProject,
		// https://compute.googleapis.com/compute/v1/projects/{project}/global/firewalls/{firewall}
		GetEndpointBaseURLFunc: projectLevelEndpointFuncWithSingleQuery("https://compute.googleapis.com/compute/v1/projects/%s/global/firewalls/%s"),
		// Reference: https://cloud.google.com/compute/docs/reference/rest/v1/firewalls/list
		// https://compute.googleapis.com/compute/v1/projects/{project}/global/firewalls
		ListEndpointFunc:    projectLevelListFunc("https://compute.googleapis.com/compute/v1/projects/%s/global/firewalls"),
		UniqueAttributeKeys: []string{"firewalls"},
	},
	ComputeInstance: {
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_COMPUTE_APPLICATION,
		Scope:              ScopeZonal,
		// https://compute.googleapis.com/compute/v1/projects/{project}/zones/{zone}/instances/{instance}
		GetEndpointBaseURLFunc: zoneLevelEndpointFuncWithSingleQuery("https://compute.googleapis.com/compute/v1/projects/%s/zones/%s/instances/%s"),
		// https://compute.googleapis.com/compute/v1/projects/{project}/zones/{zone}/instances
		ListEndpointFunc:    zoneLevelListFunc("https://compute.googleapis.com/compute/v1/projects/%s/zones/%s/instances"),
		UniqueAttributeKeys: []string{"instances"},
	},
	ComputeInstanceTemplate: {
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_COMPUTE_APPLICATION,
		Scope:              ScopeProject,
		// https://compute.googleapis.com/compute/v1/projects/{project}/global/instanceTemplates/{instanceTemplate}
		GetEndpointBaseURLFunc: projectLevelEndpointFuncWithSingleQuery("https://compute.googleapis.com/compute/v1/projects/%s/global/instanceTemplates/%s"),
		// https://compute.googleapis.com/compute/v1/projects/{project}/global/instanceTemplates
		ListEndpointFunc:    projectLevelListFunc("https://compute.googleapis.com/compute/v1/projects/%s/global/instanceTemplates"),
		UniqueAttributeKeys: []string{"instanceTemplates"},
	},
	ComputeNetwork: {
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
		Scope:              ScopeProject,
		// https://compute.googleapis.com/compute/v1/projects/{project}/global/networks/{network}
		GetEndpointBaseURLFunc: projectLevelEndpointFuncWithSingleQuery("https://compute.googleapis.com/compute/v1/projects/%s/global/networks/%s"),
		// https://compute.googleapis.com/compute/v1/projects/{project}/global/networks
		ListEndpointFunc:    projectLevelListFunc("https://compute.googleapis.com/compute/v1/projects/%s/global/networks"),
		UniqueAttributeKeys: []string{"networks"},
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
	ComputeRoute: {
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
		Scope:              ScopeProject,
		// https://compute.googleapis.com/compute/v1/projects/{project}/global/routes/{route}
		GetEndpointBaseURLFunc: projectLevelEndpointFuncWithSingleQuery("https://compute.googleapis.com/compute/v1/projects/%s/global/routes/%s"),
		// https://compute.googleapis.com/compute/v1/projects/{project}/global/routes
		ListEndpointFunc:    projectLevelListFunc("https://compute.googleapis.com/compute/v1/projects/%s/global/routes"),
		UniqueAttributeKeys: []string{"routes"},
	},
	ComputeSubnetwork: {
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
		Scope:              ScopeRegional,
		// https://compute.googleapis.com/compute/v1/projects/{project}/regions/{region}/subnetworks/{subnetwork}
		GetEndpointBaseURLFunc: regionalLevelEndpointFuncWithSingleQuery("https://compute.googleapis.com/compute/v1/projects/%s/regions/%s/subnetworks/%s"),
		// https://compute.googleapis.com/compute/v1/projects/{project}/regions/{region}/subnetworks
		ListEndpointFunc:    regionLevelListFunc("https://compute.googleapis.com/compute/v1/projects/%s/regions/%s/subnetworks"),
		UniqueAttributeKeys: []string{"subnetworks"},
	},
	DataformRepository: {
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_DATABASE,
		Scope:              ScopeProject,
		// Reference: https://cloud.google.com/dataform/reference/rest/v1/projects.locations.repositories/get
		// GET https://dataform.googleapis.com/v1/projects/*/locations/*/repositories/*
		// IAM permissions: dataform.repositories.get
		GetEndpointBaseURLFunc: projectLevelEndpointFuncWithTwoQueries("https://dataform.googleapis.com/v1/projects/%s/locations/%s/repositories/%s"),
		// Reference: https://cloud.google.com/dataform/reference/rest/v1/projects.locations.repositories/list
		// GET https://dataform.googleapis.com/v1/projects/*/locations/*/repositories
		// IAM permissions: dataform.repositories.list
		// TODO: Update this for => https://linear.app/overmind/issue/ENG-580/handle-terraform-mappings-in-search-method
		// id => projects/{{project}}/locations/{{region}}/repositories/{{name}}
		SearchEndpointFunc:  projectLevelEndpointFuncWithSingleQuery("https://dataform.googleapis.com/v1/projects/%s/locations/%s/repositories"),
		SearchDescription:   "Search for Dataform repositories in a location. Use the format {{location}} or projects/{{project}}/locations/{{location}}/repositories/{{name}} which is supported for terraform mappings.",
		UniqueAttributeKeys: []string{"locations", "repositories"},
	},
	DataplexEntryGroup: {
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_STORAGE,
		Scope:              ScopeProject,
		// Reference: https://cloud.google.com/dataplex/docs/reference/rest/v1/projects.locations.entryGroups/get
		// GET https://dataplex.googleapis.com/v1/{name=projects/*/locations/*/entryGroups/*}
		GetEndpointBaseURLFunc: projectLevelEndpointFuncWithTwoQueries("https://dataplex.googleapis.com/v1/projects/%s/locations/%s/entryGroups/%s"),
		// Reference: https://cloud.google.com/dataplex/docs/reference/rest/v1/projects.locations.entryGroups/list
		// GET https://dataplex.googleapis.com/v1/{parent=projects/*/locations/*}/entryGroups
		// TODO: Update this for => https://linear.app/overmind/issue/ENG-580/handle-terraform-mappings-in-search-method
		// id => projects/{{project}}/locations/{{location}}/entryGroups/{{entry_group_id}}
		SearchEndpointFunc:  projectLevelEndpointFuncWithSingleQuery("https://dataplex.googleapis.com/v1/projects/%s/locations/%s/entryGroups"),
		SearchDescription:   "Search for Dataplex entry groups in a location. Use the format {{location}} or projects/{{project}}/locations/{{location}}/entryGroups/{{entry_group_id}} which is supported for terraform mappings.",
		UniqueAttributeKeys: []string{"locations", "entryGroups"},
	},
	DNSManagedZone: {
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
		Scope:              ScopeProject,
		// Reference: https://cloud.google.com/dns/docs/reference/rest/v1/managedZones/get
		// GET https://dns.googleapis.com/dns/v1/projects/{project}/managedZones/{managedZone}
		GetEndpointBaseURLFunc: projectLevelEndpointFuncWithSingleQuery("https://dns.googleapis.com/dns/v1/projects/%s/managedZones/%s"),
		// Reference: https://cloud.google.com/dns/docs/reference/rest/v1/managedZones/list
		// GET https://dns.googleapis.com/dns/v1/projects/{project}/managedZones
		ListEndpointFunc:    projectLevelListFunc("https://dns.googleapis.com/dns/v1/projects/%s/managedZones"),
		UniqueAttributeKeys: []string{"managedZones"},
	},
	EssentialContactsContact: {
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_OTHER,
		Scope:              ScopeProject,
		// Reference: https://cloud.google.com/resource-manager/docs/reference/essentialcontacts/rest/v1/projects.contacts/get
		// GET https://essentialcontacts.googleapis.com/v1/projects/*/contacts/*
		// IAM permissions: essentialcontacts.contacts.get
		GetEndpointBaseURLFunc: projectLevelEndpointFuncWithSingleQuery("https://essentialcontacts.googleapis.com/v1/projects/%s/contacts/%s"),
		// Reference: https://cloud.google.com/resource-manager/docs/reference/essentialcontacts/rest/v1/projects.contacts/list
		// GET https://essentialcontacts.googleapis.com/v1/projects/*/contacts
		// IAM permissions: essentialcontacts.contacts.list
		ListEndpointFunc: projectLevelListFunc("https://essentialcontacts.googleapis.com/v1/projects/%s/contacts"),
		// This is for terraform mapping, where the query is in the form of
		// projects/{projectId}/contacts/{contact_id}
		SearchEndpointFunc: func(adapterInitParams ...string) (EndpointFunc, error) {
			if len(adapterInitParams) == 1 && adapterInitParams[0] != "" {
				return func(query string) string {
					if strings.Contains(query, "/") {
						// That means this is coming from terraform mapping, and the query is in the form of
						// projects/{projectId}/contacts/{contact_id}
						// We need to extract the relevant parts and construct the URL accordingly
						values := ExtractPathParams(query, "projects", "contacts")
						if len(values) == 2 {
							return fmt.Sprintf("https://essentialcontacts.googleapis.com/v1/projects/%s/contacts/%s", values[0], values[1])
						}
						return ""
					}
					return ""
				}, nil
			}
			return nil, fmt.Errorf("projectID cannot be empty: %v", adapterInitParams)
		},
		SearchDescription:   "Search for contacts by their ID in the form of projects/{projectId}/contacts/{contact_id}.",
		UniqueAttributeKeys: []string{"contacts"},
	},
	IAMRole: {
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_SECURITY,
		Scope:              ScopeProject,
		// Reference: https://cloud.google.com/iam/docs/reference/rest/v1/roles/get
		// https://iam.googleapis.com/v1/projects/{PROJECT_ID}/roles/{CUSTOM_ROLE_ID}
		GetEndpointBaseURLFunc: projectLevelEndpointFuncWithSingleQuery("https://iam.googleapis.com/v1/projects/%s/roles/%s"),
		// Reference: https://cloud.google.com/iam/docs/reference/rest/v1/roles/list
		// https://iam.googleapis.com/v1/projects/{PROJECT_ID}/roles
		ListEndpointFunc:    projectLevelListFunc("https://iam.googleapis.com/v1/projects/%s/roles"),
		UniqueAttributeKeys: []string{"roles"},
	},
	LoggingBucket: {
		// global is a type of location.
		// location is generally a region.
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_OBSERVABILITY,
		Scope:              ScopeProject,
		// Reference: https://cloud.google.com/logging/docs/reference/v2/rest/v2/projects.locations.buckets/get
		// GET https://logging.googleapis.com/v2/projects/*/locations/*/buckets/*
		// IAM permissions: logging.buckets.get
		GetEndpointBaseURLFunc: projectLevelEndpointFuncWithTwoQueries("https://logging.googleapis.com/v2/projects/%s/locations/%s/buckets/%s"),
		// Reference: https://cloud.google.com/logging/docs/reference/v2/rest/v2/projects.locations.buckets/list
		// GET https://logging.googleapis.com/v2/projects/*/locations/*/buckets
		// IAM permissions: logging.buckets.list
		SearchEndpointFunc:  projectLevelEndpointFuncWithSingleQuery("https://logging.googleapis.com/v2/projects/%s/locations/%s/buckets"),
		UniqueAttributeKeys: []string{"locations", "buckets"},
	},
	LoggingLink: {
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_OBSERVABILITY,
		Scope:              ScopeProject,
		// Reference: https://cloud.google.com/logging/docs/reference/v2/rest/v2/projects.locations.buckets.links/get
		// GET https://logging.googleapis.com/v2/projects/*/locations/*/buckets/*/links/*
		// IAM permissions: logging.links.get
		GetEndpointBaseURLFunc: projectLevelEndpointFuncWithThreeQueries("https://logging.googleapis.com/v2/projects/%s/locations/%s/buckets/%s/links/%s"),
		// Reference: https://cloud.google.com/logging/docs/reference/v2/rest/v2/projects.locations.buckets.links/list
		// GET https://logging.googleapis.com/v2/projects/*/locations/*/buckets/*/links
		// IAM permissions: logging.links.list
		SearchEndpointFunc:  projectLevelEndpointFuncWithTwoQueries("https://logging.googleapis.com/v2/projects/%s/locations/%s/buckets/%s/links"),
		UniqueAttributeKeys: []string{"locations", "buckets", "links"},
	},
	LoggingSavedQuery: {
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_OBSERVABILITY,
		Scope:              ScopeProject,
		// Reference: https://cloud.google.com/logging/docs/reference/v2/rest/v2/projects.locations.savedQueries/get
		// GET https://logging.googleapis.com/v2/projects/*/locations/*/savedQueries/*
		// IAM permissions: logging.savedQueries.get
		GetEndpointBaseURLFunc: projectLevelEndpointFuncWithTwoQueries("https://logging.googleapis.com/v2/projects/%s/locations/%s/savedQueries/%s"),
		// Reference: https://cloud.google.com/logging/docs/reference/v2/rest/v2/projects.locations.savedQueries/list
		// GET https://logging.googleapis.com/v2/projects/*/locations/*/savedQueries
		// IAM permissions: logging.savedQueries.list
		// Saved Query has to be shared with the project (opposite is a private one) to show up here.
		SearchEndpointFunc:  projectLevelEndpointFuncWithSingleQuery("https://logging.googleapis.com/v2/projects/%s/locations/%s/savedQueries"),
		UniqueAttributeKeys: []string{"locations", "savedQueries"},
	},
	LoggingSink: {
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_OBSERVABILITY,
		Scope:              ScopeProject,
		// Reference: https://cloud.google.com/logging/docs/reference/v2/rest/v2/projects.sinks/get
		// GET https://logging.googleapis.com/v2/projects/*/sinks/*
		// IAM permissions: logging.sinks.get
		GetEndpointBaseURLFunc: projectLevelEndpointFuncWithSingleQuery("https://logging.googleapis.com/v2/projects/%s/sinks/%s"),
		// Reference: https://cloud.google.com/logging/docs/reference/v2/rest/v2/projects.sinks/list
		// GET https://logging.googleapis.com/v2/projects/*/sinks
		// IAM permissions: logging.sinks.list
		ListEndpointFunc:    projectLevelListFunc("https://logging.googleapis.com/v2/projects/%s/sinks"),
		UniqueAttributeKeys: []string{"sinks"},
	},
	MonitoringCustomDashboard: {
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_OBSERVABILITY,
		Scope:              ScopeProject,
		// Reference: https://cloud.google.com/monitoring/api/ref_v3/rest/v1/projects.dashboards/get
		// GET https://monitoring.googleapis.com/v1/projects/[PROJECT_ID_OR_NUMBER]/dashboards/[DASHBOARD_ID] (for custom dashboards).
		// IAM Perm: monitoring.dashboards.get
		GetEndpointBaseURLFunc: projectLevelEndpointFuncWithSingleQuery("https://monitoring.googleapis.com/v1/projects/%s/dashboards/%s"),
		// Reference: https://cloud.google.com/monitoring/api/ref_v3/rest/v1/projects.dashboards/list
		// GET https://monitoring.googleapis.com/v1/{parent}/dashboards
		// IAM Perm: monitoring.dashboards.list
		ListEndpointFunc:  projectLevelListFunc("https://monitoring.googleapis.com/v1/projects/%s/dashboards"),
		SearchDescription: "Search for custom dashboards by their ID in the form of projects/{projectId}/dashboards/{dashboard_id}. This is supported for terraform mappings.",
		SearchEndpointFunc: func(adapterInitParams ...string) (EndpointFunc, error) {
			if len(adapterInitParams) == 1 && adapterInitParams[0] != "" {
				return func(query string) string {
					if strings.Contains(query, "/") {
						// That means this is coming from terraform mapping, and the query is in the form of
						// projects/{projectId}/dashboards/{dashboard_id}
						// We need to extract the relevant parts and construct the URL accordingly
						values := ExtractPathParams(query, "projects", "dashboards")
						if len(values) == 2 {
							return fmt.Sprintf("https://monitoring.googleapis.com/v1/projects/%s/dashboards/%s", values[0], values[1])
						}
						return ""
					}
					return ""
				}, nil
			}
			return nil, fmt.Errorf("projectID cannot be empty: %v", adapterInitParams)
		},
		UniqueAttributeKeys: []string{"dashboards"},
	},
	PubSubSubscription: {
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_CONFIGURATION,
		Scope:              ScopeProject,
		// https://pubsub.googleapis.com/v1/projects/{project}/subscriptions/{subscription}
		GetEndpointBaseURLFunc: projectLevelEndpointFuncWithSingleQuery("https://pubsub.googleapis.com/v1/projects/%s/subscriptions/%s"),
		// Reference: https://cloud.google.com/pubsub/docs/reference/rest/v1/projects.subscriptions/list?rep_location=global
		// https://pubsub.googleapis.com/v1/projects/{project}/subscriptions
		ListEndpointFunc:    projectLevelListFunc("https://pubsub.googleapis.com/v1/projects/%s/subscriptions"),
		UniqueAttributeKeys: []string{"subscriptions"},
	},
	PubSubTopic: {
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_CONFIGURATION,
		Scope:              ScopeProject,
		// https://pubsub.googleapis.com/v1/projects/{project}/topics/{topic}
		GetEndpointBaseURLFunc: projectLevelEndpointFuncWithSingleQuery("https://pubsub.googleapis.com/v1/projects/%s/topics/%s"),
		// https://pubsub.googleapis.com/v1/projects/{project}/topics
		ListEndpointFunc:    projectLevelListFunc("https://pubsub.googleapis.com/v1/projects/%s/topics"),
		UniqueAttributeKeys: []string{"topics"},
	},
	RunRevision: {
		/*
			A Revision is an immutable snapshot of code and configuration.
			A Revision references a container image.
			Revisions are only created by updates to its parent Service.
		*/
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_CONFIGURATION,
		Scope:              ScopeProject,
		// Reference: https://cloud.google.com/run/docs/reference/rest/v2/projects.locations.services.revisions/get
		// GET https://run.googleapis.com/v2/projects/{project}/locations/{location}/services/{service}/revisions/{revision}
		// IAM Perm: run.revisions.get
		GetEndpointBaseURLFunc: projectLevelEndpointFuncWithThreeQueries("https://run.googleapis.com/v2/projects/%s/locations/%s/services/%s/revisions/%s"),
		// Reference: https://cloud.google.com/run/docs/reference/rest/v2/projects.locations.services.revisions/list
		// GET https://run.googleapis.com/v2/projects/{project}/locations/{location}/services/{service}/revisions
		// IAM Perm: run.revisions.list
		SearchEndpointFunc:  projectLevelEndpointFuncWithTwoQueries("https://run.googleapis.com/v2/projects/%s/locations/%s/services/%s/revisions"),
		UniqueAttributeKeys: []string{"locations", "services", "revisions"},
	},
	ServiceDirectoryEndpoint: {
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_CONFIGURATION,
		Scope:              ScopeProject,
		// Reference: https://cloud.google.com/service-directory/docs/reference/rest/v1/projects.locations.namespaces.services.endpoints/get
		// GET https://servicedirectory.googleapis.com/v1/projects/*/locations/*/namespaces/*/services/*/endpoints/*
		GetEndpointBaseURLFunc: projectLevelEndpointFuncWithFourQueries("https://servicedirectory.googleapis.com/v1/projects/%s/locations/%s/namespaces/%s/services/%s/endpoints/%s"),
		// Reference: https://cloud.google.com/service-directory/docs/reference/rest/v1/projects.locations.namespaces.services.endpoints/list
		// IAM Perm: servicedirectory.endpoints.list
		// GET https://servicedirectory.googleapis.com/v1/projects/*/locations/*/namespaces/*/services/*/endpoints
		// TODO: Update this for => https://linear.app/overmind/issue/ENG-580/handle-terraform-mappings-in-search-method
		// id => projects/*/locations/*/namespaces/*/services/*/endpoints/*
		SearchEndpointFunc:  projectLevelEndpointFuncWithThreeQueries("https://servicedirectory.googleapis.com/v1/projects/%s/locations/%s/namespaces/%s/services/%s/endpoints"),
		SearchDescription:   "Search for endpoints by {location}|{namespace_id}|{service_id} or projects/{project}/locations/{location}/namespaces/{namespace_id}/services/{service_id}/endpoints/{endpoint_id} which is supported for terraform mappings.",
		UniqueAttributeKeys: []string{"locations", "namespaces", "services", "endpoints"},
	},
	ServiceUsageService: {
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_CONFIGURATION,
		Scope:              ScopeProject,
		// Reference: https://cloud.google.com/service-usage/docs/reference/rest/v1/services/get
		// GET https://serviceusage.googleapis.com/v1/{name=*/*/services/*}
		// An example name would be: projects/123/services/service
		// where 123 is the project number TODO: make sure that this is working with project ID as well
		// IAM Perm: serviceusage.services.get
		GetEndpointBaseURLFunc: projectLevelEndpointFuncWithSingleQuery("https://serviceusage.googleapis.com/v1/projects/%s/services/%s"),
		// Reference: https://cloud.google.com/service-usage/docs/reference/rest/v1/services/list
		// GET https://serviceusage.googleapis.com/v1/{parent=*/*}/services
		/*
			List all services available to the specified project, and the current state of those services with respect to the project.
			The list includes all public services, all services for which the calling user has the `servicemanagement.services.bind` permission,
			and all services that have already been enabled on the project.
			The list can be filtered to only include services in a specific state, for example to only include services enabled on the project.
		*/
		// Let's use the filter to only list enabled services.
		// IAM Perm: serviceusage.services.list
		ListEndpointFunc:    projectLevelListFunc("https://serviceusage.googleapis.com/v1/projects/%s/services?filter=state:ENABLED"),
		UniqueAttributeKeys: []string{"services"},
	},
	SQLAdminBackup: {
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_DATABASE,
		Scope:              ScopeProject,
		// Reference: https://cloud.google.com/sql/docs/mysql/admin-api/rest/v1/Backups/GetBackup
		// GET https://sqladmin.googleapis.com/v1/{name=projects/*/backups/*}
		GetEndpointBaseURLFunc: projectLevelEndpointFuncWithSingleQuery("https://sqladmin.googleapis.com/v1/projects/%s/backups/%s"),
		// Reference: https://cloud.google.com/sql/docs/mysql/admin-api/rest/v1/Backups/ListBackups
		// GET https://sqladmin.googleapis.com/v1/{parent=projects/*}/backups
		ListEndpointFunc:    projectLevelListFunc("https://sqladmin.googleapis.com/v1/projects/%s/backups"),
		UniqueAttributeKeys: []string{"backups"},
	},
	SQLAdminBackupRun: {
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_DATABASE,
		Scope:              ScopeProject,
		// Reference: https://cloud.google.com/sql/docs/mysql/admin-api/rest/v1/backupRuns/get
		// GET https://sqladmin.googleapis.com/v1/projects/{project}/instances/{instance}/backupRuns/{id}
		GetEndpointBaseURLFunc: projectLevelEndpointFuncWithTwoQueries("https://sqladmin.googleapis.com/v1/projects/%s/instances/%s/backupRuns/%s"),
		// Reference: https://cloud.google.com/sql/docs/mysql/admin-api/rest/v1/backupRuns/list
		// GET https://sqladmin.googleapis.com/v1/projects/{project}/instances/{instance}/backupRuns
		SearchEndpointFunc:  projectLevelEndpointFuncWithSingleQuery("https://sqladmin.googleapis.com/v1/projects/%s/instances/%s/backupRuns"),
		UniqueAttributeKeys: []string{"instances", "backupRuns"},
	},
	StorageBucket: {
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_STORAGE,
		Scope:              ScopeProject,
		// Reference: https://cloud.google.com/storage/docs/json_api/v1/buckets/get
		// GET https://storage.googleapis.com/storage/v1/b/{bucket}
		GetEndpointBaseURLFunc: func(queryParts ...string) (EndpointFunc, error) {
			if len(queryParts) == 1 && queryParts[0] != "" {
				return func(query string) string {
					if query != "" {
						return fmt.Sprintf("https://storage.googleapis.com/storage/v1/b/%s", query)
					}
					return ""
				}, nil
			}
			return nil, fmt.Errorf("bucket name cannot be empty: %v", queryParts)
		},
		// Reference: https://cloud.google.com/storage/docs/json_api/v1/buckets/list
		// GET https://storage.googleapis.com/storage/v1/b?project={project}
		ListEndpointFunc:    projectLevelListFunc("https://storage.googleapis.com/storage/v1/b?project=%s"),
		UniqueAttributeKeys: []string{"b"},
	},
}
