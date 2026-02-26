---
title: GCP Compute Router
sidebar_label: gcp-compute-router
---

A Google Cloud Compute Router is a fully distributed and managed Border Gateway Protocol (BGP) routing service that dynamically exchanges routes between your Virtual Private Cloud (VPC) network and on-premises or cloud networks connected via VPN or Cloud Interconnect. By advertising only the necessary prefixes, it enables highly available, scalable, and policy-driven traffic engineering without the need to run or maintain your own routing appliances. See the official documentation for full details: https://cloud.google.com/network-connectivity/docs/router

**Terrafrom Mappings:**

  * `google_compute_router.id`

## Supported Methods

* `GET`: Get a gcp-compute-router by its "name"
* `LIST`: List all gcp-compute-router
* `SEARCH`: Search with full ID: projects/[project]/regions/[region]/routers/[router] (used for terraform mapping).

## Possible Links

### [`gcp-compute-network`](/sources/gcp/Types/gcp-compute-network)

A Compute Router is created inside a specific VPC network and advertises routes for that network; therefore it is directly linked to the gcp-compute-network resource in which it resides.

### [`gcp-compute-subnetwork`](/sources/gcp/Types/gcp-compute-subnetwork)

Subnets within the parent VPC network can have their routes propagated or learned via the Compute Router, especially when using dynamic routing modes; this establishes an indirect but important relationship with each gcp-compute-subnetwork.

### [`gcp-compute-vpn-tunnel`](/sources/gcp/Types/gcp-compute-vpn-tunnel)

When Cloud VPN is configured in dynamic mode, the VPN tunnel relies on a Compute Router to exchange BGP routes with the peer gateway, making the tunnel dependent on, and logically linked to, the corresponding gcp-compute-vpn-tunnel resource.