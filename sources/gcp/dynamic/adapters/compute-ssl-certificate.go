package adapters

import (
	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

var computeSSLCertificateAdapter = registerableAdapter{ //nolint:unused
	sdpType: gcpshared.ComputeSSLCertificate,
	meta: gcpshared.AdapterMeta{
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_CONFIGURATION,
		Scope:              gcpshared.ScopeProject,
		// Reference: https://cloud.google.com/compute/docs/reference/rest/v1/sslCertificates/get
		// https://compute.googleapis.com/compute/v1/projects/{project}/global/sslCertificates/{sslCertificate}
		GetEndpointBaseURLFunc: gcpshared.ProjectLevelEndpointFuncWithSingleQuery("https://compute.googleapis.com/compute/v1/projects/%s/global/sslCertificates/%s"),
		// Reference: https://cloud.google.com/compute/docs/reference/rest/v1/sslCertificates/list
		// https://compute.googleapis.com/compute/v1/projects/{project}/global/sslCertificates
		ListEndpointFunc:    gcpshared.ProjectLevelListFunc("https://compute.googleapis.com/compute/v1/projects/%s/global/sslCertificates"),
		UniqueAttributeKeys: []string{"sslCertificates"},
		IAMPermissions:      []string{"compute.sslCertificates.get", "compute.sslCertificates.list"},
		PredefinedRole:      "roles/compute.viewer",
	},
	terraformMapping: gcpshared.TerraformMapping{
		Reference: "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/compute_ssl_certificate",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_GET,
				TerraformQueryMap: "google_compute_ssl_certificate.name",
			},
		},
	},
	blastPropagation: map[string]*gcpshared.Impact{
		// There is no blast propagation originating from Compute SSL Certificates
	},
}.Register()
