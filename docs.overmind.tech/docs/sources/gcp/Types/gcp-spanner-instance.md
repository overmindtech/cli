---
title: GCP Spanner Instance
sidebar_label: gcp-spanner-instance
---

A **Cloud Spanner instance** is the top-level container that defines the geographical placement, compute capacity and billing context for one or more Cloud Spanner databases. When you create an instance you choose an instance configuration (regional or multi-regional) and allocate compute in the form of nodes or processing units; all databases created within the instance inherit this configuration and capacity. Google manages replication, automatic fail-over and online scaling transparently within the boundaries of the instance.  
For full details see the official documentation: https://cloud.google.com/spanner/docs/instances

**Terrafrom Mappings:**

  * `google_spanner_instance.name`

## Supported Methods

* `GET`: Get a gcp-spanner-instance by its "name"
* `LIST`: List all gcp-spanner-instance
* ~~`SEARCH`~~

## Possible Links

### [`gcp-spanner-database`](/sources/gcp/Types/gcp-spanner-database)

Each Cloud Spanner instance can contain multiple Cloud Spanner databases. The `gcp-spanner-database` resource is therefore a child of the `gcp-spanner-instance`; enumerating databases or assessing their risks starts with traversing from the parent instance to its associated databases.