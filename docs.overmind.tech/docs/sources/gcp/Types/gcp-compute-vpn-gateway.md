---
title: GCP Compute Vpn Gateway
sidebar_label: gcp-compute-vpn-gateway
---

A Google Cloud Compute VPN Gateway (specifically, the High-Availability VPN Gateway) provides a managed, highly available IPsec VPN endpoint that allows encrypted traffic to flow between a Google Cloud Virtual Private Cloud (VPC) network and an on-premises network or another cloud provider. By deploying a VPN Gateway you can create site-to-site tunnels that automatically scale their throughput and offer automatic fail-over across two interfaces in different zones within the same region.  
For full details see the official documentation: https://cloud.google.com/network-connectivity/docs/vpn/concepts/overview

**Terrafrom Mappings:**

- `google_compute_ha_vpn_gateway.name`

## Supported Methods

- `GET`: Get a gcp-compute-vpn-gateway by its "name"
- `LIST`: List all gcp-compute-vpn-gateway
- ~~`SEARCH`~~

## Possible Links

### [`gcp-compute-network`](/sources/gcp/Types/gcp-compute-network)

An HA VPN Gateway is created inside, and tightly bound to, a specific VPC network and region. It inherits the network’s subnet routes and advertises them across its VPN tunnels, and all incoming VPN traffic is delivered into that network.
