---
title: GCP Ai Platform Batch Prediction Job
sidebar_label: gcp-ai-platform-batch-prediction-job
---

A GCP AI Platform (Vertex AI) Batch Prediction Job is a managed job that runs a trained model against a large, static dataset to generate predictions asynchronously. It allows you to score data stored in Cloud Storage or BigQuery and write the results back to either service, without having to manage your own compute infrastructure. For full details see the official documentation: https://docs.cloud.google.com/vertex-ai/docs/predictions/get-batch-predictions

## Supported Methods

- `GET`: Get a gcp-ai-platform-batch-prediction-job by its "locations|batchPredictionJobs"
- ~~`LIST`~~
- `SEARCH`: Search Batch Prediction Jobs within a location. Use the location name e.g., 'us-central1'

## Possible Links

### [`gcp-ai-platform-model`](/sources/gcp/Types/gcp-ai-platform-model)

The batch prediction job references a trained model that provides the prediction logic. The job cannot run without specifying this model.

### [`gcp-big-query-table`](/sources/gcp/Types/gcp-big-query-table)

Input data for a batch prediction can come from a BigQuery table, and the job can also write the prediction results to another BigQuery table.

### [`gcp-cloud-kms-crypto-key`](/sources/gcp/Types/gcp-cloud-kms-crypto-key)

Customer-managed encryption keys (CMEK) from Cloud KMS may be attached to the job to encrypt its output artefacts stored in Cloud Storage or BigQuery.

### [`gcp-iam-service-account`](/sources/gcp/Types/gcp-iam-service-account)

The job is executed under a specific IAM service account, which grants it permissions to read inputs, write outputs, and access the model.

### [`gcp-storage-bucket`](/sources/gcp/Types/gcp-storage-bucket)

Cloud Storage buckets are commonly used to supply the input files (in JSONL or CSV) and/or to store the prediction output files produced by the batch job.
