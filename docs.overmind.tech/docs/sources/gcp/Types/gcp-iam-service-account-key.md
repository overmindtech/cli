---
title: GCP Iam Service Account Key
sidebar_label: gcp-iam-service-account-key
---

A GCP IAM Service Account Key is a cryptographic key-pair (private and public) that is bound to a specific IAM service account. Possessing the private half of the key allows a workload or user to authenticate to Google Cloud APIs as that service account, making the key one of the most sensitive objects in any Google Cloud environment. Keys can be user-managed or Google-managed, rotated, disabled or deleted, and each service account can hold up to ten user-managed keys at a time. Mis-management of these keys can lead to credential leakage and unauthorised access.  
Official documentation: https://cloud.google.com/iam/docs/creating-managing-service-account-keys

**Terrafrom Mappings:**

* `google_service_account_key.id`

## Supported Methods

* `GET`: Get GCP Iam Service Account Key by "gcp-iam-service-account-email or unique_id|gcp-iam-service-account-key-name"
* ~~`LIST`~~
* `SEARCH`: Search for GCP Iam Service Account Key by "gcp-iam-service-account-email or unique_id"

## Possible Links

### [`gcp-iam-service-account`](/sources/gcp/Types/gcp-iam-service-account)

A Service Account Key is always subordinate to, and uniquely associated with, a single IAM service account. Overmind links the key back to its parent service account so you can trace which workload the key belongs to, understand the permissions it inherits, and assess the blast radius should the key be compromised.
