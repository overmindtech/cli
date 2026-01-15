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
		// Link to parent resource (project, folder, or organization) from name field
		// The name field format is: projects/{project}/locations/{location}/securityCenterServices/{service}
		// or: folders/{folder}/locations/{location}/securityCenterServices/{service}
		// or: organizations/{organization}/locations/{location}/securityCenterServices/{service}
		// The manual linker registered for CloudResourceManagerProject will detect the type based on the name prefix
		// and create the appropriate link to Project, Folder, or Organization
		"name": {
			Description:      "If the parent Project, Folder, or Organization is deleted or updated: The Security Center Service may become invalid or inaccessible. If the Security Center Service is updated: The parent resource remains unaffected.",
			ToSDPItemType:    gcpshared.CloudResourceManagerProject, // Manual linker handles detection of project/folder/organization from name prefix
			BlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
		},
		// Note: Custom modules (SecurityHealthAnalyticsCustomModule, EventThreatDetectionCustomModule, etc.)
		// are not direct children in the API path structure - they are sibling resources under the same
		// project/location scope. They don't have a direct reference field in SecurityCenterService,
		// so we don't link to them here. They would be discovered through their own adapters.
	},
	terraformMapping: gcpshared.TerraformMapping{
		// No Terraform resource found yet.
	},
}.Register()
