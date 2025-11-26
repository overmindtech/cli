package shared

import (
	"context"
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// AzureHTTPClientWithOtel creates a new HTTP client for Azure with OpenTelemetry instrumentation.
// Azure SDK clients handle authentication automatically via:
// - Federated credentials (when running in Kubernetes/EKS with workload identity)
// - Azure CLI (for local development)
// - Environment variables (AZURE_CLIENT_ID, AZURE_CLIENT_SECRET, AZURE_TENANT_ID)
// - Managed Identity (when running in Azure)
//
// This function returns an HTTP client with OpenTelemetry instrumentation that can be used
// with Azure SDK clients. The actual authentication is handled by the Azure SDK client options.
func AzureHTTPClientWithOtel(ctx context.Context) *http.Client {
	// Azure SDK handles authentication automatically, so we just need to provide
	// an HTTP client with OpenTelemetry instrumentation
	return &http.Client{
		Transport: otelhttp.NewTransport(nil),
	}
}
