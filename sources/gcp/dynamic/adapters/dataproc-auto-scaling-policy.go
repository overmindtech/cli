package adapters

import (
	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

// Dataproc AutoscalingPolicy adapter - manages autoscaling behavior for Dataproc clusters
// API Get:  https://cloud.google.com/dataproc/docs/reference/rest/v1/projects.regions.autoscalingPolicies/get
// API List: https://cloud.google.com/dataproc/docs/reference/rest/v1/projects.regions.autoscalingPolicies/list
// GET  https://dataproc.googleapis.com/v1/projects/{project}/regions/{region}/autoscalingPolicies/{autoscalingPolicyId}
// LIST https://dataproc.googleapis.com/v1/projects/{project}/regions/{region}/autoscalingPolicies
var dataprocAutoScalingPolicyAdapter = registerableAdapter{ //nolint:unused
	sdpType: gcpshared.DataprocAutoscalingPolicy,
	meta: gcpshared.AdapterMeta{
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_CONFIGURATION,
		Scope:              gcpshared.ScopeRegional,
		GetEndpointBaseURLFunc: gcpshared.RegionalLevelEndpointFuncWithSingleQuery(
			"https://dataproc.googleapis.com/v1/projects/%s/regions/%s/autoscalingPolicies/%s",
		),
		ListEndpointFunc: gcpshared.RegionLevelListFunc(
			"https://dataproc.googleapis.com/v1/projects/%s/regions/%s/autoscalingPolicies",
		),
		UniqueAttributeKeys: []string{"autoscalingPolicies"},
		IAMPermissions: []string{
			"dataproc.autoscalingPolicies.get",
			"dataproc.autoscalingPolicies.list",
		},
		PredefinedRole: "roles/dataproc.viewer",
	},
	blastPropagation: map[string]*gcpshared.Impact{
		// AutoscalingPolicies don't directly reference other resources,
		// but they are referenced by Dataproc clusters via config.autoscalingConfig.policyUri
		// The reverse relationship is handled in the cluster adapter
	},
	terraformMapping: gcpshared.TerraformMapping{
		Reference:   "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/dataproc_autoscaling_policy",
		Description: "Use GET by autoscaling policy name.",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_GET,
				TerraformQueryMap: "google_dataproc_autoscaling_policy.name",
			},
		},
	},
}.Register()
