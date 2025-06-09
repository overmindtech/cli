package shared

import (
	"github.com/overmindtech/cli/sources/shared"
)

// GCPResourceTypeInURLToSDPAssetType maps GCP resource types found in the item definitions,
// mostly in full or partial URLs, to SDP asset types.
//
// Example: "https://compute.googleapis.com/compute/v1/projects/{project}/regions/{region}/subnetworks/{subnetwork}"
// For the above URL, the GCP resource type is "subnetworks" and the SDP asset type is "ComputeSubnetwork".
var GCPResourceTypeInURLToSDPAssetType = map[string]shared.ItemType{
	"acceleratorTypes":  ComputeAcceleratorType,
	"commitments":       ComputeRegionCommitment,
	"cryptoKeyVersions": CloudKMSCryptoKeyVersion,
	"datasets":          BigQueryDataset,
	"diskTypes":         ComputeDiskType,
	"disks":             ComputeDisk,
	"firewalls":         ComputeFirewall,
	"instanceTemplates": ComputeInstanceTemplate,
	"instances":         ComputeInstance,
	"instanceSettings":  ComputeInstanceSettings,
	"licenses":          ComputeLicense,
	"machineTypes":      ComputeMachineType,
	"networks":          ComputeNetwork,
	"projects":          ComputeProject,
	"routes":            ComputeRoute,
	"serviceBindings":   NetworkServicesServiceBinding,
	"serviceLbPolicies": NetworkServicesServiceLbPolicy,
	"subnetworks":       ComputeSubnetwork,
	"subscriptions":     PubSubSubscription,
	"tables":            BigQueryTable,
	"targetPools":       ComputeTargetPool,
	"topics":            PubSubTopic,
}
