---
title: GCP Sql Admin Instance
sidebar_label: gcp-sql-admin-instance
---

A Google Cloud SQL Admin Instance represents a fully-managed relational database instance running on Google Cloud. It encapsulates the configuration for engines such as MySQL, PostgreSQL, or SQL Server, including CPU and memory sizing, version, storage, networking and encryption settings. For full details see the official documentation: https://cloud.google.com/sql/docs/introduction.

**Terrafrom Mappings:**

- `google_sql_database_instance.name`

## Supported Methods

- `GET`: Get a gcp-sql-admin-instance by its "name"
- `LIST`: List all gcp-sql-admin-instance
- ~~`SEARCH`~~

## Possible Links

### [`gcp-cloud-kms-crypto-key`](/sources/gcp/Types/gcp-cloud-kms-crypto-key)

Linked when the instance is encrypted with a Customer-Managed Encryption Key (CMEK); the instance stores the resource ID of the Cloud KMS crypto key it uses for data-at-rest encryption.

### [`gcp-compute-network`](/sources/gcp/Types/gcp-compute-network)

Appears when the instance is configured with a private IP address. The instance is reachable through a Private Service Connection residing inside a specific VPC network.

### [`gcp-compute-subnetwork`](/sources/gcp/Types/gcp-compute-subnetwork)

If private IP is enabled, the instance is bound to a particular subnetwork from which it obtains its internal IP and through which it exposes its endpoints.

### [`gcp-iam-service-account`](/sources/gcp/Types/gcp-iam-service-account)

Cloud SQL creates or uses a service account to perform administrative actions such as backup, replication and interaction with other Google Cloud services; this link surfaces that service account.

### [`gcp-sql-admin-backup-run`](/sources/gcp/Types/gcp-sql-admin-backup-run)

Each successful or scheduled backup run is a child of an instance. The link shows all backup-run resources that belong to the current database instance.

### [`gcp-sql-admin-instance`](/sources/gcp/Types/gcp-sql-admin-instance)

An instance can reference another instance as its read replica or as the source for cloning. This self-link captures those primary/replica relationships.

### [`gcp-storage-bucket`](/sources/gcp/Types/gcp-storage-bucket)

Imports, exports and point-in-time backups can read from or write to Cloud Storage. The instance therefore maintains references to buckets used for these operations.
