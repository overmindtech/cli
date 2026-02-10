---
title: GCP Compute Subnetwork
sidebar_label: gcp-compute-subnetwork
---

A GCP Compute Subnetwork is a regional segment of a Virtual Private Cloud (VPC) network that defines an IP address range from which resources such as VM instances, GKE nodes, and internal load balancers receive their internal IP addresses. Each subnetwork is bound to a single region, can be configured for automatic or custom IP allocation, and supports features such as Private Google Access and flow logs. For full details see the official Google Cloud documentation: https://cloud.google.com/vpc/docs/subnets

**Terrafrom Mappings:**

- `google_compute_subnetwork.name`

## Supported Methods

- `GET`: Get a gcp-compute-subnetwork by its "name"
- `LIST`: List all gcp-compute-subnetwork
- ~~`SEARCH`~~

## Possible Links

### [`gcp-compute-network`](/sources/gcp/Types/gcp-compute-network)

Every subnetwork is a child resource of a VPC network. The `gcp-compute-network` item represents that parent VPC; a single network can contain multiple subnetworks, while each subnetwork is associated with exactly one network.
