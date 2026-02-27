---
title: GCP Cloud Kms Crypto Key Version
sidebar_label: gcp-cloud-kms-crypto-key-version
---

A **Cloud KMS CryptoKeyVersion** is an immutable representation of a single piece of key material managed by Google Cloud Key Management Service. Each CryptoKey can have many versions, allowing you to rotate key material without changing the logical key that your workloads use. A version holds state (e.g., `ENABLED`, `DISABLED`, `DESTROYED`), an algorithm specification (RSA, AES-GCM, etc.), and lifecycle metadata such as creation and destruction timestamps. See the official Google documentation for full details: https://cloud.google.com/kms/docs/reference/rest/v1/projects.locations.keyRings.cryptoKeys.cryptoKeyVersions

**Terrafrom Mappings:**

* `google_kms_crypto_key_version.id`

## Supported Methods

* `GET`: Get GCP Cloud Kms Crypto Key Version by "gcp-cloud-kms-key-ring-location|gcp-cloud-kms-key-ring-name|gcp-cloud-kms-crypto-key-name|gcp-cloud-kms-crypto-key-version-version"
* ~~`LIST`~~
* `SEARCH`: Search for GCP Cloud Kms Crypto Key Version by "gcp-cloud-kms-key-ring-location|gcp-cloud-kms-key-ring-name|gcp-cloud-kms-crypto-key-name"

## Possible Links

### [`gcp-cloud-kms-crypto-key`](/sources/gcp/Types/gcp-cloud-kms-crypto-key)

A CryptoKeyVersion is always a child of a CryptoKey. The `gcp-cloud-kms-crypto-key` resource represents the logical key, while the current item represents a particular version of that key’s material.
