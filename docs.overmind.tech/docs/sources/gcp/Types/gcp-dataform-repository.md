---
title: GCP Dataform Repository
sidebar_label: gcp-dataform-repository
---

A Google Cloud Dataform Repository represents the source-controlled codebase that defines your Dataform workflows. It stores SQLX files, declarations and configuration that Dataform uses to build, test and deploy transformations in BigQuery. A repository can point to an internal workspace or to an external Git repository and may reference service accounts, Secret Manager secrets and customer-managed encryption keys.  
Official documentation: https://cloud.google.com/dataform/reference/rest

**Terrafrom Mappings:**

  * `google_dataform_repository.id`

## Supported Methods

* `GET`: Get a gcp-dataform-repository by its "locations|repositories"
* ~~`LIST`~~
* `SEARCH`: Search for Dataform repositories in a location. Use the format "location" or "projects/[project_id]/locations/[location]/repositories/[repository_name]" which is supported for terraform mappings.

## Possible Links

### [`gcp-cloud-kms-crypto-key`](/sources/gcp/Types/gcp-cloud-kms-crypto-key)

A repository can be configured with a customer-managed encryption key (`kms_key_name`) to encrypt its metadata and compiled artefacts, creating a dependency on the corresponding Cloud KMS crypto-key.

### [`gcp-cloud-kms-crypto-key-version`](/sources/gcp/Types/gcp-cloud-kms-crypto-key-version)

If CMEK is enabled, the repository points to a specific crypto-key version that is actually used for encryption; rotating or disabling that version will affect the repository.

### [`gcp-iam-service-account`](/sources/gcp/Types/gcp-iam-service-account)

Dataform uses a service account to fetch code from remote Git repositories and to execute compilation and workflow tasks; the repository stores the e-mail address of that service account.

### [`gcp-secret-manager-secret`](/sources/gcp/Types/gcp-secret-manager-secret)

When a repository is linked to an external Git provider, the authentication token is stored in Secret Manager. The field `authentication_token_secret_version` references the secret (and version) that holds the token.