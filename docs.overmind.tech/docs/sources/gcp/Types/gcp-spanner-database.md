---
title: GCP Spanner Database
sidebar_label: gcp-spanner-database
---

A GCP Spanner Database is a logically isolated collection of relational data that lives inside a Cloud Spanner instance. It contains the schema (tables, indexes, views) and the data itself, and it inherits the instance’s compute and storage resources. Cloud Spanner provides global consistency, horizontal scalability and automatic replication, making the database suitable for mission-critical, globally distributed workloads. Official documentation: https://cloud.google.com/spanner/docs

**Terrafrom Mappings:**

- `google_spanner_database.name`

## Supported Methods

- `GET`: Get a gcp-spanner-database by its "instances|databases"
- ~~`LIST`~~
- `SEARCH`: Search for gcp-spanner-database by its "instances"

## Possible Links

### [`gcp-cloud-kms-crypto-key`](/sources/gcp/Types/gcp-cloud-kms-crypto-key)

A Spanner database can be encrypted with a customer-managed encryption key (CMEK) stored in Cloud KMS. Overmind links the database to the KMS Crypto Key that protects its data at rest.

### [`gcp-cloud-kms-crypto-key-version`](/sources/gcp/Types/gcp-cloud-kms-crypto-key-version)

When CMEK is enabled, Spanner actually uses a specific version of the KMS key. This link shows the exact key version currently in use so you can track key rotation and ensure compliance.

### [`gcp-spanner-database`](/sources/gcp/Types/gcp-spanner-database)

Spanner databases may reference one another through backups, clones or restores. Overmind records these relationships (e.g., a database restored from another) to expose any dependency chain between databases.

### [`gcp-spanner-instance`](/sources/gcp/Types/gcp-spanner-instance)

Every Spanner database belongs to a single Spanner instance. This link lets you traverse from the database to the parent instance to understand the compute resources, regional configuration and IAM policies that ultimately govern the database.
