---
title: GCP Compute Image
sidebar_label: gcp-compute-image
---

A Google Cloud Compute Image is a read-only template that contains a boot disk configuration (including the operating system and any installed software) which can be used to create new persistent disks or VM instances. Images may be publicly provided by Google, published by third-party vendors, or built privately within your own project. They support features such as image families, deprecation, and customer-managed encryption keys (CMEK).  
For full details see the official documentation: https://cloud.google.com/compute/docs/images

**Terrafrom Mappings:**

- `google_compute_image.name`

## Supported Methods

- `GET`: Get GCP Compute Image by "gcp-compute-image-name"
- `LIST`: List all GCP Compute Image items
- `SEARCH`: Search for GCP Compute Image by "gcp-compute-image-family"

## Possible Links

### [`gcp-cloud-kms-crypto-key`](/sources/gcp/Types/gcp-cloud-kms-crypto-key)

If the image is protected with a customer-managed encryption key (CMEK), Overmind links the image to the Cloud KMS Crypto Key that encrypts its contents.

### [`gcp-cloud-kms-crypto-key-version`](/sources/gcp/Types/gcp-cloud-kms-crypto-key-version)

When CMEK protection specifies an explicit key version, the image is linked to that exact Crypto Key Version so you can trace roll-overs or revocations that might affect instance bootability.

### [`gcp-compute-disk`](/sources/gcp/Types/gcp-compute-disk)

Images can be created from existing persistent disks, and new disks can be created from an image. Overmind therefore links images to the disks that serve as their source or to the disks that have been instantiated from them.

### [`gcp-compute-image`](/sources/gcp/Types/gcp-compute-image)

Images belonging to the same image family or derived from one another (for example, when rolling a new version) are cross-linked so you can understand upgrade paths and deprecations within a family.

### [`gcp-compute-snapshot`](/sources/gcp/Types/gcp-compute-snapshot)

An image may be built from one or more snapshots of a disk, and snapshots can be exported from an image. Overmind links images to the snapshots that contributed to, or were generated from, them.

### [`gcp-iam-service-account`](/sources/gcp/Types/gcp-iam-service-account)

Access to create, deprecate or use an image is controlled through IAM roles. Overmind shows the service accounts that have permissions on the image, helping you assess who can launch VMs from it.

### [`gcp-storage-bucket`](/sources/gcp/Types/gcp-storage-bucket)

During import or export operations, raw disk files are stored in Cloud Storage. Overmind links an image to the Storage Buckets that hosted its source or export objects, enabling you to trace data residency and clean-up unused artefacts.
