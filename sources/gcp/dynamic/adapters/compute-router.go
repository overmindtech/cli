package adapters

import (
	"fmt"

	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

var computeRouterAdapter = registerableAdapter{ //nolint:unused
	sdpType: gcpshared.ComputeRouter,
	meta: gcpshared.AdapterMeta{
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
		Scope:              gcpshared.ScopeRegional,
		// Reference: https://cloud.google.com/compute/docs/reference/rest/v1/routers/get
		// https://compute.googleapis.com/compute/v1/projects/{project}/regions/{region}/routers/{router}
		GetEndpointBaseURLFunc: gcpshared.RegionalLevelEndpointFuncWithSingleQuery("https://compute.googleapis.com/compute/v1/projects/%s/regions/%s/routers/%s"),
		// Reference: https://cloud.google.com/compute/docs/reference/rest/v1/routers/list
		// https://compute.googleapis.com/compute/v1/projects/{project}/regions/{region}/routers
		ListEndpointFunc: gcpshared.RegionLevelListFunc("https://compute.googleapis.com/compute/v1/projects/%s/regions/%s/routers"),
		// Provide a no-op search for terraform mapping support with full resource ID.
		// Expected search query: projects/{project}/regions/{region}/routers/{router}
		SearchEndpointFunc: func(adapterInitParams ...string) (gcpshared.EndpointFunc, error) {
			if len(adapterInitParams) != 2 || adapterInitParams[0] == "" || adapterInitParams[1] == "" {
				return nil, fmt.Errorf("projectID and region cannot be empty: %v", adapterInitParams)
			}
			return nil, nil // runtime will use GET with provided full name
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
		"interfaces.subnetwork":           gcpshared.ComputeNetworkImpactInOnly,
		"bgpPeers.peerIpAddress":          gcpshared.IPImpactBothWays,
		"bgpPeers.ipAddress":              gcpshared.IPImpactBothWays,
		"bgpPeers.ipv4NexthopAddress":     gcpshared.IPImpactBothWays,
		"bgpPeers.peerIpv4NexthopAddress": gcpshared.IPImpactBothWays,
		"nats.natIps":                     gcpshared.IPImpactBothWays,
		"nats.drainNatIps":                gcpshared.IPImpactBothWays,
		"nats.subnetworks.name":           gcpshared.ComputeNetworkImpactInOnly,
		"nats.nat64Subnetworks.name":      gcpshared.ComputeNetworkImpactInOnly,
		"interfaces.linkedVpnTunnel": {
			ToSDPItemType: gcpshared.ComputeVpnTunnel,
			Description:   "They are tightly coupled.",
			BlastPropagation: &sdp.BlastPropagation{
				In:  true,
				Out: true,
			},
		},
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
