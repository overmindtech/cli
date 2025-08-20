package adapters

import (
	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

// Cloud Function (1st/2nd gen) resource.
// Reference: https://cloud.google.com/functions/docs/reference/rest/v2/projects.locations.functions#Function
// GET:  https://cloudfunctions.googleapis.com/v2/projects/{project}/locations/{location}/functions/{function}
// LIST: https://cloudfunctions.googleapis.com/v2/projects/{project}/locations/{location}/functions
// We treat this similar to other location-scoped project resources (e.g. DataformRepository) using Search semantics.
var cloudFunctionAdapter = registerableAdapter{ //nolint:unused
	sdpType: gcpshared.CloudFunctionsFunction,
	meta: gcpshared.AdapterMeta{
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_COMPUTE_APPLICATION,
		Scope:              gcpshared.ScopeProject,
		GetEndpointBaseURLFunc: gcpshared.ProjectLevelEndpointFuncWithTwoQueries(
			"https://cloudfunctions.googleapis.com/v2/projects/%s/locations/%s/functions/%s",
		),
		// Use SearchEndpointFunc since caller supplies a location to enumerate functions
		SearchEndpointFunc: gcpshared.ProjectLevelEndpointFuncWithSingleQuery(
			"https://cloudfunctions.googleapis.com/v2/projects/%s/locations/%s/functions",
		),
		UniqueAttributeKeys: []string{"locations", "functions"},
		IAMPermissions:      []string{"cloudfunctions.functions.get", "cloudfunctions.functions.list"},
		// HEALTH: https://cloud.google.com/compute/docs/reference/rest/v1/globalForwardingRules#Status => state
		// TODO: https://linear.app/overmind/issue/ENG-631/investigate-how-we-can-add-health-status-for-supporting-items
	},
	blastPropagation: map[string]*gcpshared.Impact{
		"kmsKeyName": gcpshared.CryptoKeyImpactInOnly,
		"buildConfig.source.storageSource.bucket": {
			ToSDPITemType:    gcpshared.StorageBucket,
			Description:      "If the Cloud Storage bucket is deleted or misconfigured: Function deployment may fail. If the function changes: The bucket remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
		},
		"buildConfig.sourceProvenance.resolvedStorageSource.bucket": {
			ToSDPITemType:    gcpshared.StorageBucket,
			Description:      "If the Cloud Storage bucket is deleted or misconfigured: Function deployment may fail. If the function changes: The bucket remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
		},
		"buildConfig.workerPool": {
			ToSDPITemType:    gcpshared.RunWorkerPool,
			Description:      "If the Cloud Run Worker Pool is deleted or misconfigured: Function deployment may fail. If the function changes: The worker pool remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
		},
		"buildConfig.dockerRepository": {
			ToSDPITemType:    gcpshared.ArtifactRegistryRepository,
			Description:      "If the Container Repository is deleted or misconfigured: Function deployment may fail. If the function changes: The repository remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
		},
		"buildConfig.serviceAccount": gcpshared.IAMServiceAccountImpactInOnly,
		"serviceConfig.vpcConnector": {
			ToSDPITemType:    gcpshared.VPCAccessConnector,
			Description:      "If the VPC Access Connector is deleted or misconfigured: Function outbound networking may fail. If the function changes: The connector remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
		},
		"serviceConfig.service": {
			ToSDPITemType:    gcpshared.RunService,
			Description:      "If the Cloud Run Service is deleted or misconfigured: Function execution may fail. If the function changes: The service remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
		},
		"serviceConfig.serviceAccountEmail": gcpshared.IAMServiceAccountImpactInOnly,
		"eventTrigger.trigger": {
			ToSDPITemType:    gcpshared.EventarcTrigger,
			Description:      "If the Eventarc Trigger is deleted or misconfigured: Function event handling may fail. If the function changes: The trigger remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
		},
		"eventTrigger.pubsubTopic": {
			ToSDPITemType:    gcpshared.PubSubTopic,
			Description:      "If the Pub/Sub Topic is deleted or misconfigured: Function event handling may fail. If the function changes: The topic remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
		},
		"eventTrigger.serviceAccountEmail": gcpshared.IAMServiceAccountImpactInOnly,
	},
}.Register()
