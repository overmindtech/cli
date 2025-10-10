package adapters

import (
	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

// Compute Firewall adapter for VPC firewall rules
var _ = registerableAdapter{
	sdpType: gcpshared.ComputeFirewall,
	meta: gcpshared.AdapterMeta{
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
		Scope:              gcpshared.ScopeProject,
		// https://compute.googleapis.com/compute/v1/projects/{project}/global/firewalls/{firewall}
		GetEndpointBaseURLFunc: gcpshared.ProjectLevelEndpointFuncWithSingleQuery("https://compute.googleapis.com/compute/v1/projects/%s/global/firewalls/%s"),
		// Reference: https://cloud.google.com/compute/docs/reference/rest/v1/firewalls/list
		// https://compute.googleapis.com/compute/v1/projects/{project}/global/firewalls
		ListEndpointFunc:    gcpshared.ProjectLevelListFunc("https://compute.googleapis.com/compute/v1/projects/%s/global/firewalls"),
		UniqueAttributeKeys: []string{"firewalls"},
		IAMPermissions:      []string{"compute.firewalls.get", "compute.firewalls.list"},
		PredefinedRole:      "roles/compute.viewer",
	},
	blastPropagation: map[string]*gcpshared.Impact{
		"network": {
			Description:      "If the Compute Network is updated: The firewall rules may no longer apply correctly. If the firewall is updated: The network remains unaffected, but its security posture may change.",
			ToSDPItemType:    gcpshared.ComputeNetwork,
			BlastPropagation: &sdp.BlastPropagation{In: true, Out: true},
		},
		"sourceServiceAccounts": gcpshared.IAMServiceAccountImpactInOnly,
		"targetServiceAccounts": gcpshared.IAMServiceAccountImpactInOnly,
	},
	terraformMapping: gcpshared.TerraformMapping{
		Reference: "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/compute_firewall",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_GET,
				TerraformQueryMap: "google_compute_firewall.name",
			},
		},
	},
}.Register()
