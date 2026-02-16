package adapters

import (
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

type registerableAdapter struct {
	sdpType          shared.ItemType
	meta             gcpshared.AdapterMeta
	linkRules        map[string]*gcpshared.Impact
	terraformMapping gcpshared.TerraformMapping
}

func (d registerableAdapter) Register() registerableAdapter {
	gcpshared.SDPAssetTypeToAdapterMeta[d.sdpType] = d.meta
	gcpshared.LinkRules[d.sdpType] = d.linkRules
	gcpshared.SDPAssetTypeToTerraformMappings[d.sdpType] = d.terraformMapping

	return d
}
