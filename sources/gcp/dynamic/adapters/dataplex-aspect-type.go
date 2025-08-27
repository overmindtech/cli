package adapters

import (
	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

// Google Cloud Dataplex Aspect Type defines the structure and metadata schema for aspects that can be attached to assets in Dataplex.
// It's part of Google Cloud's data governance and catalog capabilities, allowing users to define custom metadata schemas
// for their data assets within Dataplex lakes and zones.
// Reference: https://cloud.google.com/dataplex/docs/reference/rest/v1/projects.locations.aspectTypes/get
// GET  https://dataplex.googleapis.com/v1/projects/{project}/locations/{location}/aspectTypes/{aspectType}
// LIST https://dataplex.googleapis.com/v1/projects/{project}/locations/{location}/aspectTypes
var dataplexAspectTypeAdapter = registerableAdapter{ //nolint:unused
	sdpType: gcpshared.DataplexAspectType,
	meta: gcpshared.AdapterMeta{
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_CONFIGURATION,
		Scope:              gcpshared.ScopeProject,
		GetEndpointBaseURLFunc: gcpshared.ProjectLevelEndpointFuncWithTwoQueries(
			"https://dataplex.googleapis.com/v1/projects/%s/locations/%s/aspectTypes/%s",
		),
		SearchEndpointFunc: gcpshared.ProjectLevelEndpointFuncWithSingleQuery(
			"https://dataplex.googleapis.com/v1/projects/%s/locations/%s/aspectTypes",
		),
		SearchDescription:   "Search for Dataplex aspect types in a location. Use the format \"location\" or \"projects/project_id/locations/location/aspectTypes/aspect_type_id\" which is supported for terraform mappings.",
		UniqueAttributeKeys: []string{"locations", "aspectTypes"},
		IAMPermissions: []string{
			"dataplex.aspectTypes.get",
			"dataplex.aspectTypes.list",
		},
	},
	blastPropagation: map[string]*gcpshared.Impact{
		// Based on the AspectType structure from the API documentation,
		// AspectTypes typically define metadata schemas and don't have direct dependencies
		// on other GCP resources in their core definition. They are schema definitions
		// rather than runtime resources.
		// If future updates add resource dependencies, they would be added here.
	},
	terraformMapping: gcpshared.TerraformMapping{
		Reference:   "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/dataplex_aspect_type",
		Description: "id => projects/{{project}}/locations/{{location}}/aspectTypes/{{aspect_type_id}}",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_SEARCH,
				TerraformQueryMap: "google_dataplex_aspect_type.id",
			},
		},
	},
}.Register()
