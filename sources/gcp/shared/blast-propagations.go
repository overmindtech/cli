package shared

import (
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources/shared"
)

type Impact struct {
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
	ComputeFirewall: {
		ComputeNetwork: tightCoupledImpact,
	},
	ComputeRoute: {
		ComputeNetwork: tightCoupledImpact,
	},
}
