---
title: GCP Big Query Table
sidebar_label: gcp-big-query-table
---

A BigQuery table is the fundamental unit of storage in Google Cloud BigQuery. It holds the rows of structured data that analysts query using SQL, and it defines the schema, partitioning, clustering, and encryption settings that govern how that data is stored and accessed. For a full description see the Google Cloud documentation: https://cloud.google.com/bigquery/docs/tables

**Terrafrom Mappings:**

- `google_bigquery_table.id`

## Supported Methods

- `GET`: Get GCP Big Query Table by "gcp-big-query-dataset-id|gcp-big-query-table-id"
- ~~`LIST`~~
- `SEARCH`: Search for GCP Big Query Table by "gcp-big-query-dataset-id"

## Possible Links

### [`gcp-big-query-dataset`](/sources/gcp/Types/gcp-big-query-dataset)

Every BigQuery table is contained within exactly one dataset. This link represents that parent–child relationship, enabling Overmind to trace from a table back to the dataset that organises and administers it.

### [`gcp-cloud-kms-crypto-key`](/sources/gcp/Types/gcp-cloud-kms-crypto-key)

If a BigQuery table is encrypted with a customer-managed encryption key (CMEK), this link points to the specific Cloud KMS crypto key in use. It allows Overmind to surface risks associated with key rotation, permissions, or key deletion that could affect the table’s availability or compliance posture.
