package adapters

import (
	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

// Eventarc Trigger adapter (IN DEVELOPMENT)
// Reference: https://cloud.google.com/eventarc/docs/reference/rest/v1/projects.locations.triggers/get
// GET:  https://eventarc.googleapis.com/v1/projects/{project}/locations/{location}/triggers/{trigger}
// LIST: https://eventarc.googleapis.com/v1/projects/{project}/locations/{location}/triggers
var eventarcTriggerAdapter = registerableAdapter{ //nolint:unused
	sdpType: gcpshared.EventarcTrigger,
	meta: gcpshared.AdapterMeta{
		InDevelopment:      true,
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_CONFIGURATION,
		LocationLevel:      gcpshared.ProjectLevel,
		GetEndpointFunc: gcpshared.ProjectLevelEndpointFuncWithTwoQueries(
			"https://eventarc.googleapis.com/v1/projects/%s/locations/%s/triggers/%s",
		),
		// List requires only the location (region or global) besides project.
		SearchEndpointFunc: gcpshared.ProjectLevelEndpointFuncWithSingleQuery(
			"https://eventarc.googleapis.com/v1/projects/%s/locations/%s/triggers",
		),
		UniqueAttributeKeys: []string{"locations", "triggers"},
		IAMPermissions: []string{
			"eventarc.triggers.get",
			"eventarc.triggers.list",
		},
		PredefinedRole: "roles/eventarc.viewer",
	},
	blastPropagation: map[string]*gcpshared.Impact{
		// Service account used by the trigger to invoke the target service
		"serviceAccount": gcpshared.IAMServiceAccountImpactInOnly,
		// Channel associated with the trigger for event delivery
		"channel": {
			ToSDPItemType: gcpshared.EventarcChannel,
			Description:   "If the Eventarc Channel is deleted or updated: The trigger may fail to receive events. If the trigger is updated: The channel remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{
				In:  true,
				Out: false,
			},
		},
		// Cloud Run Service destination
		"destination.cloudRunService.service": {
			ToSDPItemType: gcpshared.RunService,
			Description:   "If the Cloud Run Service is deleted or updated: The trigger may fail to deliver events to the service. If the trigger is updated: The service remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{
				In:  true,
				Out: false,
			},
		},
		// Cloud Function destination (fully qualified resource name)
		"destination.cloudFunction": {
			ToSDPItemType: gcpshared.CloudFunctionsFunction,
			Description:   "If the Cloud Function is deleted or updated: The trigger may fail to deliver events to the function. If the trigger is updated: The function remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{
				In:  true,
				Out: false,
			},
		},
		// GKE Cluster destination
		"destination.gke.cluster": {
			ToSDPItemType: gcpshared.ContainerCluster,
			Description:   "If the GKE Cluster is deleted or updated: The trigger may fail to deliver events to services in the cluster. If the trigger is updated: The cluster remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{
				In:  true,
				Out: false,
			},
		},
		// Workflow destination (fully qualified resource name)
		"destination.workflow": {
			ToSDPItemType: gcpshared.WorkflowsWorkflow,
			Description:   "If the Workflow is deleted or updated: The trigger may fail to invoke the workflow. If the trigger is updated: The workflow remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{
				In:  true,
				Out: false,
			},
		},
		// HTTP endpoint URI (standard library resource)
		"destination.httpEndpoint.uri": {
			ToSDPItemType: stdlib.NetworkHTTP,
			Description:   "If the HTTP endpoint is unavailable or misconfigured: The trigger may fail to deliver events. If the trigger is updated: The HTTP endpoint remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{
				In:  true,
				Out: false,
			},
		},
		// Network Attachment for VPC-internal HTTP endpoints
		"destination.httpEndpoint.networkConfig.networkAttachment": {
			ToSDPItemType: gcpshared.ComputeNetworkAttachment,
			Description:   "If the Network Attachment is deleted or updated: The trigger may fail to access VPC-internal HTTP endpoints. If the trigger is updated: The network attachment remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{
				In:  true,
				Out: false,
			},
		},
		// Pub/Sub Topic used as transport mechanism
		"transport.pubsub.topic": {
			ToSDPItemType: gcpshared.PubSubTopic,
			Description:   "If the Pub/Sub Topic is deleted or updated: The trigger may fail to transport events. If the trigger is updated: The topic remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{
				In:  true,
				Out: false,
			},
		},
		// Pub/Sub Subscription created and managed by Eventarc (output only)
		"transport.pubsub.subscription": {
			ToSDPItemType: gcpshared.PubSubSubscription,
			Description:   "If the Pub/Sub Subscription is deleted or updated: The trigger may fail to receive events from the topic. If the trigger is updated: The subscription may be recreated or updated by Eventarc.",
			BlastPropagation: &sdp.BlastPropagation{
				In:  true,
				Out: true,
			},
		},
	},
}.Register()
