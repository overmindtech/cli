---
title: GCP Cloud Kms Key Ring
sidebar_label: gcp-cloud-kms-key-ring
---

A **Cloud KMS Key Ring** is a top-level container within Google Cloud KMS that groups one or more CryptoKeys in a specific GCP location (region). It acts as both an organisational unit and an IAM boundary: all CryptoKeys inside a Key Ring inherit the same location and share the same access-control policies. Creating a Key Ring is an irreversible, free operation and is a prerequisite for creating any CryptoKeys.  
For full details, see the official documentation: https://cloud.google.com/kms/docs/object-hierarchy#key_rings

**Terrafrom Mappings:**

- `google_kms_key_ring.id`

## Supported Methods

- `GET`: Get GCP Cloud Kms Key Ring by "gcp-cloud-kms-key-ring-location|gcp-cloud-kms-key-ring-name"
- `LIST`: List all GCP Cloud Kms Key Ring items
- `SEARCH`: Search for GCP Cloud Kms Key Ring by "gcp-cloud-kms-key-ring-location"

## Possible Links

### [`gcp-cloud-kms-crypto-key`](/sources/gcp/Types/gcp-cloud-kms-crypto-key)

Each CryptoKey belongs to exactly one Key Ring. Linking a Key Ring to its child `gcp-cloud-kms-crypto-key` items lets Overmind surface all encryption keys that share the same location and IAM policy, making it easier to assess the blast radius of any permission or configuration changes applied to the Key Ring.
