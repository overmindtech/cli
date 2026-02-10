---
title: GCP Compute Route
sidebar_label: gcp-compute-route
---

A GCP Compute Route is an entry in the routing table of a Google Cloud VPC network that determines how packets are forwarded from its subnets. Each route specifies a destination CIDR block and a next hop (for example, an instance, VPN tunnel, gateway, or peered network). Custom routes can be created to direct traffic through specific appliances, across VPNs, or towards on-premises networks, while system-generated routes provide default Internet and subnet behaviour.  
See the official documentation for full details: https://cloud.google.com/vpc/docs/routes

**Terrafrom Mappings:**

- `google_compute_route.name`

## Supported Methods

- `GET`: Get a gcp-compute-route by its "name"
- `LIST`: List all gcp-compute-route
- ~~`SEARCH`~~

## Possible Links

### [`gcp-compute-instance`](/sources/gcp/Types/gcp-compute-instance)

If `next_hop_instance` is set, the route forwards matching traffic to the specified VM instance. Overmind therefore links the route to that Compute Instance, as deleting or modifying the instance will break the route.

### [`gcp-compute-network`](/sources/gcp/Types/gcp-compute-network)

Every route belongs to exactly one VPC network, referenced in the `network` field. The network’s routing table is the context in which the route operates, so Overmind links the route to its parent network.

### [`gcp-compute-vpn-tunnel`](/sources/gcp/Types/gcp-compute-vpn-tunnel)

When `next_hop_vpn_tunnel` is used, the route sends traffic into a specific VPN tunnel. This dependency is captured by linking the route to the corresponding Compute VPN Tunnel, since changes to the tunnel affect the route’s viability.
