package adapters

import (
	"fmt"

	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

var cloudResourceManagerTagValueAdapter = registerableAdapter{ //nolint:unused
	sdpType: gcpshared.CloudResourceManagerTagValue,
	meta: gcpshared.AdapterMeta{
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_CONFIGURATION,
		Scope:              gcpshared.ScopeProject,
		// Reference: https://cloud.google.com/resource-manager/reference/rest/v3/tagValues/get
		// GET https://cloudresourcemanager.googleapis.com/v3/tagValues/{TAG_VALUE_ID}
		GetEndpointBaseURLFunc: func(adapterInitParams ...string) (gcpshared.EndpointFunc, error) {
			// Reuse project initialization pattern (even though endpoint itself isn't project-scoped) for consistency.
			if len(adapterInitParams) == 1 && adapterInitParams[0] != "" {
				return func(query string) string {
					if query == "" {
						return ""
					}
					return fmt.Sprintf("https://cloudresourcemanager.googleapis.com/v3/tagValues/%s", query)
				}, nil
			}
			return nil, fmt.Errorf("projectID cannot be empty: %v", adapterInitParams)
		},
		// Reference: https://cloud.google.com/resource-manager/reference/rest/v3/tagValues/list
		// LIST https://cloudresourcemanager.googleapis.com/v3/tagValues?parent=tagKeys/{TAG_KEY_ID}
		SearchEndpointFunc: func(adapterInitParams ...string) (gcpshared.EndpointFunc, error) {
			if len(adapterInitParams) == 1 && adapterInitParams[0] != "" {
				return func(query string) string {
					if query == "" { // require a parent TagKey identifier
						return ""
					}
					return fmt.Sprintf("https://cloudresourcemanager.googleapis.com/v3/tagValues?parent=tagKeys/%s", query)
				}, nil
			}
			return nil, fmt.Errorf("projectID cannot be empty: %v", adapterInitParams)
		},
		SearchDescription:   "Search for TagValues by TagKey.",
		UniqueAttributeKeys: []string{"tagValues"},
		IAMPermissions: []string{
			"resourcemanager.tagValues.get",
			"resourcemanager.tagValues.list",
		},
	},
	blastPropagation: map[string]*gcpshared.Impact{
		"parent": {
			ToSDPItemType: gcpshared.CloudResourceManagerTagKey,
			Description:   "They are tightly coupled",
			BlastPropagation: &sdp.BlastPropagation{
				In:  true,
				Out: true,
			},
		},
	},
	terraformMapping: gcpshared.TerraformMapping{
		Reference: "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/tags_tag_value",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_GET,
				TerraformQueryMap: "google_tags_tag_value.name",
			},
		},
	},
}.Register()
