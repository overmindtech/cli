package adapters

import (
	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

// HA VPN Gateway (regional) resource.
// Reference: https://cloud.google.com/compute/docs/reference/rest/v1/vpnGateways/get
// GET:  https://compute.googleapis.com/compute/v1/projects/{project}/regions/{region}/vpnGateways/{vpnGateway}
// LIST: https://compute.googleapis.com/compute/v1/projects/{project}/regions/{region}/vpnGateways
var computeVpnGatewayAdapter = registerableAdapter{ //nolint:unused
	sdpType: gcpshared.ComputeVpnGateway,
	meta: gcpshared.AdapterMeta{
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
		Scope:              gcpshared.ScopeRegional,
		GetEndpointBaseURLFunc: gcpshared.RegionalLevelEndpointFuncWithSingleQuery(
			"https://compute.googleapis.com/compute/v1/projects/%s/regions/%s/vpnGateways/%s",
		),
		ListEndpointFunc: gcpshared.RegionLevelListFunc(
			"https://compute.googleapis.com/compute/v1/projects/%s/regions/%s/vpnGateways",
		),
		// The list response uses the key "vpnGateways" for items.
		UniqueAttributeKeys: []string{"vpnGateways"},
		IAMPermissions: []string{
			"compute.vpnGateways.get",
			"compute.vpnGateways.list",
		},
	},
	blastPropagation: map[string]*gcpshared.Impact{
		// Network associated with the VPN gateway.
		"network": gcpshared.ComputeNetworkImpactInOnly,
		// IP addresses assigned to VPN interfaces (each interface may have an external IP).
		"vpnInterfaces.ipAddress":   gcpshared.IPImpactBothWays,
		"vpnInterfaces.ipv6Address": gcpshared.IPImpactBothWays,
	},
	terraformMapping: gcpshared.TerraformMapping{
		Reference: "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/compute_ha_vpn_gateway",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_GET,
				TerraformQueryMap: "google_compute_ha_vpn_gateway.name",
			},
		},
	},
}.Register()
