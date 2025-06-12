package shared

import (
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

type Impact struct {
	ToSDPITemType    shared.ItemType
	Description      string
	BlastPropagation *sdp.BlastPropagation
}

var (
	impactInOnly        = &sdp.BlastPropagation{In: true}
	impactOutOnly       = &sdp.BlastPropagation{Out: true}
	impactBothWays      = &sdp.BlastPropagation{In: true, Out: true}
	networkImpactInOnly = Impact{
		Description:      "If the (sub)network is updated: Resources may experience connectivity changes or disruptions. If the (sub)resource is updated: The network remains unaffected.",
		BlastPropagation: impactInOnly,
	}
	resourcePolicyImpactInOnly = Impact{
		Description:      "If the Compute Resource Policy is updated: The associated resource may lose or change its policy configuration. If the resource is updated: The resource policy remains unaffected.",
		BlastPropagation: impactInOnly,
	}
	tightCoupledImpact = Impact{
		Description:      "These resources are tightly coupled and updating one may affect the other.",
		BlastPropagation: impactBothWays,
	}
	cryptoKeyVersionImpactInOnly = Impact{
		Description:      "If the Cloud KMS Crypto Key Version is updated: The associated resource may lose access to the encryption key or experience encryption changes. If the resource is updated: The crypto key version remains unaffected.",
		BlastPropagation: impactInOnly,
	}
)

var BlastPropagations = map[shared.ItemType]map[shared.ItemType]Impact{
	ComputeAutoscaler: {
		ComputeInstanceGroupManager: {
			Description:      "If the Compute Autoscaler is updated: The Instance Group Manager may lose the ability to automatically scale the number of instances until the autoscaler is reconfigured. If the Compute Instance Group Manager is updated: The associated Compute Autoscaler may become misconfigured and require manual intervention.",
			BlastPropagation: impactBothWays,
		},
	},
	ComputeInstanceTemplate: {
		ComputeNetwork:    networkImpactInOnly,
		ComputeSubnetwork: networkImpactInOnly,
	},
	ComputeNetwork: {
		ComputeSubnetwork: {
			Description:      "If the Compute Network is updated: All associated subnetworks may experience configuration changes or connectivity disruptions. If the subnetwork is updated: The network remains but resources in that subnetwork may be affected.",
			BlastPropagation: impactBothWays,
		},
		ComputeFirewall: {
			Description:      "If the Compute Network is updated: All associated firewall rules may require updates to remain compatible.",
			BlastPropagation: impactBothWays,
		},
	},
	ComputeInstanceGroupManager: {
		ComputeInstanceTemplate: {
			Description:      "If the Compute Instance Template is updated: The Instance Group Manager may not be able to create new instances with the updated template until it is reconfigured. Existing instances remain unaffected.",
			BlastPropagation: impactInOnly,
		},
		ComputeInstanceGroup: {
			Description:      "If the Compute Instance Group Manager is updated: The instance group and its instances may be updated as well.",
			BlastPropagation: impactBothWays,
		},
		ComputeResourcePolicy: resourcePolicyImpactInOnly,
		ComputeTargetPool:     tightCoupledImpact,
		ComputeAutoscaler: {
			Description:      "If the Compute Autoscaler is updated: The Instance Group Manager may lose the ability to automatically scale the number of instances until the autoscaler is reconfigured. If the Compute Instance Group Manager is updated: The associated Compute Autoscaler may become misconfigured and require manual intervention.",
			BlastPropagation: impactBothWays,
		},
	},
	ComputeDisk: {
		ComputeResourcePolicy: resourcePolicyImpactInOnly,
		ComputeDiskType: {
			Description:      "If the Compute Disk Type is updated: The disk may lose or change its type configuration. If the disk is updated: The disk type remains unaffected.",
			BlastPropagation: impactInOnly,
		},
		ComputeImage: {
			Description:      "If the Compute Image is updated: The disk may not be able to be created from that image with the new configuration. If the disk is updated: The image remains unaffected.",
			BlastPropagation: impactInOnly,
		},
		ComputeSnapshot: {
			Description:      "If the Compute Snapshot is updated: The disk may not be able to be created from that snapshot with the new configuration. If the disk is updated: The snapshot remains unaffected.",
			BlastPropagation: impactInOnly,
		},
		ComputeInstance: {
			Description:      "If the Compute Instance is updated: The disk may be updated unless it is set to 'keep' on update. If the disk is updated: The instance may lose access to that disk or experience changes.",
			BlastPropagation: impactInOnly,
		},
		ComputeDisk:              tightCoupledImpact,
		ComputeResourcePolicy:    resourcePolicyImpactInOnly,
		ComputeInstance:          tightCoupledImpact,
		CloudKMSCryptoKeyVersion: cryptoKeyVersionImpactInOnly,
		ComputeResourcePolicy:    resourcePolicyImpactInOnly,
		ComputeInstantSnapshot: {
			Description:      "",
			BlastPropagation: impactInOnly,
		},
	},
	ComputeInstance: {
		ComputeDisk: {
			Description:      "If the Compute Disk is updated: The instance may lose access to that disk or experience changes. If the instance is updated: The associated disks may also be updated unless they are set to 'keep' on update.",
			BlastPropagation: impactBothWays,
		},
		ComputeSubnetwork: networkImpactInOnly,
		ComputeNetwork:    networkImpactInOnly,
		ComputeLicense: {
			Description:      "If the Compute License is updated: The instance may lose or change its license configuration. If the instance is updated: The license remains unaffected.",
			BlastPropagation: impactInOnly,
		},
		ComputeMachineType: {
			Description:      "If the Compute Machine Type is updated: The instance may lose or change its machine type configuration. If the instance is updated: The machine type remains unaffected.",
			BlastPropagation: impactInOnly,
		},
		ComputeInstanceTemplate: {
			Description:      "If the Compute Instance Template is updated: The instance may lose or change its template configuration. If the instance is updated: The template remains unaffected.",
			BlastPropagation: impactInOnly,
		},
		ComputeInstanceGroupManager: {
			Description:      "If the Compute Instance Group Manager is updated: The instance may be updated unless it is set to 'keep' on update. If the instance is updated: The instance group manager remains unaffected.",
			BlastPropagation: impactInOnly,
		},
	},
	ComputeAddress: {
		ComputeNetwork:    networkImpactInOnly,
		ComputeSubnetwork: networkImpactInOnly,
	},
	ComputeBackendService: {
		ComputeNetwork: networkImpactInOnly,
		ComputeSecurityPolicy: {
			Description:      "If the Compute Security Policy is updated: The backend service may lose or change its security policy configuration. If the backend service is updated: The security policy remains unaffected.",
			BlastPropagation: impactInOnly,
		},
		NetworkSecurityClientTlsPolicy: {
			Description:      "If the Network Security Client TLS Policy is updated: The backend service may lose or change its client TLS configuration. If the backend service is updated: The client TLS policy remains unaffected.",
			BlastPropagation: impactInOnly,
		},
		NetworkServicesServiceLbPolicy: {
			Description:      "If the Network Services Service Load Balance Policy is updated: The backend service may lose or change its load balancing policy configuration. If the backend service is updated: The service LB policy remains unaffected.",
			BlastPropagation: impactInOnly,
		},
		NetworkServicesServiceBinding: {
			Description:      "If the Network Services Service Binding is updated: The backend service may lose or change its service binding configuration. If the backend service is updated: The service binding remains unaffected.",
			BlastPropagation: impactInOnly,
		},
	},
	ComputeForwardingRule: {
		ComputeNetwork:    networkImpactInOnly,
		ComputeSubnetwork: networkImpactInOnly,
		ComputeBackendService: {
			Description:      "If the Compute Backend Service is updated: The forwarding rule may lose or change its backend service configuration. If the forwarding rule is updated: The backend service may stop receiving traffic.",
			BlastPropagation: impactBothWays,
		},
	},
	ComputeInstanceGroup: {
		ComputeNetwork:    networkImpactInOnly,
		ComputeSubnetwork: networkImpactInOnly,
	},
	ComputeInstantSnapshot: {
		ComputeDisk: {
			Description:      "If the Compute Disk is updated: The instant snapshot remains unaffected. If the instant snapshot is updated: The disk may not be able to be restored to the point where the snapshot was taken.",
			BlastPropagation: impactOutOnly,
		},
	},
	ComputeMachineImage: {
		ComputeNetwork:    networkImpactInOnly,
		ComputeSubnetwork: networkImpactInOnly,
		ComputeDisk: {
			Description:      "If the Compute Disk is updated: The machine image may lose or change its disk configuration. If the machine image is updated: The disk remains unaffected.",
			BlastPropagation: impactInOnly,
		},
		CloudKMSCryptoKeyVersion: cryptoKeyVersionImpactInOnly,
		ComputeInstance: {
			Description:      "If the Compute Instance is updated: The machine image remains unaffected. If the machine image is updated: The instance may lose or change its configuration based on the image.",
			BlastPropagation: impactOutOnly,
		},
	},
	ComputeNodeGroup: {
		ComputeNodeTemplate: {
			Description:      "If the Compute Node Template is updated: The Node Group may lose or change its template configuration. If the Node Group is updated: The node template remains unaffected.",
			BlastPropagation: impactInOnly,
		},
	},
	ComputeReservation: {
		ComputeRegionCommitment: {
			Description:      "Updating the Compute Region Commitment does not affect the reservation, but the reservation may no longer be able to use the commitment for its resources. If the reservation is updated, the region commitment remains unaffected.",
			BlastPropagation: impactInOnly,
		},
		ComputeMachineType: {
			Description:      "If the Compute Machine Type is updated: The reservation may lose or change its machine type configuration. If the reservation is updated: The machine type remains unaffected.",
			BlastPropagation: impactInOnly,
		},
		ComputeAcceleratorType: {
			Description:      "If the Compute Accelerator Type is updated: The reservation may lose or change its accelerator type configuration. If the reservation is updated: The accelerator type remains unaffected.",
			BlastPropagation: impactInOnly,
		},
		ComputeResourcePolicy: resourcePolicyImpactInOnly,
	},
	ComputeSecurityPolicy: {
		ComputeRule: {
			Description:      "If the Compute Security Policy is updated: The associated rules may be modified, reordered, or removed, affecting how traffic is filtered. If a rule is updated: The security policy remains, but its behavior may change according to the rule modifications.",
			BlastPropagation: impactBothWays,
		},
	},
	ComputeSnapshot: {
		ComputeLicense: {
			Description:      "If the Compute License is updated: The snapshot may lose or change its license configuration. If the snapshot is updated: The license remains unaffected.",
			BlastPropagation: impactInOnly,
		},
		ComputeInstantSnapshot: {
			Description:      "If the Compute Instant Snapshot is updated: The snapshot may lose or change its configuration based on the instant snapshot. If the snapshot is updated: The instant snapshot remains unaffected.",
			BlastPropagation: impactInOnly,
		},
		CloudKMSCryptoKeyVersion: cryptoKeyVersionImpactInOnly,
		ComputeDisk: {
			Description:      "If the Compute Disk is updated: The snapshot remains unaffected. If the snapshot is updated: The disk may not be able to be restored to the point where the snapshot was taken.",
			BlastPropagation: impactOutOnly,
		},
		ComputeResourcePolicy: resourcePolicyImpactInOnly,
	},
	ComputeImage: {
		ComputeDisk: {
			Description:      "If the compute image is updated: The disk may not be able to be created from that image with the new configuration. Existing disks are unaffected.",
			BlastPropagation: impactOutOnly,
		},
	},
	ComputeInstanceTemplate: {
		// Disks can appear multiple times under properties.disks[].
		// Disk API: https://cloud.google.com/compute/docs/reference/rest/v1/disks/get
		ComputeDisk: {
			Description:      "If the Compute Disk is updated: Instance creation may fail or behave unexpectedly. If the template is deleted: Existing disks can be deleted.",
			BlastPropagation: impactBothWays,
		},
		// Network source in object: properties.networkInterfaces[].network.
		// Network API: https://cloud.google.com/compute/docs/reference/rest/v1/networks/get
		ComputeNetwork: {
			Description:      "If the network is deleted: Resources may experience connectivity changes or disruptions. If the template is deleted: Network itself is not affected.",
			BlastPropagation: impactInOnly,
		},
		// Subnetwork source in object: properties.networkInterfaces[].subnetwork.
		// Subnetwork API: https://cloud.google.com/compute/docs/reference/rest/v1/subnetworks/get
		ComputeSubnetwork: {
			Description:      "If the (sub)network is deleted: Resources may experience connectivity changes or disruptions. If the template is updated: Subnetwork itself is not affected.",
			BlastPropagation: impactInOnly,
		},
		// Machine type source in object: properties.machineType.
		// MachineType API: https://cloud.google.com/compute/docs/reference/rest/v1/machineTypes/get
		ComputeMachineType: {
			Description:      "If the Compute Machine Type is deleted: The instance template becomes partially invalid. If the template is updated: Machine type itself is not affected.",
			BlastPropagation: impactInOnly,
		},
		// Source instance used to create the template: sourceInstance.
		// Instance API: https://cloud.google.com/compute/docs/reference/rest/v1/instances/get
		ComputeInstance: {
			Description:      "If the Compute Instance is updated: The template remains unaffected. If the template is deleted: recreation of instances that use the template may not be possible.",
			BlastPropagation: impactOutOnly,
		},
		// Image source in object: properties.disks[].initializeParams.sourceImage.
		// Image API: https://cloud.google.com/compute/docs/reference/rest/v1/images/get
		ComputeImage: {
			Description:      "If the Compute Image is updated: Instances created from this template may not boot correctly. If the template is updated: Image is not affected.",
			BlastPropagation: impactInOnly,
		},
		// Snapshot source in object: properties.disks[].initializeParams.sourceSnapshot.
		// Snapshot API: https://cloud.google.com/compute/docs/reference/rest/v1/snapshots/get
		ComputeSnapshot: {
			Description:      "If the Compute Snapshot is updated: The template may reference an invalid or incompatible snapshot. If the template is updated: no impact on snapshots.",
			BlastPropagation: impactInOnly,
		},
		// Static IP (natIP) source: properties.networkInterfaces[].accessConfigs[].natIP.
		// Address API: https://cloud.google.com/compute/docs/reference/rest/v1/addresses/get
		ComputeAddress: {
			Description:      "If the Compute Address is updated: The template’s networking may break for new instances. If the template is updated: no impacts, static address is not dependent on the template.",
			BlastPropagation: impactInOnly,
		},
		// Security policy source: properties.networkInterfaces[].accessConfigs[].securityPolicy.
		// SecurityPolicy API: https://cloud.google.com/compute/docs/reference/rest/v1/securityPolicies/get
		ComputeSecurityPolicy: {
			Description:      "If the Compute Security Policy is updated: Access configurations for new instances may break. If the template is deleted: Security policy is not affected.",
			BlastPropagation: impactInOnly,
		},
		// Network attachment source: properties.networkInterfaces[].networkAttachment.
		// NetworkAttachment API: https://cloud.google.com/compute/docs/reference/rest/v1/networkAttachments/get
		ComputeNetworkAttachment: {
			Description:      "If the Compute Network Attachment is updated: Instances using the template may lose access to the network services. If the template is deleted: Attachment is not affected.",
			BlastPropagation: impactInOnly,
		},
		// KMS key version (image/snapshot/disk): properties.disks[].initializeParams.sourceImageEncryptionKey.kmsKeyName,
		// and similar paths for sourceSnapshotEncryptionKey and diskEncryptionKey.
		// KMS API: https://cloud.google.com/kms/docs/reference/rest/v1/projects.locations.keyRings.cryptoKeys.cryptoKeyVersions/get
		CloudKMSCryptoKeyVersion: {
			Description:      "If the Cloud KMS Crypto Key Version is updated: The associated resource may lose access to the encryption key or experience encryption changes. If the resource is updated: The crypto key version remains unaffected.",
			BlastPropagation: impactInOnly,
		},
		// Disk type source: properties.disks[].initializeParams.diskType.
		// DiskType API: https://cloud.google.com/compute/docs/reference/rest/v1/diskTypes/get
		ComputeDiskType: {
			Description:      "If the Compute Disk Type is updated: New instances may fail to provision disks properly. If the template is updated: Disk type is not affected.",
			BlastPropagation: impactInOnly,
		},
		// IAM service account source: properties.serviceAccounts[].email.
		// IAM API: https://cloud.google.com/iam/docs/reference/rest/v1/projects.serviceAccounts/get
		IAMServiceAccount: {
			Description:      "If the IAM Service Account is updated: New instances may lose permissions. If the template is updated: Account is not impacted.",
			BlastPropagation: impactInOnly,
		},
		// Resource policy source: properties.resourcePolicies[], or properties.disks[].initializeParams.resourcePolicies[].
		// ResourcePolicy API: https://cloud.google.com/compute/docs/reference/rest/v1/resourcePolicies/get
		ComputeResourcePolicy: {
			Description:      "If the Compute Resource Policy is updated: The associated resource may lose or change its policy configuration. If the resource is updated: The resource policy remains unaffected.",
			BlastPropagation: impactInOnly,
		},
		// License source: properties.disks[].initializeParams.licenses[], or properties.disks[].licenses[].
		// License API: https://cloud.google.com/compute/docs/reference/rest/v1/licenses/get
		ComputeLicense: {
			Description:      "If the Compute License is updated: New instances may violate license agreements or lose functionality. If the template is updated: License remains unaffected..",
			BlastPropagation: impactInOnly,
		},
		// Storage pool source in object: properties.disks[].initializeParams.storagePool.
		// Storage pool API: https://cloud.google.com/compute/docs/reference/rest/v1/storagePools/get
		ComputeStoragePool: {
			Description:      "If the Compute Storage Pool is deleted: Disk provisioning for new instances may fail. If the template is updated: Pool is not affected.",
			BlastPropagation: impactInOnly,
		},
		// Node group (possible match via node affinities): properties.scheduling.nodeAffinities[].key (when referencing node groups).
		// NodeGroup API: https://cloud.google.com/compute/docs/reference/rest/v1/nodeGroups/get
		ComputeNodeGroup: {
			Description:      "If the Compute Node Group is updated: Placement policies may break for new VMs. If the template is updated: Node affinity rules may change. Changing the affinity might cause new VMs to stop using that Node Group",
			BlastPropagation: impactBothWays,
		},
		// Accelerator type source: properties.guestAccelerators[].acceleratorType.
		// AcceleratorType API: https://cloud.google.com/compute/docs/reference/rest/v1/acceleratorTypes/get
		ComputeAcceleratorType: {
			Description:      "If the Compute Accelerator Type is updated: New instances may misconfigure or fail hardware initialization. If the template is updated: Accelerator is not affected.",
			BlastPropagation: impactInOnly,
		},
		// Reservation source (if used with reservation affinity): properties.reservationAffinity.key and values.
		// Reservation API: https://cloud.google.com/compute/docs/reference/rest/v1/reservations/get
		ComputeReservation: {
			Description:      "If the Compute Reservation is updated: new instances created using it may fail to launch. If the template is updated: no impacts on reservation.",
			BlastPropagation: impactInOnly,
		},
	},
	ComputeRoute: {
		// Network source in object: network.
		// Network API: https://cloud.google.com/compute/docs/reference/rest/v1/networks/get
		ComputeNetwork: {
			Description:      "If the Compute Network is updated: The route may no longer be valid or correctly associated. If the route is updated: The network remains unaffected, but its routing behavior may change.",
			BlastPropagation: impactBothWays,
		},
		// Next hop instance source: nextHopInstance.
		// Instance API: https://cloud.google.com/compute/docs/reference/rest/v1/instances/get
		ComputeInstance: {
			Description:      "If the Compute Instance is updated: Routes using it as a next hop may break or change behavior. If the route is deleted: The instance remains unaffected but traffic that was previously using that route will be impacted.",
			BlastPropagation: impactInOnly,
		},
		// VPN tunnel source: nextHopVpnTunnel.
		// VPN Tunnel API: https://cloud.google.com/compute/docs/reference/rest/v1/vpnTunnels/get
		ComputeVpnTunnel: {
			Description:      "If the VPN Tunnel is updated: The route may no longer forward traffic properly. If the route is updated: The VPN tunnel remains unaffected but traffic routed through it may be affected.",
			BlastPropagation: impactBothWays,
		},
		// Gateway source: nextHopGateway.
		// Gateway API: https://cloud.google.com/vpc/docs/routes#default-internet-gateway
		// Link to the property: https://cloud.google.com/compute/docs/reference/rest/v1/routes/get#:~:text=handle%20matching%20packets.-,nextHopGateway,-string
		// https://www.googleapis.com/compute/v1/projects/my-project/global/gateways/default-internet-gateway
		ComputeGateway: {
			Description:      "If the Compute Gateway is updated: The route may no longer forward traffic properly. If the route is updated: The gateway remains unaffected but traffic routed through it may be affected.",
			BlastPropagation: impactInOnly,
		},
		//
		// Network peering source: nextHopPeering.
		// Peering API: https://cloud.google.com/compute/docs/reference/rest/v1/networks/get
		ComputeNetworkPeering: {
			Description:      "If the peering connection is deleted: The route may no longer be valid or usable. If the route is updated: Peering remains unaffected.",
			BlastPropagation: impactInOnly,
		},
		// Network Connectivity Center hub source: nextHopHub.
		// Couldn't find a valid API to query this, but seems like a valid link
		ComputeForwardingRule: {
			Description:      "If the Compute Forwarding Rule is updated: The route may no longer forward traffic properly. If the route is updated: The forwarding rule remains unaffected but traffic routed through it may be affected.",
			BlastPropagation: impactInOnly,
		},
	},
	ComputeFirewall: {
		ComputeNetwork: tightCoupledImpact,
		// Service account source in object: sourceServiceAccounts and targetServiceAccounts.
		// IAM API: https://cloud.google.com/iam/docs/reference/rest/v1/projects.serviceAccounts/get
		IAMServiceAccount: {
			Description:      "If the IAM Service Account is deleted: firewall may fail to work as before. If the firewall is deleted: service account is not impacted.",
			BlastPropagation: impactInOnly,
		},
	},
	ComputeSubnetwork: {
		ComputeNetwork: {
			Description:      "If the Compute Network is updated: All associated subnetworks are impacted. If the subnetwork is updated: The network remains working.",
			BlastPropagation: impactInOnly,
		},
		//"reservedInternalRange" and "secondaryIpRanges[].reservedInternalRange"
		ComputeAddress: {
			Description:      "If the Compute Address is deleted: subnetwork is not impacted. If the subnetwork is deleted: internal IP Addresses allocated from it are also deleted.",
			BlastPropagation: impactOutOnly,
		},
		ComputeGateway: {
			Description:      "If the Compute Gateway is deleted: subnetwork is not impacted. If the subnetwork is deleted: gateway is also deleted.",
			BlastPropagation: impactOutOnly,
		},
	},
	ComputeProject: {
		// Bucket source in object: usageExportLocation.bucketName.
		// Storage Bucket API: https://cloud.google.com/storage/docs/json_api/v1/buckets/get
		StorageBucket: {
			Description:      "If the Storage Bucket is deleted: project still works. If the Project is deleted: the bucket is deleted.",
			BlastPropagation: impactOutOnly,
		},
		// Service account source in object: defaultServiceAccount.
		// IAM API: https://cloud.google.com/iam/docs/reference/rest/v1/projects.serviceAccounts/get
		IAMServiceAccount: {
			Description:      "If the IAM Service Account is deleted: Project resources may fail to work as before. If the project is deleted: service account is deleted.",
			BlastPropagation: impactBothWays,
		},
	},
	IAMServiceAccount: {
		IAMServiceAccountKey: {
			Description:      "If the service account is deleted: All keys that belong to it are deleted. If the service account key is deleted: Resources using that particular key lose access to the service account  but account still works.",
			BlastPropagation: impactOutOnly,
		},
	},
	IAMServiceAccountKey: {
		IAMServiceAccount: {
			Description:      "If the service account key is deleted: Resources using that particular key lose access to the service account but account still works. If the service account is deleted: All keys that belong to it are deleted.",
			BlastPropagation: impactInOnly,
		},
	},
	CloudResourceManagerProject: {
		IAMServiceAccount: tightCoupledImpact,
	},
	CloudKMSKeyRing: {
		IAMPolicy: tightCoupledImpact,
	},
	IAMPolicy: {
		IAMPolicy: tightCoupledImpact,
	},
}

var ExplicitBlastPropagations = map[shared.ItemType]map[string]Impact{
	// https://cloud.google.com/compute/docs/reference/rest/v1/routes/get
	ComputeRoute: {
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
	},
	// https://cloud.google.com/compute/docs/reference/rest/v1/instanceTemplates/get
	ComputeInstanceTemplate: {
		"properties.disks.initializeParams.sourceImage": {
			Description:      "If the Compute Image is updated: Instances created from this template may not boot correctly. If the template is updated: Image is not affected.",
			ToSDPITemType:    ComputeImage,
			BlastPropagation: impactInOnly,
		},
	},
}
