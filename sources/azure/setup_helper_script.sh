#!/bin/bash
set -e

# Azure App Registration Setup for Overmind Azure Source
# This script creates an Azure AD app with federated credentials for EKS OIDC authentication.
#
# Prerequisites:
#   - Azure CLI installed and logged in (az login)
#   - Appropriate permissions to create app registrations and role assignments
#
# Usage:
#   ./setup_helper_script.sh --customer-name <name> --eks-oidc-issuer <url> --azure-subscription-id <id> [--namespace <ns>]
#
# Arguments:
#   --customer-name          Overmind account name/ID (required)
#   --eks-oidc-issuer        EKS OIDC issuer URL (required)
#   --azure-subscription-id  Azure subscription ID (required)
#   --namespace              Kubernetes namespace (optional, default: default)

# === DEFAULT VALUES ===
NAMESPACE="default"

# === ARGUMENT PARSING ===
usage() {
    echo "Usage: $0 --customer-name <name> --eks-oidc-issuer <url> --azure-subscription-id <id> [--namespace <ns>]"
    echo ""
    echo "Arguments:"
    echo "  --customer-name          Overmind account name/ID (required)"
    echo "  --eks-oidc-issuer        EKS OIDC issuer URL (required)"
    echo "  --azure-subscription-id  Azure subscription ID (required)"
    echo "  --namespace              Kubernetes namespace (optional, default: default)"
    echo ""
    echo "Example:"
    echo "  $0 --customer-name my-account --eks-oidc-issuer https://oidc.eks.eu-west-2.amazonaws.com/id/ABC123 --azure-subscription-id 12345678-1234-1234-1234-123456789abc"
    exit 1
}

while [[ $# -gt 0 ]]; do
    case $1 in
        --customer-name)
            CUSTOMER_NAME="$2"
            shift 2
            ;;
        --eks-oidc-issuer)
            EKS_OIDC_ISSUER="$2"
            shift 2
            ;;
        --azure-subscription-id)
            AZURE_SUBSCRIPTION_ID="$2"
            shift 2
            ;;
        --namespace)
            NAMESPACE="$2"
            shift 2
            ;;
        -h|--help)
            usage
            ;;
        *)
            echo "Error: Unknown argument: $1"
            usage
            ;;
    esac
done

# === VALIDATION ===
MISSING_ARGS=()

if [ -z "$CUSTOMER_NAME" ]; then
    MISSING_ARGS+=("--customer-name")
fi

if [ -z "$EKS_OIDC_ISSUER" ]; then
    MISSING_ARGS+=("--eks-oidc-issuer")
fi

if [ -z "$AZURE_SUBSCRIPTION_ID" ]; then
    MISSING_ARGS+=("--azure-subscription-id")
fi

if [ ${#MISSING_ARGS[@]} -ne 0 ]; then
    echo "Error: Missing required arguments: ${MISSING_ARGS[*]}"
    echo ""
    usage
fi

# === DERIVED VALUES ===
APP_NAME="overmind-azure-source-${CUSTOMER_NAME}"
SERVICE_ACCOUNT_NAME="${CUSTOMER_NAME}-azure-source-pod-sa"
# Federated credential name - unique within the app registration
# Using a descriptive name that includes context about the EKS cluster
FEDERATED_CRED_NAME="eks-federated-${CUSTOMER_NAME:0:8}"

echo "=== Configuration ==="
echo "Customer Name: $CUSTOMER_NAME"
echo "App Name: $APP_NAME"
echo "ServiceAccount: $SERVICE_ACCOUNT_NAME"
echo "Namespace: $NAMESPACE"
echo "EKS OIDC Issuer: $EKS_OIDC_ISSUER"
echo ""

# Check if app already exists
EXISTING_APP_ID=$(az ad app list --display-name "$APP_NAME" --query "[0].appId" -o tsv 2>/dev/null || true)
if [ -n "$EXISTING_APP_ID" ]; then
    echo "App registration '$APP_NAME' already exists with ID: $EXISTING_APP_ID"
    echo "Using existing app..."
    APP_ID=$EXISTING_APP_ID
else
    echo "Creating Azure AD App Registration..."
    az ad app create \
      --display-name "$APP_NAME" \
      --sign-in-audience "AzureADMyOrg"

    APP_ID=$(az ad app list --display-name "$APP_NAME" --query "[0].appId" -o tsv)
    echo "Created app with ID: $APP_ID"
fi

TENANT_ID=$(az account show --query tenantId -o tsv)

# Check if service principal exists
SP_EXISTS=$(az ad sp show --id "$APP_ID" --query "appId" -o tsv 2>/dev/null || true)
if [ -n "$SP_EXISTS" ]; then
    echo "Service Principal already exists"
else
    echo "Creating Service Principal..."
    az ad sp create --id "$APP_ID"
fi

# Check if federated credential exists
EXISTING_CRED=$(az ad app federated-credential list --id "$APP_ID" --query "[?name=='$FEDERATED_CRED_NAME'].name" -o tsv 2>/dev/null || true)
if [ -n "$EXISTING_CRED" ]; then
    echo "Federated credential '$FEDERATED_CRED_NAME' already exists, updating..."
    az ad app federated-credential delete --id "$APP_ID" --federated-credential-id "$FEDERATED_CRED_NAME"
fi

echo "Creating Federated Credential..."
# Note: The 'subject' must exactly match the Kubernetes ServiceAccount that will be created by srcman
# Format: system:serviceaccount:<namespace>:<service-account-name>
az ad app federated-credential create \
  --id "$APP_ID" \
  --parameters '{
    "name": "'"$FEDERATED_CRED_NAME"'",
    "issuer": "'"$EKS_OIDC_ISSUER"'",
    "subject": "system:serviceaccount:'"$NAMESPACE"':'"$SERVICE_ACCOUNT_NAME"'",
    "audiences": ["api://AzureADTokenExchange"],
    "description": "Federated credential for Overmind Azure source running on EKS. Customer: '"$CUSTOMER_NAME"'"
  }'

# Check if role assignment exists
EXISTING_ROLE=$(az role assignment list --assignee "$APP_ID" --scope "/subscriptions/$AZURE_SUBSCRIPTION_ID" --query "[?roleDefinitionName=='Reader'].id" -o tsv 2>/dev/null || true)
if [ -n "$EXISTING_ROLE" ]; then
    echo "Reader role assignment already exists"
else
    echo "Assigning Reader role..."
    az role assignment create \
      --role "Reader" \
      --assignee "$APP_ID" \
      --scope "/subscriptions/$AZURE_SUBSCRIPTION_ID"
fi

echo ""
echo "=========================================="
echo "=== Azure Source Configuration Values ==="
echo "=========================================="
echo ""
echo "Use these values when creating the Azure source in Overmind:"
echo ""
echo "  azure-subscription-id: $AZURE_SUBSCRIPTION_ID"
echo "  azure-tenant-id:       $TENANT_ID"
echo "  azure-client-id:       $APP_ID"
echo ""
echo "The following Kubernetes ServiceAccount will be created by srcman:"
echo "  Namespace: $NAMESPACE"
echo "  Name:      $SERVICE_ACCOUNT_NAME"
echo ""
echo "Federated credential subject (must match exactly):"
echo "  system:serviceaccount:$NAMESPACE:$SERVICE_ACCOUNT_NAME"
echo ""