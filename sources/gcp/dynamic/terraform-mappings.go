package dynamic

import (
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources/shared"
)

// SDPAssetTypeToTerraformMappings contains mappings from GCP asset types to Terraform resources.
// TODO: Populate SDPAssetTypeToTerraformMappings with mappings from GCP asset types to Terraform resources.
var SDPAssetTypeToTerraformMappings = map[shared.ItemType][]*sdp.TerraformMapping{}
