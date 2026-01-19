package adapters

import (
	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

// AI Platform Model Deployment Monitoring Job monitors deployed models for data drift and performance degradation
// GCP Ref (GET): https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.modelDeploymentMonitoringJobs/get
// GCP Ref (Schema): https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.modelDeploymentMonitoringJobs#ModelDeploymentMonitoringJob
// GET  https://aiplatform.googleapis.com/v1/projects/{project}/locations/{location}/modelDeploymentMonitoringJobs/{modelDeploymentMonitoringJob}
// LIST https://aiplatform.googleapis.com/v1/projects/{project}/locations/{location}/modelDeploymentMonitoringJobs
var _ = registerableAdapter{
	sdpType: gcpshared.AIPlatformModelDeploymentMonitoringJob,
	meta: gcpshared.AdapterMeta{
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_AI,
		LocationLevel:      gcpshared.ProjectLevel,
		GetEndpointFunc: gcpshared.ProjectLevelEndpointFuncWithTwoQueries(
			"https://aiplatform.googleapis.com/v1/projects/%s/locations/%s/modelDeploymentMonitoringJobs/%s",
		),
		SearchEndpointFunc: gcpshared.ProjectLevelEndpointFuncWithSingleQuery(
			"https://aiplatform.googleapis.com/v1/projects/%s/locations/%s/modelDeploymentMonitoringJobs",
		),
		SearchDescription:   "Search Model Deployment Monitoring Jobs within a location. Use the location name e.g., 'us-central1'",
		UniqueAttributeKeys: []string{"locations", "modelDeploymentMonitoringJobs"},
		IAMPermissions: []string{
			"aiplatform.modelDeploymentMonitoringJobs.get",
			"aiplatform.modelDeploymentMonitoringJobs.list",
		},
		PredefinedRole: "roles/aiplatform.viewer",
		// TODO: https://linear.app/overmind/issue/ENG-631/investigate-how-we-can-add-health-status-for-supporting-items
		// https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.modelDeploymentMonitoringJobs#JobState
	},
	blastPropagation: map[string]*gcpshared.Impact{
		"encryptionSpec.kmsKeyName": gcpshared.CryptoKeyImpactInOnly,
		"endpoint": {
			ToSDPItemType: gcpshared.AIPlatformEndpoint,
			Description:   "They are tightly coupled - monitoring job monitors the endpoint's deployed models.",
			BlastPropagation: &sdp.BlastPropagation{
				In:  true,
				Out: true,
			},
		},
		"modelDeploymentMonitoringObjectiveConfigs.deployedModelId": {
			ToSDPItemType: gcpshared.AIPlatformModel,
			Description:   "If the Model is deleted or updated: The monitoring job may fail to monitor. If the monitoring job is updated: The model remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{
				In: true,
			},
		},
		"modelMonitoringAlertConfig.notificationChannels": {
			ToSDPItemType: gcpshared.MonitoringNotificationChannel,
			Description:   "If the Notification Channel is deleted or updated: The monitoring job may fail to send alerts. If the monitoring job is updated: The notification channel remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{
				In: true,
			},
		},
		"bigqueryTables.bigqueryTablePath": {
			ToSDPItemType: gcpshared.BigQueryTable,
			Description:   "If the BigQuery table storing monitoring logs is deleted or inaccessible: The monitoring job may fail to write logs. If the monitoring job is updated: The table remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{
				In: true,
			},
		},
		"modelDeploymentMonitoringObjectiveConfigs.objectiveConfig.trainingDataset.gcsSource.uris": {
			ToSDPItemType: gcpshared.StorageBucket,
			Description:   "If the GCS bucket containing training data is deleted or inaccessible: The monitoring job may fail to compare predictions against training data. If the monitoring job is updated: The bucket remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{
				In: true,
			},
		},
		"modelDeploymentMonitoringObjectiveConfigs.objectiveConfig.trainingDataset.bigquerySource.inputUri": {
			ToSDPItemType: gcpshared.BigQueryTable,
			Description:   "If the BigQuery table containing training data is deleted or inaccessible: The monitoring job may fail to compare predictions against training data. If the monitoring job is updated: The table remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{
				In: true,
			},
		},
		"predictInstanceSchemaUri": {
			ToSDPItemType: gcpshared.StorageBucket,
			Description:   "If the GCS bucket containing the prediction instance schema is deleted or inaccessible: The monitoring job may fail to validate prediction requests. If the monitoring job is updated: The bucket remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{
				In: true,
			},
		},
		"analysisInstanceSchemaUri": {
			ToSDPItemType: gcpshared.StorageBucket,
			Description:   "If the GCS bucket containing the analysis instance schema is deleted or inaccessible: The monitoring job may fail to perform analysis. If the monitoring job is updated: The bucket remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{
				In: true,
			},
		},
	},
}.Register()
