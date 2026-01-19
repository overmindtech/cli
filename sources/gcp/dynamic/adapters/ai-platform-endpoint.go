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
var _ = registerableAdapter{
	sdpType: gcpshared.AIPlatformEndpoint,
	meta: gcpshared.AdapterMeta{
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_AI,
		LocationLevel:      gcpshared.ProjectLevel,
		GetEndpointFunc: gcpshared.ProjectLevelEndpointFuncWithSingleQuery(
			"https://aiplatform.googleapis.com/v1/projects/%s/locations/global/endpoints/%s",
		),
		ListEndpointFunc: gcpshared.ProjectLevelListFunc(
			"https://aiplatform.googleapis.com/v1/projects/%s/locations/global/endpoints",
		),
		UniqueAttributeKeys: []string{"endpoints"},
		IAMPermissions:      []string{"aiplatform.endpoints.get", "aiplatform.endpoints.list"},
		PredefinedRole:      "roles/aiplatform.viewer",
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
		"deployedModels.serviceAccount": {
			ToSDPItemType: gcpshared.IAMServiceAccount,
			Description:   "If the service account is deleted or its permissions are updated: The DeployedModel may fail to run or access required resources. If the DeployedModel is updated: The service account remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{
				In:  true,
				Out: false,
			},
		},
		"deployedModels.sharedResources": {
			ToSDPItemType: gcpshared.AIPlatformDeploymentResourcePool,
			Description:   "If the DeploymentResourcePool is deleted or updated: The DeployedModel may fail to run or lose access to shared resources. If the DeployedModel is updated: The DeploymentResourcePool remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{
				In:  true,
				Out: false,
			},
		},
		"deployedModels.privateEndpoints.serviceAttachment": {
			ToSDPItemType: gcpshared.ComputeServiceAttachment,
			Description:   "If the Service Attachment is deleted or updated: The DeployedModel's private endpoint connectivity may be disrupted. If the DeployedModel is updated: The Service Attachment remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{
				In:  true,
				Out: false,
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
		"predictRequestResponseLoggingConfig.bigqueryDestination.outputUri": {
			ToSDPItemType: gcpshared.BigQueryTable,
			Description:   "If the BigQuery Table is deleted or updated, the Endpoint's logging configuration may be affected.",
			BlastPropagation: &sdp.BlastPropagation{
				In: true,
			},
		},
	},
}.Register()
