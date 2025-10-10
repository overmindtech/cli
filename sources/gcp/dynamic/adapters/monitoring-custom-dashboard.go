package adapters

import (
	"fmt"

	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

// Monitoring Custom Dashboard adapter for Cloud Monitoring dashboards
var _ = registerableAdapter{
	sdpType: gcpshared.MonitoringCustomDashboard,
	meta: gcpshared.AdapterMeta{
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_OBSERVABILITY,
		Scope:              gcpshared.ScopeProject,
		// Reference: https://cloud.google.com/monitoring/api/ref_v3/rest/v1/projects.dashboards/get
		// GET https://monitoring.googleapis.com/v1/projects/[PROJECT_ID_OR_NUMBER]/dashboards/[DASHBOARD_ID] (for custom dashboards).
		// IAM Perm: monitoring.dashboards.get
		GetEndpointBaseURLFunc: gcpshared.ProjectLevelEndpointFuncWithSingleQuery("https://monitoring.googleapis.com/v1/projects/%s/dashboards/%s"),
		// Reference: https://cloud.google.com/monitoring/api/ref_v3/rest/v1/projects.dashboards/list
		// GET https://monitoring.googleapis.com/v1/{parent}/dashboards
		// IAM Perm: monitoring.dashboards.list
		ListEndpointFunc:  gcpshared.ProjectLevelListFunc("https://monitoring.googleapis.com/v1/projects/%s/dashboards"),
		SearchDescription: "Search for custom dashboards by their ID in the form of \"projects/[project_id]/dashboards/[dashboard_id]\". This is supported for terraform mappings.",
		// This is a special case where we have to define the SEARCH method for only to support Terraform Mapping.
		// We only validate the adapter initiation constraint: whether the project ID is provided or not.
		// We return a nil EndpointFunc without any error, because in the runtime we will use the
		// GET endpoint for retrieving the item for Terraform Query.
		SearchEndpointFunc: func(adapterInitParams ...string) (gcpshared.EndpointFunc, error) {
			if len(adapterInitParams) != 1 || adapterInitParams[0] == "" {
				return nil, fmt.Errorf("projectID cannot be empty: %v", adapterInitParams)
			}

			return nil, nil
		},
		UniqueAttributeKeys: []string{"dashboards"},
		IAMPermissions:      []string{"monitoring.dashboards.get", "monitoring.dashboards.list"},
		PredefinedRole:      "roles/monitoring.viewer",
	},
	blastPropagation: map[string]*gcpshared.Impact{
		// There is no links for this item type.
	},
	terraformMapping: gcpshared.TerraformMapping{
		Reference:   "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/monitoring_dashboard",
		Description: "id => projects/{{project}}/dashboards/{{dashboard_id}}",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_SEARCH,
				TerraformQueryMap: "google_monitoring_dashboard.id",
			},
		},
	},
}.Register()
