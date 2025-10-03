package adapters

import (
	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

// AI Platform Batch Prediction Job allows you to get inferences for large datasets using trained machine learning models
// GCP Ref (GET): https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.batchPredictionJobs/get
// GCP Ref (Schema): https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.batchPredictionJobs#BatchPredictionJob
// GET  https://aiplatform.googleapis.com/v1/projects/{project}/locations/{location}/batchPredictionJobs/{batchPredictionJob}
// LIST https://aiplatform.googleapis.com/v1/projects/{project}/locations/{location}/batchPredictionJobs
var _ = registerableAdapter{
	sdpType: gcpshared.AIPlatformBatchPredictionJob,
	meta: gcpshared.AdapterMeta{
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_AI,
		Scope:              gcpshared.ScopeProject,
		GetEndpointBaseURLFunc: gcpshared.ProjectLevelEndpointFuncWithTwoQueries(
			"https://aiplatform.googleapis.com/v1/projects/%s/locations/%s/batchPredictionJobs/%s",
		),
		SearchEndpointFunc: gcpshared.ProjectLevelEndpointFuncWithSingleQuery(
			"https://aiplatform.googleapis.com/v1/projects/%s/locations/%s/batchPredictionJobs",
		),
		SearchDescription:   "Search Batch Prediction Jobs within a location. Use the location name e.g., 'us-central1'",
		UniqueAttributeKeys: []string{"locations", "batchPredictionJobs"},
		IAMPermissions: []string{
			"aiplatform.batchPredictionJobs.get",
			"aiplatform.batchPredictionJobs.list",
		},
		PredefinedRole: "roles/aiplatform.viewer",
		// TODO: https://linear.app/overmind/issue/ENG-631 state
		// https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.batchPredictionJobs#JobState
	},
	blastPropagation: map[string]*gcpshared.Impact{
		"encryptionSpec.kmsKeyName": gcpshared.CryptoKeyImpactInOnly,
		"model": {
			ToSDPItemType: gcpshared.AIPlatformModel,
			Description:   "If the Model is deleted or updated: The batch prediction job may fail. If the batch prediction job is updated: The model remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{
				In: true,
			},
		},
		// TODO: https://linear.app/overmind/issue/ENG-1446/investigate-creating-a-manual-linker-for-cloud-storage
		"inputConfig.gcsSource.uris": {
			ToSDPItemType: gcpshared.StorageBucket,
			Description:   "If the GCS source bucket is deleted or inaccessible: The batch prediction job will fail to read input data. If the batch prediction job is updated: The bucket remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{
				In: true,
			},
		},
		// TODO:
		// BigQuery path. For example: bq://projectId.bqDatasetId.bqTableId.
		// Related: https://linear.app/overmind/issue/ENG-1281/add-big-query-adapters-to-manual-links
		"inputConfig.bigquerySource.inputUri": {
			ToSDPItemType: gcpshared.BigQueryTable,
			Description:   "If the BigQuery table is deleted or inaccessible: The batch prediction job will fail to read input data. If the batch prediction job is updated: The table remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{
				In: true,
			},
		},
		// TODO: https://linear.app/overmind/issue/ENG-1446/investigate-creating-a-manual-linker-for-cloud-storage
		"outputConfig.gcsDestination.outputUriPrefix": {
			ToSDPItemType: gcpshared.StorageBucket,
			Description:   "If the output GCS bucket is deleted or inaccessible: The batch prediction job will fail to write results. If the batch prediction job is updated: The bucket remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{
				In: true,
			},
		},
		// TODO:
		// BigQuery path. For example: bq://projectId.bqDatasetId.bqTableId.
		// Related: https://linear.app/overmind/issue/ENG-1281/add-big-query-adapters-to-manual-links
		"outputConfig.bigqueryDestination.outputUri": {
			ToSDPItemType: gcpshared.BigQueryTable,
			Description:   "If the BigQuery output table is deleted or inaccessible: The batch prediction job will fail to write results. If the batch prediction job is updated: The table remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{
				In: true,
			},
		},
		"serviceAccount": {
			ToSDPItemType: gcpshared.IAMServiceAccount,
			Description:   "If the Service Account is deleted or permissions are revoked: The batch prediction job may fail to access required resources. If the batch prediction job is updated: The service account remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{
				In: true,
			},
		},
	},
}.Register()
