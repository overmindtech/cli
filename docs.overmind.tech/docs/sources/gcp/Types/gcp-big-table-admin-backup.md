---
title: GCP Big Table Admin Backup
sidebar_label: gcp-big-table-admin-backup
---

A Cloud Bigtable Backup is a point-in-time copy of a Bigtable table that is managed by the Bigtable Admin API. It allows you to protect data against accidental deletion or corruption and to restore the table later, either in the same cluster or in a different one within the same instance. Each backup is stored in a specific cluster, retains the table’s schema and data as they existed at the moment the backup was taken, and can be kept for a user-defined retention period.
Official documentation: https://docs.cloud.google.com/bigtable/docs/backups

## Supported Methods

- `GET`: Get a gcp-big-table-admin-backup by its "instances|clusters|backups"
- ~~`LIST`~~
- `SEARCH`: Search for gcp-big-table-admin-backup by its "instances|clusters"

## Possible Links

### [`gcp-big-table-admin-backup`](/sources/gcp/Types/gcp-big-table-admin-backup)

The current item represents the Backup resource itself, containing metadata such as name, creation time, size, expiration time and the source table it protects.

### [`gcp-big-table-admin-cluster`](/sources/gcp/Types/gcp-big-table-admin-cluster)

Each backup is physically stored in exactly one Bigtable cluster; this link shows the parent cluster that owns and stores the backup.

### [`gcp-big-table-admin-table`](/sources/gcp/Types/gcp-big-table-admin-table)

A backup is created from a specific table; this link identifies that source table and allows you to see which tables can be restored from the backup.
