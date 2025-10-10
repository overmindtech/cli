package adapters

import (
	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

// BigTable Admin App Profile adapter for Cloud Bigtable application profiles
var _ = registerableAdapter{
	sdpType: gcpshared.BigTableAdminAppProfile,
	meta: gcpshared.AdapterMeta{
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_CONFIGURATION,
		Scope:              gcpshared.ScopeProject,
		// Reference: https://cloud.google.com/bigtable/docs/reference/admin/rest/v2/projects.instances.appProfiles/get
		// GET https://bigtableadmin.googleapis.com/v2/{name=projects/*/instances/*/appProfiles/*}
		GetEndpointBaseURLFunc: gcpshared.ProjectLevelEndpointFuncWithTwoQueries("https://bigtableadmin.googleapis.com/v2/projects/%s/instances/%s/appProfiles/%s"),
		// Reference: https://cloud.google.com/bigtable/docs/reference/admin/rest/v2/projects.instances.appProfiles/list
		SearchEndpointFunc:  gcpshared.ProjectLevelEndpointFuncWithSingleQuery("https://bigtableadmin.googleapis.com/v2/projects/%s/instances/%s/appProfiles"),
		SearchDescription:   "Search for BigTable App Profiles in an instance. Use the format \"instance\" or \"projects/[project_id]/instances/[instance_name]/appProfiles/[app_profile_id]\" which is supported for terraform mappings.",
		UniqueAttributeKeys: []string{"instances", "appProfiles"},
		IAMPermissions:      []string{"bigtable.appProfiles.get", "bigtable.appProfiles.list"},
		PredefinedRole:      "roles/bigtable.viewer",
	},
	blastPropagation: map[string]*gcpshared.Impact{
		"name": {
			ToSDPItemType:    gcpshared.BigTableAdminInstance,
			Description:      "If the BigTableAdmin Instance is deleted or updated: The AppProfile may become invalid or inaccessible. If the AppProfile is updated: The instance remains unaffected.",
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		"multiClusterRoutingUseAny.clusterIds": {
			ToSDPItemType:    gcpshared.BigTableAdminCluster,
			Description:      "If the BigTableAdmin Cluster is deleted or updated: The AppProfile may lose routing capabilities or fail to access data. If the AppProfile is updated: The cluster remains unaffected.",
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		"singleClusterRouting.clusterId": {
			ToSDPItemType:    gcpshared.BigTableAdminCluster,
			Description:      "If the BigTableAdmin Cluster is deleted or updated: The AppProfile may lose routing capabilities or fail to access data. If the AppProfile is updated: The cluster remains unaffected.",
			BlastPropagation: gcpshared.ImpactInOnly,
		},
	},
	terraformMapping: gcpshared.TerraformMapping{
		Reference:   "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/bigtable_app_profile",
		Description: "id => projects/{{project}}/instances/{{instance}}/appProfiles/{{app_profile_id}}",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_SEARCH,
				TerraformQueryMap: "google_bigtable_app_profile.id",
			},
		},
	},
}.Register()
