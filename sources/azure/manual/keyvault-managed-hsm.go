package manual

import (
	"context"
	"errors"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/keyvault/armkeyvault/v2"
	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

var KeyVaultManagedHSMsLookupByName = shared.NewItemTypeLookup("name", azureshared.KeyVaultManagedHSM)

type keyvaultManagedHSMsWrapper struct {
	client clients.ManagedHSMsClient

	*azureshared.ResourceGroupBase
}

func NewKeyVaultManagedHSM(client clients.ManagedHSMsClient, subscriptionID, resourceGroup string) sources.ListableWrapper {
	return &keyvaultManagedHSMsWrapper{
		client: client,
		ResourceGroupBase: azureshared.NewResourceGroupBase(
			subscriptionID,
			resourceGroup,
			sdp.AdapterCategory_ADAPTER_CATEGORY_SECURITY,
			azureshared.KeyVaultManagedHSM,
		),
	}
}

// ref: https://learn.microsoft.com/en-us/rest/api/keyvault/managedhsm/managed-hsms/list-by-resource-group?view=rest-keyvault-managedhsm-2024-11-01&tabs=HTTP
func (k keyvaultManagedHSMsWrapper) List(ctx context.Context, scope string) ([]*sdp.Item, *sdp.QueryError) {
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

		for _, hsm := range page.Value {
			if hsm.Name == nil {
				continue
			}

			item, sdpErr := k.azureManagedHSMToSDPItem(hsm, scope)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}

	return items, nil
}

func (k keyvaultManagedHSMsWrapper) ListStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string) {
	resourceGroup := azureshared.ResourceGroupFromScope(scope)
	if resourceGroup == "" {
		resourceGroup = k.ResourceGroup()
	}
	pager := k.client.NewListByResourceGroupPager(resourceGroup, nil)

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			stream.SendError(azureshared.QueryError(err, scope, k.Type()))
			return
		}

		for _, hsm := range page.Value {
			if hsm.Name == nil {
				continue
			}
			item, sdpErr := k.azureManagedHSMToSDPItem(hsm, scope)
			if sdpErr != nil {
				stream.SendError(sdpErr)
				continue
			}
			cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
			stream.SendItem(item)
		}
	}
}

func (k keyvaultManagedHSMsWrapper) azureManagedHSMToSDPItem(hsm *armkeyvault.ManagedHsm, scope string) (*sdp.Item, *sdp.QueryError) {
	if hsm.Name == nil {
		return nil, azureshared.QueryError(errors.New("name is nil"), scope, k.Type())
	}
	attributes, err := shared.ToAttributesWithExclude(hsm, "tags")
	if err != nil {
		return nil, azureshared.QueryError(err, scope, k.Type())
	}

	sdpItem := &sdp.Item{
		Type:              azureshared.KeyVaultManagedHSM.String(),
		UniqueAttribute:   "name",
		Attributes:        attributes,
		Scope:             scope,
		Tags:              azureshared.ConvertAzureTags(hsm.Tags),
		LinkedItemQueries: []*sdp.LinkedItemQuery{},
	}

	// Link to MHSM Private Endpoint Connections (child resources with their own GET endpoint)
	// Reference: https://learn.microsoft.com/en-us/rest/api/keyvault/managedhsm/mhsm-private-endpoint-connections/get
	// GET .../managedHSMs/{name}/privateEndpointConnections/{privateEndpointConnectionName}
	if hsm.Properties != nil && hsm.Properties.PrivateEndpointConnections != nil && hsm.Name != nil {
		for _, conn := range hsm.Properties.PrivateEndpointConnections {
			if conn != nil && conn.ID != nil && *conn.ID != "" {
				connectionName := azureshared.ExtractResourceName(*conn.ID)
				if connectionName != "" {
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.KeyVaultManagedHSMPrivateEndpointConnection.String(),
							Method: sdp.QueryMethod_GET,
							Query:  shared.CompositeLookupKey(*hsm.Name, connectionName),
							Scope:  scope,
						},
						BlastPropagation: &sdp.BlastPropagation{
							In:  true, // Connection state changes affect the Managed HSM's private connectivity
							Out: true, // Managed HSM deletion removes the connection
						},
					})
				}
			}
		}
	}

	// Link to Private Endpoints from Private Endpoint Connections
	// Reference: https://learn.microsoft.com/en-us/rest/api/virtualnetwork/private-endpoints/get
	// GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/privateEndpoints/{privateEndpointName}
	//
	// IMPORTANT: Private Endpoints can be in a different resource group than the Managed HSM.
	// We must extract the subscription ID and resource group from the private endpoint's resource ID
	// to construct the correct scope.
	if hsm.Properties != nil && hsm.Properties.PrivateEndpointConnections != nil {
		for _, conn := range hsm.Properties.PrivateEndpointConnections {
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
						peScope := fmt.Sprintf("%s.%s", subscriptionID, resourceGroupName)
						sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   azureshared.NetworkPrivateEndpoint.String(),
								Method: sdp.QueryMethod_GET,
								Query:  privateEndpointName,
								Scope:  peScope, // Use the private endpoint's scope, not the Managed HSM's scope
							},
							BlastPropagation: &sdp.BlastPropagation{
								In:  true, // Private endpoint changes (deletion, network configuration) affect the Managed HSM's private connectivity
								Out: true, // Managed HSM deletion or configuration changes may affect the private endpoint's connection state
							},
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
	// IMPORTANT: Virtual Network Subnets can be in a different resource group than the Managed HSM.
	// We must extract the subscription ID and resource group from the subnet's resource ID to construct
	// the correct scope.
	if hsm.Properties != nil && hsm.Properties.NetworkACLs != nil && hsm.Properties.NetworkACLs.VirtualNetworkRules != nil {
		for _, vnetRule := range hsm.Properties.NetworkACLs.VirtualNetworkRules {
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
							Scope:  scope, // Use the subnet's scope, not the Managed HSM's scope
						},
						BlastPropagation: &sdp.BlastPropagation{
							In:  true,  // Subnet changes (deletion, network configuration) affect the Managed HSM's network accessibility
							Out: false, // Managed HSM changes don't directly affect the subnet configuration
						}, // Managed HSM depends on subnet for network access - subnet changes impact connectivity
					})
				}
			}
		}
	}

	// Link to IP addresses (standard library) from NetworkACLs IPRules
	// Reference: https://learn.microsoft.com/en-us/rest/api/keyvault/managedhsm/managed-hsms/get?view=rest-keyvault-managedhsm-2024-11-01&tabs=HTTP
	if hsm.Properties != nil && hsm.Properties.NetworkACLs != nil && hsm.Properties.NetworkACLs.IPRules != nil {
		for _, ipRule := range hsm.Properties.NetworkACLs.IPRules {
			if ipRule != nil && ipRule.Value != nil && *ipRule.Value != "" {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   stdlib.NetworkIP.String(),
						Method: sdp.QueryMethod_GET,
						Query:  *ipRule.Value,
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

	// Link to User Assigned Managed Identities (external resources)
	// Reference: https://learn.microsoft.com/en-us/rest/api/managedidentity/user-assigned-identities/get?view=rest-managedidentity-2024-11-30&tabs=HTTP
	// GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ManagedIdentity/userAssignedIdentities/{identityName}
	//
	// IMPORTANT: User Assigned Identities can be in a different resource group than the Managed HSM.
	// We must extract the subscription ID and resource group from each identity's resource ID to construct the correct scope.
	if hsm.Identity != nil && hsm.Identity.UserAssignedIdentities != nil {
		for identityResourceID := range hsm.Identity.UserAssignedIdentities {
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
						// Managed HSM depends on managed identity for authentication and access control
						// If identity is deleted/modified, Managed HSM operations may fail
						In:  true,
						Out: false,
					},
				})
			}
		}
	}

	// Link to DNS name (standard library) from HsmURI
	// The HsmURI contains the Managed HSM endpoint URL (e.g., https://myhsm.managedhsm.azure.net)
	if hsm.Properties != nil && hsm.Properties.HsmURI != nil && *hsm.Properties.HsmURI != "" {
		hsmURI := *hsm.Properties.HsmURI
		// Extract DNS name from URL (e.g., https://myhsm.managedhsm.azure.net -> myhsm.managedhsm.azure.net)
		dnsName := azureshared.ExtractDNSFromURL(hsmURI)
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
		// Link to HTTP/HTTPS endpoint (standard library) from HsmURI
		sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   stdlib.NetworkHTTP.String(),
				Method: sdp.QueryMethod_SEARCH,
				Query:  hsmURI,
				Scope:  "global",
			},
			BlastPropagation: &sdp.BlastPropagation{
				// Endpoint connectivity affects HSM access and vice versa
				In:  true,
				Out: true,
			},
		})
	}

	return sdpItem, nil
}

// ref: https://learn.microsoft.com/en-us/rest/api/keyvault/managedhsm/managed-hsms/get?view=rest-keyvault-managedhsm-2024-11-01&tabs=HTTP
func (k keyvaultManagedHSMsWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 1 {
		return nil, azureshared.QueryError(errors.New("Get requires 1 query part: name"), scope, k.Type())
	}

	name := queryParts[0]
	if name == "" {
		return nil, azureshared.QueryError(errors.New("name cannot be empty"), scope, k.Type())
	}

	resourceGroup := azureshared.ResourceGroupFromScope(scope)
	if resourceGroup == "" {
		resourceGroup = k.ResourceGroup()
	}
	resp, err := k.client.Get(ctx, resourceGroup, name, nil)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, k.Type())
	}

	return k.azureManagedHSMToSDPItem(&resp.ManagedHsm, scope)
}

func (k keyvaultManagedHSMsWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		KeyVaultManagedHSMsLookupByName,
	}
}

// ref: https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/key_vault_managed_hardware_security_module
func (k keyvaultManagedHSMsWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "azurerm_key_vault_managed_hardware_security_module.name",
		},
	}
}

func (k keyvaultManagedHSMsWrapper) PotentialLinks() map[shared.ItemType]bool {
	return map[shared.ItemType]bool{
		azureshared.KeyVaultManagedHSMPrivateEndpointConnection: true,
		azureshared.NetworkPrivateEndpoint:                      true,
		azureshared.NetworkSubnet:                               true,
		azureshared.ManagedIdentityUserAssignedIdentity:         true,
		stdlib.NetworkDNS:                                       true,
		stdlib.NetworkHTTP:                                      true,
		stdlib.NetworkIP:                                        true,
	}
}

func (k keyvaultManagedHSMsWrapper) IAMPermissions() []string {
	return []string{
		"Microsoft.KeyVault/managedHSMs/read",
	}
}

func (k keyvaultManagedHSMsWrapper) PredefinedRole() string {
	return "Reader"
}
