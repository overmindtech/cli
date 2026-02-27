---
title: GCP Storage Transfer Transfer Job
sidebar_label: gcp-storage-transfer-transfer-job
---

Google Cloud Storage Transfer Service enables you to copy or synchronise data between Cloud Storage buckets, on-premises file systems and external cloud providers. A Storage Transfer **transfer job** is the top-level resource that defines where data should be copied from, where it should be copied to, the schedule on which the copy should run, and options such as delete or overwrite rules.  
Official documentation: https://cloud.google.com/storage-transfer/docs/create-transfers

**Terrafrom Mappings:**

* `google_storage_transfer_job.name`

## Supported Methods

* `GET`: Get a gcp-storage-transfer-transfer-job by its "name"
* `LIST`: List all gcp-storage-transfer-transfer-job
* ~~`SEARCH`~~

## Possible Links

### [`gcp-iam-service-account`](/sources/gcp/Types/gcp-iam-service-account)

The transfer job runs under a Google-managed or user-specified IAM service account, which needs roles such as `Storage Object Admin` on the destination bucket and, when applicable, permissions to access the source.

### [`gcp-pub-sub-subscription`](/sources/gcp/Types/gcp-pub-sub-subscription)

If event notifications are enabled, a Pub/Sub subscription can pull the messages that the transfer job publishes when it starts, completes, or encounters errors.

### [`gcp-pub-sub-topic`](/sources/gcp/Types/gcp-pub-sub-topic)

A transfer job can be configured with a Pub/Sub topic as its notification destination so that operational events are published for downstream processing or alerting.

### [`gcp-secret-manager-secret`](/sources/gcp/Types/gcp-secret-manager-secret)

When transferring from external providers such as AWS S3 or Azure Blob Storage, the access keys and credentials are often stored in Secret Manager secrets, which the transfer job references to authenticate to the source.

### [`gcp-storage-bucket`](/sources/gcp/Types/gcp-storage-bucket)

Every transfer job specifies at least one Cloud Storage bucket as a source and/or destination; therefore it has direct relationships to the buckets involved in the data copy.
