package manual

import (
	"context"
	"errors"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/msi/armmsi"
	"github.com/overmindtech/cli/go/discovery"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

var ManagedIdentityFederatedIdentityCredentialLookupByName = shared.NewItemTypeLookup("name", azureshared.ManagedIdentityFederatedIdentityCredential)

type managedIdentityFederatedIdentityCredentialWrapper struct {
	client clients.FederatedIdentityCredentialsClient

	*azureshared.MultiResourceGroupBase
}

func NewManagedIdentityFederatedIdentityCredential(client clients.FederatedIdentityCredentialsClient, resourceGroupScopes []azureshared.ResourceGroupScope) sources.SearchableWrapper {
	return &managedIdentityFederatedIdentityCredentialWrapper{
		client: client,
		MultiResourceGroupBase: azureshared.NewMultiResourceGroupBase(
			resourceGroupScopes,
			sdp.AdapterCategory_ADAPTER_CATEGORY_SECURITY,
			azureshared.ManagedIdentityFederatedIdentityCredential,
		),
	}
}

func (m managedIdentityFederatedIdentityCredentialWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 2 {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "Get requires 2 query parts: identityName and federatedCredentialName",
			Scope:       scope,
			ItemType:    m.Type(),
		}
	}
	identityName := queryParts[0]
	if identityName == "" {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "identityName cannot be empty",
			Scope:       scope,
			ItemType:    m.Type(),
		}
	}
	federatedCredentialName := queryParts[1]
	if federatedCredentialName == "" {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "federatedCredentialName cannot be empty",
			Scope:       scope,
			ItemType:    m.Type(),
		}
	}

	rgScope, err := m.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, m.Type())
	}

	resp, err := m.client.Get(ctx, rgScope.ResourceGroup, identityName, federatedCredentialName, nil)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, m.Type())
	}

	return m.azureFederatedIdentityCredentialToSDPItem(&resp.FederatedIdentityCredential, identityName, federatedCredentialName, scope)
}

func (m managedIdentityFederatedIdentityCredentialWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		ManagedIdentityUserAssignedIdentityLookupByName,
		ManagedIdentityFederatedIdentityCredentialLookupByName,
	}
}

func (m managedIdentityFederatedIdentityCredentialWrapper) Search(ctx context.Context, scope string, queryParts ...string) ([]*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 1 {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "Search requires 1 query part: identityName",
			Scope:       scope,
			ItemType:    m.Type(),
		}
	}
	identityName := queryParts[0]
	if identityName == "" {
		return nil, azureshared.QueryError(errors.New("identityName cannot be empty"), scope, m.Type())
	}

	rgScope, err := m.ResourceGroupScopeFromScope(scope)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, m.Type())
	}

	pager := m.client.NewListPager(rgScope.ResourceGroup, identityName, nil)

	var items []*sdp.Item
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, azureshared.QueryError(err, scope, m.Type())
		}

		for _, credential := range page.Value {
			if credential.Name == nil {
				continue
			}

			item, sdpErr := m.azureFederatedIdentityCredentialToSDPItem(credential, identityName, *credential.Name, scope)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}

	return items, nil
}

func (m managedIdentityFederatedIdentityCredentialWrapper) SearchStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string, queryParts ...string) {
	if len(queryParts) < 1 {
		stream.SendError(azureshared.QueryError(errors.New("Search requires 1 query part: identityName"), scope, m.Type()))
		return
	}
	identityName := queryParts[0]
	if identityName == "" {
		stream.SendError(azureshared.QueryError(errors.New("identityName cannot be empty"), scope, m.Type()))
		return
	}

	rgScope, err := m.ResourceGroupScopeFromScope(scope)
	if err != nil {
		stream.SendError(azureshared.QueryError(err, scope, m.Type()))
		return
	}

	pager := m.client.NewListPager(rgScope.ResourceGroup, identityName, nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			stream.SendError(azureshared.QueryError(err, scope, m.Type()))
			return
		}
		for _, credential := range page.Value {
			if credential.Name == nil {
				continue
			}
			item, sdpErr := m.azureFederatedIdentityCredentialToSDPItem(credential, identityName, *credential.Name, scope)
			if sdpErr != nil {
				stream.SendError(sdpErr)
				continue
			}
			cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
			stream.SendItem(item)
		}
	}
}

func (m managedIdentityFederatedIdentityCredentialWrapper) SearchLookups() []sources.ItemTypeLookups {
	return []sources.ItemTypeLookups{
		{
			ManagedIdentityUserAssignedIdentityLookupByName,
		},
	}
}

func (m managedIdentityFederatedIdentityCredentialWrapper) azureFederatedIdentityCredentialToSDPItem(credential *armmsi.FederatedIdentityCredential, identityName, credentialName, scope string) (*sdp.Item, *sdp.QueryError) {
	if credential.Name == nil {
		return nil, azureshared.QueryError(errors.New("credential name is nil"), scope, m.Type())
	}

	attributes, err := shared.ToAttributesWithExclude(credential)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, m.Type())
	}

	err = attributes.Set("uniqueAttr", shared.CompositeLookupKey(identityName, credentialName))
	if err != nil {
		return nil, azureshared.QueryError(err, scope, m.Type())
	}

	sdpItem := &sdp.Item{
		Type:            azureshared.ManagedIdentityFederatedIdentityCredential.String(),
		UniqueAttribute: "uniqueAttr",
		Attributes:      attributes,
		Scope:           scope,
	}

	// Link back to the parent user assigned identity
	sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   azureshared.ManagedIdentityUserAssignedIdentity.String(),
			Method: sdp.QueryMethod_GET,
			Query:  identityName,
			Scope:  scope,
		},
	})

	// Link to DNS hostname from Issuer URL (e.g., https://token.actions.githubusercontent.com)
	// The Issuer is the URL of the external identity provider
	if credential.Properties != nil && credential.Properties.Issuer != nil && *credential.Properties.Issuer != "" {
		dnsName := azureshared.ExtractDNSFromURL(*credential.Properties.Issuer)
		if dnsName != "" {
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   stdlib.NetworkDNS.String(),
					Method: sdp.QueryMethod_SEARCH,
					Query:  dnsName,
					Scope:  "global",
				},
			})
		}
	}

	return sdpItem, nil
}

func (m managedIdentityFederatedIdentityCredentialWrapper) PotentialLinks() map[shared.ItemType]bool {
	return map[shared.ItemType]bool{
		azureshared.ManagedIdentityUserAssignedIdentity: true,
		stdlib.NetworkDNS: true,
	}
}

// ref: https://learn.microsoft.com/en-us/azure/role-based-access-control/permissions/identity#microsoftmanagedidentity
func (m managedIdentityFederatedIdentityCredentialWrapper) IAMPermissions() []string {
	return []string{
		"Microsoft.ManagedIdentity/userAssignedIdentities/federatedIdentityCredentials/read",
	}
}

func (m managedIdentityFederatedIdentityCredentialWrapper) PredefinedRole() string {
	return "Reader"
}
