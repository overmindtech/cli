package adapters

import (
	"fmt"

	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

// Cloud Resource Manager TagKey adapter (IN DEVELOPMENT)
// Reference: https://cloud.google.com/resource-manager/reference/rest/v3/tagKeys/get
// GET  https://cloudresourcemanager.googleapis.com/v3/tagKeys/{TAG_KEY_ID}
// LIST https://cloudresourcemanager.googleapis.com/v3/tagKeys?parent=projects/{project_id}
var cloudResourceManagerTagKeyAdapter = registerableAdapter{ //nolint:unused
	sdpType: gcpshared.CloudResourceManagerTagKey,
	meta: gcpshared.AdapterMeta{
		InDevelopment:      true,
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_CONFIGURATION,
		LocationLevel:      gcpshared.ProjectLevel,
		GetEndpointFunc: func(query string, location gcpshared.LocationInfo) string {
			if query == "" { // require TagKey identifier (e.g. 123)
				return ""
			}
			return fmt.Sprintf("https://cloudresourcemanager.googleapis.com/v3/tagKeys/%s", query)
		},
		// List TagKeys requires a parent. We accept an organization ID (e.g. 123456789) and construct organizations/{ID}
		ListEndpointFunc:    gcpshared.ProjectLevelListFunc("https://cloudresourcemanager.googleapis.com/v3/tagKeys?parent=projects/%s"),
		UniqueAttributeKeys: []string{"tagKeys"},
		IAMPermissions: []string{
			"resourcemanager.tagKeys.get",
			"resourcemanager.tagKeys.list",
		},
		PredefinedRole: "roles/resourcemanager.tagViewer",
	},
	// No blast propagation yet. TagValue already links back to TagKey via parent attribute.
	blastPropagation: map[string]*gcpshared.Impact{},
}.Register()
