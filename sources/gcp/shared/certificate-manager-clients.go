package shared

//go:generate mockgen -destination=./mocks/mock_certificate_manager_certificate_client.go -package=mocks -source=certificate-manager-clients.go

import (
	"context"

	certificatemanager "cloud.google.com/go/certificatemanager/apiv1"
	certificatemanagerpb "cloud.google.com/go/certificatemanager/apiv1/certificatemanagerpb"
	"github.com/googleapis/gax-go/v2"
)

// CertificateManagerCertificateClient interface for Certificate Manager Certificate operations
type CertificateManagerCertificateClient interface {
	GetCertificate(ctx context.Context, req *certificatemanagerpb.GetCertificateRequest, opts ...gax.CallOption) (*certificatemanagerpb.Certificate, error)
	ListCertificates(ctx context.Context, req *certificatemanagerpb.ListCertificatesRequest, opts ...gax.CallOption) CertificateIterator
}

type CertificateIterator interface {
	Next() (*certificatemanagerpb.Certificate, error)
}

type certificateManagerCertificateClient struct {
	client *certificatemanager.Client
}

func (c *certificateManagerCertificateClient) GetCertificate(ctx context.Context, req *certificatemanagerpb.GetCertificateRequest, opts ...gax.CallOption) (*certificatemanagerpb.Certificate, error) {
	return c.client.GetCertificate(ctx, req, opts...)
}

func (c *certificateManagerCertificateClient) ListCertificates(ctx context.Context, req *certificatemanagerpb.ListCertificatesRequest, opts ...gax.CallOption) CertificateIterator {
	return c.client.ListCertificates(ctx, req, opts...)
}

// NewCertificateManagerCertificateClient creates a new CertificateManagerCertificateClient
func NewCertificateManagerCertificateClient(client *certificatemanager.Client) CertificateManagerCertificateClient {
	return &certificateManagerCertificateClient{
		client: client,
	}
}
