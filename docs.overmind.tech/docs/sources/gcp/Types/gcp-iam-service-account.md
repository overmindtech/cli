---
title: GCP Iam Service Account
sidebar_label: gcp-iam-service-account
---

A GCP IAM Service Account is a special kind of Google identity that an application or VM instance uses to make authorised calls to Google Cloud APIs, rather than an end-user. Each service account is identified by an email‐style string (e.g. `my-sa@project-id.iam.gserviceaccount.com`) and a stable numeric `unique_id`. Service accounts can be granted IAM roles, can own resources, and may have one or more cryptographic keys used for authentication.  
For full details see the official documentation: https://cloud.google.com/iam/docs/service-accounts

**Terrafrom Mappings:**

* `google_service_account.email`
* `google_service_account.unique_id`

## Supported Methods

* `GET`: Get GCP Iam Service Account by "gcp-iam-service-account-email or unique_id"
* `LIST`: List all GCP Iam Service Account items
* ~~`SEARCH`~~

## Possible Links

### [`gcp-cloud-resource-manager-project`](/sources/gcp/Types/gcp-cloud-resource-manager-project)

Every service account is created inside a single Cloud Resource Manager project. This link lets you navigate from the service account to the project that owns it, revealing project-level policies and context.

### [`gcp-iam-service-account-key`](/sources/gcp/Types/gcp-iam-service-account-key)

Service account keys are cryptographic credentials associated with a service account. This link lists all keys (active, disabled or expired) that belong to the current service account, allowing you to audit key rotation and exposure risks.
