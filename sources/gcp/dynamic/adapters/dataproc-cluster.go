package adapters

import (
	"github.com/overmindtech/cli/go/sdp-go"
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
		GetEndpointFunc: gcpshared.RegionalLevelEndpointFunc(
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
	linkRules: map[string]*gcpshared.Impact{
		"config.gceClusterConfig.networkUri":      gcpshared.ComputeNetworkImpactInOnly,
		"config.gceClusterConfig.subnetworkUri":   gcpshared.ComputeSubnetworkImpactInOnly,
		"config.gceClusterConfig.serviceAccount":  gcpshared.IAMServiceAccountImpactInOnly,
		"config.encryptionConfig.gcePdKmsKeyName": gcpshared.CryptoKeyImpactInOnly,
		"config.encryptionConfig.kmsKey":          gcpshared.CryptoKeyImpactInOnly,
		"config.masterConfig.imageUri": {
			ToSDPItemType:    gcpshared.ComputeImage,
			Description:      "If the Image is deleted or updated: The cluster may fail to create new nodes. If the cluster is updated: The existing nodes remain unaffected.",
		},
		"config.masterConfig.managedGroupConfig.instanceGroupManagerUri": {
			ToSDPItemType:    gcpshared.ComputeInstanceGroupManager,
			Description:      "If the Instance Group Manager is deleted or updated: The cluster may fail to create new nodes. If the cluster is updated: The existing nodes remain unaffected.",
		},
		"config.masterConfig.accelerators.acceleratorTypeUri": {
			ToSDPItemType:    gcpshared.ComputeAcceleratorType,
			Description:      "If the Accelerator Type is deleted or updated: The cluster may fail to create new nodes. If the cluster is updated: The existing nodes remain unaffected.",
		},
		"config.workerConfig.imageUri": {
			ToSDPItemType:    gcpshared.ComputeImage,
			Description:      "If the Image is deleted or updated: The cluster may fail to create new nodes. If the cluster is updated: The existing nodes remain unaffected.",
		},
		"config.workerConfig.managedGroupConfig.instanceGroupManagerUri": {
			ToSDPItemType:    gcpshared.ComputeInstanceGroupManager,
			Description:      "If the Instance Group Manager is deleted or updated: The cluster may fail to create new nodes. If the cluster is updated: The existing nodes remain unaffected.",
		},
		"config.workerConfig.accelerators.acceleratorTypeUri": {
			ToSDPItemType:    gcpshared.ComputeAcceleratorType,
			Description:      "If the Accelerator Type is deleted or updated: The cluster may fail to create new nodes. If the cluster is updated: The existing nodes remain unaffected.",
		},
		"config.secondaryWorkerConfig.imageUri": {
			ToSDPItemType:    gcpshared.ComputeImage,
			Description:      "If the Image is deleted or updated: The cluster may fail to create new nodes. If the cluster is updated: The existing nodes remain unaffected.",
		},
		"config.secondaryWorkerConfig.managedGroupConfig.instanceGroupManagerUri": {
			ToSDPItemType:    gcpshared.ComputeInstanceGroupManager,
			Description:      "If the Instance Group Manager is deleted or updated: The cluster may fail to create new nodes. If the cluster is updated: The existing nodes remain unaffected.",
		},
		"config.secondaryWorkerConfig.accelerators.acceleratorTypeUri": {
			ToSDPItemType:    gcpshared.ComputeAcceleratorType,
			Description:      "If the Accelerator Type is deleted or updated: The cluster may fail to create new nodes. If the cluster is updated: The existing nodes remain unaffected.",
		},
		"config.autoscalingConfig.policyUri": {
			ToSDPItemType:    gcpshared.DataprocAutoscalingPolicy,
			Description:      "If the Autoscaling Policy is deleted or updated: The cluster may fail to scale. If the cluster is updated: The existing nodes remain unaffected.",
		},
		"config.auxiliaryNodeGroups.nodeGroup.name": {
			ToSDPItemType:    gcpshared.ComputeNodeGroup,
			Description:      "If the Node Group is deleted or updated: The cluster may fail to create new nodes. If the cluster is updated: The existing nodes remain unaffected.",
		},
		"config.tempBucket": {
			ToSDPItemType:    gcpshared.StorageBucket,
			Description:      "If the Storage Bucket is deleted or updated: The cluster may fail to stage data or logs. If the cluster is updated: The bucket remains unaffected.",
		},
		"config.stagingBucket": {
			ToSDPItemType:    gcpshared.StorageBucket,
			Description:      "If the Storage Bucket is deleted or updated: The cluster may fail to stage job dependencies, configuration files, or job driver console output. If the cluster is updated: The bucket remains unaffected.",
		},
		"config.metastoreConfig.dataprocMetastoreService": {
			ToSDPItemType:    gcpshared.DataprocMetastoreService,
			Description:      "If the Dataproc Metastore Service is deleted or updated: The cluster may lose access to centralized metadata or fail to operate correctly. If the cluster is updated: The metastore service remains unaffected.",
		},
		"virtualClusterConfig.kubernetesClusterConfig.gkeClusterConfig.gkeClusterTarget": {
			ToSDPItemType:    gcpshared.ContainerCluster,
			Description:      "If the GKE Cluster is deleted or updated: The Dataproc virtual cluster may become invalid or inaccessible. If the Dataproc cluster is updated: The GKE cluster remains unaffected.",
		},
		"virtualClusterConfig.kubernetesClusterConfig.gkeClusterConfig.nodePoolTarget.nodePool": {
			ToSDPItemType:    gcpshared.ContainerNodePool,
			Description:      "If the GKE Node Pool is deleted or updated: The Dataproc virtual cluster may fail to schedule workloads or lose capacity. If the Dataproc cluster is updated: The node pool remains unaffected.",
		},
		"virtualClusterConfig.stagingBucket": {
			ToSDPItemType:    gcpshared.StorageBucket,
			Description:      "If the Storage Bucket is deleted or updated: The virtual cluster may fail to stage job dependencies, configuration files, or job driver console output. If the cluster is updated: The bucket remains unaffected.",
		},
		"virtualClusterConfig.auxiliaryServicesConfig.sparkHistoryServerConfig.dataprocCluster": {
			ToSDPItemType:    gcpshared.DataprocCluster,
			Description:      "If the Spark History Server Dataproc Cluster is deleted or updated: The cluster may lose access to Spark job history or fail to monitor Spark applications. If the cluster is updated: The Spark History Server cluster remains unaffected.",
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
