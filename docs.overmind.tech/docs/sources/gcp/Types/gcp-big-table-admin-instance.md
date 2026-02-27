---
title: GCP Big Table Admin Instance
sidebar_label: gcp-big-table-admin-instance
---

Cloud Bigtable instances are the top-level administrative containers for all tables and data stored in Bigtable. An instance defines the service tier (production or development), the geographic placement of data through its clusters, and provides the entry point for IAM policy management, encryption settings, labelling and more. For a detailed overview of instances, see the official Google Cloud documentation: https://cloud.google.com/bigtable/docs/instances-clusters-nodes

**Terrafrom Mappings:**

* `google_bigtable_instance.name`
* `google_bigtable_instance_iam_binding.instance`
* `google_bigtable_instance_iam_member.instance`
* `google_bigtable_instance_iam_policy.instance`

## Supported Methods

* `GET`: Get a gcp-big-table-admin-instance by its "name"
* `LIST`: List all gcp-big-table-admin-instance
* ~~`SEARCH`~~

## Possible Links

### [`gcp-big-table-admin-cluster`](/sources/gcp/Types/gcp-big-table-admin-cluster)

Every Bigtable instance is composed of one or more clusters. A `gcp-big-table-admin-cluster` represents the individual cluster resources that reside within, and are owned by, a given Bigtable instance.
