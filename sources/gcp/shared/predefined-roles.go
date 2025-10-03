package shared

type role struct {
	Role           string
	Link           string
	IAMPermissions []string
}

// PredefinedRoles is a map of predefined roles for the GCP source.
// The IAMPermissions field contains the exact permissions from adapter metadata that require this role.
var PredefinedRoles = map[string]role{
	"roles/aiplatform.viewer": {
		Role: "roles/aiplatform.viewer",
		Link: "https://cloud.google.com/iam/docs/roles-permissions/aiplatform#aiplatform.viewer",
		IAMPermissions: []string{
			"aiplatform.batchPredictionJobs.get",
			"aiplatform.batchPredictionJobs.list",
			"aiplatform.customJobs.get",
			"aiplatform.customJobs.list",
			"aiplatform.endpoints.get",
			"aiplatform.endpoints.list",
			"aiplatform.modelDeploymentMonitoringJobs.get",
			"aiplatform.modelDeploymentMonitoringJobs.list",
			"aiplatform.models.get",
			"aiplatform.models.list",
			"aiplatform.pipelineJobs.get",
			"aiplatform.pipelineJobs.list",
			"resourcemanager.projects.get",
		},
	},
	"roles/artifactregistry.reader": {
		Role: "roles/artifactregistry.reader",
		Link: "https://cloud.google.com/iam/docs/roles-permissions/artifactregistry#artifactregistry.reader",
		IAMPermissions: []string{
			"artifactregistry.dockerimages.get",
			"artifactregistry.dockerimages.list",
			"artifactregistry.repositories.get",
			"artifactregistry.repositories.list",
		},
	},
	"roles/bigquery.user": {
		Role: "roles/bigquery.user",
		// TODO: Confirm with the team
		// It has too much permissions, but this is the only role that is used for BigQuery Data Transfer transfer config adapter.
		// When granted on a project, this role also provides the ability to run jobs, including queries, within the project. A principal with this role can enumerate their own jobs, cancel their own jobs, and enumerate datasets within a project. Additionally, allows the creation of new datasets within the project;
		Link: "https://cloud.google.com/iam/docs/roles-permissions/bigquery#bigquery.user",
		IAMPermissions: []string{
			"bigquery.transfers.get",
		},
	},
	"roles/bigquery.metadataViewer": {
		Role: "roles/bigquery.metadataViewer",
		Link: "https://cloud.google.com/iam/docs/roles-permissions/bigquery#bigquery.metadataViewer",
		IAMPermissions: []string{
			"bigquery.datasets.get",
			"bigquery.models.getMetadata",
			"bigquery.models.list",
			"bigquery.tables.get",
			"bigquery.tables.list",
		},
	},
	"roles/bigtable.viewer": {
		Role: "roles/bigtable.viewer",
		// Provides no data access. Intended as a minimal set of permissions to access the Google Cloud console for Bigtable.
		Link: "https://cloud.google.com/iam/docs/roles-permissions/bigtable#bigtable.viewer",
		IAMPermissions: []string{
			"bigtable.clusters.get",
			"bigtable.clusters.list",
			"bigtable.instances.get",
			"bigtable.instances.list",
			"bigtable.appProfiles.get",
			"bigtable.appProfiles.list",
			"bigtable.tables.get",
			"bigtable.tables.list",
			"bigtable.backups.get",
			"bigtable.backups.list",
		},
	},
	"roles/cloudfunctions.viewer": {
		Role: "roles/cloudfunctions.viewer",
		// Read-only access to functions and locations.
		Link: "https://cloud.google.com/iam/docs/roles-permissions/cloudfunctions#cloudfunctions.viewer",
		IAMPermissions: []string{
			"cloudfunctions.functions.get",
			"cloudfunctions.functions.list",
		},
	},
	"roles/resourcemanager.tagViewer": {
		Role: "roles/resourcemanager.tagViewer",
		// Access to list Tags and their associations with resources
		Link: "https://cloud.google.com/iam/docs/roles-permissions/resourcemanager#resourcemanager.tagViewer",
		IAMPermissions: []string{
			"resourcemanager.projects.get",
			"resourcemanager.tagKeys.get",
			"resourcemanager.tagKeys.list",
			"resourcemanager.tagValues.get",
			"resourcemanager.tagValues.list",
		},
	},
	"roles/compute.viewer": {
		Role: "roles/compute.viewer",
		// Read-only access to get and list Compute Engine resources, without being able to read the data stored on them.
		Link: "https://cloud.google.com/iam/docs/roles-permissions/compute#compute.viewer",
		IAMPermissions: []string{
			"compute.acceleratorTypes.get",
			"compute.acceleratorTypes.list",
			"compute.addresses.get",
			"compute.addresses.list",
			"compute.autoscalers.get",
			"compute.autoscalers.list",
			"compute.backendServices.get",
			"compute.backendServices.list",
			"compute.commitments.get",
			"compute.commitments.list",
			"compute.diskTypes.get",
			"compute.diskTypes.list",
			"compute.disks.get",
			"compute.disks.list",
			"compute.externalVpnGateways.get",
			"compute.externalVpnGateways.list",
			"compute.firewalls.get",
			"compute.firewalls.list",
			"compute.forwardingRules.get",
			"compute.forwardingRules.list",
			"compute.healthChecks.get",
			"compute.healthChecks.list",
			"compute.httpHealthChecks.get",
			"compute.httpHealthChecks.list",
			"compute.images.get",
			"compute.images.list",
			"compute.instanceGroupManagers.get",
			"compute.instanceGroupManagers.list",
			"compute.instanceGroups.get",
			"compute.instanceGroups.list",
			"compute.instanceTemplates.get",
			"compute.instanceTemplates.list",
			"compute.instances.get",
			"compute.instances.list",
			"compute.instantSnapshots.get",
			"compute.instantSnapshots.list",
			"compute.licenses.get",
			"compute.licenses.list",
			"compute.machineImages.get",
			"compute.machineImages.list",
			"compute.networkEndpointGroups.get",
			"compute.networkEndpointGroups.list",
			"compute.networks.get",
			"compute.networks.list",
			"compute.nodeGroups.get",
			"compute.nodeGroups.list",
			"compute.nodeTemplates.get",
			"compute.nodeTemplates.list",
			"compute.projects.get",
			"compute.publicDelegatedPrefixes.get",
			"compute.publicDelegatedPrefixes.list",
			"compute.regionBackendServices.get",
			"compute.regionBackendServices.list",
			"compute.reservations.get",
			"compute.reservations.list",
			"compute.resourcePolicies.get",
			"compute.resourcePolicies.list",
			"compute.routers.get",
			"compute.routers.list",
			"compute.routes.get",
			"compute.routes.list",
			"compute.securityPolicies.get",
			"compute.securityPolicies.list",
			"compute.snapshots.get",
			"compute.snapshots.list",
			"compute.sslCertificates.get",
			"compute.sslCertificates.list",
			"compute.sslPolicies.get",
			"compute.sslPolicies.list",
			"compute.storagePools.get",
			"compute.storagePools.list",
			"compute.subnetworks.get",
			"compute.subnetworks.list",
			"compute.targetHttpProxies.get",
			"compute.targetHttpProxies.list",
			"compute.targetHttpsProxies.get",
			"compute.targetHttpsProxies.list",
			"compute.targetPools.get",
			"compute.targetPools.list",
			"compute.urlMaps.get",
			"compute.urlMaps.list",
			"compute.vpnGateways.get",
			"compute.vpnGateways.list",
			"compute.vpnTunnels.get",
			"compute.vpnTunnels.list",
		},
	},
	"roles/container.viewer": {
		Role: "roles/container.viewer",
		Link: "https://cloud.google.com/iam/docs/roles-permissions/container#container.viewer",
		IAMPermissions: []string{
			"container.clusters.get",
			"container.clusters.list",
		},
	},
	"roles/dataproc.viewer": {
		Role: "roles/dataproc.viewer",
		// Provides read-only access to Dataproc resources.
		Link: "https://cloud.google.com/iam/docs/roles-permissions/dataproc#dataproc.viewer",
		IAMPermissions: []string{
			"dataproc.autoscalingPolicies.get",
			"dataproc.autoscalingPolicies.list",
			"dataproc.clusters.get",
			"dataproc.clusters.list",
		},
	},
	"roles/monitoring.viewer": {
		Role: "roles/monitoring.viewer",
		// Provides read-only access to get and list information about all monitoring data and configurations.
		Link: "https://cloud.google.com/iam/docs/roles-permissions/monitoring#monitoring.viewer",
		IAMPermissions: []string{
			"monitoring.alertPolicies.get",
			"monitoring.alertPolicies.list",
			"monitoring.dashboards.get",
			"monitoring.dashboards.list",
			"monitoring.notificationChannels.get",
			"monitoring.notificationChannels.list",
		},
	},
	"roles/redis.viewer": {
		Role: "roles/redis.viewer",
		// Read-only access to Redis instances and related resources.
		Link: "https://cloud.google.com/iam/docs/roles-permissions/redis#redis.viewer",
		IAMPermissions: []string{
			"redis.instances.get",
			"redis.instances.list",
		},
	},
	"roles/run.viewer": {
		Role: "roles/run.viewer",
		// Can view the state of all Cloud Run resources, including IAM policies.
		Link: "https://cloud.google.com/iam/docs/roles-permissions/run#run.viewer",
		IAMPermissions: []string{
			"run.revisions.get",
			"run.revisions.list",
			"run.services.get",
			"run.services.list",
			"run.workerPools.get",
			"run.workerPools.list",
		},
	},
	"roles/secretmanager.viewer": {
		Role: "roles/secretmanager.viewer",
		// Allows viewing metadata of all Secret Manager resources
		Link: "https://cloud.google.com/iam/docs/roles-permissions/secretmanager#secretmanager.viewer",
		IAMPermissions: []string{
			"secretmanager.secrets.get",
			"secretmanager.secrets.list",
		},
	},
	"roles/spanner.viewer": {
		Role: "roles/spanner.viewer",
		/*
			A principal with this role can:
				- View all Spanner instances (but cannot modify instances).
				- View all Spanner databases (but cannot modify or read from databases).
		*/
		// TODO: Validate if spanner.databases.list is enough for the spanner instance adapter.
		// Because, spanner.databases.get is only available on roles that can read data from the database.
		// https://linear.app/overmind/issue/ENG-1468/validate-gcp-predefined-role-for-spanner-database-adapter
		Link: "https://cloud.google.com/iam/docs/roles-permissions/spanner#spanner.viewer",
		IAMPermissions: []string{
			"spanner.databases.list",
			"spanner.instanceConfigs.get",
			"spanner.instanceConfigs.list",
			"spanner.instances.get",
			"spanner.instances.list",
		},
	},
	"roles/cloudsql.viewer": {
		Role: "roles/cloudsql.viewer",
		// Provides read-only access to Cloud SQL resources.
		Link: "https://cloud.google.com/iam/docs/roles-permissions/cloudsql#cloudsql.viewer",
		IAMPermissions: []string{
			"cloudsql.backupRuns.get",
			"cloudsql.backupRuns.list",
			"cloudsql.instances.get",
			"cloudsql.instances.list",
		},
	},
	"roles/storagetransfer.viewer": {
		Role: "roles/storagetransfer.viewer",
		// Read access to storage transfer jobs and operations.
		Link: "https://cloud.google.com/iam/docs/roles-permissions/storagetransfer#storagetransfer.viewer",
		IAMPermissions: []string{
			"storagetransfer.jobs.get",
			"storagetransfer.jobs.list",
		},
	},
	"roles/storage.bucketViewer": {
		Role: "roles/storage.bucketViewer",
		// Grants permission to view buckets and their metadata, excluding IAM policies.
		// This role is in Beta mode, but we don't have any alternatives.
		Link: "https://cloud.google.com/iam/docs/roles-permissions/storage#storage.bucketViewer",
		IAMPermissions: []string{
			"storage.buckets.get",
			"storage.buckets.list",
		},
	},
	"roles/pubsub.viewer": {
		Role: "roles/pubsub.viewer",
		// Provides access to view topics and subscriptions.
		Link: "https://cloud.google.com/iam/docs/roles-permissions/pubsub#pubsub.viewer",
		IAMPermissions: []string{
			"pubsub.subscriptions.get",
			"pubsub.subscriptions.list",
			"pubsub.topics.get",
			"pubsub.topics.list",
		},
	},
	"roles/dataplex.viewer": {
		Role: "roles/dataplex.viewer",
		// Read access to Dataplex Universal Catalog resources, except for catalog resources like entries, entry groups, and glossaries.
		Link: "https://cloud.google.com/iam/docs/roles-permissions/dataplex#dataplex.viewer",
		IAMPermissions: []string{
			"dataplex.dataScans.get",
			"dataplex.dataScans.list",
		},
	},
	"roles/dataplex.catalogViewer": {
		Role: "roles/dataplex.catalogViewer",
		// Read access to catalog resources, including entries, entry groups, and glossaries. Can view IAM policies on catalog resources.
		Link: "https://cloud.google.com/iam/docs/roles-permissions/dataplex#dataplex.catalogViewer",
		IAMPermissions: []string{
			"dataplex.aspectTypes.get",
			"dataplex.aspectTypes.list",
			"dataplex.entryGroups.get",
			"dataplex.entryGroups.list",
		},
	},
	"roles/iam.roleViewer": {
		Role: "roles/iam.roleViewer",
		// Provides read access to all custom roles in the project.
		Link: "https://cloud.google.com/iam/docs/roles-permissions/iam#iam.roleViewer",
		IAMPermissions: []string{
			"iam.roles.get",
			"iam.roles.list",
		},
	},
	"roles/iam.serviceAccountViewer": {
		Role: "roles/iam.serviceAccountViewer",
		Link: "https://cloud.google.com/iam/docs/roles-permissions/iam#iam.serviceAccountViewer",
		IAMPermissions: []string{
			"iam.serviceAccountKeys.get",
			"iam.serviceAccountKeys.list",
			"iam.serviceAccounts.get",
			"iam.serviceAccounts.list",
		},
	},
	"roles/dns.reader": {
		Role: "roles/dns.reader",
		Link: "https://cloud.google.com/iam/docs/roles-permissions/dns#dns.reader",
		IAMPermissions: []string{
			"dns.managedZones.get",
			"dns.managedZones.list",
		},
	},
	"roles/logging.viewer": {
		Role: "roles/logging.viewer",
		// Provides access to view logs.
		Link: "https://cloud.google.com/iam/docs/roles-permissions/logging#logging.viewer",
		IAMPermissions: []string{
			"logging.buckets.get",
			"logging.buckets.list",
			"logging.links.get",
			"logging.links.list",
			"logging.queries.getShared",
			"logging.queries.listShared",
			"logging.sinks.get",
			"logging.sinks.list",
		},
	},
	"roles/serviceusage.serviceUsageViewer": {
		Role: "roles/serviceusage.serviceUsageViewer",
		// Ability to inspect service states and operations for a consumer project.
		Link: "https://cloud.google.com/iam/docs/roles-permissions/serviceusage#serviceusage.serviceUsageViewer",
		IAMPermissions: []string{
			"serviceusage.services.get",
			"serviceusage.services.list",
		},
	},
	"roles/servicedirectory.viewer": {
		Role: "roles/servicedirectory.viewer",
		// View Service Directory resources.
		Link: "https://cloud.google.com/iam/docs/roles-permissions/servicedirectory#servicedirectory.viewer",
		IAMPermissions: []string{
			"servicedirectory.endpoints.get",
			"servicedirectory.endpoints.list",
			"servicedirectory.services.get",
			"servicedirectory.services.list",
		},
	},
	"roles/eventarc.viewer": {
		Role: "roles/eventarc.viewer",
		// Can view the state of all Eventarc resources, including IAM policies.
		Link: "https://cloud.google.com/iam/docs/roles-permissions/eventarc#eventarc.viewer",
		IAMPermissions: []string{
			"eventarc.triggers.get",
			"eventarc.triggers.list",
		},
	},
	"roles/orgpolicy.policyViewer": {
		Role: "roles/orgpolicy.policyViewer",
		Link: "https://cloud.google.com/iam/docs/roles-permissions/orgpolicy#orgpolicy.policyViewer",
		IAMPermissions: []string{
			"orgpolicy.policy.get",
			"orgpolicy.policies.list",
		},
	},
	"roles/essentialcontacts.viewer": {
		Role: "roles/essentialcontacts.viewer",
		// Viewer for all essential contacts
		Link: "https://cloud.google.com/iam/docs/roles-permissions/essentialcontacts#essentialcontacts.viewer",
		IAMPermissions: []string{
			"essentialcontacts.contacts.get",
			"essentialcontacts.contacts.list",
		},
	},
	"roles/file.viewer": {
		Role: "roles/file.viewer",
		// Read-only access to Filestore instances and related resources.
		// This role is in Beta mode, but we don't have any alternatives.
		Link: "https://cloud.google.com/iam/docs/roles-permissions/file#file.viewer",
		IAMPermissions: []string{
			"file.instances.get",
			"file.instances.list",
		},
	},
	"roles/securitycentermanagement.viewer": {
		Role: "roles/securitycentermanagement.viewer",
		// Readonly access to Cloud Security Command Center services and custom modules configuration.
		Link: "https://cloud.google.com/iam/docs/roles-permissions/securitycentermanagement#securitycentermanagement.viewer",
		IAMPermissions: []string{
			"securitycentermanagement.securityCenterServices.get",
			"securitycentermanagement.securityCenterServices.list",
		},
	},
	"roles/cloudbuild.builds.viewer": {
		Role: "roles/cloudbuild.builds.viewer",
		// Provides access to view builds.
		Link: "https://cloud.google.com/iam/docs/roles-permissions/cloudbuild#cloudbuild.builds.viewer",
		IAMPermissions: []string{
			"cloudbuild.builds.get",
			"cloudbuild.builds.list",
		},
	},
	"roles/dataform.viewer": {
		Role: "roles/dataform.viewer",
		// Read-only access to all Dataform resources.
		Link: "https://cloud.google.com/iam/docs/roles-permissions/dataform#dataform.viewer",
		IAMPermissions: []string{
			"dataform.repositories.get",
			"dataform.repositories.list",
		},
	},
	"roles/cloudkms.viewer": {
		Role: "roles/cloudkms.viewer",
		// Read-only access to Cloud KMS resources.
		Link: "https://cloud.google.com/iam/docs/roles-permissions/cloudkms#cloudkms.viewer",
		IAMPermissions: []string{
			"cloudkms.cryptoKeys.get",
			"cloudkms.cryptoKeys.list",
			"cloudkms.keyRings.get",
			"cloudkms.keyRings.list",
		},
	},
}
