package shared

import (
	"context"

	admin "cloud.google.com/go/iam/admin/apiv1"
	"cloud.google.com/go/iam/admin/apiv1/adminpb"
	"github.com/googleapis/gax-go/v2"
)

// IAMServiceAccountClient interface for IAM ServiceAccount operations
type IAMServiceAccountClient interface {
	Get(ctx context.Context, req *adminpb.GetServiceAccountRequest, opts ...gax.CallOption) (*adminpb.ServiceAccount, error)
	List(ctx context.Context, req *adminpb.ListServiceAccountsRequest, opts ...gax.CallOption) IAMServiceAccountIterator
}

type IAMServiceAccountIterator interface {
	Next() (*adminpb.ServiceAccount, error)
}

type iamServiceAccountClient struct {
	client *admin.IamClient
}

func (c *iamServiceAccountClient) Get(ctx context.Context, req *adminpb.GetServiceAccountRequest, opts ...gax.CallOption) (*adminpb.ServiceAccount, error) {
	return c.client.GetServiceAccount(ctx, req, opts...)
}

func (c *iamServiceAccountClient) List(ctx context.Context, req *adminpb.ListServiceAccountsRequest, opts ...gax.CallOption) IAMServiceAccountIterator {
	return c.client.ListServiceAccounts(ctx, req, opts...)
}

// NewIAMServiceAccountClient creates a new IAMServiceAccountClient
func NewIAMServiceAccountClient(client *admin.IamClient) IAMServiceAccountClient {
	return &iamServiceAccountClient{
		client: client,
	}
}

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
