package shared

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	log "github.com/sirupsen/logrus"
)

// NewAzureCredential creates a new DefaultAzureCredential which automatically handles
// multiple authentication methods in the following order:
// 1. Environment variables (AZURE_CLIENT_ID, AZURE_TENANT_ID, AZURE_FEDERATED_TOKEN_FILE, etc.)
// 2. Workload Identity (Kubernetes with OIDC federation)
// 3. Managed Identity (when running in Azure)
// 4. Azure CLI (for local development)
//
// Reference: https://learn.microsoft.com/en-us/azure/developer/go/sdk/authentication/credential-chains
func NewAzureCredential(ctx context.Context) (*azidentity.DefaultAzureCredential, error) {
	log.Debug("Initializing Azure credentials using DefaultAzureCredential")

	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Azure credential: %w", err)
	}

	log.WithFields(log.Fields{
		"ovm.auth.method": "default-azure-credential",
		"ovm.auth.type":   "federated-or-environment",
	}).Info("Successfully initialized Azure credentials")

	return cred, nil
}
