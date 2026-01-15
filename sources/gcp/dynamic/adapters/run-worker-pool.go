package adapters

import (
	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

// Cloud Run Worker Pool:
// Reference: https://cloud.google.com/run/docs/reference/rest/v2/projects.locations.workerPools/get
// GET:  https://run.googleapis.com/v2/projects/{project}/locations/{location}/workerPools/{workerPool}
// LIST: https://run.googleapis.com/v2/projects/{project}/locations/{location}/workerPools
var _ = registerableAdapter{
	sdpType: gcpshared.RunWorkerPool,
	meta: gcpshared.AdapterMeta{
		InDevelopment:      true,
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_CONFIGURATION,
		Scope:              gcpshared.ScopeProject,
		GetEndpointBaseURLFunc: gcpshared.ProjectLevelEndpointFuncWithTwoQueries(
			"https://run.googleapis.com/v2/projects/%s/locations/%s/workerPools/%s",
		),
		// The list endpoint requires the location only.
		SearchEndpointFunc: gcpshared.ProjectLevelEndpointFuncWithSingleQuery(
			"https://run.googleapis.com/v2/projects/%s/locations/%s/workerPools",
		),
		// location|workerPool
		UniqueAttributeKeys: []string{"locations", "workerPools"},
		IAMPermissions: []string{
			"run.workerPools.get",
			"run.workerPools.list",
		},
		PredefinedRole: "roles/run.viewer",
	},
	blastPropagation: map[string]*gcpshared.Impact{
		// Service account used by revisions in the worker pool
		"template.serviceAccount": gcpshared.IAMServiceAccountImpactInOnly,
		// Encryption key for image encryption
		"template.encryptionKey": gcpshared.CryptoKeyImpactInOnly,
		// VPC Access Connector for network connectivity
		"template.vpcAccess.connector": {
			ToSDPItemType:    gcpshared.VPCAccessConnector,
			Description:      "If the VPC Access Connector is deleted or updated: The worker pool may lose connectivity or fail to route traffic correctly. If the worker pool is updated: The connector remains unaffected.",
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		// VPC Network for direct VPC egress
		"template.vpcAccess.networkInterfaces.network": {
			ToSDPItemType:    gcpshared.ComputeNetwork,
			Description:      "If the Compute Network is deleted or updated: The worker pool may lose connectivity or fail to route traffic correctly. If the worker pool is updated: The network remains unaffected.",
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		// VPC Subnetwork for direct VPC egress
		"template.vpcAccess.networkInterfaces.subnetwork": {
			ToSDPItemType:    gcpshared.ComputeSubnetwork,
			Description:      "If the Compute Subnetwork is deleted or updated: The worker pool may lose connectivity or fail to route traffic correctly. If the worker pool is updated: The subnetwork remains unaffected.",
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		// Service Mesh for advanced networking
		"template.serviceMesh.mesh": {
			ToSDPItemType:    gcpshared.NetworkServicesMesh,
			Description:      "If the Network Services Mesh is deleted or updated: The worker pool may lose service mesh connectivity or fail to communicate with other mesh services. If the worker pool is updated: The mesh remains unaffected.",
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		// Secret Manager secrets mounted as volumes
		"template.volumes.secret.secret": {
			ToSDPItemType:    gcpshared.SecretManagerSecret,
			Description:      "If the Secret Manager Secret is deleted or updated: The worker pool may fail to access sensitive data mounted as volumes. If the worker pool is updated: The secret remains unaffected.",
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		// Cloud SQL instances mounted as volumes
		"template.volumes.cloudSqlInstance.instances": {
			ToSDPItemType:    gcpshared.SQLAdminInstance,
			Description:      "If the Cloud SQL Instance is deleted or updated: The worker pool may fail to access the database. If the worker pool is updated: The instance remains unaffected.",
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		// GCS buckets mounted as volumes
		"template.volumes.gcs.bucket": {
			ToSDPItemType:    gcpshared.StorageBucket,
			Description:      "If the Cloud Storage Bucket is deleted or updated: The worker pool may fail to access stored data. If the worker pool is updated: The bucket remains unaffected.",
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		// NFS server (IP address or DNS name) - auto-detected by linker
		"template.volumes.nfs.server": {
			ToSDPItemType:    stdlib.NetworkIP,
			Description:      "If the NFS server (IP address or hostname) becomes unavailable: The worker pool may fail to mount the NFS volume. If the worker pool is updated: The NFS server remains unaffected. The linker automatically detects whether the value is an IP address or DNS name.",
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		// Secret Manager secrets used in environment variables
		"template.containers.env.valueSource.secretKeyRef.secret": {
			ToSDPItemType:    gcpshared.SecretManagerSecret,
			Description:      "If the referenced Secret Manager Secret is deleted or updated: The container may fail to start or access sensitive configuration. If the worker pool is updated: The secret remains unaffected.",
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		// Binary Authorization policy
		"binaryAuthorization.policy": {
			ToSDPItemType:    gcpshared.BinaryAuthorizationPlatformPolicy,
			Description:      "If the Binary Authorization policy is deleted or updated: The worker pool may fail to deploy new revisions if they don't meet policy requirements. If the worker pool is updated: The policy remains unaffected.",
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		// Latest ready revision - child resource
		"latestReadyRevision": {
			ToSDPItemType:    gcpshared.RunRevision,
			Description:      "If the Cloud Run Worker Pool is deleted or updated: Associated revisions may become orphaned or be deleted. If revisions are updated: The worker pool status may reflect the changes.",
			BlastPropagation: &sdp.BlastPropagation{Out: true},
		},
		// Latest created revision - child resource
		"latestCreatedRevision": {
			ToSDPItemType:    gcpshared.RunRevision,
			Description:      "If the Cloud Run Worker Pool is deleted or updated: Associated revisions may become orphaned or be deleted. If revisions are updated: The worker pool status may reflect the changes.",
			BlastPropagation: &sdp.BlastPropagation{Out: true},
		},
		// Instance split revisions - child resources
		"instanceSplits.revision": {
			ToSDPItemType:    gcpshared.RunRevision,
			Description:      "If the Cloud Run Worker Pool is deleted or updated: Associated revisions may become orphaned or be deleted. If revisions are updated: The worker pool status may reflect the changes.",
			BlastPropagation: &sdp.BlastPropagation{Out: true},
		},
		// Forward link from parent to child via SEARCH - discover all revisions in this worker pool
		"name": {
			ToSDPItemType: gcpshared.RunRevision,
			Description:   "If the Cloud Run Worker Pool is deleted or updated: All associated Revisions may become invalid or inaccessible. If a Revision is updated: The worker pool remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{
				In:  false,
				Out: true,
			},
			IsParentToChild: true,
		},
	},
}.Register()
