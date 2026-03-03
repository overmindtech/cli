---
title: GCP Sql Admin Backup
sidebar_label: gcp-sql-admin-backup
---

A **Cloud SQL backup** represents a point-in-time copy of the data stored in a Cloud SQL instance. Backups are created automatically on a schedule you define or manually on demand, and are retained in Google-managed Cloud Storage where they can later be used to restore the originating instance or clone a new one. Backups may be encrypted either with Google-managed keys or with a customer-managed encryption key (CMEK) from Cloud KMS.  
See the official documentation for details: https://cloud.google.com/sql/docs/mysql/backup-recovery/backups

## Supported Methods

- `GET`: Get a gcp-sql-admin-backup by its "name"
- `LIST`: List all gcp-sql-admin-backup
- ~~`SEARCH`~~

## Possible Links

### [`gcp-cloud-kms-crypto-key`](/sources/gcp/Types/gcp-cloud-kms-crypto-key)

If CMEK encryption is enabled for the Cloud SQL instance, the backup is encrypted with a specific Cloud KMS CryptoKey. This link shows which key secures the backup data at rest.

### [`gcp-cloud-kms-crypto-key-version`](/sources/gcp/Types/gcp-cloud-kms-crypto-key-version)

The actual ciphertext is tied to a particular CryptoKey **version**. Linking to the key version lets you see exactly which rotation of the key was used when the backup was taken.

### [`gcp-compute-network`](/sources/gcp/Types/gcp-compute-network)

Although backups are stored out-of-band, they are associated with the same VPC network(s) as the Cloud SQL instance that produced them. This link helps trace network-level access policies that apply when a backup is restored to an instance using private IP.

### [`gcp-sql-admin-instance`](/sources/gcp/Types/gcp-sql-admin-instance)

Every backup is generated from, and can be restored to, a specific Cloud SQL instance. This link identifies the parent instance, allowing you to evaluate how instance configuration (e.g. region, database version) affects backup usability and risk.
