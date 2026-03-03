---
title: GCP Compute Vpn Tunnel
sidebar_label: gcp-compute-vpn-tunnel
---

A Compute VPN Tunnel is the logical link that carries encrypted IP-sec traffic between Google Cloud and another network. It is created on top of a Google Cloud VPN Gateway and points at a peer gateway, defining parameters such as IKE version, shared secrets and traffic selectors. Each tunnel secures packets as they traverse the public Internet, allowing workloads in a VPC network to communicate privately with on-premises resources, other clouds, or additional GCP projects.  
Official documentation: https://cloud.google.com/compute/docs/reference/rest/v1/vpnTunnels

**Terrafrom Mappings:**

- `google_compute_vpn_tunnel.name`

## Supported Methods

- `GET`: Get a gcp-compute-vpn-tunnel by its "name"
- `LIST`: List all gcp-compute-vpn-tunnel
- ~~`SEARCH`~~

## Possible Links

### [`gcp-compute-external-vpn-gateway`](/sources/gcp/Types/gcp-compute-external-vpn-gateway)

A VPN tunnel targets an External VPN Gateway when its peer endpoint resides outside Google Cloud. The tunnel resource holds the reference that binds the Google side of the connection to the defined external gateway interface.

### [`gcp-compute-router`](/sources/gcp/Types/gcp-compute-router)

For dynamic (BGP) routing, a VPN tunnel is attached to a Cloud Router. The router exchanges routes with the peer across the tunnel, advertising VPC prefixes and learning remote prefixes.

### [`gcp-compute-vpn-gateway`](/sources/gcp/Types/gcp-compute-vpn-gateway)

Every VPN tunnel is created on a specific VPN Gateway (Classic or HA). The gateway provides the Google Cloud termination point, while the tunnel specifies the individual encrypted session parameters.
