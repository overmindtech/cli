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
		// TODO: https://linear.app/overmind/issue/ENG-631/investigate-how-we-can-add-health-status-for-supporting-items
		// state: https://cloud.google.com/bigtable/docs/reference/admin/rest/v2/projects.instances#State
	},
	// The Bigtable Instance does not contain any fields that would cause blast propagation.
	blastPropagation: map[string]*gcpshared.Impact{},
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
