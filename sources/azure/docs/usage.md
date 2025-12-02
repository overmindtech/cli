# Azure Source Usage Guide

## Quick Start

This guide provides quick configuration examples for running the Azure source in various environments.

## Prerequisites

1. **Azure Subscription**: An active Azure subscription
2. **Azure AD Application**: A registered application in Azure AD with appropriate permissions
3. **Permissions**: At minimum, Reader role on the subscription

## Configuration Methods

The Azure source can be configured using:
1. **Command-line flags**
2. **Environment variables**
3. **Configuration file** (YAML)

### Environment Variables

Environment variables use underscores instead of hyphens and are automatically uppercased:
- Flag: `--azure-subscription-id` → Environment: `AZURE_SUBSCRIPTION_ID`
- Flag: `--azure-tenant-id` → Environment: `AZURE_TENANT_ID`
- Flag: `--azure-client-id` → Environment: `AZURE_CLIENT_ID`

## Common Scenarios

### Scenario 1: Local Development with Azure CLI

**Use Case:** Testing the source on your local machine

**Prerequisites:**
```bash
# Install Azure CLI
# https://learn.microsoft.com/en-us/cli/azure/install-azure-cli

# Login to Azure
az login

# Set active subscription (optional, if you have multiple)
az account set --subscription "your-subscription-name-or-id"

# Verify current subscription
az account show
```

**Configuration:**
```bash
# Set required environment variables
export AZURE_SUBSCRIPTION_ID="00000000-0000-0000-0000-000000000000"
export AZURE_TENANT_ID="00000000-0000-0000-0000-000000000000"
export AZURE_CLIENT_ID="00000000-0000-0000-0000-000000000000"

# Run the source
./azure-source
```

**Command-line Alternative:**
```bash
./azure-source \
  --azure-subscription-id="00000000-0000-0000-0000-000000000000" \
  --azure-tenant-id="00000000-0000-0000-0000-000000000000" \
  --azure-client-id="00000000-0000-0000-0000-000000000000"
```

### Scenario 2: Service Principal with Client Secret

**Use Case:** CI/CD pipelines, Docker containers, non-Azure environments

**Setup Service Principal:**
```bash
# Create a service principal
az ad sp create-for-rbac --name "overmind-azure-source" \
  --role Reader \
  --scopes "/subscriptions/00000000-0000-0000-0000-000000000000"

# Output will include:
# {
#   "appId": "00000000-0000-0000-0000-000000000000",
#   "displayName": "overmind-azure-source",
#   "password": "your-client-secret",
#   "tenant": "00000000-0000-0000-0000-000000000000"
# }
```

**Configuration:**
```bash
export AZURE_SUBSCRIPTION_ID="00000000-0000-0000-0000-000000000000"
export AZURE_TENANT_ID="00000000-0000-0000-0000-000000000000"  # From 'tenant' in output
export AZURE_CLIENT_ID="00000000-0000-0000-0000-000000000000"  # From 'appId' in output
export AZURE_CLIENT_SECRET="your-client-secret"                 # From 'password' in output

# Run the source
./azure-source
```

**Docker Example:**
```dockerfile
FROM ubuntu:22.04

COPY azure-source /usr/local/bin/

ENV AZURE_SUBSCRIPTION_ID="00000000-0000-0000-0000-000000000000"
ENV AZURE_TENANT_ID="00000000-0000-0000-0000-000000000000"
ENV AZURE_CLIENT_ID="00000000-0000-0000-0000-000000000000"
# Client secret should be passed at runtime, not baked into image
# docker run -e AZURE_CLIENT_SECRET="..." your-image

ENTRYPOINT ["/usr/local/bin/azure-source"]
```

### Scenario 3: Kubernetes with Workload Identity

**Use Case:** Running in Kubernetes (AKS, EKS, GKE) with Azure Workload Identity

**Prerequisites:**
1. Azure Workload Identity installed in cluster
2. OIDC issuer configured
3. Federated identity credential configured in Azure AD

**Setup Azure Workload Identity:**

1. **Enable OIDC on your cluster** (example for AKS):
```bash
az aks update \
  --resource-group myResourceGroup \
  --name myAKSCluster \
  --enable-oidc-issuer \
  --enable-workload-identity
```

2. **Get OIDC Issuer URL:**
```bash
az aks show --resource-group myResourceGroup --name myAKSCluster \
  --query "oidcIssuerProfile.issuerUrl" -o tsv
```

3. **Create Azure AD Application:**
```bash
az ad app create --display-name overmind-azure-source
```

4. **Create Federated Credential:**
```bash
az ad app federated-credential create \
  --id <APPLICATION_OBJECT_ID> \
  --parameters '{
    "name": "overmind-k8s-federation",
    "issuer": "<OIDC_ISSUER_URL>",
    "subject": "system:serviceaccount:default:overmind-azure-source",
    "audiences": ["api://AzureADTokenExchange"]
  }'
```

5. **Assign Reader role:**
```bash
az role assignment create \
  --role Reader \
  --assignee <APPLICATION_CLIENT_ID> \
  --scope /subscriptions/<SUBSCRIPTION_ID>
```

**Kubernetes Deployment:**
```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: overmind-azure-source
  namespace: default
  annotations:
    azure.workload.identity/client-id: "00000000-0000-0000-0000-000000000000"
    azure.workload.identity/tenant-id: "00000000-0000-0000-0000-000000000000"
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: azure-source
  namespace: default
spec:
  replicas: 1
  selector:
    matchLabels:
      app: azure-source
  template:
    metadata:
      labels:
        app: azure-source
        azure.workload.identity/use: "true"  # Important!
    spec:
      serviceAccountName: overmind-azure-source
      containers:
      - name: azure-source
        image: your-registry/azure-source:latest
        env:
        - name: AZURE_SUBSCRIPTION_ID
          value: "00000000-0000-0000-0000-000000000000"
        - name: AZURE_TENANT_ID
          value: "00000000-0000-0000-0000-000000000000"
        - name: AZURE_CLIENT_ID
          value: "00000000-0000-0000-0000-000000000000"
        # AZURE_FEDERATED_TOKEN_FILE is set automatically by the webhook
```

### Scenario 4: Azure VM with Managed Identity

**Use Case:** Running on an Azure Virtual Machine

**Setup:**

1. **Enable System-Assigned Managed Identity on VM:**
```bash
az vm identity assign \
  --resource-group myResourceGroup \
  --name myVM
```

2. **Assign Reader role to the managed identity:**
```bash
# Get the principal ID
PRINCIPAL_ID=$(az vm show --resource-group myResourceGroup --name myVM \
  --query identity.principalId -o tsv)

# Assign role
az role assignment create \
  --role Reader \
  --assignee $PRINCIPAL_ID \
  --scope /subscriptions/<SUBSCRIPTION_ID>
```

**Configuration on VM:**
```bash
# Only subscription info is needed - managed identity is automatic
export AZURE_SUBSCRIPTION_ID="00000000-0000-0000-0000-000000000000"
export AZURE_TENANT_ID="00000000-0000-0000-0000-000000000000"
export AZURE_CLIENT_ID="00000000-0000-0000-0000-000000000000"

./azure-source
```

### Scenario 5: Specify Azure Regions (Optional)

**Use Case:** Limit discovery to specific regions for performance

**Configuration:**
```bash
export AZURE_SUBSCRIPTION_ID="00000000-0000-0000-0000-000000000000"
export AZURE_TENANT_ID="00000000-0000-0000-0000-000000000000"
export AZURE_CLIENT_ID="00000000-0000-0000-0000-000000000000"
export AZURE_REGIONS="eastus,westus2,northeurope"

./azure-source
```

**Command-line:**
```bash
./azure-source \
  --azure-subscription-id="00000000-0000-0000-0000-000000000000" \
  --azure-tenant-id="00000000-0000-0000-0000-000000000000" \
  --azure-client-id="00000000-0000-0000-0000-000000000000" \
  --azure-regions="eastus,westus2,northeurope"
```

**Note:** If regions are not specified, the source will discover resources in all regions.

## Configuration File

You can also use a YAML configuration file (default location: `/etc/srcman/config/source.yaml`):

```yaml
# Azure Configuration
azure-subscription-id: "00000000-0000-0000-0000-000000000000"
azure-tenant-id: "00000000-0000-0000-0000-000000000000"
azure-client-id: "00000000-0000-0000-0000-000000000000"
azure-regions: "eastus,westus2"

# Source Configuration
nats-url: "nats://nats:4222"
max-parallel-executions: 1000

# Logging
log: "info"  # panic, fatal, error, warn, info, debug, trace

# Health Check
health-check-port: 8080

# Tracing (Optional)
honeycomb-api-key: "your-honeycomb-key"
sentry-dsn: "your-sentry-dsn"
run-mode: "release"  # release, debug, or test
```

**Run with config file:**
```bash
./azure-source --config /path/to/config.yaml
```

## Available Flags

All configuration can be provided via command-line flags:

```bash
./azure-source --help

Flags:
  # Azure-specific flags
  --azure-subscription-id string   Azure Subscription ID that this source should operate in
  --azure-tenant-id string         Azure Tenant ID (Azure AD tenant) for authentication
  --azure-client-id string         Azure Client ID (Application ID) for federated credentials authentication
  --azure-regions string           Comma-separated list of Azure regions that this source should operate in

  # General flags
  --config string                  config file path (default "/etc/srcman/config/source.yaml")
  --log string                     Set the log level (default "info")
  --health-check-port int          The port that the health check should run on (default 8080)

  # NATS flags
  --nats-url string                NATS server URL
  --nats-name-prefix string        Prefix for NATS connection name
  --max-parallel-executions int    Max number of requests to execute in parallel

  # Tracing flags
  --honeycomb-api-key string       Honeycomb API key for tracing
  --sentry-dsn string              Sentry DSN for error tracking
  --run-mode string                Run mode: release, debug, or test (default "release")
```

## Health Check

The source exposes a health check endpoint:

```bash
# Check health
curl http://localhost:8080/healthz

# Response: "ok" (HTTP 200) if healthy
# Response: Error message (HTTP 500) if unhealthy
```

The health check verifies:
1. Source is running
2. Credentials are valid
3. Subscription is accessible

## Troubleshooting

### Check Logs

```bash
# Enable debug logging
export LOG_LEVEL=debug
./azure-source

# Or with flag
./azure-source --log=debug
```

### Verify Authentication

```bash
# Test Azure CLI authentication
az account show

# Test service principal authentication
az login --service-principal \
  --username $AZURE_CLIENT_ID \
  --password $AZURE_CLIENT_SECRET \
  --tenant $AZURE_TENANT_ID

# List resource groups to verify permissions
az group list --subscription $AZURE_SUBSCRIPTION_ID
```

### Common Issues

**Issue:** "failed to create Azure credential"
- **Solution:** Verify environment variables are set correctly. For local development, ensure `az login` is completed.

**Issue:** "failed to verify subscription access"
- **Solution:** Verify the identity has Reader role on the subscription. Check subscription ID is correct.

**Issue:** "No resource groups found"
- **Solution:** This may be normal if the subscription has no resource groups. The source will still run successfully.

## Best Practices

1. **Use Workload Identity in Production**: Most secure method, no credential management needed
2. **Never Hard-code Secrets**: Always use environment variables or secret management systems
3. **Use Least Privilege**: Grant only Reader role unless write access is needed
4. **Rotate Credentials**: If using service principals, rotate secrets regularly
5. **Monitor Health Endpoint**: Integrate health checks into your orchestration system
6. **Enable Tracing**: Use Honeycomb and Sentry for production observability

## Next Steps

- See [federated-credentials.md](./federated-credentials.md) for detailed authentication information
- See [testing-federated-auth.md](./testing-federated-auth.md) for testing scenarios with external identities
- Review [Azure RBAC documentation](https://learn.microsoft.com/en-us/azure/role-based-access-control/) for permission management

## Support

For issues or questions:
1. Check logs with `--log=debug`
2. Verify Azure permissions with Azure CLI
3. Review the federated credentials documentation
4. Check the health endpoint for status

