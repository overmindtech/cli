package shared

import (
	"context"

	"cloud.google.com/go/networksecurity/apiv1beta1"
	"cloud.google.com/go/networksecurity/apiv1beta1/networksecuritypb"
	"github.com/googleapis/gax-go/v2"
)

type NetworkSecurityClientTlsPolicyClient interface {
	Get(ctx context.Context, req *networksecuritypb.GetClientTlsPolicyRequest, opts ...gax.CallOption) (*networksecuritypb.ClientTlsPolicy, error)
	List(ctx context.Context, req *networksecuritypb.ListClientTlsPoliciesRequest, opts ...gax.CallOption) NetworkSecurityClientTlsPolicyIterator
}

type NetworkSecurityClientTlsPolicyIterator interface {
	Next() (*networksecuritypb.ClientTlsPolicy, error)
}

type networkSecurityClientTlsPolicyClient struct {
	client *networksecurity.Client
}

func (c networkSecurityClientTlsPolicyClient) Get(ctx context.Context, req *networksecuritypb.GetClientTlsPolicyRequest, opts ...gax.CallOption) (*networksecuritypb.ClientTlsPolicy, error) {
	return c.client.GetClientTlsPolicy(ctx, req, opts...)
}

func (c networkSecurityClientTlsPolicyClient) List(ctx context.Context, req *networksecuritypb.ListClientTlsPoliciesRequest, opts ...gax.CallOption) NetworkSecurityClientTlsPolicyIterator {
	return c.client.ListClientTlsPolicies(ctx, req, opts...)
}

// NewNetworkSecurityClientTlsPolicyClient creates a new NetworkSecurityClientTlsPolicyClient
func NewNetworkSecurityClientTlsPolicyClient(client *networksecurity.Client) NetworkSecurityClientTlsPolicyClient {
	return &networkSecurityClientTlsPolicyClient{
		client: client,
	}
}
