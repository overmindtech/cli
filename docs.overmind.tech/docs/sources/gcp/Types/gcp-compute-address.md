---
title: GCP Compute Address
sidebar_label: gcp-compute-address
---

A GCP Compute Address is a statically-reserved IPv4 or IPv6 address that can be assigned to Compute Engine resources such as virtual machine instances, forwarding rules, VPN gateways and load-balancers. Reserving the address stops it from changing when the attached resource is restarted and allows the address to be re-used on other resources later. Addresses may be global (for external HTTP(S) load-balancers) or regional (for most other use-cases), and internal addresses can be tied to a specific VPC network and sub-network.
For full details see the official documentation: https://docs.cloud.google.com/compute/docs/reference/rest/v1/addresses

**Terrafrom Mappings:**

- `google_compute_address.name`

## Supported Methods

- `GET`: Get GCP Compute Address by "gcp-compute-address-name"
- `LIST`: List all GCP Compute Address items
- ~~`SEARCH`~~

## Possible Links

### [`gcp-compute-address`](/sources/gcp/Types/gcp-compute-address)

A self-link that allows Overmind to relate this address to other instances of the same type (for example, distinguishing between regional and global addresses with identical names).

### [`gcp-compute-network`](/sources/gcp/Types/gcp-compute-network)

Internal (private) addresses are reserved within a specific VPC network, so an address will be linked to the `gcp-compute-network` that owns the IP range from which it is allocated.

### [`gcp-compute-subnetwork`](/sources/gcp/Types/gcp-compute-subnetwork)

When an internal address is scoped to a particular sub-network, Overmind records this dependency by linking the address to the corresponding `gcp-compute-subnetwork`.
