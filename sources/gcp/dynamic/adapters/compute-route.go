package adapters

import (
	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

// Compute Route adapter for VPC routes
var _ = registerableAdapter{
	sdpType: gcpshared.ComputeRoute,
	meta: gcpshared.AdapterMeta{
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
		LocationLevel:      gcpshared.ProjectLevel,
		// https://compute.googleapis.com/compute/v1/projects/{project}/global/routes/{route}
		GetEndpointFunc: gcpshared.ProjectLevelEndpointFuncWithSingleQuery("https://compute.googleapis.com/compute/v1/projects/%s/global/routes/%s"),
		// https://compute.googleapis.com/compute/v1/projects/{project}/global/routes
		ListEndpointFunc:    gcpshared.ProjectLevelListFunc("https://compute.googleapis.com/compute/v1/projects/%s/global/routes"),
		UniqueAttributeKeys: []string{"routes"},
		IAMPermissions:      []string{"compute.routes.get", "compute.routes.list"},
		PredefinedRole:      "roles/compute.viewer",
	},
	blastPropagation: map[string]*gcpshared.Impact{
		// https://cloud.google.com/compute/docs/reference/rest/v1/routes/get
		// Network that the route belongs to
		"network": {
			Description:   "If the Compute Network is updated: The route may no longer be valid or correctly associated. If the route is updated: The network remains unaffected, but its routing behavior may change.",
			ToSDPItemType: gcpshared.ComputeNetwork,
			BlastPropagation: &sdp.BlastPropagation{
				In:  true,
				Out: true,
			},
		},
		// Network that the route forwards traffic to, so the relationship will/may be different
		"nextHopNetwork": {
			Description:   "If the Compute Network is updated: The route may no longer forward traffic properly. If the route is updated: The network remains unaffected but traffic routed through it may be affected.",
			ToSDPItemType: gcpshared.ComputeNetwork,
			BlastPropagation: &sdp.BlastPropagation{
				In:  true,
				Out: true,
			},
		},
		"nextHopIp": {
			Description:   "The network IP address of an instance that should handle matching packets. Tightly coupled with the Compute Route.",
			ToSDPItemType: stdlib.NetworkIP,
			BlastPropagation: &sdp.BlastPropagation{
				In:  true,
				Out: true,
			},
		},
		"nextHopInstance": {
			Description:      "If the Compute Instance is updated: Routes using it as a next hop may break or change behavior. If the route is deleted: The instance remains unaffected but traffic that was previously using that route will be impacted.",
			ToSDPItemType:    gcpshared.ComputeInstance,
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		"nextHopVpnTunnel": {
			Description:   "If the VPN Tunnel is updated: The route may no longer forward traffic properly. If the route is updated: The VPN tunnel remains unaffected but traffic routed through it may be affected.",
			ToSDPItemType: gcpshared.ComputeVpnTunnel,
			BlastPropagation: &sdp.BlastPropagation{
				In:  true,
				Out: true,
			},
		},
		"nextHopGateway": {
			Description:      "If the Compute Gateway is updated: The route may no longer forward traffic properly. If the route is updated: The gateway remains unaffected but traffic routed through it may be affected.",
			ToSDPItemType:    gcpshared.ComputeGateway,
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		"nextHopHub": {
			// https://cloud.google.com/network-connectivity/docs/reference/networkconnectivity/rest/v1/projects.locations.global.hubs/get
			Description:   "The full resource name of the Network Connectivity Center hub that will handle matching packets. If the hub is updated: The route may no longer forward traffic properly. If the route is updated: The hub remains unaffected but traffic routed through it may be affected.",
			ToSDPItemType: gcpshared.NetworkConnectivityHub,
			BlastPropagation: &sdp.BlastPropagation{
				In:  true,
				Out: true,
			},
		},
		"nextHopIlb": {
			// https://cloud.google.com/compute/docs/reference/rest/v1/routes/get
			// Can be either a URL to a forwarding rule (loadBalancingScheme=INTERNAL) or an IP address
			// When it's a URL, it references the ForwardingRule. When it's an IP, it's the IP address of the forwarding rule.
			Description:   "The URL to a forwarding rule of type loadBalancingScheme=INTERNAL that should handle matching packets, or the IP address of the forwarding rule. If the Forwarding Rule is updated or deleted: The route may no longer forward traffic properly. If the route is updated: The forwarding rule remains unaffected but traffic routed through it may be affected.",
			ToSDPItemType: gcpshared.ComputeForwardingRule,
			BlastPropagation: &sdp.BlastPropagation{
				In:  true,
				Out: true,
			},
		},
		"nextHopInterconnectAttachment": {
			// https://cloud.google.com/compute/docs/reference/rest/v1/routes/get
			Description:   "The URL to an InterconnectAttachment which is the next hop for the route. If the Interconnect Attachment is updated or deleted: The route may no longer forward traffic properly. If the route is updated: The interconnect attachment remains unaffected but traffic routed through it may be affected.",
			ToSDPItemType: gcpshared.ComputeInterconnectAttachment,
			BlastPropagation: &sdp.BlastPropagation{
				In:  true,
				Out: true,
			},
		},
	},
	terraformMapping: gcpshared.TerraformMapping{
		Reference: "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/compute_route",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_GET,
				TerraformQueryMap: "google_compute_route.name",
			},
		},
	},
}.Register()
