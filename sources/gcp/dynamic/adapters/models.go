package adapters

import (
	"github.com/overmindtech/cli/sources/gcp/dynamic"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

type dynamicAdapter struct { //nolint:unused
	sdpType          shared.ItemType
	meta             gcpshared.AdapterMeta
	blastPropagation map[string]*gcpshared.Impact
	terraformMapping dynamic.TerraformMapping
}

func (d dynamicAdapter) Register() dynamicAdapter { //nolint:unused
	gcpshared.SDPAssetTypeToAdapterMeta[d.sdpType] = d.meta
	gcpshared.BlastPropagations[d.sdpType] = d.blastPropagation
	dynamic.SDPAssetTypeToTerraformMappings[d.sdpType] = d.terraformMapping

	return d
}
