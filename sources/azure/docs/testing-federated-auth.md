# Testing Azure Federated Authentication

## Overview

This document provides comprehensive testing scenarios for Azure federated authentication, including cross-cloud identity federation from AWS and GCP. These scenarios help verify that the Azure source correctly handles federated credentials in various deployment contexts.

## Table of Contents

1. [Local Testing with Azure CLI](#local-testing-with-azure-cli)
2. [Service Principal Testing](#service-principal-testing)
3. [AWS Identity to Azure Federation](#aws-identity-to-azure-federation)
4. [GCP Service Account to Azure Federation](#gcp-service-account-to-azure-federation)
5. [Kubernetes Workload Identity Testing](#kubernetes-workload-identity-testing)
6. [Verification and Validation](#verification-and-validation)

## Prerequisites

### Azure Setup

1. **Azure Subscription** with resources to discover
2. **Azure AD Application** registered
3. **Reader role** assigned to the application on the subscription
4. **Resource Groups and VMs** created for testing (optional but recommended)

### Tools Required

- Azure CLI (`az`)
- AWS CLI (`aws`) - for AWS federation testing
- GCP CLI (`gcloud`) - for GCP federation testing
- `kubectl` - for Kubernetes testing
- `curl` or similar HTTP client
- `jq` - for JSON parsing

---

## Local Testing with Azure CLI

### Objective
Verify that the Azure source works with Azure CLI credentials on a developer workstation.

### Setup

1. **Install Azure CLI:**
```bash
# macOS
brew install azure-cli

# Linux
curl -sL https://aka.ms/InstallAzureCLIDeb | sudo bash

# Windows
# Download from https://aka.ms/installazurecliwindows
```

2. **Login to Azure:**
```bash
az login
```

3. **Select subscription:**
```bash
# List available subscriptions
az account list --output table

# Set active subscription
az account set --subscription "your-subscription-id"

# Verify
az account show
```

### Configuration

```bash
# Set environment variables
export AZURE_SUBSCRIPTION_ID=$(az account show --query id -o tsv)
export AZURE_TENANT_ID=$(az account show --query tenantId -o tsv)
export AZURE_CLIENT_ID="00000000-0000-0000-0000-000000000000"  # Your app's client ID
export LOG_LEVEL=debug
```

### Run the Source

```bash
cd /workspace/sources/azure
go run main.go
```

### Expected Output

```
INFO Using config from viper
INFO Successfully initialized Azure credentials
INFO Discovered resource groups count=5
INFO Initialized Azure adapters adapter_count=5
INFO Successfully verified subscription access
INFO Starting healthcheck server port=8080
INFO Sources initialized
```

### Verification

```bash
# Check health endpoint
curl http://localhost:8080/healthz
# Expected: "ok"

# Check logs for authentication method
# Should see: "Successfully initialized Azure credentials"
```

### Success Criteria

- ✅ Source starts without errors
- ✅ Health check returns "ok"
- ✅ Resource groups discovered
- ✅ Adapters initialized for each resource group
- ✅ No authentication errors in logs

---

## Service Principal Testing

### Objective
Verify authentication using a service principal with client secret.

### Setup

1. **Create Service Principal:**
```bash
# Create with Reader role on subscription
az ad sp create-for-rbac \
  --name "test-overmind-azure-source" \
  --role Reader \
  --scopes "/subscriptions/$(az account show --query id -o tsv)" \
  --output json > sp-credentials.json

# View credentials
cat sp-credentials.json
```

2. **Extract Credentials:**
```bash
export AZURE_SUBSCRIPTION_ID=$(az account show --query id -o tsv)
export AZURE_TENANT_ID=$(jq -r '.tenant' sp-credentials.json)
export AZURE_CLIENT_ID=$(jq -r '.appId' sp-credentials.json)
export AZURE_CLIENT_SECRET=$(jq -r '.password' sp-credentials.json)
export LOG_LEVEL=debug
```

### Test Service Principal

```bash
# Verify the service principal can authenticate
az login --service-principal \
  --username $AZURE_CLIENT_ID \
  --password $AZURE_CLIENT_SECRET \
  --tenant $AZURE_TENANT_ID

# List resource groups to verify permissions
az group list --output table

# Logout (so the source uses environment variables, not CLI cache)
az logout
```

### Run the Source

```bash
cd /workspace/sources/azure
go run main.go
```

### Expected Output

```
DEBUG Initializing Azure credentials using DefaultAzureCredential
INFO Successfully initialized Azure credentials auth.method=default-azure-credential
INFO Discovered resource groups count=5
INFO Successfully verified subscription access
```

### Verification

```bash
# Monitor logs for authentication
# Should use environment variables, not Azure CLI

# Verify it still works after Azure CLI logout
curl http://localhost:8080/healthz
```

### Cleanup

```bash
# Delete test service principal
az ad sp delete --id $AZURE_CLIENT_ID

# Remove credentials file
rm sp-credentials.json
```

### Success Criteria

- ✅ Authentication works without Azure CLI session
- ✅ Service principal credentials used from environment
- ✅ All resources discovered successfully
- ✅ Health check passes

---

## AWS Identity to Azure Federation

### Objective
Configure AWS IAM identity to authenticate to Azure using OIDC federation, simulating a scenario where the Azure source runs in EKS with AWS IRSA.

### Architecture

```
AWS EKS Pod → AWS IAM Role → OIDC Token → Azure AD Federated Credential → Azure Access
```

### Prerequisites

- AWS account with EKS cluster
- Azure subscription and Azure AD tenant
- OIDC issuer configured on EKS cluster

### Step 1: Configure Azure AD Application

```bash
# Create Azure AD application
az ad app create --display-name "test-aws-to-azure-federation" \
  --output json > azure-app.json

APP_OBJECT_ID=$(jq -r '.id' azure-app.json)
APP_CLIENT_ID=$(jq -r '.appId' azure-app.json)

echo "Azure AD Application Client ID: $APP_CLIENT_ID"
```

### Step 2: Get AWS EKS OIDC Issuer

```bash
# Get OIDC issuer URL from your EKS cluster
export OIDC_ISSUER=$(aws eks describe-cluster \
  --name your-eks-cluster-name \
  --query "cluster.identity.oidc.issuer" \
  --output text)

# Remove https:// prefix
export OIDC_ISSUER_URL=${OIDC_ISSUER#https://}

echo "OIDC Issuer: $OIDC_ISSUER"
```

### Step 3: Create Federated Identity Credential in Azure

```bash
# Create federated credential that trusts AWS EKS OIDC
az ad app federated-credential create \
  --id $APP_OBJECT_ID \
  --parameters '{
    "name": "aws-eks-federation",
    "issuer": "'"$OIDC_ISSUER"'",
    "subject": "system:serviceaccount:default:azure-source-sa",
    "audiences": ["sts.amazonaws.com"],
    "description": "Federated credential for AWS EKS to Azure"
  }'

# Verify creation
az ad app federated-credential list --id $APP_OBJECT_ID
```

### Step 4: Assign Azure Permissions

```bash
# Create service principal from app
az ad sp create --id $APP_CLIENT_ID

# Assign Reader role
az role assignment create \
  --role Reader \
  --assignee $APP_CLIENT_ID \
  --scope /subscriptions/$(az account show --query id -o tsv)
```

### Step 5: Configure AWS IAM Role

```bash
# Create IAM role with trust policy for EKS service account
cat > trust-policy.json <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Federated": "arn:aws:iam::YOUR_AWS_ACCOUNT_ID:oidc-provider/$OIDC_ISSUER_URL"
      },
      "Action": "sts:AssumeRoleWithWebIdentity",
      "Condition": {
        "StringEquals": {
          "$OIDC_ISSUER_URL:sub": "system:serviceaccount:default:azure-source-sa",
          "$OIDC_ISSUER_URL:aud": "sts.amazonaws.com"
        }
      }
    }
  ]
}
EOF

# Create IAM role
aws iam create-role \
  --role-name azure-source-eks-role \
  --assume-role-policy-document file://trust-policy.json

# Get role ARN
ROLE_ARN=$(aws iam get-role --role-name azure-source-eks-role --query 'Role.Arn' --output text)
echo "Role ARN: $ROLE_ARN"
```

### Step 6: Deploy to EKS

```yaml
# azure-source-deployment.yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: azure-source-sa
  namespace: default
  annotations:
    eks.amazonaws.com/role-arn: "arn:aws:iam::YOUR_ACCOUNT:role/azure-source-eks-role"
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
    spec:
      serviceAccountName: azure-source-sa
      containers:
      - name: azure-source
        image: your-registry/azure-source:latest
        env:
        - name: AZURE_SUBSCRIPTION_ID
          value: "your-subscription-id"
        - name: AZURE_TENANT_ID
          value: "your-tenant-id"
        - name: AZURE_CLIENT_ID
          value: "your-app-client-id"  # From Step 1
        - name: LOG_LEVEL
          value: "debug"
        # AWS will inject AWS_WEB_IDENTITY_TOKEN_FILE automatically
        ports:
        - containerPort: 8080
          name: health
      restartPolicy: Always
```

```bash
# Deploy
kubectl apply -f azure-source-deployment.yaml

# Wait for pod to be running
kubectl wait --for=condition=ready pod -l app=azure-source --timeout=60s
```

### Step 7: Verify

```bash
# Check pod logs
kubectl logs -l app=azure-source --tail=50

# Expected to see:
# - Successfully initialized Azure credentials
# - Discovered resource groups
# - Successfully verified subscription access

# Test health endpoint
kubectl port-forward deployment/azure-source 8080:8080 &
curl http://localhost:8080/healthz
```

### Troubleshooting

**Issue:** "DefaultAzureCredential failed to retrieve a token"

```bash
# Check if AWS token is being injected
kubectl exec -it deployment/azure-source -- env | grep AWS

# Should see:
# AWS_WEB_IDENTITY_TOKEN_FILE=/var/run/secrets/eks.amazonaws.com/serviceaccount/token
# AWS_ROLE_ARN=arn:aws:iam::...

# Check federated credential configuration
az ad app federated-credential list --id $APP_OBJECT_ID

# Verify OIDC issuer URL matches
kubectl exec -it deployment/azure-source -- cat /var/run/secrets/eks.amazonaws.com/serviceaccount/token | \
  jq -R 'split(".") | .[1] | @base64d | fromjson'
```

### Cleanup

```bash
# Delete Kubernetes resources
kubectl delete -f azure-source-deployment.yaml

# Delete Azure federated credential
az ad app federated-credential delete \
  --id $APP_OBJECT_ID \
  --federated-credential-id aws-eks-federation

# Delete Azure AD app
az ad app delete --id $APP_OBJECT_ID

# Delete AWS IAM role
aws iam delete-role --role-name azure-source-eks-role
```

### Success Criteria

- ✅ AWS OIDC token successfully exchanged for Azure token
- ✅ Azure resources discovered from EKS pod
- ✅ No authentication errors
- ✅ Health check passes continuously

---

## GCP Service Account to Azure Federation

### Objective
Configure GCP service account to authenticate to Azure using workload identity federation.

### Architecture

```
GKE Pod → GCP Service Account → OIDC Token → Azure AD Federated Credential → Azure Access
```

### Prerequisites

- GCP project with GKE cluster
- Workload Identity enabled on GKE
- Azure subscription and Azure AD tenant

### Step 1: Setup GCP Workload Identity

```bash
# Set variables
export PROJECT_ID="your-gcp-project"
export CLUSTER_NAME="your-gke-cluster"
export REGION="us-central1"

# Enable Workload Identity on cluster (if not already enabled)
gcloud container clusters update $CLUSTER_NAME \
  --workload-pool=$PROJECT_ID.svc.id.goog \
  --region=$REGION

# Create GCP service account
gcloud iam service-accounts create azure-source-gsa \
  --display-name="Azure Source GKE Service Account" \
  --project=$PROJECT_ID

export GSA_EMAIL="azure-source-gsa@${PROJECT_ID}.iam.gserviceaccount.com"
```

### Step 2: Get GKE OIDC Issuer

```bash
# GKE OIDC issuer format
export OIDC_ISSUER="https://container.googleapis.com/v1/projects/$PROJECT_ID/locations/$REGION/clusters/$CLUSTER_NAME"

echo "GKE OIDC Issuer: $OIDC_ISSUER"
```

### Step 3: Configure Azure AD Application

```bash
# Create Azure AD application
az ad app create --display-name "test-gcp-to-azure-federation" \
  --output json > azure-app-gcp.json

APP_OBJECT_ID=$(jq -r '.id' azure-app-gcp.json)
APP_CLIENT_ID=$(jq -r '.appId' azure-app-gcp.json)

# Create federated credential
az ad app federated-credential create \
  --id $APP_OBJECT_ID \
  --parameters '{
    "name": "gcp-gke-federation",
    "issuer": "'"$OIDC_ISSUER"'",
    "subject": "system:serviceaccount:default:azure-source-ksa",
    "audiences": ["azure"],
    "description": "Federated credential for GCP GKE to Azure"
  }'

# Create service principal and assign Reader role
az ad sp create --id $APP_CLIENT_ID
az role assignment create \
  --role Reader \
  --assignee $APP_CLIENT_ID \
  --scope /subscriptions/$(az account show --query id -o tsv)
```

### Step 4: Configure GKE Resources

```yaml
# azure-source-gke.yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: azure-source-ksa
  namespace: default
  annotations:
    iam.gke.io/gcp-service-account: azure-source-gsa@YOUR_PROJECT.iam.gserviceaccount.com
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
    spec:
      serviceAccountName: azure-source-ksa
      containers:
      - name: azure-source
        image: your-registry/azure-source:latest
        env:
        - name: AZURE_SUBSCRIPTION_ID
          value: "your-azure-subscription-id"
        - name: AZURE_TENANT_ID
          value: "your-azure-tenant-id"
        - name: AZURE_CLIENT_ID
          value: "your-azure-app-client-id"
        - name: LOG_LEVEL
          value: "debug"
        # GKE will inject GOOGLE_APPLICATION_CREDENTIALS automatically
```

### Step 5: Bind Service Accounts

```bash
# Allow Kubernetes service account to impersonate GCP service account
gcloud iam service-accounts add-iam-policy-binding $GSA_EMAIL \
  --role roles/iam.workloadIdentityUser \
  --member "serviceAccount:$PROJECT_ID.svc.id.goog[default/azure-source-ksa]"

# Deploy to GKE
kubectl apply -f azure-source-gke.yaml

# Wait for pod
kubectl wait --for=condition=ready pod -l app=azure-source --timeout=60s
```

### Step 6: Verify

```bash
# Check logs
kubectl logs -l app=azure-source --tail=50

# Check health
kubectl port-forward deployment/azure-source 8080:8080 &
curl http://localhost:8080/healthz

# Verify GCP token is available
kubectl exec -it deployment/azure-source -- env | grep GOOGLE
```

### Troubleshooting

```bash
# Check workload identity binding
gcloud iam service-accounts get-iam-policy $GSA_EMAIL

# Verify token can be obtained
kubectl exec -it deployment/azure-source -- \
  gcloud auth print-identity-token

# Check Azure federated credential
az ad app federated-credential list --id $APP_OBJECT_ID
```

### Cleanup

```bash
# Delete GKE resources
kubectl delete -f azure-source-gke.yaml

# Delete GCP service account
gcloud iam service-accounts delete $GSA_EMAIL --quiet

# Delete Azure resources
az ad app federated-credential delete \
  --id $APP_OBJECT_ID \
  --federated-credential-id gcp-gke-federation
az ad app delete --id $APP_OBJECT_ID
```

### Success Criteria

- ✅ GCP OIDC token exchanged for Azure credentials
- ✅ Source authenticates to Azure from GKE
- ✅ Resources discovered successfully
- ✅ Health check passes

---

## Kubernetes Workload Identity Testing

### Objective
Test native Azure Workload Identity in AKS (Azure Kubernetes Service).

### Prerequisites

- AKS cluster with OIDC issuer and Workload Identity enabled
- Azure AD application registered
- Azure Workload Identity webhook installed

### Setup

```bash
# Enable OIDC and Workload Identity on AKS
az aks update \
  --resource-group myResourceGroup \
  --name myAKSCluster \
  --enable-oidc-issuer \
  --enable-workload-identity

# Install Azure Workload Identity webhook (if not installed)
helm repo add azure-workload-identity https://azure.github.io/azure-workload-identity/charts
helm install workload-identity-webhook azure-workload-identity/workload-identity-webhook \
  --namespace azure-workload-identity-system \
  --create-namespace

# Get OIDC issuer URL
export OIDC_ISSUER_URL=$(az aks show \
  --resource-group myResourceGroup \
  --name myAKSCluster \
  --query "oidcIssuerProfile.issuerUrl" -o tsv)
```

### Configure Azure AD

```bash
# Create application
az ad app create --display-name "azure-source-aks-workload-id" \
  --output json > app.json

APP_OBJECT_ID=$(jq -r '.id' app.json)
APP_CLIENT_ID=$(jq -r '.appId' app.json)

# Create federated credential
az ad app federated-credential create \
  --id $APP_OBJECT_ID \
  --parameters "{
    \"name\": \"aks-workload-identity\",
    \"issuer\": \"$OIDC_ISSUER_URL\",
    \"subject\": \"system:serviceaccount:default:azure-source-sa\",
    \"audiences\": [\"api://AzureADTokenExchange\"]
  }"

# Assign permissions
az ad sp create --id $APP_CLIENT_ID
az role assignment create \
  --role Reader \
  --assignee $APP_CLIENT_ID \
  --scope /subscriptions/$(az account show --query id -o tsv)
```

### Deploy

```yaml
# azure-source-aks.yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: azure-source-sa
  annotations:
    azure.workload.identity/client-id: "YOUR_APP_CLIENT_ID"
    azure.workload.identity/tenant-id: "YOUR_TENANT_ID"
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: azure-source
spec:
  replicas: 1
  selector:
    matchLabels:
      app: azure-source
  template:
    metadata:
      labels:
        app: azure-source
        azure.workload.identity/use: "true"
    spec:
      serviceAccountName: azure-source-sa
      containers:
      - name: azure-source
        image: your-registry/azure-source:latest
        env:
        - name: AZURE_SUBSCRIPTION_ID
          value: "your-subscription-id"
        - name: AZURE_TENANT_ID
          value: "your-tenant-id"
        - name: AZURE_CLIENT_ID
          value: "your-client-id"
```

```bash
kubectl apply -f azure-source-aks.yaml
kubectl wait --for=condition=ready pod -l app=azure-source --timeout=60s
kubectl logs -l app=azure-source
```

### Success Criteria

- ✅ Workload Identity webhook injects token volume
- ✅ Source authenticates using projected token
- ✅ Resources discovered
- ✅ Health check passes

---

## Verification and Validation

### Standard Checks

After completing any test scenario, perform these verification steps:

#### 1. Health Check

```bash
# Forward port
kubectl port-forward deployment/azure-source 8080:8080 &

# Check health
curl http://localhost:8080/healthz

# Expected: "ok"
```

#### 2. Log Analysis

```bash
# Check for successful authentication
kubectl logs -l app=azure-source | grep "Successfully initialized Azure credentials"

# Check for resource discovery
kubectl logs -l app=azure-source | grep "Discovered resource groups"

# Check for subscription verification
kubectl logs -l app=azure-source | grep "Successfully verified subscription access"

# Look for errors
kubectl logs -l app=azure-source | grep -i error
```

#### 3. Metrics and Observability

If Honeycomb/Sentry integration is enabled:

```bash
# Check traces in Honeycomb for:
# - Authentication attempts
# - Resource discovery operations
# - Health check calls

# Check Sentry for any error reports
```

### Validation Checklist

- [ ] Source starts successfully
- [ ] No authentication errors
- [ ] Subscription access verified
- [ ] Resource groups discovered
- [ ] Adapters initialized
- [ ] Health check returns 200 OK
- [ ] Logs show expected authentication method
- [ ] No error traces in observability tools
- [ ] Source survives pod restarts
- [ ] Token refresh works (for long-running tests)

### Performance Testing

```bash
# Measure startup time
kubectl logs -l app=azure-source --timestamps | \
  awk '/Started/ {print $1}'

# Check memory usage
kubectl top pod -l app=azure-source

# Monitor over time
watch -n 5 'kubectl top pod -l app=azure-source'
```

### Common Issues and Solutions

| Issue | Possible Cause | Solution |
|-------|---------------|----------|
| "DefaultAzureCredential failed" | No auth method available | Check environment variables, verify OIDC token injection |
| "Failed to verify subscription access" | Insufficient permissions | Verify Reader role assignment |
| "Failed to list resource groups" | Network/permissions issue | Check network policies, verify subscription ID |
| Pod crashloops | Invalid configuration | Check logs with `kubectl logs`, verify all required env vars |
| Health check fails | Credentials expired/invalid | Check credential validity, verify RBAC |

## Summary

This testing guide covers:
- ✅ Local development with Azure CLI
- ✅ Service principal authentication
- ✅ AWS to Azure federation (EKS→Azure)
- ✅ GCP to Azure federation (GKE→Azure)
- ✅ Native Azure Workload Identity (AKS)
- ✅ Comprehensive verification steps

These scenarios ensure the Azure source correctly handles federated credentials across all deployment contexts.

