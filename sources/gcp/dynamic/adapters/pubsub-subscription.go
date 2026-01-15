package adapters

import (
	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

// Pub/Sub Subscription adapter for Google Cloud Pub/Sub subscriptions
var _ = registerableAdapter{
	sdpType: gcpshared.PubSubSubscription,
	meta: gcpshared.AdapterMeta{
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_CONFIGURATION,
		Scope:              gcpshared.ScopeProject,
		// https://pubsub.googleapis.com/v1/projects/{project}/subscriptions/{subscription}
		GetEndpointBaseURLFunc: gcpshared.ProjectLevelEndpointFuncWithSingleQuery("https://pubsub.googleapis.com/v1/projects/%s/subscriptions/%s"),
		// Reference: https://cloud.google.com/pubsub/docs/reference/rest/v1/projects.subscriptions/list?rep_location=global
		// https://pubsub.googleapis.com/v1/projects/{project}/subscriptions
		ListEndpointFunc:    gcpshared.ProjectLevelListFunc("https://pubsub.googleapis.com/v1/projects/%s/subscriptions"),
		UniqueAttributeKeys: []string{"subscriptions"},
		// HEALTH: https://cloud.google.com/pubsub/docs/reference/rest/v1/projects.subscriptions#state_2
		// TODO: https://linear.app/overmind/issue/ENG-631/investigate-how-we-can-add-health-status-for-supporting-items
		IAMPermissions: []string{"pubsub.subscriptions.get", "pubsub.subscriptions.list"},
		PredefinedRole: "roles/pubsub.viewer",
	},
	blastPropagation: map[string]*gcpshared.Impact{
		"topic": {
			ToSDPItemType:    gcpshared.PubSubTopic,
			Description:      "If the Pub/Sub Topic is deleted or updated: The Subscription may fail to receive messages. If the Subscription is updated: The topic remains unaffected.",
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		"deadLetterPolicy.deadLetterTopic": {
			ToSDPItemType:    gcpshared.PubSubTopic,
			Description:      "If the Dead Letter Topic is deleted or updated: The Subscription may fail to deliver failed messages. If the Subscription is updated: The dead letter topic remains unaffected.",
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		"pushConfig.pushEndpoint": {
			ToSDPItemType:    stdlib.NetworkHTTP,
			Description:      "If the HTTP push endpoint is unavailable or updated: The Subscription may fail to deliver messages via push. If the Subscription is updated: The endpoint remains unaffected.",
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		"pushConfig.oidcToken.serviceAccountEmail": gcpshared.IAMServiceAccountImpactInOnly,
		"bigqueryConfig.table": {
			// The name of the table to which to write data, of the form {projectId}.{datasetId}.{tableId}
			// We have a manual adapter for this.
			ToSDPItemType:    gcpshared.BigQueryTable,
			Description:      "If the BigQuery Table is deleted or updated: The Subscription may fail to write data. If the Subscription is updated: The table remains unaffected.",
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		"bigqueryConfig.serviceAccountEmail": gcpshared.IAMServiceAccountImpactInOnly,
		"cloudStorageConfig.bucket": {
			ToSDPItemType:    gcpshared.StorageBucket,
			Description:      "If the Cloud Storage Bucket is deleted or updated: The Subscription may fail to write data. If the Subscription is updated: The bucket remains unaffected.",
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		"cloudStorageConfig.serviceAccountEmail": gcpshared.IAMServiceAccountImpactInOnly,
		"analyticsHubSubscriptionInfo.subscription": {
			ToSDPItemType:    gcpshared.PubSubSubscription,
			Description:      "If the Pub/Sub Subscription is deleted or updated: The Analytics Hub Subscription may fail to receive messages. If the Analytics Hub Subscription is updated: The Pub/Sub Subscription remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{Out: true},
		},
	},
	terraformMapping: gcpshared.TerraformMapping{
		Reference: "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/pubsub_subscription",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_GET,
				TerraformQueryMap: "google_pubsub_subscription.name",
			},
		},
	},
}.Register()
