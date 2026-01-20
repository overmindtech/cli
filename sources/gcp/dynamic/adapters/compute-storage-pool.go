package adapters

import (
	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

// Compute Storage Pool adapter for storage pools
var _ = registerableAdapter{
	sdpType: gcpshared.ComputeStoragePool,
	meta: gcpshared.AdapterMeta{
		// Reference: https://cloud.google.com/compute/docs/reference/rest/v1/storagePools/get
		InDevelopment:      true,
		SDPAdapterCategory: sdp.AdapterCategory_ADAPTER_CATEGORY_STORAGE,
		LocationLevel:      gcpshared.ZonalLevel,
		// https://compute.googleapis.com/compute/v1/projects/{project}/zones/{zone}/storagePools/{storagePool}
		GetEndpointFunc: gcpshared.ZoneLevelEndpointFunc("https://compute.googleapis.com/compute/v1/projects/%s/zones/%s/storagePools/%s"),
		// https://compute.googleapis.com/compute/v1/projects/{project}/zones/{zone}/storagePools
		ListEndpointFunc:    gcpshared.ZoneLevelListFunc("https://compute.googleapis.com/compute/v1/projects/%s/zones/%s/storagePools"),
		UniqueAttributeKeys: []string{"storagePools"},
		IAMPermissions:      []string{"compute.storagePools.get", "compute.storagePools.list"},
		PredefinedRole:      "roles/compute.viewer",
	},
	blastPropagation: map[string]*gcpshared.Impact{
		// Link to the storage pool type that defines the characteristics of this storage pool
		"storagePoolType": {
			ToSDPItemType: gcpshared.ComputeStoragePoolType,
			Description:   "If the Storage Pool Type is deleted or updated: The Storage Pool may fail to operate correctly or become invalid. If the Storage Pool is updated: The Storage Pool Type remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{
				In:  true,
				Out: false,
			},
		},
		// Link to the zone where the storage pool resides
		"zone": {
			ToSDPItemType: gcpshared.ComputeZone,
			Description:   "If the Zone is deleted or becomes unavailable: The Storage Pool may become inaccessible. If the Storage Pool is updated: The Zone remains unaffected.",
			BlastPropagation: &sdp.BlastPropagation{
				In:  true,
				Out: false,
			},
		},
	},
	terraformMapping: gcpshared.TerraformMapping{
		Description: "There is no terraform resource for this type.",
	},
}.Register()
