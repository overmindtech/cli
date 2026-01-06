package adapters

import (
	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

// Compute Instance Template adapter for VM instance templates
var _ = registerableAdapter{
	sdpType: gcpshared.ComputeInstanceTemplate,
	meta: gcpshared.AdapterMeta{
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_COMPUTE_APPLICATION,
		Scope:              gcpshared.ScopeProject,
		// https://compute.googleapis.com/compute/v1/projects/{project}/global/instanceTemplates/{instanceTemplate}
		GetEndpointBaseURLFunc: gcpshared.ProjectLevelEndpointFuncWithSingleQuery("https://compute.googleapis.com/compute/v1/projects/%s/global/instanceTemplates/%s"),
		// https://compute.googleapis.com/compute/v1/projects/{project}/global/instanceTemplates
		ListEndpointFunc:    gcpshared.ProjectLevelListFunc("https://compute.googleapis.com/compute/v1/projects/%s/global/instanceTemplates"),
		UniqueAttributeKeys: []string{"instanceTemplates"},
		IAMPermissions:      []string{"compute.instanceTemplates.get", "compute.instanceTemplates.list"},
		PredefinedRole:      "roles/compute.viewer",
	},
	blastPropagation: map[string]*gcpshared.Impact{
		// https://cloud.google.com/compute/docs/reference/rest/v1/instanceTemplates/get
		"properties.networkInterfaces.network": {
			Description:      "If the network is deleted: Resources may experience connectivity changes or disruptions. If the template is deleted: Network itself is not affected.",
			ToSDPItemType:    gcpshared.ComputeNetwork,
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		"properties.networkInterfaces.subnetwork": {
			Description:      "If the (sub)network is deleted: Resources may experience connectivity changes or disruptions. If the template is updated: Subnetwork itself is not affected.",
			ToSDPItemType:    gcpshared.ComputeSubnetwork,
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		"properties.networkInterfaces.networkIP": {
			Description:      "IP address are always tightly coupled with the Compute Instance Template.",
			ToSDPItemType:    stdlib.NetworkIP,
			BlastPropagation: &sdp.BlastPropagation{In: true, Out: true},
		},
		"properties.networkInterfaces.ipv6Address":                      gcpshared.IPImpactBothWays,
		"properties.networkInterfaces.accessConfigs.natIP":              gcpshared.IPImpactBothWays,
		"properties.networkInterfaces.accessConfigs.externalIpv6":       gcpshared.IPImpactBothWays,
		"properties.networkInterfaces.accessConfigs.securityPolicy":     gcpshared.SecurityPolicyImpactInOnly,
		"properties.networkInterfaces.ipv6AccessConfigs.natIP":          gcpshared.IPImpactBothWays,
		"properties.networkInterfaces.ipv6AccessConfigs.externalIpv6":   gcpshared.IPImpactBothWays,
		"properties.networkInterfaces.ipv6AccessConfigs.securityPolicy": gcpshared.SecurityPolicyImpactInOnly,
		"properties.disks.source": {
			Description:      "If the Compute Disk is updated: Instance creation may fail or behave unexpectedly. If the template is deleted: Existing disks can be deleted.",
			ToSDPItemType:    gcpshared.ComputeDisk,
			BlastPropagation: &sdp.BlastPropagation{In: true, Out: true},
		},
		"properties.disks.initializeParams.diskName": {
			Description:      "If the Compute Disk is updated: Instance creation may fail or behave unexpectedly. If the template is deleted: Existing disks can be deleted.",
			ToSDPItemType:    gcpshared.ComputeDisk,
			BlastPropagation: &sdp.BlastPropagation{In: true, Out: true},
		},
		"properties.disks.initializeParams.sourceImage": {
			Description:      "If the Compute Image is updated: Instances created from this template may not boot correctly. If the template is updated: Image is not affected.",
			ToSDPItemType:    gcpshared.ComputeImage,
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		"properties.disks.initializeParams.diskType": {
			Description:      "If the Compute Disk Type is updated: New instances may fail to provision disks properly. If the template is updated: Disk type is not affected.",
			ToSDPItemType:    gcpshared.ComputeDiskType,
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		"properties.disks.initializeParams.sourceImageEncryptionKey.kmsKeyName":           gcpshared.CryptoKeyImpactInOnly,
		"properties.disks.initializeParams.sourceImageEncryptionKey.kmsKeyServiceAccount": gcpshared.IAMServiceAccountImpactInOnly,
		"properties.disks.initializeParams.sourceSnapshot": {
			Description:      "If the Compute Snapshot is updated: The template may reference an invalid or incompatible snapshot. If the template is updated: no impact on snapshots.",
			ToSDPItemType:    gcpshared.ComputeSnapshot,
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		"properties.disks.initializeParams.sourceSnapshotEncryptionKey.kmsKeyName":           gcpshared.CryptoKeyImpactInOnly,
		"properties.disks.initializeParams.sourceSnapshotEncryptionKey.kmsKeyServiceAccount": gcpshared.IAMServiceAccountImpactInOnly,
		"properties.disks.initializeParams.resourcePolicies":                                 gcpshared.ResourcePolicyImpactInOnly,
		"properties.disks.initializeParams.storagePool": {
			Description:      "If the Compute Storage Pool is deleted: Disk provisioning for new instances may fail. If the template is updated: Pool is not affected.",
			ToSDPItemType:    gcpshared.ComputeStoragePool,
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		"properties.disks.diskEncryptionKey.kmsKeyName":           gcpshared.CryptoKeyImpactInOnly,
		"properties.disks.diskEncryptionKey.kmsKeyServiceAccount": gcpshared.IAMServiceAccountImpactInOnly,
		"properties.guestAccelerators.acceleratorType": {
			Description:      "If the Compute Accelerator Type is updated: New instances may misconfigure or fail hardware initialization. If the template is updated: Accelerator is not affected.",
			ToSDPItemType:    gcpshared.ComputeAcceleratorType,
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		"sourceInstance": {
			Description:      "If the Compute Instance is updated: The template may reference an invalid or incompatible instance. If the template is deleted: The instance remains unaffected.",
			ToSDPItemType:    gcpshared.ComputeInstance,
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		"sourceInstanceParams.diskConfigs.customImage": {
			Description:      "If the Compute Image is updated: Instances created from this template may not boot correctly. If the template is updated: Image is not affected.",
			ToSDPItemType:    gcpshared.ComputeImage,
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		"properties.networkInterfaces.networkAttachment": {
			Description:      "If the Compute Network Attachment is updated: Instances using the template may lose access to the network services. If the template is deleted: Attachment is not affected.",
			ToSDPItemType:    gcpshared.ComputeNetworkAttachment,
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		"properties.disks.initializeParams.licenses": {
			Description:      "If the Compute License is updated: New instances may violate license agreements or lose functionality. If the template is updated: License remains unaffected.",
			ToSDPItemType:    gcpshared.ComputeLicense,
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		"properties.disks.licenses": {
			Description:      "If the Compute License is updated: New instances may violate license agreements or lose functionality. If the template is updated: License remains unaffected.",
			ToSDPItemType:    gcpshared.ComputeLicense,
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		"properties.reservationAffinity.values": {
			Description:      "If the Compute Reservation is updated: new instances created using it may fail to launch. If the template is updated: no impacts on reservation.",
			ToSDPItemType:    gcpshared.ComputeReservation,
			BlastPropagation: gcpshared.ImpactInOnly,
		},
		"properties.scheduling.nodeAffinities.values": {
			Description:      "If the Compute Node Group is updated: Placement policies may break for new VMs. If the template is updated: Node affinity rules may change. Changing the affinity might cause new VMs to stop using that Node Group",
			ToSDPItemType:    gcpshared.ComputeNodeGroup,
			BlastPropagation: &sdp.BlastPropagation{In: true, Out: true},
		},
	},
	terraformMapping: gcpshared.TerraformMapping{
		Reference: "https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/compute_instance_template",
		Mappings: []*sdp.TerraformMapping{
			{
				TerraformMethod:   sdp.QueryMethod_GET,
				TerraformQueryMap: "google_compute_instance_template.name",
			},
		},
	},
}.Register()
