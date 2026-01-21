#!/bin/bash

# Script to check if the Overmind service account has the necessary permissions
# Can use command-line arguments or environment variables

set -euo pipefail  # Exit on error, undefined vars, and pipe failures

# Display usage information
function show_usage() {
    echo "Usage: $0 [options]"
    echo "Options:"
    echo "  -p, --project-id PROJECT_ID    GCP Project ID"
    echo "  -s, --service-account SA_EMAIL  Overmind service account email"
    echo "  -h, --help                     Show this help message"
    echo ""
    echo "You can also set these values through environment variables:"
    echo "  GCP_PROJECT_ID and GCP_OVERMIND_SA"
    exit 1
}

# Parse command-line arguments
while [[ $# -gt 0 ]]; do
    case "$1" in
        -p|--project-id)
            if [[ -n "${2:-}" ]]; then
                GCP_PROJECT_ID="$2"
                shift 2
            else
                echo "ERROR: Value for --project-id is missing"
                show_usage
            fi
            ;;
        -s|--service-account)
            if [[ -n "${2:-}" ]]; then
                GCP_OVERMIND_SA="$2"
                shift 2
            else
                echo "ERROR: Value for --service-account is missing"
                show_usage
            fi
            ;;
        -h|--help)
            show_usage
            ;;
        *)
            echo "ERROR: Unknown argument: $1"
            show_usage
            ;;
    esac
done

# Source environment variables from the local file if it exists and parameters weren't provided
# shellcheck source=/dev/null
if [[ (-z "${GCP_PROJECT_ID:-}" || -z "${GCP_OVERMIND_SA:-}") && -f ./.gcp-source-setup-env ]]; then
    source ./.gcp-source-setup-env
    echo "Successfully loaded environment variables from ./.gcp-source-setup-env"
fi

# Check if GCP_PROJECT_ID environment variable is set
if [[ -z "${GCP_PROJECT_ID:-}" ]]; then
    echo "ERROR: GCP Project ID is not provided"
    echo "Please specify the project ID using the --project-id option or run the overmind-gcp-source-setup.sh script first"
    show_usage
fi

# Check if GCP_OVERMIND_SA environment variable is set
if [[ -z "${GCP_OVERMIND_SA:-}" ]]; then
    echo "ERROR: Overmind service account email is not provided"
    echo "Please specify the service account using the --service-account option or run the overmind-gcp-source-setup.sh script first"
    show_usage
fi

echo "Checking permissions for service account: ${GCP_OVERMIND_SA}"
echo "on project: ${GCP_PROJECT_ID}"
echo ""

# @generator:inline-start:overmind-gcp-roles.sh
# This block is replaced with inlined role definitions during TypeScript generation
source "$(dirname "$0")/overmind-gcp-roles.sh"
# @generator:inline-end

# Fetch the current IAM policy
echo "Fetching current IAM policy for project ${GCP_PROJECT_ID}..."
IAM_POLICY=$(gcloud projects get-iam-policy "${GCP_PROJECT_ID}" --format=json)

# Check if fetch was successful
if [[ -z "${IAM_POLICY}" ]]; then
    echo "ERROR: Failed to fetch IAM policy for project ${GCP_PROJECT_ID}"
    exit 1
fi

# Create a temporary file for the policy
TEMP_FILE=$(mktemp)
echo "${IAM_POLICY}" > "${TEMP_FILE}"

# Counter for roles check
TOTAL_ROLES=${#ROLES[@]}
FOUND_ROLES=0
MISSING_ROLES=0

echo ""
echo "Checking for ${TOTAL_ROLES} required roles..."
echo "----------------------------------------"

for ROLE in "${ROLES[@]}"; do
    # Check if the role exists in the policy for the service account
    if grep -q "\"role\": \"${ROLE}\"" "${TEMP_FILE}" && \
       jq -e --arg ROLE "$ROLE" --arg SA "serviceAccount:${GCP_OVERMIND_SA}" \
        '.bindings[] | select(.role == $ROLE) | .members[] | select(. == $SA)' \
        "${TEMP_FILE}" >/dev/null; then
        echo "✓ Role exists: ${ROLE}"
        ((FOUND_ROLES++))
    else
        echo "✗ Role missing: ${ROLE}"
        ((MISSING_ROLES++))
    fi
done

# Clean up
rm "${TEMP_FILE}"

echo "----------------------------------------"
echo "Permission check completed:"
echo "  - Found roles: ${FOUND_ROLES}/${TOTAL_ROLES}"
echo "  - Missing roles: ${MISSING_ROLES}/${TOTAL_ROLES}"
echo ""

if [[ ${MISSING_ROLES} -eq 0 ]]; then
    echo "✅ All required permissions are correctly assigned to the Overmind service account."
    echo "   Your GCP source is ready for Overmind to access."
else
    echo "❌ Some required permissions are missing. Please run the setup script again:"
    echo "   ./overmind-gcp-source-setup.sh"
fi
