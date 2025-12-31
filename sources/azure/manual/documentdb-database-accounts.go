package manual

import (
	"context"
	"errors"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/cosmos/armcosmos"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/shared"
)

var (
	DocumentDBDatabaseAccountsLookupByName = shared.NewItemTypeLookup("name", azureshared.DocumentDBDatabaseAccounts)
)

type documentDBDatabaseAccountsWrapper struct {
	client clients.DocumentDBDatabaseAccountsClient

	*azureshared.ResourceGroupBase
}

func NewDocumentDBDatabaseAccounts(client clients.DocumentDBDatabaseAccountsClient, subscriptionID, resourceGroup string) sources.ListableWrapper {
	return &documentDBDatabaseAccountsWrapper{
		client: client,
		ResourceGroupBase: azureshared.NewResourceGroupBase(
			subscriptionID,
			resourceGroup,
			sdp.AdapterCategory_ADAPTER_CATEGORY_DATABASE,
			azureshared.DocumentDBDatabaseAccounts,
		),
	}
}

func (s documentDBDatabaseAccountsWrapper) List(ctx context.Context) ([]*sdp.Item, *sdp.QueryError) {
	pager := s.client.ListByResourceGroup(s.ResourceGroup())

	var items []*sdp.Item
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, azureshared.QueryError(err, s.DefaultScope(), s.Type())
		}

		for _, account := range page.Value {

			item, sdpErr := s.azureDocumentDBDatabaseAccountToSDPItem(account)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}
	return items, nil
}

func (s documentDBDatabaseAccountsWrapper) Get(ctx context.Context, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 1 {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "Get requires 1 query part: name",
			Scope:       s.DefaultScope(),
			ItemType:    s.Type(),
		}
	}
	accountName := queryParts[0]
	if accountName == "" {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "name cannot be empty",
			Scope:       s.DefaultScope(),
			ItemType:    s.Type(),
		}
	}

	resp, err := s.client.Get(ctx, s.ResourceGroup(), accountName)
	if err != nil {
		return nil, azureshared.QueryError(err, s.DefaultScope(), s.Type())
	}

	return s.azureDocumentDBDatabaseAccountToSDPItem(&resp.DatabaseAccountGetResults)
}

func (s documentDBDatabaseAccountsWrapper) azureDocumentDBDatabaseAccountToSDPItem(account *armcosmos.DatabaseAccountGetResults) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(account, "tags")
	if err != nil {
		return nil, azureshared.QueryError(err, s.DefaultScope(), s.Type())
	}

	if account.Name == nil {
		return nil, azureshared.QueryError(errors.New("name cannot be empty"), s.DefaultScope(), s.Type())
	}

	sdpItem := &sdp.Item{
		Type:            azureshared.DocumentDBDatabaseAccounts.String(),
		UniqueAttribute: "name",
		Attributes:      attributes,
		Scope:           s.DefaultScope(),
		Tags:            azureshared.ConvertAzureTags(account.Tags),
	}

	//reference : https://learn.microsoft.com/en-us/rest/api/cosmos-db-resource-provider/private-endpoint-connections/list-by-database-account?view=rest-cosmos-db-resource-provider-2025-10-15&tabs=HTTP
	if account.Properties != nil && account.Properties.PrivateEndpointConnections != nil {
		sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   azureshared.DocumentDBPrivateEndpointConnection.String(),
				Method: sdp.QueryMethod_SEARCH,
				Query:  *account.Name,
				Scope:  s.DefaultScope(),
			},
			BlastPropagation: &sdp.BlastPropagation{
				In:  true, // Private endpoint connection changes (deletion, status changes) affect the Database Account's network connectivity and accessibility
				Out: true, // Database Account deletion makes the private endpoint connection invalid, and account configuration changes may affect connection status
			}, // Private endpoint connections are tightly coupled to the Database Account - changes on either side affect connectivity and validity
		})

		// Link to Private Endpoint resources
		// Reference: https://learn.microsoft.com/en-us/rest/api/virtualnetwork/private-endpoints/get
		// GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/privateEndpoints/{privateEndpointName}
		//
		// IMPORTANT: Private Endpoints can be in a different resource group than the Cosmos DB account.
		// We must extract the subscription ID and resource group from the private endpoint's resource ID
		// to construct the correct scope.
		for _, conn := range account.Properties.PrivateEndpointConnections {
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
								Scope:  scope, // Use the private endpoint's scope, not the database account's scope
							},
							BlastPropagation: &sdp.BlastPropagation{
								In:  true, // Private endpoint changes (deletion, network configuration) affect the Database Account's private connectivity
								Out: true, // Database Account deletion or configuration changes may affect the private endpoint's connection state
							}, // Private endpoints are tightly coupled to the Database Account - changes affect connectivity
						})
					}
				}
			}
		}
	}

	// Link to Virtual Network Subnets from VirtualNetworkRules
	// Reference: https://learn.microsoft.com/en-us/rest/api/virtualnetwork/subnets/get
	// GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/virtualNetworks/{virtualNetworkName}/subnets/{subnetName}
	//
	// IMPORTANT: Virtual Network Subnets can be in a different resource group than the Cosmos DB account.
	// We must extract the subscription ID and resource group from the subnet's resource ID to construct
	// the correct scope.
	if account.Properties != nil && account.Properties.VirtualNetworkRules != nil {
		for _, vnetRule := range account.Properties.VirtualNetworkRules {
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
							Scope:  scope, // Use the subnet's scope, not the database account's scope
						},
						BlastPropagation: &sdp.BlastPropagation{
							In:  true,  // Subnet changes (deletion, network configuration) affect the Database Account's network accessibility
							Out: false, // Database Account changes don't directly affect the subnet configuration
						}, // Database Account depends on subnet for network access - subnet changes impact connectivity
					})
				}
			}
		}
	}

	// Link to Key Vault from KeyVaultKeyUri
	// Reference: https://learn.microsoft.com/en-us/rest/api/keyvault/keyvault/vaults/get?view=rest-keyvault-keyvault-2024-11-01&tabs=HTTP
	// GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.KeyVault/vaults/{vaultName}
	//
	// NOTE: Key Vaults can be in a different resource group than the Cosmos DB account. However, the Key Vault URI
	// format (https://{vaultName}.vault.azure.net/keys/{keyName}/{version}) does not contain resource group information.
	// Key Vault names are globally unique within a subscription, so we use the database account's scope as a best-effort
	// approach. If the Key Vault is in a different resource group, the query may fail and would need to be manually corrected
	// or the Key Vault adapter would need to support subscription-level search.
	if account.Properties != nil && account.Properties.KeyVaultKeyURI != nil {
		keyVaultURI := *account.Properties.KeyVaultKeyURI
		// Key Vault URI format: https://{vaultName}.vault.azure.net/keys/{keyName}/{version}
		vaultName := azureshared.ExtractVaultNameFromURI(keyVaultURI)
		if vaultName != "" {
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   azureshared.KeyVaultVault.String(),
					Method: sdp.QueryMethod_GET,
					Query:  vaultName,
					Scope:  s.DefaultScope(), // Limitation: Key Vault URI doesn't contain resource group info
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,  // Key Vault changes (key deletion, rotation, access policy) affect the Database Account's encryption
					Out: false, // Database Account changes don't directly affect the Key Vault
				}, // Database Account depends on Key Vault for encryption keys - key changes impact encryption/decryption
			})
		}
	}

	// Link to User-Assigned Managed Identities
	// Reference: https://learn.microsoft.com/en-us/rest/api/managedidentity/user-assigned-identities/list-by-resource-group?view=rest-managedidentity-2024-11-30&tabs=HTTP
	// GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ManagedIdentity/userAssignedIdentities?api-version=2024-11-30
	//
	// IMPORTANT: User-Assigned Managed Identities can be in a different resource group (or even subscription)
	// than the Cosmos DB account. User-assigned managed identities are standalone Azure resources that can
	// be assigned to multiple services across different resource groups. Therefore, we must extract the
	// subscription ID and resource group from each identity's resource ID to construct the correct scope.
	// Using the database account's scope (s.DefaultScope()) would fail if the identity is in a different
	// resource group, as the query would look in the wrong location.
	if account.Identity != nil && account.Identity.UserAssignedIdentities != nil {
		// Track scopes (subscription.resourceGroup) to avoid duplicate queries
		// Key: scope string (e.g., "subscription-id.resource-group-name")
		// Value: resource group name for the query parameter
		scopes := make(map[string]string)
		for identityID := range account.Identity.UserAssignedIdentities {
			// Identity ID format: /subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.ManagedIdentity/userAssignedIdentities/{identityName}
			// Extract subscription ID and resource group using the utility function
			params := azureshared.ExtractPathParamsFromResourceID(identityID, []string{"subscriptions", "resourceGroups"})
			if len(params) >= 2 {
				subscriptionID := params[0]
				resourceGroupName := params[1]
				// Construct scope in format: {subscriptionID}.{resourceGroupName}
				// This ensures we query the correct resource group where the identity actually exists,
				// which may be different from the database account's resource group
				scope := fmt.Sprintf("%s.%s", subscriptionID, resourceGroupName)
				// Only add one query per scope to list all identities in that resource group
				if _, exists := scopes[scope]; !exists {
					scopes[scope] = resourceGroupName
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.ManagedIdentityUserAssignedIdentity.String(),
							Method: sdp.QueryMethod_SEARCH,
							Query:  resourceGroupName,
							Scope:  scope, // Use the identity's scope, not the database account's scope
						},
						BlastPropagation: &sdp.BlastPropagation{
							In:  true,  // Identity changes (deletion, role assignments) affect the Database Account's authentication and authorization
							Out: false, // Database Account changes don't directly affect the managed identity
						}, // Database Account depends on managed identity for authentication - identity changes impact access
					})
				}
			}
		}
	}

	return sdpItem, nil
}

func (s documentDBDatabaseAccountsWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		DocumentDBDatabaseAccountsLookupByName,
	}
}

func (s documentDBDatabaseAccountsWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		azureshared.DocumentDBPrivateEndpointConnection,
		azureshared.NetworkPrivateEndpoint,
		azureshared.NetworkSubnet,
		azureshared.KeyVaultVault,
		azureshared.ManagedIdentityUserAssignedIdentity,
	)
}

func (s documentDBDatabaseAccountsWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "azurerm_cosmosdb_account.name",
		},
	}
}
