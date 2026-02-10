---
title: GCP Big Query Routine
sidebar_label: gcp-big-query-routine
---

A BigQuery Routine represents a user-defined piece of reusable logic—such as a stored procedure or user-defined function—that is stored inside a BigQuery dataset and can be invoked from SQL. Routines let teams encapsulate data-processing logic, share it across queries, and manage it with version control and Infrastructure-as-Code tools. For a full description of the capabilities and configuration options, see the Google Cloud documentation on routines (https://cloud.google.com/bigquery/docs/routines-intro).

**Terrafrom Mappings:**

- `google_bigquery_routine.routine_id`

## Supported Methods

- `GET`: Get GCP Big Query Routine by "gcp-big-query-dataset-id|gcp-big-query-routine-id"
- ~~`LIST`~~
- `SEARCH`: Search for GCP Big Query Routine by "gcp-big-query-routine-id"

## Possible Links

### [`gcp-big-query-dataset`](/sources/gcp/Types/gcp-big-query-dataset)

A routine is defined within a specific BigQuery dataset; the link shows the parent dataset that contains the routine.
