---
title: GCP Compute Global Address
sidebar_label: gcp-compute-global-address
---

A Compute Global Address is a static, reserved IP address that is accessible from any Google Cloud region. It can be either external (public) or internal, and is typically used by globally distributed resources such as HTTP(S) load balancers, Cloud Run services, or global internal load balancers. Once reserved, the address can be bound to forwarding rules or other network endpoints, ensuring that the same IP is advertised worldwide.  
For full details, see the official documentation: https://cloud.google.com/compute/docs/ip-addresses/reserve-static-external-ip-address#global_addresses

**Terrafrom Mappings:**

- `google_compute_global_address.name`

## Supported Methods

- `GET`: Get a gcp-compute-global-address by its "name"
- `LIST`: List all gcp-compute-global-address
- ~~`SEARCH`~~

## Possible Links

### [`gcp-compute-network`](/sources/gcp/Types/gcp-compute-network)

Global internal addresses must be created within a specific VPC network, and the `network` attribute on the address points to that VPC. Overmind therefore links a gcp-compute-global-address to the corresponding gcp-compute-network so that you can understand which network context the IP address belongs to and assess any related risks.
