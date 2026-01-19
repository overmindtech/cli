package adapters

import (
	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

// VPN Tunnel resource (Classic or HA VPN) scoped to a region.
// Reference: https://cloud.google.com/compute/docs/reference/rest/v1/vpnTunnels/get
// GET:  https://compute.googleapis.com/compute/v1/projects/{project}/regions/{region}/vpnTunnels/{vpnTunnel}
// LIST: https://compute.googleapis.com/compute/v1/projects/{project}/regions/{region}/vpnTunnels
var _ = registerableAdapter{
	sdpType: gcpshared.ComputeVpnTunnel,
	meta: gcpshared.AdapterMeta{
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
		LocationLevel:      gcpshared.RegionalLevel,
		GetEndpointFunc: gcpshared.RegionalLevelEndpointFuncWithSingleQuery(
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
		PredefinedRole: "roles/compute.viewer",
		// HEALTH: https://cloud.google.com/compute/docs/reference/rest/v1/vpnTunnels#Status => status
		// TODO: https://linear.app/overmind/issue/ENG-631/investigate-how-we-can-add-health-status-for-supporting-items
	},
	blastPropagation: map[string]*gcpshared.Impact{
		// The peer IP address of the remote VPN gateway.
		"peerIp": gcpshared.IPImpactBothWays,
		"targetVpnGateway": {
			ToSDPItemType: gcpshared.ComputeTargetVpnGateway,
			Description:   "If the Target VPN Gateway (Classic VPN) is deleted or updated: The VPN Tunnel may become invalid or fail to establish connections. If the VPN Tunnel is updated or deleted: The Target VPN Gateway may be affected as tunnels are tightly coupled to their gateway.",
			BlastPropagation: &sdp.BlastPropagation{
				In:  true,
				Out: true,
			},
		},
		"vpnGateway": {
			ToSDPItemType: gcpshared.ComputeVpnGateway,
			Description:   "If the HA VPN Gateway is deleted or updated: The VPN Tunnel may become invalid or fail to establish connections. If the VPN Tunnel is updated or deleted: The HA VPN Gateway may be affected as tunnels are tightly coupled to their gateway.",
			BlastPropagation: &sdp.BlastPropagation{
				In:  true,
				Out: true,
			},
		},
		"peerExternalGateway": {
			ToSDPItemType: gcpshared.ComputeExternalVpnGateway,
			Description:   "If the External VPN Gateway is deleted or updated: The VPN Tunnel may fail to establish connections with the peer. If the VPN Tunnel is updated or deleted: The External VPN Gateway remains unaffected, but the tunnel endpoint becomes inactive.",
			BlastPropagation: &sdp.BlastPropagation{
				In:  true,
				Out: false,
			},
		},
		"peerGcpGateway": {
			ToSDPItemType: gcpshared.ComputeVpnGateway,
			Description:   "If the peer HA VPN Gateway is deleted or updated: The VPN Tunnel may fail to establish VPC-to-VPC connections. If the VPN Tunnel is updated or deleted: The peer HA VPN Gateway remains unaffected, but the tunnel endpoint becomes inactive.",
			BlastPropagation: &sdp.BlastPropagation{
				In:  true,
				Out: false,
			},
		},
		"router": {
			ToSDPItemType: gcpshared.ComputeRouter,
			Description:   "If the Cloud Router is deleted or updated: The VPN Tunnel may lose dynamic routing capabilities (BGP). If the VPN Tunnel is updated or deleted: The Cloud Router may lose routes advertised through this tunnel.",
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
