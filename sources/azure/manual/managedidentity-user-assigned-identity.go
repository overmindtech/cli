package manual

import (
	"context"
	"errors"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/msi/armmsi"
	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/shared"
)

var (
	ManagedIdentityUserAssignedIdentityLookupByName = shared.NewItemTypeLookup("name", azureshared.ManagedIdentityUserAssignedIdentity)
)

type managedIdentityUserAssignedIdentityWrapper struct {
	client clients.UserAssignedIdentitiesClient

	*azureshared.ResourceGroupBase
}

func NewManagedIdentityUserAssignedIdentity(client clients.UserAssignedIdentitiesClient, subscriptionID, resourceGroup string) sources.ListableWrapper {
	return &managedIdentityUserAssignedIdentityWrapper{
		client: client,
		ResourceGroupBase: azureshared.NewResourceGroupBase(
			subscriptionID,
			resourceGroup,
			sdp.AdapterCategory_ADAPTER_CATEGORY_SECURITY,
			azureshared.ManagedIdentityUserAssignedIdentity,
		),
	}
}

func (m managedIdentityUserAssignedIdentityWrapper) List(ctx context.Context) ([]*sdp.Item, *sdp.QueryError) {
	pager := m.client.ListByResourceGroup(m.ResourceGroup(), nil)

	var items []*sdp.Item
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, azureshared.QueryError(err, m.DefaultScope(), m.Type())
		}
		for _, identity := range page.Value {
			if identity.Name == nil {
				continue
			}
			item, sdpErr := m.azureManagedIdentityUserAssignedIdentityToSDPItem(identity)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}
	return items, nil
}

func (m managedIdentityUserAssignedIdentityWrapper) ListStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey) {
	pager := m.client.ListByResourceGroup(m.ResourceGroup(), nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			stream.SendError(azureshared.QueryError(err, m.DefaultScope(), m.Type()))
			return
		}
		for _, identity := range page.Value {
			if identity.Name == nil {
				continue
			}
			item, sdpErr := m.azureManagedIdentityUserAssignedIdentityToSDPItem(identity)
			if sdpErr != nil {
				stream.SendError(sdpErr)
				continue
			}
			cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
			stream.SendItem(item)
		}
	}
}

func (m managedIdentityUserAssignedIdentityWrapper) azureManagedIdentityUserAssignedIdentityToSDPItem(identity *armmsi.Identity) (*sdp.Item, *sdp.QueryError) {
	if identity.Name == nil {
		return nil, azureshared.QueryError(errors.New("name is nil"), m.DefaultScope(), m.Type())
	}
	attributes, err := shared.ToAttributesWithExclude(identity, "tags")
	if err != nil {
		return nil, azureshared.QueryError(err, m.DefaultScope(), m.Type())
	}

	sdpItem := &sdp.Item{
		Type:            azureshared.ManagedIdentityUserAssignedIdentity.String(),
		UniqueAttribute: "name",
		Attributes:      attributes,
		Scope:           m.DefaultScope(),
		Tags:            azureshared.ConvertAzureTags(identity.Tags),
	}

	// Link to federated identity credentials (child resource)
	// Federated identity credentials can be listed using the identity's resource group and name
	// Reference: https://learn.microsoft.com/en-us/rest/api/managedidentity/2023-01-31/federated-identity-credentials/list
	// The Azure SDK provides FederatedIdentityCredentialsClient with NewListPager(resourceGroupName, resourceName, options)
	// Since we can list all federated credentials for this identity, we use SEARCH method
	sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   azureshared.ManagedIdentityFederatedIdentityCredential.String(),
			Method: sdp.QueryMethod_SEARCH,
			Query:  *identity.Name, // Identity name is sufficient since resource group is available to the adapter
			Scope:  m.DefaultScope(),
		},
		BlastPropagation: &sdp.BlastPropagation{
			// Federated credentials are tightly coupled to the identity
			// Changes to the identity affect credentials, and credential changes affect identity usage
			In:  true,
			Out: true,
		},
	})

	return sdpItem, nil
}

func (m managedIdentityUserAssignedIdentityWrapper) Get(ctx context.Context, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) != 1 {
		return nil, azureshared.QueryError(errors.New("user assigned identity name is required"), m.DefaultScope(), m.Type())
	}
	name := queryParts[0]
	if name == "" {
		return nil, azureshared.QueryError(errors.New("user assigned identity name cannot be empty"), m.DefaultScope(), m.Type())
	}
	identity, err := m.client.Get(ctx, m.ResourceGroup(), name, nil)
	if err != nil {
		return nil, azureshared.QueryError(err, m.DefaultScope(), m.Type())
	}
	return m.azureManagedIdentityUserAssignedIdentityToSDPItem(&identity.Identity)
}

func (m managedIdentityUserAssignedIdentityWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		ManagedIdentityUserAssignedIdentityLookupByName,
	}
}

// ref: https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/user_assigned_identity
func (m managedIdentityUserAssignedIdentityWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "azurerm_user_assigned_identity.name",
		},
	}
}

func (m managedIdentityUserAssignedIdentityWrapper) PotentialLinks() map[shared.ItemType]bool {
	return map[shared.ItemType]bool{
		azureshared.ManagedIdentityFederatedIdentityCredential: true,
	}
}

// ref: https://learn.microsoft.com/en-us/azure/role-based-access-control/permissions/identity#microsoftmanagedidentity
func (m managedIdentityUserAssignedIdentityWrapper) IAMPermissions() []string {
	return []string{
		"Microsoft.ManagedIdentity/userAssignedIdentities/read",
	}
}

func (m managedIdentityUserAssignedIdentityWrapper) PredefinedRole() string {
	return "Reader"
}
