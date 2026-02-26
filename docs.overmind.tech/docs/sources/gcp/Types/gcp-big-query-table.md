---
title: GCP Big Query Table
sidebar_label: gcp-big-query-table
---

A BigQuery table is the fundamental storage unit inside Google Cloud BigQuery. It holds the actual rows of structured data that can be queried with SQL, shared, exported or used to build materialised views and machine-learning models. Tables live inside a dataset, can be partitioned or clustered, and may be encrypted either with Google-managed keys or customer-managed keys stored in Cloud KMS. They can also act as logical wrappers around external data held in Cloud Storage.  
Official documentation: https://cloud.google.com/bigquery/docs/tables

**Terrafrom Mappings:**

  * `google_bigquery_table.id`
  * `google_bigquery_table_iam_binding.dataset_id`
  * `google_bigquery_table_iam_member.dataset_id`
  * `google_bigquery_table_iam_policy.dataset_id`

## Supported Methods

* `GET`: Get GCP Big Query Table by "gcp-big-query-dataset-id|gcp-big-query-table-id"
* ~~`LIST`~~
* `SEARCH`: Search for GCP Big Query Table by "gcp-big-query-dataset-id"

## Possible Links

### [`gcp-big-query-dataset`](/sources/gcp/Types/gcp-big-query-dataset)

The dataset is the immediate parent container of the table; every table must belong to exactly one dataset and inherits default encryption, location and IAM settings from it.

### [`gcp-big-query-table`](/sources/gcp/Types/gcp-big-query-table)

BigQuery tables can reference, copy from, or be copied to other tables (for example when creating snapshots, clones, views with explicit table references or COPY jobs). Such relationships are captured as links between table resources.

### [`gcp-cloud-kms-crypto-key`](/sources/gcp/Types/gcp-cloud-kms-crypto-key)

If the table (or its parent dataset) is configured to use customer-managed encryption, it points to the Cloud KMS CryptoKey that protects the data at rest.

### [`gcp-storage-bucket`](/sources/gcp/Types/gcp-storage-bucket)

An external BigQuery table may use objects stored in a Cloud Storage bucket as its underlying data source; in that case the table is linked to the bucket holding those objects.