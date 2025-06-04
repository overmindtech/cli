package shared

import (
	"github.com/overmindtech/cli/sources/shared"
)

// GCPResourceTypeInURLToSDPAssetType maps GCP resource types found in the item definitions,
// mostly in full or partial URLs, to SDP asset types.
// This map will be used as an attempt to find the correct SDP asset type for a GCP resource type
// if we haven't already defined it until that point.
//
// Example: "https://compute.googleapis.com/compute/v1/projects/{project}/regions/{region}/subnetworks/{subnetwork}"
// For the above URL, the GCP resource type is "subnetworks" and the SDP asset type is "ComputeSubnetwork".
var GCPResourceTypeInURLToSDPAssetType = map[string]shared.ItemType{
	"acceleratorTypes":  ComputeAcceleratorType,
	"commitments":       ComputeRegionCommitment,
	"cryptoKeyVersions": CloudKMSCryptoKeyVersion,
	"diskTypes":         ComputeDiskType,
	"disks":             ComputeDisk,
	"instanceTemplates": ComputeInstanceTemplate,
	"licenses":          ComputeLicense,
	"machineTypes":      ComputeMachineType,
	"serviceBindings":   NetworkServicesServiceBinding,
	"serviceLbPolicies": NetworkServicesServiceLbPolicy,
	"targetPools":       ComputeTargetPool,
}
