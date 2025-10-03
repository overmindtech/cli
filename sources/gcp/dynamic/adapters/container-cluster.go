package adapters

import (
	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

// GKE Container Cluster represents a managed Kubernetes cluster in GCP.
// It provides a scalable, secure, and fully managed Kubernetes service for running containerized applications.
//
// Adapter for GCP GKE Container Cluster
// API Get: https://cloud.google.com/kubernetes-engine/docs/reference/rest/v1/projects.locations.clusters/get
// API List: https://cloud.google.com/kubernetes-engine/docs/reference/rest/v1/projects.locations.clusters/list
var _ = registerableAdapter{
	sdpType: gcpshared.ContainerCluster,
	meta: gcpshared.AdapterMeta{
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_COMPUTE_APPLICATION,
		Scope:              gcpshared.ScopeProject,
		// GET https://container.googleapis.com/v1/projects/{project}/locations/{location}/clusters/{cluster}
		GetEndpointBaseURLFunc: gcpshared.ProjectLevelEndpointFuncWithTwoQueries(
			"https://container.googleapis.com/v1/projects/%s/locations/%s/clusters/%s",
		),
		// LIST https://container.googleapis.com/v1/projects/{project}/locations/{location}/clusters
		SearchEndpointFunc: gcpshared.ProjectLevelEndpointFuncWithSingleQuery(
			"https://container.googleapis.com/v1/projects/%s/locations/%s/clusters",
		),
		SearchDescription:   "Search for GKE clusters in a location. Use the format \"location\" or the full resource name supported for terraform mappings.",
		UniqueAttributeKeys: []string{"locations", "clusters"},
		IAMPermissions: []string{
			"container.clusters.get",
			"container.clusters.list",
		},
		PredefinedRole: "roles/container.viewer",
	},
	blastPropagation: map[string]*gcpshared.Impact{
		"network":                  gcpshared.ComputeNetworkImpactInOnly,
		"subnetwork":               gcpshared.ComputeNetworkImpactInOnly,
		"nodePools.serviceAccount": gcpshared.IAMServiceAccountImpactInOnly,
		"nodePools.bootDiskKmsKey": gcpshared.CryptoKeyImpactInOnly,
		"nodePools.nodeGroup": {
			ToSDPItemType:    gcpshared.ComputeNodeGroup,
			Description:      "If the referenced Node Group is deleted or updated: Node pools backed by it may fail to create or manage nodes. Updates to the node pool will not affect the node group.",
			BlastPropagation: &sdp.BlastPropagation{In: true},
		},
		"notificationConfig.pubsub.topic": {
			ToSDPItemType:    gcpshared.PubSubTopic,
			Description:      "If the referenced Pub/Sub topic is deleted or updated: Notifications may fail to be sent. Updates to the cluster will not affect the topic.",
			BlastPropagation: &sdp.BlastPropagation{In: true},
		},
		"userManagedKeysConfig.serviceAccountSigningKeys":      gcpshared.CryptoKeyVersionImpactInOnly,
		"userManagedKeysConfig.serviceAccountVerificationKeys": gcpshared.CryptoKeyVersionImpactInOnly,
		"userManagedKeysConfig.controlPlaneDiskEncryptionKey":  gcpshared.CryptoKeyImpactInOnly,
		"userManagedKeysConfig.gkeopsEtcdBackupEncryptionKey":  gcpshared.CryptoKeyImpactInOnly,
	},
	terraformMapping: gcpshared.TerraformMapping{
		Reference:   "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/container_cluster",
		Description: "id => projects/{{project}}/locations/{{zone}}/clusters/{{name}}",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_SEARCH,
				TerraformQueryMap: "google_container_cluster.id",
			},
		},
	},
}.Register()
