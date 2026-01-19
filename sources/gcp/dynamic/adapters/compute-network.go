package adapters

import (
	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

// Compute Network adapter for VPC networks
var _ = registerableAdapter{
	sdpType: gcpshared.ComputeNetwork,
	meta: gcpshared.AdapterMeta{
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
		LocationLevel:      gcpshared.ProjectLevel,
		// https://compute.googleapis.com/compute/v1/projects/{project}/global/networks/{network}
		GetEndpointFunc: gcpshared.ProjectLevelEndpointFuncWithSingleQuery("https://compute.googleapis.com/compute/v1/projects/%s/global/networks/%s"),
		// https://compute.googleapis.com/compute/v1/projects/{project}/global/networks
		ListEndpointFunc:    gcpshared.ProjectLevelListFunc("https://compute.googleapis.com/compute/v1/projects/%s/global/networks"),
		UniqueAttributeKeys: []string{"networks"},
		IAMPermissions:      []string{"compute.networks.get", "compute.networks.list"},
		PredefinedRole:      "roles/compute.viewer",
	},
	blastPropagation: map[string]*gcpshared.Impact{
		"gatewayIPv4": gcpshared.IPImpactBothWays,
		"subnetworks": {
			Description:      "If the Compute Subnetwork is deleted: The network remains unaffected, but its subnetwork configuration may change. If the network is deleted: All associated subnetworks are also deleted.",
			ToSDPItemType:    gcpshared.ComputeSubnetwork,
			BlastPropagation: &sdp.BlastPropagation{In: true, Out: true},
		},
		"peerings.network": {
			Description:      "If the Compute Network Peering is deleted: The network remains unaffected, but its peering configuration may change. If the network is deleted: All associated peerings are also deleted.",
			ToSDPItemType:    gcpshared.ComputeNetwork,
			BlastPropagation: &sdp.BlastPropagation{In: true, Out: true},
		},
		"firewallPolicy": {
			Description:      "If the Compute Firewall Policy is updated: The network's security posture may change. If the network is updated: The firewall policy remains unaffected, but its application to the network may change.",
			ToSDPItemType:    gcpshared.ComputeFirewallPolicy,
			BlastPropagation: gcpshared.ImpactInOnly,
		},
	},
	terraformMapping: gcpshared.TerraformMapping{
		Reference: "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/compute_network",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_GET,
				TerraformQueryMap: "google_compute_network.name",
			},
		},
	},
}.Register()
