package adapters

import (
	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

// Target HTTP Proxy (global, project-level) resource.
// Reference: https://cloud.google.com/compute/docs/reference/rest/v1/targetHttpProxies/get
// GET:  https://compute.googleapis.com/compute/v1/projects/{project}/global/targetHttpProxies/{targetHttpProxy}
// LIST: https://compute.googleapis.com/compute/v1/projects/{project}/global/targetHttpProxies
var computeTargetHttpProxyAdapter = registerableAdapter{ //nolint:unused
	sdpType: gcpshared.ComputeTargetHttpProxy,
	meta: gcpshared.AdapterMeta{
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
		Scope:              gcpshared.ScopeProject,
		GetEndpointBaseURLFunc: gcpshared.ProjectLevelEndpointFuncWithSingleQuery(
			"https://compute.googleapis.com/compute/v1/projects/%s/global/targetHttpProxies/%s",
		),
		ListEndpointFunc: gcpshared.ProjectLevelListFunc(
			"https://compute.googleapis.com/compute/v1/projects/%s/global/targetHttpProxies",
		),
		UniqueAttributeKeys: []string{"targetHttpProxies"},
		IAMPermissions: []string{
			"compute.targetHttpProxies.get",
			"compute.targetHttpProxies.list",
		},
	},
	blastPropagation: map[string]*gcpshared.Impact{
		"urlMap": {
			ToSDPItemType: gcpshared.ComputeUrlMap,
			Description:   "If the URL Map is updated or deleted: The HTTP proxy routing behavior may change or break. If the proxy changes: The URL map remains structurally unaffected.",
			BlastPropagation: &sdp.BlastPropagation{
				In: true,
			},
		},
	},
	terraformMapping: gcpshared.TerraformMapping{
		Reference: "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/compute_target_http_proxy",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_GET,
				TerraformQueryMap: "google_compute_target_http_proxy.name",
			},
		},
	},
}.Register()
