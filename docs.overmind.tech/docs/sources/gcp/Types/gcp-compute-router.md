---
title: GCP Compute Router
sidebar_label: gcp-compute-router
---

A Google Cloud **Compute Router** is a regional, fully distributed control-plane resource that learns and exchanges dynamic routes between your Virtual Private Cloud (VPC) network and on-premises or partner networks. It implements the Border Gateway Protocol (BGP) on your behalf, allowing Cloud VPN tunnels and Cloud Interconnect attachments (VLANs) to advertise and receive custom routes without manual updates. Compute Routers are attached to a specific VPC network and region, but they propagate learned routes across the entire VPC through Google’s global backbone.  
For a comprehensive overview, refer to the official Google Cloud documentation: https://cloud.google.com/network-connectivity/docs/router/how-to/creating-routers

**Terrafrom Mappings:**

- `google_compute_router.id`

## Supported Methods

- `GET`: Get a gcp-compute-router by its "name"
- `LIST`: List all gcp-compute-router
- `SEARCH`: Search with full ID: projects/[project]/regions/[region]/routers/[router] (used for terraform mapping).

## Possible Links

### [`gcp-compute-network`](/sources/gcp/Types/gcp-compute-network)

Every Compute Router is created inside a particular VPC network; the router exchanges routes on behalf of that network. Therefore, a gcp-compute-router will always have an owning gcp-compute-network.

### [`gcp-compute-subnetwork`](/sources/gcp/Types/gcp-compute-subnetwork)

Subnets define the IP ranges that the Compute Router ultimately advertises (or learns routes for) within the VPC. Routes learned or propagated by the router directly affect traffic flowing to and from gcp-compute-subnetwork resources.

### [`gcp-compute-vpn-tunnel`](/sources/gcp/Types/gcp-compute-vpn-tunnel)

Compute Routers terminate the BGP sessions used by Cloud VPN (HA VPN) tunnels. Each gcp-compute-vpn-tunnel can be configured to peer with a Compute Router interface, enabling dynamic route exchange between the tunnel and the VPC.
