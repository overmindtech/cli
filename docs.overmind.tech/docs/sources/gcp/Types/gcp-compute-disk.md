---
title: GCP Compute Disk
sidebar_label: gcp-compute-disk
---

A GCP Compute Disk—formally known as a Persistent Disk—is block-level storage that can be attached to Google Compute Engine virtual machine (VM) instances. Disks may be zonal or regional, support features such as snapshots, replication, and Customer-Managed Encryption Keys (CMEK), and can be resized or detached without data loss. Official documentation: https://cloud.google.com/compute/docs/disks

**Terrafrom Mappings:**

  * `google_compute_disk.name`

## Supported Methods

* `GET`: Get GCP Compute Disk by "gcp-compute-disk-name"
* `LIST`: List all GCP Compute Disk items
* ~~`SEARCH`~~

## Possible Links

### [`gcp-cloud-kms-crypto-key-version`](/sources/gcp/Types/gcp-cloud-kms-crypto-key-version)

Indicates the specific Cloud KMS key version used when the disk is encrypted with a customer-managed encryption key.

### [`gcp-compute-disk`](/sources/gcp/Types/gcp-compute-disk)

For regional or replicated disks, the resource records the relationship to its source or replica peer disk.

### [`gcp-compute-image`](/sources/gcp/Types/gcp-compute-image)

Shows the image from which the disk was created, or images that have been built from this disk.

### [`gcp-compute-instance`](/sources/gcp/Types/gcp-compute-instance)

Lists the VM instances to which the disk is currently attached or has been attached historically.

### [`gcp-compute-instant-snapshot`](/sources/gcp/Types/gcp-compute-instant-snapshot)

Captures the association between the disk and any instant snapshots taken for rapid backup or restore operations.

### [`gcp-compute-snapshot`](/sources/gcp/Types/gcp-compute-snapshot)

Represents traditional snapshots for the disk, enabling point-in-time recovery or disk cloning.

### [`gcp-storage-bucket`](/sources/gcp/Types/gcp-storage-bucket)

If disk snapshots or images are exported to Cloud Storage, this link records the destination bucket holding those exports.