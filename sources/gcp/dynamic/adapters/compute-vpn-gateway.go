package adapters

import (
	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

// HA VPN Gateway (regional) resource.
// Reference: https://cloud.google.com/compute/docs/reference/rest/v1/vpnGateways/get
// GET:  https://compute.googleapis.com/compute/v1/projects/{project}/regions/{region}/vpnGateways/{vpnGateway}
// LIST: https://compute.googleapis.com/compute/v1/projects/{project}/regions/{region}/vpnGateways
var _ = registerableAdapter{
	sdpType: gcpshared.ComputeVpnGateway,
	meta: gcpshared.AdapterMeta{
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
		LocationLevel:      gcpshared.RegionalLevel,
		GetEndpointFunc: gcpshared.RegionalLevelEndpointFunc(
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
		PredefinedRole: "roles/compute.viewer",
	},
	blastPropagation: map[string]*gcpshared.Impact{
		// Network associated with the VPN gateway.
		"network": gcpshared.ComputeNetworkImpactInOnly,
		// IP addresses assigned to VPN interfaces (each interface may have an external IP).
		"vpnInterfaces.ipAddress":   gcpshared.IPImpactBothWays,
		"vpnInterfaces.ipv6Address": gcpshared.IPImpactBothWays,
		// Interconnect attachment used for HA VPN over Cloud Interconnect.
		"vpnInterfaces.interconnectAttachment": {
			ToSDPItemType: gcpshared.ComputeInterconnectAttachment,
			Description:   "If the Interconnect Attachment is deleted or updated: The VPN gateway interface may fail to operate correctly. If the VPN gateway is deleted or updated: The interconnect attachment may become disconnected or unusable. They are tightly coupled.",
			BlastPropagation: &sdp.BlastPropagation{
				In:  true,
				Out: true,
			},
		},
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
