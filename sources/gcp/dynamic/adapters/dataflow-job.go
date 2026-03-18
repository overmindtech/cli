package adapters

import (
	"github.com/overmindtech/cli/go/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

// Dataflow Job adapter for Google Cloud Dataflow jobs.
// Reference: https://cloud.google.com/dataflow/docs/reference/rest/v1b3/projects.locations.jobs#Job
// GET:    https://dataflow.googleapis.com/v1b3/projects/{project}/locations/{location}/jobs/{jobId}
// LIST:   https://dataflow.googleapis.com/v1b3/projects/{project}/jobs:aggregated
// SEARCH: https://dataflow.googleapis.com/v1b3/projects/{project}/locations/{location}/jobs
var _ = registerableAdapter{
	sdpType: gcpshared.DataflowJob,
	meta: gcpshared.AdapterMeta{
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_COMPUTE_APPLICATION,
		LocationLevel:      gcpshared.ProjectLevel,
		GetEndpointFunc: gcpshared.ProjectLevelEndpointFuncWithTwoQueries(
			"https://dataflow.googleapis.com/v1b3/projects/%s/locations/%s/jobs/%s",
		),
		SearchEndpointFunc: gcpshared.ProjectLevelEndpointFuncWithSingleQuery(
			"https://dataflow.googleapis.com/v1b3/projects/%s/locations/%s/jobs",
		),
		ListEndpointFunc: gcpshared.ProjectLevelListFunc(
			"https://dataflow.googleapis.com/v1b3/projects/%s/jobs:aggregated",
		),
		UniqueAttributeKeys: []string{"locations", "jobs"},
		IAMPermissions:      []string{"dataflow.jobs.get", "dataflow.jobs.list"},
		PredefinedRole:      "roles/dataflow.viewer",
	},
	linkRules: map[string]*gcpshared.Impact{
		// Pub/Sub links (critical for ENG-3217 outage detection)
		"jobMetadata.pubsubDetails.topic": {
			ToSDPItemType: gcpshared.PubSubTopic,
			Description:   "If the Pub/Sub Topic is deleted or misconfigured: The Dataflow job may fail to read/write messages. If the Dataflow job changes: The topic remains unaffected.",
		},
		"jobMetadata.pubsubDetails.subscription": {
			ToSDPItemType: gcpshared.PubSubSubscription,
			Description:   "If the Pub/Sub Subscription is deleted or misconfigured: The Dataflow job may fail to consume messages. If the Dataflow job changes: The subscription remains unaffected.",
		},

		// BigQuery links
		"jobMetadata.bigqueryDetails.table": {
			ToSDPItemType: gcpshared.BigQueryTable,
			Description:   "If the BigQuery Table is deleted or misconfigured: The Dataflow job may fail to read/write data. If the Dataflow job changes: The table remains unaffected.",
		},
		"jobMetadata.bigqueryDetails.dataset": {
			ToSDPItemType: gcpshared.BigQueryDataset,
			Description:   "If the BigQuery Dataset is deleted or misconfigured: The Dataflow job may fail to access tables. If the Dataflow job changes: The dataset remains unaffected.",
		},

		// Spanner links
		"jobMetadata.spannerDetails.instanceId": {
			ToSDPItemType: gcpshared.SpannerInstance,
			Description:   "If the Spanner Instance is deleted or misconfigured: The Dataflow job may fail to read/write data. If the Dataflow job changes: The instance remains unaffected.",
		},
		// Bigtable links
		"jobMetadata.bigTableDetails.instanceId": {
			ToSDPItemType: gcpshared.BigTableAdminInstance,
			Description:   "If the Bigtable Instance is deleted or misconfigured: The Dataflow job may fail to read/write data. If the Dataflow job changes: The instance remains unaffected.",
		},
		// Environment/infra links
		"environment.serviceAccountEmail": gcpshared.IAMServiceAccountImpactInOnly,
		"environment.serviceKmsKeyName":   gcpshared.CryptoKeyImpactInOnly,
		"environment.workerPools.network": gcpshared.ComputeNetworkImpactInOnly,
		"environment.workerPools.subnetwork": {
			ToSDPItemType: gcpshared.ComputeSubnetwork,
			Description:   "If the Compute Subnetwork is deleted or misconfigured: Dataflow workers may lose connectivity or fail to start. If the Dataflow job changes: The subnetwork remains unaffected.",
		},
	},
	terraformMapping: gcpshared.TerraformMapping{
		Reference: "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/dataflow_job",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_GET,
				TerraformQueryMap: "google_dataflow_job.job_id",
			},
			{
				TerraformMethod:   sdp.QueryMethod_GET,
				TerraformQueryMap: "google_dataflow_flex_template_job.job_id",
			},
		},
	},
}.Register()
