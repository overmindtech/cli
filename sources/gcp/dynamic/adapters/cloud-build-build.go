package adapters

import (
	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

// Cloud Build Build adapter for Cloud Build builds
var _ = registerableAdapter{
	sdpType: gcpshared.CloudBuildBuild,
	meta: gcpshared.AdapterMeta{
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_CONFIGURATION,
		Scope:              gcpshared.ScopeProject,
		// Reference: https://cloud.google.com/build/docs/api/reference/rest/v1/projects.builds/get
		// GET https://cloudbuild.googleapis.com/v1/projects/{projectId}/builds/{id}
		// IAM permissions: cloudbuild.builds.get
		GetEndpointBaseURLFunc: gcpshared.ProjectLevelEndpointFuncWithSingleQuery("https://cloudbuild.googleapis.com/v1/projects/%s/builds/%s"),
		// Reference: https://cloud.google.com/build/docs/api/reference/rest/v1/projects.builds/list
		// GET https://cloudbuild.googleapis.com/v1/projects/{projectId}/builds
		ListEndpointFunc:    gcpshared.ProjectLevelListFunc("https://cloudbuild.googleapis.com/v1/projects/%s/builds"),
		UniqueAttributeKeys: []string{"builds"},
		// HEALTH: https://cloud.google.com/build/docs/api/reference/rest/v1/projects.builds#Build.Status
		// TODO: https://linear.app/overmind/issue/ENG-631/investigate-how-we-can-add-health-status-for-supporting-items
		IAMPermissions: []string{"cloudbuild.builds.get", "cloudbuild.builds.list"},
		PredefinedRole: "roles/cloudbuild.builds.viewer",
	},
	blastPropagation: map[string]*gcpshared.Impact{
		"source.storageSource.bucket": {
			ToSDPItemType:    gcpshared.StorageBucket,
			Description:      "If the Storage Bucket is deleted or updated: The Cloud Build may fail to access source files. If the Cloud Build is updated: The bucket remains unaffected.",
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		"steps.name": {
			ToSDPItemType:    gcpshared.ArtifactRegistryDockerImage,
			Description:      "If the Artifact Registry Docker Image is deleted or updated: The Cloud Build may fail to pull the image. If the Cloud Build is updated: The Docker image remains unaffected.",
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		"results.images": {
			ToSDPItemType:    gcpshared.ArtifactRegistryDockerImage,
			Description:      "If the Cloud Build is updated or deleted: The Artifact Registry Docker Images may no longer be valid or accessible. If the Docker Images are updated: The Cloud Build remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{Out: true},
		},
		"images": {
			ToSDPItemType:    gcpshared.ArtifactRegistryDockerImage,
			Description:      "If any of the images fail to be pushed, the build status is marked FAILURE.",
			BlastPropagation: &sdp.BlastPropagation{Out: true},
		},
		"logsBucket": {
			ToSDPItemType:    gcpshared.LoggingBucket,
			Description:      "If the Logging Bucket is deleted or updated: The Cloud Build may fail to write logs. If the Cloud Build is updated: The bucket remains unaffected.",
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		"serviceAccount": gcpshared.IAMServiceAccountImpactInOnly,
		"buildTriggerId": {
			// The ID of the BuildTrigger that triggered this build, if it was triggered automatically.
			ToSDPItemType:    gcpshared.CloudBuildTrigger,
			Description:      "If the Cloud Build Trigger is deleted or updated: The Cloud Build may not be retriggered as expected. If the Cloud Build is updated: The trigger remains unaffected.",
			BlastPropagation: gcpshared.ImpactInOnly,
		},
	},
	terraformMapping: gcpshared.TerraformMapping{
		Description: "There is no terraform resource for this type.",
	},
}.Register()
