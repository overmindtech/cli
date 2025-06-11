package dynamic

import (
	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

type TerraformMapping struct {
	Reference   string
	Description string
	Mappings    []*sdp.TerraformMapping
}

var SDPAssetTypeToTerraformMappings = map[shared.ItemType]TerraformMapping{
	gcpshared.AIPlatformCustomJob: {
		Description: "There is no terraform resource for this type.",
	},
	gcpshared.AIPlatformPipelineJob: {
		Description: "There is no terraform resource for this type.",
	},
	gcpshared.ArtifactRegistryDockerImage: {
		Reference:   "https://registry.terraform.io/providers/hashicorp/google/latest/docs/data-sources/artifact_registry_docker_image",
		Description: "name => projects/{{project}}/locations/{{location}}/repository/{{repository_id}}/dockerImages/{{docker_image}}. We should use search to extract relevant fields.",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_SEARCH,
				TerraformQueryMap: "google_artifact_registry_docker_image.name",
			},
		},
	},
	gcpshared.BigQueryDataset: {
		Reference: "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/bigquery_dataset",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_GET,
				TerraformQueryMap: "google_bigquery_dataset.dataset_id",
			},
		},
	},
	gcpshared.BigQueryTable: {
		Reference:   "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/bigquery_table",
		Description: "id => projects/{{project}}/datasets/{{dataset}}/tables/{{table}}",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_SEARCH,
				TerraformQueryMap: "google_bigquery_table.id",
			},
		},
	},
	gcpshared.BigTableAdminAppProfile: {
		Reference:   "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/bigtable_app_profile",
		Description: "id => projects/{{project}}/instances/{{instance}}/appProfiles/{{app_profile_id}}",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_SEARCH,
				TerraformQueryMap: "google_bigtable_app_profile.id",
			},
		},
	},
	gcpshared.BigTableAdminBackup: {
		Description: "There is no terraform resource for this type.",
	},
	gcpshared.BigTableAdminTable: {
		Reference:   "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/bigtable_table",
		Description: "id => projects/{{project}}/instances/{{instance_name}}/tables/{{name}}",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_SEARCH,
				TerraformQueryMap: "google_bigtable_table.id",
			},
		},
	},
	gcpshared.CloudBuildBuild: {
		Description: "There is no terraform resource for this type.",
	},
	gcpshared.CloudBillingBillingInfo: {
		Description: "There is no terraform resource for this type.",
	},
	gcpshared.CloudResourceManagerProject: {
		Description: "There is no terraform resource for this type.",
	},
	gcpshared.ComputeFirewall: {
		Reference: "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/compute_firewall",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_GET,
				TerraformQueryMap: "google_compute_firewall.name",
			},
		},
	},
	gcpshared.ComputeInstance: {
		Reference: "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/compute_instance",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_GET,
				TerraformQueryMap: "google_compute_instance.name",
			},
		},
	},
	gcpshared.ComputeInstanceTemplate: {
		Reference: "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/compute_instance_template",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_GET,
				TerraformQueryMap: "google_compute_instance_template.name",
			},
		},
	},
	gcpshared.ComputeNetwork: {
		Reference: "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/compute_network",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_GET,
				TerraformQueryMap: "google_compute_network.name",
			},
		},
	},
	gcpshared.ComputeProject: {
		Description: "There is no terraform resource for this type.",
	},
	gcpshared.ComputeRoute: {
		Reference: "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/compute_route",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_GET,
				TerraformQueryMap: "google_compute_route.name",
			},
		},
	},
	gcpshared.ComputeSubnetwork: {
		Reference: "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/compute_subnetwork",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_GET,
				TerraformQueryMap: "google_compute_subnetwork.name",
			},
		},
	},
	gcpshared.DataformRepository: {
		Reference:   "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/dataform_repository",
		Description: "id => projects/{{project}}/locations/{{region}}/repositories/{{name}}",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_SEARCH,
				TerraformQueryMap: "google_dataform_repository.id",
			},
		},
	},
	gcpshared.DataplexEntryGroup: {
		Reference:   "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/dataplex_entry_group#entry_group_id",
		Description: "id => projects/{{project}}/locations/{{location}}/entryGroups/{{entry_group_id}}",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_SEARCH,
				TerraformQueryMap: "google_dataplex_entry_group.id",
			},
		},
	},
	gcpshared.DNSManagedZone: {
		Reference: "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/dns_managed_zone#name",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_GET,
				TerraformQueryMap: "google_dns_managed_zone.name",
			},
		},
	},
	gcpshared.EssentialContactsContact: {
		Reference:   "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/essential_contacts_contact#email",
		Description: "id => {resourceType}/{resource_id}/contacts/{contact_id}",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_SEARCH,
				TerraformQueryMap: "google_essential_contacts_contact.id",
			},
		},
	},
	gcpshared.IAMRole: {
		Description: "There is no terraform resource for this type.",
	},
	gcpshared.LoggingBucket: {
		Description: "There is no terraform resource for this type.",
	},
	gcpshared.LoggingLink: {
		Description: "There is no terraform resource for this type.",
	},
	gcpshared.LoggingSavedQuery: {
		Description: "There is no terraform resource for this type.",
	},
	gcpshared.LoggingSink: {
		Reference: "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/logging_project_sink",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_GET,
				TerraformQueryMap: "google_logging_project_sink.name",
			},
		},
	},
	gcpshared.MonitoringCustomDashboard: {
		Reference:   "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/monitoring_dashboard",
		Description: "id => projects/{{project}}/dashboards/{{dashboard_id}}",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_SEARCH,
				TerraformQueryMap: "google_monitoring_dashboard.id",
			},
		},
	},
	gcpshared.PubSubSubscription: {
		Reference: "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/pubsub_subscription",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_GET,
				TerraformQueryMap: "google_pubsub_subscription.name",
			},
		},
	},
	gcpshared.PubSubTopic: {
		Reference: "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/pubsub_topic",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_GET,
				TerraformQueryMap: "google_pubsub_topic.name",
			},
		},
	},
	gcpshared.RunRevision: {
		Description: "There is no terraform resource for this type.",
	},
	gcpshared.ServiceDirectoryEndpoint: {
		Reference:   "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/service_directory_endpoint",
		Description: "id => projects/*/locations/*/namespaces/*/services/*/endpoints/*",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_SEARCH,
				TerraformQueryMap: "google_service_directory_endpoint.id",
			},
		},
	},
	gcpshared.ServiceUsageService: {
		Description: "There is no terraform resource for this type.",
	},
	gcpshared.SQLAdminBackup: {
		Description: "There is no terraform resource for this type.",
	},
	gcpshared.SQLAdminBackupRun: {
		Description: "There is no terraform resource for this type.",
	},
	gcpshared.StorageBucket: {
		Reference: "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/storage_bucket",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_GET,
				TerraformQueryMap: "google_storage_bucket.name",
			},
		},
	},
}
