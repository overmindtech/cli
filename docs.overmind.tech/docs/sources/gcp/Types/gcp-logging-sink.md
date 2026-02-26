---
title: GCP Logging Sink
sidebar_label: gcp-logging-sink
---

A Logging Sink in Google Cloud Platform (GCP) is a routing rule that selects log entries with a user-defined filter and exports them to a chosen destination such as BigQuery, Cloud Storage, Pub/Sub, or another Cloud Logging bucket. Sinks are the building blocks of GCP’s Log Router and are used to retain, analyse or stream logs outside of the originating project, folder or organisation.  
Official documentation: https://cloud.google.com/logging/docs/export

## Supported Methods

* `GET`: Get GCP Logging Sink by "gcp-logging-sink-name"
* `LIST`: List all GCP Logging Sink items
* ~~`SEARCH`~~

## Possible Links

### [`gcp-big-query-dataset`](/sources/gcp/Types/gcp-big-query-dataset)

If the sink’s destination is a BigQuery table, it must reference a BigQuery dataset where the tables will be created and written to. The dataset therefore appears as a child dependency of the logging sink.

### [`gcp-iam-service-account`](/sources/gcp/Types/gcp-iam-service-account)

Every sink is assigned a writer_identity, which is an IAM service account that needs permission to write into the chosen destination. The sink’s correct operation depends on this service account having the required roles on the target resource.

### [`gcp-logging-bucket`](/sources/gcp/Types/gcp-logging-bucket)

A sink can route logs to another Cloud Logging bucket (including aggregated buckets at the folder or organisation level). In this case the sink targets, and must have write access to, the specified logging bucket.

### [`gcp-pub-sub-topic`](/sources/gcp/Types/gcp-pub-sub-topic)

When the destination is Pub/Sub, the sink exports each matching log entry as a message on a particular topic. The topic therefore represents an external linkage for onward streaming or event-driven processing.

### [`gcp-storage-bucket`](/sources/gcp/Types/gcp-storage-bucket)

For archival purposes a sink may export logs to a Cloud Storage bucket. The bucket must exist and grant the sink’s writer service account permission to create objects, making the storage bucket a direct dependency of the sink.