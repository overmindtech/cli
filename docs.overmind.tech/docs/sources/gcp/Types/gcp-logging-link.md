---
title: GCP Logging Link
sidebar_label: gcp-logging-link
---

A GCP Logging Link is a Cloud Logging resource that connects a Log Bucket to an external analytics destination, currently a BigQuery dataset. Once the link is created, every entry that is written to the bucket is replicated to the linked BigQuery dataset in near real time, letting you query your logs with standard BigQuery SQL without having to configure or manage a separate Log Router sink.  
Logging Links are created under the path  
`projects/{project}/locations/{location}/buckets/{bucket}/links/{link}` and inherit the life-cycle and IAM policies of their parent bucket. They are regional, can optionally back-fill historical log data at creation time, and can be updated or deleted independently of the bucket or dataset.

For more information see the official documentation: https://cloud.google.com/logging/docs/reference/v2/rest/v2/projects.locations.buckets.links

## Supported Methods

- `GET`: Get a gcp-logging-link by its "locations|buckets|links"
- ~~`LIST`~~
- `SEARCH`: Search for gcp-logging-link by its "locations|buckets"

## Possible Links

### [`gcp-big-query-dataset`](/sources/gcp/Types/gcp-big-query-dataset)

A Logging Link points to the BigQuery dataset that serves as the analytics destination. The linked `gcp-big-query-dataset` receives a continuous copy of the logs contained in the parent bucket.

### [`gcp-logging-bucket`](/sources/gcp/Types/gcp-logging-bucket)

Every Logging Link is defined inside a specific `gcp-logging-bucket`. The bucket is the source of the log entries that are streamed to the linked BigQuery dataset.
