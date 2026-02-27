---
title: GCP Logging Saved Query
sidebar_label: gcp-logging-saved-query
---

A GCP Logging Saved Query is a reusable, named log query that is stored in Google Cloud Logging’s Logs Explorer. It contains the filter expression (or Log Query Language statement), any configured time-range presets and display options, allowing teams to quickly rerun common searches, share queries across projects, and use them as the basis for dashboards, log-based metrics or alerting policies. Because Saved Queries are resources in their own right, they can be created, read, updated and deleted through the Cloud Logging API, and are uniquely identified by the combination of the Google Cloud location and the query name.  
Official documentation: https://cloud.google.com/logging/docs/view/building-queries

## Supported Methods

* `GET`: Get a gcp-logging-saved-query by its "locations|savedQueries"
* ~~`LIST`~~
* `SEARCH`: Search for gcp-logging-saved-query by its "locations"
