package manual

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage/v3"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/shared"
	"github.com/overmindtech/cli/sources/stdlib"
	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdpcache"
)

var StorageAccountLookupByName = shared.NewItemTypeLookup("name", azureshared.StorageAccount)

type storageAccountWrapper struct {
	client clients.StorageAccountsClient

	*azureshared.MultiResourceGroupBase
}

func NewStorageAccount(client clients.StorageAccountsClient, resourceGroupScopes []azureshared.ResourceGroupScope) sources.ListableWrapper {
	return &storageAccountWrapper{
		client: client,
		MultiResourceGroupBase: azureshared.NewMultiResourceGroupBase(
			resourceGroupScopes,
			sdp.AdapterCategory_ADAPTER_CATEGORY_STORAGE,
			azureshared.StorageAccount,
		),
	}
}

func (s storageAccountWrapper) List(ctx context.Context, scope string) ([]*sdp.Item, *sdp.QueryError) {
	rgScope, err := s.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}
	pager := s.client.NewListByResourceGroupPager(rgScope.ResourceGroup, nil)

	var items []*sdp.Item
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, azureshared.QueryError(err, scope, s.Type())
		}

		for _, account := range page.Value {
			if account.Name == nil {
				continue
			}

			item, sdpErr := s.azureStorageAccountToSDPItem(account, *account.Name, scope)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}

	return items, nil
}

func (s storageAccountWrapper) ListStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string) {
	rgScope, err := s.ResourceGroupScopeFromScope(scope)
	if err != nil {
		stream.SendError(azureshared.QueryError(err, scope, s.Type()))
		return
	}
	pager := s.client.NewListByResourceGroupPager(rgScope.ResourceGroup, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			stream.SendError(azureshared.QueryError(err, scope, s.Type()))
			return
		}
		for _, account := range page.Value {
			if account.Name == nil {
				continue
			}
			item, sdpErr := s.azureStorageAccountToSDPItem(account, *account.Name, scope)
			if sdpErr != nil {
				stream.SendError(sdpErr)
				continue
			}
			cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
			stream.SendItem(item)
		}
}
}

func (s storageAccountWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 1 {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "Get requires 1 query part: name",
			Scope:       scope,
			ItemType:    s.Type(),
		}
	}
	accountName := queryParts[0]
	if accountName == "" {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "name cannot be empty",
			Scope:       scope,
			ItemType:    s.Type(),
		}
	}

	rgScope, err := s.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}
	resp, err := s.client.Get(ctx, rgScope.ResourceGroup, accountName)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}

	return s.azureStorageAccountToSDPItem(&resp.Account, accountName, scope)
}

func (s storageAccountWrapper) azureStorageAccountToSDPItem(account *armstorage.Account, accountName, scope string) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(account, "tags")
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}

	sdpItem := &sdp.Item{
		Type:            azureshared.StorageAccount.String(),
		UniqueAttribute: "name",
		Attributes:      attributes,
		Scope:           scope,
		Tags:            azureshared.ConvertAzureTags(account.Tags),
	}

	sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   azureshared.StorageBlobContainer.String(),
			Method: sdp.QueryMethod_SEARCH,
			Query:  accountName,
			Scope:  scope,
		},
		BlastPropagation: &sdp.BlastPropagation{
			In:  false, // Storage account is NOT affected if blob containers change
			Out: true,  // Blob containers ARE affected if storage account changes/deletes
		},
	})

	sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   azureshared.StorageFileShare.String(),
			Method: sdp.QueryMethod_SEARCH,
			Query:  accountName,
			Scope:  scope,
		},
		BlastPropagation: &sdp.BlastPropagation{
			In:  false, // Storage account is NOT affected if file shares change
			Out: true,  // File shares ARE affected if storage account changes/deletes
		},
	})

	sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   azureshared.StorageTable.String(),
			Method: sdp.QueryMethod_SEARCH,
			Query:  accountName,
			Scope:  scope,
		},
		BlastPropagation: &sdp.BlastPropagation{
			In:  false, // Storage account is NOT affected if tables change
			Out: true,  // Tables ARE affected if storage account changes/deletes
		},
	})

	sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   azureshared.StorageQueue.String(),
			Method: sdp.QueryMethod_SEARCH,
			Query:  accountName,
			Scope:  scope,
		},
		BlastPropagation: &sdp.BlastPropagation{
			In:  false, // Storage account is NOT affected if queues change
			Out: true,  // Queues ARE affected if storage account changes/deletes
		},
	})

	// Link to Private Endpoint Connections (child resource)
	// Reference: https://learn.microsoft.com/en-us/rest/api/storagerp/private-endpoint-connections/list?view=rest-storagerp-2025-06-01
	// Private endpoint connections can be listed using the storage account name
	sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   azureshared.StoragePrivateEndpointConnection.String(),
			Method: sdp.QueryMethod_SEARCH,
			Query:  accountName,
			Scope:  scope,
		},
		BlastPropagation: &sdp.BlastPropagation{
			// Private endpoint connections are child resources of the storage account
			// Changes to storage account affect connections, and connection state affects storage access
			In:  true,
			Out: true,
		},
	})

	// Link to User Assigned Managed Identities (external resources)
	// Reference: https://learn.microsoft.com/en-us/rest/api/managedidentity/user-assigned-identities/get?view=rest-managedidentity-2024-11-30&tabs=HTTP
	if account.Identity != nil && account.Identity.UserAssignedIdentities != nil {
		for identityResourceID := range account.Identity.UserAssignedIdentities {
			identityName := azureshared.ExtractResourceName(identityResourceID)
			if identityName != "" {
				// Extract scope from resource ID if it's in a different resource group
				linkedScope := scope
				if extractedScope := azureshared.ExtractScopeFromResourceID(identityResourceID); extractedScope != "" {
					linkedScope = extractedScope
				}
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   azureshared.ManagedIdentityUserAssignedIdentity.String(),
						Method: sdp.QueryMethod_GET,
						Query:  identityName,
						Scope:  linkedScope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						// Storage account depends on managed identity for authentication
						// If identity is deleted/modified, storage account operations may fail
						In:  true,
						Out: false,
					},
				})
			}
		}
	}

	// Link to Key Vault (external resource) from Encryption KeyVaultProperties
	// Reference: https://learn.microsoft.com/en-us/rest/api/keyvault/keyvault/vaults/get?view=rest-keyvault-keyvault-2024-11-01&tabs=HTTP
	// GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.KeyVault/vaults/{vaultName}
	//
	// NOTE: Key Vaults can be in a different resource group than the Storage account. However, the Key Vault URI
	// format (https://{vaultName}.vault.azure.net/keys/{keyName}/{version}) does not contain resource group information.
	// Key Vault names are globally unique within a subscription, so we use the storage account's scope as a best-effort
	// approach. If the Key Vault is in a different resource group, the query may fail and would need to be manually corrected
	// or the Key Vault adapter would need to support subscription-level search.
	if account.Properties != nil && account.Properties.Encryption != nil && account.Properties.Encryption.KeyVaultProperties != nil {
		if account.Properties.Encryption.KeyVaultProperties.KeyVaultURI != nil {
			keyVaultURI := *account.Properties.Encryption.KeyVaultProperties.KeyVaultURI
			// Key Vault URI format: https://{vaultName}.vault.azure.net/keys/{keyName}/{version}
			vaultName := azureshared.ExtractVaultNameFromURI(keyVaultURI)
			if vaultName != "" {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   azureshared.KeyVaultVault.String(),
						Method: sdp.QueryMethod_GET,
						Query:  vaultName,
						Scope:  scope, // Limitation: Key Vault URI doesn't contain resource group info
					},
					BlastPropagation: &sdp.BlastPropagation{
						// Storage account depends on Key Vault for customer-managed encryption keys
						// If Key Vault is deleted/modified or key is rotated, storage account encryption may be affected
						In:  true,
						Out: false,
					},
				})
			}
		}
	}

	// Link to User Assigned Managed Identity (external resource) from Encryption EncryptionIdentity
	// Reference: https://learn.microsoft.com/en-us/rest/api/managedidentity/user-assigned-identities/get?view=rest-managedidentity-2024-11-30&tabs=HTTP
	if account.Properties != nil && account.Properties.Encryption != nil && account.Properties.Encryption.EncryptionIdentity != nil {
		if account.Properties.Encryption.EncryptionIdentity.EncryptionUserAssignedIdentity != nil {
			identityResourceID := *account.Properties.Encryption.EncryptionIdentity.EncryptionUserAssignedIdentity
			identityName := azureshared.ExtractResourceName(identityResourceID)
			if identityName != "" {
				// Extract scope from resource ID if it's in a different resource group
				linkedScope := scope
				if extractedScope := azureshared.ExtractScopeFromResourceID(identityResourceID); extractedScope != "" {
					linkedScope = extractedScope
				}
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   azureshared.ManagedIdentityUserAssignedIdentity.String(),
						Method: sdp.QueryMethod_GET,
						Query:  identityName,
						Scope:  linkedScope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						// Storage account depends on managed identity for encryption key access
						// If identity is deleted/modified, storage account encryption operations may fail
						In:  true,
						Out: false,
					},
				})
			}
		}
	}

	// Link to Subnets (external resources) from NetworkRuleSet VirtualNetworkRules
	// Reference: https://learn.microsoft.com/en-us/rest/api/virtualnetwork/subnets/get
	if account.Properties != nil && account.Properties.NetworkRuleSet != nil && account.Properties.NetworkRuleSet.VirtualNetworkRules != nil {
		for _, vnetRule := range account.Properties.NetworkRuleSet.VirtualNetworkRules {
			if vnetRule != nil && vnetRule.VirtualNetworkResourceID != nil {
				subnetID := *vnetRule.VirtualNetworkResourceID
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
							Scope:  scope, // Use the subnet's scope, not the storage account's scope
						},
						BlastPropagation: &sdp.BlastPropagation{
							// Storage account depends on subnet for network access control
							// If subnet is deleted/modified, storage account network access may be affected
							In:  true,
							Out: false,
						},
					})
				}
			}
		}
	}

	// Link to IP addresses (standard library) from NetworkRuleSet IPRules
	if account.Properties != nil && account.Properties.NetworkRuleSet != nil && account.Properties.NetworkRuleSet.IPRules != nil {
		for _, ipRule := range account.Properties.NetworkRuleSet.IPRules {
			if ipRule != nil && ipRule.IPAddressOrRange != nil && *ipRule.IPAddressOrRange != "" {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "ip",
						Method: sdp.QueryMethod_GET,
						Query:  *ipRule.IPAddressOrRange,
						Scope:  "global",
					},
					BlastPropagation: &sdp.BlastPropagation{
						// IPs are always linked
						In:  true,
						Out: true,
					},
				})
			}
		}
	}

	// Link to IP addresses (standard library) from NetworkRuleSet IPv6Rules
	if account.Properties != nil && account.Properties.NetworkRuleSet != nil && account.Properties.NetworkRuleSet.IPv6Rules != nil {
		for _, ipRule := range account.Properties.NetworkRuleSet.IPv6Rules {
			if ipRule != nil && ipRule.IPAddressOrRange != nil && *ipRule.IPAddressOrRange != "" {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "ip",
						Method: sdp.QueryMethod_GET,
						Query:  *ipRule.IPAddressOrRange,
						Scope:  "global",
					},
					BlastPropagation: &sdp.BlastPropagation{
						// IPs are always linked
						In:  true,
						Out: true,
					},
				})
			}
		}
	}

	// Link to Private Endpoints (external resources)
	// Reference: https://learn.microsoft.com/en-us/rest/api/virtualnetwork/private-endpoints/get
	if account.Properties != nil && account.Properties.PrivateEndpointConnections != nil {
		for _, peConnection := range account.Properties.PrivateEndpointConnections {
			if peConnection.Properties != nil && peConnection.Properties.PrivateEndpoint != nil && peConnection.Properties.PrivateEndpoint.ID != nil {
				privateEndpointID := *peConnection.Properties.PrivateEndpoint.ID
				privateEndpointName := azureshared.ExtractResourceName(privateEndpointID)
				if privateEndpointName != "" {
					// Extract scope from resource ID if it's in a different resource group
					linkedScope := scope
					if extractedScope := azureshared.ExtractScopeFromResourceID(privateEndpointID); extractedScope != "" {
						linkedScope = extractedScope
					}
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.NetworkPrivateEndpoint.String(),
							Method: sdp.QueryMethod_GET,
							Query:  privateEndpointName,
							Scope:  linkedScope,
						},
						BlastPropagation: &sdp.BlastPropagation{
							// Private endpoint connection is tightly coupled with the storage account
							// Changes to either affect the other
							In:  true,
							Out: true,
						},
					})
				}
			}
		}
	}

	// Link to DNS names (standard library) from PrimaryEndpoints
	if account.Properties != nil && account.Properties.PrimaryEndpoints != nil {
		endpoints := []struct {
			name  string
			value *string
		}{
			{"blob", account.Properties.PrimaryEndpoints.Blob},
			{"queue", account.Properties.PrimaryEndpoints.Queue},
			{"table", account.Properties.PrimaryEndpoints.Table},
			{"file", account.Properties.PrimaryEndpoints.File},
			{"dfs", account.Properties.PrimaryEndpoints.Dfs},
			{"web", account.Properties.PrimaryEndpoints.Web},
		}

		for _, endpoint := range endpoints {
			if endpoint.value != nil && *endpoint.value != "" {
				// Extract DNS name from URL (e.g., https://account.blob.core.windows.net/ -> account.blob.core.windows.net)
				dnsName := azureshared.ExtractDNSFromURL(*endpoint.value)
				if dnsName != "" {
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   "dns",
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
		}
	}

	// Link to DNS names (standard library) from SecondaryEndpoints
	if account.Properties != nil && account.Properties.SecondaryEndpoints != nil {
		endpoints := []struct {
			name  string
			value *string
		}{
			{"blob", account.Properties.SecondaryEndpoints.Blob},
			{"queue", account.Properties.SecondaryEndpoints.Queue},
			{"table", account.Properties.SecondaryEndpoints.Table},
			{"file", account.Properties.SecondaryEndpoints.File},
			{"dfs", account.Properties.SecondaryEndpoints.Dfs},
			{"web", account.Properties.SecondaryEndpoints.Web},
		}

		for _, endpoint := range endpoints {
			if endpoint.value != nil && *endpoint.value != "" {
				// Extract DNS name from URL (e.g., https://account-secondary.blob.core.windows.net/ -> account-secondary.blob.core.windows.net)
				dnsName := azureshared.ExtractDNSFromURL(*endpoint.value)
				if dnsName != "" {
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   "dns",
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
		}
	}

	// Link to DNS name (standard library) from CustomDomain
	if account.Properties != nil && account.Properties.CustomDomain != nil && account.Properties.CustomDomain.Name != nil && *account.Properties.CustomDomain.Name != "" {
		sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   "dns",
				Method: sdp.QueryMethod_SEARCH,
				Query:  *account.Properties.CustomDomain.Name,
				Scope:  "global",
			},
			BlastPropagation: &sdp.BlastPropagation{
				// DNS names are always linked
				In:  true,
				Out: true,
			},
		})
	}

	return sdpItem, nil
}

func (s storageAccountWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		StorageAccountLookupByName,
	}
}

// PotentialLinks returns the potential links for the storage account wrapper
func (s storageAccountWrapper) PotentialLinks() map[shared.ItemType]bool {
	return map[shared.ItemType]bool{
		// Child resources
		azureshared.StorageBlobContainer:             true,
		azureshared.StorageFileShare:                 true,
		azureshared.StorageTable:                     true,
		azureshared.StorageQueue:                     true,
		azureshared.StoragePrivateEndpointConnection: true,
		// External resources
		azureshared.ManagedIdentityUserAssignedIdentity: true,
		azureshared.KeyVaultVault:                       true,
		azureshared.NetworkSubnet:                       true,
		azureshared.NetworkPrivateEndpoint:              true,
		// Standard library types
		stdlib.NetworkIP:  true,
		stdlib.NetworkDNS: true,
	}
}

func (s storageAccountWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod: sdp.QueryMethod_GET,
			// https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/storage_account
			TerraformQueryMap: "azurerm_storage_account.name",
		},
	}
}

func (s storageAccountWrapper) IAMPermissions() []string {
	return []string{
		"Microsoft.Storage/storageAccounts/read",
	}
}

func (s storageAccountWrapper) PredefinedRole() string {
	return "Reader" // there is no predefined role for storage accounts, so we use the most restrictive role (Reader)
}
