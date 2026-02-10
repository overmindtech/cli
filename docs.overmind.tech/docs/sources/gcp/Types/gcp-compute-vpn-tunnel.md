---
title: GCP Compute Vpn Tunnel
sidebar_label: gcp-compute-vpn-tunnel
---

A **GCP Compute VPN Tunnel** represents a single IPSec tunnel that is part of a Cloud VPN connection. It contains the parameters needed to establish and maintain the encrypted link – peer IP address, shared secret, IKE version, traffic selectors, and the attachment to either a Classic VPN gateway or an HA VPN gateway. In most deployments two or more tunnels are created for redundancy.  
For the full specification see the official Google documentation: https://cloud.google.com/compute/docs/reference/rest/v1/vpnTunnels

**Terrafrom Mappings:**

- `google_compute_vpn_tunnel.name`

## Supported Methods

- `GET`: Get a gcp-compute-vpn-tunnel by its "name"
- `LIST`: List all gcp-compute-vpn-tunnel
- ~~`SEARCH`~~

## Possible Links

### [`gcp-compute-external-vpn-gateway`](/sources/gcp/Types/gcp-compute-external-vpn-gateway)

When the tunnel terminates on equipment outside Google Cloud, the `externalVpnGateway` field is set. This creates a relationship between the VPN tunnel and the corresponding External VPN Gateway resource.

### [`gcp-compute-router`](/sources/gcp/Types/gcp-compute-router)

If dynamic routing is enabled (HA VPN or dynamic Classic VPN), the tunnel is attached to a Cloud Router, which advertises and learns routes via BGP. The `router` field therefore links the VPN tunnel to a specific Cloud Router.

### [`gcp-compute-vpn-gateway`](/sources/gcp/Types/gcp-compute-vpn-gateway)

Every tunnel belongs to a Google-managed VPN gateway (`targetVpnGateway` for Classic VPN or `vpnGateway` for HA VPN). This link captures that parent-child relationship, allowing Overmind to evaluate the impact of gateway changes on its tunnels.
