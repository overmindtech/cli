package adapters

import (
	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

// Compute Subnetwork adapter for VPC subnetworks
var _ = registerableAdapter{
	sdpType: gcpshared.ComputeSubnetwork,
	meta: gcpshared.AdapterMeta{
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
		LocationLevel:      gcpshared.RegionalLevel,
		// Reference: https://cloud.google.com/compute/docs/reference/rest/v1/subnetworks/get
		// https://compute.googleapis.com/compute/v1/projects/{project}/regions/{region}/subnetworks/{subnetwork}
		GetEndpointFunc: gcpshared.RegionalLevelEndpointFuncWithSingleQuery("https://compute.googleapis.com/compute/v1/projects/%s/regions/%s/subnetworks/%s"),
		// https://compute.googleapis.com/compute/v1/projects/{project}/regions/{region}/subnetworks
		ListEndpointFunc:    gcpshared.RegionLevelListFunc("https://compute.googleapis.com/compute/v1/projects/%s/regions/%s/subnetworks"),
		UniqueAttributeKeys: []string{"subnetworks"},
		IAMPermissions:      []string{"compute.subnetworks.get", "compute.subnetworks.list"},
		PredefinedRole:      "roles/compute.viewer",
	},
	blastPropagation: map[string]*gcpshared.Impact{
		"network": {
			Description:      "If the Compute Network is updated: The firewall rules may no longer apply correctly. If the firewall is updated: The network remains unaffected, but its security posture may change.",
			ToSDPItemType:    gcpshared.ComputeNetwork,
			BlastPropagation: &sdp.BlastPropagation{In: true, Out: true},
		},
		"gatewayAddress": gcpshared.IPImpactBothWays,
		"secondaryIpRanges.reservedInternalRange": {
			Description:      "If the Reserved Internal Range is deleted or updated: The subnetwork's secondary IP range configuration may become invalid. If the subnetwork is updated: The internal range remains unaffected.",
			ToSDPItemType:    gcpshared.NetworkConnectivityInternalRange,
			BlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
		},
		"ipCollection": {
			Description:      "If the Public Delegated Prefix is deleted or updated: The subnetwork may lose its IP allocation source (BYOIP). If the subnetwork is updated: The prefix remains unaffected.",
			ToSDPItemType:    gcpshared.ComputePublicDelegatedPrefix,
			BlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
		},
	},
	terraformMapping: gcpshared.TerraformMapping{
		Reference: "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/compute_subnetwork",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_GET,
				TerraformQueryMap: "google_compute_subnetwork.name",
			},
		},
	},
}.Register()
