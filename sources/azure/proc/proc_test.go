package proc

import (
	"context"
	"testing"

	// TODO: Uncomment when Azure dynamic adapters are implemented
	// _ "github.com/overmindtech/cli/sources/azure/dynamic"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
)

func Test_adapters(t *testing.T) {
	ctx := context.Background()
	discoveryAdapters, err := adapters(
		ctx,
		"subscription",
		"tenant",
		"client",
		[]string{"region"},
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

	// Check if the Compute Virtual Machine adapter is present
	// This is a key Azure adapter that should be registered
	vmAdapterFound := false
	for _, adapter := range discoveryAdapters {
		if adapter.Type() == azureshared.ComputeVirtualMachine.String() {
			vmAdapterFound = true
			break
		}
	}

	if !vmAdapterFound {
		t.Fatal("Expected to find Compute Virtual Machine adapter in the list of adapters")
	}

	t.Logf("Azure Adapters found: %v", len(discoveryAdapters))
}

func Test_ensureMandatoryFieldsInDynamicAdapters(t *testing.T) {
	// TODO: Implement this test when Azure dynamic adapters are available
	// This test validates that dynamic adapters have all required fields
	// For now, we skip it since Azure dynamic adapters may not be implemented yet
	t.Skip("Azure dynamic adapters not yet implemented")

	// TODO: Uncomment when SDPAssetTypeToAdapterMeta and PredefinedRoles are implemented for Azure
	/*
		predefinedRoles := make(map[string]bool, len(azureshared.SDPAssetTypeToAdapterMeta))
		for sdpItemType, meta := range azureshared.SDPAssetTypeToAdapterMeta {
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

				role, ok := azureshared.PredefinedRoles[meta.PredefinedRole]
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

		// roles := make([]string, 0, len(predefinedRoles))
		// for r := range azureshared.PredefinedRoles {
		// 	roles = append(roles, r)
		// }
		// sort.Strings(roles)
		// for _, r := range roles {
		// 	fmt.Println("\"" + r + "\"")
		// }
	*/
}
