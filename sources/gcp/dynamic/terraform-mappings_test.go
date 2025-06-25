package dynamic

import (
	"testing"

	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

func TestMissingMappings(t *testing.T) {
	for sdpItemType := range gcpshared.SDPAssetTypeToAdapterMeta {
		if gcpshared.SDPAssetTypeToAdapterMeta[sdpItemType].InDevelopment {
			t.Logf("Skipping %s as it is in development", sdpItemType)
			continue
		}

		if _, ok := SDPAssetTypeToTerraformMappings[sdpItemType]; !ok {
			t.Errorf("Missing Terraform mapping for %s", sdpItemType)
		}
	}
}
