package adapters

import (
	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

var _ = registerableAdapter{
	sdpType: gcpshared.ComputeRouter,
	meta: gcpshared.AdapterMeta{
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
		LocationLevel:      gcpshared.RegionalLevel,
		// Reference: https://cloud.google.com/compute/docs/reference/rest/v1/routers/get
		// https://compute.googleapis.com/compute/v1/projects/{project}/regions/{region}/routers/{router}
		GetEndpointFunc: gcpshared.RegionalLevelEndpointFunc("https://compute.googleapis.com/compute/v1/projects/%s/regions/%s/routers/%s"),
		// Reference: https://cloud.google.com/compute/docs/reference/rest/v1/routers/list
		// https://compute.googleapis.com/compute/v1/projects/{project}/regions/{region}/routers
		ListEndpointFunc: gcpshared.RegionLevelListFunc("https://compute.googleapis.com/compute/v1/projects/%s/regions/%s/routers"),
		// Provide a no-op search for terraform mapping support with full resource ID.
		// Expected search query: projects/{project}/regions/{region}/routers/{router}
		// Returns empty URL to trigger GET with the provided full name.
		SearchEndpointFunc: func(query string, location gcpshared.LocationInfo) string {
			return ""
		},
		SearchDescription:   "Search with full ID: projects/[project]/regions/[region]/routers/[router] (used for terraform mapping).",
		UniqueAttributeKeys: []string{"routers"},
		IAMPermissions:      []string{"compute.routers.get", "compute.routers.list"},
		PredefinedRole:      "roles/compute.viewer",
	},
	blastPropagation: map[string]*gcpshared.Impact{
		"network": gcpshared.ComputeNetworkImpactInOnly,
		"interfaces.linkedInterconnectAttachment": {
			ToSDPItemType: gcpshared.ComputeInterconnectAttachment,
			Description:   "They are tightly coupled.",
			BlastPropagation: &sdp.BlastPropagation{
				In:  true,
				Out: true,
			},
		},
		"interfaces.privateIpAddress":     gcpshared.IPImpactBothWays,
		"interfaces.subnetwork":           gcpshared.ComputeSubnetworkImpactInOnly,
		"bgpPeers.peerIpAddress":          gcpshared.IPImpactBothWays,
		"bgpPeers.ipAddress":              gcpshared.IPImpactBothWays,
		"bgpPeers.ipv4NexthopAddress":     gcpshared.IPImpactBothWays,
		"bgpPeers.peerIpv4NexthopAddress": gcpshared.IPImpactBothWays,
		"nats.natIps": {
			ToSDPItemType: stdlib.NetworkIP,
			Description:   "If the NAT IP address is deleted or updated: The Router NAT may fail to function correctly. If the Router NAT is updated: The IP address remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{
				In:  true,
				Out: false,
			},
		},
		"nats.drainNatIps": {
			ToSDPItemType: stdlib.NetworkIP,
			Description:   "If the draining NAT IP address is deleted or updated: The Router NAT may fail to drain correctly. If the Router NAT is updated: The IP address remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{
				In:  true,
				Out: false,
			},
		},
		"nats.subnetworks.name":      gcpshared.ComputeSubnetworkImpactInOnly,
		"nats.nat64Subnetworks.name": gcpshared.ComputeSubnetworkImpactInOnly,
		"interfaces.linkedVpnTunnel": {
			ToSDPItemType: gcpshared.ComputeVpnTunnel,
			Description:   "They are tightly coupled.",
			BlastPropagation: &sdp.BlastPropagation{
				In:  true,
				Out: true,
			},
		},
		// Child resource: RoutePolicy - Router can list all its route policies via listRoutePolicies
		// This is a link from parent to child via SEARCH
		// The child adapter must support SEARCH method that accepts router name as a parameter
		"name": {
			ToSDPItemType: gcpshared.ComputeRoutePolicy,
			Description:   "If the Router is deleted or updated: All associated Route Policies may become invalid or inaccessible. If a Route Policy is updated: The router remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{
				In:  false,
				Out: true,
			},
			IsParentToChild: true, // Router discovers all its Route Policies via SEARCH
		},
		// Note: BgpRoute is also a child resource with listBgpRoutes endpoint, but we can only use "name"
		// once in the blastPropagation map. When BgpRoute adapter is created with SEARCH support,
		// we can consider using a different field or handling it separately.
	},
	terraformMapping: gcpshared.TerraformMapping{
		Reference:   "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/compute_router",
		Description: "id => projects/{{project}}/regions/{{region}}/routers/{{router}}",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_SEARCH,
				TerraformQueryMap: "google_compute_router.id",
			},
		},
	},
}.Register()
