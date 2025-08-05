package adapters

import (
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

type registerableAdapter struct { //nolint:unused
	sdpType          shared.ItemType
	meta             gcpshared.AdapterMeta
	blastPropagation map[string]*gcpshared.Impact
	terraformMapping gcpshared.TerraformMapping
}

func (d registerableAdapter) Register() registerableAdapter { //nolint:unused
	gcpshared.SDPAssetTypeToAdapterMeta[d.sdpType] = d.meta
	gcpshared.BlastPropagations[d.sdpType] = d.blastPropagation
	gcpshared.SDPAssetTypeToTerraformMappings[d.sdpType] = d.terraformMapping

	return d
}
