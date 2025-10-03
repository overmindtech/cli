package adapters

import (
	"fmt"

	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

// Monitoring Alert Policy adapter.
// GCP API Get Reference: https://cloud.google.com/monitoring/api/ref_v3/rest/v3/projects.alertPolicies/get
// GET  https://monitoring.googleapis.com/v3/projects/{project}/alertPolicies/{alert_policy_id}
// LIST https://monitoring.googleapis.com/v3/projects/{project}/alertPolicies
// NOTE: Search is only implemented to support Terraform mapping where the full name
// (projects/{project}/alertPolicies/{policy_id}) may be provided.
var monitoringAlertPolicyAdapter = registerableAdapter{ //nolint:unused
	sdpType: gcpshared.MonitoringAlertPolicy,
	meta: gcpshared.AdapterMeta{
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_OBSERVABILITY,
		Scope:              gcpshared.ScopeProject,
		GetEndpointBaseURLFunc: gcpshared.ProjectLevelEndpointFuncWithSingleQuery(
			"https://monitoring.googleapis.com/v3/projects/%s/alertPolicies/%s",
		),
		ListEndpointFunc: gcpshared.ProjectLevelListFunc(
			"https://monitoring.googleapis.com/v3/projects/%s/alertPolicies",
		),
		// Provide a no-op search (same pattern as other adapters) for terraform mapping support.
		SearchEndpointFunc: func(adapterInitParams ...string) (gcpshared.EndpointFunc, error) {
			if len(adapterInitParams) != 1 || adapterInitParams[0] == "" {
				return nil, fmt.Errorf("projectID cannot be empty: %v", adapterInitParams)
			}
			return nil, nil // runtime will use GET with provided full name
		},
		SearchDescription:   "Search by full resource name: projects/[project]/alertPolicies/[alert_policy_id] (used for terraform mapping).",
		UniqueAttributeKeys: []string{"alertPolicies"},
		IAMPermissions: []string{
			"monitoring.alertPolicies.get",
			"monitoring.alertPolicies.list",
		},
		PredefinedRole: "roles/monitoring.viewer",
	},
	blastPropagation: map[string]*gcpshared.Impact{
		"notificationChannels": {
			ToSDPItemType:    gcpshared.MonitoringNotificationChannel,
			Description:      "The notification channels that are used to notify when this alert policy is triggered. If notification channels are deleted, the alert policy will not be able to notify when triggered. If the alert policy is deleted, the notification channels will not be affected.",
			BlastPropagation: &sdp.BlastPropagation{In: true},
		},
	},
	terraformMapping: gcpshared.TerraformMapping{
		Reference:   "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/monitoring_alert_policy",
		Description: "id => projects/{{project}}/alertPolicies/{{alert_policy_id}}",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_SEARCH,
				TerraformQueryMap: "google_monitoring_alert_policy.id",
			},
		},
	},
}.Register()
