package adapters

import (
	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

// Compute License adapter for software licenses
var _ = registerableAdapter{
	sdpType: gcpshared.ComputeLicense,
	meta: gcpshared.AdapterMeta{
		InDevelopment: true,
		// Reference: https://cloud.google.com/compute/docs/reference/rest/v1/licenses/get
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_CONFIGURATION,
		Scope:              gcpshared.ScopeProject,
		// https://compute.googleapis.com/compute/v1/projects/{project}/global/licenses/{license}
		GetEndpointBaseURLFunc: gcpshared.ProjectLevelEndpointFuncWithSingleQuery("https://compute.googleapis.com/compute/v1/projects/%s/global/licenses/%s"),
		// https://compute.googleapis.com/compute/v1/projects/{project}/global/licenses
		ListEndpointFunc:    gcpshared.ProjectLevelListFunc("https://compute.googleapis.com/compute/v1/projects/%s/global/licenses"),
		UniqueAttributeKeys: []string{"licenses"},
		// compute.licenses.list is only supported at TESTING stage.
		// Which means it can behave unexpectedly, and not recommended for production use.
		// https://cloud.google.com/iam/docs/custom-roles-permissions-support
		// TODO: Decide whether to support this officially or not.
		IAMPermissions: []string{"compute.licenses.get", "compute.licenses.list"},
		PredefinedRole: "roles/compute.viewer",
	},
	blastPropagation: map[string]*gcpshared.Impact{},
	terraformMapping: gcpshared.TerraformMapping{
		Description: "There is no terraform resource for this type.",
	},
}.Register()
