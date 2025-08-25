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
var dataprocClusterAdapter = registerableAdapter{ //nolint:unused
	sdpType: gcpshared.DataprocCluster,
	meta: gcpshared.AdapterMeta{
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_COMPUTE_APPLICATION,
		Scope:              gcpshared.ScopeRegional,
		GetEndpointBaseURLFunc: gcpshared.RegionalLevelEndpointFuncWithSingleQuery(
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
		NameSelector: "clusterName", // https://cloud.google.com/dataproc/docs/reference/rest/v1/projects.regions.clusters#resource:-cluster
		// TODO: https://linear.app/overmind/issue/ENG-631/investigate-how-we-can-add-health-status-for-supporting-items
		// https://cloud.google.com/dataproc/docs/reference/rest/v1/projects.regions.clusters#clusterstatus
	},
	blastPropagation: map[string]*gcpshared.Impact{
		"config.gceClusterConfig.networkUri":      gcpshared.ComputeNetworkImpactInOnly,
		"config.gceClusterConfig.subnetworkUri":   gcpshared.ComputeNetworkImpactInOnly,
		"config.gceClusterConfig.serviceAccount":  gcpshared.IAMServiceAccountImpactInOnly,
		"config.encryptionConfig.gcePdKmsKeyName": gcpshared.CryptoKeyImpactInOnly,
		"config.encryptionConfig.kmsKey":          gcpshared.CryptoKeyImpactInOnly,
		"config.masterConfig.imageUri": {
			ToSDPItemType:    gcpshared.ComputeImage,
			Description:      "If the Image is deleted or updated: The cluster may fail to create new nodes. If the cluster is updated: The existing nodes remain unaffected.",
			BlastPropagation: &sdp.BlastPropagation{In: true},
		},
		"config.masterConfig.machineTypeUri": {
			ToSDPItemType:    gcpshared.ComputeMachineType,
			Description:      "If the Machine Type is deleted or updated: The cluster may fail to create new nodes. If the cluster is updated: The existing nodes remain unaffected.",
			BlastPropagation: &sdp.BlastPropagation{In: true},
		},
		"config.masterConfig.managedGroupConfig.instanceGroupManagerUri": {
			ToSDPItemType:    gcpshared.ComputeInstanceGroupManager,
			Description:      "If the Instance Group Manager is deleted or updated: The cluster may fail to create new nodes. If the cluster is updated: The existing nodes remain unaffected.",
			BlastPropagation: &sdp.BlastPropagation{In: true},
		},
		"config.masterConfig.accelerators": {
			ToSDPItemType:    gcpshared.ComputeAcceleratorType,
			Description:      "If the Accelerator Type is deleted or updated: The cluster may fail to create new nodes. If the cluster is updated: The existing nodes remain unaffected.",
			BlastPropagation: &sdp.BlastPropagation{In: true},
		},
		"config.workerConfig.imageUri": {
			ToSDPItemType:    gcpshared.ComputeImage,
			Description:      "If the Image is deleted or updated: The cluster may fail to create new nodes. If the cluster is updated: The existing nodes remain unaffected.",
			BlastPropagation: &sdp.BlastPropagation{In: true},
		},
		"config.workerConfig.machineTypeUri": {
			ToSDPItemType:    gcpshared.ComputeMachineType,
			Description:      "If the Machine Type is deleted or updated: The cluster may fail to create new nodes. If the cluster is updated: The existing nodes remain unaffected.",
			BlastPropagation: &sdp.BlastPropagation{In: true},
		},
		"config.workerConfig.managedGroupConfig.instanceGroupManagerUri": {
			ToSDPItemType:    gcpshared.ComputeInstanceGroupManager,
			Description:      "If the Instance Group Manager is deleted or updated: The cluster may fail to create new nodes. If the cluster is updated: The existing nodes remain unaffected.",
			BlastPropagation: &sdp.BlastPropagation{In: true},
		},
		"config.workerConfig.accelerators": {
			ToSDPItemType:    gcpshared.ComputeAcceleratorType,
			Description:      "If the Accelerator Type is deleted or updated: The cluster may fail to create new nodes. If the cluster is updated: The existing nodes remain unaffected.",
			BlastPropagation: &sdp.BlastPropagation{In: true},
		},
		"config.secondaryWorkerConfig.imageUri": {
			ToSDPItemType:    gcpshared.ComputeImage,
			Description:      "If the Image is deleted or updated: The cluster may fail to create new nodes. If the cluster is updated: The existing nodes remain unaffected.",
			BlastPropagation: &sdp.BlastPropagation{In: true},
		},
		"config.secondaryWorkerConfig.machineTypeUri": {
			ToSDPItemType:    gcpshared.ComputeMachineType,
			Description:      "If the Machine Type is deleted or updated: The cluster may fail to create new nodes. If the cluster is updated: The existing nodes remain unaffected.",
			BlastPropagation: &sdp.BlastPropagation{In: true},
		},
		"config.secondaryWorkerConfig.managedGroupConfig.instanceGroupManagerUri": {
			ToSDPItemType:    gcpshared.ComputeInstanceGroupManager,
			Description:      "If the Instance Group Manager is deleted or updated: The cluster may fail to create new nodes. If the cluster is updated: The existing nodes remain unaffected.",
			BlastPropagation: &sdp.BlastPropagation{In: true},
		},
		"config.secondaryWorkerConfig.accelerators": {
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
