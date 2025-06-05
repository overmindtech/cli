package shared

import (
	"context"

	admin "cloud.google.com/go/iam/admin/apiv1"
	"cloud.google.com/go/iam/admin/apiv1/adminpb"
	"github.com/googleapis/gax-go/v2"
)

// IAMServiceAccountKeyClient defines the interface for ServiceAccountKey operations
type IAMServiceAccountKeyClient interface {
	Get(ctx context.Context, req *adminpb.GetServiceAccountKeyRequest, opts ...gax.CallOption) (*adminpb.ServiceAccountKey, error)
	Search(ctx context.Context, req *adminpb.ListServiceAccountKeysRequest, opts ...gax.CallOption) (*adminpb.ListServiceAccountKeysResponse, error)
}

type iamServiceAccountKeyClient struct {
	client *admin.IamClient
}

func (c iamServiceAccountKeyClient) Get(ctx context.Context, req *adminpb.GetServiceAccountKeyRequest, opts ...gax.CallOption) (*adminpb.ServiceAccountKey, error) {
	return c.client.GetServiceAccountKey(ctx, req, opts...)
}

func (c iamServiceAccountKeyClient) Search(ctx context.Context, req *adminpb.ListServiceAccountKeysRequest, opts ...gax.CallOption) (*adminpb.ListServiceAccountKeysResponse, error) {
	return c.client.ListServiceAccountKeys(ctx, req, opts...)
}

// NewIAMServiceAccountKeyClient creates a new IAMServiceAccountKeyClient
func NewIAMServiceAccountKeyClient(client *admin.IamClient) IAMServiceAccountKeyClient {
	return &iamServiceAccountKeyClient{
		client: client,
	}
}
