---
title: GCP Big Query Dataset
sidebar_label: gcp-big-query-dataset
---

A BigQuery Dataset is a top-level container that holds BigQuery tables, views, models and routines, and defines the geographic location where that data is stored. It also acts as the unit for access control, default encryption configuration and data lifecycle policies.  
For full details see the Google Cloud documentation: https://cloud.google.com/bigquery/docs/datasets

**Terrafrom Mappings:**

- `google_bigquery_dataset.dataset_id`

## Supported Methods

- `GET`: Get GCP Big Query Dataset by "gcp-big-query-dataset-id"
- `LIST`: List all GCP Big Query Dataset items
- ~~`SEARCH`~~

## Possible Links

### [`gcp-big-query-dataset`](/sources/gcp/Types/gcp-big-query-dataset)

A dataset can reference other datasets via authorised views or cross-dataset access entries. Those referenced datasets will be linked to the current item.

### [`gcp-big-query-model`](/sources/gcp/Types/gcp-big-query-model)

Every BigQuery ML model belongs to exactly one dataset. All models whose `dataset_id` matches this dataset will be linked.

### [`gcp-big-query-table`](/sources/gcp/Types/gcp-big-query-table)

Tables and views are stored inside a dataset. All tables whose `dataset_id` equals this dataset will be linked.

### [`gcp-cloud-kms-crypto-key`](/sources/gcp/Types/gcp-cloud-kms-crypto-key)

If the dataset is encrypted with a customer-managed key, the KMS Crypto Key used for default encryption will be linked here.

### [`gcp-iam-service-account`](/sources/gcp/Types/gcp-iam-service-account)

Service accounts that appear in the dataset’s IAM policy (for example as editors, owners, readers or custom roles) will be linked to show who can access or manage the dataset.
