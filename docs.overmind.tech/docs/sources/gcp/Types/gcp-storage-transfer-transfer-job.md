---
title: GCP Storage Transfer Transfer Job
sidebar_label: gcp-storage-transfer-transfer-job
---

A Storage Transfer Service Job represents a scheduled or on-demand operation that copies data between cloud storage systems or from on-premises sources into Google Cloud Storage. A job defines source and destination locations, transfer options (such as whether to delete objects after transfer), scheduling, and optional notifications. For full details see the official Google documentation: https://cloud.google.com/storage-transfer/docs/overview

**Terrafrom Mappings:**

- `google_storage_transfer_job.name`

## Supported Methods

- `GET`: Get a gcp-storage-transfer-transfer-job by its "name"
- `LIST`: List all gcp-storage-transfer-transfer-job
- ~~`SEARCH`~~

## Possible Links

### [`gcp-iam-service-account`](/sources/gcp/Types/gcp-iam-service-account)

Storage Transfer Service creates and utilises a dedicated service account to read from the source and write to the destination. The transfer job must have the correct IAM roles granted on this service account, making the two resources inherently linked.

### [`gcp-pub-sub-subscription`](/sources/gcp/Types/gcp-pub-sub-subscription)

If transfer job notifications are configured, the Storage Transfer Service publishes messages to a Pub/Sub topic. A subscription attached to that topic receives the events, so a job that emits notifications will be related to the downstream subscriptions.

### [`gcp-pub-sub-topic`](/sources/gcp/Types/gcp-pub-sub-topic)

The transfer job can be configured to send success, failure, or progress notifications to a specific Pub/Sub topic. That topic therefore has a direct relationship with the job.

### [`gcp-storage-bucket`](/sources/gcp/Types/gcp-storage-bucket)

Buckets are commonly used as both sources and destinations for transfer jobs. Any bucket referenced in the `transferSpec` of a job (either as a source or destination) is linked to that job.
