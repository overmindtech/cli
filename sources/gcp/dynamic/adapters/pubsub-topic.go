package adapters

import (
	"github.com/overmindtech/cli/sdp-go"
	aws "github.com/overmindtech/cli/sources/aws/shared"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
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
		// Settings for ingestion from a data source into this topic.
		"ingestionDataSourceSettings.cloudStorage.bucket": {
			ToSDPItemType:    gcpshared.StorageBucket,
			Description:      "If the Cloud Storage Bucket is deleted or updated: The Pub/Sub Topic may fail to receive data. If the Topic is updated: The bucket remains unaffected.",
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		"ingestionDataSourceSettings.awsKinesis.streamAr": {
			ToSDPItemType:    aws.KinesisStream,
			Description:      "The Kinesis stream ARN to ingest data from.",
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		"ingestionDataSourceSettings.awsKinesis.consumerArn": {
			ToSDPItemType:    aws.KinesisStreamConsumer,
			Description:      "The Kinesis consumer ARN to used for ingestion in Enhanced Fan-Out mode. The consumer must be already created and ready to be used.",
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		"ingestionDataSourceSettings.awsKinesis.awsRoleArn": {
			ToSDPItemType:    aws.IAMRole,
			Description:      "AWS role to be used for Federated Identity authentication with Kinesis.",
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
