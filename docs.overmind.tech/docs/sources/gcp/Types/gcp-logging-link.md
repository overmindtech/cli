---
title: GCP Logging Link
sidebar_label: gcp-logging-link
---

A GCP Logging Link is a Cloud Logging resource that continuously streams the log entries stored in a specific Log Bucket into an external BigQuery dataset. By configuring a link you enable near-real-time analytics of your logs with BigQuery without the need for manual exports or scheduled jobs. Links are created under the path

`projects|folders|organizations|billingAccounts / locations / buckets / links`

and each link specifies the destination BigQuery dataset, IAM writer identity, and lifecycle state.  
For further details see Google’s official documentation: https://cloud.google.com/logging/docs/reference/v2/rest/v2/projects.locations.buckets.links

## Supported Methods

- `GET`: Get a gcp-logging-link by its "locations|buckets|links"
- ~~`LIST`~~
- `SEARCH`: Search for gcp-logging-link by its "locations|buckets"

## Possible Links

### [`gcp-big-query-dataset`](/sources/gcp/Types/gcp-big-query-dataset)

A logging link targets exactly one BigQuery dataset; Overmind establishes this edge so you can trace which dataset is receiving log entries from the bucket.

### [`gcp-logging-bucket`](/sources/gcp/Types/gcp-logging-bucket)

The logging link is defined inside a specific Log Bucket; this relationship lets you see which buckets are sending their logs onwards and to which destinations.
