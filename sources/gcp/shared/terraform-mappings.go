package shared

import (
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources/shared"
)

type TerraformMapping struct {
	Reference   string
	Description string
	Mappings    []*sdp.TerraformMapping
}

var SDPAssetTypeToTerraformMappings = map[shared.ItemType]TerraformMapping{
	AIPlatformCustomJob: {
		Description: "There is no terraform resource for this type.",
	},
	AIPlatformPipelineJob: {
		Description: "There is no terraform resource for this type.",
	},
	ArtifactRegistryDockerImage: {
		Reference:   "https://registry.terraform.io/providers/hashicorp/google/latest/docs/data-sources/artifact_registry_docker_image",
		Description: "name => projects/{{project}}/locations/{{location}}/repository/{{repository_id}}/dockerImages/{{docker_image}}. We should use search to extract relevant fields.",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_SEARCH,
				TerraformQueryMap: "google_artifact_registry_docker_image.name",
			},
		},
	},
	BigQueryDataset: {
		Reference: "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/bigquery_dataset",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_GET,
				TerraformQueryMap: "google_bigquery_dataset.dataset_id",
			},
		},
	},
	BigQueryTable: {
		Reference:   "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/bigquery_table",
		Description: "id => projects/{{project}}/datasets/{{dataset}}/tables/{{table}}",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_SEARCH,
				TerraformQueryMap: "google_bigquery_table.id",
			},
		},
	},
	BigTableAdminAppProfile: {
		Reference:   "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/bigtable_app_profile",
		Description: "id => projects/{{project}}/instances/{{instance}}/appProfiles/{{app_profile_id}}",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_SEARCH,
				TerraformQueryMap: "google_bigtable_app_profile.id",
			},
		},
	},
	BigTableAdminBackup: {
		Description: "There is no terraform resource for this type.",
	},
	BigTableAdminTable: {
		Reference:   "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/bigtable_table",
		Description: "id => projects/{{project}}/instances/{{instance_name}}/tables/{{name}}",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_SEARCH,
				TerraformQueryMap: "google_bigtable_table.id",
			},
		},
	},
	CloudBuildBuild: {
		Description: "There is no terraform resource for this type.",
	},
	CloudBillingBillingInfo: {
		Description: "There is no terraform resource for this type.",
	},
	CloudResourceManagerProject: {
		Description: "There is no terraform resource for this type.",
	},
	ComputeFirewall: {
		Reference: "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/compute_firewall",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_GET,
				TerraformQueryMap: "google_compute_firewall.name",
			},
		},
	},
	ComputeInstance: {
		Reference: "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/compute_instance",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_GET,
				TerraformQueryMap: "google_compute_instance.name",
			},
		},
	},
	ComputeInstanceTemplate: {
		Reference: "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/compute_instance_template",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_GET,
				TerraformQueryMap: "google_compute_instance_template.name",
			},
		},
	},
	ComputeNetwork: {
		Reference: "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/compute_network",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_GET,
				TerraformQueryMap: "google_compute_network.name",
			},
		},
	},
	ComputeProject: {
		Description: "There is no terraform resource for this type.",
	},
	ComputeRoute: {
		Reference: "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/compute_route",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_GET,
				TerraformQueryMap: "google_compute_route.name",
			},
		},
	},
	ComputeSubnetwork: {
		Reference: "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/compute_subnetwork",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_GET,
				TerraformQueryMap: "google_compute_subnetwork.name",
			},
		},
	},
	DataformRepository: {
		Reference:   "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/dataform_repository",
		Description: "id => projects/{{project}}/locations/{{region}}/repositories/{{name}}",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_SEARCH,
				TerraformQueryMap: "google_dataform_repository.id",
			},
		},
	},
	DataplexEntryGroup: {
		Reference:   "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/dataplex_entry_group#entry_group_id",
		Description: "id => projects/{{project}}/locations/{{location}}/entryGroups/{{entry_group_id}}",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_SEARCH,
				TerraformQueryMap: "google_dataplex_entry_group.id",
			},
		},
	},
	DNSManagedZone: {
		Reference: "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/dns_managed_zone#name",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_GET,
				TerraformQueryMap: "google_dns_managed_zone.name",
			},
		},
	},
	EssentialContactsContact: {
		Reference:   "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/essential_contacts_contact#email",
		Description: "id => {resourceType}/{resource_id}/contacts/{contact_id}",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_SEARCH,
				TerraformQueryMap: "google_essential_contacts_contact.id",
			},
		},
	},
	IAMRole: {
		Description: "There is no terraform resource for this type.",
	},
	LoggingBucket: {
		Description: "There is no terraform resource for this type.",
	},
	LoggingLink: {
		Description: "There is no terraform resource for this type.",
	},
	LoggingSavedQuery: {
		Description: "There is no terraform resource for this type.",
	},
	LoggingSink: {
		Reference: "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/logging_project_sink",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_GET,
				TerraformQueryMap: "google_logging_project_sink.name",
			},
		},
	},
	MonitoringCustomDashboard: {
		Reference:   "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/monitoring_dashboard",
		Description: "id => projects/{{project}}/dashboards/{{dashboard_id}}",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_SEARCH,
				TerraformQueryMap: "google_monitoring_dashboard.id",
			},
		},
	},
	PubSubSubscription: {
		Reference: "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/pubsub_subscription",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_GET,
				TerraformQueryMap: "google_pubsub_subscription.name",
			},
		},
	},
	PubSubTopic: {
		Reference: "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/pubsub_topic",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_GET,
				TerraformQueryMap: "google_pubsub_topic.name",
			},
		},
	},
	RunRevision: {
		Description: "There is no terraform resource for this type.",
	},
	ServiceDirectoryEndpoint: {
		Reference:   "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/service_directory_endpoint",
		Description: "id => projects/*/locations/*/namespaces/*/services/*/endpoints/*",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_SEARCH,
				TerraformQueryMap: "google_service_directory_endpoint.id",
			},
		},
	},
	ServiceUsageService: {
		Description: "There is no terraform resource for this type.",
	},
	SpannerDatabase: {
		Reference: "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/spanner_database.html",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_GET,
				TerraformQueryMap: "google_spanner_database.name",
			},
		},
	},
	SQLAdminBackup: {
		Description: "There is no terraform resource for this type.",
	},
	SQLAdminBackupRun: {
		Description: "There is no terraform resource for this type.",
	},
	StorageBucket: {
		Reference: "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/storage_bucket",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_GET,
				TerraformQueryMap: "google_storage_bucket.name",
			},
		},
	},
}
