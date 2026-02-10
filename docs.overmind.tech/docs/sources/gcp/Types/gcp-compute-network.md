---
title: GCP Compute Network
sidebar_label: gcp-compute-network
---

A Google Cloud VPC (Virtual Private Cloud) network is a global, logically-isolated network that spans all regions within a Google Cloud project. It defines the IP address space, routing tables, firewall rules and connectivity options (for example, VPN, Cloud Interconnect and peering) for the resources that are attached to it. Each VPC network can contain one or more regional subnetworks that allocate IP addresses to individual resources.  
For a full description see the official Google Cloud documentation: https://cloud.google.com/vpc/docs/vpc.

**Terrafrom Mappings:**

- `google_compute_network.name`

## Supported Methods

- `GET`: Get a gcp-compute-network by its "name"
- `LIST`: List all gcp-compute-network
- ~~`SEARCH`~~

## Possible Links

### [`gcp-compute-network`](/sources/gcp/Types/gcp-compute-network)

A gcp-compute-network can be linked to another gcp-compute-network when the two are connected using VPC Network Peering. This relationship allows traffic to flow privately between the two VPC networks and is modelled in Overmind as a link between the respective network resources.

### [`gcp-compute-subnetwork`](/sources/gcp/Types/gcp-compute-subnetwork)

Each gcp-compute-network contains one or more gcp-compute-subnetwork resources. Overmind links a network to all of its subnetworks to show the hierarchy and to surface any risks that originate in the subnetwork configuration.
