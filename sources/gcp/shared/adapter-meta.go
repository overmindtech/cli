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
	InDevelopment       bool     // If true, the adapter is in development and should not be used in production.
	IAMPermissions      []string // List of IAM permissions required to access this resource.
}

// We have group of functions that are similar in nature, however they cannot simplified into a generic function because
// of the different number of query parts they accept.
// Also, we want to keep the explicit logic for now for the sake of human readability.

func ProjectLevelEndpointFuncWithSingleQuery(format string) func(queryParts ...string) (EndpointFunc, error) {
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

func ProjectLevelEndpointFuncWithTwoQueries(format string) func(queryParts ...string) (EndpointFunc, error) {
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

func ProjectLevelEndpointFuncWithThreeQueries(format string) func(queryParts ...string) (EndpointFunc, error) {
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

func ProjectLevelEndpointFuncWithFourQueries(format string) func(queryParts ...string) (EndpointFunc, error) {
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

func ZoneLevelEndpointFuncWithSingleQuery(format string) func(queryParts ...string) (EndpointFunc, error) {
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

func RegionalLevelEndpointFuncWithSingleQuery(format string) func(queryParts ...string) (EndpointFunc, error) {
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

func ZoneLevelEndpointFuncWithTwoQueries(format string) func(queryParts ...string) (EndpointFunc, error) {
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

func RegionalLevelEndpointFuncWithTwoQueries(format string) func(queryParts ...string) (EndpointFunc, error) {
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

func ProjectLevelListFunc(format string) func(adapterInitParams ...string) (string, error) {
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

func RegionLevelListFunc(format string) func(adapterInitParams ...string) (string, error) {
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

func ZoneLevelListFunc(format string) func(adapterInitParams ...string) (string, error) {
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
		GetEndpointBaseURLFunc: ProjectLevelEndpointFuncWithSingleQuery("https://aiplatform.googleapis.com/v1/projects/%s/locations/global/customJobs/%s"),
		// Reference: https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.customJobs/list
		// https://aiplatform.googleapis.com/v1/projects/{project}/locations/{location}/customJobs
		// Expected location is `global` for the default endpoint.
		ListEndpointFunc:    ProjectLevelListFunc("https://aiplatform.googleapis.com/v1/projects/%s/locations/global/customJobs"),
		UniqueAttributeKeys: []string{"customJobs"},
		IAMPermissions:      []string{"aiplatform.customJobs.get", "aiplatform.customJobs.list"},
	},
	AIPlatformPipelineJob: {
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_AI,
		Scope:              ScopeProject,
		// When using the default endpoint, the location must be set to `global`.
		//  Format: projects/{project}/locations/{location}/pipelineJobs/{pipelineJob}
		GetEndpointBaseURLFunc: ProjectLevelEndpointFuncWithSingleQuery("https://aiplatform.googleapis.com/v1/projects/%s/locations/global/pipelineJobs/%s"),
		// Reference: https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.pipelineJobs/list
		ListEndpointFunc:    ProjectLevelListFunc("https://aiplatform.googleapis.com/v1/projects/%s/locations/global/pipelineJobs"),
		UniqueAttributeKeys: []string{"pipelineJobs"},
		IAMPermissions:      []string{"aiplatform.pipelineJobs.get", "aiplatform.pipelineJobs.list"},
	},
	ArtifactRegistryDockerImage: {
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_STORAGE,
		Scope:              ScopeProject,
		// Reference: https://cloud.google.com/artifact-registry/docs/reference/rest/v1/projects.locations.repositories.dockerImages/get?rep_location=global
		// GET https://artifactregistry.googleapis.com/v1/{name=projects/*/locations/*/repositories/*/dockerImages/*}
		// IAM permissions: artifactregistry.dockerImages.get
		GetEndpointBaseURLFunc: ProjectLevelEndpointFuncWithThreeQueries("https://artifactregistry.googleapis.com/v1/projects/%s/locations/%s/repositories/%s/dockerImages/%s"),
		// Reference: https://cloud.google.com/artifact-registry/docs/reference/rest/v1/projects.locations.repositories.dockerImages/list?rep_location=global
		// GET https://artifactregistry.googleapis.com/v1/{parent=projects/*/locations/*/repositories/*}/dockerImages
		// IAM permissions: artifactregistry.dockerImages.list
		SearchEndpointFunc:  ProjectLevelEndpointFuncWithTwoQueries("https://artifactregistry.googleapis.com/v1/projects/%s/locations/%s/repositories/%s/dockerImages"),
		SearchDescription:   "Search for Docker images in Artifact Registry. Use the format \"location|repository_id\" or \"projects/project/locations/location/repository/repository_id/dockerImages/docker_image\" which is supported for terraform mappings.",
		UniqueAttributeKeys: []string{"locations", "repositories", "dockerImages"},
		IAMPermissions:      []string{"artifactregistry.dockerimages.get", "artifactregistry.dockerimages.list"},
	},
	ArtifactRegistryRepository: {
		// Reference: https://cloud.google.com/artifact-registry/docs/reference/rest/v1/projects.locations.repositories/get?rep_location=global
		InDevelopment:      true,
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_STORAGE,
		Scope:              ScopeProject,
		// https://artifactregistry.googleapis.com/v1/projects/*/locations/*/repositories/*
		GetEndpointBaseURLFunc: ProjectLevelEndpointFuncWithTwoQueries("https://artifactregistry.googleapis.com/v1/projects/%s/locations/%s/repositories/%s"),
		// https://artifactregistry.googleapis.com/v1/{parent=projects/*/locations/*}/repositories
		SearchEndpointFunc:  ProjectLevelEndpointFuncWithSingleQuery("https://artifactregistry.googleapis.com/v1/projects/%s/locations/%s/repositories"),
		UniqueAttributeKeys: []string{"locations", "repositories"},
		IAMPermissions:      []string{"artifactregistry.repositories.get", "artifactregistry.repositories.list"},
	},
	BigTableAdminAppProfile: {
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_CONFIGURATION,
		Scope:              ScopeProject,
		// Reference: https://cloud.google.com/bigtable/docs/reference/admin/rest/v2/projects.instances.appProfiles/get
		// GET https://bigtableadmin.googleapis.com/v2/{name=projects/*/instances/*/appProfiles/*}
		GetEndpointBaseURLFunc: ProjectLevelEndpointFuncWithTwoQueries("https://bigtableadmin.googleapis.com/v2/projects/%s/instances/%s/appProfiles/%s"),
		// Reference: https://cloud.google.com/bigtable/docs/reference/admin/rest/v2/projects.instances.appProfiles/list
		SearchEndpointFunc:  ProjectLevelEndpointFuncWithSingleQuery("https://bigtableadmin.googleapis.com/v2/projects/%s/instances/%s/appProfiles"),
		SearchDescription:   "Search for BigTable App Profiles in an instance. Use the format \"instance\" or \"projects/project_id/instances/instance_name/appProfiles/app_profile_id\" which is supported for terraform mappings.",
		UniqueAttributeKeys: []string{"instances", "appProfiles"},
		IAMPermissions:      []string{"bigtable.appProfiles.get", "bigtable.appProfiles.list"},
	},
	BigTableAdminInstance: {
		// Reference: https://cloud.google.com/bigtable/docs/reference/admin/rest/v2/projects.instances/get
		InDevelopment:      true,
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_CONFIGURATION,
		Scope:              ScopeProject,
		// https://bigtableadmin.googleapis.com/v2/projects/*/instances/*
		GetEndpointBaseURLFunc: ProjectLevelEndpointFuncWithSingleQuery("https://bigtableadmin.googleapis.com/v2/projects/%s/instances/%s"),
		// https://bigtableadmin.googleapis.com/v2/projects/*/instances
		ListEndpointFunc:    ProjectLevelListFunc("https://bigtableadmin.googleapis.com/v2/projects/%s/instances"),
		UniqueAttributeKeys: []string{"instances"},
		IAMPermissions:      []string{"bigtable.instances.get", "bigtable.instances.list"},
	},
	BigTableAdminBackup: {
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_OTHER,
		Scope:              ScopeProject,
		// Reference: https://cloud.google.com/bigtable/docs/reference/admin/rest/v2/projects.instances.clusters.backups/get
		// GET https://bigtableadmin.googleapis.com/v2/{name=projects/*/instances/*/clusters/*/backups/*}
		GetEndpointBaseURLFunc: ProjectLevelEndpointFuncWithThreeQueries("https://bigtableadmin.googleapis.com/v2/projects/%s/instances/%s/clusters/%s/backups/%s"),
		// GET https://bigtableadmin.googleapis.com/v2/{parent=projects/*/instances/*/clusters/*}/backups
		SearchEndpointFunc:  ProjectLevelEndpointFuncWithTwoQueries("https://bigtableadmin.googleapis.com/v2/projects/%s/instances/%s/clusters/%s/backups"),
		UniqueAttributeKeys: []string{"instances", "clusters", "backups"},
		// HEALTH: https://cloud.google.com/bigtable/docs/reference/admin/rest/v2/projects.instances.clusters.backups#state
		// TODO: https://linear.app/overmind/issue/ENG-631/investigate-how-we-can-add-health-status-for-supporting-items
		IAMPermissions: []string{"bigtable.backups.get", "bigtable.backups.list"},
	},
	BigTableAdminCluster: {
		InDevelopment: true,
		// Reference: https://cloud.google.com/bigtable/docs/reference/admin/rest/v2/projects.instances.clusters/get
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_CONFIGURATION,
		Scope:              ScopeProject,
		// https://bigtableadmin.googleapis.com/v2/projects/*/instances/*/clusters/*
		GetEndpointBaseURLFunc: ProjectLevelEndpointFuncWithTwoQueries("https://bigtableadmin.googleapis.com/v2/projects/%s/instances/%s/clusters/%s"),
		// https://bigtableadmin.googleapis.com/v2/projects/*/instances/*/clusters
		SearchEndpointFunc:  ProjectLevelEndpointFuncWithSingleQuery("https://bigtableadmin.googleapis.com/v2/projects/%s/instances/%s/clusters"),
		UniqueAttributeKeys: []string{"instances", "clusters"},
		IAMPermissions:      []string{"bigtable.clusters.get", "bigtable.clusters.list"},
	},
	BigTableAdminTable: {
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_DATABASE,
		Scope:              ScopeProject,
		// Reference: https://cloud.google.com/bigtable/docs/reference/admin/rest/v2/projects.instances.tables/get
		// GET https://bigtableadmin.googleapis.com/v2/{name=projects/*/instances/*/tables/*}
		// IAM permissions: bigtable.tables.get
		GetEndpointBaseURLFunc: ProjectLevelEndpointFuncWithTwoQueries("https://bigtableadmin.googleapis.com/v2/projects/%s/instances/%s/tables/%s"),
		// Reference: https://cloud.google.com/bigtable/docs/reference/admin/rest/v2/projects.instances.tables/list
		// GET https://bigtableadmin.googleapis.com/v2/{parent=projects/*/instances/*}/tables
		SearchEndpointFunc:  ProjectLevelEndpointFuncWithSingleQuery("https://bigtableadmin.googleapis.com/v2/projects/%s/instances/%s/tables"),
		SearchDescription:   "Search for BigTable tables in an instance. Use the format \"instance_name\" or \"projects/project_id/instances/instance_name/tables/table_name\" which is supported for terraform mappings.",
		UniqueAttributeKeys: []string{"instances", "tables"},
		IAMPermissions:      []string{"bigtable.tables.get", "bigtable.tables.list"},
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
		IAMPermissions:      []string{"resourcemanager.projects.get"},
	},
	CloudBuildBuild: {
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_CONFIGURATION,
		Scope:              ScopeProject,
		// Reference: https://cloud.google.com/build/docs/api/reference/rest/v1/projects.builds/get
		// GET https://cloudbuild.googleapis.com/v1/projects/{projectId}/builds/{id}
		// IAM permissions: cloudbuild.builds.get
		GetEndpointBaseURLFunc: ProjectLevelEndpointFuncWithSingleQuery("https://cloudbuild.googleapis.com/v1/projects/%s/builds/%s"),
		// Reference: https://cloud.google.com/build/docs/api/reference/rest/v1/projects.builds/list
		// GET https://cloudbuild.googleapis.com/v1/projects/{projectId}/builds
		ListEndpointFunc:    ProjectLevelListFunc("https://cloudbuild.googleapis.com/v1/projects/%s/builds"),
		UniqueAttributeKeys: []string{"builds"},
		// HEALTH: https://cloud.google.com/build/docs/api/reference/rest/v1/projects.builds#Build.Status
		// TODO: https://linear.app/overmind/issue/ENG-631/investigate-how-we-can-add-health-status-for-supporting-items
		IAMPermissions: []string{"cloudbuild.builds.get", "cloudbuild.builds.list"},
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
		// HEALTH: https://cloud.google.com/resource-manager/reference/rest/v3/projects#State
		// TODO: https://linear.app/overmind/issue/ENG-631/investigate-how-we-can-add-health-status-for-supporting-items
		IAMPermissions: []string{"resourcemanager.projects.get"},
	},
	ComputeAcceleratorType: {
		// Reference: https://cloud.google.com/compute/docs/reference/rest/v1/acceleratorTypes/get
		InDevelopment:      true,
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_COMPUTE_APPLICATION,
		Scope:              ScopeZonal,
		// https://compute.googleapis.com/compute/v1/projects/{project}/zones/{zone}/acceleratorTypes/{acceleratorType}
		GetEndpointBaseURLFunc: ZoneLevelEndpointFuncWithSingleQuery("https://compute.googleapis.com/compute/v1/projects/%s/zones/%s/acceleratorTypes/%s"),
		// https://compute.googleapis.com/compute/v1/projects/{project}/zones/{zone}/acceleratorTypes
		ListEndpointFunc:    ZoneLevelListFunc("https://compute.googleapis.com/compute/v1/projects/%s/zones/%s/acceleratorTypes"),
		UniqueAttributeKeys: []string{"acceleratorTypes"},
		IAMPermissions:      []string{"compute.acceleratorTypes.get", "compute.acceleratorTypes.list"},
	},
	ComputeFirewall: {
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
		Scope:              ScopeProject,
		// https://compute.googleapis.com/compute/v1/projects/{project}/global/firewalls/{firewall}
		GetEndpointBaseURLFunc: ProjectLevelEndpointFuncWithSingleQuery("https://compute.googleapis.com/compute/v1/projects/%s/global/firewalls/%s"),
		// Reference: https://cloud.google.com/compute/docs/reference/rest/v1/firewalls/list
		// https://compute.googleapis.com/compute/v1/projects/{project}/global/firewalls
		ListEndpointFunc:    ProjectLevelListFunc("https://compute.googleapis.com/compute/v1/projects/%s/global/firewalls"),
		UniqueAttributeKeys: []string{"firewalls"},
		IAMPermissions:      []string{"compute.firewalls.get", "compute.firewalls.list"},
	},
	ComputeInstance: {
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_COMPUTE_APPLICATION,
		Scope:              ScopeZonal,
		// https://compute.googleapis.com/compute/v1/projects/{project}/zones/{zone}/instances/{instance}
		GetEndpointBaseURLFunc: ZoneLevelEndpointFuncWithSingleQuery("https://compute.googleapis.com/compute/v1/projects/%s/zones/%s/instances/%s"),
		// https://compute.googleapis.com/compute/v1/projects/{project}/zones/{zone}/instances
		ListEndpointFunc:    ZoneLevelListFunc("https://compute.googleapis.com/compute/v1/projects/%s/zones/%s/instances"),
		UniqueAttributeKeys: []string{"instances"},
		IAMPermissions:      []string{"compute.instances.get", "compute.instances.list"},
	},
	ComputeInstanceTemplate: {
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_COMPUTE_APPLICATION,
		Scope:              ScopeProject,
		// https://compute.googleapis.com/compute/v1/projects/{project}/global/instanceTemplates/{instanceTemplate}
		GetEndpointBaseURLFunc: ProjectLevelEndpointFuncWithSingleQuery("https://compute.googleapis.com/compute/v1/projects/%s/global/instanceTemplates/%s"),
		// https://compute.googleapis.com/compute/v1/projects/{project}/global/instanceTemplates
		ListEndpointFunc:    ProjectLevelListFunc("https://compute.googleapis.com/compute/v1/projects/%s/global/instanceTemplates"),
		UniqueAttributeKeys: []string{"instanceTemplates"},
		IAMPermissions:      []string{"compute.instanceTemplates.get", "compute.instanceTemplates.list"},
	},
	ComputeLicense: {
		InDevelopment: true,
		// Reference: https://cloud.google.com/compute/docs/reference/rest/v1/licenses/get
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_CONFIGURATION,
		Scope:              ScopeProject,
		// https://compute.googleapis.com/compute/v1/projects/{project}/global/licenses/{license}
		GetEndpointBaseURLFunc: ProjectLevelEndpointFuncWithSingleQuery("https://compute.googleapis.com/compute/v1/projects/%s/global/licenses/%s"),
		// https://compute.googleapis.com/compute/v1/projects/{project}/global/licenses
		ListEndpointFunc:    ProjectLevelListFunc("https://compute.googleapis.com/compute/v1/projects/%s/global/licenses"),
		UniqueAttributeKeys: []string{"licenses"},
		// compute.licenses.list is only supported at TESTING stage.
		// Which means it can behave unexpectedly, and not recommended for production use.
		// https://cloud.google.com/iam/docs/custom-roles-permissions-support
		// TODO: Decide whether to support this officially or not.
		IAMPermissions: []string{"compute.licenses.get", "compute.licenses.list"},
	},
	ComputeNetwork: {
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
		Scope:              ScopeProject,
		// https://compute.googleapis.com/compute/v1/projects/{project}/global/networks/{network}
		GetEndpointBaseURLFunc: ProjectLevelEndpointFuncWithSingleQuery("https://compute.googleapis.com/compute/v1/projects/%s/global/networks/%s"),
		// https://compute.googleapis.com/compute/v1/projects/{project}/global/networks
		ListEndpointFunc:    ProjectLevelListFunc("https://compute.googleapis.com/compute/v1/projects/%s/global/networks"),
		UniqueAttributeKeys: []string{"networks"},
		IAMPermissions:      []string{"compute.networks.get", "compute.networks.list"},
	},
	ComputeDiskType: {
		// Reference: https://cloud.google.com/compute/docs/reference/rest/v1/diskTypes/get
		InDevelopment:      true,
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_STORAGE,
		Scope:              ScopeZonal,
		// https://compute.googleapis.com/compute/v1/projects/{project}/zones/{zone}/diskTypes/{diskType}
		GetEndpointBaseURLFunc: ZoneLevelEndpointFuncWithSingleQuery("https://compute.googleapis.com/compute/v1/projects/%s/zones/%s/diskTypes/%s"),
		// https://compute.googleapis.com/compute/v1/projects/{project}/zones/{zone}/diskTypes
		ListEndpointFunc:    ZoneLevelListFunc("https://compute.googleapis.com/compute/v1/projects/%s/zones/%s/diskTypes"),
		UniqueAttributeKeys: []string{"diskTypes"},
		IAMPermissions:      []string{"compute.diskTypes.get", "compute.diskTypes.list"},
	},
	ComputeProject: {
		// Reference: https://cloud.google.com/compute/docs/reference/rest/v1/projects/get
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_CONFIGURATION,
		Scope:              ScopeProject,
		// https://compute.googleapis.com/compute/v1/projects/{project}
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
						return fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s?fields=name", query)
					}
					return ""
				}, nil
			}
			return nil, fmt.Errorf("projectID cannot be empty: %v", adapterInitParams)
		},
		UniqueAttributeKeys: []string{"projects"},
		IAMPermissions:      []string{"compute.projects.get"},
	},
	ComputeResourcePolicy: {
		// Reference: https://cloud.google.com/compute/docs/reference/rest/v1/resourcePolicies/get
		InDevelopment:      true,
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_CONFIGURATION,
		Scope:              ScopeRegional,
		// https://compute.googleapis.com/compute/v1/projects/{project}/regions/{region}/resourcePolicies/{resourcePolicy}
		GetEndpointBaseURLFunc: RegionalLevelEndpointFuncWithSingleQuery("https://compute.googleapis.com/compute/v1/projects/%s/regions/%s/resourcePolicies/%s"),
		// https://cloud.google.com/compute/docs/reference/rest/v1/resourcePolicies/list
		ListEndpointFunc:    RegionLevelListFunc("https://compute.googleapis.com/compute/v1/projects/%s/regions/%s/resourcePolicies"),
		UniqueAttributeKeys: []string{"resourcePolicies"},
		IAMPermissions:      []string{"compute.resourcePolicies.get", "compute.resourcePolicies.list"},
	},
	ComputeRoute: {
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
		Scope:              ScopeProject,
		// https://compute.googleapis.com/compute/v1/projects/{project}/global/routes/{route}
		GetEndpointBaseURLFunc: ProjectLevelEndpointFuncWithSingleQuery("https://compute.googleapis.com/compute/v1/projects/%s/global/routes/%s"),
		// https://compute.googleapis.com/compute/v1/projects/{project}/global/routes
		ListEndpointFunc:    ProjectLevelListFunc("https://compute.googleapis.com/compute/v1/projects/%s/global/routes"),
		UniqueAttributeKeys: []string{"routes"},
		IAMPermissions:      []string{"compute.routes.get", "compute.routes.list"},
	},
	ComputeSubnetwork: {
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
		Scope:              ScopeRegional,
		// https://compute.googleapis.com/compute/v1/projects/{project}/regions/{region}/subnetworks/{subnetwork}
		GetEndpointBaseURLFunc: RegionalLevelEndpointFuncWithSingleQuery("https://compute.googleapis.com/compute/v1/projects/%s/regions/%s/subnetworks/%s"),
		// https://compute.googleapis.com/compute/v1/projects/{project}/regions/{region}/subnetworks
		ListEndpointFunc:    RegionLevelListFunc("https://compute.googleapis.com/compute/v1/projects/%s/regions/%s/subnetworks"),
		UniqueAttributeKeys: []string{"subnetworks"},
		IAMPermissions:      []string{"compute.subnetworks.get", "compute.subnetworks.list"},
	},
	ComputeStoragePool: {
		// Reference: https://cloud.google.com/compute/docs/reference/rest/v1/storagePools/get
		InDevelopment:      true,
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_STORAGE,
		Scope:              ScopeZonal,
		// https://compute.googleapis.com/compute/v1/projects/{project}/zones/{zone}/storagePools/{storagePool}
		GetEndpointBaseURLFunc: ZoneLevelEndpointFuncWithSingleQuery("https://compute.googleapis.com/compute/v1/projects/%s/zones/%s/storagePools/%s"),
		// https://compute.googleapis.com/compute/v1/projects/{project}/zones/{zone}/storagePools
		ListEndpointFunc:    ZoneLevelListFunc("https://compute.googleapis.com/compute/v1/projects/%s/zones/%s/storagePools"),
		UniqueAttributeKeys: []string{"storagePools"},
		IAMPermissions:      []string{"compute.storagePools.get", "compute.storagePools.list"},
	},
	DataformRepository: {
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_DATABASE,
		Scope:              ScopeProject,
		// Reference: https://cloud.google.com/dataform/reference/rest/v1/projects.locations.repositories/get
		// GET https://dataform.googleapis.com/v1/projects/*/locations/*/repositories/*
		// IAM permissions: dataform.repositories.get
		GetEndpointBaseURLFunc: ProjectLevelEndpointFuncWithTwoQueries("https://dataform.googleapis.com/v1/projects/%s/locations/%s/repositories/%s"),
		// Reference: https://cloud.google.com/dataform/reference/rest/v1/projects.locations.repositories/list
		// GET https://dataform.googleapis.com/v1/projects/*/locations/*/repositories
		SearchEndpointFunc:  ProjectLevelEndpointFuncWithSingleQuery("https://dataform.googleapis.com/v1/projects/%s/locations/%s/repositories"),
		SearchDescription:   "Search for Dataform repositories in a location. Use the format \"location\" or \"projects/project_id/locations/location/repositories/repository_name\" which is supported for terraform mappings.",
		UniqueAttributeKeys: []string{"locations", "repositories"},
		IAMPermissions:      []string{"dataform.repositories.get", "dataform.repositories.list"},
	},
	DataplexEntryGroup: {
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_STORAGE,
		Scope:              ScopeProject,
		// Reference: https://cloud.google.com/dataplex/docs/reference/rest/v1/projects.locations.entryGroups/get
		// GET https://dataplex.googleapis.com/v1/{name=projects/*/locations/*/entryGroups/*}
		// IAM permissions: dataplex.entryGroups.get
		GetEndpointBaseURLFunc: ProjectLevelEndpointFuncWithTwoQueries("https://dataplex.googleapis.com/v1/projects/%s/locations/%s/entryGroups/%s"),
		// Reference: https://cloud.google.com/dataplex/docs/reference/rest/v1/projects.locations.entryGroups/list
		// GET https://dataplex.googleapis.com/v1/{parent=projects/*/locations/*}/entryGroups
		SearchEndpointFunc:  ProjectLevelEndpointFuncWithSingleQuery("https://dataplex.googleapis.com/v1/projects/%s/locations/%s/entryGroups"),
		SearchDescription:   "Search for Dataplex entry groups in a location. Use the format \"location\" or \"projects/project_id/locations/location/entryGroups/entry_group_id\" which is supported for terraform mappings.",
		UniqueAttributeKeys: []string{"locations", "entryGroups"},
		// HEALTH: https://cloud.google.com/dataplex/docs/reference/rest/v1/TransferStatus
		// TODO: https://linear.app/overmind/issue/ENG-631/investigate-how-we-can-add-health-status-for-supporting-items
		IAMPermissions: []string{"dataplex.entryGroups.get", "dataplex.entryGroups.list"},
	},
	DNSManagedZone: {
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
		Scope:              ScopeProject,
		// Reference: https://cloud.google.com/dns/docs/reference/rest/v1/managedZones/get
		// GET https://dns.googleapis.com/dns/v1/projects/{project}/managedZones/{managedZone}
		GetEndpointBaseURLFunc: ProjectLevelEndpointFuncWithSingleQuery("https://dns.googleapis.com/dns/v1/projects/%s/managedZones/%s"),
		// Reference: https://cloud.google.com/dns/docs/reference/rest/v1/managedZones/list
		// GET https://dns.googleapis.com/dns/v1/projects/{project}/managedZones
		ListEndpointFunc:    ProjectLevelListFunc("https://dns.googleapis.com/dns/v1/projects/%s/managedZones"),
		UniqueAttributeKeys: []string{"managedZones"},
		IAMPermissions:      []string{"dns.managedZones.get", "dns.managedZones.list"},
	},
	EssentialContactsContact: {
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_OTHER,
		Scope:              ScopeProject,
		// Reference: https://cloud.google.com/resource-manager/docs/reference/essentialcontacts/rest/v1/projects.contacts/get
		// GET https://essentialcontacts.googleapis.com/v1/projects/*/contacts/*
		// IAM permissions: essentialcontacts.contacts.get
		GetEndpointBaseURLFunc: ProjectLevelEndpointFuncWithSingleQuery("https://essentialcontacts.googleapis.com/v1/projects/%s/contacts/%s"),
		// Reference: https://cloud.google.com/resource-manager/docs/reference/essentialcontacts/rest/v1/projects.contacts/list
		// GET https://essentialcontacts.googleapis.com/v1/projects/*/contacts
		// IAM permissions: essentialcontacts.contacts.list
		ListEndpointFunc: ProjectLevelListFunc("https://essentialcontacts.googleapis.com/v1/projects/%s/contacts"),
		// This is a special case where we have to define the SEARCH method for only to support Terraform Mapping.
		// We only validate the adapter initiation constraint: whether the project ID is provided or not.
		// We return a nil EndpointFunc without any error, because in the runtime we will use the
		// GET endpoint for retrieving the item for Terraform Query.
		SearchEndpointFunc: func(adapterInitParams ...string) (EndpointFunc, error) {
			if len(adapterInitParams) != 1 || adapterInitParams[0] == "" {
				return nil, fmt.Errorf("projectID cannot be empty: %v", adapterInitParams)
			}

			return nil, nil
		},
		SearchDescription:   "Search for contacts by their ID in the form of \"projects/project_id/contacts/contact_id\".",
		UniqueAttributeKeys: []string{"contacts"},
		// HEALTH: https://cloud.google.com/resource-manager/docs/reference/essentialcontacts/rest/v1/folders.contacts#validationstate
		// TODO: https://linear.app/overmind/issue/ENG-631/investigate-how-we-can-add-health-status-for-supporting-items
		IAMPermissions: []string{"essentialcontacts.contacts.get", "essentialcontacts.contacts.list"},
	},
	IAMRole: {
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_SECURITY,
		Scope:              ScopeProject,
		// Reference: https://cloud.google.com/iam/docs/reference/rest/v1/roles/get
		// https://iam.googleapis.com/v1/projects/{PROJECT_ID}/roles/{CUSTOM_ROLE_ID}
		GetEndpointBaseURLFunc: ProjectLevelEndpointFuncWithSingleQuery("https://iam.googleapis.com/v1/projects/%s/roles/%s"),
		// Reference: https://cloud.google.com/iam/docs/reference/rest/v1/roles/list
		// https://iam.googleapis.com/v1/projects/{PROJECT_ID}/roles
		ListEndpointFunc:    ProjectLevelListFunc("https://iam.googleapis.com/v1/projects/%s/roles"),
		UniqueAttributeKeys: []string{"roles"},
		IAMPermissions:      []string{"iam.roles.get", "iam.roles.list"},
	},
	LoggingBucket: {
		// global is a type of location.
		// location is generally a region.
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_OBSERVABILITY,
		Scope:              ScopeProject,
		// Reference: https://cloud.google.com/logging/docs/reference/v2/rest/v2/projects.locations.buckets/get
		// GET https://logging.googleapis.com/v2/projects/*/locations/*/buckets/*
		// IAM permissions: logging.buckets.get
		GetEndpointBaseURLFunc: ProjectLevelEndpointFuncWithTwoQueries("https://logging.googleapis.com/v2/projects/%s/locations/%s/buckets/%s"),
		// Reference: https://cloud.google.com/logging/docs/reference/v2/rest/v2/projects.locations.buckets/list
		// GET https://logging.googleapis.com/v2/projects/*/locations/*/buckets
		// IAM permissions: logging.buckets.list
		SearchEndpointFunc:  ProjectLevelEndpointFuncWithSingleQuery("https://logging.googleapis.com/v2/projects/%s/locations/%s/buckets"),
		UniqueAttributeKeys: []string{"locations", "buckets"},
		// HEALTH: Supports Health status: https://cloud.google.com/logging/docs/reference/v2/rest/v2/LifecycleState
		// TODO: https://linear.app/overmind/issue/ENG-631/investigate-how-we-can-add-health-status-for-supporting-items
		IAMPermissions: []string{"logging.buckets.get", "logging.buckets.list"},
	},
	LoggingLink: {
		// HEALTH: https://cloud.google.com/logging/docs/reference/v2/rest/v2/LifecycleState
		// TODO: https://linear.app/overmind/issue/ENG-631/investigate-how-we-can-add-health-status-for-supporting-items
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_OBSERVABILITY,
		Scope:              ScopeProject,
		// Reference: https://cloud.google.com/logging/docs/reference/v2/rest/v2/projects.locations.buckets.links/get
		// GET https://logging.googleapis.com/v2/projects/*/locations/*/buckets/*/links/*
		// IAM permissions: logging.links.get
		GetEndpointBaseURLFunc: ProjectLevelEndpointFuncWithThreeQueries("https://logging.googleapis.com/v2/projects/%s/locations/%s/buckets/%s/links/%s"),
		// Reference: https://cloud.google.com/logging/docs/reference/v2/rest/v2/projects.locations.buckets.links/list
		// GET https://logging.googleapis.com/v2/projects/*/locations/*/buckets/*/links
		// IAM permissions: logging.links.list
		SearchEndpointFunc:  ProjectLevelEndpointFuncWithTwoQueries("https://logging.googleapis.com/v2/projects/%s/locations/%s/buckets/%s/links"),
		UniqueAttributeKeys: []string{"locations", "buckets", "links"},
		IAMPermissions:      []string{"logging.links.get", "logging.links.list"},
	},
	LoggingSavedQuery: {
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_OBSERVABILITY,
		Scope:              ScopeProject,
		// Reference: https://cloud.google.com/logging/docs/reference/v2/rest/v2/projects.locations.savedQueries/get
		// GET https://logging.googleapis.com/v2/projects/*/locations/*/savedQueries/*
		// IAM permissions: logging.savedQueries.get
		GetEndpointBaseURLFunc: ProjectLevelEndpointFuncWithTwoQueries("https://logging.googleapis.com/v2/projects/%s/locations/%s/savedQueries/%s"),
		// Reference: https://cloud.google.com/logging/docs/reference/v2/rest/v2/projects.locations.savedQueries/list
		// GET https://logging.googleapis.com/v2/projects/*/locations/*/savedQueries
		// IAM permissions: logging.savedQueries.list
		// Saved Query has to be shared with the project (opposite is a private one) to show up here.
		SearchEndpointFunc:  ProjectLevelEndpointFuncWithSingleQuery("https://logging.googleapis.com/v2/projects/%s/locations/%s/savedQueries"),
		UniqueAttributeKeys: []string{"locations", "savedQueries"},
		IAMPermissions:      []string{"logging.queries.get", "logging.queries.list"},
	},
	MonitoringCustomDashboard: {
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_OBSERVABILITY,
		Scope:              ScopeProject,
		// Reference: https://cloud.google.com/monitoring/api/ref_v3/rest/v1/projects.dashboards/get
		// GET https://monitoring.googleapis.com/v1/projects/[PROJECT_ID_OR_NUMBER]/dashboards/[DASHBOARD_ID] (for custom dashboards).
		// IAM Perm: monitoring.dashboards.get
		GetEndpointBaseURLFunc: ProjectLevelEndpointFuncWithSingleQuery("https://monitoring.googleapis.com/v1/projects/%s/dashboards/%s"),
		// Reference: https://cloud.google.com/monitoring/api/ref_v3/rest/v1/projects.dashboards/list
		// GET https://monitoring.googleapis.com/v1/{parent}/dashboards
		// IAM Perm: monitoring.dashboards.list
		ListEndpointFunc:  ProjectLevelListFunc("https://monitoring.googleapis.com/v1/projects/%s/dashboards"),
		SearchDescription: "Search for custom dashboards by their ID in the form of \"projects/project_id/dashboards/dashboard_id\". This is supported for terraform mappings.",
		// This is a special case where we have to define the SEARCH method for only to support Terraform Mapping.
		// We only validate the adapter initiation constraint: whether the project ID is provided or not.
		// We return a nil EndpointFunc without any error, because in the runtime we will use the
		// GET endpoint for retrieving the item for Terraform Query.
		SearchEndpointFunc: func(adapterInitParams ...string) (EndpointFunc, error) {
			if len(adapterInitParams) != 1 || adapterInitParams[0] == "" {
				return nil, fmt.Errorf("projectID cannot be empty: %v", adapterInitParams)
			}

			return nil, nil
		},
		UniqueAttributeKeys: []string{"dashboards"},
		IAMPermissions:      []string{"monitoring.dashboards.get", "monitoring.dashboards.list"},
	},
	PubSubSubscription: {
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_CONFIGURATION,
		Scope:              ScopeProject,
		// https://pubsub.googleapis.com/v1/projects/{project}/subscriptions/{subscription}
		GetEndpointBaseURLFunc: ProjectLevelEndpointFuncWithSingleQuery("https://pubsub.googleapis.com/v1/projects/%s/subscriptions/%s"),
		// Reference: https://cloud.google.com/pubsub/docs/reference/rest/v1/projects.subscriptions/list?rep_location=global
		// https://pubsub.googleapis.com/v1/projects/{project}/subscriptions
		ListEndpointFunc:    ProjectLevelListFunc("https://pubsub.googleapis.com/v1/projects/%s/subscriptions"),
		UniqueAttributeKeys: []string{"subscriptions"},
		// HEALTH: https://cloud.google.com/pubsub/docs/reference/rest/v1/projects.subscriptions#state_2
		// TODO: https://linear.app/overmind/issue/ENG-631/investigate-how-we-can-add-health-status-for-supporting-items
		IAMPermissions: []string{"pubsub.subscriptions.get", "pubsub.subscriptions.list"},
	},
	PubSubTopic: {
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_CONFIGURATION,
		Scope:              ScopeProject,
		// https://pubsub.googleapis.com/v1/projects/{project}/topics/{topic}
		GetEndpointBaseURLFunc: ProjectLevelEndpointFuncWithSingleQuery("https://pubsub.googleapis.com/v1/projects/%s/topics/%s"),
		// https://pubsub.googleapis.com/v1/projects/{project}/topics
		ListEndpointFunc:    ProjectLevelListFunc("https://pubsub.googleapis.com/v1/projects/%s/topics"),
		UniqueAttributeKeys: []string{"topics"},
		// HEALTH: https://cloud.google.com/pubsub/docs/reference/rest/v1/projects.topics#state
		// TODO: https://linear.app/overmind/issue/ENG-631/investigate-how-we-can-add-health-status-for-supporting-items
		IAMPermissions: []string{"pubsub.topics.get", "pubsub.topics.list"},
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
		GetEndpointBaseURLFunc: ProjectLevelEndpointFuncWithThreeQueries("https://run.googleapis.com/v2/projects/%s/locations/%s/services/%s/revisions/%s"),
		// Reference: https://cloud.google.com/run/docs/reference/rest/v2/projects.locations.services.revisions/list
		// GET https://run.googleapis.com/v2/projects/{project}/locations/{location}/services/{service}/revisions
		// IAM Perm: run.revisions.list
		SearchEndpointFunc:  ProjectLevelEndpointFuncWithTwoQueries("https://run.googleapis.com/v2/projects/%s/locations/%s/services/%s/revisions"),
		UniqueAttributeKeys: []string{"locations", "services", "revisions"},
		IAMPermissions:      []string{"run.revisions.get", "run.revisions.list"},
	},
	ServiceDirectoryEndpoint: {
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_CONFIGURATION,
		Scope:              ScopeProject,
		// Reference: https://cloud.google.com/service-directory/docs/reference/rest/v1/projects.locations.namespaces.services.endpoints/get
		// GET https://servicedirectory.googleapis.com/v1/projects/*/locations/*/namespaces/*/services/*/endpoints/*
		GetEndpointBaseURLFunc: ProjectLevelEndpointFuncWithFourQueries("https://servicedirectory.googleapis.com/v1/projects/%s/locations/%s/namespaces/%s/services/%s/endpoints/%s"),
		// Reference: https://cloud.google.com/service-directory/docs/reference/rest/v1/projects.locations.namespaces.services.endpoints/list
		// IAM Perm: servicedirectory.endpoints.list
		// GET https://servicedirectory.googleapis.com/v1/projects/*/locations/*/namespaces/*/services/*/endpoints
		SearchEndpointFunc:  ProjectLevelEndpointFuncWithThreeQueries("https://servicedirectory.googleapis.com/v1/projects/%s/locations/%s/namespaces/%s/services/%s/endpoints"),
		SearchDescription:   "Search for endpoints by \"location|namespace_id|service_id\" or \"projects/project_id/locations/location/namespaces/namespace_id/services/service_id/endpoints/endpoint_id\" which is supported for terraform mappings.",
		UniqueAttributeKeys: []string{"locations", "namespaces", "services", "endpoints"},
		IAMPermissions:      []string{"servicedirectory.endpoints.get", "servicedirectory.endpoints.list"},
	},
	ServiceDirectoryService: {
		InDevelopment: true,
		// Reference: https://cloud.google.com/service-directory/docs/reference/rest/v1/projects.locations.namespaces.services/get
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_CONFIGURATION,
		Scope:              ScopeProject,
		// https://servicedirectory.googleapis.com/v1/projects/*/locations/*/namespaces/*/services/*
		GetEndpointBaseURLFunc: ProjectLevelEndpointFuncWithThreeQueries("https://servicedirectory.googleapis.com/v1/projects/%s/locations/%s/namespaces/%s/services/%s"),
		// https://servicedirectory.googleapis.com/v1/projects/*/locations/*/namespaces/*/services
		// IAM Perm: servicedirectory.services.list
		SearchEndpointFunc:  ProjectLevelEndpointFuncWithTwoQueries("https://servicedirectory.googleapis.com/v1/projects/%s/locations/%s/namespaces/%s/services"),
		UniqueAttributeKeys: []string{"locations", "namespaces", "services"},
		IAMPermissions:      []string{"servicedirectory.services.get", "servicedirectory.services.list"},
	},
	ServiceUsageService: {
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_CONFIGURATION,
		Scope:              ScopeProject,
		// Reference: https://cloud.google.com/service-usage/docs/reference/rest/v1/services/get
		// GET https://serviceusage.googleapis.com/v1/{name=*/*/services/*}
		// An example name would be: projects/123/services/service
		// where 123 is the project number TODO: make sure that this is working with project ID as well
		// IAM Perm: serviceusage.services.get
		GetEndpointBaseURLFunc: ProjectLevelEndpointFuncWithSingleQuery("https://serviceusage.googleapis.com/v1/projects/%s/services/%s"),
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
		ListEndpointFunc:    ProjectLevelListFunc("https://serviceusage.googleapis.com/v1/projects/%s/services?filter=state:ENABLED"),
		UniqueAttributeKeys: []string{"services"},
		// HEALTH: https://cloud.google.com/service-usage/docs/reference/rest/v1/services#state
		// TODO: https://linear.app/overmind/issue/ENG-631/investigate-how-we-can-add-health-status-for-supporting-items
		IAMPermissions: []string{"serviceusage.services.get", "serviceusage.services.list"},
	},
	SpannerBackup: {
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_DATABASE,
		InDevelopment:      true,
		Scope:              ScopeProject,
		// Reference:https://cloud.google.com/spanner/docs/reference/rest/v1/projects.instances.backups/get?rep_location=global
		// https://spanner.googleapis.com/v1/projects/*/instances/*/backups/*
		GetEndpointBaseURLFunc: ProjectLevelEndpointFuncWithTwoQueries("https://spanner.googleapis.com/v1/projects/%s/instances/%s/backups/%s"),
		// https://spanner.googleapis.com/v1/projects/*/instances/*/backups
		SearchEndpointFunc:  ProjectLevelEndpointFuncWithSingleQuery("https://spanner.googleapis.com/v1/projects/%s/instances/%s/backups"),
		UniqueAttributeKeys: []string{"instances", "backups"},
		IAMPermissions:      []string{"spanner.backups.get", "spanner.backups.list"},
	},
	SpannerDatabase: {
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_DATABASE,
		Scope:              ScopeProject,
		// Reference: https://cloud.google.com/spanner/docs/reference/rest/v1/projects.instances.databases/get?rep_location=global
		// https://spanner.googleapis.com/v1/projects/*/instances/*/databases/*
		GetEndpointBaseURLFunc: ProjectLevelEndpointFuncWithTwoQueries("https://spanner.googleapis.com/v1/projects/%s/instances/%s/databases/%s"),
		// Reference: https://cloud.google.com/spanner/docs/reference/rest/v1/projects.instances.databases/list?rep_location=global
		// https://spanner.googleapis.com/v1/{parent=projects/*/instances/*}/databases
		SearchEndpointFunc:  ProjectLevelEndpointFuncWithSingleQuery("https://spanner.googleapis.com/v1/projects/%s/instances/%s/databases"),
		UniqueAttributeKeys: []string{"instances", "databases"},
		// HEALTH: https://cloud.google.com/spanner/docs/reference/rest/v1/projects.instances.databases#state
		// TODO: https://linear.app/overmind/issue/ENG-631/investigate-how-we-can-add-health-status-for-supporting-items
		IAMPermissions: []string{"spanner.databases.get", "spanner.databases.list"},
	},
	SpannerInstanceConfig: {
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_CONFIGURATION,
		InDevelopment:      true,
		Scope:              ScopeProject,
		// Reference: https://cloud.google.com/spanner/docs/reference/rest/v1/projects.instanceConfigs/get?rep_location=global
		// https://spanner.googleapis.com/v1/projects/*/instanceConfigs/*
		GetEndpointBaseURLFunc: ProjectLevelEndpointFuncWithSingleQuery("https://spanner.googleapis.com/v1/projects/%s/instanceConfigs/%s"),
		// https://// https://spanner.googleapis.com/v1/projects/*/instanceConfigs
		ListEndpointFunc:    ProjectLevelListFunc("https://spanner.googleapis.com/v1/projects/%s/instanceConfigs"),
		UniqueAttributeKeys: []string{"instanceConfigs"},
		IAMPermissions:      []string{"spanner.instanceConfigs.get", "spanner.instanceConfigs.list"},
	},
	SQLAdminBackup: {
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_DATABASE,
		Scope:              ScopeProject,
		// Reference: https://cloud.google.com/sql/docs/mysql/admin-api/rest/v1/Backups/GetBackup
		// GET https://sqladmin.googleapis.com/v1/{name=projects/*/backups/*}
		GetEndpointBaseURLFunc: ProjectLevelEndpointFuncWithSingleQuery("https://sqladmin.googleapis.com/v1/projects/%s/backups/%s"),
		// Reference: https://cloud.google.com/sql/docs/mysql/admin-api/rest/v1/Backups/ListBackups
		// GET https://sqladmin.googleapis.com/v1/{parent=projects/*}/backups
		ListEndpointFunc:    ProjectLevelListFunc("https://sqladmin.googleapis.com/v1/projects/%s/backups"),
		UniqueAttributeKeys: []string{"backups"},
		// HEALTH: https://cloud.google.com/sql/docs/mysql/admin-api/rest/v1/Backups#sqlbackupstate
		// TODO: https://linear.app/overmind/issue/ENG-631/investigate-how-we-can-add-health-status-for-supporting-items
		// https://cloud.google.com/sql/docs/mysql/iam-permissions#permissions-gcloud
		IAMPermissions: []string{"cloudsql.backupRuns.get", "cloudsql.backupRuns.list"},
	},
	SQLAdminBackupRun: {
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_DATABASE,
		Scope:              ScopeProject,
		// Reference: https://cloud.google.com/sql/docs/mysql/admin-api/rest/v1/backupRuns/get
		// GET https://sqladmin.googleapis.com/v1/projects/{project}/instances/{instance}/backupRuns/{id}
		GetEndpointBaseURLFunc: ProjectLevelEndpointFuncWithTwoQueries("https://sqladmin.googleapis.com/v1/projects/%s/instances/%s/backupRuns/%s"),
		// Reference: https://cloud.google.com/sql/docs/mysql/admin-api/rest/v1/backupRuns/list
		// GET https://sqladmin.googleapis.com/v1/projects/{project}/instances/{instance}/backupRuns
		SearchEndpointFunc:  ProjectLevelEndpointFuncWithSingleQuery("https://sqladmin.googleapis.com/v1/projects/%s/instances/%s/backupRuns"),
		UniqueAttributeKeys: []string{"instances", "backupRuns"},
		// HEALTH: https://cloud.google.com/sql/docs/mysql/admin-api/rest/v1/backupRuns#sqlbackuprunstatus
		// TODO: https://linear.app/overmind/issue/ENG-631/investigate-how-we-can-add-health-status-for-supporting-items
		// https://cloud.google.com/sql/docs/mysql/iam-permissions#permissions-gcloud
		IAMPermissions: []string{"cloudsql.backupRuns.get", "cloudsql.backupRuns.list"},
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
		ListEndpointFunc:    ProjectLevelListFunc("https://storage.googleapis.com/storage/v1/b?project=%s"),
		UniqueAttributeKeys: []string{"b"},
		IAMPermissions:      []string{"storage.buckets.get", "storage.buckets.list"},
	},
}
