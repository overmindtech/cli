package shared

import (
	"fmt"
	"net/http"

	"cloud.google.com/go/auth/credentials"
	"cloud.google.com/go/auth/httptransport"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// GCPHTTPClientWithOtel creates a new HTTP client for GCP with OpenTelemetry instrumentation.
func GCPHTTPClientWithOtel() (*http.Client, error) {
	creds, err := credentials.DetectDefault(&credentials.DetectOptions{
		// Broad access to all GCP resources
		// It is restricted by the IAM permissions of the service account
		Scopes: []string{"https://www.googleapis.com/auth/cloud-platform"},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to detect default credentials: %w", err)
	}

	gcpHTTPCli, err := httptransport.NewClient(&httptransport.Options{
		Credentials:      creds,
		BaseRoundTripper: otelhttp.NewTransport(nil),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client with credentials: %w", err)
	}

	return gcpHTTPCli, nil
}
