package adapters

import (
	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

// Run Revision adapter for Cloud Run revisions
var _ = registerableAdapter{
	sdpType: gcpshared.RunRevision,
	meta: gcpshared.AdapterMeta{
		/*
			A Revision is an immutable snapshot of code and configuration.
			A Revision references a container image.
			Revisions are only created by updates to its parent Service.
		*/
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_CONFIGURATION,
		Scope:              gcpshared.ScopeProject,
		// Reference: https://cloud.google.com/run/docs/reference/rest/v2/projects.locations.services.revisions/get
		// GET https://run.googleapis.com/v2/projects/{project}/locations/{location}/services/{service}/revisions/{revision}
		// IAM Perm: run.revisions.get
		GetEndpointBaseURLFunc: gcpshared.ProjectLevelEndpointFuncWithThreeQueries("https://run.googleapis.com/v2/projects/%s/locations/%s/services/%s/revisions/%s"),
		// Reference: https://cloud.google.com/run/docs/reference/rest/v2/projects.locations.services.revisions/list
		// GET https://run.googleapis.com/v2/projects/{project}/locations/{location}/services/{service}/revisions
		// IAM Perm: run.revisions.list
		SearchEndpointFunc:  gcpshared.ProjectLevelEndpointFuncWithTwoQueries("https://run.googleapis.com/v2/projects/%s/locations/%s/services/%s/revisions"),
		UniqueAttributeKeys: []string{"locations", "services", "revisions"},
		IAMPermissions:      []string{"run.revisions.get", "run.revisions.list"},
		PredefinedRole:      "roles/run.viewer",
	},
	blastPropagation: map[string]*gcpshared.Impact{
		"service": {
			ToSDPItemType:    gcpshared.RunService,
			Description:      "If the Run Service is deleted or updated: The Revision may lose its association or fail to run. If the Revision is updated: The service remains unaffected.",
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		"vpcAccess.networkInterfaces.network": {
			ToSDPItemType:    gcpshared.ComputeNetwork,
			Description:      "If the Compute Network is deleted or updated: The Revision may lose connectivity or fail to run as expected. If the Revision is updated: The network remains unaffected.",
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		"vpcAccess.networkInterfaces.subnetwork": {
			ToSDPItemType:    gcpshared.ComputeSubnetwork,
			Description:      "If the Compute Subnetwork is deleted or updated: The Revision may lose connectivity or fail to run as expected. If the Revision is updated: The subnetwork remains unaffected.",
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		"vpcAccess.connector": {
			ToSDPItemType:    gcpshared.VPCAccessConnector,
			Description:      "If the VPC Access Connector is deleted or updated: The Revision may lose connectivity or fail to run as expected. If the Revision is updated: The connector remains unaffected.",
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		"serviceAccount": gcpshared.IAMServiceAccountImpactInOnly,
		"containers.image": {
			ToSDPItemType:    gcpshared.ArtifactRegistryDockerImage,
			Description:      "If the Artifact Registry Docker Image is deleted or updated: The Revision may fail to pull the image. If the Revision is updated: The Docker image remains unaffected.",
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		"volumes.cloudSqlInstance.instances": {
			// Format: {project}:{location}:{instance}
			// The manual adapter linker handles this format automatically.
			ToSDPItemType:    gcpshared.SQLAdminInstance,
			Description:      "If the Cloud SQL Instance is deleted or updated: The Revision may fail to access the database. If the Revision is updated: The instance remains unaffected.",
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		"volumes.gcs.bucket": {
			ToSDPItemType:    gcpshared.StorageBucket,
			Description:      "If the Cloud Storage Bucket is deleted or updated: The Revision may fail to access the GCS volume. If the Revision is updated: The bucket remains unaffected.",
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		"volumes.secret.secret": {
			ToSDPItemType:    gcpshared.SecretManagerSecret,
			Description:      "If the Secret Manager Secret is deleted or updated: The Revision may fail to access sensitive data mounted as a volume. If the Revision is updated: The secret remains unaffected.",
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		"volumes.nfs.server": {
			ToSDPItemType:    stdlib.NetworkIP,
			Description:      "If the NFS server (IP address or hostname) becomes unavailable: The Revision may fail to mount the NFS volume. If the Revision is updated: The NFS server remains unaffected. The linker automatically detects whether the value is an IP address or DNS name.",
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		"logUri": {
			ToSDPItemType:    stdlib.NetworkHTTP,
			Description:      "If the log URI endpoint becomes unavailable: The Revision logs may not be accessible. If the Revision is updated: The log URI endpoint remains unaffected.",
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		"encryptionKey": gcpshared.CryptoKeyImpactInOnly,
	},
	terraformMapping: gcpshared.TerraformMapping{
		Description: "There is no terraform resource for this type.",
	},
}.Register()
