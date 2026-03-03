---
title: GCP Compute Global Address
sidebar_label: gcp-compute-global-address
---

A **Compute Global Address** in Google Cloud Platform is a statically-reserved IP address that is reachable from, or usable across, all regions. It can be external (used, for example, by a global HTTP(S) load balancer) or internal (used by regional resources that require a routable, private global IP). Reserving the address ensures it does not change while it is in use, and allows it to be assigned to resources at creation time or later.  
Official documentation: https://cloud.google.com/compute/docs/ip-addresses/reserve-static-external-ip-address

**Terrafrom Mappings:**

- `google_compute_global_address.name`

## Supported Methods

- `GET`: Get a gcp-compute-global-address by its "name"
- `LIST`: List all gcp-compute-global-address
- ~~`SEARCH`~~

## Possible Links

### [`gcp-compute-network`](/sources/gcp/Types/gcp-compute-network)

A global address may be bound to a specific VPC network when it is reserved as an internal global IP. Overmind links the address to the `gcp-compute-network` so you can see in which network the address is routable and assess overlapping CIDR or routing risks.

### [`gcp-compute-public-delegated-prefix`](/sources/gcp/Types/gcp-compute-public-delegated-prefix)

If the address is carved out of a public delegated prefix that your project controls, Overmind links it to that `gcp-compute-public-delegated-prefix` to show the parent block and enable checks for exhaustion or mis-allocation.

### [`gcp-compute-subnetwork`](/sources/gcp/Types/gcp-compute-subnetwork)

For internal global addresses that are further scoped to a particular subnetwork, Overmind establishes a link to the `gcp-compute-subnetwork` so you can trace which subnet’s routing table and firewall rules apply to traffic destined for the address.
