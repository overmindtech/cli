---
title: GCP Compute Node Group
sidebar_label: gcp-compute-node-group
---

A **Google Cloud Compute Node Group** is a logical grouping of one or more sole-tenant nodes – dedicated physical Compute Engine servers that are exclusively reserved for your projects. Node groups let you manage the life-cycle, scheduling policies and placement of these nodes as a single resource. They are typically used when you need hardware isolation for licensing or security reasons, or when you require predictable performance unaffected by noisy neighbours. Each node in the group is created from a Node Template that defines the machine type, CPU platform, labels and maintenance behaviour for the nodes.  
Official documentation: https://cloud.google.com/compute/docs/nodes/sole-tenant-nodes

**Terrafrom Mappings:**

- `google_compute_node_group.name`
- `google_compute_node_template.name`

## Supported Methods

- `GET`: Get GCP Compute Node Group by "gcp-compute-node-group-name"
- `LIST`: List all GCP Compute Node Group items
- `SEARCH`: Search for GCP Compute Node Group by "gcp-compute-node-group-nodeTemplateName"
