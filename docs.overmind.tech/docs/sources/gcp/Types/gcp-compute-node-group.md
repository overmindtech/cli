---
title: GCP Compute Node Group
sidebar_label: gcp-compute-node-group
---

A GCP Compute Node Group is a managed collection of sole-tenant nodes that are all created from the same node template. These groups allow you to provision and administer dedicated physical servers for your Compute Engine virtual machines, giving you fine-grained control over workload isolation, hardware affinity, licensing, and maintenance windows. For a detailed explanation, see the official Google Cloud documentation: https://cloud.google.com/compute/docs/nodes.

**Terrafrom Mappings:**

* `google_compute_node_group.name`
* `google_compute_node_template.name`

## Supported Methods

* `GET`: Get GCP Compute Node Group by "gcp-compute-node-group-name"
* `LIST`: List all GCP Compute Node Group items
* `SEARCH`: Search for GCP Compute Node Group by "gcp-compute-node-group-nodeTemplateName"
