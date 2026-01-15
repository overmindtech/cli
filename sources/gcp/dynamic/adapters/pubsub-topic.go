package adapters

import (
	"github.com/overmindtech/cli/sdp-go"
	aws "github.com/overmindtech/cli/sources/aws/shared"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

// Pub/Sub Topic adapter for Google Cloud Pub/Sub topics
var _ = registerableAdapter{
	sdpType: gcpshared.PubSubTopic,
	meta: gcpshared.AdapterMeta{
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_CONFIGURATION,
		Scope:              gcpshared.ScopeProject,
		// https://pubsub.googleapis.com/v1/projects/{project}/topics/{topic}
		GetEndpointBaseURLFunc: gcpshared.ProjectLevelEndpointFuncWithSingleQuery("https://pubsub.googleapis.com/v1/projects/%s/topics/%s"),
		// https://pubsub.googleapis.com/v1/projects/{project}/topics
		ListEndpointFunc:    gcpshared.ProjectLevelListFunc("https://pubsub.googleapis.com/v1/projects/%s/topics"),
		UniqueAttributeKeys: []string{"topics"},
		// HEALTH: https://cloud.google.com/pubsub/docs/reference/rest/v1/projects.topics#state
		// TODO: https://linear.app/overmind/issue/ENG-631/investigate-how-we-can-add-health-status-for-supporting-items
		IAMPermissions: []string{"pubsub.topics.get", "pubsub.topics.list"},
		PredefinedRole: "roles/pubsub.viewer",
	},
	blastPropagation: map[string]*gcpshared.Impact{
		"kmsKeyName": gcpshared.CryptoKeyImpactInOnly,
		// Schema settings for message validation
		"schemaSettings.schema": {
			ToSDPItemType:    gcpshared.PubSubSchema,
			Description:      "If the Pub/Sub Schema is deleted or updated: The Topic may fail to validate messages. If the Topic is updated: The schema remains unaffected.",
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		// Settings for ingestion from a data source into this topic.
		"ingestionDataSourceSettings.cloudStorage.bucket": {
			ToSDPItemType:    gcpshared.StorageBucket,
			Description:      "If the Cloud Storage Bucket is deleted or updated: The Pub/Sub Topic may fail to receive data. If the Topic is updated: The bucket remains unaffected.",
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		"ingestionDataSourceSettings.awsKinesis.streamArn": {
			ToSDPItemType:    aws.KinesisStream,
			Description:      "The Kinesis stream ARN to ingest data from. If the Kinesis stream is deleted or updated: The Pub/Sub Topic may fail to receive data. If the Topic is updated: The stream remains unaffected.",
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		"ingestionDataSourceSettings.awsKinesis.consumerArn": {
			ToSDPItemType:    aws.KinesisStreamConsumer,
			Description:      "The Kinesis consumer ARN used for ingestion in Enhanced Fan-Out mode. The consumer must be already created and ready to be used. If the consumer is deleted or updated: The Pub/Sub Topic may fail to receive data. If the Topic is updated: The consumer remains unaffected.",
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		"ingestionDataSourceSettings.awsKinesis.awsRoleArn": {
			ToSDPItemType:    aws.IAMRole,
			Description:      "AWS role to be used for Federated Identity authentication with Kinesis. If the AWS IAM role is deleted or updated: The Pub/Sub Topic may fail to authenticate and receive data. If the Topic is updated: The role remains unaffected.",
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		"ingestionDataSourceSettings.awsKinesis.gcpServiceAccount": {
			ToSDPItemType:    gcpshared.IAMServiceAccount,
			Description:      "GCP service account used for federated identity authentication with AWS Kinesis. If the service account is deleted or updated: The Pub/Sub Topic may fail to authenticate and receive data. If the Topic is updated: The service account remains unaffected.",
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		"ingestionDataSourceSettings.awsMsk.clusterArn": {
			ToSDPItemType:    aws.MSKCluster,
			Description:      "AWS MSK cluster ARN to ingest data from. If the MSK cluster is deleted or updated: The Pub/Sub Topic may fail to receive data. If the Topic is updated: The cluster remains unaffected.",
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		"ingestionDataSourceSettings.awsMsk.awsRoleArn": {
			ToSDPItemType:    aws.IAMRole,
			Description:      "AWS role to be used for Federated Identity authentication with AWS MSK. If the AWS IAM role is deleted or updated: The Pub/Sub Topic may fail to authenticate and receive data. If the Topic is updated: The role remains unaffected.",
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		"ingestionDataSourceSettings.awsMsk.gcpServiceAccount": {
			ToSDPItemType:    gcpshared.IAMServiceAccount,
			Description:      "GCP service account used for federated identity authentication with AWS MSK. If the service account is deleted or updated: The Pub/Sub Topic may fail to authenticate and receive data. If the Topic is updated: The service account remains unaffected.",
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		"ingestionDataSourceSettings.confluentCloud.bootstrapServer": {
			ToSDPItemType:    stdlib.NetworkDNS,
			Description:      "Confluent Cloud bootstrap server endpoint (hostname:port). The linker automatically detects whether the value is a DNS name or IP address and creates the appropriate link. If the bootstrap server is unreachable: The Pub/Sub Topic may fail to receive data. If the Topic is updated: The bootstrap server remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{
				In:  true,
				Out: true,
			},
		},
		"ingestionDataSourceSettings.confluentCloud.gcpServiceAccount": {
			ToSDPItemType:    gcpshared.IAMServiceAccount,
			Description:      "GCP service account used for federated identity authentication with Confluent Cloud. If the service account is deleted or updated: The Pub/Sub Topic may fail to authenticate and receive data. If the Topic is updated: The service account remains unaffected.",
			BlastPropagation: gcpshared.ImpactInOnly,
		},
	},
	terraformMapping: gcpshared.TerraformMapping{
		Reference: "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/pubsub_topic",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_GET,
				TerraformQueryMap: "google_pubsub_topic.name",
			},
		},
	},
}.Register()
