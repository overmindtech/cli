---
title: GCP Cloud Kms Crypto Key
sidebar_label: gcp-cloud-kms-crypto-key
---

A Google Cloud KMS Crypto Key is a logical key resource that performs cryptographic operations such as encryption/de-encryption, signing, and message authentication. Each Crypto Key sits inside a Key Ring, which in turn lives in a specific GCP location (region). The key material for a Crypto Key can be rotated, versioned, and protected by Cloud KMS or by customer-managed hardware security modules, and it is referenced by other Google Cloud services whenever those services need to encrypt or sign data on your behalf.  
Official documentation: https://cloud.google.com/kms/docs/object-hierarchy#key

## Supported Methods

- `GET`: Get GCP Cloud Kms Crypto Key by "gcp-cloud-kms-key-ring-location|gcp-cloud-kms-key-ring-name|gcp-cloud-kms-crypto-key-name"
- ~~`LIST`~~
- `SEARCH`: Search for GCP Cloud Kms Crypto Key by "gcp-cloud-kms-key-ring-location|gcp-cloud-kms-key-ring-name"

## Possible Links

### [`gcp-cloud-kms-key-ring`](/sources/gcp/Types/gcp-cloud-kms-key-ring)

A Crypto Key is always a child resource of a Key Ring. The `gcp-cloud-kms-key-ring` link allows Overmind to trace from the key to its parent container, establishing the hierarchical relationship needed to understand inheritance of IAM policies, location constraints, and aggregated risk.
