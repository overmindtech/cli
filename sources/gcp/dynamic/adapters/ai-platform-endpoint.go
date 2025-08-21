package adapters

import (
	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

// AI Platform Endpoint adapter.
// GCP Ref (GET): https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.endpoints/get
// GCP Ref (Endpoint schema): https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.endpoints#Endpoint
// GET  https://aiplatform.googleapis.com/v1/projects/{project}/locations/{location}/endpoints/{endpoint}
// LIST https://aiplatform.googleapis.com/v1/projects/{project}/locations/{location}/endpoints
// NOTE: We use "global" for the location in the URL, because we use the global service endpoint.
var aiPlatformEndpointAdapter = registerableAdapter{ //nolint:unused
	sdpType: gcpshared.AIPlatformEndpoint,
	meta: gcpshared.AdapterMeta{
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_AI,
		Scope:              gcpshared.ScopeProject,
		GetEndpointBaseURLFunc: gcpshared.ProjectLevelEndpointFuncWithSingleQuery(
			"https://aiplatform.googleapis.com/v1/projects/%s/locations/global/endpoints/%s",
		),
		ListEndpointFunc: gcpshared.ProjectLevelListFunc(
			"https://aiplatform.googleapis.com/v1/projects/%s/locations/global/endpoints",
		),
		UniqueAttributeKeys: []string{"endpoints"},
		IAMPermissions:      []string{"aiplatform.endpoints.get", "aiplatform.endpoints.list"},
	},
	blastPropagation: map[string]*gcpshared.Impact{
		"encryptionSpec.kmsKeyName": gcpshared.CryptoKeyImpactInOnly,
		"network":                   gcpshared.ComputeNetworkImpactInOnly,
		"deployedModels.model": {
			ToSDPItemType: gcpshared.AIPlatformModel,
			Description:   "They are tightly coupled.",
			BlastPropagation: &sdp.BlastPropagation{
				In:  true,
				Out: true,
			},
		},
		"modelDeploymentMonitoringJob": {
			ToSDPItemType: gcpshared.AIPlatformModelDeploymentMonitoringJob,
			Description:   "They are tightly coupled.",
			BlastPropagation: &sdp.BlastPropagation{
				In:  true,
				Out: true,
			},
		},
		"dedicatedEndpointDns": {
			ToSDPItemType: stdlib.NetworkDNS,
			Description:   "The DNS name for the dedicated endpoint. If the Endpoint is deleted, this DNS name will no longer resolve.",
			BlastPropagation: &sdp.BlastPropagation{
				In:  true,
				Out: true,
			},
		},
		"predictRequestResponseLoggingConfig.bigqueryDestination": {
			ToSDPItemType: gcpshared.BigQueryTable,
			Description:   "If the BigQuery Table is deleted or updated, the Endpoint's logging configuration may be affected.",
			BlastPropagation: &sdp.BlastPropagation{
				In: true,
			},
		},
	},
}.Register()
