---
title: GCP Spanner Instance
sidebar_label: gcp-spanner-instance
---

A **Cloud Spanner instance** is the top-level container for Cloud Spanner resources in Google Cloud. It specifies the geographic placement of the underlying nodes, the amount of compute capacity allocated (measured in processing units), and the instance’s name and labels. All Cloud Spanner databases and their data live inside an instance, and the instance’s configuration determines their availability and latency characteristics.  
For full details see the Google Cloud documentation: https://cloud.google.com/spanner/docs/instances

**Terrafrom Mappings:**

- `google_spanner_instance.name`

## Supported Methods

- `GET`: Get a gcp-spanner-instance by its "name"
- `LIST`: List all gcp-spanner-instance
- ~~`SEARCH`~~

## Possible Links

### [`gcp-spanner-database`](/sources/gcp/Types/gcp-spanner-database)

A Cloud Spanner instance can host one or more Cloud Spanner databases. Each `gcp-spanner-database` discovered by Overmind will therefore be linked to the `gcp-spanner-instance` that contains it, allowing you to see which databases would be affected by changes to, or deletion of, the parent instance.
