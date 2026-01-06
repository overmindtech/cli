package manual

import (
	"context"
	"errors"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/batch/armbatch"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

var (
	BatchAccountLookupByName = shared.NewItemTypeLookup("name", azureshared.BatchBatchAccount)
)

type batchAccountWrapper struct {
	client clients.BatchAccountsClient

	*azureshared.ResourceGroupBase
}

func NewBatchAccount(client clients.BatchAccountsClient, subscriptionID, resourceGroup string) sources.ListableWrapper {
	return &batchAccountWrapper{
		client: client,
		ResourceGroupBase: azureshared.NewResourceGroupBase(
			subscriptionID,
			resourceGroup,
			sdp.AdapterCategory_ADAPTER_CATEGORY_COMPUTE_APPLICATION,
			azureshared.BatchBatchAccount,
		),
	}
}

func (b batchAccountWrapper) List(ctx context.Context) ([]*sdp.Item, *sdp.QueryError) {
	pager := b.client.ListByResourceGroup(ctx, b.ResourceGroup())

	var items []*sdp.Item
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, azureshared.QueryError(err, b.DefaultScope(), b.Type())
		}

		for _, account := range page.Value {
			if account.Name == nil {
				continue
			}

			item, sdpErr := b.azureBatchAccountToSDPItem(account)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}

	return items, nil
}

func (b batchAccountWrapper) azureBatchAccountToSDPItem(account *armbatch.Account) (*sdp.Item, *sdp.QueryError) {
	if account.Name == nil {
		return nil, azureshared.QueryError(errors.New("name is nil"), b.DefaultScope(), b.Type())
	}
	attributes, err := shared.ToAttributesWithExclude(account, "tags")
	if err != nil {
		return nil, azureshared.QueryError(err, b.DefaultScope(), b.Type())
	}

	sdpItem := &sdp.Item{
		Type:            azureshared.BatchBatchAccount.String(),
		UniqueAttribute: "name",
		Attributes:      attributes,
		Scope:           b.DefaultScope(),
		Tags:            azureshared.ConvertAzureTags(account.Tags),
	}

	accountName := *account.Name

	// Link to Storage Account (external resource)
	// Reference: https://learn.microsoft.com/en-us/rest/api/storagerp/storage-accounts/get-properties
	if account.Properties != nil && account.Properties.AutoStorage != nil && account.Properties.AutoStorage.StorageAccountID != nil {
		storageAccountID := *account.Properties.AutoStorage.StorageAccountID
		storageAccountName := azureshared.ExtractResourceName(storageAccountID)
		if storageAccountName != "" {
			// Extract scope from resource ID if it's in a different resource group
			scope := b.DefaultScope()
			if extractedScope := azureshared.ExtractScopeFromResourceID(storageAccountID); extractedScope != "" {
				scope = extractedScope
			}
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   azureshared.StorageAccount.String(),
					Method: sdp.QueryMethod_GET,
					Query:  storageAccountName,
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					// Batch account depends on storage account for auto-storage functionality
					// If storage account is deleted/modified, batch account operations may fail
					In:  true,
					Out: false,
				},
			})
		}
	}

	// Link to Key Vault (external resource) from KeyVaultReference
	// Reference: https://learn.microsoft.com/en-us/rest/api/keyvault/keyvault/vaults/get?view=rest-keyvault-keyvault-2024-11-01&tabs=HTTP
	if account.Properties != nil && account.Properties.KeyVaultReference != nil && account.Properties.KeyVaultReference.ID != nil {
		keyVaultID := *account.Properties.KeyVaultReference.ID
		keyVaultName := azureshared.ExtractResourceName(keyVaultID)
		if keyVaultName != "" {
			// Extract scope from resource ID if it's in a different resource group
			scope := b.DefaultScope()
			if extractedScope := azureshared.ExtractScopeFromResourceID(keyVaultID); extractedScope != "" {
				scope = extractedScope
			}
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   azureshared.KeyVaultVault.String(),
					Method: sdp.QueryMethod_GET,
					Query:  keyVaultName,
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					// Batch account depends on Key Vault for certificate management and encryption
					// If Key Vault is deleted/modified, batch account operations may fail
					In:  true,
					Out: false,
				},
			})
		}
	}

	// Link to Key Vault (external resource) from Encryption KeyVaultProperties
	// Reference: https://learn.microsoft.com/en-us/rest/api/keyvault/keyvault/vaults/get?view=rest-keyvault-keyvault-2024-11-01&tabs=HTTP
	// GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.KeyVault/vaults/{vaultName}
	//
	// NOTE: Key Vaults can be in a different resource group than the Batch account. However, the Key Vault URI
	// format (https://{vaultName}.vault.azure.net/keys/{keyName}/{version}) does not contain resource group information.
	// Key Vault names are globally unique within a subscription, so we use the batch account's scope as a best-effort
	// approach. If the Key Vault is in a different resource group, the query may fail and would need to be manually corrected
	// or the Key Vault adapter would need to support subscription-level search.
	if account.Properties != nil && account.Properties.Encryption != nil && account.Properties.Encryption.KeyVaultProperties != nil {
		if account.Properties.Encryption.KeyVaultProperties.KeyIdentifier != nil {
			keyIdentifier := *account.Properties.Encryption.KeyVaultProperties.KeyIdentifier
			// Key Vault URI format: https://{vaultName}.vault.azure.net/keys/{keyName}/{version}
			vaultName := azureshared.ExtractVaultNameFromURI(keyIdentifier)
			if vaultName != "" {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   azureshared.KeyVaultVault.String(),
						Method: sdp.QueryMethod_GET,
						Query:  vaultName,
						Scope:  b.DefaultScope(), // Limitation: Key Vault URI doesn't contain resource group info
					},
					BlastPropagation: &sdp.BlastPropagation{
						// Batch account depends on Key Vault for customer-managed encryption keys
						// If Key Vault is deleted/modified or key is rotated, batch account encryption may be affected
						In:  true,
						Out: false,
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
					scope := b.DefaultScope()
					if extractedScope := azureshared.ExtractScopeFromResourceID(privateEndpointID); extractedScope != "" {
						scope = extractedScope
					}
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.NetworkPrivateEndpoint.String(),
							Method: sdp.QueryMethod_GET,
							Query:  privateEndpointName,
							Scope:  scope,
						},
						BlastPropagation: &sdp.BlastPropagation{
							// Private endpoint connection is tightly coupled with the batch account
							// Changes to either affect the other
							In:  true,
							Out: true,
						},
					})
				}
			}
		}
	}

	// Link to User Assigned Managed Identities (external resources)
	// Reference: https://learn.microsoft.com/en-us/rest/api/managedidentity/user-assigned-identities/get?view=rest-managedidentity-2024-11-30&tabs=HTTP
	if account.Identity != nil && account.Identity.UserAssignedIdentities != nil {
		for identityResourceID := range account.Identity.UserAssignedIdentities {
			identityName := azureshared.ExtractResourceName(identityResourceID)
			if identityName != "" {
				// Extract scope from resource ID if it's in a different resource group
				scope := b.DefaultScope()
				if extractedScope := azureshared.ExtractScopeFromResourceID(identityResourceID); extractedScope != "" {
					scope = extractedScope
				}
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   azureshared.ManagedIdentityUserAssignedIdentity.String(),
						Method: sdp.QueryMethod_GET,
						Query:  identityName,
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						// Batch account depends on managed identity for authentication
						// If identity is deleted/modified, batch account operations may fail
						In:  true,
						Out: false,
					},
				})
			}
		}
	}

	// Link to Node Identity Reference (external resource)
	// Reference: https://learn.microsoft.com/en-us/rest/api/managedidentity/user-assigned-identities/get?view=rest-managedidentity-2024-11-30&tabs=HTTP
	if account.Properties != nil && account.Properties.AutoStorage != nil && account.Properties.AutoStorage.NodeIdentityReference != nil && account.Properties.AutoStorage.NodeIdentityReference.ResourceID != nil {
		nodeIdentityID := *account.Properties.AutoStorage.NodeIdentityReference.ResourceID
		nodeIdentityName := azureshared.ExtractResourceName(nodeIdentityID)
		if nodeIdentityName != "" {
			// Extract scope from resource ID if it's in a different resource group
			scope := b.DefaultScope()
			if extractedScope := azureshared.ExtractScopeFromResourceID(nodeIdentityID); extractedScope != "" {
				scope = extractedScope
			}
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   azureshared.ManagedIdentityUserAssignedIdentity.String(),
					Method: sdp.QueryMethod_GET,
					Query:  nodeIdentityName,
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					// Batch account compute nodes depend on managed identity for auto-storage access
					// If identity is deleted/modified, compute nodes may fail to access storage
					In:  true,
					Out: false,
				},
			})
		}
	}

	// Link to Applications (child resource)
	// Reference: https://learn.microsoft.com/en-us/rest/api/batchmanagement/application/list?view=rest-batchmanagement-2024-07-01&tabs=HTTP
	// Applications can be listed using the batch account name
	sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   azureshared.BatchBatchApplication.String(),
			Method: sdp.QueryMethod_SEARCH,
			Query:  accountName,
			Scope:  b.DefaultScope(),
		},
		BlastPropagation: &sdp.BlastPropagation{
			// Applications are child resources of the batch account
			// Changes to batch account affect applications, and vice versa
			In:  true,
			Out: true,
		},
	})

	// Link to Pools (child resource)
	// Reference: https://learn.microsoft.com/en-us/rest/api/batchmanagement/pool/list-by-batch-account?view=rest-batchmanagement-2024-07-01&tabs=HTTP
	// Pools can be listed using the batch account name
	sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   azureshared.BatchBatchPool.String(),
			Method: sdp.QueryMethod_SEARCH,
			Query:  accountName,
			Scope:  b.DefaultScope(),
		},
		BlastPropagation: &sdp.BlastPropagation{
			// Pools are child resources of the batch account
			// Changes to batch account affect pools, and vice versa
			In:  true,
			Out: true,
		},
	})

	// Link to Certificates (child resource)
	// Reference: https://learn.microsoft.com/en-us/rest/api/batchmanagement/certificate/list-by-batch-account?view=rest-batchmanagement-2024-07-01&tabs=HTTP
	// Certificates can be listed using the batch account name
	sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   azureshared.BatchBatchCertificate.String(),
			Method: sdp.QueryMethod_SEARCH,
			Query:  accountName,
			Scope:  b.DefaultScope(),
		},
		BlastPropagation: &sdp.BlastPropagation{
			// Certificates are child resources of the batch account
			// Changes to batch account affect certificates, and vice versa
			In:  true,
			Out: true,
		},
	})

	// Link to Private Endpoint Connections (child resource)
	// Reference: https://learn.microsoft.com/en-us/rest/api/batchmanagement/private-endpoint-connection/list-by-batch-account?view=rest-batchmanagement-2024-07-01&tabs=HTTP
	// Private endpoint connections can be listed using the batch account name
	sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   azureshared.BatchBatchPrivateEndpointConnection.String(),
			Method: sdp.QueryMethod_SEARCH,
			Query:  accountName,
			Scope:  b.DefaultScope(),
		},
		BlastPropagation: &sdp.BlastPropagation{
			// Private endpoint connections are child resources of the batch account
			// Changes to batch account affect connections, and vice versa
			In:  true,
			Out: true,
		},
	})

	// Link to Private Link Resources (child resource)
	// Reference: https://learn.microsoft.com/en-us/rest/api/batchmanagement/private-link-resource/list-by-batch-account?view=rest-batchmanagement-2024-07-01&tabs=HTTP
	// Private link resources can be listed using the batch account name
	sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   azureshared.BatchBatchPrivateLinkResource.String(),
			Method: sdp.QueryMethod_SEARCH,
			Query:  accountName,
			Scope:  b.DefaultScope(),
		},
		BlastPropagation: &sdp.BlastPropagation{
			// Private link resources are child resources of the batch account
			// Changes to batch account affect resources, and vice versa
			In:  true,
			Out: true,
		},
	})

	// Link to Detectors (child resource)
	// Reference: https://learn.microsoft.com/en-us/rest/api/batchmanagement/batch-account/list-detectors?view=rest-batchmanagement-2024-07-01&tabs=HTTP
	// Detectors can be listed using the batch account name
	sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   azureshared.BatchBatchDetector.String(),
			Method: sdp.QueryMethod_SEARCH,
			Query:  accountName,
			Scope:  b.DefaultScope(),
		},
		BlastPropagation: &sdp.BlastPropagation{
			// Detectors are child resources of the batch account
			// Changes to batch account affect detectors, and vice versa
			In:  true,
			Out: true,
		},
	})

	// Link to DNS name (standard library) if AccountEndpoint is configured
	if account.Properties != nil && account.Properties.AccountEndpoint != nil && *account.Properties.AccountEndpoint != "" {
		sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   "dns",
				Method: sdp.QueryMethod_SEARCH,
				Query:  *account.Properties.AccountEndpoint,
				Scope:  "global",
			},
			BlastPropagation: &sdp.BlastPropagation{
				// DNS names are always linked
				In:  true,
				Out: true,
			},
		})
	}

	// Link to IP addresses (standard library) from NetworkProfile AccountAccess IPRules
	if account.Properties != nil && account.Properties.NetworkProfile != nil && account.Properties.NetworkProfile.AccountAccess != nil {
		if account.Properties.NetworkProfile.AccountAccess.IPRules != nil {
			for _, ipRule := range account.Properties.NetworkProfile.AccountAccess.IPRules {
				if ipRule != nil && ipRule.Value != nil && *ipRule.Value != "" {
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   "ip",
							Method: sdp.QueryMethod_GET,
							Query:  *ipRule.Value,
							Scope:  "global",
						},
						BlastPropagation: &sdp.BlastPropagation{
							// Batch account depends on IP rules for network access control
							// If IP rules change, batch account access may be affected
							In:  true,
							Out: false,
						},
					})
				}
			}
		}
	}

	// Link to IP addresses (standard library) from NetworkProfile NodeManagementAccess IPRules
	if account.Properties != nil && account.Properties.NetworkProfile != nil && account.Properties.NetworkProfile.NodeManagementAccess != nil {
		if account.Properties.NetworkProfile.NodeManagementAccess.IPRules != nil {
			for _, ipRule := range account.Properties.NetworkProfile.NodeManagementAccess.IPRules {
				if ipRule != nil && ipRule.Value != nil && *ipRule.Value != "" {
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   "ip",
							Method: sdp.QueryMethod_GET,
							Query:  *ipRule.Value,
							Scope:  "global",
						},
						BlastPropagation: &sdp.BlastPropagation{
							// Batch account depends on IP rules for node management access control
							// If IP rules change, batch account node management access may be affected
							In:  true,
							Out: false,
						},
					})
				}
			}
		}
	}

	return sdpItem, nil
}

func (b batchAccountWrapper) Get(ctx context.Context, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 1 {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "Get requires 1 query part: accountName",
			Scope:       b.DefaultScope(),
			ItemType:    b.Type(),
		}
	}
	accountName := queryParts[0]

	if accountName == "" {
		return nil, azureshared.QueryError(errors.New("accountName is empty"), b.DefaultScope(), b.Type())
	}

	resp, err := b.client.Get(ctx, b.ResourceGroup(), accountName)
	if err != nil {
		return nil, azureshared.QueryError(err, b.DefaultScope(), b.Type())
	}

	return b.azureBatchAccountToSDPItem(&resp.Account)
}

func (b batchAccountWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		BatchAccountLookupByName,
	}
}

func (b batchAccountWrapper) PotentialLinks() map[shared.ItemType]bool {
	return map[shared.ItemType]bool{
		// External resources
		azureshared.StorageAccount:                      true,
		azureshared.KeyVaultVault:                       true,
		azureshared.NetworkPrivateEndpoint:              true,
		azureshared.ManagedIdentityUserAssignedIdentity: true,
		// Child resources
		azureshared.BatchBatchApplication:               true,
		azureshared.BatchBatchPool:                      true,
		azureshared.BatchBatchCertificate:               true,
		azureshared.BatchBatchPrivateEndpointConnection: true,
		azureshared.BatchBatchPrivateLinkResource:       true,
		azureshared.BatchBatchDetector:                  true,
		// DNS
		stdlib.NetworkDNS: true,
		// IP
		stdlib.NetworkIP: true,
	}
}

func (b batchAccountWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod: sdp.QueryMethod_GET,
			// https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/batch_account
			TerraformQueryMap: "azurerm_batch_account.name",
		},
	}
}

// ref : https://learn.microsoft.com/en-us/azure/role-based-access-control/permissions/compute
func (b batchAccountWrapper) IAMPermissions() []string {
	return []string{
		"Microsoft.Batch/batchAccounts/read",
	}
}

// ref: https://learn.microsoft.com/en-us/azure/role-based-access-control/built-in-roles/compute#azure-batch-account-reader
func (b batchAccountWrapper) PredefinedRole() string {
	return "Azure Batch Account Reader"
}
