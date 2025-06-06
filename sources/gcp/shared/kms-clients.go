package shared

import (
	"context"

	kms "cloud.google.com/go/kms/apiv1"
	"cloud.google.com/go/kms/apiv1/kmspb"
	"github.com/googleapis/gax-go/v2"
)

// CloudKMSKeyRingIterator is an interface for iterating over KMS KeyRings
type CloudKMSKeyRingIterator interface {
	Next() (*kmspb.KeyRing, error)
}

// CloudKMSKeyRingClient is an interface for the KMS KeyRing client
type CloudKMSKeyRingClient interface {
	Get(ctx context.Context, req *kmspb.GetKeyRingRequest, opts ...gax.CallOption) (*kmspb.KeyRing, error)
	Search(ctx context.Context, req *kmspb.ListKeyRingsRequest, opts ...gax.CallOption) CloudKMSKeyRingIterator
}

// cloudKMSKeyRingClient is a concrete implementation of CloudKMSKeyRingClient
type cloudKMSKeyRingClient struct {
	client *kms.KeyManagementClient
}

// NewCloudKMSKeyRingClient creates a new CloudKMSKeyRingClient
func NewCloudKMSKeyRingClient(keyRingClient *kms.KeyManagementClient) CloudKMSKeyRingClient {
	return &cloudKMSKeyRingClient{
		client: keyRingClient,
	}
}

// Get retrieves a KMS KeyRing
func (c cloudKMSKeyRingClient) Get(ctx context.Context, req *kmspb.GetKeyRingRequest, opts ...gax.CallOption) (*kmspb.KeyRing, error) {
	return c.client.GetKeyRing(ctx, req, opts...)
}

// List lists KMS KeyRings and returns an iterator
func (c cloudKMSKeyRingClient) Search(ctx context.Context, req *kmspb.ListKeyRingsRequest, opts ...gax.CallOption) CloudKMSKeyRingIterator {
	return c.client.ListKeyRings(ctx, req, opts...)
}
