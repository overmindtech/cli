---
title: GCP Compute Network
sidebar_label: gcp-compute-network
---

A Google Cloud Platform (GCP) Compute Network—commonly called a Virtual Private Cloud (VPC) network—provides the fundamental isolation and IP address space in which all other networking resources (subnetworks, routes, firewall rules, VPNs, etc.) are created. It is a global resource that spans all regions in a project, allowing workloads to communicate securely inside Google’s backbone and to the internet where required. For a full description see the official documentation: https://cloud.google.com/vpc/docs/vpc

**Terrafrom Mappings:**

  * `google_compute_network.name`

## Supported Methods

* `GET`: Get a gcp-compute-network by its "name"
* `LIST`: List all gcp-compute-network
* ~~`SEARCH`~~

## Possible Links

### [`gcp-compute-network`](/sources/gcp/Types/gcp-compute-network)

A Compute Network can be peered with, or shared to, another Compute Network. Overmind records these peer or shared-VPC relationships by linking one `gcp-compute-network` item to the other(s).

### [`gcp-compute-subnetwork`](/sources/gcp/Types/gcp-compute-subnetwork)

Every subnetwork is created inside exactly one VPC network. Overmind therefore links each `gcp-compute-subnetwork` back to its parent `gcp-compute-network`, and conversely shows the network’s collection of subnetworks.