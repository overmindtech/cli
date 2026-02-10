---
title: GCP Ai Platform Model Deployment Monitoring Job
sidebar_label: gcp-ai-platform-model-deployment-monitoring-job
---

A Model Deployment Monitoring Job in Vertex AI (formerly AI Platform) performs continuous evaluation of a model that has been deployed to an endpoint. The job collects prediction requests and responses, analyses them for data drift, feature skew, and other anomalies, and can raise alerts when thresholds are exceeded. This enables teams to detect issues in production models early and take corrective action before business impact occurs.

Official documentation: https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.modelDeploymentMonitoringJobs

## Supported Methods

- `GET`: Get a gcp-ai-platform-model-deployment-monitoring-job by its "locations|modelDeploymentMonitoringJobs"
- ~~`LIST`~~
- `SEARCH`: Search Model Deployment Monitoring Jobs within a location. Use the location name e.g., 'us-central1'

## Possible Links

### [`gcp-ai-platform-endpoint`](/sources/gcp/Types/gcp-ai-platform-endpoint)

A Model Deployment Monitoring Job is always attached to a specific Vertex AI endpoint; it monitors one or more model deployments that live on that endpoint. The link represents the `endpoint` field inside the job resource.

### [`gcp-ai-platform-model`](/sources/gcp/Types/gcp-ai-platform-model)

Within `modelDeploymentMonitoringObjectiveConfigs`, the job specifies the deployed model(s) it should watch. This link captures that relationship between the monitoring job and the underlying Vertex AI model resources.

### [`gcp-cloud-kms-crypto-key`](/sources/gcp/Types/gcp-cloud-kms-crypto-key)

If the job is created with `encryptionSpec`, it uses a customer-managed Cloud KMS key to encrypt monitoring logs and metadata. The linked Crypto Key represents that key.

### [`gcp-monitoring-notification-channel`](/sources/gcp/Types/gcp-monitoring-notification-channel)

Alerting for drift or skew relies on Cloud Monitoring notification channels listed in the job’s `alertConfig.notificationChannels`. This link connects the monitoring job to those channels so users can trace how alerts will be delivered.
