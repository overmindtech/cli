---
title: GCP Big Query Model
sidebar_label: gcp-big-query-model
---

A BigQuery Model is a logical resource that stores the metadata and artefacts produced by BigQuery ML when you train a machine-learning model. It lives inside a BigQuery dataset and can subsequently be queried, evaluated, exported or further trained. For a full description see the official Google Cloud documentation: https://cloud.google.com/bigquery/docs/reference/rest/v2/models

## Supported Methods

- `GET`: Get GCP Big Query Model by "gcp-big-query-dataset-id|gcp-big-query-model-id"
- ~~`LIST`~~
- `SEARCH`: Search for GCP Big Query Model by "gcp-big-query-model-id"

## Possible Links

### [`gcp-big-query-dataset`](/sources/gcp/Types/gcp-big-query-dataset)

Each model is contained within exactly one BigQuery dataset. The link represents this parent–child relationship and allows Overmind to surface the impact of changes to the dataset on the model.

### [`gcp-big-query-table`](/sources/gcp/Types/gcp-big-query-table)

A model is usually trained from, and may reference, one or more BigQuery tables (for example, the training, validation and prediction input tables). This link lets Overmind trace how alterations to those tables could affect the model’s behaviour or validity.

### [`gcp-cloud-kms-crypto-key`](/sources/gcp/Types/gcp-cloud-kms-crypto-key)

If customer-managed encryption keys (CMEK) are enabled, the model’s data is encrypted with a Cloud KMS crypto-key. Linking the model to the crypto-key allows Overmind to assess the consequences of key rotation, deletion or permission changes on the model’s availability.
