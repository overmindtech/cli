package adapters

import (
	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

// GKE Container Node Pool adapter.
// GCP Ref:
//   - API Call structure (GET): https://cloud.google.com/kubernetes-engine/docs/reference/rest/v1/projects.locations.clusters.nodePools/get
//     GET https://container.googleapis.com/v1/projects/{project}/locations/{location}/clusters/{cluster}/nodePools/{node_pool}
//   - Type Definition (NodePool): https://cloud.google.com/kubernetes-engine/docs/reference/rest/v1/projects.locations.clusters.nodePools#NodePool
//   - LIST (per-cluster): https://cloud.google.com/kubernetes-engine/docs/reference/rest/v1/projects.locations.clusters.nodePools/list
//
// Scope: Project-level (uses locations path parameter; unique attributes include location+cluster+nodePool).
var containerNodePoolAdapter = registerableAdapter{ //nolint:unused
	sdpType: gcpshared.ContainerNodePool,
	meta: gcpshared.AdapterMeta{
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_COMPUTE_APPLICATION,
		Scope:              gcpshared.ScopeProject,
		GetEndpointBaseURLFunc: gcpshared.ProjectLevelEndpointFuncWithThreeQueries(
			"https://container.googleapis.com/v1/projects/%s/locations/%s/clusters/%s/nodePools/%s",
		),
		// Listing node pools requires location and cluster, so we support Search rather than List.
		SearchEndpointFunc: gcpshared.ProjectLevelEndpointFuncWithTwoQueries(
			"https://container.googleapis.com/v1/projects/%s/locations/%s/clusters/%s/nodePools",
		),
		SearchDescription:   "Search GKE Node Pools within a cluster. Use \"[location]|[cluster]\" or the full resource name supported by Terraform mappings: \"[project]/[location]/[cluster]/[node_pool_name]\"",
		UniqueAttributeKeys: []string{"locations", "clusters", "nodePools"},
		IAMPermissions: []string{
			"container.clusters.get",
			"container.clusters.list",
		},
		PredefinedRole: "roles/container.viewer",
		// TODO: https://linear.app/overmind/issue/ENG-631/investigate-how-we-can-add-health-status-for-supporting-items
		// https://cloud.google.com/kubernetes-engine/docs/reference/rest/v1/projects.locations.clusters.nodePools#NodePool.Status
	},
	blastPropagation: map[string]*gcpshared.Impact{
		"config.bootDiskKmsKey": gcpshared.CryptoKeyImpactInOnly,
		"config.serviceAccount": gcpshared.IAMServiceAccountImpactInOnly,
		"config.nodeGroup": {
			ToSDPItemType:    gcpshared.ComputeNodeGroup,
			Description:      "If the node pool is backed by a node group, then changes to the node group may affect the node pool. Changes to the node pool will not affect the node group.",
			BlastPropagation: &sdp.BlastPropagation{In: true},
		},
	},
	terraformMapping: gcpshared.TerraformMapping{
		Reference:   "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/container_node_pool",
		Description: "id => {project}/{location}/{cluster}/{node_pool_name}",
		// TODO: https://linear.app/overmind/issue/ENG-1258/support-terraform-mapping-for-queries-without-keys
		// There is no code change required for he adapter itself, just the framework to support
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_SEARCH,
				TerraformQueryMap: "google_container_node_pool.id",
			},
		},
	},
}.Register()
