---
title: GCP Dataform Repository
sidebar_label: gcp-dataform-repository
---

A GCP Dataform Repository is the top-level, version-controlled container that stores all the SQL workflow code, configuration files and commit history used by Dataform in Google Cloud. It functions much like a Git repository, allowing data teams to develop, test and deploy BigQuery pipelines through branches, pull requests and releases. Repositories live under a specific project and location and can be connected to Cloud Source Repositories or external Git providers.  
Official documentation: https://cloud.google.com/dataform/docs/repositories

**Terrafrom Mappings:**

- `google_dataform_repository.id`

## Supported Methods

- `GET`: Get a gcp-dataform-repository by its "locations|repositories"
- ~~`LIST`~~
- `SEARCH`: Search for Dataform repositories in a location. Use the format "location" or "projects/[project_id]/locations/[location]/repositories/[repository_name]" which is supported for terraform mappings.

## Possible Links

### [`gcp-cloud-kms-crypto-key`](/sources/gcp/Types/gcp-cloud-kms-crypto-key)

If Customer-Managed Encryption Keys (CMEK) are enabled for the repository, it contains a reference to the Cloud KMS crypto key that encrypts its metadata. Overmind follows this link to verify key existence, rotation policy and wider blast radius.

### [`gcp-iam-service-account`](/sources/gcp/Types/gcp-iam-service-account)

Dataform executes queries and workflow steps using a service account specified in the repository or workspace settings. Linking to the IAM service account lets Overmind trace which identities can act on behalf of the repository and assess permission risks.

### [`gcp-secret-manager-secret`](/sources/gcp/Types/gcp-secret-manager-secret)

A repository may reference secrets (such as connection strings or API tokens) stored in Secret Manager via environment variables or workflow configurations. Overmind links to these secrets to ensure they exist, are properly protected and are not about to be rotated or deleted.
