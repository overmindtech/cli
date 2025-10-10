package adapters

import (
	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

// AI Platform Pipeline Job adapter for Vertex AI pipeline jobs
var _ = registerableAdapter{
	sdpType: gcpshared.AIPlatformPipelineJob,
	meta: gcpshared.AdapterMeta{
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_AI,
		Scope:              gcpshared.ScopeProject,
		// When using the default endpoint, the location must be set to `global`.
		//  Format: projects/{project}/locations/{location}/pipelineJobs/{pipelineJob}
		GetEndpointBaseURLFunc: gcpshared.ProjectLevelEndpointFuncWithSingleQuery("https://aiplatform.googleapis.com/v1/projects/%s/locations/global/pipelineJobs/%s"),
		// Reference: https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.pipelineJobs/list
		ListEndpointFunc:    gcpshared.ProjectLevelListFunc("https://aiplatform.googleapis.com/v1/projects/%s/locations/global/pipelineJobs"),
		UniqueAttributeKeys: []string{"pipelineJobs"},
		IAMPermissions:      []string{"aiplatform.pipelineJobs.get", "aiplatform.pipelineJobs.list"},
		PredefinedRole:      "roles/aiplatform.viewer",
	},
	blastPropagation: map[string]*gcpshared.Impact{
		// The service account that the pipeline workload runs as (root-level).
		"serviceAccount": gcpshared.IAMServiceAccountImpactInOnly,
		// The full name of the network to which the job should be peered (root-level).
		"network": gcpshared.ComputeNetworkImpactInOnly,
		// The Cloud KMS key used to encrypt PipelineJob outputs.
		"encryptionSpec.kmsKeyName": gcpshared.CryptoKeyImpactInOnly,
		// The Cloud Storage location to store the output of this PipelineJob.
		"runtimeConfig.gcsOutputDirectory": {
			Description:      "If the Storage Bucket is deleted or updated: The PipelineJob may fail to write outputs. If the PipelineJob is updated: The bucket remains unaffected.",
			ToSDPItemType:    gcpshared.StorageBucket,
			BlastPropagation: gcpshared.ImpactInOnly,
		},
	},
	terraformMapping: gcpshared.TerraformMapping{
		Description: "There is no terraform resource for this type.",
	},
}.Register()
