package adapters

import (
	"fmt"

	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

// Monitoring Notification Channel adapter
// GCP Ref (GET): https://cloud.google.com/monitoring/api/ref_v3/rest/v3/projects.notificationChannels/get
// GCP Ref (Schema): https://cloud.google.com/monitoring/api/ref_v3/rest/v3/projects.notificationChannels#NotificationChannel
// GET  https://monitoring.googleapis.com/v3/projects/{project}/notificationChannels/{notificationChannel}
// LIST https://monitoring.googleapis.com/v3/projects/{project}/notificationChannels
var monitoringNotificationChannelAdapter = registerableAdapter{ //nolint:unused
	sdpType: gcpshared.MonitoringNotificationChannel,
	meta: gcpshared.AdapterMeta{
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_OBSERVABILITY,
		Scope:              gcpshared.ScopeProject,
		GetEndpointBaseURLFunc: gcpshared.ProjectLevelEndpointFuncWithSingleQuery(
			"https://monitoring.googleapis.com/v3/projects/%s/notificationChannels/%s",
		),
		ListEndpointFunc: gcpshared.ProjectLevelListFunc(
			"https://monitoring.googleapis.com/v3/projects/%s/notificationChannels",
		),
		// Provide a no-op search (same pattern as other adapters) for terraform mapping support.
		SearchEndpointFunc: func(adapterInitParams ...string) (gcpshared.EndpointFunc, error) {
			if len(adapterInitParams) != 1 || adapterInitParams[0] == "" {
				return nil, fmt.Errorf("projectID cannot be empty: %v", adapterInitParams)
			}
			return nil, nil // runtime will use GET with provided full name
		},
		SearchDescription:   "Search by full resource name: projects/[project]/notificationChannels/[notificationChannel] (used for terraform mapping).",
		UniqueAttributeKeys: []string{"notificationChannels"},
		IAMPermissions: []string{
			"monitoring.notificationChannels.get",
			"monitoring.notificationChannels.list",
		},
		PredefinedRole: "roles/monitoring.viewer",
	},
	// No blast propagation defined for this adapter
	blastPropagation: map[string]*gcpshared.Impact{},
	terraformMapping: gcpshared.TerraformMapping{
		Reference:   "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/monitoring_notification_channel",
		Description: "id => projects/{{project}}/notificationChannels/{{notificationChannel}}",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_GET,
				TerraformQueryMap: "google_monitoring_notification_channel.name",
			},
		},
	},
}.Register()
