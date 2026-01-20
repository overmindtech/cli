package adapters

import (
	"fmt"

	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

// Cloud Billing Billing Info adapter for project billing information
var _ = registerableAdapter{
	sdpType: gcpshared.CloudBillingBillingInfo,
	meta: gcpshared.AdapterMeta{
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_CONFIGURATION,
		LocationLevel:      gcpshared.ProjectLevel,
		// Reference: https://cloud.google.com/billing/docs/reference/rest/v1/projects/getBillingInfo
		// Gets the billing information for a project.
		// GET https://cloudbilling.googleapis.com/v1/{name=projects/*}/billingInfo
		// IAM permissions: resourcemanager.projects.get
		// Note: This adapter uses the query as the project ID, and validates it
		// against the adapter's configured project via location.ProjectID.
		GetEndpointFunc: func(query string, location gcpshared.LocationInfo) string {
			if query == "" {
				return ""
			}
			if query != location.ProjectID {
				return ""
			}
			return fmt.Sprintf("https://cloudbilling.googleapis.com/v1/projects/%s/billingInfo", query)
		},
		UniqueAttributeKeys: []string{"billingInfo"},
		IAMPermissions:      []string{"resourcemanager.projects.get"},
		// This role is required via ai adapters and it gives this exact permission.
		PredefinedRole: "roles/aiplatform.viewer",
	},
	blastPropagation: map[string]*gcpshared.Impact{
		"projectId": {
			ToSDPItemType:    gcpshared.CloudResourceManagerProject,
			Description:      "If the Cloud Resource Manager Project is deleted or updated: The billing information may become invalid or inaccessible. If the billing info is updated: The project remains unaffected.",
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		"billingAccountName": {
			ToSDPItemType: gcpshared.CloudBillingBillingAccount,
			Description:   "If the Cloud Billing Billing Account is deleted or updated: The billing information may become invalid or inaccessible. If the billing info is updated: The billing account is impacted as well.",
			BlastPropagation: &sdp.BlastPropagation{
				In:  true,
				Out: true,
			},
		},
	},
	terraformMapping: gcpshared.TerraformMapping{
		Description: "There is no terraform resource for this type.",
	},
}.Register()
