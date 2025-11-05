package shared

import (
	"context"
	"fmt"
	"net/http"

	"cloud.google.com/go/auth/credentials"
	"cloud.google.com/go/auth/credentials/impersonate"
	"cloud.google.com/go/auth/httptransport"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// GCPHTTPClientWithOtel creates a new HTTP client for GCP with OpenTelemetry instrumentation.
// If impersonationServiceAccountEmail is non-empty, it will impersonate that service account.
func GCPHTTPClientWithOtel(ctx context.Context, impersonationServiceAccountEmail string) (*http.Client, error) {
	// Use default credentials
	creds, err := credentials.DetectDefault(&credentials.DetectOptions{
		// Broad access to all GCP resources
		// It is restricted by the IAM permissions of the service account
		Scopes: []string{"https://www.googleapis.com/auth/cloud-platform"},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to detect default credentials: %w", err)
	}

	if impersonationServiceAccountEmail != "" {
		// Use impersonation credentials
		creds, err = impersonate.NewCredentials(&impersonate.CredentialsOptions{
			TargetPrincipal: impersonationServiceAccountEmail,
			// Broad access to all GCP resources
			// It is restricted by the IAM permissions of the service account
			Scopes: []string{"https://www.googleapis.com/auth/cloud-platform"},

			// piggy-back on top of the detected default credentials
			Credentials: creds,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create impersonated credentials: %w", err)
		}
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
