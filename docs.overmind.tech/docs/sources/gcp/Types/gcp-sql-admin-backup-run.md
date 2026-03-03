---
title: GCP Sql Admin Backup Run
sidebar_label: gcp-sql-admin-backup-run
---

A **Cloud SQL Backup Run** represents a single on-demand or automated backup operation for a Cloud SQL instance. It records when the backup was initiated, its status, size, location, encryption information and other metadata. Backup runs allow administrators to restore an instance to a previous state or to clone data into a new instance.  
Official documentation: https://cloud.google.com/sql/docs/mysql/admin-api/rest/v1/backupRuns

## Supported Methods

- `GET`: Get a gcp-sql-admin-backup-run by its "instances|backupRuns"
- ~~`LIST`~~
- `SEARCH`: Search for gcp-sql-admin-backup-run by its "instances"

## Possible Links

### [`gcp-cloud-kms-crypto-key`](/sources/gcp/Types/gcp-cloud-kms-crypto-key)

If Customer-Managed Encryption Keys (CMEK) are enabled for the instance, the backup run is encrypted with a Cloud KMS Crypto Key. This link points to the parent key that protects the specific key version used for the backup.

### [`gcp-cloud-kms-crypto-key-version`](/sources/gcp/Types/gcp-cloud-kms-crypto-key-version)

The `encryptionInfo` block inside the backup run references the exact Cloud KMS Crypto Key Version that encrypted the backup file. This relationship lets you trace which key version must be available to decrypt or restore the backup.

### [`gcp-sql-admin-instance`](/sources/gcp/Types/gcp-sql-admin-instance)

Every backup run belongs to a single Cloud SQL instance. This link connects the backup run to its parent instance so you can see which database the backup protects and assess the impact of restoring or deleting it.
