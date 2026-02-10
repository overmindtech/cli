---
title: GCP Iam Service Account
sidebar_label: gcp-iam-service-account
---

A GCP IAM Service Account is a non-human identity that represents a workload such as a VM, Cloud Function or CI/CD pipeline. It can be granted IAM roles and used to obtain access tokens for calling Google Cloud APIs, allowing software to authenticate securely without relying on end-user credentials. Each service account lives inside a single project (or, less commonly, an organisation or folder) and can be equipped with one or more private keys for external use. See the official documentation for further details: [Google Cloud – Service Accounts](https://cloud.google.com/iam/docs/service-accounts).

**Terrafrom Mappings:**

- `google_service_account.email`
- `google_service_account.unique_id`

## Supported Methods

- `GET`: Get GCP Iam Service Account by "gcp-iam-service-account-email or unique_id"
- `LIST`: List all GCP Iam Service Account items
- ~~`SEARCH`~~

## Possible Links

### [`gcp-cloud-resource-manager-project`](/sources/gcp/Types/gcp-cloud-resource-manager-project)

Every service account is created within exactly one Cloud Resource Manager project. Overmind links the service account to its parent project so that you can trace inheritance of IAM policies and understand the blast radius of changes to either resource.

### [`gcp-iam-service-account-key`](/sources/gcp/Types/gcp-iam-service-account-key)

A service account may have multiple keys (managed by Google or user-managed). These keys allow external systems to impersonate the service account. Overmind enumerates and links all keys associated with a service account, helping you identify stale or over-privileged credentials.
