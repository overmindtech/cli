package adapters

import (
	"fmt"

	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

// Public Delegated Prefix (regional) resource.
// Reference: https://cloud.google.com/compute/docs/reference/rest/v1/publicDelegatedPrefixes/get
// GET:  https://compute.googleapis.com/compute/v1/projects/{project}/regions/{region}/publicDelegatedPrefixes/{publicDelegatedPrefix}
// LIST: https://compute.googleapis.com/compute/v1/projects/{project}/regions/{region}/publicDelegatedPrefixes
var computePublicDelegatedPrefixAdapter = registerableAdapter{ //nolint:unused
	sdpType: gcpshared.ComputePublicDelegatedPrefix,
	meta: gcpshared.AdapterMeta{
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
		Scope:              gcpshared.ScopeRegional,
		GetEndpointBaseURLFunc: gcpshared.RegionalLevelEndpointFuncWithSingleQuery(
			"https://compute.googleapis.com/compute/v1/projects/%s/regions/%s/publicDelegatedPrefixes/%s",
		),
		ListEndpointFunc: gcpshared.RegionLevelListFunc(
			"https://compute.googleapis.com/compute/v1/projects/%s/regions/%s/publicDelegatedPrefixes",
		),
		// Provide a no-op search for terraform mapping support with full resource ID.
		// Expected search query: projects/{project}/regions/{region}/publicDelegatedPrefixes/{name}
		SearchEndpointFunc: func(adapterInitParams ...string) (gcpshared.EndpointFunc, error) {
			if len(adapterInitParams) != 2 || adapterInitParams[0] == "" || adapterInitParams[1] == "" {
				return nil, fmt.Errorf("projectID and region cannot be empty: %v", adapterInitParams)
			}
			return nil, nil // runtime will use GET with provided full name
		},
		SearchDescription:   "Search with full ID: projects/{project}/regions/{region}/publicDelegatedPrefixes/{name} (used for terraform mapping).",
		UniqueAttributeKeys: []string{"publicDelegatedPrefixes"},
		IAMPermissions: []string{
			"compute.publicDelegatedPrefixes.get",
			"compute.publicDelegatedPrefixes.list",
		},
		// HEALTH: status (e.g., LIVE/TO_BE_DELETED) may be present on the resource
		// TODO: https://linear.app/overmind/issue/ENG-631
	},
	blastPropagation: map[string]*gcpshared.Impact{
		// Parent Public Advertised Prefix from which this delegated prefix is allocated.
		"parentPrefix": {
			ToSDPItemType:    gcpshared.ComputePublicAdvertisedPrefix,
			Description:      "If the Public Advertised Prefix is updated or deleted: the delegated prefix may become invalid or withdrawn. If the delegated prefix changes: the parent advertised prefix remains structurally unaffected.",
			BlastPropagation: &sdp.BlastPropagation{In: true},
		},
		// Each sub-prefix may be delegated to a specific project.
		"publicDelegatedSubPrefixs.delegateeProject": {
			ToSDPItemType:    gcpshared.CloudResourceManagerProject,
			Description:      "If the delegatee Project is deleted or disabled: usage of the delegated sub-prefix may stop working. If the delegated prefix changes: the project resource remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{In: true},
		},
		"publicDelegatedSubPrefixs.name": {
			ToSDPItemType:    gcpshared.ComputePublicDelegatedPrefix,
			Description:      "If the delegated sub-prefix is updated or deleted: usage of the sub-prefix may stop working. If the parent delegated prefix changes: the sub-prefix remains structurally unaffected.",
			BlastPropagation: &sdp.BlastPropagation{In: true},
		},
	},
	terraformMapping: gcpshared.TerraformMapping{
		Reference:   "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/compute_public_delegated_prefix",
		Description: "id => projects/{{project}}/regions/{{region}}/publicDelegatedPrefixes/{{name}}",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_SEARCH,
				TerraformQueryMap: "google_compute_public_delegated_prefix.id",
			},
		},
	},
}.Register()
