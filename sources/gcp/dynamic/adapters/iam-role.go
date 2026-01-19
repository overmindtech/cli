package adapters

import (
	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

// IAM Role adapter for custom IAM roles
var _ = registerableAdapter{
	sdpType: gcpshared.IAMRole,
	meta: gcpshared.AdapterMeta{
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_SECURITY,
		LocationLevel:      gcpshared.ProjectLevel,
		// Reference: https://cloud.google.com/iam/docs/reference/rest/v1/roles/get
		// https://iam.googleapis.com/v1/projects/{PROJECT_ID}/roles/{CUSTOM_ROLE_ID}
		GetEndpointFunc: gcpshared.ProjectLevelEndpointFuncWithSingleQuery("https://iam.googleapis.com/v1/projects/%s/roles/%s"),
		// Reference: https://cloud.google.com/iam/docs/reference/rest/v1/roles/list
		// https://iam.googleapis.com/v1/projects/{PROJECT_ID}/roles
		ListEndpointFunc:    gcpshared.ProjectLevelListFunc("https://iam.googleapis.com/v1/projects/%s/roles"),
		UniqueAttributeKeys: []string{"roles"},
		IAMPermissions:      []string{"iam.roles.get", "iam.roles.list"},
		PredefinedRole:      "roles/iam.roleViewer",
	},
	blastPropagation: map[string]*gcpshared.Impact{
		// There is no links for this item type.
	},
	terraformMapping: gcpshared.TerraformMapping{
		Description: "There is no terraform resource for this type.",
	},
}.Register()
