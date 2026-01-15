package adapters

import (
	"fmt"

	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/stdlib"
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
		// AWS S3 data source credentials secret (Secret Manager)
		"transferSpec.awsS3DataSource.credentialsSecret": {
			ToSDPItemType: gcpshared.SecretManagerSecret,
			Description:   "If the Secret Manager secret containing AWS credentials is deleted or updated: The transfer job may fail to authenticate with AWS S3. If the transfer job is updated: The secret remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{
				In: true,
			},
		},
		// AWS S3 data source CloudFront domain (HTTP endpoint)
		"transferSpec.awsS3DataSource.cloudfrontDomain": {
			ToSDPItemType: stdlib.NetworkHTTP,
			Description:   "If the CloudFront domain endpoint is unreachable: The transfer job will fail to access the source data via CloudFront. If the transfer job is updated: The CloudFront endpoint remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{
				In:  true,
				Out: false,
			},
		},
		// Azure Blob Storage data source credentials secret (Secret Manager)
		"transferSpec.azureBlobStorageDataSource.credentialsSecret": {
			ToSDPItemType: gcpshared.SecretManagerSecret,
			Description:   "If the Secret Manager secret containing Azure SAS token is deleted or updated: The transfer job may fail to authenticate with Azure Blob Storage. If the transfer job is updated: The secret remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{
				In: true,
			},
		},
		// Agent pool for POSIX source
		"transferSpec.sourceAgentPoolName": {
			ToSDPItemType: gcpshared.StorageTransferAgentPool,
			Description:   "If the source Agent Pool is deleted or updated: The transfer job may fail to access POSIX source file systems. If the transfer job is updated: The agent pool remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{
				In: true,
			},
		},
		// Agent pool for POSIX sink
		"transferSpec.sinkAgentPoolName": {
			ToSDPItemType: gcpshared.StorageTransferAgentPool,
			Description:   "If the sink Agent Pool is deleted or updated: The transfer job may fail to write to POSIX sink file systems. If the transfer job is updated: The agent pool remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{
				In: true,
			},
		},
		// Transfer manifest location (gs:// URI pointing to manifest file)
		"transferSpec.transferManifest.location": {
			ToSDPItemType: gcpshared.StorageBucket,
			Description:   "If the Storage Bucket containing the transfer manifest is deleted or inaccessible: The transfer job may fail to read the manifest file. If the transfer job is updated: The bucket remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{
				In: true,
			},
		},
		// HTTP data source URL - link to HTTP endpoint using stdlib
		"transferSpec.httpDataSource.listUrl": {
			ToSDPItemType: stdlib.NetworkHTTP,
			Description:   "HTTP data source URL for transfer operations. If the HTTP endpoint is unreachable: The transfer job will fail to access the source data.",
			BlastPropagation: &sdp.BlastPropagation{
				In:  true,
				Out: false,
			},
		},
		"transferSpec.gcsIntermediateDataLocation.bucketName": {
			ToSDPItemType: gcpshared.StorageBucket,
			Description:   "If the intermediate GCS bucket is deleted or inaccessible: The transfer job will fail. If the transfer job is updated: The intermediate bucket remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{
				In: true,
			},
		},
		// Replication spec source bucket
		"replicationSpec.gcsDataSource.bucketName": {
			ToSDPItemType: gcpshared.StorageBucket,
			Description:   "If the source GCS bucket for replication is deleted or inaccessible: The replication job will fail. If the replication job is updated: The source bucket remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{
				In: true,
			},
		},
		// Replication spec destination bucket
		"replicationSpec.gcsDataSink.bucketName": {
			ToSDPItemType: gcpshared.StorageBucket,
			Description:   "If the destination GCS bucket for replication is deleted or inaccessible: The replication job will fail. If the replication job is updated: The destination bucket remains unaffected.",
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
		// Required. Specifies a unique name of the resource such as AWS SQS ARN in the form 'arn:aws:sqs:region:account_id:queue_name',
		// or Pub/Sub subscription resource name in the form 'projects/{project}/subscriptions/{sub}'.
		"eventStream.name": {
			ToSDPItemType: gcpshared.PubSubSubscription,
			Description:   "If the Pub/Sub Subscription for event streaming is deleted: Transfer job events will not be consumed. If the transfer job is updated: The Pub/Sub subscription remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{
				In: true,
			},
		},
		// Latest transfer operation (child resource)
		"latestOperationName": {
			ToSDPItemType: gcpshared.StorageTransferTransferOperation,
			Description:   "If the Transfer Operation is deleted or updated: The transfer job's latest operation reference may become invalid. If the transfer job is updated: The operation remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{
				In:  true,
				Out: false,
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
