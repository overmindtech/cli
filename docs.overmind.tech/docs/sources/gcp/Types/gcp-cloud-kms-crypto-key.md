---
title: GCP Cloud Kms Crypto Key
sidebar_label: gcp-cloud-kms-crypto-key
---

A **Cloud KMS CryptoKey** is the logical resource in Google Cloud that represents a single cryptographic key and its primary metadata. It defines the algorithm, purpose (encryption/decryption, signing/verification, MAC, etc.), rotation schedule, and IAM policy for the key. Each CryptoKey lives inside a Key Ring, can have multiple immutable versions, and is used by Google-managed services (or your own applications) to perform cryptographic operations.  
Official documentation: https://cloud.google.com/kms/docs/object-hierarchy#key

**Terrafrom Mappings:**

* `google_kms_crypto_key.id`

## Supported Methods

* `GET`: Get GCP Cloud Kms Crypto Key by "gcp-cloud-kms-key-ring-location|gcp-cloud-kms-key-ring-name|gcp-cloud-kms-crypto-key-name"
* ~~`LIST`~~
* `SEARCH`: Search for GCP Cloud Kms Crypto Key by "gcp-cloud-kms-key-ring-location|gcp-cloud-kms-key-ring-name"

## Possible Links

### [`gcp-cloud-kms-crypto-key-version`](/sources/gcp/Types/gcp-cloud-kms-crypto-key-version)

A CryptoKey is the parent of one or more CryptoKeyVersions. Each version contains the actual key material and its own state (enabled, disabled, destroyed, etc.). Overmind links to these versions so you can inspect individual key material lifecycles and detect risks such as disabled or scheduled-for-destruction versions.

### [`gcp-cloud-kms-key-ring`](/sources/gcp/Types/gcp-cloud-kms-key-ring)

Every CryptoKey resides within a Key Ring, which provides a namespace and location boundary. This link shows the Key Ring that owns the CryptoKey, allowing you to trace location-specific compliance requirements or IAM inheritance issues.
