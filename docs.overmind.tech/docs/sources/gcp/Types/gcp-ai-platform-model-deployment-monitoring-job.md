---
title: GCP Ai Platform Model Deployment Monitoring Job
sidebar_label: gcp-ai-platform-model-deployment-monitoring-job
---

Google Cloud’s Model Deployment Monitoring Job is a managed Vertex AI (formerly AI Platform) service that continuously analyses a deployed model’s predictions to detect data drift, prediction drift and skew between training and online data. A job is attached to one or more deployed models on an Endpoint and periodically samples incoming predictions, calculates statistics, raises alerts and writes monitoring reports to BigQuery or Cloud Storage.  
Official documentation: https://cloud.google.com/vertex-ai/docs/model-monitoring/overview

## Supported Methods

* `GET`: Get a gcp-ai-platform-model-deployment-monitoring-job by its "locations|modelDeploymentMonitoringJobs"
* ~~`LIST`~~
* `SEARCH`: Search Model Deployment Monitoring Jobs within a location. Use the location name e.g., 'us-central1'

## Possible Links

### [`gcp-ai-platform-endpoint`](/sources/gcp/Types/gcp-ai-platform-endpoint)

The monitoring job is created against a specific Endpoint; it inspects the request/response traffic that the Endpoint receives for the deployed model versions.

### [`gcp-ai-platform-model`](/sources/gcp/Types/gcp-ai-platform-model)

Each job’s `modelDeploymentMonitoringObjectiveConfigs` identifies the Model (or model version) whose predictions are being monitored for drift or skew.

### [`gcp-big-query-table`](/sources/gcp/Types/gcp-big-query-table)

If BigQuery is chosen as the analysis destination, the job writes sampled prediction data and computed statistics into a BigQuery table referenced by this link.

### [`gcp-cloud-kms-crypto-key`](/sources/gcp/Types/gcp-cloud-kms-crypto-key)

The `encryptionSpec.kmsKeyName` field can point to a customer-managed KMS key that encrypts all monitoring artefacts produced by the job.

### [`gcp-monitoring-notification-channel`](/sources/gcp/Types/gcp-monitoring-notification-channel)

Alerting rules created by the job use Cloud Monitoring notification channels (e-mail, Pub/Sub, SMS, etc.) to notify operators when drift thresholds are breached.

### [`gcp-storage-bucket`](/sources/gcp/Types/gcp-storage-bucket)

When Cloud Storage is selected, the job stores prediction samples, intermediate files and final monitoring reports in a user-provided bucket.
