package adapters

import (
	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

// AI Platform Model adapter.
// GCP Ref: https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.models/get
// GET  https://aiplatform.googleapis.com/v1/projects/{project}/locations/{location}/models/{model}
// LIST https://aiplatform.googleapis.com/v1/projects/{project}/locations/{location}/models
// NOTE: We use "global" for the location in the URL, because we use the global service endpoint.
var aiPlatformModelAdapter = registerableAdapter{ //nolint:unused
	sdpType: gcpshared.AIPlatformModel,
	meta: gcpshared.AdapterMeta{
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_AI,
		Scope:              gcpshared.ScopeProject,
		GetEndpointBaseURLFunc: gcpshared.ProjectLevelEndpointFuncWithSingleQuery(
			"https://aiplatform.googleapis.com/v1/projects/%s/locations/global/models/%s",
		),
		ListEndpointFunc: gcpshared.ProjectLevelListFunc(
			"https://aiplatform.googleapis.com/v1/projects/%s/locations/global/models",
		),
		UniqueAttributeKeys: []string{"models"},
		IAMPermissions:      []string{"aiplatform.models.get", "aiplatform.models.list"},
		PredefinedRole:      "roles/aiplatform.viewer",
	},
	blastPropagation: map[string]*gcpshared.Impact{
		"encryptionSpec.kmsKeyName": gcpshared.CryptoKeyImpactInOnly,
		// Container image used for prediction (containerSpec.imageUri).
		"containerSpec.imageUri": {
			ToSDPItemType: gcpshared.ArtifactRegistryDockerImage,
			Description:   "If the Artifact Registry Docker Image is updated or deleted: The Model may fail to serve predictions. If the Model is updated: The Docker image remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{
				In: true,
			},
		},
		"pipelineJob": {
			ToSDPItemType: gcpshared.AIPlatformPipelineJob,
			Description:   "If the Pipeline Job is deleted: The Model may not be retrievable. If the Model is updated: The Pipeline Job remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{
				In: true,
			},
		},
		"deployedModels.endpoint": {
			ToSDPItemType: gcpshared.AIPlatformEndpoint,
			Description:   "They are tightly coupled.",
			BlastPropagation: &sdp.BlastPropagation{
				In:  true,
				Out: true,
			},
		},
	},
}.Register()
