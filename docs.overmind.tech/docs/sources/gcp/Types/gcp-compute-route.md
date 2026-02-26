---
title: GCP Compute Route
sidebar_label: gcp-compute-route
---

A **GCP Compute Route** is a routing rule attached to a Google Cloud Virtual Private Cloud (VPC) network that determines how packets are forwarded from instances towards their destinations. Each route contains a destination CIDR block and a single next-hop target, such as an instance, VPN tunnel, gateway or internal load-balancer forwarding rule. Routes can be either system-generated (e.g. subnet and peering routes) or user-defined to control custom traffic flows, enforce security boundaries or implement hybrid-connectivity scenarios.  
Official documentation: https://cloud.google.com/vpc/docs/routes

**Terrafrom Mappings:**

  * `google_compute_route.name`

## Supported Methods

* `GET`: Get a gcp-compute-route by its "name"
* `LIST`: List all gcp-compute-route
* `SEARCH`: Search for routes by network tag. The query is a plain network tag name.

## Possible Links

### [`gcp-compute-forwarding-rule`](/sources/gcp/Types/gcp-compute-forwarding-rule)

A route may specify an internal TCP/UDP load balancer (ILB) forwarding rule as its `nextHopIlb`, so the route is linked to the forwarding rule that receives the traffic.

### [`gcp-compute-instance`](/sources/gcp/Types/gcp-compute-instance)

When `nextHopInstance` is used, the route points to a specific Compute Engine instance that acts as a gateway. Instances are therefore linked as potential next hops for the route.

### [`gcp-compute-network`](/sources/gcp/Types/gcp-compute-network)

Every route is created inside exactly one VPC network, referenced by the `network` field. The relationship ties the route to the network whose traffic it influences.

### [`gcp-compute-vpn-tunnel`](/sources/gcp/Types/gcp-compute-vpn-tunnel)

If `nextHopVpnTunnel` is set, the route forwards matching traffic into a Cloud VPN tunnel. The route is consequently linked to the VPN tunnel resource it targets.