---
title: GCP Compute Node Template
sidebar_label: gcp-compute-node-template
---

A GCP Compute Node Template is a reusable description of the hardware configuration and host maintenance policies that will be applied to one or more Sole-Tenant Nodes in Google Cloud. The template specifies attributes such as CPU platform, virtual CPU count, memory, node affinity labels, and automatic restart behaviour. When you later create a Node Group, the group references a single Node Template, ensuring that every node in the group is created with an identical shape.  
For a full specification of the resource, see the official Google Cloud documentation: https://cloud.google.com/compute/docs/nodes/sole-tenant-nodes

**Terrafrom Mappings:**

- `google_compute_node_template.name`

## Supported Methods

- `GET`: Get GCP Compute Node Template by "gcp-compute-node-template-name"
- `LIST`: List all GCP Compute Node Template items
- ~~`SEARCH`~~

## Possible Links

### [`gcp-compute-node-group`](/sources/gcp/Types/gcp-compute-node-group)

A GCP Compute Node Group consumes a single Node Template. Overmind creates a link from a node group back to the template it references so that you can assess how changes to the template (for example, switching CPU platforms) will affect every node that belongs to the group.
