package adapters

import (
	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

// External VPN Gateway (project-level, global) resource representing an on-premises VPN device for Classic/HA VPN.
// Reference: https://cloud.google.com/compute/docs/reference/rest/v1/externalVpnGateways/get
// GET:  https://compute.googleapis.com/compute/v1/projects/{project}/global/externalVpnGateways/{externalVpnGateway}
// LIST: https://compute.googleapis.com/compute/v1/projects/{project}/global/externalVpnGateways
var computeExternalVpnGatewayAdapter = registerableAdapter{ //nolint:unused
	sdpType: gcpshared.ComputeExternalVpnGateway,
	meta: gcpshared.AdapterMeta{
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
		Scope:              gcpshared.ScopeProject,
		GetEndpointBaseURLFunc: gcpshared.ProjectLevelEndpointFuncWithSingleQuery(
			"https://compute.googleapis.com/compute/v1/projects/%s/global/externalVpnGateways/%s",
		),
		ListEndpointFunc: gcpshared.ProjectLevelListFunc(
			"https://compute.googleapis.com/compute/v1/projects/%s/global/externalVpnGateways",
		),
		UniqueAttributeKeys: []string{"externalVpnGateways"},
		IAMPermissions: []string{
			"compute.externalVpnGateways.get",
			"compute.externalVpnGateways.list",
		},
	},
	blastPropagation: map[string]*gcpshared.Impact{
		"interfaces.ipAddress":   gcpshared.IPImpactBothWays,
		"interfaces.ipv6Address": gcpshared.IPImpactBothWays,
	},
	terraformMapping: gcpshared.TerraformMapping{
		Reference: "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/compute_external_vpn_gateway",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_GET,
				TerraformQueryMap: "google_compute_external_vpn_gateway.name",
			},
		},
	},
}.Register()
