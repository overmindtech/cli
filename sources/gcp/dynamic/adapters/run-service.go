package adapters

import (
	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

// Cloud Run Service adapter - Manages stateless containerized applications with automatic scaling
// Reference: https://cloud.google.com/run/docs/reference/rest/v2/projects.locations.services/get
// GET:  https://run.googleapis.com/v2/projects/{project}/locations/{location}/services/{service}
// LIST: https://run.googleapis.com/v2/projects/{project}/locations/{location}/services
var _ = registerableAdapter{
	sdpType: gcpshared.RunService,
	meta: gcpshared.AdapterMeta{
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_COMPUTE_APPLICATION,
		LocationLevel:      gcpshared.ProjectLevel,
		GetEndpointFunc: gcpshared.ProjectLevelEndpointFuncWithTwoQueries(
			"https://run.googleapis.com/v2/projects/%s/locations/%s/services/%s",
		),
		// List requires location parameter, so use Search
		SearchEndpointFunc: gcpshared.ProjectLevelEndpointFuncWithSingleQuery(
			"https://run.googleapis.com/v2/projects/%s/locations/%s/services",
		),
		UniqueAttributeKeys: []string{"locations", "services"},
		IAMPermissions: []string{
			"run.services.get",
			"run.services.list",
		},
		PredefinedRole: "roles/run.viewer",
		// TODO: https://linear.app/overmind/issue/ENG-631 - status field for health monitoring
	},
	blastPropagation: map[string]*gcpshared.Impact{
		"template.serviceAccount": gcpshared.IAMServiceAccountImpactInOnly,
		"template.vpcAccess.connector": {
			ToSDPItemType:    gcpshared.VPCAccessConnector,
			Description:      "If the VPC Access Connector is deleted or updated: The service may lose connectivity or fail to route traffic correctly. If the service is updated: The connector remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{In: true},
		},
		"template.vpcAccess.networkInterfaces.network": {
			ToSDPItemType:    gcpshared.ComputeNetwork,
			Description:      "If the Compute Network is deleted or updated: The service may lose connectivity or fail to route traffic correctly. If the service is updated: The network remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{In: true},
		},
		"template.vpcAccess.networkInterfaces.subnetwork": {
			ToSDPItemType:    gcpshared.ComputeSubnetwork,
			Description:      "If the Compute Subnetwork is deleted or updated: The service may lose connectivity or fail to route traffic correctly. If the service is updated: The subnetwork remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{In: true},
		},
		"template.containers.image": {
			ToSDPItemType:    gcpshared.ArtifactRegistryDockerImage,
			Description:      "If the Artifact Registry Docker Image is deleted or updated: The service may fail to deploy new revisions. If the service is updated: The Docker image remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{In: true},
		},
		"template.containers.env.valueSource.secretKeyRef.secret": {
			ToSDPItemType:    gcpshared.SecretManagerSecret,
			Description:      "If the referenced Secret Manager Secret is deleted or updated: the container may fail to start or access sensitive configuration. If the service is updated: the secret remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{In: true},
		},
		"template.volumes.secret.secret": {
			ToSDPItemType:    gcpshared.SecretManagerSecret,
			Description:      "If the Secret Manager Secret is deleted or updated: The service may fail to access sensitive data. If the service is updated: The secret remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{In: true},
		},
		"template.volumes.cloudSqlInstance.instances": {
			ToSDPItemType:    gcpshared.SQLAdminInstance,
			Description:      "If the Cloud SQL Instance is deleted or updated: The service may fail to access the database. If the service is updated: The instance remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{In: true},
		},
		"template.volumes.gcs.bucket": {
			ToSDPItemType:    gcpshared.StorageBucket,
			Description:      "If the Cloud Storage Bucket is deleted or updated: The service may fail to access stored data. If the service is updated: The bucket remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{In: true},
		},
		"template.encryptionKey": gcpshared.CryptoKeyImpactInOnly,
		"latestCreatedRevision": {
			ToSDPItemType:    gcpshared.RunRevision,
			Description:      "If the Cloud Run Service is deleted or updated: Associated revisions may become orphaned or be deleted. If revisions are updated: The service status may reflect the changes.",
			BlastPropagation: &sdp.BlastPropagation{Out: true},
		},
		"latestReadyRevision": {
			ToSDPItemType:    gcpshared.RunRevision,
			Description:      "If the Cloud Run Service is deleted or updated: Associated revisions may become orphaned or be deleted. If revisions are updated: The service status may reflect the changes.",
			BlastPropagation: &sdp.BlastPropagation{Out: true},
		},
		"traffic.revision": {
			ToSDPItemType:    gcpshared.RunRevision,
			Description:      "If the Cloud Run Service is deleted or updated: Traffic allocation to revisions will be lost. If revisions are updated: The service traffic configuration may need updates.",
			BlastPropagation: &sdp.BlastPropagation{Out: true},
		},
		// Forward link from parent to child via SEARCH
		// Link to all revisions in this service
		"name": {
			ToSDPItemType: gcpshared.RunRevision,
			Description:   "If the Cloud Run Service is deleted or updated: All associated Revisions may become invalid or inaccessible. If a Revision is updated: The service remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{
				In:  false,
				Out: true,
			},
			IsParentToChild: true,
		},
		// Link to Binary Authorization platform policy (when explicitly specified via policy field)
		// Note: When useDefault is true, the service uses the project's default policy,
		// but we can't link to it here since there's no explicit policy field value
		"binaryAuthorization.policy": {
			ToSDPItemType:    gcpshared.BinaryAuthorizationPlatformPolicy,
			Description:      "If the Binary Authorization platform policy is updated: The service may fail to deploy new revisions if images don't meet policy requirements. If the service is updated: The policy remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{In: true},
		},
		// Link to Cloud Storage bucket used in buildConfig source (if buildConfig is used)
		"buildConfig.source.storageSource.bucket": {
			ToSDPItemType:    gcpshared.StorageBucket,
			Description:      "If the Cloud Storage Bucket containing source code is deleted or updated: The service may fail to build new revisions. If the service is updated: The bucket remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{In: true},
		},
		// Link to HTTP/HTTPS URLs serving traffic for this service
		"urls": {
			ToSDPItemType:    stdlib.NetworkHTTP,
			Description:      "If the HTTP endpoint becomes unavailable: The service cannot serve traffic. If the service is updated: The endpoint URL may change.",
			BlastPropagation: &sdp.BlastPropagation{In: true, Out: true},
		},
		// Link to main URI serving traffic for this service
		"uri": {
			ToSDPItemType:    stdlib.NetworkHTTP,
			Description:      "If the HTTP endpoint becomes unavailable: The service cannot serve traffic. If the service is updated: The endpoint URI may change.",
			BlastPropagation: &sdp.BlastPropagation{In: true, Out: true},
		},
	},
	terraformMapping: gcpshared.TerraformMapping{
		Reference:   "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/cloud_run_v2_service",
		Description: "id => projects/{{project}}/locations/{{location}}/services/{{name}}",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_SEARCH,
				TerraformQueryMap: "google_cloud_run_v2_service.id",
			},
		},
	},
}.Register()
