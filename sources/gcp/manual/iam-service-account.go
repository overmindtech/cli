package manual

import (
	"context"
	"errors"
	"strings"

	"cloud.google.com/go/iam/admin/apiv1/adminpb"
	"google.golang.org/api/iterator"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

var IAMServiceAccountLookupByEmailOrUniqueID = shared.NewItemTypeLookup("email or unique_id", gcpshared.IAMServiceAccount)

type iamServiceAccountWrapper struct {
	client gcpshared.IAMServiceAccountClient

	*gcpshared.ProjectBase
}

// NewIAMServiceAccount creates a new iamServiceAccountWrapper
func NewIAMServiceAccount(client gcpshared.IAMServiceAccountClient, projectID string) sources.ListableWrapper {
	return &iamServiceAccountWrapper{
		client: client,
		ProjectBase: gcpshared.NewProjectBase(
			projectID,
			sdp.AdapterCategory_ADAPTER_CATEGORY_SECURITY,
			gcpshared.IAMServiceAccount,
		),
	}
}

func (c iamServiceAccountWrapper) IAMPermissions() []string {
	return []string{
		"iam.serviceAccounts.get",
		"iam.serviceAccounts.list",
	}
}

// PotentialLinks returns the potential links for the IAM ServiceAccount wrapper
func (c iamServiceAccountWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		gcpshared.CloudResourceManagerProject,
		gcpshared.IAMServiceAccountKey,
	)
}

// TerraformMappings returns the Terraform mappings for the IAM ServiceAccount wrapper
func (c iamServiceAccountWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		// https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/compute_snapshot#argument-reference
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "google_service_account.email",
		},
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "google_service_account.unique_id",
		},
	}
}

// GetLookups returns the lookups for the IAM ServiceAccount wrapper
func (c iamServiceAccountWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		IAMServiceAccountLookupByEmailOrUniqueID,
	}
}

// Get retrieves a ServiceAccount by its email or unique_id
func (c iamServiceAccountWrapper) Get(ctx context.Context, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	// Can be either an email or a unique_id.
	resourceIdentifier := queryParts[0]
	name := "projects/" + c.ProjectID() + "/serviceAccounts/" + resourceIdentifier

	req := &adminpb.GetServiceAccountRequest{
		Name: name,
	}

	serviceAccount, err := c.client.Get(ctx, req)
	if err != nil {
		return nil, gcpshared.QueryError(err)
	}

	item, sdpErr := c.gcpIAMServiceAccountToSDPItem(serviceAccount)
	if sdpErr != nil {
		return nil, sdpErr
	}

	// If the resourceIdentifier is an email, set the unique attribute to "email"
	// This is to ensure tha the get method query parameter matches the unique attribute
	if strings.Contains(resourceIdentifier, "@") {
		// SDP item has an attribute of "email".
		item.UniqueAttribute = "email"
	}

	return item, nil
}

// List lists IAM ServiceAccounts and converts them to sdp.Items.
func (c iamServiceAccountWrapper) List(ctx context.Context) ([]*sdp.Item, *sdp.QueryError) {
	req := &adminpb.ListServiceAccountsRequest{
		Name: "projects/" + c.ProjectID(),
	}

	results := c.client.List(ctx, req)

	var items []*sdp.Item
	for {
		sa, err := results.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, gcpshared.QueryError(err)
		}

		item, sdpErr := c.gcpIAMServiceAccountToSDPItem(sa)
		if sdpErr != nil {
			return nil, sdpErr
		}

		items = append(items, item)
	}

	return items, nil
}

// ListStream lists IAM ServiceAccounts and sends them as sdp.Items to the stream.
func (c iamServiceAccountWrapper) ListStream(ctx context.Context, stream discovery.QueryResultStream) {
	req := &adminpb.ListServiceAccountsRequest{
		Name: "projects/" + c.ProjectID(),
	}

	results := c.client.List(ctx, req)

	for {
		sa, err := results.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			stream.SendError(gcpshared.QueryError(err))
			return
		}

		item, sdpErr := c.gcpIAMServiceAccountToSDPItem(sa)
		if sdpErr != nil {
			stream.SendError(sdpErr)
			continue
		}

		stream.SendItem(item)
	}
}

// gcpIAMServiceAccountToSDPItem converts a GCP ServiceAccount to an SDP Item, linking GCP resource fields.
// See: https://cloud.google.com/iam/docs/reference/rest/v1/projects.serviceAccounts/get
func (c iamServiceAccountWrapper) gcpIAMServiceAccountToSDPItem(serviceAccount *adminpb.ServiceAccount) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(serviceAccount)
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: err.Error(),
		}
	}

	sdpItem := &sdp.Item{
		Type:            gcpshared.IAMServiceAccount.String(),
		UniqueAttribute: "unique_id",
		Attributes:      attributes,
		Scope:           c.DefaultScope(),
	}

	// Link to the project that owns this service account
	// GET https://cloudresourcemanager.googleapis.com/v1/projects/{projectId}
	// https://cloud.google.com/resource-manager/reference/rest/v1/projects/get
	if projectID := serviceAccount.GetProjectId(); projectID != "" {
		sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   gcpshared.CloudResourceManagerProject.String(),
				Method: sdp.QueryMethod_GET,
				Query:  projectID,
				Scope:  c.ProjectID(),
			},
			BlastPropagation: &sdp.BlastPropagation{
				// Deleting a project deletes the associated ServiceAccounts
				// Account permissions affect projects
				In:  true,
				Out: true,
			},
		})
	}

	// The URL for the ServiceAccount related to this ServiceAccountKey
	// GET https://iam.googleapis.com/v1/{name=projects/*/serviceAccounts/*}/keys
	// https://cloud.google.com/iam/docs/reference/rest/v1/projects.serviceAccounts.keys/list
	if serviceAccountName := serviceAccount.GetName(); serviceAccountName != "" {
		if strings.Contains(serviceAccountName, "/") {
			serviceAccountID := gcpshared.ExtractPathParam("serviceAccounts", serviceAccountName)
			if serviceAccountID != "" {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   gcpshared.IAMServiceAccountKey.String(),
						Method: sdp.QueryMethod_SEARCH,
						Query:  serviceAccountID,
						Scope:  c.ProjectID(),
					},
					BlastPropagation: &sdp.BlastPropagation{
						// If key is deleted resources using the key are affected but account itself is not
						// If service account is deleted, all keys that belong to it are deleted
						In:  false,
						Out: true,
					},
				})
			}
		}
	}

	//It's also possible to get Oauth2ClientId from the serviceAccount, but no request is available in GCP to get data about the Oauth2Client.

	return sdpItem, nil
}
