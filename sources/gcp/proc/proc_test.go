package proc

import (
	"context"
	"fmt"
	"sort"
	"testing"

	_ "github.com/overmindtech/cli/sources/gcp/dynamic"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

func Test_adapters(t *testing.T) {
	ctx := context.Background()
	discoveryAdapters, err := adapters(
		ctx,
		"project",
		[]string{"region"},
		[]string{"zone"},
		nil,
		false,
	)
	if err != nil {
		t.Fatalf("error creating adapters: %v", err)
	}

	numberOfAdapters := len(discoveryAdapters)

	if numberOfAdapters == 0 {
		t.Fatal("Expected at least one adapter, got none")
	}

	if len(Metadata.AllAdapterMetadata()) != numberOfAdapters {
		t.Fatalf("Expected %d adapters in metadata, got %d", numberOfAdapters, len(Metadata.AllAdapterMetadata()))
	}

	// Check if the Spanner adapter is present
	// Because it is created externally and it needs to be registered during the initialization of the source
	// we need to ensure that it is included in the discoveryAdapters list.
	spannerAdapterFound := false
	for _, adapter := range discoveryAdapters {
		if adapter.Type() == gcpshared.SpannerDatabase.String() {
			spannerAdapterFound = true
			break
		}
	}

	if !spannerAdapterFound {
		t.Fatal("Expected to find Spanner adapter in the list of adapters")
	}

	aiPlatformCustomJobFound := false
	for _, adapter := range discoveryAdapters {
		if adapter.Type() == gcpshared.AIPlatformCustomJob.String() {
			aiPlatformCustomJobFound = true
			break
		}
	}

	if !aiPlatformCustomJobFound {
		t.Fatal("Expected to find AIPlatform Custom Job adapter in the list of adapters")
	}

	t.Logf("GCP Adapters found: %v", len(discoveryAdapters))
}

func Test_ensureMandatoryFieldsInDynamicAdapters(t *testing.T) {
	predefinedRoles := make(map[string]bool, len(gcpshared.SDPAssetTypeToAdapterMeta))
	for sdpItemType, meta := range gcpshared.SDPAssetTypeToAdapterMeta {
		t.Run(sdpItemType.String(), func(t *testing.T) {
			if meta.InDevelopment == true {
				t.Skipf("InDevelopment is true for %s", sdpItemType.String())
			}

			if meta.GetEndpointBaseURLFunc == nil {
				t.Errorf("GetEndpointBaseURLFunc is nil for %s", sdpItemType)
			}

			if meta.Scope == "" {
				t.Errorf("Scope is empty for %s", sdpItemType)
			}

			if len(meta.UniqueAttributeKeys) == 0 {
				t.Errorf("UniqueAttributeKeys is empty for %s", sdpItemType)
			}

			if len(meta.IAMPermissions) == 0 {
				t.Errorf("IAMPermissions is empty for %s", sdpItemType)
			}

			if len(meta.PredefinedRole) == 0 {
				t.Errorf("PredefinedRoles is empty for %s", sdpItemType)
			}

			role, ok := gcpshared.PredefinedRoles[meta.PredefinedRole]
			if !ok {
				t.Errorf("PredefinedRole %s is not in the PredefinedRoles map", meta.PredefinedRole)
			}

			foundPerm := false
			for _, perm := range role.IAMPermissions {
				for _, iamPerm := range meta.IAMPermissions {
					if perm == iamPerm {
						foundPerm = true
						break
					}
				}
			}

			if !foundPerm {
				t.Errorf("IAMPermissions %s is not in the PredefinedRole %s", meta.IAMPermissions, meta.PredefinedRole)
			}

			predefinedRoles[meta.PredefinedRole] = true
		})
	}

	roles := make([]string, 0, len(predefinedRoles))
	for r := range predefinedRoles {
		roles = append(roles, r)
	}
	sort.Strings(roles)

	for _, r := range roles {
		fmt.Println("\"" + r + "\"")
	}
}
