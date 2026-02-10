---
title: GCP Big Table Admin Table
sidebar_label: gcp-big-table-admin-table
---

Google Cloud Bigtable tables are the primary data containers inside a Bigtable instance. A table holds rows of schemaless, wide-column data that can scale to petabytes while maintaining low-latency access. The Admin Table resource represents the configuration and lifecycle metadata for a table (for example, column families, garbage-collection rules, encryption settings and replication state). For a detailed explanation see the official documentation: https://docs.cloud.google.com/bigtable/docs/reference/admin/rpc.

**Terrafrom Mappings:**

- `google_bigtable_table.id`

## Supported Methods

- `GET`: Get a gcp-big-table-admin-table by its "instances|tables"
- ~~`LIST`~~
- `SEARCH`: Search for BigTable tables in an instance. Use the format "instance_name" or "projects/[project_id]/instances/[instance_name]/tables/[table_name]" which is supported for terraform mappings.

## Possible Links

### [`gcp-big-table-admin-backup`](/sources/gcp/Types/gcp-big-table-admin-backup)

A backup is a point-in-time snapshot that is created from a specific table. From a table resource you can enumerate the backups that protect it, or follow a backup back to the source table from which it was taken.

### [`gcp-big-table-admin-instance`](/sources/gcp/Types/gcp-big-table-admin-instance)

Every table belongs to exactly one Bigtable instance. The instance is the parent container that defines the clusters, replication topology and IAM policy under which the table operates.

### [`gcp-big-table-admin-table`](/sources/gcp/Types/gcp-big-table-admin-table)

Tables of the same type within the same project or instance can be cross-referenced for comparison, migration or restore operations (for example, when restoring a backup into a new table). Overmind links tables to other tables so you can trace relationships such as clone targets, restore destinations or sibling tables in the same instance.
