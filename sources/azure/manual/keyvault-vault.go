package manual

import (
	"context"
	"errors"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/keyvault/armkeyvault/v2"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

var KeyVaultVaultLookupByName = shared.NewItemTypeLookup("name", azureshared.KeyVaultVault)

type keyvaultVaultWrapper struct {
	client clients.VaultsClient

	*azureshared.ResourceGroupBase
}

func NewKeyVaultVault(client clients.VaultsClient, subscriptionID, resourceGroup string) sources.ListableWrapper {
	return &keyvaultVaultWrapper{
		client: client,
		ResourceGroupBase: azureshared.NewResourceGroupBase(
			subscriptionID,
			resourceGroup,
			sdp.AdapterCategory_ADAPTER_CATEGORY_SECURITY,
			azureshared.KeyVaultVault,
		),
	}
}

func (k keyvaultVaultWrapper) List(ctx context.Context, scope string) ([]*sdp.Item, *sdp.QueryError) {
	resourceGroup := azureshared.ResourceGroupFromScope(scope)
	if resourceGroup == "" {
		resourceGroup = k.ResourceGroup()
	}
	pager := k.client.NewListByResourceGroupPager(resourceGroup, nil)

	var items []*sdp.Item
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, azureshared.QueryError(err, scope, k.Type())
		}

		for _, vault := range page.Value {
			if vault.Name == nil {
				continue
			}

			item, sdpErr := k.azureKeyVaultToSDPItem(vault, scope)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}

	return items, nil
}

func (k keyvaultVaultWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 1 {
		return nil, azureshared.QueryError(errors.New("Get requires 1 query part: vaultName"), scope, k.Type())
	}

	vaultName := queryParts[0]
	if vaultName == "" {
		return nil, azureshared.QueryError(errors.New("vaultName cannot be empty"), scope, k.Type())
	}

	resourceGroup := azureshared.ResourceGroupFromScope(scope)
	if resourceGroup == "" {
		resourceGroup = k.ResourceGroup()
	}
	resp, err := k.client.Get(ctx, resourceGroup, vaultName, nil)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, k.Type())
	}

	return k.azureKeyVaultToSDPItem(&resp.Vault, scope)
}

func (k keyvaultVaultWrapper) azureKeyVaultToSDPItem(vault *armkeyvault.Vault, scope string) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(vault, "tags")
	if err != nil {
		return nil, azureshared.QueryError(err, scope, k.Type())
	}

	if vault.Name == nil {
		return nil, azureshared.QueryError(errors.New("vault name is nil"), scope, k.Type())
	}

	sdpItem := &sdp.Item{
		Type:            azureshared.KeyVaultVault.String(),
		UniqueAttribute: "name",
		Attributes:      attributes,
		Scope:           scope,
		Tags:            azureshared.ConvertAzureTags(vault.Tags),
	}

	// Link to Private Endpoints from Private Endpoint Connections
	// Reference: https://learn.microsoft.com/en-us/rest/api/virtualnetwork/private-endpoints/get
	// GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/privateEndpoints/{privateEndpointName}
	//
	// IMPORTANT: Private Endpoints can be in a different resource group than the Key Vault.
	// We must extract the subscription ID and resource group from the private endpoint's resource ID
	// to construct the correct scope.
	if vault.Properties != nil && vault.Properties.PrivateEndpointConnections != nil {
		for _, conn := range vault.Properties.PrivateEndpointConnections {
			if conn.Properties != nil && conn.Properties.PrivateEndpoint != nil && conn.Properties.PrivateEndpoint.ID != nil {
				privateEndpointID := *conn.Properties.PrivateEndpoint.ID
				// Private Endpoint ID format: /subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Network/privateEndpoints/{peName}
				params := azureshared.ExtractPathParamsFromResourceID(privateEndpointID, []string{"subscriptions", "resourceGroups"})
				if len(params) >= 2 {
					subscriptionID := params[0]
					resourceGroupName := params[1]
					privateEndpointName := azureshared.ExtractResourceName(privateEndpointID)
					if privateEndpointName != "" {
						// Construct scope in format: {subscriptionID}.{resourceGroupName}
						// This ensures we query the correct resource group where the private endpoint actually exists
						scope := fmt.Sprintf("%s.%s", subscriptionID, resourceGroupName)
						sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   azureshared.NetworkPrivateEndpoint.String(),
								Method: sdp.QueryMethod_GET,
								Query:  privateEndpointName,
								Scope:  scope, // Use the private endpoint's scope, not the vault's scope
							},
							BlastPropagation: &sdp.BlastPropagation{
								In:  true, // Private endpoint changes (deletion, network configuration) affect the Key Vault's private connectivity
								Out: true, // Key Vault deletion or configuration changes may affect the private endpoint's connection state
							}, // Private endpoints are tightly coupled to the Key Vault - changes affect connectivity
						})
					}
				}
			}
		}
	}

	// Link to Virtual Network Subnets from Network ACLs
	// Reference: https://learn.microsoft.com/en-us/rest/api/virtualnetwork/subnets/get
	// GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/virtualNetworks/{virtualNetworkName}/subnets/{subnetName}
	//
	// IMPORTANT: Virtual Network Subnets can be in a different resource group than the Key Vault.
	// We must extract the subscription ID and resource group from the subnet's resource ID to construct
	// the correct scope.
	if vault.Properties != nil && vault.Properties.NetworkACLs != nil && vault.Properties.NetworkACLs.VirtualNetworkRules != nil {
		for _, vnetRule := range vault.Properties.NetworkACLs.VirtualNetworkRules {
			if vnetRule.ID != nil {
				subnetID := *vnetRule.ID
				// Subnet ID format: /subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Network/virtualNetworks/{vnetName}/subnets/{subnetName}
				// Extract subscription, resource group, virtual network name, and subnet name
				scopeParams := azureshared.ExtractPathParamsFromResourceID(subnetID, []string{"subscriptions", "resourceGroups"})
				subnetParams := azureshared.ExtractPathParamsFromResourceID(subnetID, []string{"virtualNetworks", "subnets"})
				if len(scopeParams) >= 2 && len(subnetParams) >= 2 {
					subscriptionID := scopeParams[0]
					resourceGroupName := scopeParams[1]
					vnetName := subnetParams[0]
					subnetName := subnetParams[1]
					// Subnet adapter requires: resourceGroup, virtualNetworkName, subnetName
					// Use composite lookup key to join them
					query := shared.CompositeLookupKey(vnetName, subnetName)
					// Construct scope in format: {subscriptionID}.{resourceGroupName}
					// This ensures we query the correct resource group where the subnet actually exists
					scope := fmt.Sprintf("%s.%s", subscriptionID, resourceGroupName)
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.NetworkSubnet.String(),
							Method: sdp.QueryMethod_GET,
							Query:  query,
							Scope:  scope, // Use the subnet's scope, not the vault's scope
						},
						BlastPropagation: &sdp.BlastPropagation{
							In:  true,  // Subnet changes (deletion, network configuration) affect the Key Vault's network accessibility
							Out: false, // Key Vault changes don't directly affect the subnet configuration
						}, // Key Vault depends on subnet for network access - subnet changes impact connectivity
					})
				}
			}
		}
	}

	// Link to IP addresses (standard library) from NetworkACLs IPRules
	// Reference: https://learn.microsoft.com/en-us/rest/api/keyvault/vaults/get
	if vault.Properties != nil && vault.Properties.NetworkACLs != nil && vault.Properties.NetworkACLs.IPRules != nil {
		for _, ipRule := range vault.Properties.NetworkACLs.IPRules {
			if ipRule != nil && ipRule.Value != nil && *ipRule.Value != "" {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   stdlib.NetworkIP.String(),
						Method: sdp.QueryMethod_GET,
						Query:  *ipRule.Value,
						Scope:  "global",
					},
					BlastPropagation: &sdp.BlastPropagation{
						// IPs are always linked - IP rule changes affect Key Vault network accessibility
						In:  true,
						Out: true,
					},
				})
			}
		}
	}

	// Link to stdlib.NetworkHTTP for the vault URI (HTTPS endpoint for keys and secrets operations)
	if vault.Properties != nil && vault.Properties.VaultURI != nil && *vault.Properties.VaultURI != "" {
		vaultURI := *vault.Properties.VaultURI
		sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   stdlib.NetworkHTTP.String(),
				Method: sdp.QueryMethod_SEARCH,
				Query:  vaultURI,
				Scope:  "global",
			},
			BlastPropagation: &sdp.BlastPropagation{
				// Vault endpoint connectivity affects Key Vault operations; Key Vault changes may affect endpoint
				In:  true,
				Out: true,
			},
		})
	}

	// Link to Managed HSM from HsmPoolResourceID
	// Reference: https://learn.microsoft.com/en-us/rest/api/keyvault/keyvault/managed-hsms/get
	// GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.KeyVault/managedHSMs/{name}
	//
	// IMPORTANT: Managed HSM can be in a different resource group than the Key Vault.
	// We must extract the subscription ID and resource group from the HSM Pool resource ID
	// to construct the correct scope.
	if vault.Properties != nil && vault.Properties.HsmPoolResourceID != nil {
		hsmPoolResourceID := *vault.Properties.HsmPoolResourceID
		// HSM Pool Resource ID format: /subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.KeyVault/managedHSMs/{name}
		params := azureshared.ExtractPathParamsFromResourceID(hsmPoolResourceID, []string{"subscriptions", "resourceGroups"})
		if len(params) >= 2 {
			subscriptionID := params[0]
			resourceGroupName := params[1]
			hsmName := azureshared.ExtractResourceName(hsmPoolResourceID)
			if hsmName != "" {
				// Construct scope in format: {subscriptionID}.{resourceGroupName}
				// This ensures we query the correct resource group where the Managed HSM actually exists
				scope := fmt.Sprintf("%s.%s", subscriptionID, resourceGroupName)
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   azureshared.KeyVaultManagedHSM.String(),
						Method: sdp.QueryMethod_GET,
						Query:  hsmName,
						Scope:  scope, // Use the Managed HSM's scope, not the vault's scope
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true,  // Managed HSM changes (deletion, configuration) affect the Key Vault's functionality and availability
						Out: false, // Key Vault changes don't directly affect the Managed HSM itself
					}, // Key Vault depends on Managed HSM for hardware-backed security - HSM changes impact vault operations
				})
			}
		}
	}

	return sdpItem, nil
}

func (k keyvaultVaultWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		KeyVaultVaultLookupByName,
	}
}

func (k keyvaultVaultWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "azurerm_key_vault.name",
		},
	}
}

func (k keyvaultVaultWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		azureshared.NetworkPrivateEndpoint,
		azureshared.NetworkSubnet,
		azureshared.KeyVaultManagedHSM,
		stdlib.NetworkIP,
		stdlib.NetworkHTTP,
	)
}

func (k keyvaultVaultWrapper) IAMPermissions() []string {
	return []string{
		"Microsoft.KeyVault/vaults/*/read",
	}
}

// Reference: https://learn.microsoft.com/en-us/azure/role-based-access-control/built-in-roles/security#key-vault-reader
func (k keyvaultVaultWrapper) PredefinedRole() string {
	return "Key Vault Reader"
}
