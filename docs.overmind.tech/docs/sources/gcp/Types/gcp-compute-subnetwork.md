---
title: GCP Compute Subnetwork
sidebar_label: gcp-compute-subnetwork
---

A GCP Compute Subnetwork is a regional, layer-3 virtual network segment that belongs to a single Google Cloud VPC network. It defines an internal RFC 1918 IP address range (primary and optional secondary ranges) from which VM instances, containers and other resources receive their internal IPs. Within each subnetwork you can enable or disable Private Google Access, set flow-log export settings, IPv6 configurations, and control access through firewall rules inherited from the parent VPC. For a comprehensive overview refer to the official documentation: https://cloud.google.com/vpc/docs/subnets.

**Terrafrom Mappings:**

* `google_compute_subnetwork.name`

## Supported Methods

* `GET`: Get a gcp-compute-subnetwork by its "name"
* `LIST`: List all gcp-compute-subnetwork
* ~~`SEARCH`~~

## Possible Links

### [`gcp-compute-network`](/sources/gcp/Types/gcp-compute-network)

Every subnetwork is created inside exactly one VPC network. This link represents that parent–child relationship, allowing Overmind to show which VPC a particular subnetwork belongs to and, conversely, to enumerate all subnetworks within a given VPC.

### [`gcp-compute-public-delegated-prefix`](/sources/gcp/Types/gcp-compute-public-delegated-prefix)

A public delegated prefix can be assigned to a subnetwork so that resources inside the subnet can use public IPv4 addresses from that prefix. This link highlights which delegated prefixes are associated with, or routed through, the subnetwork, helping users trace external IP allocations and their exposure.
