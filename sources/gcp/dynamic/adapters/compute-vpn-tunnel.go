package adapters

import (
	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

// VPN Tunnel resource (Classic or HA VPN) scoped to a region.
// Reference: https://cloud.google.com/compute/docs/reference/rest/v1/vpnTunnels/get
// GET:  https://compute.googleapis.com/compute/v1/projects/{project}/regions/{region}/vpnTunnels/{vpnTunnel}
// LIST: https://compute.googleapis.com/compute/v1/projects/{project}/regions/{region}/vpnTunnels
var computeVpnTunnelAdapter = registerableAdapter{ //nolint:unused
	sdpType: gcpshared.ComputeVpnTunnel,
	meta: gcpshared.AdapterMeta{
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
		Scope:              gcpshared.ScopeRegional,
		GetEndpointBaseURLFunc: gcpshared.RegionalLevelEndpointFuncWithSingleQuery(
			"https://compute.googleapis.com/compute/v1/projects/%s/regions/%s/vpnTunnels/%s",
		),
		ListEndpointFunc: gcpshared.RegionLevelListFunc(
			"https://compute.googleapis.com/compute/v1/projects/%s/regions/%s/vpnTunnels",
		),
		// The list response uses the key "vpnTunnels" for items.
		UniqueAttributeKeys: []string{"vpnTunnels"},
		IAMPermissions: []string{
			"compute.vpnTunnels.get",
			"compute.vpnTunnels.list",
		},
		// HEALTH: https://cloud.google.com/compute/docs/reference/rest/v1/vpnTunnels#Status => status
		// TODO: https://linear.app/overmind/issue/ENG-631/investigate-how-we-can-add-health-status-for-supporting-items
	},
	blastPropagation: map[string]*gcpshared.Impact{
		// The peer IP address of the remote VPN gateway.
		"peerIp": gcpshared.IPImpactBothWays,
		"targetVpnGateway": {
			ToSDPItemType: gcpshared.ComputeVpnGateway,
			Description:   "They are tightly coupled.",
			BlastPropagation: &sdp.BlastPropagation{
				In:  true,
				Out: true,
			},
		},
		"vpnGateway": {
			ToSDPItemType: gcpshared.ComputeVpnGateway,
			Description:   "They are tightly coupled.",
			BlastPropagation: &sdp.BlastPropagation{
				In:  true,
				Out: true,
			},
		},
		"peerExternalGateway": {
			ToSDPItemType: gcpshared.ComputeExternalVpnGateway,
			Description:   "They are tightly coupled.",
			BlastPropagation: &sdp.BlastPropagation{
				In:  true,
				Out: true,
			},
		},
		"peerGcpGateway": {
			ToSDPItemType: gcpshared.ComputeVpnGateway,
			Description:   "They are tightly coupled.",
			BlastPropagation: &sdp.BlastPropagation{
				In:  true,
				Out: true,
			},
		},
		"router": {
			ToSDPItemType: gcpshared.ComputeRouter,
			Description:   "They are tightly coupled.",
			BlastPropagation: &sdp.BlastPropagation{
				In:  true,
				Out: true,
			},
		},
	},
	terraformMapping: gcpshared.TerraformMapping{
		Reference: "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/compute_vpn_tunnel",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_GET,
				TerraformQueryMap: "google_compute_vpn_tunnel.name",
			},
		},
	},
}.Register()
