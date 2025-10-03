package adapters

import (
	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

// Global Forwarding Rule (project-level) resource.
// Reference: https://cloud.google.com/compute/docs/reference/rest/v1/globalForwardingRules/get
// GET:  https://compute.googleapis.com/compute/v1/projects/{project}/global/forwardingRules/{forwardingRule}
// LIST: https://compute.googleapis.com/compute/v1/projects/{project}/global/forwardingRules
var computeGlobalForwardingRuleAdapter = registerableAdapter{ //nolint:unused
	sdpType: gcpshared.ComputeGlobalForwardingRule,
	meta: gcpshared.AdapterMeta{
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
		Scope:              gcpshared.ScopeProject,
		GetEndpointBaseURLFunc: gcpshared.ProjectLevelEndpointFuncWithSingleQuery(
			"https://compute.googleapis.com/compute/v1/projects/%s/global/forwardingRules/%s",
		),
		ListEndpointFunc: gcpshared.ProjectLevelListFunc(
			"https://compute.googleapis.com/compute/v1/projects/%s/global/forwardingRules",
		),
		// Path segment used for lookups: forwardingRules
		UniqueAttributeKeys: []string{"forwardingRules"},
		IAMPermissions: []string{
			// Same permission set as regional forwarding rules
			"compute.forwardingRules.get",
			"compute.forwardingRules.list",
		},
		PredefinedRole: "roles/compute.viewer",
		// HEALTH: https://cloud.google.com/compute/docs/reference/rest/v1/globalForwardingRules#Status => pscConnectionStatus
		// TODO: https://linear.app/overmind/issue/ENG-631/investigate-how-we-can-add-health-status-for-supporting-items
	},
	blastPropagation: map[string]*gcpshared.Impact{
		// Network reference (global). If the network is changed it may impact the forwarding rule; forwarding rule updates don't impact the network.
		"network":    gcpshared.ComputeNetworkImpactInOnly,
		"subnetwork": gcpshared.ComputeSubnetworkImpactInOnly,
		// IP address assigned to the forwarding rule (may be ephemeral or static).
		"IPAddress": gcpshared.IPImpactBothWays,
		// Backend service (global) tightly coupled for traffic delivery.
		"backendService": {
			ToSDPItemType: gcpshared.ComputeBackendService,
			Description:   "If the Backend Service is updated or deleted: The forwarding rule routing behavior changes or breaks. If the forwarding rule is updated or deleted: Traffic will stop or be re-routed affecting the backend service load.",
			BlastPropagation: &sdp.BlastPropagation{
				In:  true,
				Out: true,
			},
		},
	},
	terraformMapping: gcpshared.TerraformMapping{
		Reference: "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/compute_global_forwarding_rule",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_GET,
				TerraformQueryMap: "google_compute_global_forwarding_rule.name",
			},
		},
	},
}.Register()
