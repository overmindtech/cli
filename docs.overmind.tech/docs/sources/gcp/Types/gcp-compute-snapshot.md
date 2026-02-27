---
title: GCP Compute Snapshot
sidebar_label: gcp-compute-snapshot
---

A **GCP Compute Snapshot** is a point-in-time, incremental backup of a Compute Engine persistent or regional disk. Snapshots can be stored in multiple regions, encrypted with customer-managed keys, and used to create new disks, thereby providing a simple mechanism for backup, disaster recovery and environment cloning.  
Official documentation: https://cloud.google.com/compute/docs/disks/create-snapshots

**Terrafrom Mappings:**

* `google_compute_snapshot.name`

## Supported Methods

* `GET`: Get GCP Compute Snapshot by "gcp-compute-snapshot-name"
* `LIST`: List all GCP Compute Snapshot items
* ~~`SEARCH`~~

## Possible Links

### [`gcp-cloud-kms-crypto-key-version`](/sources/gcp/Types/gcp-cloud-kms-crypto-key-version)

If the snapshot is encrypted with a customer-managed encryption key (CMEK), it references the specific Cloud KMS CryptoKeyVersion that holds the key material. Overmind links the snapshot to that key version so you can trace encryption dependencies and confirm key rotation policies.

### [`gcp-compute-disk`](/sources/gcp/Types/gcp-compute-disk)

Every snapshot originates from a source disk. This link shows which Compute Engine disk (zonal or regional) was used to create the snapshot, letting you assess blast radius and recovery workflows.

### [`gcp-compute-instant-snapshot`](/sources/gcp/Types/gcp-compute-instant-snapshot)

An instant snapshot is a fast, crash-consistent capture that can later be converted into a regular snapshot. When such a conversion occurs, Overmind links the resulting standard snapshot to its originating instant snapshot, giving visibility into the lineage of your backups.
