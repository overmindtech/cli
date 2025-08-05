package adapters

import (
	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

var spannerInstanceAdapter = registerableAdapter{ //nolint:unused
	sdpType: gcpshared.SpannerInstance,
	meta: gcpshared.AdapterMeta{
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_DATABASE,
		Scope:              gcpshared.ScopeProject,
		// Reference: https://cloud.google.com/spanner/docs/reference/rest/v1/projects.instances/get?rep_location=global
		// https://spanner.googleapis.com/v1/projects/*/instances/*
		GetEndpointBaseURLFunc: gcpshared.ProjectLevelEndpointFuncWithSingleQuery("https://spanner.googleapis.com/v1/projects/%s/instances/%s"),
		// Reference: https://cloud.google.com/spanner/docs/reference/rest/v1/projects.instances/list?rep_location=global
		// https://spanner.googleapis.com/v1/projects/*/instances
		ListEndpointFunc:    gcpshared.ProjectLevelListFunc("https://spanner.googleapis.com/v1/projects/%s/instances"),
		UniqueAttributeKeys: []string{"instances"},
		// HEALTH: https://cloud.google.com/spanner/docs/reference/rest/v1/projects.instances#State
		// TODO: https://linear.app/overmind/issue/ENG-631/investigate-how-we-can-add-health-status-for-supporting-items
	},
	blastPropagation: map[string]*gcpshared.Impact{
		"config": {
			ToSDPITemType: gcpshared.SpannerInstanceConfig,
			Description:   "If the Spanner Instance Config is deleted or updated: The Spanner Instance may fail to operate correctly. If the Spanner Instance is updated: The config remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{
				In:  true,
				Out: false,
			},
		},
	},
	terraformMapping: gcpshared.TerraformMapping{
		Reference: "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/spanner_instance",
		Mappings: []*sdp.TerraformMapping{
			{
				// TODO: Confirm this is the name that we want to use
				TerraformMethod:   sdp.QueryMethod_GET,
				TerraformQueryMap: "google_spanner_instance.name",
			},
		},
	},
}.Register()
