---
title: GCP Big Query Routine
sidebar_label: gcp-big-query-routine
---

A BigQuery Routine is a reusable piece of SQL or JavaScript logic—such as a stored procedure, user-defined function (UDF), or table-valued function—stored inside a BigQuery dataset. Routines let you encapsulate complex transformations, calculations, or business rules and call them from queries just like native BigQuery functions. They can reference other BigQuery objects (tables, views, models, etc.) and may be version-controlled and secured independently of the data they operate on.  
Official documentation: https://cloud.google.com/bigquery/docs/reference/rest/v2/routines

**Terrafrom Mappings:**

  * `google_bigquery_routine.id`

## Supported Methods

* `GET`: Get GCP Big Query Routine by "gcp-big-query-dataset-id|gcp-big-query-routine-id"
* ~~`LIST`~~
* `SEARCH`: Search for GCP Big Query Routine by "gcp-big-query-routine-id"

## Possible Links

### [`gcp-big-query-dataset`](/sources/gcp/Types/gcp-big-query-dataset)

A routine is always contained within exactly one BigQuery dataset. The link lets you trace from a routine to its parent dataset to understand data location, access controls, and retention policies that also apply to the routine.

### [`gcp-storage-bucket`](/sources/gcp/Types/gcp-storage-bucket)

If a routine’s SQL references an external table backed by Cloud Storage, or if the routine loads/stages data via the `LOAD DATA` or `EXPORT DATA` statements, the routine implicitly depends on the corresponding Cloud Storage bucket. This link surfaces that dependency so you can assess the impact of bucket-level permissions and lifecycle rules on the routine’s execution.