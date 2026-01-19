package adapters

import (
	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

// Dataproc Cluster adapter
// API Get:  https://cloud.google.com/dataproc/docs/reference/rest/v1/projects.regions.clusters/get
// API List: https://cloud.google.com/dataproc/docs/reference/rest/v1/projects.regions.clusters/list
// GET  https://dataproc.googleapis.com/v1/projects/{project}/regions/{region}/clusters/{cluster}
// LIST https://dataproc.googleapis.com/v1/projects/{project}/regions/{region}/clusters
var _ = registerableAdapter{
	sdpType: gcpshared.DataprocCluster,
	meta: gcpshared.AdapterMeta{
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_COMPUTE_APPLICATION,
		LocationLevel:      gcpshared.RegionalLevel,
		GetEndpointFunc: gcpshared.RegionalLevelEndpointFuncWithSingleQuery(
			"https://dataproc.googleapis.com/v1/projects/%s/regions/%s/clusters/%s",
		),
		ListEndpointFunc: gcpshared.RegionLevelListFunc(
			"https://dataproc.googleapis.com/v1/projects/%s/regions/%s/clusters",
		),
		UniqueAttributeKeys: []string{"clusters"},
		IAMPermissions: []string{
			"dataproc.clusters.get",
			"dataproc.clusters.list",
		},
		PredefinedRole: "roles/dataproc.viewer",
		NameSelector:   "clusterName", // https://cloud.google.com/dataproc/docs/reference/rest/v1/projects.regions.clusters#resource:-cluster
		// TODO: https://linear.app/overmind/issue/ENG-631/investigate-how-we-can-add-health-status-for-supporting-items
		// https://cloud.google.com/dataproc/docs/reference/rest/v1/projects.regions.clusters#clusterstatus
	},
	blastPropagation: map[string]*gcpshared.Impact{
		"config.gceClusterConfig.networkUri":      gcpshared.ComputeNetworkImpactInOnly,
		"config.gceClusterConfig.subnetworkUri":   gcpshared.ComputeSubnetworkImpactInOnly,
		"config.gceClusterConfig.serviceAccount":  gcpshared.IAMServiceAccountImpactInOnly,
		"config.encryptionConfig.gcePdKmsKeyName": gcpshared.CryptoKeyImpactInOnly,
		"config.encryptionConfig.kmsKey":          gcpshared.CryptoKeyImpactInOnly,
		"config.masterConfig.imageUri": {
			ToSDPItemType:    gcpshared.ComputeImage,
			Description:      "If the Image is deleted or updated: The cluster may fail to create new nodes. If the cluster is updated: The existing nodes remain unaffected.",
			BlastPropagation: &sdp.BlastPropagation{In: true},
		},
		"config.masterConfig.managedGroupConfig.instanceGroupManagerUri": {
			ToSDPItemType:    gcpshared.ComputeInstanceGroupManager,
			Description:      "If the Instance Group Manager is deleted or updated: The cluster may fail to create new nodes. If the cluster is updated: The existing nodes remain unaffected.",
			BlastPropagation: &sdp.BlastPropagation{In: true},
		},
		"config.masterConfig.accelerators.acceleratorTypeUri": {
			ToSDPItemType:    gcpshared.ComputeAcceleratorType,
			Description:      "If the Accelerator Type is deleted or updated: The cluster may fail to create new nodes. If the cluster is updated: The existing nodes remain unaffected.",
			BlastPropagation: &sdp.BlastPropagation{In: true},
		},
		"config.workerConfig.imageUri": {
			ToSDPItemType:    gcpshared.ComputeImage,
			Description:      "If the Image is deleted or updated: The cluster may fail to create new nodes. If the cluster is updated: The existing nodes remain unaffected.",
			BlastPropagation: &sdp.BlastPropagation{In: true},
		},
		"config.workerConfig.managedGroupConfig.instanceGroupManagerUri": {
			ToSDPItemType:    gcpshared.ComputeInstanceGroupManager,
			Description:      "If the Instance Group Manager is deleted or updated: The cluster may fail to create new nodes. If the cluster is updated: The existing nodes remain unaffected.",
			BlastPropagation: &sdp.BlastPropagation{In: true},
		},
		"config.workerConfig.accelerators.acceleratorTypeUri": {
			ToSDPItemType:    gcpshared.ComputeAcceleratorType,
			Description:      "If the Accelerator Type is deleted or updated: The cluster may fail to create new nodes. If the cluster is updated: The existing nodes remain unaffected.",
			BlastPropagation: &sdp.BlastPropagation{In: true},
		},
		"config.secondaryWorkerConfig.imageUri": {
			ToSDPItemType:    gcpshared.ComputeImage,
			Description:      "If the Image is deleted or updated: The cluster may fail to create new nodes. If the cluster is updated: The existing nodes remain unaffected.",
			BlastPropagation: &sdp.BlastPropagation{In: true},
		},
		"config.secondaryWorkerConfig.managedGroupConfig.instanceGroupManagerUri": {
			ToSDPItemType:    gcpshared.ComputeInstanceGroupManager,
			Description:      "If the Instance Group Manager is deleted or updated: The cluster may fail to create new nodes. If the cluster is updated: The existing nodes remain unaffected.",
			BlastPropagation: &sdp.BlastPropagation{In: true},
		},
		"config.secondaryWorkerConfig.accelerators.acceleratorTypeUri": {
			ToSDPItemType:    gcpshared.ComputeAcceleratorType,
			Description:      "If the Accelerator Type is deleted or updated: The cluster may fail to create new nodes. If the cluster is updated: The existing nodes remain unaffected.",
			BlastPropagation: &sdp.BlastPropagation{In: true},
		},
		"config.autoscalingConfig.policyUri": {
			ToSDPItemType:    gcpshared.DataprocAutoscalingPolicy,
			Description:      "If the Autoscaling Policy is deleted or updated: The cluster may fail to scale. If the cluster is updated: The existing nodes remain unaffected.",
			BlastPropagation: &sdp.BlastPropagation{In: true},
		},
		"config.auxiliaryNodeGroups.nodeGroup.name": {
			ToSDPItemType:    gcpshared.ComputeNodeGroup,
			Description:      "If the Node Group is deleted or updated: The cluster may fail to create new nodes. If the cluster is updated: The existing nodes remain unaffected.",
			BlastPropagation: &sdp.BlastPropagation{In: true},
		},
		"config.tempBucket": {
			ToSDPItemType:    gcpshared.StorageBucket,
			Description:      "If the Storage Bucket is deleted or updated: The cluster may fail to stage data or logs. If the cluster is updated: The bucket remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{In: true},
		},
		"config.stagingBucket": {
			ToSDPItemType:    gcpshared.StorageBucket,
			Description:      "If the Storage Bucket is deleted or updated: The cluster may fail to stage job dependencies, configuration files, or job driver console output. If the cluster is updated: The bucket remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{In: true},
		},
		"config.metastoreConfig.dataprocMetastoreService": {
			ToSDPItemType:    gcpshared.DataprocMetastoreService,
			Description:      "If the Dataproc Metastore Service is deleted or updated: The cluster may lose access to centralized metadata or fail to operate correctly. If the cluster is updated: The metastore service remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{In: true},
		},
		"virtualClusterConfig.kubernetesClusterConfig.gkeClusterConfig.gkeClusterTarget": {
			ToSDPItemType:    gcpshared.ContainerCluster,
			Description:      "If the GKE Cluster is deleted or updated: The Dataproc virtual cluster may become invalid or inaccessible. If the Dataproc cluster is updated: The GKE cluster remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{In: true},
		},
		"virtualClusterConfig.kubernetesClusterConfig.gkeClusterConfig.nodePoolTarget.nodePool": {
			ToSDPItemType:    gcpshared.ContainerNodePool,
			Description:      "If the GKE Node Pool is deleted or updated: The Dataproc virtual cluster may fail to schedule workloads or lose capacity. If the Dataproc cluster is updated: The node pool remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{In: true},
		},
		"virtualClusterConfig.stagingBucket": {
			ToSDPItemType:    gcpshared.StorageBucket,
			Description:      "If the Storage Bucket is deleted or updated: The virtual cluster may fail to stage job dependencies, configuration files, or job driver console output. If the cluster is updated: The bucket remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{In: true},
		},
		"virtualClusterConfig.auxiliaryServicesConfig.sparkHistoryServerConfig.dataprocCluster": {
			ToSDPItemType:    gcpshared.DataprocCluster,
			Description:      "If the Spark History Server Dataproc Cluster is deleted or updated: The cluster may lose access to Spark job history or fail to monitor Spark applications. If the cluster is updated: The Spark History Server cluster remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{In: true},
		},
	},
	terraformMapping: gcpshared.TerraformMapping{
		Reference:   "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/dataproc_cluster",
		Description: "Use GET by cluster name.",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_GET,
				TerraformQueryMap: "google_dataproc_cluster.name",
			},
		},
	},
}.Register()
