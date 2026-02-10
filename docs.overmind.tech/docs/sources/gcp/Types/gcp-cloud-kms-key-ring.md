---
title: GCP Cloud Kms Key Ring
sidebar_label: gcp-cloud-kms-key-ring
---

A Cloud KMS Key Ring is a logical container used to group related customer-managed encryption keys within Google Cloud’s Key Management Service (KMS). All Crypto Keys created inside the same Key Ring share the same geographic location, and access control can be applied at the Key Ring level to govern every key it contains. For more information, refer to the [official documentation](https://cloud.google.com/kms/docs/create-key-ring).

**Terrafrom Mappings:**

- `google_kms_key_ring.name`

## Supported Methods

- `GET`: Get GCP Cloud Kms Key Ring by "gcp-cloud-kms-key-ring-location|gcp-cloud-kms-key-ring-name"
- `LIST`: List all GCP Cloud Kms Key Rings across all locations in the project
- `SEARCH`: Search for GCP Cloud Kms Key Ring by "gcp-cloud-kms-key-ring-location"

## Possible Links

### [`gcp-cloud-kms-crypto-key`](/sources/gcp/Types/gcp-cloud-kms-crypto-key)

A Key Ring is the direct parent of one or more Crypto Keys. Every Crypto Key resource must belong to exactly one Key Ring, so Overmind creates this link to allow navigation from the Key Ring to all the keys it contains (and vice-versa), making it easier to assess the full cryptographic surface associated with a given deployment.
