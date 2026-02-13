package adapters

import (
	"github.com/overmindtech/workspace/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

// Pub/Sub Subscription adapter for Google Cloud Pub/Sub subscriptions
var _ = registerableAdapter{
	sdpType: gcpshared.PubSubSubscription,
	meta: gcpshared.AdapterMeta{
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_CONFIGURATION,
		LocationLevel:      gcpshared.ProjectLevel,
		// https://pubsub.googleapis.com/v1/projects/{project}/subscriptions/{subscription}
		GetEndpointFunc: gcpshared.ProjectLevelEndpointFuncWithSingleQuery("https://pubsub.googleapis.com/v1/projects/%s/subscriptions/%s"),
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
		},
		"deadLetterPolicy.deadLetterTopic": {
			ToSDPItemType:    gcpshared.PubSubTopic,
			Description:      "If the Dead Letter Topic is deleted or updated: The Subscription may fail to deliver failed messages. If the Subscription is updated: The dead letter topic remains unaffected.",
		},
		"pushConfig.pushEndpoint": {
			ToSDPItemType:    stdlib.NetworkHTTP,
			Description:      "If the HTTP push endpoint is unavailable or updated: The Subscription may fail to deliver messages via push. If the Subscription is updated: The endpoint remains unaffected.",
		},
		"pushConfig.oidcToken.serviceAccountEmail": gcpshared.IAMServiceAccountImpactInOnly,
		"bigqueryConfig.table": {
			// The name of the table to which to write data, of the form {projectId}.{datasetId}.{tableId}
			// We have a manual adapter for this.
			ToSDPItemType:    gcpshared.BigQueryTable,
			Description:      "If the BigQuery Table is deleted or updated: The Subscription may fail to write data. If the Subscription is updated: The table remains unaffected.",
		},
		"bigqueryConfig.serviceAccountEmail": gcpshared.IAMServiceAccountImpactInOnly,
		"cloudStorageConfig.bucket": {
			ToSDPItemType:    gcpshared.StorageBucket,
			Description:      "If the Cloud Storage Bucket is deleted or updated: The Subscription may fail to write data. If the Subscription is updated: The bucket remains unaffected.",
		},
		"cloudStorageConfig.serviceAccountEmail": gcpshared.IAMServiceAccountImpactInOnly,
		"analyticsHubSubscriptionInfo.subscription": {
			ToSDPItemType:    gcpshared.PubSubSubscription,
			Description:      "If the Pub/Sub Subscription is deleted or updated: The Analytics Hub Subscription may fail to receive messages. If the Analytics Hub Subscription is updated: The Pub/Sub Subscription remains unaffected.",
		},
	},
	terraformMapping: gcpshared.TerraformMapping{
		Reference: "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/pubsub_subscription",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_GET,
				TerraformQueryMap: "google_pubsub_subscription.name",
			},
			// IAM resources for Pub/Sub Subscriptions. These are Terraform-only
			// constructs (no standalone GCP API resource exists for them). When an
			// IAM binding/member/policy changes in a Terraform plan, we resolve it
			// to the parent subscription so that blast radius analysis can show the
			// downstream impact of the access change.
			//
			// Reference: https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/pubsub_subscription_iam
			{
				// Authoritative for a given role — grants the role to a list of members.
				TerraformMethod:   sdp.QueryMethod_GET,
				TerraformQueryMap: "google_pubsub_subscription_iam_binding.subscription",
			},
			{
				// Non-authoritative — grants a single member a single role.
				TerraformMethod:   sdp.QueryMethod_GET,
				TerraformQueryMap: "google_pubsub_subscription_iam_member.subscription",
			},
			{
				// Authoritative for the entire IAM policy on the subscription.
				TerraformMethod:   sdp.QueryMethod_GET,
				TerraformQueryMap: "google_pubsub_subscription_iam_policy.subscription",
			},
		},
	},
}.Register()
