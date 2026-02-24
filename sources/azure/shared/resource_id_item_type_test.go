package shared_test

import (
	"testing"

	azureshared "github.com/overmindtech/cli/sources/azure/shared"
)

func TestCamelCaseToKebab(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"virtualNetworks", "virtualNetworks", "virtual-networks"},
		{"managedInstances", "managedInstances", "managed-instances"},
		{"applicationGateways", "applicationGateways", "application-gateways"},
		{"publicIPAddresses (acronym)", "publicIPAddresses", "public-ip-addresses"},
		{"empty", "", ""},
		{"single word lowercase", "subnet", "subnet"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := azureshared.CamelCaseToKebab(tc.input)
			if got != tc.expected {
				t.Errorf("CamelCaseToKebab(%q) = %q; want %q", tc.input, got, tc.expected)
			}
		})
	}
}

func TestSingularizeResourceType(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"virtual-networks", "virtual-networks", "virtual-network"},
		{"managed-instances", "managed-instances", "managed-instance"},
		{"galleries -> gallery", "galleries", "gallery"},
		{"user-assigned-identities -> user-assigned-identity", "user-assigned-identities", "user-assigned-identity"},
		{"public-ip-addresses -> public-ip-address", "public-ip-addresses", "public-ip-address"},
		{"no trailing s", "virtual-network", "virtual-network"},
		{"empty", "", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := azureshared.SingularizeResourceType(tc.input)
			if got != tc.expected {
				t.Errorf("SingularizeResourceType(%q) = %q; want %q", tc.input, got, tc.expected)
			}
		})
	}
}

func TestItemTypeFromLinkedResourceID(t *testing.T) {
	tests := []struct {
		name       string
		resourceID string
		expected   string
	}{
		{
			name:       "Microsoft.Network virtualNetworks",
			resourceID: "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/myRg/providers/Microsoft.Network/virtualNetworks/myVnet",
			expected:   "azure-network-virtual-network",
		},
		{
			name:       "Microsoft.Sql managedInstances",
			resourceID: "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/myRg/providers/Microsoft.Sql/managedInstances/myMI",
			expected:   "azure-sql-managed-instance",
		},
		{
			name:       "Microsoft.Compute virtualMachines",
			resourceID: "/subscriptions/sub/resourceGroups/rg/providers/Microsoft.Compute/virtualMachines/vm1",
			expected:   "azure-compute-virtual-machine",
		},
		{
			name:       "unknown provider returns empty",
			resourceID: "/subscriptions/sub/resourceGroups/rg/providers/Microsoft.Unknown/fooBars/name",
			expected:   "",
		},
		{
			name:       "empty ID returns empty",
			resourceID: "",
			expected:   "",
		},
		{
			name:       "no providers segment returns empty",
			resourceID: "/not/a/valid/resource/id",
			expected:   "",
		},
		{
			name:       "Microsoft.Compute galleries",
			resourceID: "/subscriptions/sub/resourceGroups/rg/providers/Microsoft.Compute/galleries/myGallery",
			expected:   "azure-compute-gallery",
		},
		{
			name:       "Microsoft.ManagedIdentity userAssignedIdentities",
			resourceID: "/subscriptions/sub/resourceGroups/rg/providers/Microsoft.ManagedIdentity/userAssignedIdentities/myIdentity",
			expected:   "azure-managedidentity-user-assigned-identity",
		},
		{
			name:       "Microsoft.Network publicIPAddresses (acronym)",
			resourceID: "/subscriptions/sub/resourceGroups/rg/providers/Microsoft.Network/publicIPAddresses/myPublicIP",
			expected:   "azure-network-public-ip-address",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := azureshared.ItemTypeFromLinkedResourceID(tc.resourceID)
			if got != tc.expected {
				t.Errorf("ItemTypeFromLinkedResourceID(%q) = %q; want %q", tc.resourceID, got, tc.expected)
			}
		})
	}
}
