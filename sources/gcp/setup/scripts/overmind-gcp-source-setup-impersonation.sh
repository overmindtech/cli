#!/bin/bash

# Script to add IAM policy bindings to a service account in GCP
# Takes GCP Parent (organizations/123, folders/456, or projects/my-project), Overmind service account and Impersonation service account as arguments
#
# Usage: ./overmind-gcp-source-setup-impersonation.sh <parent> <overmind-service-account-email> <impersonation-service-account-email>
#
# NOTE: The service accounts should be the service account emails
# presented in the Overmind application when creating a new GCP source.

set -euo pipefail  # Exit on error, undefined vars, and pipe failures

# Check if all arguments are provided
if [[ $# -ne 3 ]]; then
    echo "ERROR: All of the following arguments are required: parent, overmind service account email and impersonation service account email"
    echo "Usage: $0 <parent> <overmind-service-account-email> <impersonation-service-account-email>"
    echo "Parent format: organizations/123, folders/456, or projects/my-project"
    exit 1
fi

# Get arguments
GCP_PARENT="$1"
GCP_OVERMIND_SA="$2"
GCP_IMPERSONATION_SA="$3"

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

# Check if GCP_IMPERSONATION_SA is empty
if [[ -z "${GCP_IMPERSONATION_SA}" ]]; then
    echo "ERROR: Impersonation service account email cannot be empty"
    echo "NOTE: Use the service account email presented in the Impersonation application when creating a GCP source"
    exit 1
fi

# Grant the necessary permissions to the Overmind Service Account to access the resources in the parent
source "$(dirname "$0")/overmind-gcp-source-setup.sh" "${GCP_PARENT}" "${GCP_OVERMIND_SA}"

echo "Impersonation Service Account: ${GCP_IMPERSONATION_SA}"

# Extract project ID from impersonation service account email for the impersonation binding
if [[ "${GCP_IMPERSONATION_SA}" =~ @([^.]+)\.iam\.gserviceaccount\.com$ ]]; then
    IMPERSONATION_PROJECT="${BASH_REMATCH[1]}"
else
    echo "✗ Failed to extract project from impersonation service account email"
    exit 1
fi

# Grant the necessary permissions to allow Overmind SA to impersonate your SA
if gcloud iam service-accounts add-iam-policy-binding \
    "${GCP_IMPERSONATION_SA}" \
    --project "${IMPERSONATION_PROJECT}" \
    --member="serviceAccount:${GCP_OVERMIND_SA}" \
    --role="roles/iam.serviceAccountTokenCreator" \
    --quiet > /dev/null 2>&1; then
    echo "✓ Successfully granted roles/iam.serviceAccountTokenCreator to allow Overmind SA to impersonate: ${GCP_IMPERSONATION_SA}"
else
    echo "✗ Failed to grant roles/iam.serviceAccountTokenCreator"
    # Print the error output
    gcloud iam service-accounts add-iam-policy-binding \
        "${GCP_IMPERSONATION_SA}" \
        --project "${IMPERSONATION_PROJECT}" \
        --member="serviceAccount:${GCP_OVERMIND_SA}" \
        --role="roles/iam.serviceAccountTokenCreator" \
        --quiet
    exit 1
fi

# Save the variables to a local file for other scripts to use. This needs to be done after the source setup script is run to ensure the target file is not overwritten.
echo "export GCP_IMPERSONATION_SA=\"${GCP_IMPERSONATION_SA}\"" >> ./.gcp-source-setup-env
