package adapters

import (
	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

// Security Center Management Security Center Service adapter
// Manages Security Center service configurations for organizations and projects.
// Reference: https://cloud.google.com/security-command-center/docs/reference/security-center-management/rest/v1/projects.locations.securityCenterServices/get
// GET:  https://securitycentermanagement.googleapis.com/v1/projects/{project}/locations/{location}/securityCenterServices/{securityCenterService}
// LIST: https://securitycentermanagement.googleapis.com/v1/projects/{project}/locations/{location}/securityCenterServices
var _ = registerableAdapter{
	sdpType: gcpshared.SecurityCenterManagementSecurityCenterService,
	meta: gcpshared.AdapterMeta{
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_SECURITY,
		Scope:              gcpshared.ScopeProject,
		GetEndpointBaseURLFunc: gcpshared.ProjectLevelEndpointFuncWithTwoQueries(
			"https://securitycentermanagement.googleapis.com/v1/projects/%s/locations/%s/securityCenterServices/%s",
		),
		SearchEndpointFunc: gcpshared.ProjectLevelEndpointFuncWithSingleQuery(
			"https://securitycentermanagement.googleapis.com/v1/projects/%s/locations/%s/securityCenterServices",
		),
		SearchDescription:   "Search Security Center services in a location. Use the format \"location\".",
		UniqueAttributeKeys: []string{"locations", "securityCenterServices"},
		IAMPermissions: []string{
			"securitycentermanagement.securityCenterServices.get",
			"securitycentermanagement.securityCenterServices.list",
		},
		PredefinedRole: "roles/securitycentermanagement.viewer",
		// TODO: https://linear.app/overmind/issue/ENG-631 - check if SecurityCenterService has status/state attribute
	},
	blastPropagation: map[string]*gcpshared.Impact{
		// No blast propagation defined yet.
	},
	terraformMapping: gcpshared.TerraformMapping{
		// No Terraform resource found yet.
	},
}.Register()
