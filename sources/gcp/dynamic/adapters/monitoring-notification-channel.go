package adapters

import (
	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

// Monitoring Notification Channel adapter
// GCP Ref (GET): https://cloud.google.com/monitoring/api/ref_v3/rest/v3/projects.notificationChannels/get
// GCP Ref (Schema): https://cloud.google.com/monitoring/api/ref_v3/rest/v3/projects.notificationChannels#NotificationChannel
// GET  https://monitoring.googleapis.com/v3/projects/{project}/notificationChannels/{notificationChannel}
// LIST https://monitoring.googleapis.com/v3/projects/{project}/notificationChannels
var _ = registerableAdapter{
	sdpType: gcpshared.MonitoringNotificationChannel,
	meta: gcpshared.AdapterMeta{
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_OBSERVABILITY,
		LocationLevel:      gcpshared.ProjectLevel,
		GetEndpointFunc: gcpshared.ProjectLevelEndpointFuncWithSingleQuery(
			"https://monitoring.googleapis.com/v3/projects/%s/notificationChannels/%s",
		),
		ListEndpointFunc: gcpshared.ProjectLevelListFunc(
			"https://monitoring.googleapis.com/v3/projects/%s/notificationChannels",
		),
		// Provide a no-op search (same pattern as other adapters) for terraform mapping support.
		// Returns empty URL to trigger GET with the provided full name.
		SearchEndpointFunc: func(query string, location gcpshared.LocationInfo) string {
			return ""
		},
		SearchDescription:   "Search by full resource name: projects/[project]/notificationChannels/[notificationChannel] (used for terraform mapping).",
		UniqueAttributeKeys: []string{"notificationChannels"},
		IAMPermissions: []string{
			"monitoring.notificationChannels.get",
			"monitoring.notificationChannels.list",
		},
		PredefinedRole: "roles/monitoring.viewer",
	},
	blastPropagation: map[string]*gcpshared.Impact{
		// For pubsub type notification channels, the topic label contains the Pub/Sub topic resource name
		// Format: projects/{project}/topics/{topic}
		"labels.topic": {
			ToSDPItemType:    gcpshared.PubSubTopic,
			Description:      "If the Pub/Sub Topic is deleted or updated: The Notification Channel may fail to send alerts. If the Notification Channel is updated: The topic remains unaffected.",
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		// For webhook_basicauth and webhook_tokenauth type notification channels, the url label contains the HTTP/HTTPS endpoint
		"labels.url": {
			ToSDPItemType:    stdlib.NetworkHTTP,
			Description:      "If the HTTP endpoint is unavailable or updated: The Notification Channel may fail to send alerts. If the Notification Channel is updated: The endpoint remains unaffected.",
			BlastPropagation: gcpshared.ImpactInOnly,
		},
	},
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
