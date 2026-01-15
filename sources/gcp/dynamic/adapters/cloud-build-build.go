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
		// Artifacts storage location (Cloud Storage bucket for build artifacts)
		"artifacts.objects.location": {
			ToSDPItemType:    gcpshared.StorageBucket,
			Description:      "If the Storage Bucket for artifacts is deleted or updated: The Cloud Build may fail to store build artifacts. If the Cloud Build is updated: The bucket remains unaffected.",
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		// Maven artifacts repository in Artifact Registry
		"artifacts.mavenArtifacts.repository": {
			ToSDPItemType:    gcpshared.ArtifactRegistryRepository,
			Description:      "If the Artifact Registry Repository for Maven artifacts is deleted or updated: The Cloud Build may fail to store Maven artifacts. If the Cloud Build is updated: The repository remains unaffected.",
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		// NPM packages repository in Artifact Registry
		"artifacts.npmPackages.repository": {
			ToSDPItemType:    gcpshared.ArtifactRegistryRepository,
			Description:      "If the Artifact Registry Repository for NPM packages is deleted or updated: The Cloud Build may fail to store NPM packages. If the Cloud Build is updated: The repository remains unaffected.",
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		// Python packages repository in Artifact Registry
		"artifacts.pythonPackages.repository": {
			ToSDPItemType:    gcpshared.ArtifactRegistryRepository,
			Description:      "If the Artifact Registry Repository for Python packages is deleted or updated: The Cloud Build may fail to store Python packages. If the Cloud Build is updated: The repository remains unaffected.",
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		// Secret Manager secrets used in the build (availableSecrets.secretManager[].version)
		// The version field contains the full path: projects/{project}/secrets/{secret}/versions/{version}
		// The framework will automatically extract the secret name from this path and handle array elements
		"availableSecrets.secretManager.version": {
			ToSDPItemType:    gcpshared.SecretManagerSecret,
			Description:      "If the Secret Manager Secret is deleted or its access is revoked: The Cloud Build may fail to access required secrets during execution. If the Cloud Build is updated: The secret remains unaffected.",
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		// Worker pool used for the build (same as Cloud Functions - Run Worker Pool)
		"options.pool.name": {
			ToSDPItemType:    gcpshared.RunWorkerPool,
			Description:      "If the Cloud Run Worker Pool is deleted or misconfigured: The Cloud Build may fail to execute. If the Cloud Build is updated: The worker pool remains unaffected.",
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		// KMS key for encrypting build logs (if using CMEK)
		"options.kmsKeyName": gcpshared.CryptoKeyImpactInOnly,
	},
	terraformMapping: gcpshared.TerraformMapping{
		Description: "There is no terraform resource for this type.",
	},
}.Register()
