package adapters

import (
	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

// Target HTTPS Proxy (global, project-level) resource.
// Reference: https://cloud.google.com/compute/docs/reference/rest/v1/targetHttpsProxies/get
// GET:  https://compute.googleapis.com/compute/v1/projects/{project}/global/targetHttpsProxies/{targetHttpsProxy}
// LIST: https://compute.googleapis.com/compute/v1/projects/{project}/global/targetHttpsProxies
var computeTargetHttpsProxyAdapter = registerableAdapter{ //nolint:unused
	sdpType: gcpshared.ComputeTargetHttpsProxy,
	meta: gcpshared.AdapterMeta{
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
		Scope:              gcpshared.ScopeProject,
		GetEndpointBaseURLFunc: gcpshared.ProjectLevelEndpointFuncWithSingleQuery(
			"https://compute.googleapis.com/compute/v1/projects/%s/global/targetHttpsProxies/%s",
		),
		ListEndpointFunc: gcpshared.ProjectLevelListFunc(
			"https://compute.googleapis.com/compute/v1/projects/%s/global/targetHttpsProxies",
		),
		UniqueAttributeKeys: []string{"targetHttpsProxies"},
		IAMPermissions: []string{
			"compute.targetHttpsProxies.get",
			"compute.targetHttpsProxies.list",
		},
	},
	blastPropagation: map[string]*gcpshared.Impact{
		"urlMap": {
			ToSDPItemType: gcpshared.ComputeUrlMap,
			Description:   "If the URL Map is updated or deleted: The HTTPS proxy routing behavior may change or break. If the proxy changes: The URL map remains structurally unaffected.",
			BlastPropagation: &sdp.BlastPropagation{
				In: true,
			},
		},
		"sslCertificates": {
			ToSDPItemType: gcpshared.ComputeSSLCertificate,
			Description:   "If the SSL Certificate is updated or deleted: TLS handshakes may fail for the HTTPS proxy. If the proxy changes: The certificate resource remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{
				In: true,
			},
		},
		"sslPolicy": {
			ToSDPItemType: gcpshared.ComputeSSLPolicy,
			Description:   "If the SSL Policy is updated or deleted: TLS handshakes may fail for the HTTPS proxy. If the proxy changes: The SSL policy resource remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{
				In: true,
			},
		},
	},
	terraformMapping: gcpshared.TerraformMapping{
		Reference: "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/compute_target_https_proxy",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_GET,
				TerraformQueryMap: "google_compute_target_https_proxy.name",
			},
		},
	},
}.Register()
