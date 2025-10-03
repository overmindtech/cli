package adapters

import (
	"fmt"

	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

// Org Policy Policy (V2) adapter
// API Get:  https://cloud.google.com/resource-manager/docs/reference/orgpolicy/rest/v2/projects.policies/get
// API List: https://cloud.google.com/resource-manager/docs/reference/orgpolicy/rest/v2/projects.policies/list
// GET  https://orgpolicy.googleapis.com/v2/projects/{project}/policies/{constraint}
// LIST https://orgpolicy.googleapis.com/v2/projects/{project}/policies
var orgPolicyPolicyAdapter = registerableAdapter{ //nolint:unused
	sdpType: gcpshared.OrgPolicyPolicy,
	meta: gcpshared.AdapterMeta{
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_CONFIGURATION,
		Scope:              gcpshared.ScopeProject,
		GetEndpointBaseURLFunc: gcpshared.ProjectLevelEndpointFuncWithSingleQuery(
			"https://orgpolicy.googleapis.com/v2/projects/%s/policies/%s",
		),
		ListEndpointFunc: gcpshared.ProjectLevelListFunc(
			"https://orgpolicy.googleapis.com/v2/projects/%s/policies",
		),
		// Provide a no-op search (same pattern as other adapters) for terraform mapping support.
		SearchEndpointFunc: func(adapterInitParams ...string) (gcpshared.EndpointFunc, error) {
			if len(adapterInitParams) != 1 || adapterInitParams[0] == "" {
				return nil, fmt.Errorf("projectID cannot be empty: %v", adapterInitParams)
			}
			return nil, nil // runtime will use GET with provided full name
		},
		SearchDescription:   "Search with the full policy name: projects/[project]/policies/[constraint] (used for terraform mapping).",
		UniqueAttributeKeys: []string{"policies"},
		IAMPermissions: []string{
			"orgpolicy.policy.get",
			"orgpolicy.policies.list",
		},
		PredefinedRole: "roles/orgpolicy.policyViewer",
	},
	blastPropagation: map[string]*gcpshared.Impact{},
	terraformMapping: gcpshared.TerraformMapping{
		Reference:   "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/org_policy_policy",
		Description: "Use SEARCH with the full policy name: projects/{project}/policies/{constraint}",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_SEARCH,
				TerraformQueryMap: "google_org_policy_policy.name",
			},
		},
	},
}.Register()
