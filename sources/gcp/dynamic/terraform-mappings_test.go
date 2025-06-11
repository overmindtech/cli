package dynamic

import (
	"testing"

	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

func TestMissingMappings(t *testing.T) {
	for sdpItemType := range gcpshared.SDPAssetTypeToAdapterMeta {
		if _, ok := SDPAssetTypeToTerraformMappings[sdpItemType]; !ok {
			t.Errorf("Missing Terraform mapping for %s", sdpItemType)
		}
	}
}
