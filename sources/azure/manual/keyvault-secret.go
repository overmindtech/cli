package manual

import (
	"context"
	"errors"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/keyvault/armkeyvault/v2"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

var KeyVaultSecretLookupByName = shared.NewItemTypeLookup("name", azureshared.KeyVaultSecret)

type keyvaultSecretWrapper struct {
	client clients.SecretsClient

	*azureshared.MultiResourceGroupBase
}

func NewKeyVaultSecret(client clients.SecretsClient, resourceGroupScopes []azureshared.ResourceGroupScope) sources.SearchableWrapper {
	return &keyvaultSecretWrapper{
		client: client,
		MultiResourceGroupBase: azureshared.NewMultiResourceGroupBase(
			resourceGroupScopes,
			sdp.AdapterCategory_ADAPTER_CATEGORY_SECURITY,
			azureshared.KeyVaultSecret,
		),
	}
}

func (k keyvaultSecretWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 2 {
		return nil, azureshared.QueryError(errors.New("Get requires 2 query parts: vaultName and secretName"), scope, k.Type())
	}

	vaultName := queryParts[0]
	if vaultName == "" {
		return nil, azureshared.QueryError(errors.New("vaultName cannot be empty"), scope, k.Type())
	}

	secretName := queryParts[1]
	if secretName == "" {
		return nil, azureshared.QueryError(errors.New("secretName cannot be empty"), scope, k.Type())
	}

	rgScope, err := k.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, k.Type())
	}
	resp, err := k.client.Get(ctx, rgScope.ResourceGroup, vaultName, secretName, nil)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, k.Type())
	}

	return k.azureSecretToSDPItem(&resp.Secret, vaultName, secretName, scope)
}

// ref: https://learn.microsoft.com/en-us/rest/api/keyvault/secrets/get-secrets/get-secrets?view=rest-keyvault-secrets-2025-07-01&tabs=HTTP
func (k keyvaultSecretWrapper) Search(ctx context.Context, scope string, queryParts ...string) ([]*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 1 {
		return nil, azureshared.QueryError(errors.New("Search requires 1 query part: vaultName"), scope, k.Type())
	}

	vaultName := queryParts[0]
	if vaultName == "" {
		return nil, azureshared.QueryError(errors.New("vaultName cannot be empty"), scope, k.Type())
	}

	rgScope, err := k.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, k.Type())
	}
	pager := k.client.NewListPager(rgScope.ResourceGroup, vaultName, nil)

	var items []*sdp.Item
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, azureshared.QueryError(err, scope, k.Type())
		}
		for _, secret := range page.Value {
			if secret.Name == nil {
				continue
			}
			// Extract vault name from secret ID for composite key
			// Format: /subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.KeyVault/vaults/{vaultName}/secrets/{secretName}
			var secretVaultName string
			if secret.ID != nil && *secret.ID != "" {
				vaultParams := azureshared.ExtractPathParamsFromResourceID(*secret.ID, []string{"vaults"})
				if len(vaultParams) > 0 {
					secretVaultName = vaultParams[0]
				}
			}
			// Fallback to queryParts vaultName if extraction fails
			if secretVaultName == "" {
				secretVaultName = vaultName
			}
			item, sdpErr := k.azureSecretToSDPItem(secret, secretVaultName, *secret.Name, scope)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}

	return items, nil
}

func (k keyvaultSecretWrapper) azureSecretToSDPItem(secret *armkeyvault.Secret, vaultName, secretName, scope string) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(secret, "tags")
	if err != nil {
		return nil, azureshared.QueryError(err, scope, k.Type())
	}

	if secret.Name == nil {
		return nil, azureshared.QueryError(errors.New("secret name is nil"), scope, k.Type())
	}

	// Set composite unique attribute to prevent collisions when secrets with the same name exist in different vaults
	err = attributes.Set("uniqueAttr", shared.CompositeLookupKey(vaultName, secretName))
	if err != nil {
		return nil, azureshared.QueryError(err, scope, k.Type())
	}

	sdpItem := &sdp.Item{
		Type:            azureshared.KeyVaultSecret.String(),
		UniqueAttribute: "uniqueAttr",
		Attributes:      attributes,
		Scope:           scope,
		Tags:            azureshared.ConvertAzureTags(secret.Tags),
	}

	// Link to parent Key Vault from ID
	// Reference: https://learn.microsoft.com/en-us/rest/api/keyvault/keyvault/vaults/get
	// GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.KeyVault/vaults/{vaultName}
	//
	// IMPORTANT: The Key Vault can be in a different resource group than the secret's resource group.
	// We must extract the subscription ID and resource group from the secret's resource ID
	// to construct the correct scope.
	if secret.ID != nil && *secret.ID != "" {
		// Extract vault name from resource ID
		// Format: /subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.KeyVault/vaults/{vaultName}/secrets/{secretName}
		vaultParams := azureshared.ExtractPathParamsFromResourceID(*secret.ID, []string{"vaults"})
		if len(vaultParams) > 0 {
			vaultName := vaultParams[0]
			if vaultName != "" {
				// Extract scope from resource ID (subscription and resource group)
				linkedScope := azureshared.ExtractScopeFromResourceID(*secret.ID)
				if linkedScope == "" {
					// Fallback to default scope if extraction fails
					linkedScope = scope
				}
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   azureshared.KeyVaultVault.String(),
						Method: sdp.QueryMethod_GET,
						Query:  vaultName,
						Scope:  linkedScope, // Use the vault's scope from the resource ID
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true,  // If Key Vault is deleted/modified → secret access and configuration are affected (In: true)
						Out: false, // If secret is deleted → Key Vault remains (Out: false)
					}, // Secret depends on Key Vault - vault changes impact secret availability and access
				})
			}
		}
	}

	// Link to DNS name and HTTP endpoints (standard library) from SecretURI and SecretURIWithVersion.
	// Both URIs share the same Key Vault hostname (e.g., myvault.vault.azure.net), so we add the DNS link only once.
	var linkedDNSName string
	if secret.Properties != nil && secret.Properties.SecretURI != nil && *secret.Properties.SecretURI != "" {
		secretURI := *secret.Properties.SecretURI
		dnsName := azureshared.ExtractDNSFromURL(secretURI)
		if dnsName != "" {
			linkedDNSName = dnsName
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   stdlib.NetworkDNS.String(),
					Method: sdp.QueryMethod_SEARCH,
					Query:  dnsName,
					Scope:  "global",
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: true,
				},
			})
		}
		sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   stdlib.NetworkHTTP.String(),
				Method: sdp.QueryMethod_SEARCH,
				Query:  secretURI,
				Scope:  "global",
			},
			BlastPropagation: &sdp.BlastPropagation{
				In:  true,
				Out: true,
			},
		})
	}

	// SecretURIWithVersion is the versioned URL; add HTTP link. Skip DNS link if same hostname already linked.
	if secret.Properties != nil && secret.Properties.SecretURIWithVersion != nil && *secret.Properties.SecretURIWithVersion != "" {
		secretURIWithVersion := *secret.Properties.SecretURIWithVersion
		dnsName := azureshared.ExtractDNSFromURL(secretURIWithVersion)
		if dnsName != "" && dnsName != linkedDNSName {
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   stdlib.NetworkDNS.String(),
					Method: sdp.QueryMethod_SEARCH,
					Query:  dnsName,
					Scope:  "global",
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: true,
				},
			})
		}
		sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   stdlib.NetworkHTTP.String(),
				Method: sdp.QueryMethod_SEARCH,
				Query:  secretURIWithVersion,
				Scope:  "global",
			},
			BlastPropagation: &sdp.BlastPropagation{
				In:  true,
				Out: true,
			},
		})
	}

	return sdpItem, nil
}

func (k keyvaultSecretWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		KeyVaultVaultLookupByName,  // First key: vault name (queryParts[0])
		KeyVaultSecretLookupByName, // Second key: secret name (queryParts[1])
	}
}

func (k keyvaultSecretWrapper) SearchLookups() []sources.ItemTypeLookups {
	return []sources.ItemTypeLookups{
		{
			KeyVaultVaultLookupByName,
		},
	}
}

// ref: https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/ephemeral-resources/key_vault_secret
func (k keyvaultSecretWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_SEARCH,
			TerraformQueryMap: "azurerm_key_vault_secret.id",
		},
	}
}

func (k keyvaultSecretWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		azureshared.KeyVaultVault,
		stdlib.NetworkDNS,
		stdlib.NetworkHTTP,
	)
}

func (k keyvaultSecretWrapper) IAMPermissions() []string {
	return []string{
		"Microsoft.KeyVault/vaults/secrets/read",
	}
}

func (k keyvaultSecretWrapper) PredefinedRole() string {
	return "Reader"
}
