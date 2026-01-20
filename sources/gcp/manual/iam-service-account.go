package manual

import (
	"context"
	"errors"
	"strings"

	"cloud.google.com/go/iam/admin/apiv1/adminpb"
	"google.golang.org/api/iterator"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
	"github.com/overmindtech/cli/sources"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

var IAMServiceAccountLookupByEmailOrUniqueID = shared.NewItemTypeLookup("email or unique_id", gcpshared.IAMServiceAccount)

type iamServiceAccountWrapper struct {
	client gcpshared.IAMServiceAccountClient
	*gcpshared.ProjectBase
}

// NewIAMServiceAccount creates a new iamServiceAccountWrapper.
func NewIAMServiceAccount(client gcpshared.IAMServiceAccountClient, locations []gcpshared.LocationInfo) sources.ListStreamableWrapper {
	return &iamServiceAccountWrapper{
		client: client,
		ProjectBase: gcpshared.NewProjectBase(
			locations,
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

func (c iamServiceAccountWrapper) PredefinedRole() string {
	return "roles/iam.serviceAccountViewer"
}

func (c iamServiceAccountWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		gcpshared.CloudResourceManagerProject,
		gcpshared.IAMServiceAccountKey,
	)
}

func (c iamServiceAccountWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
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

func (c iamServiceAccountWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		IAMServiceAccountLookupByEmailOrUniqueID,
	}
}

func (c iamServiceAccountWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	location, err := c.LocationFromScope(scope)
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: err.Error(),
		}
	}

	resourceIdentifier := queryParts[0]
	name := "projects/" + location.ProjectID + "/serviceAccounts/" + resourceIdentifier

	req := &adminpb.GetServiceAccountRequest{
		Name: name,
	}

	serviceAccount, getErr := c.client.Get(ctx, req)
	if getErr != nil {
		return nil, gcpshared.QueryError(getErr, scope, c.Type())
	}

	item, sdpErr := c.gcpIAMServiceAccountToSDPItem(serviceAccount, location)
	if sdpErr != nil {
		return nil, sdpErr
	}

	if strings.Contains(resourceIdentifier, "@") {
		item.UniqueAttribute = "email"
	}

	return item, nil
}

func (c iamServiceAccountWrapper) List(ctx context.Context, scope string) ([]*sdp.Item, *sdp.QueryError) {
	location, err := c.LocationFromScope(scope)
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: err.Error(),
		}
	}

	req := &adminpb.ListServiceAccountsRequest{
		Name: "projects/" + location.ProjectID,
	}

	results := c.client.List(ctx, req)

	var items []*sdp.Item
	for {
		sa, iterErr := results.Next()
		if errors.Is(iterErr, iterator.Done) {
			break
		}
		if iterErr != nil {
			return nil, gcpshared.QueryError(iterErr, scope, c.Type())
		}

		item, sdpErr := c.gcpIAMServiceAccountToSDPItem(sa, location)
		if sdpErr != nil {
			return nil, sdpErr
		}

		items = append(items, item)
	}

	return items, nil
}

func (c iamServiceAccountWrapper) ListStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string) {
	location, err := c.LocationFromScope(scope)
	if err != nil {
		stream.SendError(&sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: err.Error(),
		})
		return
	}

	req := &adminpb.ListServiceAccountsRequest{
		Name: "projects/" + location.ProjectID,
	}

	results := c.client.List(ctx, req)

	for {
		sa, iterErr := results.Next()
		if errors.Is(iterErr, iterator.Done) {
			break
		}
		if iterErr != nil {
			stream.SendError(gcpshared.QueryError(iterErr, scope, c.Type()))
			return
		}

		item, sdpErr := c.gcpIAMServiceAccountToSDPItem(sa, location)
		if sdpErr != nil {
			stream.SendError(sdpErr)
			continue
		}

		cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
		stream.SendItem(item)
	}
}

func (c iamServiceAccountWrapper) gcpIAMServiceAccountToSDPItem(serviceAccount *adminpb.ServiceAccount, location gcpshared.LocationInfo) (*sdp.Item, *sdp.QueryError) {
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
		Scope:           location.ToScope(),
	}

	if projectID := serviceAccount.GetProjectId(); projectID != "" {
		sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   gcpshared.CloudResourceManagerProject.String(),
				Method: sdp.QueryMethod_GET,
				Query:  projectID,
				Scope:  location.ProjectID,
			},
			BlastPropagation: &sdp.BlastPropagation{
				In:  true,
				Out: true,
			},
		})
	}

	if serviceAccountName := serviceAccount.GetName(); serviceAccountName != "" {
		if strings.Contains(serviceAccountName, "/") {
			serviceAccountID := gcpshared.ExtractPathParam("serviceAccounts", serviceAccountName)
			if serviceAccountID != "" {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   gcpshared.IAMServiceAccountKey.String(),
						Method: sdp.QueryMethod_SEARCH,
						Query:  serviceAccountID,
						Scope:  location.ProjectID,
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  false,
						Out: true,
					},
				})
			}
		}
	}

	return sdpItem, nil
}
