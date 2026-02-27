---
title: GCP Big Table Admin Backup
sidebar_label: gcp-big-table-admin-backup
---

A Cloud Bigtable Admin Backup represents a point-in-time copy of a single Bigtable table that is stored within the same Bigtable cluster for a user-defined retention period. Back-ups allow you to restore data that has been deleted or corrupted without replaying your entire write history, and they can also be copied to other regions for disaster-recovery purposes. The resource is created, managed and deleted through the Cloud Bigtable Admin API.  
Official documentation: https://cloud.google.com/bigtable/docs/backups

## Supported Methods

* `GET`: Get a gcp-big-table-admin-backup by its "instances|clusters|backups"
* ~~`LIST`~~
* `SEARCH`: Search for gcp-big-table-admin-backup by its "instances|clusters"

## Possible Links

### [`gcp-big-table-admin-backup`](/sources/gcp/Types/gcp-big-table-admin-backup)

If the current backup is used as the source for a cross-cluster copy, or if multiple back-ups are chained through copy operations, Overmind links the related `gcp-big-table-admin-backup` resources together so you can trace provenance and inheritance of data.

### [`gcp-big-table-admin-cluster`](/sources/gcp/Types/gcp-big-table-admin-cluster)

Every backup is physically stored in the Bigtable cluster where it was created. The backup therefore links to its parent `gcp-big-table-admin-cluster`, enabling you to understand locality, storage costs and the failure domain that may affect both the cluster and its back-ups.

### [`gcp-big-table-admin-table`](/sources/gcp/Types/gcp-big-table-admin-table)

A backup is a snapshot of a specific Bigtable table at the moment the backup was taken. This link points back to that source `gcp-big-table-admin-table`, allowing you to see which dataset the backup protects and to assess the impact of schema or data changes.

### [`gcp-cloud-kms-crypto-key-version`](/sources/gcp/Types/gcp-cloud-kms-crypto-key-version)

When customer-managed encryption (CMEK) is enabled, the backup’s data is encrypted with a particular Cloud KMS key version. Linking to `gcp-cloud-kms-crypto-key-version` lets you audit encryption lineage and verify that the correct key material is being used for protecting the backup.
