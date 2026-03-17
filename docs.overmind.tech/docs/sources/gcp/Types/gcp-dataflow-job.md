---
title: GCP Dataflow Job
sidebar_label: gcp-dataflow-job
---

A **Google Cloud Dataflow Job** is a managed Apache Beam pipeline that processes streaming or batch data at scale. Dataflow handles resource provisioning, autoscaling, and fault tolerance, allowing you to run data processing workloads without managing the underlying infrastructure. Jobs can read from and write to Pub/Sub, BigQuery, Spanner, Bigtable, and other GCP services. See the official documentation for full details: https://cloud.google.com/dataflow/docs.

**Terraform Mappings:**

- `google_dataflow_job.job_id`
- `google_dataflow_flex_template_job.job_id`

## Supported Methods

- `GET`: Get a gcp-dataflow-job by its "locations|jobs"
- ~~`LIST`~~
- `SEARCH`: Search for gcp-dataflow-job by location

## Possible Links

### [`gcp-big-query-dataset`](/sources/gcp/Types/gcp-big-query-dataset)

Dataflow jobs that read from or write to BigQuery reference the dataset containing the tables they use. If the dataset is deleted or misconfigured, the job may fail to access data.

### [`gcp-big-query-table`](/sources/gcp/Types/gcp-big-query-table)

Dataflow jobs can read from or write to specific BigQuery tables. If a table is deleted or its schema changes, the job may fail.

### [`gcp-big-table-admin-instance`](/sources/gcp/Types/gcp-big-table-admin-instance)

Dataflow jobs that use Bigtable as a source or sink reference the Bigtable instance. If the instance is deleted or misconfigured, the job may fail.

### [`gcp-cloud-kms-crypto-key`](/sources/gcp/Types/gcp-cloud-kms-crypto-key)

When customer-managed encryption keys (CMEK) are enabled for the Dataflow job environment, the job references the Cloud KMS Crypto Key used for encryption.

### [`gcp-compute-network`](/sources/gcp/Types/gcp-compute-network)

Dataflow worker VMs are attached to a VPC network. If the network is deleted or misconfigured, workers may lose connectivity or fail to start.

### [`gcp-compute-subnetwork`](/sources/gcp/Types/gcp-compute-subnetwork)

Dataflow workers run in a specific subnetwork. If the subnetwork is deleted or misconfigured, workers may lose connectivity or fail to start.

### [`gcp-iam-service-account`](/sources/gcp/Types/gcp-iam-service-account)

Dataflow workers run under a service account that grants them permissions to access other GCP services. If the service account is deleted or its permissions change, the job may fail.

### [`gcp-pub-sub-subscription`](/sources/gcp/Types/gcp-pub-sub-subscription)

Dataflow jobs that consume messages from Pub/Sub reference the subscription. If the subscription is deleted or misconfigured, the job may fail to consume messages.

### [`gcp-pub-sub-topic`](/sources/gcp/Types/gcp-pub-sub-topic)

Dataflow jobs that publish to or consume from Pub/Sub reference the topic. If the topic is deleted or misconfigured, the job may fail to read or write messages.

### [`gcp-spanner-instance`](/sources/gcp/Types/gcp-spanner-instance)

Dataflow jobs that use Spanner reference the Spanner instance. If the instance is deleted or misconfigured, the job may fail.
