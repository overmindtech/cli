# Azure Federated Credentials Implementation

## Overview

The Azure source now supports federated credential authentication using the Azure SDK's `DefaultAzureCredential`. This provides a flexible authentication mechanism that automatically handles multiple authentication methods, making it suitable for various deployment scenarios including Kubernetes workload identity, managed identity, and local development.

## How It Works

### DefaultAzureCredential Chain

The `DefaultAzureCredential` attempts authentication using multiple methods in the following order:

1. **Environment Variables** - Service principal or workload identity via environment variables
2. **Workload Identity** - Kubernetes/EKS with OIDC federation (via `AZURE_FEDERATED_TOKEN_FILE`)
3. **Managed Identity** - When running on Azure infrastructure (VMs, App Service, Functions, etc.)
4. **Azure CLI** - Uses credentials from `az login` (ideal for local development)

The first successful authentication method is used, and subsequent methods are not attempted.

### Implementation Details

#### Credential Initialization

The credential initialization is handled in `sources/azure/shared/credentials.go`:

```go
func NewAzureCredential(ctx context.Context) (*azidentity.DefaultAzureCredential, error) {
    cred, err := azidentity.NewDefaultAzureCredential(nil)
    if err != nil {
        return nil, fmt.Errorf("failed to create Azure credential: %w", err)
    }
    return cred, nil
}
```

#### Client Initialization

Azure SDK clients are initialized with the credential in `sources/azure/proc/proc.go`:

```go
// Initialize Azure credentials
cred, err := azureshared.NewAzureCredential(ctx)
if err != nil {
    return fmt.Errorf("error creating Azure credentials: %w", err)
}

// Pass credentials to adapters
discoveryAdapters, err := adapters(ctx, cfg.SubscriptionID, cfg.TenantID,
    cfg.ClientID, cfg.Regions, cred, linker, true)
```

#### Resource Group Discovery

The implementation automatically discovers all resource groups in the subscription and creates adapters for each:

```go
// Discover resource groups in the subscription
rgClient, err := armresources.NewResourceGroupsClient(subscriptionID, cred, nil)
pager := rgClient.NewListPager(nil)
for pager.More() {
    page, err := pager.NextPage(ctx)
    for _, rg := range page.Value {
        resourceGroups = append(resourceGroups, *rg.Name)
    }
}
```

#### Permission Verification

The source verifies subscription access at startup:

```go
func checkSubscriptionAccess(ctx context.Context, subscriptionID string,
    cred *azidentity.DefaultAzureCredential) error {

    client, err := armresources.NewResourceGroupsClient(subscriptionID, cred, nil)
    if err != nil {
        return fmt.Errorf("failed to create resource groups client: %w", err)
    }

    // Try to list resource groups to verify access
    pager := client.NewListPager(nil)
    _, err = pager.NextPage(ctx)
    if err != nil {
        return fmt.Errorf("failed to verify subscription access: %w", err)
    }

    return nil
}
```

## Environment Variables

### Required Variables

These variables must be set for the Azure source to function:

- `AZURE_SUBSCRIPTION_ID` - The Azure subscription ID to discover resources in
- `AZURE_TENANT_ID` - The Azure AD tenant ID
- `AZURE_CLIENT_ID` - The application/client ID

### Authentication Method Variables

Depending on your authentication method, you may need additional variables:

#### Service Principal with Client Secret

```bash
export AZURE_CLIENT_SECRET="your-client-secret"
```

#### Service Principal with Certificate

```bash
export AZURE_CLIENT_CERTIFICATE_PATH="/path/to/certificate.pem"
```

#### Federated Workload Identity (Kubernetes/EKS)

```bash
export AZURE_FEDERATED_TOKEN_FILE="/var/run/secrets/azure/tokens/azure-identity-token"
```

This is typically set automatically by the Azure Workload Identity webhook when running in Kubernetes with proper annotations.

## Authentication Methods

### 1. Workload Identity (Kubernetes with OIDC Federation)

**Use Case:** Running in Kubernetes clusters (AKS, EKS, GKE) with Azure Workload Identity configured.

**How It Works:**
- The Kubernetes pod is annotated with an Azure AD application
- Azure AD trusts the OIDC token from the Kubernetes cluster
- A federated token file is mounted into the pod
- `DefaultAzureCredential` reads this token and exchanges it for Azure credentials

**Configuration:**
```yaml
# Pod annotation
azure.workload.identity/client-id: "00000000-0000-0000-0000-000000000000"
azure.workload.identity/tenant-id: "00000000-0000-0000-0000-000000000000"

# Environment variables (set automatically by webhook)
AZURE_CLIENT_ID: "00000000-0000-0000-0000-000000000000"
AZURE_TENANT_ID: "00000000-0000-0000-0000-000000000000"
AZURE_FEDERATED_TOKEN_FILE: "/var/run/secrets/azure/tokens/azure-identity-token"
```

**Reference:** [Azure Workload Identity Documentation](https://azure.github.io/azure-workload-identity/docs/)

### 2. Service Principal (Environment Variables)

**Use Case:** CI/CD pipelines, containerized deployments, or any scenario where you have a service principal.

**Configuration:**
```bash
export AZURE_SUBSCRIPTION_ID="00000000-0000-0000-0000-000000000000"
export AZURE_TENANT_ID="00000000-0000-0000-0000-000000000000"
export AZURE_CLIENT_ID="00000000-0000-0000-0000-000000000000"
export AZURE_CLIENT_SECRET="your-client-secret"
```

### 3. Managed Identity

**Use Case:** Running on Azure infrastructure (VMs, App Service, Container Instances, etc.)

**How It Works:**
- Azure automatically provides credentials to the service
- No credentials need to be stored or configured
- `DefaultAzureCredential` automatically detects and uses managed identity

**Configuration:**
- System-assigned identity: No configuration needed
- User-assigned identity: Set `AZURE_CLIENT_ID` to the identity's client ID

### 4. Azure CLI (Local Development)

**Use Case:** Local development and testing

**Setup:**
```bash
# Login with Azure CLI
az login

# Set the subscription
az account set --subscription "your-subscription-id"
```

**Configuration:**
```bash
# Only subscription ID is needed from environment
export AZURE_SUBSCRIPTION_ID="00000000-0000-0000-0000-000000000000"
export AZURE_TENANT_ID="00000000-0000-0000-0000-000000000000"
export AZURE_CLIENT_ID="00000000-0000-0000-0000-000000000000"
```

The Azure source will use the credentials from `az login` automatically.

## Required Azure Permissions

The Azure source requires the following permissions on the subscription:

### Built-in Role
The minimum required role is **Reader** at the subscription level.

### Specific Permissions
- `Microsoft.Resources/subscriptions/resourceGroups/read` - List resource groups
- `Microsoft.Compute/virtualMachines/read` - Read virtual machines
- Additional read permissions for other resource types as adapters are added

## Troubleshooting

### Common Issues

#### 1. "DefaultAzureCredential failed to retrieve a token"

**Cause:** No valid authentication method is available.

**Solution:**
- Verify environment variables are set correctly
- For local development, run `az login`
- For workload identity, verify pod annotations and service account configuration

#### 2. "Failed to verify subscription access"

**Cause:** Credentials don't have access to the subscription, or subscription ID is incorrect.

**Solution:**
- Verify the subscription ID is correct
- Ensure the identity has at least Reader role on the subscription
- Check Azure AD tenant ID matches the subscription's tenant

#### 3. "Failed to list resource groups"

**Cause:** Missing permissions or network connectivity issues.

**Solution:**
- Verify the identity has `Microsoft.Resources/subscriptions/resourceGroups/read` permission
- Check network connectivity to Azure (firewall, proxy)
- Verify subscription ID is correct

### Debugging

Enable debug logging to see authentication details:

```bash
export LOG_LEVEL=debug
```

The logs will show:
- Which authentication method is being used
- Subscription access verification results
- Resource group discovery progress
- Adapter initialization details

## Security Best Practices

1. **Use Workload Identity in Kubernetes**: Preferred method as it avoids storing credentials
2. **Use Managed Identity on Azure**: No credential management needed
3. **Avoid Client Secrets in Code**: Always use environment variables
4. **Rotate Credentials Regularly**: If using service principals with secrets
5. **Principle of Least Privilege**: Grant only Reader role unless more is needed
6. **Separate Identities per Environment**: Don't reuse production credentials in development

## References

- [Azure Identity SDK for Go](https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/azidentity)
- [DefaultAzureCredential Documentation](https://learn.microsoft.com/en-us/azure/developer/go/sdk/authentication/credential-chains)
- [Azure Workload Identity](https://azure.github.io/azure-workload-identity/docs/)
- [Azure Managed Identity](https://learn.microsoft.com/en-us/azure/active-directory/managed-identities-azure-resources/overview)
- [Azure RBAC Roles](https://learn.microsoft.com/en-us/azure/role-based-access-control/built-in-roles)

