package adapters

import (
	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

var bigTableAdminInstanceAdapter = registerableAdapter{ //nolint:unused
	sdpType: gcpshared.BigTableAdminInstance,
	meta: gcpshared.AdapterMeta{
		// Reference: https://cloud.google.com/bigtable/docs/reference/admin/rest/v2/projects.instances/get
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_CONFIGURATION,
		Scope:              gcpshared.ScopeProject,
		// https://bigtableadmin.googleapis.com/v2/projects/*/instances/*
		GetEndpointBaseURLFunc: gcpshared.ProjectLevelEndpointFuncWithSingleQuery("https://bigtableadmin.googleapis.com/v2/projects/%s/instances/%s"),
		// https://bigtableadmin.googleapis.com/v2/projects/*/instances
		ListEndpointFunc:    gcpshared.ProjectLevelListFunc("https://bigtableadmin.googleapis.com/v2/projects/%s/instances"),
		UniqueAttributeKeys: []string{"instances"},
		IAMPermissions:      []string{"bigtable.instances.get", "bigtable.instances.list"},
		PredefinedRole:      "roles/bigtable.viewer",
		// TODO: https://linear.app/overmind/issue/ENG-631/investigate-how-we-can-add-health-status-for-supporting-items
		// state: https://cloud.google.com/bigtable/docs/reference/admin/rest/v2/projects.instances#State
	},
	blastPropagation: map[string]*gcpshared.Impact{
		// Forward link from parent to child via SEARCH
		// Link to all clusters in this instance (most fundamental infrastructure component)
		"name": {
			ToSDPItemType: gcpshared.BigTableAdminCluster,
			Description:   "If the BigTableAdmin Instance is deleted or updated: All associated Clusters may become invalid or inaccessible. If a Cluster is updated: The instance remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{
			    In: false,
				Out: true,
			},
			IsParentToChild: true,
		},
	},
	terraformMapping: gcpshared.TerraformMapping{
		Reference: "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/bigtable_instance",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_GET,
				TerraformQueryMap: "google_bigtable_instance.name",
			},
		},
	},
}.Register()
