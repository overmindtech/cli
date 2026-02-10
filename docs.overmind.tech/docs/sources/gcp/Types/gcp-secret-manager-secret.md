---
title: GCP Secret Manager Secret
sidebar_label: gcp-secret-manager-secret
---

A Google Cloud Secret Manager Secret is the logical container for sensitive data such as API keys, passwords and certificates stored in Secret Manager. The secret resource defines metadata and access-control policies, while one or more numbered “versions” hold the actual payload, enabling safe rotation and roll-back. Secrets are encrypted at rest with Google-managed keys by default, or with a user-supplied Cloud KMS key, and access is governed through IAM. For further information see the official documentation: https://cloud.google.com/secret-manager/docs

**Terrafrom Mappings:**

- `google_secret_manager_secret.secret_id`

## Supported Methods

- `GET`: Get a gcp-secret-manager-secret by its "name"
- `LIST`: List all gcp-secret-manager-secret
- ~~`SEARCH`~~

## Possible Links

### [`gcp-cloud-kms-crypto-key`](/sources/gcp/Types/gcp-cloud-kms-crypto-key)

If a customer-managed encryption key (CMEK) has been configured for this secret, the secret’s `kms_key_name` field will reference a Cloud KMS Crypto Key. Overmind surfaces that link so that you can trace how the secret is encrypted and assess key-management risks.

### [`gcp-pub-sub-topic`](/sources/gcp/Types/gcp-pub-sub-topic)

Secret Manager can be set to publish notifications (e.g. when a new secret version is added or destroyed) to a Pub/Sub topic. When such a notification configuration exists, the secret will link to the relevant Pub/Sub topic, allowing you to review who can subscribe to, or forward, these events.
