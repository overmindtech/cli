package adapters

import (
	"fmt"

	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

// Storage Transfer Transfer Job facilitates data transfers between cloud storage systems and on-premises data
// GCP Ref (GET): https://cloud.google.com/storage-transfer/docs/reference/rest/v1/transferJobs/get
// GCP Ref (Schema): https://cloud.google.com/storage-transfer/docs/reference/rest/v1/transferJobs#TransferJob
// GET  https://storagetransfer.googleapis.com/v1/transferJobs/{jobName}
// LIST https://storagetransfer.googleapis.com/v1/transferJobs
var _ = registerableAdapter{
	sdpType: gcpshared.StorageTransferTransferJob,
	meta: gcpshared.AdapterMeta{
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_STORAGE,
		Scope:              gcpshared.ScopeProject,
		GetEndpointBaseURLFunc: func(adapterInitParams ...string) (gcpshared.EndpointFunc, error) {
			if len(adapterInitParams) == 1 && adapterInitParams[0] != "" {
				return func(query string) string {
					if query != "" {
						// query is the job name, adapterInitParams[0] is the project ID
						return fmt.Sprintf("https://storagetransfer.googleapis.com/v1/transferJobs/%s?projectId=%s", query, adapterInitParams[0])
					}
					return ""
				}, nil
			}
			return nil, fmt.Errorf("projectID cannot be empty: %v", adapterInitParams)
		},
		ListEndpointFunc:    gcpshared.ProjectLevelListFunc("https://storagetransfer.googleapis.com/v1/transferJobs?filter={\"projectId\":\"%s\"}"),
		UniqueAttributeKeys: []string{"transferJobs"},
		IAMPermissions: []string{
			"storagetransfer.jobs.get",
			"storagetransfer.jobs.list",
		},
		PredefinedRole: "roles/storagetransfer.viewer",
		// TODO: https://linear.app/overmind/issue/ENG-631 status
		// https://cloud.google.com/storage-transfer/docs/reference/rest/v1/transferJobs#TransferJob.status
	},
	blastPropagation: map[string]*gcpshared.Impact{
		// Transfer spec references to source and destination storage
		"transferSpec.gcsDataSource.bucketName": {
			ToSDPItemType: gcpshared.StorageBucket,
			Description:   "If the source GCS bucket is deleted or inaccessible: The transfer job will fail. If the transfer job is updated: The source bucket remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{
				In: true,
			},
		},
		"transferSpec.gcsDataSink.bucketName": {
			ToSDPItemType: gcpshared.StorageBucket,
			Description:   "If the destination GCS bucket is deleted or inaccessible: The transfer job will fail. If the transfer job is updated: The destination bucket remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{
				In: true,
			},
		},
		// TODO: Investigate how we can link to AWS and Azure source when the account id (scope) is not available
		// https://cloud.google.com/storage-transfer/docs/reference/rest/v1/TransferSpec#AwsS3Data
		// https://cloud.google.com/storage-transfer/docs/reference/rest/v1/TransferSpec#AzureBlobStorageData
		"transferSpec.httpDataSource.listUrl": {
			ToSDPItemType: gcpshared.StorageBucket,
			Description:   "HTTP data source URL for transfer operations. If the HTTP endpoint is unreachable: The transfer job will fail to access the source data.",
			BlastPropagation: &sdp.BlastPropagation{
				In: true,
			},
		},
		"transferSpec.gcsIntermediateDataLocation.bucketName": {
			ToSDPItemType: gcpshared.StorageBucket,
			Description:   "If the destination GCS bucket is deleted or inaccessible: The transfer job will fail. If the transfer job is updated: The destination bucket remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{
				In: true,
			},
		},
		"serviceAccount": {
			ToSDPItemType: gcpshared.IAMServiceAccount,
			Description:   "If the Service Account is deleted or permissions are revoked: The transfer job may fail to execute. If the transfer job is updated: The service account remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{
				In: true,
			},
		},
		// Notification configuration
		"notificationConfig.pubsubTopic": {
			ToSDPItemType: gcpshared.PubSubTopic,
			Description:   "If the Pub/Sub Topic is deleted: Transfer job notifications will fail. If the transfer job is updated: The Pub/Sub topic remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{
				In: true,
			},
		},
		// TODO: Investigate whether we can/should support multiple items for a given key.
		// In this case, the eventStream can be an AWS SQS ARN in the form 'arn:aws:sqs:region:account_id:queue_name'
		// https://linear.app/overmind/issue/ENG-1348/investigate-supporting-multiple-items-in-blast-propagations
		"eventStream.name": {
			ToSDPItemType: gcpshared.PubSubTopic,
			Description:   "If the Pub/Sub Topic for event streaming is deleted: Transfer job events will not be published. If the transfer job is updated: The Pub/Sub topic remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{
				In: true,
			},
		},
	},
	terraformMapping: gcpshared.TerraformMapping{
		Reference:   "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/storage_transfer_job",
		Description: "name => transferJobs/{jobName}",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_GET,
				TerraformQueryMap: "google_storage_transfer_job.name",
			},
		},
	},
}.Register()
