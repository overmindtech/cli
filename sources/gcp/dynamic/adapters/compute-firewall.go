package adapters

import (
	"fmt"
	"strings"

	"github.com/overmindtech/cli/go/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

// Compute Firewall adapter for VPC firewall rules
var _ = registerableAdapter{
	sdpType: gcpshared.ComputeFirewall,
	meta: gcpshared.AdapterMeta{
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
		LocationLevel:      gcpshared.ProjectLevel,
		// https://compute.googleapis.com/compute/v1/projects/{project}/global/firewalls/{firewall}
		GetEndpointFunc: gcpshared.ProjectLevelEndpointFuncWithSingleQuery("https://compute.googleapis.com/compute/v1/projects/%s/global/firewalls/%s"),
		// Reference: https://cloud.google.com/compute/docs/reference/rest/v1/firewalls/list
		// https://compute.googleapis.com/compute/v1/projects/{project}/global/firewalls
		ListEndpointFunc:    gcpshared.ProjectLevelListFunc("https://compute.googleapis.com/compute/v1/projects/%s/global/firewalls"),
		UniqueAttributeKeys: []string{"firewalls"},
		IAMPermissions:      []string{"compute.firewalls.get", "compute.firewalls.list"},
		PredefinedRole:      "roles/compute.viewer",
		// Tag-based SEARCH: list all firewalls then filter by tag.
		SearchEndpointFunc: func(query string, location gcpshared.LocationInfo) string {
			if query == "" || strings.Contains(query, "/") {
				return ""
			}
			return fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/global/firewalls", location.ProjectID)
		},
		SearchDescription: "Search for firewalls by network tag. The query is a plain network tag name.",
		SearchFilterFunc:  firewallTagFilter,
	},
	linkRules: map[string]*gcpshared.Impact{
		"network": {
			Description:   "If the Compute Network is updated: The firewall rules may no longer apply correctly. If the firewall is updated: The network remains unaffected, but its security posture may change.",
			ToSDPItemType: gcpshared.ComputeNetwork,
		},
		"sourceServiceAccounts": gcpshared.IAMServiceAccountImpactInOnly,
		"targetServiceAccounts": gcpshared.IAMServiceAccountImpactInOnly,
		"targetTags": {
			Description:   "Firewall rule specifies target_tags to control traffic to VM instances and instance templates with those tags. Overmind automatically discovers these relationships by searching for instances and templates with matching network tags, enabling accurate blast radius analysis when tags change on either firewalls or instances.",
			ToSDPItemType: gcpshared.ComputeInstance,
		},
		"sourceTags": {
			Description:   "Firewall rule specifies source_tags to control traffic from VM instances with those tags. Overmind automatically discovers these relationships by searching for instances with matching network tags, enabling accurate blast radius analysis when tags change on either firewalls or instances.",
			ToSDPItemType: gcpshared.ComputeInstance,
		},
	},
	terraformMapping: gcpshared.TerraformMapping{
		Reference: "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/compute_firewall",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_GET,
				TerraformQueryMap: "google_compute_firewall.name",
			},
		},
	},
}.Register()

// firewallTagFilter keeps firewalls whose targetTags or sourceTags contain the query tag.
func firewallTagFilter(query string, item *sdp.Item) bool {
	return itemAttributeContainsTag(item, "targetTags", query) ||
		itemAttributeContainsTag(item, "sourceTags", query)
}

// itemAttributeContainsTag checks whether an item attribute (expected to be a
// list of strings) contains the given tag value.
func itemAttributeContainsTag(item *sdp.Item, attrKey, tag string) bool {
	val, err := item.GetAttributes().Get(attrKey)
	if err != nil {
		return false
	}
	list, ok := val.([]any)
	if !ok {
		return false
	}
	for _, elem := range list {
		if s, ok := elem.(string); ok && s == tag {
			return true
		}
	}
	return false
}
