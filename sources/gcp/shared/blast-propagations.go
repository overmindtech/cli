package shared

import (
	"github.com/overmindtech/cli/sdp-go"
	aws "github.com/overmindtech/cli/sources/aws/shared"
	"github.com/overmindtech/cli/sources/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

type Impact struct {
	ToSDPITemType    shared.ItemType
	Description      string
	BlastPropagation *sdp.BlastPropagation
}

var (
	impactInOnly   = &sdp.BlastPropagation{In: true}
	impactOutOnly  = &sdp.BlastPropagation{Out: true}
	impactBothWays = &sdp.BlastPropagation{In: true, Out: true}
)

var (
	ipImpactBothWays = &Impact{
		Description:      "IP addresses are tightly coupled with the source type.",
		ToSDPITemType:    stdlib.NetworkIP,
		BlastPropagation: impactBothWays,
	}
	securityPolicyImpactInOnly = &Impact{
		Description:      "Any change on the security policy impacts the source, but not the other way around.",
		ToSDPITemType:    ComputeSecurityPolicy,
		BlastPropagation: impactInOnly,
	}
	cryptoKeyImpactInOnly = &Impact{
		Description:      "If the crypto key is updated: The source may not be able to access encrypted data. If the source is updated: The crypto key remains unaffected.",
		ToSDPITemType:    CloudKMSCryptoKey,
		BlastPropagation: impactInOnly,
	}
	cryptoKeyVersionImpactInOnly = &Impact{
		Description:      "If the crypto key version is updated: The source may not be able to access encrypted data. If the source is updated: The crypto key version remains unaffected.",
		ToSDPITemType:    CloudKMSCryptoKeyVersion,
		BlastPropagation: impactInOnly,
	}
	iamServiceAccountImpactInOnly = &Impact{
		Description:      "If the service account is updated: The source may not be able to access encrypted data. If the source is updated: The service account remains unaffected.",
		ToSDPITemType:    IAMServiceAccount,
		BlastPropagation: impactInOnly,
	}
	resourcePolicyImpactInOnly = &Impact{
		Description:      "If the resource policy is updated: The source may not be able to access the resource as expected. If the source is updated: The resource policy remains unaffected.",
		ToSDPITemType:    ComputeResourcePolicy,
		BlastPropagation: impactInOnly,
	}
)

var BlastPropagations = map[shared.ItemType]map[string]*Impact{
	AIPlatformCustomJob: {
		// The Cloud KMS key that will be used to encrypt the output artifacts.
		"encryptionSpec.kmsKeyName": {
			Description:      "If the Cloud KMS CryptoKey is updated: The CustomJob may not be able to access encrypted output artifacts. If the CustomJob is updated: The CryptoKey remains unaffected.",
			ToSDPITemType:    CloudKMSCryptoKey,
			BlastPropagation: impactInOnly,
		},
		// The full name of the network to which the job should be peered.
		"jobSpec.network": {
			Description:      "If the Compute Network is deleted or updated: The CustomJob may lose connectivity or fail to run as expected. If the CustomJob is updated: The network remains unaffected.",
			ToSDPITemType:    ComputeNetwork,
			BlastPropagation: impactInOnly,
		},
		// The service account that the job runs as.
		"jobSpec.serviceAccount": {
			Description:      "If the IAM Service Account is deleted or updated: The CustomJob may fail to run or lose permissions. If the CustomJob is updated: The service account remains unaffected.",
			ToSDPITemType:    IAMServiceAccount,
			BlastPropagation: impactInOnly,
		},
		// The Cloud Storage location to store the output of this CustomJob.
		"jobSpec.baseOutputDirectory.gcsOutputDirectory": {
			Description:      "If the Storage Bucket is deleted or updated: The CustomJob may fail to write outputs. If the CustomJob is updated: The bucket remains unaffected.",
			ToSDPITemType:    StorageBucket,
			BlastPropagation: impactInOnly,
		},
		// Optional. The name of a Vertex AI Tensorboard resource to which this CustomJob will upload Tensorboard logs.
		"jobSpec.tensorboard": {
			Description:      "If the Vertex AI Tensorboard is deleted or updated: The CustomJob may fail to upload logs or lose access to previous logs. If the CustomJob is updated: The tensorboard remains unaffected.",
			ToSDPITemType:    AIPlatformTensorBoard,
			BlastPropagation: impactInOnly,
		},
		// Optional. The name of an experiment to associate with the CustomJob.
		"jobSpec.experiment": {
			Description:      "If the Vertex AI Experiment is deleted or updated: The CustomJob may lose experiment tracking or association. If the CustomJob is updated: The experiment remains unaffected.",
			ToSDPITemType:    AIPlatformExperiment,
			BlastPropagation: impactInOnly,
		},
		// Optional. The name of an experiment run to associate with the CustomJob.
		"jobSpec.experimentRun": {
			Description:      "If the Vertex AI ExperimentRun is deleted or updated: The CustomJob may lose run tracking or association. If the CustomJob is updated: The experiment run remains unaffected.",
			ToSDPITemType:    AIPlatformExperimentRun,
			BlastPropagation: impactInOnly,
		},
		// Optional. The name of a model to upload the trained Model to upon job completion.
		"jobSpec.models": {
			Description:      "If the Vertex AI Model is deleted or updated: The CustomJob may fail to upload or associate the trained model. If the CustomJob is updated: The model remains unaffected.",
			ToSDPITemType:    AIPlatformModel,
			BlastPropagation: impactInOnly,
		},
	},
	AIPlatformPipelineJob: {
		// The service account that the pipeline workload runs as (root-level).
		"serviceAccount": {
			Description:      "If the IAM Service Account is deleted or updated: The PipelineJob may fail to run or lose permissions. If the PipelineJob is updated: The service account remains unaffected.",
			ToSDPITemType:    IAMServiceAccount,
			BlastPropagation: impactInOnly,
		},
		// The full name of the network to which the job should be peered (root-level).
		"network": {
			Description:      "If the Compute Network is deleted or updated: The PipelineJob may lose connectivity or fail to run as expected. If the PipelineJob is updated: The network remains unaffected.",
			ToSDPITemType:    ComputeNetwork,
			BlastPropagation: impactInOnly,
		},
		// The Cloud KMS key used to encrypt PipelineJob outputs.
		"encryptionSpec.kmsKeyName": {
			Description:      "If the Cloud KMS CryptoKey is updated: The PipelineJob may not be able to access encrypted output artifacts. If the PipelineJob is updated: The CryptoKey remains unaffected.",
			ToSDPITemType:    CloudKMSCryptoKey,
			BlastPropagation: impactInOnly,
		},
		// The Cloud Storage location to store the output of this PipelineJob.
		"runtimeConfig.gcsOutputDirectory": {
			Description:      "If the Storage Bucket is deleted or updated: The PipelineJob may fail to write outputs. If the PipelineJob is updated: The bucket remains unaffected.",
			ToSDPITemType:    StorageBucket,
			BlastPropagation: impactInOnly,
		},
	},
	BigTableAdminAppProfile: {
		"multiClusterRoutingUseAny.clusterIds": {
			ToSDPITemType:    BigTableAdminCluster,
			Description:      "If the BigTableAdmin Cluster is deleted or updated: The AppProfile may lose routing capabilities or fail to access data. If the AppProfile is updated: The cluster remains unaffected.",
			BlastPropagation: impactInOnly,
		},
		"singleClusterRouting.clusterId": {
			ToSDPITemType:    BigTableAdminCluster,
			Description:      "If the BigTableAdmin Cluster is deleted or updated: The AppProfile may lose routing capabilities or fail to access data. If the AppProfile is updated: The cluster remains unaffected.",
			BlastPropagation: impactInOnly,
		},
	},
	BigTableAdminBackup: {
		"sourceTable": {
			ToSDPITemType:    BigTableAdminTable,
			Description:      "If the BigTableAdmin Table is deleted or updated: The Backup may become invalid or inaccessible. If the Backup is updated: The table remains unaffected.",
			BlastPropagation: impactInOnly,
		},
		"sourceBackup": {
			ToSDPITemType:    BigTableAdminBackup,
			Description:      "If the source Backup is deleted or updated: The Backup may become invalid or inaccessible. If the Backup is updated: The source backup remains unaffected.",
			BlastPropagation: impactInOnly,
		},
		"encryptionInfo.kmsKeyVersion": cryptoKeyVersionImpactInOnly,
	},
	BigTableAdminTable: {
		// If this table was restored from another data source (e.g. a backup), this field, restoreInfo, will be populated with information about the restore.
		"restoreInfo.backupInfo.sourceTable": {
			ToSDPITemType:    BigTableAdminTable,
			Description:      "If the source BigTableAdmin Table is deleted or updated: The restored table may become invalid or inaccessible. If the restored table is updated: The source table remains unaffected.",
			BlastPropagation: impactInOnly,
		},
		"restoreInfo.backupInfo.sourceBackup": {
			ToSDPITemType:    BigTableAdminBackup,
			Description:      "If the source BigTableAdmin Backup is deleted or updated: The restored table may become invalid or inaccessible. If the restored table is updated: The source backup remains unaffected.",
			BlastPropagation: impactInOnly,
		},
	},
	CloudBuildBuild: {
		"source.storageSource.bucket": {
			ToSDPITemType:    StorageBucket,
			Description:      "If the Storage Bucket is deleted or updated: The Cloud Build may fail to access source files. If the Cloud Build is updated: The bucket remains unaffected.",
			BlastPropagation: impactInOnly,
		},
		"steps.name": {
			ToSDPITemType:    ArtifactRegistryDockerImage,
			Description:      "If the Artifact Registry Docker Image is deleted or updated: The Cloud Build may fail to pull the image. If the Cloud Build is updated: The Docker image remains unaffected.",
			BlastPropagation: impactInOnly,
		},
		"results.images": {
			ToSDPITemType:    ArtifactRegistryDockerImage,
			Description:      "If the Cloud Build is updated or deleted: The Artifact Registry Docker Images may no longer be valid or accessible. If the Docker Images are updated: The Cloud Build remains unaffected.",
			BlastPropagation: impactOutOnly,
		},
		"images": {
			ToSDPITemType:    ArtifactRegistryDockerImage,
			Description:      "If any of the images fail to be pushed, the build status is marked FAILURE.",
			BlastPropagation: impactOutOnly,
		},
		"logsBucket": {
			ToSDPITemType:    LoggingBucket,
			Description:      "If the Logging Bucket is deleted or updated: The Cloud Build may fail to write logs. If the Cloud Build is updated: The bucket remains unaffected.",
			BlastPropagation: impactInOnly,
		},
		"serviceAccount": iamServiceAccountImpactInOnly,
		"buildTriggerId": {
			// The ID of the BuildTrigger that triggered this build, if it was triggered automatically.
			ToSDPITemType:    CloudBuildTrigger,
			Description:      "If the Cloud Build Trigger is deleted or updated: The Cloud Build may not be retriggered as expected. If the Cloud Build is updated: The trigger remains unaffected.",
			BlastPropagation: impactInOnly,
		},
	},
	ComputeFirewall: {
		"network": {
			Description:      "If the Compute Network is updated: The firewall rules may no longer apply correctly. If the firewall is updated: The network remains unaffected, but its security posture may change.",
			ToSDPITemType:    ComputeNetwork,
			BlastPropagation: impactBothWays,
		},
		"sourceServiceAccounts": iamServiceAccountImpactInOnly,
		"targetServiceAccounts": iamServiceAccountImpactInOnly,
	},
	ComputeInstanceTemplate: {
		// https://cloud.google.com/compute/docs/reference/rest/v1/instanceTemplates/get
		"properties.machineType": {
			Description:      "If the Compute Machine Type is deleted: The instance template becomes partially invalid. If the template is updated: Machine type itself is not affected.",
			ToSDPITemType:    ComputeMachineType,
			BlastPropagation: impactInOnly,
		},
		"properties.networkInterfaces.network": {
			Description:      "If the network is deleted: Resources may experience connectivity changes or disruptions. If the template is deleted: Network itself is not affected.",
			ToSDPITemType:    ComputeNetwork,
			BlastPropagation: impactInOnly,
		},
		"properties.networkInterfaces.subnetwork": {
			Description:      "If the (sub)network is deleted: Resources may experience connectivity changes or disruptions. If the template is updated: Subnetwork itself is not affected.",
			ToSDPITemType:    ComputeSubnetwork,
			BlastPropagation: impactInOnly,
		},
		"properties.networkInterfaces.networkIP": {
			Description:      "IP address are always tightly coupled with the Compute Instance Template.",
			ToSDPITemType:    stdlib.NetworkIP,
			BlastPropagation: impactBothWays,
		},
		"properties.networkInterfaces.ipv6Address":                      ipImpactBothWays,
		"properties.networkInterfaces.accessConfigs.natIP":              ipImpactBothWays,
		"properties.networkInterfaces.accessConfigs.externalIpv6":       ipImpactBothWays,
		"properties.networkInterfaces.accessConfigs.securityPolicy":     securityPolicyImpactInOnly,
		"properties.networkInterfaces.ipv6AccessConfigs.natIP":          ipImpactBothWays,
		"properties.networkInterfaces.ipv6AccessConfigs.externalIpv6":   ipImpactBothWays,
		"properties.networkInterfaces.ipv6AccessConfigs.securityPolicy": securityPolicyImpactInOnly,
		"properties.disks.source": {
			Description:      "If the Compute Disk is updated: Instance creation may fail or behave unexpectedly. If the template is deleted: Existing disks can be deleted.",
			ToSDPITemType:    ComputeDisk,
			BlastPropagation: impactBothWays,
		},
		"properties.disks.initializeParams.diskName": {
			Description:      "If the Compute Disk is updated: Instance creation may fail or behave unexpectedly. If the template is deleted: Existing disks can be deleted.",
			ToSDPITemType:    ComputeDisk,
			BlastPropagation: impactBothWays,
		},
		"properties.disks.initializeParams.sourceImage": {
			Description:      "If the Compute Image is updated: Instances created from this template may not boot correctly. If the template is updated: Image is not affected.",
			ToSDPITemType:    ComputeImage,
			BlastPropagation: impactInOnly,
		},
		"properties.disks.initializeParams.diskType": {
			Description:      "If the Compute Disk Type is updated: New instances may fail to provision disks properly. If the template is updated: Disk type is not affected.",
			ToSDPITemType:    ComputeDiskType,
			BlastPropagation: impactInOnly,
		},
		"properties.disks.initializeParams.sourceImageEncryptionKey.kmsKeyName":           cryptoKeyImpactInOnly,
		"properties.disks.initializeParams.sourceImageEncryptionKey.kmsKeyServiceAccount": iamServiceAccountImpactInOnly,
		"properties.disks.initializeParams.sourceSnapshot": {
			Description:      "If the Compute Snapshot is updated: The template may reference an invalid or incompatible snapshot. If the template is updated: no impact on snapshots.",
			ToSDPITemType:    ComputeSnapshot,
			BlastPropagation: impactInOnly,
		},
		"properties.disks.initializeParams.sourceSnapshotEncryptionKey.kmsKeyName":           cryptoKeyImpactInOnly,
		"properties.disks.initializeParams.sourceSnapshotEncryptionKey.kmsKeyServiceAccount": iamServiceAccountImpactInOnly,
		"properties.disks.initializeParams.resourcePolicies":                                 resourcePolicyImpactInOnly,
		"properties.disks.initializeParams.storagePool": {
			Description:      "If the Compute Storage Pool is deleted: Disk provisioning for new instances may fail. If the template is updated: Pool is not affected.",
			ToSDPITemType:    ComputeStoragePool,
			BlastPropagation: impactInOnly,
		},
		"properties.disks.diskEncryptionKey.kmsKeyName":           cryptoKeyImpactInOnly,
		"properties.disks.diskEncryptionKey.kmsKeyServiceAccount": cryptoKeyImpactInOnly,
		"properties.guestAccelerators.acceleratorType": {
			Description:      "If the Compute Accelerator Type is updated: New instances may misconfigure or fail hardware initialization. If the template is updated: Accelerator is not affected.",
			ToSDPITemType:    ComputeAcceleratorType,
			BlastPropagation: impactInOnly,
		},
		"sourceInstance": {
			Description:      "If the Compute Instance is updated: The template may reference an invalid or incompatible instance. If the template is deleted: The instance remains unaffected.",
			ToSDPITemType:    ComputeInstance,
			BlastPropagation: impactInOnly,
		},
		"sourceInstanceParams.diskConfigs.customImage": {
			Description:      "If the Compute Image is updated: Instances created from this template may not boot correctly. If the template is updated: Image is not affected.",
			ToSDPITemType:    ComputeImage,
			BlastPropagation: impactInOnly,
		},
		"properties.networkInterfaces.networkAttachment": {
			Description:      "If the Compute Network Attachment is updated: Instances using the template may lose access to the network services. If the template is deleted: Attachment is not affected.",
			ToSDPITemType:    ComputeNetworkAttachment,
			BlastPropagation: impactInOnly,
		},
		"properties.disks.initializeParams.licenses": {
			Description:      "If the Compute License is updated: New instances may violate license agreements or lose functionality. If the template is updated: License remains unaffected..",
			ToSDPITemType:    ComputeLicense,
			BlastPropagation: impactInOnly,
		},
		"properties.disks.licenses": {
			Description:      "If the Compute License is updated: New instances may violate license agreements or lose functionality. If the template is updated: License remains unaffected..",
			ToSDPITemType:    ComputeLicense,
			BlastPropagation: impactInOnly,
		},
		"properties.reservationAffinity.values": {
			Description:      "If the Compute Reservation is updated: new instances created using it may fail to launch. If the template is updated: no impacts on reservation.",
			ToSDPITemType:    ComputeReservation,
			BlastPropagation: impactInOnly,
		},
		"properties.scheduling.nodeAffinities.values": {
			Description:      "If the Compute Node Group is updated: Placement policies may break for new VMs. If the template is updated: Node affinity rules may change. Changing the affinity might cause new VMs to stop using that Node Group",
			ToSDPITemType:    ComputeNodeGroup,
			BlastPropagation: impactBothWays,
		},
	},
	ComputeNetwork: {
		"gatewayIPv4": ipImpactBothWays,
		"subnetworks": {
			Description:      "If the Compute Subnetwork is deleted: The network remains unaffected, but its subnetwork configuration may change. If the network is deleted: All associated subnetworks are also deleted.",
			ToSDPITemType:    ComputeSubnetwork,
			BlastPropagation: impactBothWays,
		},
		"peerings.network": {
			Description:      "If the Compute Network Peering is deleted: The network remains unaffected, but its peering configuration may change. If the network is deleted: All associated peerings are also deleted.",
			ToSDPITemType:    ComputeNetwork,
			BlastPropagation: impactBothWays,
		},
		"firewallPolicy": {
			Description:      "If the Compute Firewall Policy is updated: The network's security posture may change. If the network is updated: The firewall policy remains unaffected, but its application to the network may change.",
			ToSDPITemType:    ComputeFirewallPolicy,
			BlastPropagation: impactInOnly,
		},
	},
	ComputeProject: {
		"defaultServiceAccount": {
			Description:      "If the IAM Service Account is deleted: Project resources may fail to work as before. If the project is deleted: service account is deleted.",
			ToSDPITemType:    IAMServiceAccount,
			BlastPropagation: impactBothWays,
		},
		"usageExportLocation.bucketName": {
			Description:      "If the Compute Bucket is deleted: Project usage export may fail. If the project is deleted: bucket is deleted.",
			ToSDPITemType:    StorageBucket,
			BlastPropagation: impactBothWays,
		},
	},
	ComputeRoute: {
		// https://cloud.google.com/compute/docs/reference/rest/v1/routes/get
		// Network that the route belongs to
		"network": {
			Description:      "If the Compute Network is updated: The route may no longer be valid or correctly associated. If the route is updated: The network remains unaffected, but its routing behavior may change.",
			ToSDPITemType:    ComputeNetwork,
			BlastPropagation: impactBothWays,
		},
		// Network that the route forwards traffic to, so the relationship will/may be different
		"nextHopNetwork": {
			Description:      "If the Compute Network is updated: The route may no longer forward traffic properly. If the route is updated: The network remains unaffected but traffic routed through it may be affected.",
			ToSDPITemType:    ComputeNetwork,
			BlastPropagation: impactBothWays,
		},
		"nextHopIp": {
			Description:      "The network IP address of an instance that should handle matching packets. Tightly coupled with the Compute Route.",
			ToSDPITemType:    stdlib.NetworkIP,
			BlastPropagation: impactBothWays,
		},
		"nextHopInstance": {
			Description:      "If the Compute Instance is updated: Routes using it as a next hop may break or change behavior. If the route is deleted: The instance remains unaffected but traffic that was previously using that route will be impacted.",
			ToSDPITemType:    ComputeInstance,
			BlastPropagation: impactInOnly,
		},
		"nextHopVpnTunnel": {
			Description:      "If the VPN Tunnel is updated: The route may no longer forward traffic properly. If the route is updated: The VPN tunnel remains unaffected but traffic routed through it may be affected.",
			ToSDPITemType:    ComputeVpnTunnel,
			BlastPropagation: impactBothWays,
		},
		"nextHopGateway": {
			Description:      "If the Compute Gateway is updated: The route may no longer forward traffic properly. If the route is updated: The gateway remains unaffected but traffic routed through it may be affected.",
			ToSDPITemType:    ComputeGateway,
			BlastPropagation: impactInOnly,
		},
		"nextHopHub": {
			// https://cloud.google.com/network-connectivity/docs/reference/networkconnectivity/rest/v1/projects.locations.global.hubs/get
			Description:      "The full resource name of the Network Connectivity Center hub that will handle matching packets. If the hub is updated: The route may no longer forward traffic properly. If the route is updated: The hub remains unaffected but traffic routed through it may be affected.",
			ToSDPITemType:    NetworkConnectivityHub,
			BlastPropagation: impactBothWays,
		},
	},
	ComputeSubnetwork: {
		"network": {
			Description:      "If the Compute Network is updated: The firewall rules may no longer apply correctly. If the firewall is updated: The network remains unaffected, but its security posture may change.",
			ToSDPITemType:    ComputeNetwork,
			BlastPropagation: impactBothWays,
		},
		"gatewayAddress": {
			Description:      "If the Compute Gateway is deleted: subnetwork is not impacted. If the subnetwork is deleted: gateway is also deleted.",
			ToSDPITemType:    ComputeGateway,
			BlastPropagation: impactOutOnly,
		},
	},
	DataplexEntryGroup: {
		// There is no links for this item type.
	},
	LoggingLink: {
		"bigqueryDataset.datasetId": {
			Description:      "They are tightly coupled with the Logging Link.",
			ToSDPITemType:    BigQueryDataset,
			BlastPropagation: impactBothWays,
		},
	},
	LoggingSavedQuery: {
		// There is no links for this item type.
	},
	LoggingBucket: {
		"cmekSettings.kmsKeyName":        cryptoKeyImpactInOnly,
		"cmekSettings.kmsKeyVersionName": cryptoKeyVersionImpactInOnly,
		"cmekSettings.serviceAccountId":  iamServiceAccountImpactInOnly,
	},
	IAMRole: {
		// There is no links for this item type.
	},
	PubSubSubscription: {
		"topic": {
			ToSDPITemType:    PubSubTopic,
			Description:      "If the Pub/Sub Topic is deleted or updated: The Subscription may fail to receive messages. If the Subscription is updated: The topic remains unaffected.",
			BlastPropagation: impactInOnly,
		},
		"bigqueryConfig.table": {
			// The name of the table to which to write data, of the form {projectId}.{datasetId}.{tableId}
			// We have a manual adapter for this.
			ToSDPITemType:    BigQueryTable,
			Description:      "If the BigQuery Table is deleted or updated: The Subscription may fail to write data. If the Subscription is updated: The table remains unaffected.",
			BlastPropagation: impactInOnly,
		},
		"cloudStorageConfig.bucket": {
			ToSDPITemType:    StorageBucket,
			Description:      "If the Cloud Storage Bucket is deleted or updated: The Subscription may fail to write data. If the Subscription is updated: The bucket remains unaffected.",
			BlastPropagation: impactInOnly,
		},
		"analyticsHubSubscriptionInfo.subscription": {
			ToSDPITemType:    PubSubSubscription,
			Description:      "If the Pub/Sub Subscription is deleted or updated: The Analytics Hub Subscription may fail to receive messages. If the Analytics Hub Subscription is updated: The Pub/Sub Subscription remains unaffected.",
			BlastPropagation: impactOutOnly,
		},
	},
	PubSubTopic: {
		"kmsKeyName": cryptoKeyImpactInOnly,
		// Settings for ingestion from a data source into this topic.
		"ingestionDataSourceSettings.cloudStorage.bucket": {
			ToSDPITemType:    StorageBucket,
			Description:      "If the Cloud Storage Bucket is deleted or updated: The Pub/Sub Topic may fail to receive data. If the Topic is updated: The bucket remains unaffected.",
			BlastPropagation: impactInOnly,
		},
		"ingestionDataSourceSettings.awsKinesis.streamAr": {
			ToSDPITemType:    aws.KinesisStream,
			Description:      "The Kinesis stream ARN to ingest data from.",
			BlastPropagation: impactInOnly,
		},
		"ingestionDataSourceSettings.awsKinesis.consumerArn": {
			ToSDPITemType:    aws.KinesisStreamConsumer,
			Description:      "The Kinesis consumer ARN to used for ingestion in Enhanced Fan-Out mode. The consumer must be already created and ready to be used.",
			BlastPropagation: impactInOnly,
		},
		"ingestionDataSourceSettings.awsKinesis.awsRoleArn": {
			Description:      "AWS role ARN to be used for Federated Identity authentication with Kinesis.",
			BlastPropagation: impactInOnly,
		},
	},
	RunRevision: {
		"service": {
			ToSDPITemType:    RunService,
			Description:      "If the Run Service is deleted or updated: The Revision may lose its association or fail to run. If the Revision is updated: The service remains unaffected.",
			BlastPropagation: impactInOnly,
		},
		"vpcAccess.networkInterfaces.network": {
			ToSDPITemType:    ComputeNetwork,
			Description:      "If the Compute Network is deleted or updated: The Revision may lose connectivity or fail to run as expected. If the Revision is updated: The network remains unaffected.",
			BlastPropagation: impactInOnly,
		},
		"vpcAccess.networkInterfaces.subnetwork": {
			ToSDPITemType:    ComputeSubnetwork,
			Description:      "If the Compute Subnetwork is deleted or updated: The Revision may lose connectivity or fail to run as expected. If the Revision is updated: The subnetwork remains unaffected.",
			BlastPropagation: impactInOnly,
		},
		"vpcAccess.connector": {
			ToSDPITemType:    VPCAccessConnector,
			Description:      "If the VPC Access Connector is deleted or updated: The Revision may lose connectivity or fail to run as expected. If the Revision is updated: The connector remains unaffected.",
			BlastPropagation: impactInOnly,
		},
		"serviceAccount": iamServiceAccountImpactInOnly,
		"containers.image": {
			ToSDPITemType:    ArtifactRegistryDockerImage,
			Description:      "If the Artifact Registry Docker Image is deleted or updated: The Revision may fail to pull the image. If the Revision is updated: The Docker image remains unaffected.",
			BlastPropagation: impactInOnly,
		},
		"volumes.cloudSqlInstance.instances": {
			// Format: {project}:{location}:{instance}
			// We need a manual adapter link for this.
			ToSDPITemType:    SQLAdminInstance,
			Description:      "If the Cloud SQL Instance is deleted or updated: The Revision may fail to access the database. If the Revision is updated: The instance remains unaffected.",
			BlastPropagation: impactInOnly,
		},
		"volumes.gcs.bucket": {
			ToSDPITemType:    StorageBucket,
			Description:      "If the Cloud Storage Bucket is deleted or updated: The Revision may fail to access the GCS volume. If the Revision is updated: The bucket remains unaffected.",
			BlastPropagation: impactInOnly,
		},
		"encryptionKey": cryptoKeyImpactInOnly,
	},
	ServiceUsageService: {
		"config.name": {
			ToSDPITemType:    stdlib.NetworkDNS,
			Description:      "The DNS address at which this service is available. They are tightly coupled with the Service Usage Service.",
			BlastPropagation: impactBothWays,
		},
		"config.usage.producerNotificationChannel": {
			// Google Service Management currently only supports Google Cloud Pub/Sub as a notification channel.
			// To use Google Cloud Pub/Sub as the channel, this must be the name of a Cloud Pub/Sub topic
			ToSDPITemType:    PubSubTopic,
			Description:      "If the Pub/Sub Topic is deleted or updated: The Service Usage Service may fail to send notifications. If the Service Usage Service is updated: The topic remains unaffected.",
			BlastPropagation: impactInOnly,
		},
	},
	SQLAdminBackup: {
		"instance": {
			ToSDPITemType:    SQLAdminInstance,
			Description:      "If the Cloud SQL Instance is deleted or updated: The Backup may become invalid or inaccessible. If the Backup is updated: The instance cannot recover from the backup.",
			BlastPropagation: impactBothWays,
		},
		"kmsKey":        cryptoKeyImpactInOnly,
		"kmsKeyVersion": cryptoKeyVersionImpactInOnly,
		"backupRun": {
			ToSDPITemType:    SQLAdminBackupRun,
			Description:      "They are tightly coupled with the SQL Admin Backup.",
			BlastPropagation: impactBothWays,
		},
	},
	SQLAdminBackupRun: {
		"instance": {
			ToSDPITemType:    SQLAdminInstance,
			Description:      "They are tightly coupled",
			BlastPropagation: impactBothWays,
		},
		"diskEncryptionConfiguration.kmsKeyName": cryptoKeyImpactInOnly,
	},
}
