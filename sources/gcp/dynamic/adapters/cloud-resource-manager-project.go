package adapters

import (
	"fmt"

	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

// Cloud Resource Manager Project adapter for GCP projects
var _ = registerableAdapter{
	sdpType: gcpshared.CloudResourceManagerProject,
	meta: gcpshared.AdapterMeta{
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_CONFIGURATION,
		Scope:              gcpshared.ScopeProject,
		// Reference: https://cloud.google.com/resource-manager/reference/rest/v3/projects/get
		// GET https://cloudresourcemanager.googleapis.com/v3/projects/*
		// IAM permissions: resourcemanager.projects.get
		GetEndpointBaseURLFunc: func(adapterInitParams ...string) (gcpshared.EndpointFunc, error) {
			if len(adapterInitParams) == 1 && adapterInitParams[0] != "" {
				return func(query string) string {
					if query != "" {
						return fmt.Sprintf("https://cloudresourcemanager.googleapis.com/v3/projects/%s", query)
					}
					return ""
				}, nil
			}
			return nil, fmt.Errorf("projectID cannot be empty: %v", adapterInitParams)
		},
		UniqueAttributeKeys: []string{"projects"},
		// HEALTH: https://cloud.google.com/resource-manager/reference/rest/v3/projects#State
		// TODO: https://linear.app/overmind/issue/ENG-631/investigate-how-we-can-add-health-status-for-supporting-items
		IAMPermissions: []string{"resourcemanager.projects.get"},
		PredefinedRole: "roles/resourcemanager.tagViewer",
	},
	blastPropagation: map[string]*gcpshared.Impact{
		// There are no links for this item type.
		// TODO: Currently our highest level of scope is the project.
		// This item has `parent` attribute that refers to organization or folder which are higher level scopes that we don't support yet.
		// If we support those scopes in the future, we can add links to them.
		// https://cloud.google.com/resource-manager/reference/rest/v3/projects#Project
	},
	terraformMapping: gcpshared.TerraformMapping{
		Description: "There is no terraform resource for this type.",
	},
}.Register()
