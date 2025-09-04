package adapters

import (
	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

// HTTP Health Check (global, project-level) resource.
// Reference: https://cloud.google.com/compute/docs/reference/rest/v1/httpHealthChecks/get
// GET:  https://compute.googleapis.com/compute/v1/projects/{project}/global/httpHealthChecks/{httpHealthCheck}
// LIST: https://compute.googleapis.com/compute/v1/projects/{project}/global/httpHealthChecks
var _ = registerableAdapter{
	sdpType: gcpshared.ComputeHttpHealthCheck,
	meta: gcpshared.AdapterMeta{
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
		Scope:              gcpshared.ScopeProject,
		GetEndpointBaseURLFunc: gcpshared.ProjectLevelEndpointFuncWithSingleQuery(
			"https://compute.googleapis.com/compute/v1/projects/%s/global/httpHealthChecks/%s",
		),
		ListEndpointFunc: gcpshared.ProjectLevelListFunc(
			"https://compute.googleapis.com/compute/v1/projects/%s/global/httpHealthChecks",
		),
		// The list response uses the key "httpHealthChecks" for items.
		UniqueAttributeKeys: []string{"httpHealthChecks"},
		IAMPermissions: []string{
			"compute.httpHealthChecks.get",
			"compute.httpHealthChecks.list",
		},
	},
	// HTTP health checks are referenced by backend services and target pools for health monitoring.
	// Updates to health checks can affect traffic distribution and service availability.
	blastPropagation: map[string]*gcpshared.Impact{
		"host": gcpshared.IPImpactBothWays,
	},
	terraformMapping: gcpshared.TerraformMapping{
		Reference: "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/compute_http_health_check",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_GET,
				TerraformQueryMap: "google_compute_http_health_check.name",
			},
		},
	},
}.Register()