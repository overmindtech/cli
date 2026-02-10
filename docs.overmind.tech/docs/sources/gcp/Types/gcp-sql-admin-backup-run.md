---
title: GCP Sql Admin Backup Run
sidebar_label: gcp-sql-admin-backup-run
---

A GCP SQL Admin Backup Run represents an individual on-demand or automatically-scheduled backup created for a Cloud SQL instance. Each backup run records metadata such as its status, start and end times, location, encryption information and size. Backup runs are managed through the Cloud SQL Admin API and can be listed, retrieved or deleted by project administrators. For full details see Google’s documentation: https://cloud.google.com/sql/docs/mysql/admin-api/rest/v1/backupRuns

## Supported Methods

- `GET`: Get a gcp-sql-admin-backup-run by its "instances|backupRuns"
- ~~`LIST`~~
- `SEARCH`: Search for gcp-sql-admin-backup-run by its "instances"

## Possible Links

### [`gcp-cloud-kms-crypto-key`](/sources/gcp/Types/gcp-cloud-kms-crypto-key)

If a Cloud SQL instance is configured with customer-managed encryption keys (CMEK), the backup run is encrypted with the specified KMS CryptoKey. The backup run therefore references the CryptoKey used for encryption.

### [`gcp-sql-admin-instance`](/sources/gcp/Types/gcp-sql-admin-instance)

Every backup run belongs to exactly one Cloud SQL instance; the instance is the parent resource under which the backup run is created.
