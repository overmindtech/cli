package adapters

import (
	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

// AI Platform Custom Job adapter for Vertex AI custom training jobs
// There are multiple service endpoints: https://cloud.google.com/vertex-ai/docs/reference/rest#rest_endpoints
// We stick to the default one for now: https://aiplatform.googleapis.com
// Other endpoints are in the form of https://{region}-aiplatform.googleapis.com
// If we use the default endpoint the location must be set to `global`.
// So, for simplicity, we can get custom jobs by their name globally, list globally,
// otherwise we have to check the validity of the location and use the regional endpoint.
var _ = registerableAdapter{
	sdpType: gcpshared.AIPlatformCustomJob,
	meta: gcpshared.AdapterMeta{
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_AI,
		LocationLevel:      gcpshared.ProjectLevel,
		// Vertex AI API must be enabled for the project!
		// Reference: https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.customJobs/get
		// https://aiplatform.googleapis.com/v1/projects/{project}/locations/{location}/customJobs/{customJob}
		GetEndpointFunc: gcpshared.ProjectLevelEndpointFuncWithSingleQuery("https://aiplatform.googleapis.com/v1/projects/%s/locations/global/customJobs/%s"),
		// Reference: https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.customJobs/list
		// https://aiplatform.googleapis.com/v1/projects/{project}/locations/{location}/customJobs
		// Expected location is `global` for the default endpoint.
		ListEndpointFunc:    gcpshared.ProjectLevelListFunc("https://aiplatform.googleapis.com/v1/projects/%s/locations/global/customJobs"),
		UniqueAttributeKeys: []string{"customJobs"},
		IAMPermissions:      []string{"aiplatform.customJobs.get", "aiplatform.customJobs.list"},
		PredefinedRole:      "roles/aiplatform.viewer",
	},
	blastPropagation: map[string]*gcpshared.Impact{
		// The Cloud KMS key that will be used to encrypt the output artifacts.
		"encryptionSpec.kmsKeyName": {
			Description:      "If the Cloud KMS CryptoKey is updated: The CustomJob may not be able to access encrypted output artifacts. If the CustomJob is updated: The CryptoKey remains unaffected.",
			ToSDPItemType:    gcpshared.CloudKMSCryptoKey,
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		// The full name of the network to which the job should be peered.
		"jobSpec.network": {
			Description:      "If the Compute Network is deleted or updated: The CustomJob may lose connectivity or fail to run as expected. If the CustomJob is updated: The network remains unaffected.",
			ToSDPItemType:    gcpshared.ComputeNetwork,
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		// The service account that the job runs as.
		"jobSpec.serviceAccount": {
			Description:      "If the IAM Service Account is deleted or updated: The CustomJob may fail to run or lose permissions. If the CustomJob is updated: The service account remains unaffected.",
			ToSDPItemType:    gcpshared.IAMServiceAccount,
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		// The Cloud Storage location to store the output of this CustomJob.
		"jobSpec.baseOutputDirectory.gcsOutputDirectory": {
			Description:      "If the Storage Bucket is deleted or updated: The CustomJob may fail to write outputs. If the CustomJob is updated: The bucket remains unaffected.",
			ToSDPItemType:    gcpshared.StorageBucket,
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		// Optional. The name of a Vertex AI Tensorboard resource to which this CustomJob will upload Tensorboard logs.
		"jobSpec.tensorboard": {
			Description:      "If the Vertex AI Tensorboard is deleted or updated: The CustomJob may fail to upload logs or lose access to previous logs. If the CustomJob is updated: The tensorboard remains unaffected.",
			ToSDPItemType:    gcpshared.AIPlatformTensorBoard,
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		// Optional. The name of an experiment to associate with the CustomJob.
		"jobSpec.experiment": {
			Description:      "If the Vertex AI Experiment is deleted or updated: The CustomJob may lose experiment tracking or association. If the CustomJob is updated: The experiment remains unaffected.",
			ToSDPItemType:    gcpshared.AIPlatformExperiment,
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		// Optional. The name of an experiment run to associate with the CustomJob.
		"jobSpec.experimentRun": {
			Description:      "If the Vertex AI ExperimentRun is deleted or updated: The CustomJob may lose run tracking or association. If the CustomJob is updated: The experiment run remains unaffected.",
			ToSDPItemType:    gcpshared.AIPlatformExperimentRun,
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		// Optional. The name of a model to upload the trained Model to upon job completion.
		"jobSpec.models": {
			Description:      "If the Vertex AI Model is deleted or updated: The CustomJob may fail to upload or associate the trained model. If the CustomJob is updated: The model remains unaffected.",
			ToSDPItemType:    gcpshared.AIPlatformModel,
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		// Optional. The ID of a PersistentResource to run the job on existing machines.
		"jobSpec.persistentResourceId": {
			Description:      "If the Vertex AI PersistentResource is deleted or updated: The CustomJob may fail to run or lose access to the persistent resources. If the CustomJob is updated: The PersistentResource remains unaffected.",
			ToSDPItemType:    gcpshared.AIPlatformPersistentResource,
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		// Container image URI used in worker pool specs (for containerSpec).
		"jobSpec.workerPoolSpecs.containerSpec.imageUri": {
			Description:      "If the Artifact Registry Docker Image is updated or deleted: The CustomJob may fail to run or use an incorrect container image. If the CustomJob is updated: The Docker image remains unaffected.",
			ToSDPItemType:    gcpshared.ArtifactRegistryDockerImage,
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		// Executor container image URI used in worker pool specs (for pythonPackageSpec).
		"jobSpec.workerPoolSpecs.pythonPackageSpec.executorImageUri": {
			Description:      "If the Artifact Registry Docker Image is updated or deleted: The CustomJob may fail to run or use an incorrect executor image. If the CustomJob is updated: The Docker image remains unaffected.",
			ToSDPItemType:    gcpshared.ArtifactRegistryDockerImage,
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		// GCS URIs of Python package files used in worker pool specs.
		"jobSpec.workerPoolSpecs.pythonPackageSpec.packageUris": {
			Description:      "If the Storage Bucket containing the Python packages is deleted or updated: The CustomJob may fail to access required package files. If the CustomJob is updated: The bucket remains unaffected.",
			ToSDPItemType:    gcpshared.StorageBucket,
			BlastPropagation: gcpshared.ImpactInOnly,
		},
	},
	terraformMapping: gcpshared.TerraformMapping{
		Description: "There is no terraform resource for this type.",
	},
}.Register()
