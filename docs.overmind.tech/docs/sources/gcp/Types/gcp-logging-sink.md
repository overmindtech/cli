---
title: GCP Logging Sink
sidebar_label: gcp-logging-sink
---

A GCP Logging Sink is an export rule within Google Cloud Logging that continuously routes selected log entries to a destination such as BigQuery, Cloud Storage, Pub/Sub or another Logging bucket. Sinks allow you to retain logs for longer, perform analytics, or trigger near-real-time workflows outside Cloud Logging. Each sink is defined by three core elements: a filter that selects which log entries to export, a destination, and an IAM service account that is granted permission to write to that destination.  
For full details see the official documentation: https://cloud.google.com/logging/docs/export/configure_export

## Supported Methods

- `GET`: Get GCP Logging Sink by "gcp-logging-sink-name"
- `LIST`: List all GCP Logging Sink items
- ~~`SEARCH`~~

## Possible Links

### [`gcp-big-query-dataset`](/sources/gcp/Types/gcp-big-query-dataset)

If the sink’s destination is set to a BigQuery dataset, Overmind will create a link from the sink to that `gcp-big-query-dataset` resource because the sink writes log rows directly into the dataset’s `_TABLE_SUFFIX` sharded tables.

### [`gcp-logging-bucket`](/sources/gcp/Types/gcp-logging-bucket)

A sink can either originate from a Logging bucket (when the sink is scoped to that bucket) or target a Logging bucket in another project or billing account. Overmind therefore links the sink to the relevant `gcp-logging-bucket` to show where logs are pulled from or pushed to.

### [`gcp-pub-sub-topic`](/sources/gcp/Types/gcp-pub-sub-topic)

When a sink exports logs to Pub/Sub, it references a specific topic. Overmind links the sink to the corresponding `gcp-pub-sub-topic` so that users can trace event-driven pipelines or alerting mechanisms that rely on those published log messages.

### [`gcp-storage-bucket`](/sources/gcp/Types/gcp-storage-bucket)

If the sink is configured to deliver logs to Cloud Storage, the destination bucket appears as a linked `gcp-storage-bucket`. This highlights where log files are archived and the IAM relationship required for the sink’s writer identity to upload objects.
