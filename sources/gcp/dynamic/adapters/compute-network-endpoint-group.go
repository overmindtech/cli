package adapters

import (
	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

// Network Endpoint Group (NEG) zonal resource.
// Reference: https://cloud.google.com/compute/docs/reference/rest/v1/networkEndpointGroups/get
// GET:  https://compute.googleapis.com/compute/v1/projects/{project}/zones/{zone}/networkEndpointGroups/{networkEndpointGroup}
// LIST: https://compute.googleapis.com/compute/v1/projects/{project}/zones/{zone}/networkEndpointGroups
var computeNetworkEndpointGroupAdapter = registerableAdapter{ //nolint:unused
	sdpType: gcpshared.ComputeNetworkEndpointGroup,
	meta: gcpshared.AdapterMeta{
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
		Scope:              gcpshared.ScopeZonal,
		GetEndpointBaseURLFunc: gcpshared.ZoneLevelEndpointFuncWithSingleQuery(
			"https://compute.googleapis.com/compute/v1/projects/%s/zones/%s/networkEndpointGroups/%s",
		),
		ListEndpointFunc: gcpshared.ZoneLevelListFunc(
			"https://compute.googleapis.com/compute/v1/projects/%s/zones/%s/networkEndpointGroups",
		),
		// The list response uses the key "networkEndpointGroups" for items.
		UniqueAttributeKeys: []string{"networkEndpointGroups"},
		IAMPermissions: []string{
			"compute.networkEndpointGroups.get",
			"compute.networkEndpointGroups.list",
		},
	},
	blastPropagation: map[string]*gcpshared.Impact{
		// Parent VPC network reference (changes to network can impact NEG reachability; NEG changes do not impact network)
		"network": gcpshared.ComputeNetworkImpactInOnly,
		// Subnetwork reference (regional) – subnetwork changes can affect endpoints, NEG changes do not affect subnetwork
		"subnetwork": {
			ToSDPItemType:    gcpshared.ComputeSubnetwork,
			Description:      "If the Compute Subnetwork is updated: Endpoint reachability or configuration for the NEG may change. If the NEG is updated: The subnetwork remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
		},
		// Serverless NEG referencing a Cloud Run Service
		"cloudRun.service": {
			ToSDPItemType:    gcpshared.RunService,
			Description:      "If the Cloud Run Service is updated or deleted: Requests routed via the NEG may fail or change behavior. If the NEG changes: The Cloud Run service remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
		},
		// Serverless NEG referencing an App Engine service
		"appEngine.service": {
			ToSDPItemType:    gcpshared.AppEngineService,
			Description:      "If the App Engine Service is updated or deleted: Requests routed via the NEG may fail or change behavior. If the NEG changes: The App Engine service remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
		},
		// Serverless NEG referencing a Cloud Function
		"cloudFunction.function": {
			ToSDPItemType:    gcpshared.CloudFunctionsFunction,
			Description:      "If the Cloud Function is updated or deleted: Requests routed via the NEG may fail or change behavior. If the NEG changes: The Cloud Function remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
		},
	},
	terraformMapping: gcpshared.TerraformMapping{
		Reference: "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/compute_network_endpoint_group",
		Mappings: []*sdp.TerraformMapping{{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "google_compute_network_endpoint_group.name",
		}},
	},
}.Register()
