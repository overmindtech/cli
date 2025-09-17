package adapters

import (
	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

// SSL Policy (global, project-level) defines SSL/TLS connection settings for secure network communications in Google Cloud Load Balancers
// GCP Ref (GET): https://cloud.google.com/compute/docs/reference/rest/v1/sslPolicies/get
// GET  https://compute.googleapis.com/compute/v1/projects/{project}/global/sslPolicies/{sslPolicy}
// LIST https://compute.googleapis.com/compute/v1/projects/{project}/global/sslPolicies
var _ = registerableAdapter{
	sdpType: gcpshared.ComputeSSLPolicy,
	meta: gcpshared.AdapterMeta{
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_SECURITY,
		Scope:              gcpshared.ScopeProject,
		GetEndpointBaseURLFunc: gcpshared.ProjectLevelEndpointFuncWithSingleQuery(
			"https://compute.googleapis.com/compute/v1/projects/%s/global/sslPolicies/%s",
		),
		ListEndpointFunc: gcpshared.ProjectLevelListFunc(
			"https://compute.googleapis.com/compute/v1/projects/%s/global/sslPolicies",
		),
		UniqueAttributeKeys: []string{"sslPolicies"},
		IAMPermissions: []string{
			"compute.sslPolicies.get",
			"compute.sslPolicies.list",
		},
	},
	blastPropagation: map[string]*gcpshared.Impact{
		// SSL Policies are configuration-only resources that define TLS/SSL parameters
		// They don't have dependencies on other GCP resources, but are referenced by:
		// - Target HTTPS Proxies (via sslPolicy field)
		// - Target SSL Proxies (via sslPolicy field)
	},
	terraformMapping: gcpshared.TerraformMapping{
		Reference: "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/compute_ssl_policy",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_GET,
				TerraformQueryMap: "google_compute_ssl_policy.name",
			},
		},
	},
}.Register()
