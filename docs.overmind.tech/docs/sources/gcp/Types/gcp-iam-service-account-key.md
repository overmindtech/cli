---
title: GCP Iam Service Account Key
sidebar_label: gcp-iam-service-account-key
---

A GCP IAM Service Account Key is a cryptographic key-pair that allows code or users outside Google Cloud to authenticate as a specific service account. Each key consists of a public key stored by Google and a private key material that can be downloaded once and should be stored securely. Because anyone in possession of the private key can act with all the permissions of the associated service account, these keys are highly sensitive and should be rotated or disabled when no longer required.  
For full details, see the official documentation: https://cloud.google.com/iam/docs/creating-managing-service-account-keys

**Terrafrom Mappings:**

- `google_service_account_key.id`

## Supported Methods

- `GET`: Get GCP Iam Service Account Key by "gcp-iam-service-account-email or unique_id|gcp-iam-service-account-key-name"
- ~~`LIST`~~
- `SEARCH`: Search for GCP Iam Service Account Key by "gcp-iam-service-account-email or unique_id"

## Possible Links

### [`gcp-iam-service-account`](/sources/gcp/Types/gcp-iam-service-account)

Every Service Account Key is attached to exactly one Service Account; this link allows you to trace which principal will be able to use the key and to evaluate the permissions that could be exercised if the key were compromised.
