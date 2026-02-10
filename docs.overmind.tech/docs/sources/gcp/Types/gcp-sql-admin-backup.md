---
title: GCP Sql Admin Backup
sidebar_label: gcp-sql-admin-backup
---

A **GCP Sql Admin Backup** represents the backup configuration that protects a Cloud SQL instance.  
The object contains the settings that determine when and how Google Cloud takes automatic or on-demand snapshots of the instance, including the backup window, retention period, and (when Customer-Managed Encryption Keys are used) the CryptoKey that encrypts the resulting files.  
For a detailed description of Cloud SQL backups see the official documentation: https://cloud.google.com/sql/docs/mysql/backup-recovery/backups.

## Supported Methods

- `GET`: Get a gcp-sql-admin-backup by its "name"
- `LIST`: List all gcp-sql-admin-backup
- ~~`SEARCH`~~

## Possible Links

### [`gcp-cloud-kms-crypto-key`](/sources/gcp/Types/gcp-cloud-kms-crypto-key)

If the backup is encrypted with a Customer-Managed Encryption Key (CMEK), Overmind links the backup to the `gcp-cloud-kms-crypto-key` that holds the key material. Analysing this relationship lets you verify that the key exists, is in the correct state, and has the appropriate IAM policy.

### [`gcp-sql-admin-backup-run`](/sources/gcp/Types/gcp-sql-admin-backup-run)

Every time the backup configuration is executed it produces a Backup Run. This link connects the configuration to those individual `gcp-sql-admin-backup-run` objects, allowing you to trace whether recent runs succeeded and to inspect metadata such as the size and status of each run.

### [`gcp-sql-admin-instance`](/sources/gcp/Types/gcp-sql-admin-instance)

The backup configuration belongs to a specific Cloud SQL instance. This link points from the backup resource to the parent `gcp-sql-admin-instance`, helping you understand which database workload the backup protects and enabling dependency traversal from the instance to its safety mechanisms.
