---
title: GCP Secret Manager Secret
sidebar_label: gcp-secret-manager-secret
---

A Secret in Google Cloud Secret Manager is a secure, version-controlled container for sensitive data such as passwords, API keys, certificates, or any arbitrary text or binary payload. Each Secret holds one or more Secret Versions, allowing you to rotate or roll back the underlying data without changing the resource identifier that your applications refer to. Secrets are encrypted at rest with Google-managed keys by default, or you can supply a customer-managed Cloud KMS key. You can also configure Pub/Sub notifications to be emitted whenever a new version is added or other lifecycle events occur.  
For full details see the official documentation: https://cloud.google.com/secret-manager/docs

**Terrafrom Mappings:**

* `google_secret_manager_secret.secret_id`

## Supported Methods

* `GET`: Get a gcp-secret-manager-secret by its "name"
* `LIST`: List all gcp-secret-manager-secret
* ~~`SEARCH`~~

## Possible Links

### [`gcp-cloud-kms-crypto-key`](/sources/gcp/Types/gcp-cloud-kms-crypto-key)

If a Secret is configured to use customer-managed encryption (CMEK), it references a Cloud KMS Crypto Key that performs the envelope encryption of all Secret Versions. Compromise or mis-configuration of the referenced KMS key directly affects the confidentiality and availability of the Secret’s payloads.

### [`gcp-pub-sub-topic`](/sources/gcp/Types/gcp-pub-sub-topic)

Secret Manager can publish events—such as the creation of a new Secret Version—to a Pub/Sub topic. This enables automated workflows like triggering Cloud Functions for secret rotation or auditing. The Secret therefore holds an optional link to any Pub/Sub topic configured for such notifications.
