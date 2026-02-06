package manual

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

var ComputeVirtualMachineExtensionLookupByName = shared.NewItemTypeLookup("name", azureshared.ComputeVirtualMachineExtension)

type computeVirtualMachineExtensionWrapper struct {
	client clients.VirtualMachineExtensionsClient

	*azureshared.MultiResourceGroupBase
}

func NewComputeVirtualMachineExtension(client clients.VirtualMachineExtensionsClient, resourceGroupScopes []azureshared.ResourceGroupScope) sources.SearchableWrapper {
	return &computeVirtualMachineExtensionWrapper{
		client: client,
		MultiResourceGroupBase: azureshared.NewMultiResourceGroupBase(
			resourceGroupScopes,
			sdp.AdapterCategory_ADAPTER_CATEGORY_COMPUTE_APPLICATION,
			azureshared.ComputeVirtualMachineExtension,
		),
	}
}

func (c computeVirtualMachineExtensionWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) != 2 {
		return nil, azureshared.QueryError(fmt.Errorf("queryParts must be 2 query parts: virtualMachineName and extensionName, got %d", len(queryParts)), scope, c.Type())
	}
	virtualMachineName := queryParts[0]
	extensionName := queryParts[1]
	if virtualMachineName == "" {
		return nil, azureshared.QueryError(fmt.Errorf("virtualMachineName cannot be empty"), scope, c.Type())
	}
	if extensionName == "" {
		return nil, azureshared.QueryError(fmt.Errorf("extensionName cannot be empty"), scope, c.Type())
	}

	rgScope, err := c.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}
	resp, err := c.client.Get(ctx, rgScope.ResourceGroup, virtualMachineName, extensionName, nil)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}

	return c.azureVirtualMachineExtensionToSDPItem(&resp.VirtualMachineExtension, virtualMachineName, extensionName, scope)
}

func (c computeVirtualMachineExtensionWrapper) azureVirtualMachineExtensionToSDPItem(extension *armcompute.VirtualMachineExtension, virtualMachineName, extensionName, scope string) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(extension, "tags")
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}

	uniqueAttr := shared.CompositeLookupKey(virtualMachineName, extensionName)
	err = attributes.Set("uniqueAttr", uniqueAttr)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}

	sdpItem := &sdp.Item{
		Type:            azureshared.ComputeVirtualMachineExtension.String(),
		UniqueAttribute: "uniqueAttr",
		Attributes:      attributes,
		Scope:           scope,
		Tags:            azureshared.ConvertAzureTags(extension.Tags),
	}

	// Link to Virtual Machine (parent resource)
	// Reference: https://learn.microsoft.com/en-us/rest/api/compute/virtual-machines/get?view=rest-compute-2025-04-01
	// GET https://management.azure.com/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Compute/virtualMachines/{vmName}?api-version=2025-04-01
	if virtualMachineName != "" {
		sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   azureshared.ComputeVirtualMachine.String(),
				Method: sdp.QueryMethod_GET,
				Query:  virtualMachineName,
				Scope:  scope,
			},
			BlastPropagation: &sdp.BlastPropagation{
				In:  true,  // If VM is deleted → Extension becomes invalid/unusable (In: true)
				Out: false, // If Extension is deleted → VM remains functional (Out: false)
			}, // Extension is a child resource of VM
		})
	}

	// Link to Key Vault for extension protected settings
	// Reference: https://learn.microsoft.com/en-us/rest/api/keyvault/keyvault/vaults/get?view=rest-keyvault-keyvault-2024-11-01
	// GET https://management.azure.com/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.KeyVault/vaults/{vaultName}?api-version=2024-11-01
	if extension.Properties != nil && extension.Properties.ProtectedSettingsFromKeyVault != nil &&
		extension.Properties.ProtectedSettingsFromKeyVault.SourceVault != nil &&
		extension.Properties.ProtectedSettingsFromKeyVault.SourceVault.ID != nil {
		vaultName := azureshared.ExtractResourceName(*extension.Properties.ProtectedSettingsFromKeyVault.SourceVault.ID)
		if vaultName != "" {
			// Check if Key Vault is in a different resource group
			extractedScope := azureshared.ExtractScopeFromResourceID(*extension.Properties.ProtectedSettingsFromKeyVault.SourceVault.ID)
			if extractedScope == "" {
				extractedScope = scope
			}
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   azureshared.KeyVaultVault.String(),
					Method: sdp.QueryMethod_GET,
					Query:  vaultName,
					Scope:  extractedScope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,  // If Key Vault changes → Extension settings access changes (In: true)
					Out: false, // If Extension is deleted → Key Vault remains (Out: false)
				}, // Extension depends on Key Vault for protected settings
			})
		}
	}

	// Link to DNS name (standard library) from SecretURL
	// Reference: https://learn.microsoft.com/en-us/rest/api/keyvault/keyvault/secrets/get-secret?view=rest-keyvault-keyvault-2024-11-01
	// SecretURL format: https://{vault}.vault.azure.net/secrets/{secret}/{version}
	if extension.Properties != nil && extension.Properties.ProtectedSettingsFromKeyVault != nil &&
		extension.Properties.ProtectedSettingsFromKeyVault.SecretURL != nil &&
		*extension.Properties.ProtectedSettingsFromKeyVault.SecretURL != "" {
		secretURL := *extension.Properties.ProtectedSettingsFromKeyVault.SecretURL
		dnsName := azureshared.ExtractDNSFromURL(secretURL)
		if dnsName != "" {
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   stdlib.NetworkDNS.String(),
					Method: sdp.QueryMethod_SEARCH,
					Query:  dnsName,
					Scope:  "global",
				},
				BlastPropagation: &sdp.BlastPropagation{
					// DNS names are always linked
					In:  true,
					Out: true,
				},
			})
		}
	}

	// Extract links from settings JSON (may contain URLs, DNS names, or IP addresses)
	// Extension settings are extension-specific JSON that may contain resource references
	if extension.Properties != nil && extension.Properties.Settings != nil {
		settingsLinks, err := sdp.ExtractLinksFrom(extension.Properties.Settings)
		if err == nil && settingsLinks != nil {
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, settingsLinks...)
			// Also extract DNS links from HTTP URLs
			for _, link := range settingsLinks {
				if link.GetQuery().GetType() == stdlib.NetworkHTTP.String() {
					dnsName := azureshared.ExtractDNSFromURL(link.GetQuery().GetQuery())
					if dnsName != "" {
						sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   stdlib.NetworkDNS.String(),
								Method: sdp.QueryMethod_SEARCH,
								Query:  dnsName,
								Scope:  "global",
							},
							BlastPropagation: &sdp.BlastPropagation{
								In:  true, // If DNS name is unavailable → Extension cannot resolve endpoint (In: true)
								Out: true, // If Extension is deleted → DNS name may still be used by other resources (Out: true)
							}, // Extension depends on DNS name for endpoint resolution
						})
					}
				}
			}
		}
	}

	// Extract links from protectedSettings JSON (may contain URLs, DNS names, or IP addresses)
	// Protected settings are encrypted but may still contain resource references
	if extension.Properties != nil && extension.Properties.ProtectedSettings != nil {
		protectedSettingsLinks, err := sdp.ExtractLinksFrom(extension.Properties.ProtectedSettings)
		if err == nil && protectedSettingsLinks != nil {
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, protectedSettingsLinks...)
			// Also extract DNS links from HTTP URLs
			for _, link := range protectedSettingsLinks {
				if link.GetQuery().GetType() == stdlib.NetworkHTTP.String() {
					dnsName := azureshared.ExtractDNSFromURL(link.GetQuery().GetQuery())
					if dnsName != "" {
						sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   stdlib.NetworkDNS.String(),
								Method: sdp.QueryMethod_SEARCH,
								Query:  dnsName,
								Scope:  "global",
							},
							BlastPropagation: &sdp.BlastPropagation{
								In:  true, // If DNS name is unavailable → Extension cannot resolve endpoint (In: true)
								Out: true, // If Extension is deleted → DNS name may still be used by other resources (Out: true)
							}, // Extension depends on DNS name for endpoint resolution
						})
					}
				}
			}
		}
	}

	return sdpItem, nil
}

func (c computeVirtualMachineExtensionWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		ComputeVirtualMachineLookupByName,
		ComputeVirtualMachineExtensionLookupByName,
	}
}

func (c computeVirtualMachineExtensionWrapper) Search(ctx context.Context, scope string, queryParts ...string) ([]*sdp.Item, *sdp.QueryError) {
	if len(queryParts) != 1 {
		return nil, azureshared.QueryError(fmt.Errorf("queryParts must be 1 query part: virtualMachineName, got %d", len(queryParts)), scope, c.Type())
	}
	virtualMachineName := queryParts[0]
	if virtualMachineName == "" {
		return nil, azureshared.QueryError(fmt.Errorf("virtualMachineName cannot be empty"), scope, c.Type())
	}

	rgScope, err := c.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}

	resp, err := c.client.List(ctx, rgScope.ResourceGroup, virtualMachineName, nil)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, c.Type())
	}

	items := make([]*sdp.Item, 0)
	for _, extension := range resp.Value {
		if extension.Name == nil {
			continue
		}
		item, err := c.azureVirtualMachineExtensionToSDPItem(extension, virtualMachineName, *extension.Name, scope)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (c computeVirtualMachineExtensionWrapper) SearchLookups() []sources.ItemTypeLookups {
	return []sources.ItemTypeLookups{
		{
			ComputeVirtualMachineLookupByName,
		},
	}
}

func (c computeVirtualMachineExtensionWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		azureshared.ComputeVirtualMachine,
		azureshared.KeyVaultVault,
		stdlib.NetworkHTTP,
		stdlib.NetworkDNS,
		stdlib.NetworkIP,
	)
}

func (c computeVirtualMachineExtensionWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_SEARCH,
			TerraformQueryMap: "azurerm_virtual_machine_extension.id",
		},
	}
}

func (c computeVirtualMachineExtensionWrapper) IAMPermissions() []string {
	return []string{
		"Microsoft.Compute/virtualMachines/extensions/read",
	}
}

func (c computeVirtualMachineExtensionWrapper) PredefinedRole() string {
	return "Reader"
}
