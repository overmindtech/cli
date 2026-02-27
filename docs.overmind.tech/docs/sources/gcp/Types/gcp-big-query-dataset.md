---
title: GCP Big Query Dataset
sidebar_label: gcp-big-query-dataset
---

A Google Cloud BigQuery Dataset is a logical container that holds tables, views, routines (stored procedures and functions) and metadata, and defines the geographic location where the underlying data is stored. Datasets also act as the administrative boundary for access-control policies and encryption configuration. For a full description, see the official documentation: https://cloud.google.com/bigquery/docs/datasets-intro

**Terrafrom Mappings:**

* `google_bigquery_dataset.dataset_id`
* `google_bigquery_dataset_iam_binding.dataset_id`
* `google_bigquery_dataset_iam_member.dataset_id`
* `google_bigquery_dataset_iam_policy.dataset_id`

## Supported Methods

* `GET`: Get GCP Big Query Dataset by "gcp-big-query-dataset-id"
* `LIST`: List all GCP Big Query Dataset items
* ~~`SEARCH`~~

## Possible Links

### [`gcp-big-query-dataset`](/sources/gcp/Types/gcp-big-query-dataset)

Datasets can reference, copy from or authorise access to other BigQuery datasets, so Overmind may surface links where cross-dataset operations or shared access exist.

### [`gcp-big-query-routine`](/sources/gcp/Types/gcp-big-query-routine)

Every BigQuery routine (stored procedure or user-defined function) resides inside a specific dataset; therefore routines are children of the current dataset.

### [`gcp-big-query-table`](/sources/gcp/Types/gcp-big-query-table)

Tables and views are stored within a dataset. All tables that belong to this dataset will be linked here.

### [`gcp-cloud-kms-crypto-key`](/sources/gcp/Types/gcp-cloud-kms-crypto-key)

If customer-managed encryption is enabled, the dataset (and everything inside it) may be encrypted with a specific Cloud KMS crypto key. This link shows which key is in use.

### [`gcp-iam-service-account`](/sources/gcp/Types/gcp-iam-service-account)

Access to a dataset is granted via IAM, often to service accounts. Linked service accounts represent principals that have explicit permissions on the dataset.
