package adapters

import (
	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

// Monitoring Alert Policy adapter.
// GCP API Get Reference: https://cloud.google.com/monitoring/api/ref_v3/rest/v3/projects.alertPolicies/get
// GET  https://monitoring.googleapis.com/v3/projects/{project}/alertPolicies/{alert_policy_id}
// LIST https://monitoring.googleapis.com/v3/projects/{project}/alertPolicies
// NOTE: Search is only implemented to support Terraform mapping where the full name
// (projects/{project}/alertPolicies/{policy_id}) may be provided.
var _ = registerableAdapter{
	sdpType: gcpshared.MonitoringAlertPolicy,
	meta: gcpshared.AdapterMeta{
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_OBSERVABILITY,
		LocationLevel:      gcpshared.ProjectLevel,
		GetEndpointFunc: gcpshared.ProjectLevelEndpointFuncWithSingleQuery(
			"https://monitoring.googleapis.com/v3/projects/%s/alertPolicies/%s",
		),
		ListEndpointFunc: gcpshared.ProjectLevelListFunc(
			"https://monitoring.googleapis.com/v3/projects/%s/alertPolicies",
		),
		// Provide a no-op search (same pattern as other adapters) for terraform mapping support.
		// Returns empty URL to trigger GET with the provided full name.
		SearchEndpointFunc: func(query string, location gcpshared.LocationInfo) string {
			return ""
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
		"alertStrategy.notificationChannelStrategy.notificationChannelNames": {
			ToSDPItemType:    gcpshared.MonitoringNotificationChannel,
			Description:      "The notification channels specified in the alert strategy for channel-specific renotification behavior. If these notification channels are deleted, the alert policy will not be able to notify when triggered. If the alert policy is deleted, the notification channels will not be affected.",
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
