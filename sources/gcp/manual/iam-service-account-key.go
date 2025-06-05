package manual

import (
	"context"
	"strings"

	"cloud.google.com/go/iam/admin/apiv1/adminpb"

	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

var (
	IAMServiceAccountKey = shared.NewItemType(gcpshared.GCP, gcpshared.IAM, gcpshared.ServiceAccountKey)

	IAMServiceAccountKeyLookupByName = shared.NewItemTypeLookup("name", IAMServiceAccountKey)
)

type iamServiceAccountKeyWrapper struct {
	client gcpshared.IAMServiceAccountKeyClient
	*gcpshared.ProjectBase
}

// NewIAMServiceAccountKey creates a new IAM Service Account Key adapter
func NewIAMServiceAccountKey(client gcpshared.IAMServiceAccountKeyClient, projectID string) sources.SearchableWrapper {
	return &iamServiceAccountKeyWrapper{
		client: client,
		ProjectBase: gcpshared.NewProjectBase(
			projectID,
			sdp.AdapterCategory_ADAPTER_CATEGORY_SECURITY,
			IAMServiceAccountKey,
		),
	}
}

// PotentialLinks returns the potential links for the iam service account wrapper
func (c iamServiceAccountKeyWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		IAMServiceAccount,
	)
}

// TerraformMappings returns the Terraform mappings for the IAM Service Account Key wrapper
func (c iamServiceAccountKeyWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "google_service_account_key.name",
		},
	}
}

// GetLookups returns the lookups for the IAM Service Account Key wrapper
func (c iamServiceAccountKeyWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		IAMServiceAccountLookupByEmailOrUniqueID,
		IAMServiceAccountKeyLookupByName,
	}
}

// Get retrieves a Service Account Key by its name and related serviceAccount
// See: https://cloud.google.com/iam/docs/reference/rest/v1/projects.serviceAccounts.keys/get
// Format: GET https://iam.googleapis.com/v1/{name=projects/*/serviceAccounts/*/keys/*}
func (c iamServiceAccountKeyWrapper) Get(ctx context.Context, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	serviceAccountIdentifier := queryParts[0]
	keyName := queryParts[1]

	req := &adminpb.GetServiceAccountKeyRequest{
		Name: "projects/" + c.ProjectID() + "/serviceAccounts/" + serviceAccountIdentifier + "/keys/" + keyName,
	}

	key, err := c.client.Get(ctx, req)
	if err != nil {
		return nil, gcpshared.QueryError(err)
	}

	item, sdpErr := c.gcpIAMServiceAccountKeyToSDPItem(key)
	if sdpErr != nil {
		return nil, sdpErr
	}

	return item, nil
}

// SearchLookups defines how the source can be searched for specific items.
func (c iamServiceAccountKeyWrapper) SearchLookups() []sources.ItemTypeLookups {
	return []sources.ItemTypeLookups{
		{
			IAMServiceAccountLookupByEmailOrUniqueID,
		},
	}
}

// Search retrieves Service Account Keys by name (or other supported fields in the future)
func (c iamServiceAccountKeyWrapper) Search(ctx context.Context, queryParts ...string) ([]*sdp.Item, *sdp.QueryError) {
	serviceAccountIdentifier := queryParts[0]

	it, err := c.client.Search(ctx, &adminpb.ListServiceAccountKeysRequest{
		Name: "projects/" + c.ProjectID() + "/serviceAccounts/" + serviceAccountIdentifier,
	})
	if err != nil {
		return nil, gcpshared.QueryError(err)
	}

	var items []*sdp.Item
	for _, key := range it.GetKeys() {
		item, sdpErr := c.gcpIAMServiceAccountKeyToSDPItem(key)
		if sdpErr != nil {
			return nil, sdpErr
		}
		items = append(items, item)
	}

	return items, nil
}

// gcpIAMServiceAccountKeyToSDPItem converts a ServiceAccountKey to an sdp.Item
func (c iamServiceAccountKeyWrapper) gcpIAMServiceAccountKeyToSDPItem(key *adminpb.ServiceAccountKey) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(key)
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: err.Error(),
		}
	}

	sdpItem := &sdp.Item{
		Type:            IAMServiceAccountKey.String(),
		UniqueAttribute: "name",
		Attributes:      attributes,
		Scope:           c.ProjectID(),
	}

	// The URL for the ServiceAccount related to this ServiceAccountKey
	// GET https://iam.googleapis.com/v1/{name=projects/*/serviceAccounts/*}
	// https://cloud.google.com/iam/docs/reference/rest/v1/projects.serviceAccounts
	if serviceAccountKeyName := key.GetName(); serviceAccountKeyName != "" {
		if strings.Contains(serviceAccountKeyName, "/") {
			serviceAccountName := gcpshared.ExtractPathParam("serviceAccounts", serviceAccountKeyName)
			if serviceAccountName != "" {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   IAMServiceAccount.String(),
						Method: sdp.QueryMethod_GET,
						Query:  serviceAccountName,
						Scope:  c.ProjectID(),
					},
					BlastPropagation: &sdp.BlastPropagation{
						// If service account is deleted, all keys that belong to it are deleted
						// If key is deleted, resources using that particular key lose access to service-account.
						// But account itself keeps working.
						In:  true,
						Out: false,
					},
				})
			}

		}
	}

	return sdpItem, nil
}
