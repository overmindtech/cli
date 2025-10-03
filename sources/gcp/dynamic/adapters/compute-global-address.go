package adapters

import (
	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

// Global (external) IP address allocated at the project level.
// Reference: https://cloud.google.com/compute/docs/reference/rest/v1/globalAddresses/get
// GET:  https://compute.googleapis.com/compute/v1/projects/{project}/global/addresses/{address}
// LIST: https://compute.googleapis.com/compute/v1/projects/{project}/global/addresses
var computeGlobalAddressAdapter = registerableAdapter{ //nolint:unused
	sdpType: gcpshared.ComputeGlobalAddress,
	meta: gcpshared.AdapterMeta{
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
		Scope:              gcpshared.ScopeProject,
		GetEndpointBaseURLFunc: gcpshared.ProjectLevelEndpointFuncWithSingleQuery(
			"https://compute.googleapis.com/compute/v1/projects/%s/global/addresses/%s",
		),
		ListEndpointFunc: gcpshared.ProjectLevelListFunc(
			"https://compute.googleapis.com/compute/v1/projects/%s/global/addresses",
		),
		// The list response uses the key "addresses" for items.
		UniqueAttributeKeys: []string{"addresses"},
		IAMPermissions: []string{
			// Permissions required to read global addresses (Compute Engine)
			"compute.addresses.get",
			"compute.addresses.list",
		},
		PredefinedRole: "roles/compute.viewer",
		// HEALTH: https://cloud.google.com/compute/docs/reference/rest/v1/globalAddresses#Status => status
		// TODO: https://linear.app/overmind/issue/ENG-631/investigate-how-we-can-add-health-status-for-supporting-items
	},
	blastPropagation: map[string]*gcpshared.Impact{
		"subnetwork": gcpshared.ComputeNetworkImpactInOnly,
		"network":    gcpshared.ComputeNetworkImpactInOnly,
		"address":    gcpshared.IPImpactBothWays,
	},
	terraformMapping: gcpshared.TerraformMapping{
		Reference: "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/compute_global_address",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_GET,
				TerraformQueryMap: "google_compute_global_address.name",
			},
		},
	},
}.Register()
