---
title: GCP Compute Vpn Gateway
sidebar_label: gcp-compute-vpn-gateway
---

A GCP Compute High-Availability (HA) VPN Gateway is a regional resource that provides secure, encrypted IPsec tunnels between a Google Cloud Virtual Private Cloud (VPC) network and peer networks (on-premises data centres, other clouds, or different GCP projects). The gateway offers redundancy by using two external interfaces, each of which can establish a pair of active tunnels, ensuring traffic continues to flow even during maintenance events or failures. Because the gateway is tightly coupled to a specific VPC network and region, it influences routing, firewall behaviour and overall network reachability.  
See the official Google Cloud documentation for full details: https://cloud.google.com/network-connectivity/docs/vpn/concepts/overview

**Terrafrom Mappings:**

  * `google_compute_ha_vpn_gateway.name`

## Supported Methods

* `GET`: Get a gcp-compute-vpn-gateway by its "name"
* `LIST`: List all gcp-compute-vpn-gateway
* ~~`SEARCH`~~

## Possible Links

### [`gcp-compute-network`](/sources/gcp/Types/gcp-compute-network)

Each HA VPN Gateway is created inside a single VPC network. Linking the gateway to its `gcp-compute-network` allows Overmind to trace which IP ranges, routes and firewall rules may be affected by the gateway’s tunnels, and to evaluate the blast radius of any proposed changes to either resource.