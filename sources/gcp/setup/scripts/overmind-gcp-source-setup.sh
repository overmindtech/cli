#!/bin/bash

# Script to add IAM policy bindings to a service account in GCP
# Takes GCP Parent (organizations/123, folders/456, or projects/my-project) and Overmind service account as arguments
#
# Usage: ./overmind-gcp-source-setup.sh <parent> <service-account-email>
#
# NOTE: The Overmind service account should be the service account email presented
# in the Overmind application when creating a new GCP source.

set -euo pipefail  # Exit on error, undefined vars, and pipe failures

# Check if both arguments are provided
if [[ $# -ne 2 ]]; then
    echo "ERROR: Both parent and service account email are required"
    echo "Usage: $0 <parent> <service-account-email>"
    echo "Parent format: organizations/123, folders/456, or projects/my-project"
    exit 1
fi

# Get arguments
GCP_PARENT="$1"
GCP_OVERMIND_SA="$2"

# Check if GCP_PARENT is empty
if [[ -z "${GCP_PARENT}" ]]; then
    echo "ERROR: GCP Parent cannot be empty"
    exit 1
fi

# Check if GCP_OVERMIND_SA is empty
if [[ -z "${GCP_OVERMIND_SA}" ]]; then
    echo "ERROR: Overmind service account email cannot be empty"
    echo "NOTE: Use the service account email presented in the Overmind application when creating a GCP source"
    exit 1
fi

# Parse parent to determine type and ID
PARENT="${GCP_PARENT}"
if [[ ${PARENT} =~ ^organizations?/([0-9]+)$ ]]; then
    PARENT_TYPE="organization"
    PARENT_ID="${BASH_REMATCH[1]}"
elif [[ ${PARENT} =~ ^folders?/([0-9]+)$ ]]; then
    PARENT_TYPE="folder"
    PARENT_ID="${BASH_REMATCH[1]}"
elif [[ ${PARENT} =~ ^projects?/([a-z][a-z0-9-]*[a-z0-9])$ ]]; then
    PARENT_TYPE="project"
    PARENT_ID="${BASH_REMATCH[1]}"
else
    echo "✗ Invalid parent format: ${PARENT}"
    echo "Must be: organizations/123, folders/456, or projects/my-project"
    exit 1
fi

echo "Detected parent type: ${PARENT_TYPE}"
echo "Parent ID: ${PARENT_ID}"

# Save the variables to a local file for other scripts to use
echo "export GCP_PARENT=\"${GCP_PARENT}\"" > ./.gcp-source-setup-env
echo "export GCP_PARENT_TYPE=\"${PARENT_TYPE}\"" >> ./.gcp-source-setup-env
echo "export GCP_PARENT_ID=\"${PARENT_ID}\"" >> ./.gcp-source-setup-env
echo "export GCP_OVERMIND_SA=\"${GCP_OVERMIND_SA}\"" >> ./.gcp-source-setup-env

echo "Using GCP Parent: ${GCP_PARENT}"
echo "Service Account: ${GCP_OVERMIND_SA}"

# @generator:inline-start:overmind-gcp-roles.sh
# This block is replaced with inlined role definitions during TypeScript generation
source "$(dirname "$0")/overmind-gcp-roles.sh"
# @generator:inline-end

# For project-level parents, create custom role
if [ "${PARENT_TYPE}" = "project" ]; then
    echo "Creating custom role for additional BigQuery and Spanner permissions..."
    if gcloud iam roles create overmindCustomRole \
        --project="${PARENT_ID}" \
        --title="Overmind Custom Role" \
        --description="Custom role for Overmind service account with additional BigQuery and Spanner permissions" \
        --permissions="bigquery.transfers.get,spanner.databases.get,spanner.databases.list" \
        --quiet > /dev/null 2>&1; then
        echo "✓ Successfully created custom role: overmindCustomRole"
    else
        echo "ℹ Custom role may already exist, continuing..."
    fi
fi

# Display the roles that will be added
echo ""
echo "This script will assign the following predefined GCP roles to ${GCP_OVERMIND_SA} on the ${PARENT_TYPE} ${PARENT_ID}:"
echo ""

for ROLE in "${ROLES[@]}"; do
    echo "  - ${ROLE}"
done

if [ "${PARENT_TYPE}" = "project" ]; then
    for ROLE in "${PROJECT_ONLY_ROLES[@]}"; do
        echo "  - ${ROLE} (project-level only)"
    done
    echo "  - projects/${PARENT_ID}/roles/overmindCustomRole (custom role with additional BigQuery and Spanner permissions)"
fi

echo ""
echo "These permissions are read-only and allow Overmind to inspect your GCP resources without making any changes."
echo ""

# Ask for confirmation
read -p "Do you want to continue? (Yes/No): " CONFIRMATION
if [[ ! "$(echo "$CONFIRMATION" | tr '[:upper:]' '[:lower:]')" =~ ^(yes|y)$ ]]; then
    echo "Operation canceled by user."
    exit 0
fi

# Counter for successful operations
SUCCESS_COUNT=0
TOTAL_ROLES=${#ROLES[@]}

echo ""
echo "Starting to add IAM policy bindings..."
echo "----------------------------------------"

# Loop through each role and add the policy binding
for ROLE in "${ROLES[@]}"; do
    echo "Adding role: ${ROLE}"

    # Determine the correct command based on parent type
    if [ "${PARENT_TYPE}" = "organization" ]; then
        CMD="gcloud organizations add-iam-policy-binding ${PARENT_ID}"
    elif [ "${PARENT_TYPE}" = "folder" ]; then
        CMD="gcloud resource-manager folders add-iam-policy-binding ${PARENT_ID}"
    else
        CMD="gcloud projects add-iam-policy-binding ${PARENT_ID}"
    fi

    if ${CMD} \
        --member="serviceAccount:${GCP_OVERMIND_SA}" \
        --role="${ROLE}" \
        --quiet > /dev/null 2>&1; then
        echo "✓ Successfully added role: ${ROLE}"
        ((SUCCESS_COUNT++)) || true
    else
        echo "✗ Failed to add role: ${ROLE}"
        # Print the error output
        ${CMD} \
            --member="serviceAccount:${GCP_OVERMIND_SA}" \
            --role="${ROLE}" \
            --quiet
        exit 1
    fi
done

# Add project-only roles if parent is a project
if [ "${PARENT_TYPE}" = "project" ]; then
    echo "Adding project-level-only IAM roles..."
    for ROLE in "${PROJECT_ONLY_ROLES[@]}"; do
        echo "Adding role: ${ROLE}"

        if gcloud projects add-iam-policy-binding "${PARENT_ID}" \
            --member="serviceAccount:${GCP_OVERMIND_SA}" \
            --role="${ROLE}" \
            --quiet > /dev/null 2>&1; then
            echo "✓ Successfully added role: ${ROLE}"
            ((SUCCESS_COUNT++)) || true
            ((TOTAL_ROLES++)) || true
        else
            echo "✗ Failed to add role: ${ROLE}"
            # Print the error output
            gcloud projects add-iam-policy-binding "${PARENT_ID}" \
                --member="serviceAccount:${GCP_OVERMIND_SA}" \
                --role="${ROLE}" \
                --quiet
            exit 1
        fi
    done

    # Add custom role only for project-level parents
    echo "Adding custom role: projects/${PARENT_ID}/roles/overmindCustomRole"
    if gcloud projects add-iam-policy-binding "${PARENT_ID}" \
        --member="serviceAccount:${GCP_OVERMIND_SA}" \
        --role="projects/${PARENT_ID}/roles/overmindCustomRole" \
        --quiet > /dev/null 2>&1; then
        echo "✓ Successfully added custom role"
        ((SUCCESS_COUNT++)) || true
        ((TOTAL_ROLES++)) || true
    else
        echo "✗ Failed to add custom role"
        exit 1
    fi
fi

echo "----------------------------------------"
echo "✓ All IAM policy bindings completed successfully!"
echo "✓ Added ${SUCCESS_COUNT}/${TOTAL_ROLES} roles to service account: ${GCP_OVERMIND_SA}"
echo "✓ Parent: ${GCP_PARENT}"
echo ""
echo "These variables have also been saved to ./.gcp-source-setup-env for other scripts to use."
echo "You can use these variables in subsequent commands."
