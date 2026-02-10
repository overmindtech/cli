---
title: GCP Sql Admin Instance
sidebar_label: gcp-sql-admin-instance
---

A GCP SQL Admin Instance represents a managed Cloud SQL database instance in Google Cloud Platform. It encapsulates the configuration of the database engine (MySQL, PostgreSQL or SQL Server), machine tier, storage, high-availability settings, networking and encryption options. The resource is managed through the Cloud SQL Admin API, which is documented here: https://cloud.google.com/sql/docs/mysql/admin-api/. Creating or modifying an instance via Terraform, the Cloud Console or gcloud ultimately results in API calls against this object.

**Terrafrom Mappings:**

- `google_sql_database_instance.name`

## Supported Methods

- `GET`: Get a gcp-sql-admin-instance by its "name"
- `LIST`: List all gcp-sql-admin-instance
- ~~`SEARCH`~~

## Possible Links

### [`gcp-cloud-kms-crypto-key`](/sources/gcp/Types/gcp-cloud-kms-crypto-key)

If Customer-Managed Encryption Keys (CMEK) are enabled for the instance, the instance is encrypted with a specific Cloud KMS Crypto Key. Overmind links the instance to the `gcp-cloud-kms-crypto-key` that provides its disk-level encryption key.

### [`gcp-compute-network`](/sources/gcp/Types/gcp-compute-network)

When an instance is configured for private IP or has authorised networks for public IP access, it attaches to one or more VPC networks. Overmind therefore links the instance to the `gcp-compute-network` resources that define those VPCs.

### [`gcp-iam-service-account`](/sources/gcp/Types/gcp-iam-service-account)

Cloud SQL automatically creates or uses service accounts to perform backups, replication and other administrative tasks. The instance is linked to the `gcp-iam-service-account` identities that act on its behalf, allowing you to trace permissions and potential privilege escalation paths.

### [`gcp-sql-admin-backup-run`](/sources/gcp/Types/gcp-sql-admin-backup-run)

Each automated or on-demand backup of an instance is represented by a Backup Run resource. Overmind links every `gcp-sql-admin-backup-run` to the parent instance so you can see the full backup history and retention compliance.

### [`gcp-sql-admin-instance`](/sources/gcp/Types/gcp-sql-admin-instance)

Instances may reference other instances when configured for read replicas, high-availability failover or cloning. Overmind links an instance to any peer `gcp-sql-admin-instance` that serves as its primary, replica or clone source/target.

### [`gcp-storage-bucket`](/sources/gcp/Types/gcp-storage-bucket)

Cloud SQL supports import/export of SQL dump files and automatic log exports to Cloud Storage. The instance is linked to any `gcp-storage-bucket` that it reads from or writes to during these operations, revealing data-exfiltration or retention risks.
