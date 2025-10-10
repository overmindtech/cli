package shared

import (
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources/shared"
)

type TerraformMapping struct {
	Reference   string
	Description string
	Mappings    []*sdp.TerraformMapping
}

// SDPAssetTypeToTerraformMappings maps GCP asset types to their terraform mappings.
// This map is populated during source initiation by individual adapter files.
var SDPAssetTypeToTerraformMappings = map[shared.ItemType]TerraformMapping{}
