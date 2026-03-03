---
title: GCP Big Table Admin Table
sidebar_label: gcp-big-table-admin-table
---

Google Cloud Bigtable is a scalable NoSQL database service for large analytical and operational workloads. A Bigtable **table** is the primary data container within an instance, organised into rows and column families. The Bigtable Admin API allows you to create, configure, list, and delete tables, as well as manage their IAM policies and column–family schemas. Full details can be found in the official documentation: https://cloud.google.com/bigtable/docs/reference/admin/rest

**Terrafrom Mappings:**

- `google_bigtable_table.id`
- `google_bigtable_table_iam_binding.instance_name`
- `google_bigtable_table_iam_member.instance_name`
- `google_bigtable_table_iam_policy.instance_name`

## Supported Methods

- `GET`: Get a gcp-big-table-admin-table by its "instances|tables"
- ~~`LIST`~~
- `SEARCH`: Search for BigTable tables in an instance. Use the format "instance_name" or "projects/[project_id]/instances/[instance_name]/tables/[table_name]" which is supported for terraform mappings.

## Possible Links

### [`gcp-big-table-admin-backup`](/sources/gcp/Types/gcp-big-table-admin-backup)

A Bigtable table can have one or more backups. Overmind links a table to its related `gcp-big-table-admin-backup` resources, making it easy to assess how backup configurations might be impacted by changes to the table.

### [`gcp-big-table-admin-instance`](/sources/gcp/Types/gcp-big-table-admin-instance)

Every table is created inside a single Bigtable instance. This link shows the parent `gcp-big-table-admin-instance` that owns the table so you can understand instance-level settings (such as clusters and IAM) that may affect the table.

### [`gcp-big-table-admin-table`](/sources/gcp/Types/gcp-big-table-admin-table)

Tables may reference each other indirectly through IAM policies or schema design. Overmind links tables to other tables when such relationships are detected, allowing you to trace dependencies across multiple Bigtable tables within or across instances.
