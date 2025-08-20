package adapters

import (
	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

// Cloud Function (1st/2nd gen) resource.
// Reference: https://cloudfunctions.googleapis.com/v1/projects/{project}/locations/{location}/functions/{function}
// GET:  https://cloudfunctions.googleapis.com/v1/projects/{project}/locations/{location}/functions/{function}
// LIST: https://cloudfunctions.googleapis.com/v1/projects/{project}/locations/{location}/functions
// We treat this similar to other location-scoped project resources (e.g. DataformRepository) using Search semantics.
var cloudFunctionAdapter = registerableAdapter{ //nolint:unused
	sdpType: gcpshared.CloudFunctionsFunction,
	meta: gcpshared.AdapterMeta{
		InDevelopment:      true,
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_COMPUTE_APPLICATION,
		Scope:              gcpshared.ScopeProject,
		GetEndpointBaseURLFunc: gcpshared.ProjectLevelEndpointFuncWithTwoQueries(
			"https://cloudfunctions.googleapis.com/v1/projects/%s/locations/%s/functions/%s",
		),
		// Use SearchEndpointFunc since caller supplies a location to enumerate functions
		SearchEndpointFunc: gcpshared.ProjectLevelEndpointFuncWithSingleQuery(
			"https://cloudfunctions.googleapis.com/v1/projects/%s/locations/%s/functions",
		),
		UniqueAttributeKeys: []string{"locations", "functions"},
	},
	blastPropagation: map[string]*gcpshared.Impact{
		// Service account the function executes as.
		"serviceAccountEmail": gcpshared.IAMServiceAccountImpactInOnly,
		"network":             gcpshared.ComputeNetworkImpactInOnly,
		// VPC Connector reference (serverless VPC access).
		"vpcConnector": {
			ToSDPITemType:    gcpshared.VPCAccessConnector,
			Description:      "If the VPC Access Connector is deleted or misconfigured: Function outbound networking may fail. If the function changes: The connector remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
		},
	},
	terraformMapping: gcpshared.TerraformMapping{
		Reference: "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/cloudfunctions_function",
		Mappings: []*sdp.TerraformMapping{{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "google_cloudfunctions_function.name",
		}},
	},
}.Register()
