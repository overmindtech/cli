---
title: GCP Logging Saved Query
sidebar_label: gcp-logging-saved-query
---

A GCP Logging Saved Query is a reusable, shareable filter definition for Google Cloud Logging (Logs Explorer). It stores the log filter expression, as well as optional display preferences and metadata, so that complex queries can be rerun or shared without having to rewrite the filter each time. Saved queries can be created at the project, folder, billing-account or organisation level and are particularly useful for operational run-books, incident response and dashboards.  
Official documentation: https://cloud.google.com/logging/docs/reference/v2/rest/v2/projects.locations.savedQueries

## Supported Methods

- `GET`: Get a gcp-logging-saved-query by its "locations|savedQueries"
- ~~`LIST`~~
- `SEARCH`: Search for gcp-logging-saved-query by its "locations"
