package shared

import (
	"testing"
)

func TestMissingMappings(t *testing.T) {
	for sdpItemType := range SDPAssetTypeToAdapterMeta {
		if SDPAssetTypeToAdapterMeta[sdpItemType].InDevelopment {
			t.Logf("Skipping %s as it is in development", sdpItemType)
			continue
		}

		if _, ok := SDPAssetTypeToTerraformMappings[sdpItemType]; !ok {
			t.Errorf("Missing Terraform mapping for %s", sdpItemType)
		}
	}
}
