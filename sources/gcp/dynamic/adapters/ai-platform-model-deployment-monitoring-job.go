package adapters

import (
	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

// AI Platform Model Deployment Monitoring Job adapter (IN DEVELOPMENT)
// GCP Ref (GET): https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.modelDeploymentMonitoringJobs/get
// GCP Ref (Schema): https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.modelDeploymentMonitoringJobs#ModelDeploymentMonitoringJob
// GET  https://aiplatform.googleapis.com/v1/projects/{project}/locations/{location}/modelDeploymentMonitoringJobs/{modelDeploymentMonitoringJob}
// LIST https://aiplatform.googleapis.com/v1/projects/{project}/locations/{location}/modelDeploymentMonitoringJobs
// NOTE: We use "global" for the location in the URL, because we use the global service endpoint.
var aiPlatformModelDeploymentMonitoringJobAdapter = registerableAdapter{ //nolint:unused
	sdpType: gcpshared.AIPlatformModelDeploymentMonitoringJob,
	meta: gcpshared.AdapterMeta{
		InDevelopment:      true,
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_AI,
		Scope:              gcpshared.ScopeProject,
		GetEndpointBaseURLFunc: gcpshared.ProjectLevelEndpointFuncWithSingleQuery(
			"https://aiplatform.googleapis.com/v1/projects/%s/locations/global/modelDeploymentMonitoringJobs/%s",
		),
		ListEndpointFunc: gcpshared.ProjectLevelListFunc(
			"https://aiplatform.googleapis.com/v1/projects/%s/locations/global/modelDeploymentMonitoringJobs",
		),
		UniqueAttributeKeys: []string{"modelDeploymentMonitoringJobs"},
		IAMPermissions: []string{
			"aiplatform.modelDeploymentMonitoringJobs.get",
			"aiplatform.modelDeploymentMonitoringJobs.list",
		},
	},
	// TODO: Evaluate references (e.g. endpoint, models, KMS key) for blast propagation after schema review.
	blastPropagation: map[string]*gcpshared.Impact{},
}.Register()
